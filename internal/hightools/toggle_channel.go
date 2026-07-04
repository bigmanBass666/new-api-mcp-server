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

var tracerToggleChannel = otel.Tracer("hightools.toggle_channel")

// NewToggleChannelTool returns a ToolDef for toggling a channel's enabled state.
//
// The tool calls POST /api/channel/{id}/status with a JSON body containing
// status=1 (enabled) or status=2 (manually disabled). It validates that id
// is an integer and enabled is a boolean before making the upstream request.
func NewToggleChannelTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "toggle_channel",
		Description: "Toggle (enable/disable) a channel by ID. Sends a POST request to /api/channel/{id}/status with the status code.",
		InputSchema: inputSchemaToggleChannel(),
		Handler:    handleToggleChannel(c, metrics),
	}
}

func inputSchemaToggleChannel() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "integer",
				"description": "Channel ID",
			},
			"enabled": map[string]any{
				"type":        "boolean",
				"description": "New enabled state (true=enable, false=disable)",
			},
		},
		"required": []any{"id", "enabled"},
	}
}

func handleToggleChannel(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerToggleChannel.Start(ctx, "toggle_channel",
			trace.WithAttributes(
				attribute.String("tool", "toggle_channel"),
			),
		)
		defer span.End()

		start := time.Now()

		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultToggleChannel(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		// Validate and extract id
		idRaw, ok := args["id"]
		if !ok {
			return errorResultToggleChannel("missing required argument: id"), nil
		}
		id, ok := toInt64(idRaw)
		if !ok {
			return errorResultToggleChannel(fmt.Sprintf("id must be an integer, got %T", idRaw)), nil
		}

		// Validate and extract enabled
		enabledRaw, ok := args["enabled"]
		if !ok {
			return errorResultToggleChannel("missing required argument: enabled"), nil
		}
		enabled, ok := enabledRaw.(bool)
		if !ok {
			return errorResultToggleChannel(fmt.Sprintf("enabled must be a boolean, got %T", enabledRaw)), nil
		}

		// Build path with id substituted
		path := strings.ReplaceAll("/api/channel/{id}/status", "{id}", fmt.Sprintf("%d", id))

		// Build request body — map enabled bool to upstream status integer.
		// ChannelStatusEnabled = 1, ChannelStatusManuallyDisabled = 2
		statusValue := 2
		if enabled {
			statusValue = 1
		}
		body := map[string]any{"status": statusValue}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return errorResultToggleChannel(fmt.Sprintf("marshal body: %v", err)), nil
		}

		// Call upstream
		upstreamStart := time.Now()
		resp, err := c.Do(ctx, client.SourceAPI, "POST", path, nil, nil, bodyBytes)
		upstreamDuration := time.Since(upstreamStart)
		if err != nil {
			slog.ErrorContext(ctx, "upstream request failed",
				"tool", "toggle_channel",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return errorResultToggleChannel(fmt.Sprintf("upstream error: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResultToggleChannel(fmt.Sprintf("read response: %v", err)), nil
		}

		toolDuration := time.Since(start)
		isError := resp.StatusCode < 200 || resp.StatusCode >= 300
		status := "success"
		if isError {
			status = "error"
		}

		// Record metrics
		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("toggle_channel", status).Inc()
			metrics.ToolRequestDuration.WithLabelValues("toggle_channel").Observe(toolDuration.Seconds())
			metrics.UpstreamRequestsTotal.WithLabelValues("POST", "/api/channel/{id}/status", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			metrics.UpstreamRequestDuration.WithLabelValues("POST", "/api/channel/{id}/status").Observe(upstreamDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "toggle_channel",
			"channel_id", id,
			"enabled", enabled,
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

func errorResultToggleChannel(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}