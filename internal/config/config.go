package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BaseURL   string
	APIKey    string
	SystemKey string
	Timeout   time.Duration

	Transport     string
	HTTPAddr      string
	HTTPAuthToken string

	RelayEnabledGroups []string
	RelayAllGroups     bool
	APIToolsEnabled    bool

	LogLevel          string
	LogFormat         string
	LogConsoleEnabled bool

	OTLPEndpoint string
	ServiceName  string

	MetricsAddr string
	MetricsPath string

	UserID             string
	HTTPCORSOrigins    string
	HTTPMaxBodySize    int64
	RateLimitRPS      int           // 速率限制 RPS，0=不限速
	RateLimitBurst    int           // 速率限制 Burst
	ShutdownTimeout   time.Duration // 优雅关闭超时时间
}

func Load() (*Config, error) {
	baseURL := os.Getenv("NEW_API_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("NEW_API_BASE_URL is required")
	}

	cfg := &Config{
		BaseURL:           baseURL,
		APIKey:            os.Getenv("NEW_API_KEY"),
		SystemKey:         os.Getenv("NEW_API_SYSTEM_KEY"),
		Timeout:           parseDuration("NEW_API_TIMEOUT", 30*time.Second),
		Transport:         envOrDefault("MCP_TRANSPORT", "stdio"),
		HTTPAddr:          envOrDefault("MCP_HTTP_ADDR", ":8080"),
		HTTPAuthToken:     os.Getenv("MCP_HTTP_AUTH_TOKEN"),
		APIToolsEnabled:   os.Getenv("MCP_API_TOOLS_ENABLED") == "true",
		LogLevel:          envOrDefault("MCP_LOG_LEVEL", "info"),
		LogFormat:         envOrDefault("MCP_LOG_FORMAT", "json"),
		LogConsoleEnabled: os.Getenv("MCP_LOG_CONSOLE_ENABLED") != "false",
		OTLPEndpoint:      os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		ServiceName:       envOrDefault("OTEL_SERVICE_NAME", "new-api-mcp-server"),
		MetricsAddr:       envOrDefault("MCP_METRICS_ADDR", ":9090"),
		MetricsPath:       envOrDefault("MCP_METRICS_PATH", "/metrics"),
		UserID:            envOrDefault("MCP_USER_ID", "1"),
		HTTPCORSOrigins:   envOrDefault("MCP_HTTP_CORS_ORIGINS", "*"),
		HTTPMaxBodySize:   10 * 1024 * 1024, // 10MB default
		RateLimitRPS:      envOrDefaultInt("MCP_RATE_LIMIT_RPS", 0),
		RateLimitBurst:    envOrDefaultInt("MCP_RATE_LIMIT_BURST", 0),
		ShutdownTimeout:   parseDuration("MCP_SHUTDOWN_TIMEOUT", 15*time.Second),
	}

	if groups := os.Getenv("MCP_RELAY_ENABLED_GROUPS"); groups != "" {
		if groups == "all" {
			cfg.RelayAllGroups = true
		} else {
			cfg.RelayEnabledGroups = strings.Split(groups, ",")
		}
	}

	// Validate BaseURL format
	if u, err := url.Parse(cfg.BaseURL); err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid base URL: %s", cfg.BaseURL)
	}

	// Validate transport
	if cfg.Transport != "stdio" && cfg.Transport != "http" {
		return nil, fmt.Errorf("unsupported transport: %s (must be 'stdio' or 'http')", cfg.Transport)
	}

	// Validate timeout minimum
	if cfg.Timeout < time.Second {
		return nil, fmt.Errorf("timeout too low: %v (minimum 1s)", cfg.Timeout)
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fallback
		}
		return n
	}
	return fallback
}

func parseDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}