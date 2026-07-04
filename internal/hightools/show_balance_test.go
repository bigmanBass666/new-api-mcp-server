package hightools

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mockBalanceChannel builds a channel JSON object with balance fields for mock upstream.
func mockBalanceChannel(id int, name string, status int, balance float64, usedQuota int64, balanceUpdatedTime int64) map[string]any {
	return map[string]any{
		"id":                  id,
		"name":                name,
		"type":                1,
		"status":              status,
		"models":              "gpt-4",
		"group":               "default",
		"priority":            10,
		"weight":              1,
		"balance":             balance,
		"used_quota":          usedQuota,
		"balance_updated_time": balanceUpdatedTime,
	}
}

// mockBalanceListResponse builds the upstream channel list response JSON.
func mockBalanceListResponse(channels []map[string]any) string {
	data := map[string]any{
		"items":       channels,
		"total":       len(channels),
		"page":        1,
		"page_size":   100,
		"type_counts": map[string]any{},
	}
	resp := map[string]any{
		"success": true,
		"message": "",
		"data":    data,
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

// mockUpdateBalanceResponse builds the upstream balance refresh response JSON.
func mockUpdateBalanceResponse(success bool, message string) string {
	resp := map[string]any{
		"success": success,
		"message": message,
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

// runShowBalanceTool is a convenience to create a client, tool handler, and call the tool.
func runShowBalanceTool(t *testing.T, upstreamURL string, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	t.Helper()
	ctx := context.Background()
	c := client.New(upstreamURL, "", "test-system-key", "1", 0)
	tool := NewShowBalanceTool(c, nil)
	return tool.Handler(ctx, req)
}

func runShowBalanceToolWithArgs(t *testing.T, upstreamURL string, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()
	req := makeRequest(t, args)
	return runShowBalanceTool(t, upstreamURL, req)
}

func TestShowBalance_Normal(t *testing.T) {
	var callCount int
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			// First call: update_balance
			if r.Method != "GET" {
				t.Errorf("call 1: expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/api/channel/update_balance" {
				t.Errorf("call 1: expected /api/channel/update_balance, got %s", r.URL.Path)
			}
			w.Write([]byte(mockUpdateBalanceResponse(true, "")))
		case 2:
			// Second call: channel list
			if r.Method != "GET" {
				t.Errorf("call 2: expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/api/channel/" {
				t.Errorf("call 2: expected /api/channel/, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("page_size") != "100" {
				t.Errorf("call 2: expected page_size=100, got %s", r.URL.Query().Get("page_size"))
			}

			channels := []map[string]any{
				mockBalanceChannel(1, "GPT-4-OpenAI", 1, 100.50, 500000, 1720080000),
				mockBalanceChannel(2, "Claude-Anthropic", 1, 250.00, 1200000, 1719993600),
				mockBalanceChannel(3, "Gemini-Google", 0, 0.00, 800000, 0),
			}
			w.Write([]byte(mockBalanceListResponse(channels)))
		}
	})
	defer mux.Close()

	result, err := runShowBalanceToolWithArgs(t, mux.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	if callCount != 2 {
		t.Errorf("expected 2 upstream calls, got %d", callCount)
	}

	output := result.Content[0].(*mcp.TextContent).Text

	// Should contain balance table header
	if !strings.Contains(output, "剩余额度") {
		t.Errorf("expected '剩余额度' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "已用额度") {
		t.Errorf("expected '已用额度' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "最后更新时间") {
		t.Errorf("expected '最后更新时间' in output, got:\n%s", output)
	}

	// Should contain channel names
	if !strings.Contains(output, "GPT-4-OpenAI") {
		t.Errorf("expected channel 'GPT-4-OpenAI' in output")
	}
	if !strings.Contains(output, "Claude-Anthropic") {
		t.Errorf("expected channel 'Claude-Anthropic' in output")
	}
	if !strings.Contains(output, "Gemini-Google") {
		t.Errorf("expected channel 'Gemini-Google' in output")
	}

	// Should contain balance values
	if !strings.Contains(output, "100.50") {
		t.Errorf("expected balance 100.50 in output")
	}
	if !strings.Contains(output, "250.00") {
		t.Errorf("expected balance 250.00 in output")
	}
	if !strings.Contains(output, "0.00") {
		t.Errorf("expected balance 0.00 in output")
	}

	// Should contain status indicators
	if !strings.Contains(output, "✅ 启用") {
		t.Errorf("expected enabled status indicator in output")
	}
	if !strings.Contains(output, "❌ 禁用") {
		t.Errorf("expected disabled status indicator in output")
	}

	// Should show "(从未更新)" for zero timestamp
	if !strings.Contains(output, "从未更新") {
		t.Errorf("expected '(从未更新)' for zero balance_updated_time, got:\n%s", output)
	}

	// Should NOT contain refresh warning
	if strings.Contains(output, "缓存数据") {
		t.Errorf("unexpected refresh warning when refresh succeeded")
	}

	t.Logf("Output:\n%s", output)
}

func TestShowBalance_RefreshFailureTolerated(t *testing.T) {
	var callCount int
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			// First call: update_balance fails
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"success":false,"message":"refresh error"}`))
		case 2:
			// Second call: channel list succeeds
			channels := []map[string]any{
				mockBalanceChannel(1, "GPT-4-OpenAI", 1, 50.00, 100000, 1720080000),
			}
			w.Write([]byte(mockBalanceListResponse(channels)))
		}
	})
	defer mux.Close()

	result, err := runShowBalanceToolWithArgs(t, mux.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success despite refresh failure, got IsError=true")
	}

	output := result.Content[0].(*mcp.TextContent).Text

	// Should contain refresh warning
	if !strings.Contains(output, "缓存数据") {
		t.Errorf("expected refresh warning in output when refresh fails, got:\n%s", output)
	}

	// Should still contain channel data
	if !strings.Contains(output, "GPT-4-OpenAI") {
		t.Errorf("expected channel data despite refresh failure, got:\n%s", output)
	}
}

func TestShowBalance_RefreshSuccessFalseTolerated(t *testing.T) {
	var callCount int
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			// First call: update_balance returns success=false
			w.Write([]byte(mockUpdateBalanceResponse(false, "rate limited")))
		case 2:
			// Second call: channel list succeeds
			channels := []map[string]any{
				mockBalanceChannel(1, "GPT-4-OpenAI", 1, 50.00, 100000, 1720080000),
			}
			w.Write([]byte(mockBalanceListResponse(channels)))
		}
	})
	defer mux.Close()

	result, err := runShowBalanceToolWithArgs(t, mux.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success despite refresh returning success=false, got IsError=true")
	}

	output := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "缓存数据") {
		t.Errorf("expected refresh warning when refresh returns success=false, got:\n%s", output)
	}
	if !strings.Contains(output, "GPT-4-OpenAI") {
		t.Errorf("expected channel data despite refresh failure")
	}
}

func TestShowBalance_EmptyList(t *testing.T) {
	var callCount int
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			w.Write([]byte(mockUpdateBalanceResponse(true, "")))
		case 2:
			channels := []map[string]any{}
			w.Write([]byte(mockBalanceListResponse(channels)))
		}
	})
	defer mux.Close()

	result, err := runShowBalanceToolWithArgs(t, mux.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success for empty list, got error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	output := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "没有找到任何渠道") {
		t.Errorf("expected '没有找到任何渠道' for empty list, got:\n%s", output)
	}
}

func TestShowBalance_ChannelListError(t *testing.T) {
	var callCount int
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			w.Write([]byte(mockUpdateBalanceResponse(true, "")))
		case 2:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"success":false,"message":"internal error"}`))
		}
	})
	defer mux.Close()

	result, err := runShowBalanceToolWithArgs(t, mux.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when channel list fails")
	}

	output := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "500") {
		t.Errorf("expected 500 status in error message, got:\n%s", output)
	}
}

func TestShowBalance_ChannelListSuccessFalse(t *testing.T) {
	var callCount int
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			w.Write([]byte(mockUpdateBalanceResponse(true, "")))
		case 2:
			w.Write([]byte(`{"success":false,"message":"permission denied"}`))
		}
	})
	defer mux.Close()

	result, err := runShowBalanceToolWithArgs(t, mux.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for success=false channel list response")
	}

	output := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "permission denied") {
		t.Errorf("expected 'permission denied' in error message, got:\n%s", output)
	}
}

