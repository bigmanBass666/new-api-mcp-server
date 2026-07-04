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

var tracerSwitchGroup = otel.Tracer("hightools.switch_group")

// NewSwitchGroupTool returns a ToolDef for switching a token's group.
//
// The tool calls PUT /api/token/ with a JSON body containing the token ID
// and the new group name. It validates that token_id is an integer and
// group is a non-empty string before making the upstream request.
func NewSwitchGroupTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "switch_group",
		Description: "Switch a token to a different group. Sends a PUT request to /api/token/ with the token ID and new group name.",
		InputSchema: inputSchemaSwitchGroup(),
		Handler:    handleSwitchGroup(c, metrics),
	}
}

func inputSchemaSwitchGroup() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"token_id": map[string]any{
				"type":        "integer",
				"description": "Token ID to switch group for",
			},
			"group": map[string]any{
				"type":        "string",
				"description": "New group name to assign to the token (e.g., 'default', 'vip', 'free')",
			},
		},
		"required": []any{"token_id", "group"},
	}
}

func handleSwitchGroup(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerSwitchGroup.Start(ctx, "switch_group",
			trace.WithAttributes(
				attribute.String("tool", "switch_group"),
			),
		)
		defer span.End()

		start := time.Now()

		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultSwitchGroup(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		// Validate and extract token_id
		idRaw, ok := args["token_id"]
		if !ok {
			return errorResultSwitchGroup("missing required argument: token_id"), nil
		}
		tokenID, ok := toInt64(idRaw)
		if !ok {
			return errorResultSwitchGroup(fmt.Sprintf("token_id must be an integer, got %T", idRaw)), nil
		}

		// Validate and extract group
		groupRaw, ok := args["group"]
		if !ok {
			return errorResultSwitchGroup("missing required argument: group"), nil
		}
		group, ok := groupRaw.(string)
		if !ok {
			return errorResultSwitchGroup(fmt.Sprintf("group must be a string, got %T", groupRaw)), nil
		}
		if strings.TrimSpace(group) == "" {
			return errorResultSwitchGroup("group must not be empty"), nil
		}

		// Build request body
		body := map[string]any{
			"id":    tokenID,
			"group": group,
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return errorResultSwitchGroup(fmt.Sprintf("marshal body: %v", err)), nil
		}

		// Call upstream PUT /api/token/
		upstreamStart := time.Now()
		resp, err := c.Do(ctx, client.SourceAPI, "PUT", "/api/token/", nil, nil, bodyBytes)
		upstreamDuration := time.Since(upstreamStart)
		if err != nil {
			slog.ErrorContext(ctx, "upstream request failed",
				"tool", "switch_group",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return errorResultSwitchGroup(fmt.Sprintf("upstream error: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResultSwitchGroup(fmt.Sprintf("read response: %v", err)), nil
		}

		toolDuration := time.Since(start)
		isError := resp.StatusCode < 200 || resp.StatusCode >= 300
		status := "success"
		if isError {
			status = "error"
		}

		// Record metrics
		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("switch_group", status).Inc()
			metrics.ToolRequestDuration.WithLabelValues("switch_group").Observe(toolDuration.Seconds())
			metrics.UpstreamRequestsTotal.WithLabelValues("PUT", "/api/token/", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			metrics.UpstreamRequestDuration.WithLabelValues("PUT", "/api/token/").Observe(upstreamDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "switch_group",
			"token_id", tokenID,
			"group", group,
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

func errorResultSwitchGroup(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}