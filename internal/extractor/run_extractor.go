//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/QuantumNous/new-api-mcp-server/internal/extractor"
)

func main() {
	cfg := extractor.ExtractorConfig{
		BaseURL:   "http://localhost:4050",
		SystemKey: "5tXB6g4BYmuLLQqRx5gGCb59OZBYFQ==",
		UserID:    "1",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ext := extractor.NewExtractor(cfg)
	result, err := ext.Extract(ctx, "openapi/api.json")
	if err != nil {
		log.Fatalf("extract failed: %v", err)
	}

	fmt.Printf("Extracted %d endpoints, %d components\n",
		len(result.Endpoints), len(result.Components))

	merger := extractor.NewMerger("openapi/api.json")
	if err := merger.Merge(result, "openapi/api.new.json"); err != nil {
		log.Fatalf("merge failed: %v", err)
	}

	// Verify output
	data, err := os.ReadFile("openapi/api.new.json")
	if err != nil {
		log.Fatalf("read output: %v", err)
	}
	fmt.Printf("Output size: %d bytes\n", len(data))
	fmt.Println("Extraction complete! Output written to openapi/api.new.json")
}