func TestShowBalance_ChannelID(t *testing.T) {
	var callCount int
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			// First call: update_balance/{id}
			if r.URL.Path != "/api/channel/update_balance/5" {
				t.Errorf("call 1: expected /api/channel/update_balance/5, got %s", r.URL.Path)
			}
			w.Write([]byte(mockUpdateBalanceResponse(true, "")))
		case 2:
			// Second call: channel list
			channels := []map[string]any{
				mockBalanceChannel(5, "Specific-Channel", 1, 999.99, 50000, 1720080000),
			}
			w.Write([]byte(mockBalanceListResponse(channels)))
		}
	})
	defer mux.Close()

	result, err := runShowBalanceToolWithArgs(t, mux.URL, map[string]any{"channel_id": 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	if callCount != 2 {
		t.Errorf("expected 2 upstream calls, got %d", callCount)
	}

	output := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "Specific-Channel") {
		t.Errorf("expected channel 'Specific-Channel' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "999.99") {
		t.Errorf("expected balance 999.99 in output")
	}
}

func TestShowBalance_GroupFilter(t *testing.T) {
	var callCount int
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			w.Write([]byte(mockUpdateBalanceResponse(true, "")))
		case 2:
			// Verify group filter was passed upstream
			group := r.URL.Query().Get("group")
			if group != "vip" {
				t.Errorf("expected group=vip query param, got %q", group)
			}

			channels := []map[string]any{
				mockBalanceChannel(1, "VIP-Channel", 1, 500.00, 200000, 1720080000),
			}
			w.Write([]byte(mockBalanceListResponse(channels)))
		}
	})
	defer mux.Close()

	result, err := runShowBalanceToolWithArgs(t, mux.URL, map[string]any{"group": "vip"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	output := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "VIP-Channel") {
		t.Errorf("expected channel 'VIP-Channel' in output, got:\n%s", output)
	}
}

