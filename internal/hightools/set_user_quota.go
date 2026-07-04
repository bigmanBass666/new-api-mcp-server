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

var tracerSetUserQuota = otel.Tracer("hightools.set_user_quota")

// NewSetUserQuotaTool returns a ToolDef for setting a user's quota.
//
// The tool calls PUT /api/user/{id} with a JSON body containing the new
// quota value. It validates that quota is a non-negative integer before
// making the upstream request.
func NewSetUserQuotaTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "set_user_quota",
		Description: "Set the quota for a user by ID. Quota controls the total tokens/credits available to the user.",
		InputSchema: inputSchemaSetUserQuota(),
		Handler:     handleSetUserQuota(c, metrics),
	}
}

func inputSchemaSetUserQuota() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "integer",
				"description": "User ID",
			},
			"quota": map[string]any{
				"type":        "integer",
				"description": "New quota value for the user (must be a non-negative integer)",
			},
		},
		"required": []any{"id", "quota"},
	}
}

func handleSetUserQuota(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerSetUserQuota.Start(ctx, "set_user_quota",
			trace.WithAttributes(
				attribute.String("tool", "set_user_quota"),
			),
		)
		defer span.End()

		start := time.Now()

		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultSetUserQuota(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		// Validate and extract id
		idRaw, ok := args["id"]
		if !ok {
			return errorResultSetUserQuota("missing required argument: id"), nil
		}
		id, ok := toInt64(idRaw)
		if !ok {
			return errorResultSetUserQuota(fmt.Sprintf("id must be an integer, got %T", idRaw)), nil
		}

		// Validate and extract quota
		quotaRaw, ok := args["quota"]
		if !ok {
			return errorResultSetUserQuota("missing required argument: quota"), nil
		}
		quota, ok := toInt64(quotaRaw)
		if !ok {
			return errorResultSetUserQuota(fmt.Sprintf("quota must be an integer, got %T", quotaRaw)), nil
		}
		if quota < 0 {
			return errorResultSetUserQuota(fmt.Sprintf("quota must be a non-negative integer, got %d", quota)), nil
		}

		// Build path with id substituted
		path := strings.ReplaceAll("/api/user/{id}", "{id}", fmt.Sprintf("%d", id))

		// Build request body
		body := map[string]any{"quota": quota}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return errorResultSetUserQuota(fmt.Sprintf("marshal body: %v", err)), nil
		}

		// Call upstream
		upstreamStart := time.Now()
		resp, err := c.Do(ctx, client.SourceAPI, "PUT", path, nil, nil, bodyBytes)
		upstreamDuration := time.Since(upstreamStart)
		if err != nil {
			slog.ErrorContext(ctx, "upstream request failed",
				"tool", "set_user_quota",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return errorResultSetUserQuota(fmt.Sprintf("upstream error: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResultSetUserQuota(fmt.Sprintf("read response: %v", err)), nil
		}

		toolDuration := time.Since(start)
		isError := resp.StatusCode < 200 || resp.StatusCode >= 300
		status := "success"
		if isError {
			status = "error"
		}

		// Record metrics
		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("set_user_quota", status).Inc()
			metrics.ToolRequestDuration.WithLabelValues("set_user_quota").Observe(toolDuration.Seconds())
			metrics.UpstreamRequestsTotal.WithLabelValues("PUT", "/api/user/{id}", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			metrics.UpstreamRequestDuration.WithLabelValues("PUT", "/api/user/{id}").Observe(upstreamDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "set_user_quota",
			"user_id", id,
			"quota", quota,
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

func errorResultSetUserQuota(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}