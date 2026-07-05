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

var tracerCreateToken = otel.Tracer("hightools.create_token")

// NewCreateTokenTool returns a ToolDef for creating a new API token.
//
// The tool constructs the request body and sends POST /api/token/.
// Required: name. Optional: remain_quota, unlimited_quota, group,
// model_limits_enabled, model_limits.
func NewCreateTokenTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "create_token",
		Description: "Create a new API token. Sends a POST request to /api/token/. Required: name (string). Optional: remain_quota (integer), unlimited_quota (boolean, default false), group (string), model_limits_enabled (boolean), model_limits (string, comma-separated).",
		InputSchema: inputSchemaCreateToken(),
		Handler:    handleCreateToken(c, metrics),
	}
}

func inputSchemaCreateToken() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Token name",
			},
			"remain_quota": map[string]any{
				"type":        "integer",
				"description": "Optional: quota limit for this token",
			},
			"unlimited_quota": map[string]any{
				"type":        "boolean",
				"description": "Optional: whether the token has unlimited quota (default false)",
			},
			"group": map[string]any{
				"type":        "string",
				"description": "Optional: token group assignment",
			},
			"model_limits_enabled": map[string]any{
				"type":        "boolean",
				"description": "Optional: enable model restriction on this token",
			},
			"model_limits": map[string]any{
				"type":        "string",
				"description": "Optional: comma-separated model names allowed for this token",
			},
		},
		"required": []any{"name"},
	}
}

func handleCreateToken(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerCreateToken.Start(ctx, "create_token",
			trace.WithAttributes(attribute.String("tool", "create_token")),
		)
		defer span.End()

		start := time.Now()

		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultCreateToken(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		// Validate required: name
		nameRaw, ok := args["name"]
		if !ok {
			return errorResultCreateToken("required parameter 'name' is missing"), nil
		}
		name, ok := nameRaw.(string)
		if !ok {
			return errorResultCreateToken(fmt.Sprintf("name must be a string, got %T", nameRaw)), nil
		}

		// Build token object
		tokenObj := map[string]any{
			"name": name,
		}

		// Optional: remain_quota
		if remainQuotaRaw, ok := args["remain_quota"]; ok {
			remainQuota, ok := toInt64(remainQuotaRaw)
			if !ok {
				return errorResultCreateToken(fmt.Sprintf("remain_quota must be an integer, got %T", remainQuotaRaw)), nil
			}
			tokenObj["remain_quota"] = remainQuota
		}

		// Optional: unlimited_quota (default false)
		if unlimitedRaw, ok := args["unlimited_quota"]; ok {
			unlimited, ok := unlimitedRaw.(bool)
			if !ok {
				return errorResultCreateToken(fmt.Sprintf("unlimited_quota must be a boolean, got %T", unlimitedRaw)), nil
			}
			tokenObj["unlimited_quota"] = unlimited
		}

		// Optional: group
		if groupRaw, ok := args["group"]; ok {
			group, ok := groupRaw.(string)
			if !ok {
				return errorResultCreateToken(fmt.Sprintf("group must be a string, got %T", groupRaw)), nil
			}
			tokenObj["group"] = group
		}

		// Optional: model_limits_enabled
		if modelLimitsEnabledRaw, ok := args["model_limits_enabled"]; ok {
			modelLimitsEnabled, ok := modelLimitsEnabledRaw.(bool)
			if !ok {
				return errorResultCreateToken(fmt.Sprintf("model_limits_enabled must be a boolean, got %T", modelLimitsEnabledRaw)), nil
			}
			tokenObj["model_limits_enabled"] = modelLimitsEnabled
		}

		// Optional: model_limits
		if modelLimitsRaw, ok := args["model_limits"]; ok {
			modelLimits, ok := modelLimitsRaw.(string)
			if !ok {
				return errorResultCreateToken(fmt.Sprintf("model_limits must be a string, got %T", modelLimitsRaw)), nil
			}
			tokenObj["model_limits"] = modelLimits
		}

		bodyBytes, err := json.Marshal(tokenObj)
		if err != nil {
			return errorResultCreateToken(fmt.Sprintf("marshal body: %v", err)), nil
		}

		// Call upstream
		upstreamStart := time.Now()
		resp, err := c.Do(ctx, client.SourceAPI, "POST", "/api/token/", nil, nil, bodyBytes)
		upstreamDuration := time.Since(upstreamStart)
		if err != nil {
			slog.ErrorContext(ctx, "upstream request failed",
				"tool", "create_token",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return errorResultCreateToken(fmt.Sprintf("upstream error: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResultCreateToken(fmt.Sprintf("read response: %v", err)), nil
		}

		toolDuration := time.Since(start)
		isError := resp.StatusCode < 200 || resp.StatusCode >= 300
		status := "success"
		if isError {
			status = "error"
		}

		// Record metrics
		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("create_token", status).Inc()
			metrics.ToolRequestDuration.WithLabelValues("create_token").Observe(toolDuration.Seconds())
			metrics.UpstreamRequestsTotal.WithLabelValues("POST", "/api/token/", fmt.Sprintf("%d", resp.StatusCode)).Inc()
			metrics.UpstreamRequestDuration.WithLabelValues("POST", "/api/token/").Observe(upstreamDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "create_token",
			"name", name,
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

func errorResultCreateToken(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}