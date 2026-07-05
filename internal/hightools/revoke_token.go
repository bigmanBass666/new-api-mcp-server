package hightools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
	"github.com/QuantumNous/new-api-mcp-server/internal/observability"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracerRevokeToken = otel.Tracer("hightools.revoke_token")

// NewRevokeTokenTool returns a ToolDef for revoking (deleting) an API token.
//
// The tool sends DELETE /api/token/{id} with the token ID substituted into
// the URL path. Only the id parameter is required.
func NewRevokeTokenTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "revoke_token",
		Description: "Revoke (delete) an API token by ID. Sends a DELETE request to /api/token/{id}.",
		InputSchema: inputSchemaRevokeToken(),
		Handler:    handleRevokeToken(c, metrics),
	}
}

func inputSchemaRevokeToken() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "integer",
				"description": "Token ID to revoke",
			},
		},
		"required": []any{"id"},
	}
}

func handleRevokeToken(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerRevokeToken.Start(ctx, "revoke_token",
			trace.WithAttributes(attribute.String("tool", "revoke_token")),
		)
		defer span.End()

		start := time.Now()

		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultRevokeToken(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		// Validate and extract id
		idRaw, ok := args["id"]
		if !ok {
			return errorResultRevokeToken("missing required argument: id"), nil
		}
		id, ok := toInt64(idRaw)
		if !ok {
			return errorResultRevokeToken(fmt.Sprintf("id must be an integer, got %T", idRaw)), nil
		}

		// Build path with id substituted
		path := strings.ReplaceAll("/api/token/{id}", "{id}", fmt.Sprintf("%d", id))

		// Call upstream
		upstreamStart := time.Now()
		resp, err := c.Do(ctx, client.SourceAPI, "DELETE", path, nil, nil, nil)
		upstreamDuration := time.Since(upstreamStart)
		if err != nil {
			slog.ErrorContext(ctx, "upstream request failed",
				"tool", "revoke_token",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return errorResultRevokeToken(fmt.Sprintf("upstream error: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResultRevokeToken(fmt.Sprintf("read response: %v", err)), nil
		}

		toolDuration := time.Since(start)
		isError := resp.StatusCode < 200 || resp.StatusCode >= 300
		status := "success"
		if isError {
			status = "error"
		}

		// Record metrics
		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("revoke_token", status).Inc()
			metrics.ToolRequestDuration.WithLabelValues("revoke_token").Observe(toolDuration.Seconds())
			metrics.UpstreamRequestsTotal.WithLabelValues("DELETE", "/api/token/{id}", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			metrics.UpstreamRequestDuration.WithLabelValues("DELETE", "/api/token/{id}").Observe(upstreamDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "revoke_token",
			"token_id", id,
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

func errorResultRevokeToken(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}