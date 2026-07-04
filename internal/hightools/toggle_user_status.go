package hightools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
	"github.com/QuantumNous/new-api-mcp-server/internal/observability"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracerToggleUserStatus = otel.Tracer("hightools.toggle_user_status")

// NewToggleUserStatusTool returns a ToolDef for toggling a user's enabled state.
//
// The tool calls POST /api/user/manage with a JSON body containing the user ID
// and the action ("enable" or "disable"). It validates that id is an integer
// and enabled is a boolean before making the upstream request.
func NewToggleUserStatusTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "toggle_user_status",
		Description: "Toggle (enable/disable) a user by ID. Sends a POST request to /api/user/manage with the user ID and action ('enable' or 'disable').",
		InputSchema: inputSchemaToggleUserStatus(),
		Handler:    handleToggleUserStatus(c, metrics),
	}
}

func inputSchemaToggleUserStatus() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "integer",
				"description": "User ID",
			},
			"enabled": map[string]any{
				"type":        "boolean",
				"description": "New enabled state (true=enable, false=disable)",
			},
		},
		"required": []any{"id", "enabled"},
	}
}

func handleToggleUserStatus(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerToggleUserStatus.Start(ctx, "toggle_user_status",
			trace.WithAttributes(
				attribute.String("tool", "toggle_user_status"),
			),
		)
		defer span.End()

		start := time.Now()

		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultToggleUserStatus(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		// Validate and extract id
		idRaw, ok := args["id"]
		if !ok {
			return errorResultToggleUserStatus("missing required argument: id"), nil
		}
		id, ok := toInt64(idRaw)
		if !ok {
			return errorResultToggleUserStatus(fmt.Sprintf("id must be an integer, got %T", idRaw)), nil
		}

		// Validate and extract enabled
		enabledRaw, ok := args["enabled"]
		if !ok {
			return errorResultToggleUserStatus("missing required argument: enabled"), nil
		}
		enabled, ok := enabledRaw.(bool)
		if !ok {
			return errorResultToggleUserStatus(fmt.Sprintf("enabled must be a boolean, got %T", enabledRaw)), nil
		}

		// Map enabled bool to action string
		action := "disable"
		if enabled {
			action = "enable"
		}

		// Build request body with id and action
		body := map[string]any{"id": id, "action": action}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return errorResultToggleUserStatus(fmt.Sprintf("marshal body: %v", err)), nil
		}

		// Call upstream POST /api/user/manage
		upstreamStart := time.Now()
		resp, err := c.Do(ctx, client.SourceAPI, "POST", "/api/user/manage", nil, nil, bodyBytes)
		upstreamDuration := time.Since(upstreamStart)
		if err != nil {
			slog.ErrorContext(ctx, "upstream request failed",
				"tool", "toggle_user_status",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return errorResultToggleUserStatus(fmt.Sprintf("upstream error: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResultToggleUserStatus(fmt.Sprintf("read response: %v", err)), nil
		}

		toolDuration := time.Since(start)
		isError := resp.StatusCode < 200 || resp.StatusCode >= 300
		status := "success"
		if isError {
			status = "error"
		}

		// Record metrics
		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("toggle_user_status", status).Inc()
			metrics.ToolRequestDuration.WithLabelValues("toggle_user_status").Observe(toolDuration.Seconds())
			metrics.UpstreamRequestsTotal.WithLabelValues("POST", "/api/user/manage", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			metrics.UpstreamRequestDuration.WithLabelValues("POST", "/api/user/manage").Observe(upstreamDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "toggle_user_status",
			"user_id", id,
			"action", action,
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

func errorResultToggleUserStatus(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}