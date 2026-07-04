package hightools

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
)

// mockUser is a helper to build a user JSON object for the mock upstream.
func mockUser(id int, username, displayName string, role, status int, group string, quota, usedQuota int64) map[string]any {
	return map[string]any{
		"id":            id,
		"username":      username,
		"display_name":  displayName,
		"role":          role,
		"status":        status,
		"email":         username + "@example.com",
		"group":         group,
		"quota":         quota,
		"used_quota":    usedQuota,
		"request_count": 0,
	}
}

// mockListUsersResponse builds the upstream response JSON for user listing.
func mockListUsersResponse(users []map[string]any) string {
	data := map[string]any{
		"items":     users,
		"total":     len(users),
		"page":      1,
		"page_size": 100,
	}
	resp := map[string]any{
		"success": true,
		"message": "",
		"data":    data,
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func TestListUsers_Success(t *testing.T) {
	users := []map[string]any{
		mockUser(1, "admin", "Admin", 100, 1, "default", 1000000, 5000),
		mockUser(2, "user1", "User One", 1, 1, "default", 500000, 100),
		mockUser(3, "user2", "User Two", 1, 0, "", 100000, 9999),
	}

	server := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockListUsersResponse(users)))
	})
	defer server.Close()

	c := client.New(server.URL, "", "sk-xxx", "", 0)
	tool := NewListUsersTool(c, nil)
	req := makeRequest(t, nil)

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected no error, got IsError=true with text: %s", extractText(result))
	}

	output := extractText(result)

	// Check header row
	if !strings.Contains(output, "| ID | 用户名 | 角色 | 状态 | 分组 | 已用配额 | 总配额 |") {
		t.Errorf("output missing table header: %s", output)
	}

	// Check admin row (role 100 = 超级管理员, status 1 = enabled)
	if !strings.Contains(output, "admin") {
		t.Errorf("output missing admin user: %s", output)
	}
	if !strings.Contains(output, "超级管理员") {
		t.Errorf("output missing super admin role text: %s", output)
	}
	if !strings.Contains(output, "✅") {
		t.Errorf("output missing enabled indicator for admin: %s", output)
	}

	// Check user1 row (role 1 = 普通用户, group = default)
	if !strings.Contains(output, "user1") {
		t.Errorf("output missing user1: %s", output)
	}
	if !strings.Contains(output, "普通用户") {
		t.Errorf("output missing normal user role text: %s", output)
	}
	if !strings.Contains(output, "default") {
		t.Errorf("output missing group name: %s", output)
	}

	// Check user2 row (status 0 = disabled, empty group = (无))
	if !strings.Contains(output, "❌") {
		t.Errorf("output missing disabled indicator for user2: %s", output)
	}
	if !strings.Contains(output, "(无)") {
		t.Errorf("output missing '(无)' for empty group: %s", output)
	}

	// Check thousand separators on quota
	if !strings.Contains(output, "1,000,000") {
		t.Errorf("output missing formatted quota (1,000,000): %s", output)
	}
	if !strings.Contains(output, "5,000") {
		t.Errorf("output missing formatted used_quota (5,000): %s", output)
	}
}

func TestListUsers_Empty(t *testing.T) {
	server := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockListUsersResponse(nil)))
	})
	defer server.Close()

	c := client.New(server.URL, "", "sk-xxx", "", 0)
	tool := NewListUsersTool(c, nil)
	req := makeRequest(t, nil)

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected no error for empty list, got IsError=true")
	}

	output := extractText(result)
	if !strings.Contains(output, "没有找到任何用户") {
		t.Errorf("expected '没有找到任何用户', got: %s", output)
	}
}

func TestListUsers_UpstreamError(t *testing.T) {
	server := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`internal error`))
	})
	defer server.Close()

	c := client.New(server.URL, "", "sk-xxx", "", 0)
	tool := NewListUsersTool(c, nil)
	req := makeRequest(t, nil)

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if !result.IsError {
		t.Fatalf("expected IsError=true for upstream 500")
	}

	output := extractText(result)
	if !strings.Contains(output, "upstream returned status 500") {
		t.Errorf("expected upstream error message, got: %s", output)
	}
}

func TestListUsers_SuccessFalse(t *testing.T) {
	server := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":false,"message":"permission denied"}`))
	})
	defer server.Close()

	c := client.New(server.URL, "", "sk-xxx", "", 0)
	tool := NewListUsersTool(c, nil)
	req := makeRequest(t, nil)

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if !result.IsError {
		t.Fatalf("expected IsError=true for success=false")
	}

	output := extractText(result)
	if !strings.Contains(output, "permission denied") {
		t.Errorf("expected 'permission denied', got: %s", output)
	}
}

func TestListUsers_InvalidJSON(t *testing.T) {
	server := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	})
	defer server.Close()

	c := client.New(server.URL, "", "sk-xxx", "", 0)
	tool := NewListUsersTool(c, nil)
	req := makeRequest(t, nil)

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if !result.IsError {
		t.Fatalf("expected IsError=true for invalid JSON")
	}

	output := extractText(result)
	if !strings.Contains(output, "parse response") {
		t.Errorf("expected parse error, got: %s", output)
	}
}

func TestListUsers_NullData(t *testing.T) {
	server := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"message":"","data":null}`))
	})
	defer server.Close()

	c := client.New(server.URL, "", "sk-xxx", "", 0)
	tool := NewListUsersTool(c, nil)
	req := makeRequest(t, nil)

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if !result.IsError {
		t.Fatalf("expected IsError=true for null data")
	}

	output := extractText(result)
	if !strings.Contains(output, "upstream returned empty data") {
		t.Errorf("expected empty data error, got: %s", output)
	}
}

// extractText extracts the first text content from a CallToolResult for assertions.
func extractText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}