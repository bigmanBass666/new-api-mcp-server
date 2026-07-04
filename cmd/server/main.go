package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
	"github.com/QuantumNous/new-api-mcp-server/internal/config"
	"github.com/QuantumNous/new-api-mcp-server/internal/handler"
	"github.com/QuantumNous/new-api-mcp-server/internal/hightools"
	"github.com/QuantumNous/new-api-mcp-server/internal/middleware"
	"github.com/QuantumNous/new-api-mcp-server/internal/observability"
	openapipkg "github.com/QuantumNous/new-api-mcp-server/internal/openapi"
	"github.com/QuantumNous/new-api-mcp-server/internal/registry"
	embeddedSpecs "github.com/QuantumNous/new-api-mcp-server/openapi"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/prometheus/client_golang/prometheus"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	observability.SetupLogging(cfg.LogLevel, cfg.LogFormat, cfg.LogConsoleEnabled, cfg.OTLPEndpoint, nil)

	slog.Info("starting new-api-mcp-server", "version", version, "transport", cfg.Transport)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracing, err := observability.SetupTracing(ctx, cfg.OTLPEndpoint, cfg.ServiceName)
	if err != nil {
		return fmt.Errorf("setup tracing: %w", err)
	}
	defer shutdownTracing(context.Background())

	promRegistry := prometheus.NewRegistry()
	metrics := observability.NewMetrics(promRegistry)

	metricsMux := http.NewServeMux()
	metricsMux.Handle(cfg.MetricsPath, observability.Handler(promRegistry))
	metricsServer := &http.Server{
		Addr:    cfg.MetricsAddr,
		Handler: metricsMux,
	}

	go func() {
		slog.Info("metrics server starting", "addr", cfg.MetricsAddr, "path", cfg.MetricsPath)
		if err := metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("metrics server error", "error", err)
		}
	}()

	relayClient := client.New(cfg.BaseURL, cfg.APIKey, cfg.SystemKey, cfg.UserID, cfg.Timeout)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "new-api-mcp-server",
		Version: version,
	}, &mcp.ServerOptions{
		Instructions: "Available tool categories: Channel management (add, toggle, set priority, test channels), User management (list, toggle status, set quota), Token/group management (switch group), Provider listing (list providers with balance). API admin tools (api_ prefix) for channel/user/token/log CRUD. Relay/model tools for AI model inference (chat, image, video generation). Use descriptive queries to search for specific tools.",
		PageSize:     100,
	})

	// Register relay tools
	if cfg.APIKey != "" {
		relayDefs, err := openapipkg.Parse(embeddedSpecs.RelaySpec)
		if err != nil {
			return fmt.Errorf("parse relay spec: %w", err)
		}

		if err := registry.ValidateUniqueNames(relayDefs, ""); err != nil {
			return fmt.Errorf("relay tools: %w", err)
		}

		relayHandler := handler.New(relayClient, client.SourceRelay, metrics)
		count := registry.RegisterTools(server, relayDefs, registry.Options{
			EnabledGroups: cfg.RelayEnabledGroups,
			AllGroups:     cfg.RelayAllGroups,
			NamePrefix:    "",
		}, relayHandler.MakeHandler)

		slog.Info("registered relay tools", "count", count)
	} else {
		slog.Warn("NEW_API_KEY not set, relay tools disabled")
	}

	// Register API tools
	if cfg.SystemKey != "" && cfg.APIToolsEnabled {
		apiDefs, err := openapipkg.Parse(embeddedSpecs.APISpec)
		if err != nil {
			return fmt.Errorf("parse api spec: %w", err)
		}

		if err := registry.ValidateUniqueNames(apiDefs, "api_"); err != nil {
			return fmt.Errorf("api tools: %w", err)
		}

		apiHandler := handler.New(relayClient, client.SourceAPI, metrics)
		count := registry.RegisterTools(server, apiDefs, registry.Options{
			AllGroups:  true,
			NamePrefix: "api_",
		}, apiHandler.MakeHandler)

		slog.Info("registered api tools", "count", count)
	} else {
		slog.Info("API tools disabled")
	}

	// Register high-level tools
	if cfg.SystemKey != "" && cfg.APIToolsEnabled {
		highDefs := hightools.RegisterAll(relayClient, metrics)
		for _, def := range highDefs {
			tool := &mcp.Tool{
				Name:        def.Name,
				Description: def.Description,
				InputSchema: def.InputSchema,
			}
			server.AddTool(tool, def.Handler)
		}
		slog.Info("registered high-level tools", "count", len(highDefs))
	}

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)

		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()

		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("metrics server shutdown failed", "error", err)
		}

		slog.Info("graceful shutdown complete")
	}()

	// Run transport
	switch cfg.Transport {
	case "http":
		slog.Info("starting HTTP transport",
			"addr", cfg.HTTPAddr,
			"auth_enabled", cfg.HTTPAuthToken != "",
			"cors_origins", cfg.HTTPCORSOrigins,
			"max_body_size", cfg.HTTPMaxBodySize,
			"rate_limit_rps", cfg.RateLimitRPS,
			"rate_limit_burst", cfg.RateLimitBurst,
		)
		mcpHandler := mcp.NewStreamableHTTPHandler(
			func(r *http.Request) *mcp.Server { return server },
			nil,
		)
		var httpHandler http.Handler = mcpHandler
		if cfg.HTTPAuthToken != "" {
			httpHandler = authMiddleware(cfg.HTTPAuthToken, httpHandler)
		}
		httpHandler = corsMiddleware(cfg.HTTPCORSOrigins, httpHandler)
		httpHandler = maxBodyMiddleware(cfg.HTTPMaxBodySize, httpHandler)
		if cfg.RateLimitRPS > 0 {
			httpHandler = middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)(httpHandler)
			slog.Info("rate limiting enabled", "rps", cfg.RateLimitRPS, "burst", cfg.RateLimitBurst)
		}
		// Health check is outermost — checked before auth/rate-limit
		httpHandler = healthCheckMiddleware(cfg.BaseURL, httpHandler)
		httpServer := &http.Server{
			Addr:    cfg.HTTPAddr,
			Handler: httpHandler,
		}
		go func() {
			<-ctx.Done()
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer shutdownCancel()
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				slog.Error("http server shutdown failed", "error", err)
			}
		}()
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			return fmt.Errorf("http server: %w", err)
		}
	default:
		slog.Info("starting stdio transport")
		if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
			return fmt.Errorf("stdio server: %w", err)
		}
	}

	return nil
}

func authMiddleware(token string, next http.Handler) http.Handler {
	expected := "Bearer " + token
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != expected {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers to HTTP responses and handles preflight OPTIONS requests.
func corsMiddleware(origins string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origins)
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// maxBodyMiddleware limits the size of incoming request bodies to prevent abuse.
func maxBodyMiddleware(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}

// healthCheckMiddleware handles /healthz and /readyz endpoints for container
// orchestration. It wraps the MCP handler, allowing health checks to bypass
// auth and rate limiting.
func healthCheckMiddleware(baseURL string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		case "/readyz":
			w.Header().Set("Content-Type", "application/json")
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodHead, baseURL, nil)
			if err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
				return
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil || resp.StatusCode >= 500 {
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
				if resp != nil {
					resp.Body.Close()
				}
				return
			}
			resp.Body.Close()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		default:
			next.ServeHTTP(w, r)
		}
	})
}