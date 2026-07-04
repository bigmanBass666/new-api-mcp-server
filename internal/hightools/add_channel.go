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

var tracerAddChannel = otel.Tracer("hightools.add_channel")

// NewAddChannelTool returns a ToolDef for creating a new AI provider channel.
//
// The tool constructs the upstream request body with mode:"single" and
// sends POST /api/channel/. Required: name, type, key. Optional: models, group, priority.
func NewAddChannelTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "add_channel",
		Description: "Add a new AI provider channel. Sends a POST request to /api/channel/. Required: name (string), type (integer), key (string). Optional: models (string), group (string), priority (integer).",
		InputSchema: inputSchemaAddChannel(),
		Handler:    handleAddChannel(c, metrics),
	}
}

func inputSchemaAddChannel() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Channel name (e.g., 'Azure OpenAI East US')",
			},
			"type": map[string]any{
				"type":        "integer",
				"description": "Channel type code (1=OpenAI, 2=Midjourney, 3=Azure, 4=Ollama, 5=Midjourney+, 8=Custom)",
			},
			"key": map[string]any{
				"type":        "string",
				"description": "API key for this channel (e.g., 'sk-proj-...')",
			},
			"models": map[string]any{
				"type":        "string",
				"description": "Optional: comma-separated model list (e.g., 'gpt-4,gpt-3.5-turbo')",
			},
			"group": map[string]any{
				"type":        "string",
				"description": "Optional: channel group (e.g., 'default', 'vip')",
			},
			"priority": map[string]any{
				"type":        "integer",
				"description": "Optional: channel priority (higher = preferred in load balancing)",
			},
		},
		"required": []any{"name", "type", "key"},
	}
}

func handleAddChannel(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerAddChannel.Start(ctx, "add_channel",
			trace.WithAttributes(attribute.String("tool", "add_channel")),
		)
		defer span.End()

		start := time.Now()

		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultAddChannel(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		// Validate required: name
		nameRaw, ok := args["name"]
		if !ok {
			return errorResultAddChannel("required parameter 'name' is missing"), nil
		}
		name, ok := nameRaw.(string)
		if !ok {
			return errorResultAddChannel(fmt.Sprintf("name must be a string, got %T", nameRaw)), nil
		}

		// Validate required: type
		typeRaw, ok := args["type"]
		if !ok {
			return errorResultAddChannel("required parameter 'type' is missing"), nil
		}
		channelType, ok := toInt64(typeRaw)
		if !ok {
			return errorResultAddChannel(fmt.Sprintf("type must be an integer, got %T", typeRaw)), nil
		}

		// Validate required: key
		keyRaw, ok := args["key"]
		if !ok {
			return errorResultAddChannel("required parameter 'key' is missing"), nil
		}
		key, ok := keyRaw.(string)
		if !ok {
			return errorResultAddChannel(fmt.Sprintf("key must be a string, got %T", keyRaw)), nil
		}

		// Build channel object
		channel := map[string]any{
			"name": name,
			"type": channelType,
			"key":  key,
		}

		// Optional: models
		if modelsRaw, ok := args["models"]; ok {
			models, ok := modelsRaw.(string)
			if !ok {
				return errorResultAddChannel(fmt.Sprintf("models must be a string, got %T", modelsRaw)), nil
			}
			channel["models"] = models
		}

		// Optional: group
		if groupRaw, ok := args["group"]; ok {
			group, ok := groupRaw.(string)
			if !ok {
				return errorResultAddChannel(fmt.Sprintf("group must be a string, got %T", groupRaw)), nil
			}
			channel["group"] = group
		}

		// Optional: priority
		if priorityRaw, ok := args["priority"]; ok {
			priority, ok := toInt64(priorityRaw)
			if !ok {
				return errorResultAddChannel(fmt.Sprintf("priority must be an integer, got %T", priorityRaw)), nil
			}
			channel["priority"] = priority
		}

		// Construct request body
		body := map[string]any{
			"mode":    "single",
			"channel": channel,
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return errorResultAddChannel(fmt.Sprintf("marshal body: %v", err)), nil
		}

		// Call upstream
		upstreamStart := time.Now()
		resp, err := c.Do(ctx, client.SourceAPI, "POST", "/api/channel/", nil, nil, bodyBytes)
		upstreamDuration := time.Since(upstreamStart)
		if err != nil {
			slog.ErrorContext(ctx, "upstream request failed",
				"tool", "add_channel",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return errorResultAddChannel(fmt.Sprintf("upstream error: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResultAddChannel(fmt.Sprintf("read response: %v", err)), nil
		}

		toolDuration := time.Since(start)
		isError := resp.StatusCode < 200 || resp.StatusCode >= 300
		status := "success"
		if isError {
			status = "error"
		}

		// Record metrics
		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("add_channel", status).Inc()
			metrics.ToolRequestDuration.WithLabelValues("add_channel").Observe(toolDuration.Seconds())
			metrics.UpstreamRequestsTotal.WithLabelValues("POST", "/api/channel/", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			metrics.UpstreamRequestDuration.WithLabelValues("POST", "/api/channel/").Observe(upstreamDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "add_channel",
			"name", name,
			"type", channelType,
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

func errorResultAddChannel(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}