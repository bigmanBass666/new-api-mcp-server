package hightools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
	"github.com/QuantumNous/new-api-mcp-server/internal/observability"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracerListProviders = otel.Tracer("hightools.list_providers")

// Channel represents a single channel from the upstream GET /api/channel/ response.
type Channel struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	Status   int    `json:"status"`
	Models   string `json:"models"`
	Groups   string `json:"groups"`
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
	BaseURL  string `json:"base_url"`
	Tag      string `json:"tag"`
}

// listChannelsResponse is the upstream response wrapper for GET /api/channel/.
type listChannelsResponse struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Data    *listData     `json:"data"`
}

type listData struct {
	Items      []Channel    `json:"items"`
	Total      int          `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TypeCounts map[string]int64 `json:"type_counts"`
}

// NewListProvidersTool creates a tool that lists all channels grouped by their
// group assignments. The output is a structured text view with channel ID,
// name, status, models, and priority for each group.
func NewListProvidersTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "list_providers",
		Description: "List all channels grouped by their group assignments. Returns a structured text view with channel ID, name, status, models, and priority for each group.",
		InputSchema: inputSchemaListProviders(),
		Handler:    handleListProviders(c, metrics),
	}
}

func inputSchemaListProviders() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"group": map[string]any{
				"type":        "string",
				"description": "Optional: filter by group name (e.g., 'default', 'vip'). Only shows channels belonging to this group.",
			},
			"status": map[string]any{
				"type":        "integer",
				"description": "Optional: filter by status (1=enabled, other=disabled).",
			},
		},
	}
}

func handleListProviders(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerListProviders.Start(ctx, "list_providers",
			trace.WithAttributes(attribute.String("tool", "list_providers")),
		)
		defer span.End()

		start := time.Now()

		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultListProviders(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		// Build query params — pass group and status upstream so it filters at source
		queryParams := map[string]string{
			"page_size": "100",
		}

		if groupRaw, ok := args["group"]; ok {
			group, ok := groupRaw.(string)
			if !ok {
				return errorResultListProviders(fmt.Sprintf("group must be a string, got %T", groupRaw)), nil
			}
			queryParams["group"] = group
		}

		if statusRaw, ok := args["status"]; ok {
			status, ok := toInt64(statusRaw)
			if !ok {
				return errorResultListProviders(fmt.Sprintf("status must be an integer, got %T", statusRaw)), nil
			}
			queryParams["status"] = fmt.Sprintf("%d", status)
		}

		// Call upstream API
		upstreamStart := time.Now()
		resp, err := c.Do(ctx, client.SourceAPI, "GET", "/api/channel/", queryParams, nil, nil)
		upstreamDuration := time.Since(upstreamStart)
		if err != nil {
			slog.ErrorContext(ctx, "upstream request failed",
				"tool", "list_providers",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return errorResultListProviders(fmt.Sprintf("upstream error: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResultListProviders(fmt.Sprintf("read response: %v", err)), nil
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return errorResultListProviders(fmt.Sprintf("upstream returned status %d: %s", resp.StatusCode, string(respBody))), nil
		}

		// Parse upstream JSON response
		var listResp listChannelsResponse
		if err := json.Unmarshal(respBody, &listResp); err != nil {
			return errorResultListProviders(fmt.Sprintf("parse response: %v", err)), nil
		}

		if !listResp.Success {
			msg := listResp.Message
			if msg == "" {
				msg = "upstream returned success=false"
			}
			return errorResultListProviders(msg), nil
		}

		if listResp.Data == nil {
			return errorResultListProviders("upstream returned empty data"), nil
		}

		channels := listResp.Data.Items

		if len(channels) == 0 {
			toolDuration := time.Since(start)
			slog.InfoContext(ctx, "tool call completed — no channels found",
				"tool", "list_providers",
				"duration_ms", toolDuration.Milliseconds(),
			)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "没有找到任何渠道"},
				},
			}, nil
		}

		// Group channels by their groups field
		grouped := groupChannels(channels)

		// Sort each group by priority descending
		for _, chs := range grouped {
			sort.Slice(chs, func(i, j int) bool {
				return chs[i].Priority > chs[j].Priority
			})
		}

		// Format output as markdown
		output := formatGroupedOutput(grouped)

		toolDuration := time.Since(start)
		isError := false

		// Record metrics
		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("list_providers", "success").Inc()
			metrics.ToolRequestDuration.WithLabelValues("list_providers").Observe(toolDuration.Seconds())
			metrics.UpstreamRequestsTotal.WithLabelValues("GET", "/api/channel/", "200").Inc()
			metrics.UpstreamRequestDuration.WithLabelValues("GET", "/api/channel/").Observe(upstreamDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "list_providers",
			"channel_count", len(channels),
			"group_count", len(grouped),
			"duration_ms", toolDuration.Milliseconds(),
		)

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: output},
			},
		}
		if isError {
			result.IsError = true
			span.SetAttributes(attribute.Int("http.status_code", 200))
		}

		return result, nil
	}
}

// groupChannels groups channels by their comma-separated groups field.
// A channel can appear in multiple groups if its groups field contains
// multiple comma-separated group names.
func groupChannels(channels []Channel) map[string][]Channel {
	grouped := make(map[string][]Channel)
	for _, ch := range channels {
		groupNames := parseGroups(ch.Groups)
		for _, g := range groupNames {
			grouped[g] = append(grouped[g], ch)
		}
	}
	return grouped
}

// parseGroups splits a comma-separated groups string into individual group names.
// Empty strings default to "(无分组)".
func parseGroups(groups string) []string {
	if groups == "" {
		return []string{"(无分组)"}
	}
	parts := strings.Split(groups, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			p = "(无分组)"
		}
		result = append(result, p)
	}
	return result
}

// formatGroupedOutput formats grouped channels as structured markdown text.
func formatGroupedOutput(grouped map[string][]Channel) string {
	// Sort group names for consistent output
	groupNames := make([]string, 0, len(grouped))
	for name := range grouped {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	var b strings.Builder
	for _, groupName := range groupNames {
		chs := grouped[groupName]
		fmt.Fprintf(&b, "## 分组名（%s）\n", groupName)
		fmt.Fprintf(&b, "| ID | 名称 | 状态 | 模型 | 优先级 |\n")
		fmt.Fprintf(&b, "|----|------|------|------|--------|\n")
		for _, ch := range chs {
			models := strings.ReplaceAll(ch.Models, ",", ", ")
			fmt.Fprintf(&b, "| %d | %s | %s | %s | %d |\n",
				ch.ID, ch.Name, statusText(ch.Status), models, ch.Priority)
		}
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

// statusText converts a channel status integer to display text.
func statusText(status int) string {
	if status == 1 {
		return "✅ 启用"
	}
	return "❌ 禁用"
}

func errorResultListProviders(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}