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

var tracerListUsers = otel.Tracer("hightools.list_users")

// User represents a single user from the upstream GET /api/user/ response.
type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Role         int    `json:"role"`
	Status       int    `json:"status"`
	Email        string `json:"email"`
	Group        string `json:"group"`
	Quota        int64  `json:"quota"`
	UsedQuota    int64  `json:"used_quota"`
	RequestCount int    `json:"request_count"`
}

// listUsersResponse is the upstream response wrapper for GET /api/user/.
type listUsersResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    *listUsersData  `json:"data"`
}

type listUsersData struct {
	Items    []User `json:"items"`
	Total    int    `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// roleName maps user role integers to display names.
func roleName(role int) string {
	switch role {
	case 100:
		return "超级管理员"
	case 10:
		return "管理员"
	case 1:
		return "普通用户"
	default:
		return fmt.Sprintf("未知(%d)", role)
	}
}

// userStatusText converts a user status integer to display text.
func userStatusText(status int) string {
	if status == 1 {
		return "✅"
	}
	return "❌"
}

// formatInt formats an integer with comma thousands separators.
func formatInt(n int64) string {
	in := fmt.Sprintf("%d", n)
	if len(in) <= 3 {
		return in
	}
	var parts []string
	for i := len(in); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		parts = append([]string{in[start:i]}, parts...)
	}
	return strings.Join(parts, ",")
}

// NewListUsersTool creates a tool that lists all users.
// The output is a structured text table with user ID, username, role, status,
// group, used quota, and total quota for each user.
func NewListUsersTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "list_users",
		Description: "List all users. Returns a structured text table with user ID, username, role, status, group, used quota, and total quota for each user.",
		InputSchema: inputSchemaListUsers(),
		Handler:     handleListUsers(c, metrics),
	}
}

func inputSchemaListUsers() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func handleListUsers(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerListUsers.Start(ctx, "list_users",
			trace.WithAttributes(attribute.String("tool", "list_users")),
		)
		defer span.End()

		start := time.Now()

		// Build query params — request all users in one page
		queryParams := map[string]string{
			"page_size": "100",
		}

		// Call upstream API
		upstreamStart := time.Now()
		resp, err := c.Do(ctx, client.SourceAPI, "GET", "/api/user/", queryParams, nil, nil)
		upstreamDuration := time.Since(upstreamStart)
		if err != nil {
			slog.ErrorContext(ctx, "upstream request failed",
				"tool", "list_users",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return errorResultListUsers(fmt.Sprintf("upstream error: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResultListUsers(fmt.Sprintf("read response: %v", err)), nil
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return errorResultListUsers(fmt.Sprintf("upstream returned status %d: %s", resp.StatusCode, string(respBody))), nil
		}

		// Parse upstream JSON response
		var listResp listUsersResponse
		if err := json.Unmarshal(respBody, &listResp); err != nil {
			return errorResultListUsers(fmt.Sprintf("parse response: %v", err)), nil
		}

		if !listResp.Success {
			msg := listResp.Message
			if msg == "" {
				msg = "upstream returned success=false"
			}
			return errorResultListUsers(msg), nil
		}

		if listResp.Data == nil {
			return errorResultListUsers("upstream returned empty data"), nil
		}

		users := listResp.Data.Items

		if len(users) == 0 {
			toolDuration := time.Since(start)
			slog.InfoContext(ctx, "tool call completed — no users found",
				"tool", "list_users",
				"duration_ms", toolDuration.Milliseconds(),
			)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "没有找到任何用户"},
				},
			}, nil
		}

		// Format output as markdown table
		output := formatUsersTable(users)

		toolDuration := time.Since(start)
		isError := false

		// Record metrics
		if metrics != nil {
			metrics.ToolRequestsTotal.WithLabelValues("list_users", "success").Inc()
			metrics.ToolRequestDuration.WithLabelValues("list_users").Observe(toolDuration.Seconds())
			metrics.UpstreamRequestsTotal.WithLabelValues("GET", "/api/user/", "200").Inc()
			metrics.UpstreamRequestDuration.WithLabelValues("GET", "/api/user/").Observe(upstreamDuration.Seconds())
		}

		slog.InfoContext(ctx, "tool call completed",
			"tool", "list_users",
			"user_count", len(users),
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

// formatUsersTable formats users as a markdown table.
func formatUsersTable(users []User) string {
	var b strings.Builder
	fmt.Fprintf(&b, "| ID | 用户名 | 角色 | 状态 | 分组 | 已用配额 | 总配额 |\n")
	fmt.Fprintf(&b, "|----|--------|------|------|------|----------|--------|\n")
	for _, u := range users {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s | %s |\n",
			u.ID, u.Username, roleName(u.Role), userStatusText(u.Status),
			emptyGroup(u.Group), formatInt(u.UsedQuota), formatInt(u.Quota))
	}
	return strings.TrimRight(b.String(), "\n")
}

// emptyGroup returns "(无)" if the group string is empty.
func emptyGroup(group string) string {
	if group == "" {
		return "(无)"
	}
	return group
}

func errorResultListUsers(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}