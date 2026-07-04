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

var tracerSetChannelPriority = otel.Tracer("hightools.set_channel_priority")

// NewSetChannelPriorityTool returns a ToolDef for setting a channel's priority.
//
// The tool calls PUT /api/channel/ with a JSON body containing the channel id
// and the new priority value. It validates that both id and priority are
// integers before making the upstream request.
func NewSetChannelPriorityTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "set_channel_priority",
		Description: "Set the priority of a channel by ID. Priority controls request distribution weight — higher values take precedence.",
		InputSchema: inputSchemaSetChannelPriority(),
		Handler:    handleSetChannelPriority(c, metrics),
	}
}

func inputSchemaSetChannelPriority() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "integer",
				"description": "Channel ID",
			},
			"priority": map[string]any{
				"type":        "integer",
				"description": "New priority value for the channel",
			},
		},
		"required": []any{"id", "priority"},
	}
}

func handleSetChannelPriority(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerSetChannelPriority.Start(ctx, "set_channel_priority",
			trace.WithAttributes(
				attribute.String("tool", "set_channel_priority"),
			),
		)
		defer span.End()

		start := time.Now()

		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultSetChannelPriority(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		// Validate and extract id
		idRaw, ok := args["id"]
		if !ok {
			return errorResultSetChannelPriority("missing required argument: id"), nil
		}
		id, ok := toInt64(idRaw)
		if !ok {
			return errorResultSetChannelPriority(fmt.Sprintf("id must be an integer, got %T", idRaw)), nil
		}

		// Validate and extract priority
		priorityRaw, ok := args["priority"]
		if !ok {
			return errorResultSetChannelPriority("missing required argument: priority"), nil
		}
		priority, ok := toInt64(priorityRaw)
		if !ok {
			return errorResultSetChannelPriority(fmt.Sprintf("priority must be an integer, got %T", priorityRaw)), nil
		}

		// Build request body — id goes in body, not URL path
		body := map[string]any{"id": id, "priority": priority}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return errorResultSetChannelPriority(fmt.Sprintf("marshal body: %v", err)), nil
		}

		// Call upstream
		upstreamStart := time.Now()
		resp, err := c.Do(ctx, client.SourceAPI, "PUT", "/api/channel/", nil, nil, bodyBytes)
		upstreamDuration := time.Since(upstreamStart)
		if err != nil {
			slog.ErrorContext(ctx, "upstream request failed",
				"tool", "set_channel_priority",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return errorResultSetChannelPriority(fmt.Sprintf("upstream error: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResultSetChannelPriority(fmt.Sprintf("read response: %v", err)), nil
		}

		toolDuration := time.Since(start)
		isError := resp.StatusCode < 200 || resp.StatusCode >= 300
		status := "success"
		if isError {
			status = "error"
		}

		// Record metrics
		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("set_channel_priority", status).Inc()
			metrics.ToolRequestDuration.WithLabelValues("set_channel_priority").Observe(toolDuration.Seconds())
			metrics.UpstreamRequestsTotal.WithLabelValues("PUT", "/api/channel/", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			metrics.UpstreamRequestDuration.WithLabelValues("PUT", "/api/channel/").Observe(upstreamDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "set_channel_priority",
			"channel_id", id,
			"priority", priority,
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

func errorResultSetChannelPriority(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}