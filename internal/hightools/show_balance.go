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

var tracerShowBalance = otel.Tracer("hightools.show_balance")

// ChannelWithBalance represents a single channel from the upstream GET /api/channel/ response
// with balance-related fields included.
type ChannelWithBalance struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	Type               int     `json:"type"`
	Status             int     `json:"status"`
	Models             string  `json:"models"`
	Group              string  `json:"group"`
	Priority           int64   `json:"priority"`
	Weight             int     `json:"weight"`
	Balance            float64 `json:"balance"`
	UsedQuota          int64   `json:"used_quota"`
	BalanceUpdatedTime int64   `json:"balance_updated_time"`
}

// listBalanceChannelsResponse is the upstream response wrapper for GET /api/channel/.
type listBalanceChannelsResponse struct {
	Success bool               `json:"success"`
	Message string             `json:"message"`
	Data    *listBalanceData   `json:"data"`
}

type listBalanceData struct {
	Items      []ChannelWithBalance `json:"items"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TypeCounts map[string]int64    `json:"type_counts"`
}

// updateBalanceResponse maps the upstream GET /api/channel/update_balance[/{id}] response.
type updateBalanceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// NewShowBalanceTool creates a tool that shows balance overview for all channels.
// It first triggers a balance refresh via /api/channel/update_balance, then fetches
// the channel list with balance data from /api/channel/.
func NewShowBalanceTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "show_balance",
		Description: "Show balance overview for all channels. First triggers a balance refresh, then displays remaining balance, used quota, and last update time for each channel in a structured table.",
		InputSchema: inputSchemaShowBalance(),
		Handler:     handleShowBalance(c, metrics),
	}
}

func inputSchemaShowBalance() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"channel_id": map[string]any{
				"type":        "integer",
				"description": "Optional: refresh and show balance for a specific channel ID only.",
			},
			"group": map[string]any{
				"type":        "string",
				"description": "Optional: filter channels by group name (e.g., 'default', 'vip').",
			},
		},
	}
}

func handleShowBalance(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerShowBalance.Start(ctx, "show_balance",
			trace.WithAttributes(attribute.String("tool", "show_balance")),
		)
		defer span.End()

		start := time.Now()

		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultShowBalance(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		// Validate arguments before making upstream calls
		if err := validateShowBalanceArgs(args); err != nil {
			return errorResultShowBalance(err.Error()), nil
		}

		// Step 1: Trigger balance refresh
		_, refreshErr := triggerBalanceRefresh(ctx, c, metrics, args)
		var refreshWarning string
		if refreshErr != nil {
			slog.WarnContext(ctx, "balance refresh failed, will show cached data",
				"tool", "show_balance",
				"error", refreshErr,
			)
			refreshWarning = "refresh_failed"
		}

		// Step 2: Fetch channel list with balance data
		channels, fetchErr := fetchChannelsWithBalance(ctx, c, metrics, args)
		if fetchErr != nil {
			span.SetAttributes(attribute.Bool("error", true))
			return errorResultShowBalance(fmt.Sprintf("failed to fetch channel balance data: %v", fetchErr)), nil
		}

		if len(channels) == 0 {
			toolDuration := time.Since(start)
			slog.InfoContext(ctx, "tool call completed — no channels found",
				"tool", "show_balance",
				"duration_ms", toolDuration.Milliseconds(),
			)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "没有找到任何渠道"},
				},
			}, nil
		}

		// Build output
		output := formatBalanceOutput(channels, refreshWarning)

		toolDuration := time.Since(start)

		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("show_balance", "success").Inc()
			metrics.ToolRequestDuration.WithLabelValues("show_balance").Observe(toolDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "show_balance",
			"channel_count", len(channels),
			"duration_ms", toolDuration.Milliseconds(),
		)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: output},
			},
		}, nil
	}
}

// triggerBalanceRefresh calls the balance refresh endpoint. Errors are tolerated.
func triggerBalanceRefresh(ctx context.Context, c *client.Client, metrics *observability.Metrics, args map[string]any) (string, error) {
	ctx, span := tracerShowBalance.Start(ctx, "trigger_balance_refresh")
	defer span.End()

	start := time.Now()
	path := "/api/channel/update_balance"

	// Check if specific channel_id is provided
	var queryParams map[string]string
	if channelIDRaw, ok := args["channel_id"]; ok {
		channelID, ok := toInt64(channelIDRaw)
		if !ok {
			return "", fmt.Errorf("channel_id must be an integer, got %T", channelIDRaw)
		}
		path = fmt.Sprintf("/api/channel/update_balance/%d", channelID)
	}

	resp, err := c.Do(ctx, client.SourceAPI, "GET", path, queryParams, nil, nil)
	upstreamDuration := time.Since(start)
	if err != nil {
		recordUpstreamMetric(metrics, "GET", path, "error")
		return "", fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		recordUpstreamMetric(metrics, "GET", path, "error")
		return "", fmt.Errorf("read refresh response: %w", err)
	}

	statusCode := resp.StatusCode
	statusStr := fmt.Sprintf("%d", statusCode)

	if metrics != nil {
		metrics.UpstreamRequestsTotal.WithLabelValues("GET", path, statusStr).Inc()
		metrics.UpstreamRequestDuration.WithLabelValues("GET", path).Observe(upstreamDuration.Seconds())
	}

	if statusCode < 200 || statusCode >= 300 {
		return "", fmt.Errorf("refresh returned status %d: %s", statusCode, string(body))
	}

	var refreshResp updateBalanceResponse
	if err := json.Unmarshal(body, &refreshResp); err != nil {
		return "", fmt.Errorf("parse refresh response: %w", err)
	}

	if !refreshResp.Success {
		msg := refreshResp.Message
		if msg == "" {
			msg = "upstream returned success=false"
		}
		return "", fmt.Errorf("refresh failed: %s", msg)
	}

	return "", nil
}

// fetchChannelsWithBalance fetches the channel list with balance data.
func fetchChannelsWithBalance(ctx context.Context, c *client.Client, metrics *observability.Metrics, args map[string]any) ([]ChannelWithBalance, error) {
	ctx, span := tracerShowBalance.Start(ctx, "fetch_channels")
	defer span.End()

	start := time.Now()
	queryParams := map[string]string{
		"page_size": "100",
	}

	if groupRaw, ok := args["group"]; ok {
		group, ok := groupRaw.(string)
		if !ok {
			return nil, fmt.Errorf("group must be a string, got %T", groupRaw)
		}
		queryParams["group"] = group
	}

	resp, err := c.Do(ctx, client.SourceAPI, "GET", "/api/channel/", queryParams, nil, nil)
	upstreamDuration := time.Since(start)
	if err != nil {
		recordUpstreamMetric(metrics, "GET", "/api/channel/", "error")
		return nil, fmt.Errorf("upstream request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	statusCode := resp.StatusCode
	statusStr := fmt.Sprintf("%d", statusCode)

	if metrics != nil {
		metrics.UpstreamRequestsTotal.WithLabelValues("GET", "/api/channel/", statusStr).Inc()
		metrics.UpstreamRequestDuration.WithLabelValues("GET", "/api/channel/").Observe(upstreamDuration.Seconds())
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("upstream returned status %d: %s", statusCode, string(respBody))
	}

	var listResp listBalanceChannelsResponse
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if !listResp.Success {
		msg := listResp.Message
		if msg == "" {
			msg = "upstream returned success=false"
		}
		return nil, fmt.Errorf("%s", msg)
	}

	if listResp.Data == nil {
		return nil, fmt.Errorf("upstream returned empty data")
	}

	return listResp.Data.Items, nil
}

// formatBalanceOutput builds a Markdown table of channel balance information.
func formatBalanceOutput(channels []ChannelWithBalance, refreshWarning string) string {
	// Sort by ID for stable output
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].ID < channels[j].ID
	})

	var b strings.Builder

	if refreshWarning != "" {
		fmt.Fprintf(&b, "**注意：** 余额刷新失败，显示的是缓存数据\n\n")
	}

	b.WriteString("## 余额概览\n\n")
	b.WriteString("| ID | 名称 | 状态 | 剩余额度(USD) | 已用额度 | 最后更新时间 |\n")
	b.WriteString("|----|------|------|--------------|---------|-------------|\n")

	for _, ch := range channels {
		balanceStr := fmt.Sprintf("%.2f", ch.Balance)
		quotaStr := fmt.Sprintf("%d", ch.UsedQuota)
		timeStr := formatBalanceTime(ch.BalanceUpdatedTime)

		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s |\n",
			ch.ID, ch.Name, balanceStatusText(ch.Status), balanceStr, quotaStr, timeStr,
		)
	}

	return strings.TrimRight(b.String(), "\n")
}

// formatBalanceTime converts a Unix timestamp to readable format.
func formatBalanceTime(ts int64) string {
	if ts == 0 {
		return "(从未更新)"
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}

// balanceStatusText converts a channel status integer to display text.
func balanceStatusText(status int) string {
	if status == 1 {
		return "✅ 启用"
	}
	return "❌ 禁用"
}

// validateShowBalanceArgs checks argument types before making upstream calls.
// Returns nil if all args are valid, or an error describing the first invalid arg.
func validateShowBalanceArgs(args map[string]any) error {
	for k, v := range args {
		switch k {
		case "channel_id":
			if _, ok := toInt64(v); !ok {
				return fmt.Errorf("channel_id must be an integer, got %T", v)
			}
		case "group":
			if _, ok := v.(string); !ok {
				return fmt.Errorf("group must be a string, got %T", v)
			}
		}
	}
	return nil
}

// recordUpstreamMetric records an upstream request metric safely.
func recordUpstreamMetric(metrics *observability.Metrics, method, path, status string) {
	if metrics != nil {
		metrics.UpstreamRequestsTotal.WithLabelValues(method, path, status).Inc()
	}
}

func errorResultShowBalance(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}