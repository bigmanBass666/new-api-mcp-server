package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/QuantumNous/new-api-mcp-server/internal/extractor"
)

func main() {
	var (
		baseURL    = flag.String("base-url", envOr("NEW_API_BASE_URL", "http://localhost:4050"), "New API base URL")
		systemKey  = flag.String("system-key", envOr("NEW_API_SYSTEM_KEY", ""), "Admin access token")
		userID     = flag.String("user-id", envOr("NEW_API_USER_ID", "2"), "User ID for New-Api-User header")
		skeleton   = flag.String("skeleton", "openapi/api.json", "Path to skeleton spec")
		output     = flag.String("output", "openapi/api.json", "Output path for updated spec")
		verbose    = flag.Bool("verbose", false, "Enable debug logging")
	)
	flag.Parse()

	if *verbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}

	if *systemKey == "" {
		fmt.Fprintln(os.Stderr, "error: NEW_API_SYSTEM_KEY is required (set env var or pass --system-key)")
		os.Exit(1)
	}

	cfg := extractor.ExtractorConfig{
		BaseURL:   *baseURL,
		SystemKey: *systemKey,
		UserID:    *userID,
	}

	ext := extractor.NewExtractor(cfg)
	ctx := context.Background()

	fmt.Fprintf(os.Stderr, "Extracting from %s (skeleton: %s)...\n", *baseURL, *skeleton)
	result, err := ext.Extract(ctx, *skeleton)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract failed: %v\n", err)
		os.Exit(1)
	}

	merger := extractor.NewMerger(*skeleton)
	if err := merger.Merge(result, *output); err != nil {
		fmt.Fprintf(os.Stderr, "merge failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Done. Updated spec written to %s\n", *output)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
