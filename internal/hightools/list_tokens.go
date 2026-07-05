package hightools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
	"github.com/QuantumNous/new-api-mcp-server/internal/observability"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracerListTokens = otel.Tracer("hightools.list_tokens")

// NewListTokensTool returns a ToolDef for listing all API tokens.
//
// The tool sends GET /api/token/ with optional pagination parameters.
// Optional: page (integer, default 1), page_size (integer, default 10).
func NewListTokensTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "list_tokens",
		Description: "List all API tokens with optional pagination. Sends a GET request to /api/token/. Optional: page (integer, default 1), page_size (integer, default 10).",
		InputSchema: inputSchemaListTokens(),
		Handler:    handleListTokens(c, metrics),
	}
}

func inputSchemaListTokens() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"page": map[string]any{
				"type":        "integer",
				"description": "Optional: page number (default 1)",
			},
			"page_size": map[string]any{
				"type":        "integer",
				"description": "Optional: number of tokens per page (default 10)",
			},
		},
		"required": []any{},
	}
}

func handleListTokens(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerListTokens.Start(ctx, "list_tokens",
			trace.WithAttributes(attribute.String("tool", "list_tokens")),
		)
		defer span.End()

		start := time.Now()

		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultListTokens(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		// Build query parameters
		queryParams := make(map[string]string)

		// Optional: page (default 1)
		page := int64(1)
		if pageRaw, ok := args["page"]; ok {
			pageVal, ok := toInt64(pageRaw)
			if !ok {
				return errorResultListTokens(fmt.Sprintf("page must be an integer, got %T", pageRaw)), nil
			}
			page = pageVal
		}
		queryParams["page"] = strconv.FormatInt(page, 10)

		// Optional: page_size (default 10)
		pageSize := int64(10)
		if pageSizeRaw, ok := args["page_size"]; ok {
			pageSizeVal, ok := toInt64(pageSizeRaw)
			if !ok {
				return errorResultListTokens(fmt.Sprintf("page_size must be an integer, got %T", pageSizeRaw)), nil
			}
			pageSize = pageSizeVal
		}
		queryParams["page_size"] = strconv.FormatInt(pageSize, 10)

		// Call upstream
		upstreamStart := time.Now()
		resp, err := c.Do(ctx, client.SourceAPI, "GET", "/api/token/", queryParams, nil, nil)
		upstreamDuration := time.Since(upstreamStart)
		if err != nil {
			slog.ErrorContext(ctx, "upstream request failed",
				"tool", "list_tokens",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return errorResultListTokens(fmt.Sprintf("upstream error: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResultListTokens(fmt.Sprintf("read response: %v", err)), nil
		}

		toolDuration := time.Since(start)
		isError := resp.StatusCode < 200 || resp.StatusCode >= 300
		status := "success"
		if isError {
			status = "error"
		}

		// Record metrics
		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("list_tokens", status).Inc()
			metrics.ToolRequestDuration.WithLabelValues("list_tokens").Observe(toolDuration.Seconds())
			metrics.UpstreamRequestsTotal.WithLabelValues("GET", "/api/token/", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			metrics.UpstreamRequestDuration.WithLabelValues("GET", "/api/token/").Observe(upstreamDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "list_tokens",
			"page", page,
			"page_size", pageSize,
			"status_code", resp.StatusCode,
			"duration_ms", toolDuration.Milliseconds(),
		)

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(respBody)},
			},
		}

		if isError {
			result.IsError = true
			span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
		}

		return result, nil
	}
}

func errorResultListTokens(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}