func TestShowBalance_ZeroTimestamp(t *testing.T) {
	var callCount int
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			w.Write([]byte(mockUpdateBalanceResponse(true, "")))
		case 2:
			channels := []map[string]any{
				mockBalanceChannel(1, "No-Balance-Channel", 1, 0.00, 0, 0),
			}
			w.Write([]byte(mockBalanceListResponse(channels)))
		}
	})
	defer mux.Close()

	result, err := runShowBalanceToolWithArgs(t, mux.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	output := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "从未更新") {
		t.Errorf("expected '(从未更新)' for zero timestamp, got:\n%s", output)
	}
}

func TestShowBalance_InvalidArgs(t *testing.T) {
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected upstream call for invalid args")
	})
	defer mux.Close()

	// Test invalid channel_id type
	result, err := runShowBalanceToolWithArgs(t, mux.URL, map[string]any{"channel_id": "abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for invalid channel_id type")
	}

	// Test invalid group type
	result, err = runShowBalanceToolWithArgs(t, mux.URL, map[string]any{"group": 123})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for invalid group type")
	}
}

func TestShowBalance_FormatBalanceTime(t *testing.T) {
	tests := []struct {
		ts   int64
		want string
	}{
		{0, "从未更新"},
		{1720080000, "2024-07-04"},
	}
	for _, tt := range tests {
		got := formatBalanceTime(tt.ts)
		if tt.ts == 0 {
			if got != "(从未更新)" {
				t.Errorf("formatBalanceTime(0) = %q, want %q", got, "(从未更新)")
			}
		} else {
			if !strings.Contains(got, "2024") {
				t.Errorf("formatBalanceTime(%d) = %q, want containing '2024'", tt.ts, got)
			}
		}
	}
}

func TestShowBalance_BalanceStatusText(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{1, "启用"},
		{0, "禁用"},
		{2, "禁用"},
		{-1, "禁用"},
	}
	for _, tt := range tests {
		got := balanceStatusText(tt.status)
		if !strings.Contains(got, tt.want) {
			t.Errorf("balanceStatusText(%d) = %q, want containing %q", tt.status, got, tt.want)
		}
	}
}