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

var tracerAddChannelKeys = otel.Tracer("hightools.add_channel_keys")

// NewAddChannelKeysTool returns a ToolDef for appending API keys to an existing channel.
//
// The tool constructs the upstream request body and sends PUT /api/channel/
// with key_mode="append". Required: channel_id (integer), keys (string).
// Optional: multi_key_mode (string).
func NewAddChannelKeysTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "add_channel_keys",
		Description: "Append API keys to an existing channel. Sends a PUT request to /api/channel/ with key_mode='append'. Required: channel_id (integer), keys (string). Optional: multi_key_mode (string).",
		InputSchema: inputSchemaAddChannelKeys(),
		Handler:     handleAddChannelKeys(c, metrics),
	}
}

func inputSchemaAddChannelKeys() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"channel_id": map[string]any{
				"type":        "integer",
				"description": "Channel ID to append keys to",
			},
			"keys": map[string]any{
				"type":        "string",
				"description": "API keys to append, one per line (separated by newline)",
			},
			"multi_key_mode": map[string]any{
				"type":        "string",
				"description": "Optional: multi-key load balancing strategy, 'polling' or 'random'",
				"enum":        []any{"polling", "random"},
			},
		},
		"required": []any{"channel_id", "keys"},
	}
}

func handleAddChannelKeys(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerAddChannelKeys.Start(ctx, "add_channel_keys",
			trace.WithAttributes(attribute.String("tool", "add_channel_keys")),
		)
		defer span.End()

		start := time.Now()

		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultAddChannelKeys(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		// Validate required: channel_id
		channelIDRaw, ok := args["channel_id"]
		if !ok {
			return errorResultAddChannelKeys("required parameter 'channel_id' is missing"), nil
		}
		channelID, ok := toInt64(channelIDRaw)
		if !ok {
			return errorResultAddChannelKeys(fmt.Sprintf("channel_id must be an integer, got %T", channelIDRaw)), nil
		}

		// Validate required: keys
		keysRaw, ok := args["keys"]
		if !ok {
			return errorResultAddChannelKeys("required parameter 'keys' is missing"), nil
		}
		keys, ok := keysRaw.(string)
		if !ok {
			return errorResultAddChannelKeys(fmt.Sprintf("keys must be a string, got %T", keysRaw)), nil
		}

		// Split keys by newline and filter empty/whitespace-only lines
		keyLines := strings.Split(keys, "\n")
		var nonEmptyKeys []string
		for _, k := range keyLines {
			trimmed := strings.TrimSpace(k)
			if trimmed != "" {
				nonEmptyKeys = append(nonEmptyKeys, trimmed)
			}
		}
		if len(nonEmptyKeys) == 0 {
			return errorResultAddChannelKeys("keys must contain at least one non-empty API key"), nil
		}

		// Read optional: multi_key_mode
		var multiKeyMode string
		if multiKeyModeRaw, ok := args["multi_key_mode"]; ok {
			mkmStr, ok := multiKeyModeRaw.(string)
			if !ok {
				return errorResultAddChannelKeys(fmt.Sprintf("multi_key_mode must be a string, got %T", multiKeyModeRaw)), nil
			}
			if mkmStr != "polling" && mkmStr != "random" {
				return errorResultAddChannelKeys(fmt.Sprintf("multi_key_mode must be 'polling' or 'random', got %q", mkmStr)), nil
			}
			multiKeyMode = mkmStr
		}

		// Build request body
		body := map[string]any{
			"id":       channelID,
			"key_mode": "append",
			"key":      strings.Join(nonEmptyKeys, "\n"),
		}
		if multiKeyMode != "" {
			body["multi_key_mode"] = multiKeyMode
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return errorResultAddChannelKeys(fmt.Sprintf("marshal body: %v", err)), nil
		}

		// Call upstream
		upstreamStart := time.Now()
		resp, err := c.Do(ctx, client.SourceAPI, "PUT", "/api/channel/", nil, nil, bodyBytes)
		upstreamDuration := time.Since(upstreamStart)
		if err != nil {
			slog.ErrorContext(ctx, "upstream request failed",
				"tool", "add_channel_keys",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return errorResultAddChannelKeys(fmt.Sprintf("upstream error: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResultAddChannelKeys(fmt.Sprintf("read response: %v", err)), nil
		}

		toolDuration := time.Since(start)
		isError := resp.StatusCode < 200 || resp.StatusCode >= 300
		status := "success"
		if isError {
			status = "error"
		}

		// Record metrics
		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("add_channel_keys", status).Inc()
			metrics.ToolRequestDuration.WithLabelValues("add_channel_keys").Observe(toolDuration.Seconds())
			metrics.UpstreamRequestsTotal.WithLabelValues("PUT", "/api/channel/", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			metrics.UpstreamRequestDuration.WithLabelValues("PUT", "/api/channel/").Observe(upstreamDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "add_channel_keys",
			"channel_id", channelID,
			"key_count", len(nonEmptyKeys),
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

func errorResultAddChannelKeys(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}