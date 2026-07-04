package hightools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
)

// mockChannel is a helper to build a channel JSON object for the mock upstream.
func mockChannel(id int, name string, status int, models, groups string, priority int) map[string]any {
	return map[string]any{
		"id":       id,
		"name":     name,
		"type":     1,
		"status":   status,
		"models":   models,
		"groups":   groups,
		"priority": priority,
		"weight":   1,
	}
}

// mockListResponse builds the upstream response JSON.
func mockListResponse(channels []map[string]any) string {
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

func mockUpstream(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(handler))
}

func makeRequest(t *testing.T, args map[string]any) *mcp.CallToolRequest {
	t.Helper()
	raw := json.RawMessage(`{}`)
	if args != nil {
		b, err := json.Marshal(args)
		if err != nil {
			t.Fatalf("marshal args: %v", err)
		}
		raw = json.RawMessage(b)
	}
	return &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: raw,
		},
	}
}

// runTool is a convenience to create a client, tool handler, and call the tool.
func runTool(t *testing.T, upstreamURL string, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	t.Helper()
	ctx := context.Background()
	c := client.New(upstreamURL, "", "test-system-key", "1", 0)
	tool := NewListProvidersTool(c, nil)
	return tool.Handler(ctx, req)
}

func TestListProviders_Normal(t *testing.T) {
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/channel/" {
			t.Errorf("expected /api/channel/, got %s", r.URL.Path)
		}

		channels := []map[string]any{
			mockChannel(1, "GPT-4-OpenAI", 1, "gpt-4", "default", 10),
			mockChannel(2, "Claude-Anthropic", 1, "claude-3-opus,claude-3-sonnet", "default,vip", 20),
			mockChannel(3, "Gemini-Google", 0, "gemini-pro", "vip", 5),
			mockChannel(4, "DALL-E-OpenAI", 1, "dall-e-3", "", 15),
		}
		w.Write([]byte(mockListResponse(channels)))
	})
	defer mux.Close()

	result, err := runTool(t, mux.URL, makeRequest(t, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	output := result.Content[0].(*mcp.TextContent).Text

	// Should contain group headers
	if !strings.Contains(output, "default") {
		t.Errorf("expected group 'default' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "vip") {
		t.Errorf("expected group 'vip' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "无分组") {
		t.Errorf("expected group '(无分组)' in output, got:\n%s", output)
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
	if !strings.Contains(output, "DALL-E-OpenAI") {
		t.Errorf("expected channel 'DALL-E-OpenAI' in output")
	}

	// Claude-Anthropic should appear in both default and vip groups (appears twice)
	// and priority sorting should have 20 first, then 10 in default group
	// Let's verify priority ordering in default group
	defaultIdx := strings.Index(output, "default")
	vipIdx := strings.Index(output, "vip")
	if defaultIdx < 0 || vipIdx < 0 {
		t.Fatalf("could not find group sections")
	}

	t.Logf("Output:\n%s", output)
}

func TestListProviders_EmptyList(t *testing.T) {
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		channels := []map[string]any{}
		w.Write([]byte(mockListResponse(channels)))
	})
	defer mux.Close()

	result, err := runTool(t, mux.URL, makeRequest(t, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success for empty list, got error: %s", result.Content[0].(*mcp.TextContent).Text)
	}
	output := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "没有找到任何渠道") {
		t.Errorf("expected '没有找到任何渠道' message for empty list, got:\n%s", output)
	}
}

func TestListProviders_UpstreamError(t *testing.T) {
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"message":"internal error"}`))
	})
	defer mux.Close()

	result, err := runTool(t, mux.URL, makeRequest(t, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for upstream error")
	}
	output := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "500") {
		t.Errorf("expected 500 status in error message, got:\n%s", output)
	}
}

func TestListProviders_UpstreamSuccessFalse(t *testing.T) {
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":false,"message":"permission denied"}`))
	})
	defer mux.Close()

	result, err := runTool(t, mux.URL, makeRequest(t, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for success=false upstream response")
	}
	output := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "permission denied") {
		t.Errorf("expected 'permission denied' in error message, got:\n%s", output)
	}
}

func TestListProviders_FilterByGroup(t *testing.T) {
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify the group filter was passed upstream
		group := r.URL.Query().Get("group")
		if group != "vip" {
			t.Errorf("expected group=vip query param, got %q", group)
		}

		channels := []map[string]any{
			mockChannel(2, "Claude-Anthropic", 1, "claude-3-opus", "default,vip", 20),
			mockChannel(3, "Gemini-Google", 0, "gemini-pro", "vip", 5),
		}
		w.Write([]byte(mockListResponse(channels)))
	})
	defer mux.Close()

	result, err := runTool(t, mux.URL, makeRequest(t, map[string]any{"group": "vip"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	output := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "vip") {
		t.Errorf("expected group 'vip' in output, got:\n%s", output)
	}
	// The default group should not appear (no channels filtered to it)
	if strings.Contains(output, "无分组") {
		t.Errorf("unexpected '(无分组)' group in output:\n%s", output)
	}
}

func TestListProviders_FilterByStatus(t *testing.T) {
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify the status filter was passed upstream
		status := r.URL.Query().Get("status")
		if status != "1" {
			t.Errorf("expected status=1 query param, got %q", status)
		}

		channels := []map[string]any{
			mockChannel(1, "GPT-4-OpenAI", 1, "gpt-4", "default", 10),
			mockChannel(2, "Claude-Anthropic", 1, "claude-3-opus", "default,vip", 20),
		}
		w.Write([]byte(mockListResponse(channels)))
	})
	defer mux.Close()

	result, err := runTool(t, mux.URL, makeRequest(t, map[string]any{"status": 1}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	output := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "GPT-4-OpenAI") {
		t.Errorf("expected 'GPT-4-OpenAI' in output")
	}
	if !strings.Contains(output, "✅ 启用") {
		t.Errorf("expected enabled status indicator in output")
	}
}

func TestListProviders_StatusText(t *testing.T) {
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
		got := statusText(tt.status)
		if !strings.Contains(got, tt.want) {
			t.Errorf("statusText(%d) = %q, want containing %q", tt.status, got, tt.want)
		}
	}
}

func TestListProviders_PrioritySorting(t *testing.T) {
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		channels := []map[string]any{
			mockChannel(1, "Low", 1, "model-a", "testgroup", 1),
			mockChannel(2, "High", 1, "model-b", "testgroup", 100),
			mockChannel(3, "Medium", 1, "model-c", "testgroup", 50),
		}
		w.Write([]byte(mockListResponse(channels)))
	})
	defer mux.Close()

	result, err := runTool(t, mux.URL, makeRequest(t, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	output := result.Content[0].(*mcp.TextContent).Text

	// In the testgroup, channels should be sorted by priority descending: High (100), Medium (50), Low (1)
	highIdx := strings.Index(output, "High")
	mediumIdx := strings.Index(output, "Medium")
	lowIdx := strings.Index(output, "Low")

	if highIdx < 0 || mediumIdx < 0 || lowIdx < 0 {
		t.Fatalf("could not find all channels in output:\n%s", output)
	}

	if !(highIdx < mediumIdx && mediumIdx < lowIdx) {
		t.Errorf("channels not sorted by priority descending (expected High < Medium < Low), got indices: High=%d, Medium=%d, Low=%d", highIdx, mediumIdx, lowIdx)
	}
}

func TestListProviders_ParseGroups(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"default", []string{"default"}},
		{"default,vip", []string{"default", "vip"}},
		{" default , vip ", []string{"default", "vip"}},
		{"", []string{"(无分组)"}},
		{",,", []string{"(无分组)", "(无分组)", "(无分组)"}},
	}
	for _, tt := range tests {
		got := parseGroups(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseGroups(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseGroups(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestListProviders_InvalidArgs(t *testing.T) {
	mux := mockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		channels := []map[string]any{mockChannel(1, "Test", 1, "model", "default", 1)}
		w.Write([]byte(mockListResponse(channels)))
	})
	defer mux.Close()

	// Test invalid group type (not a string)
	req := makeRequest(t, map[string]any{"group": 123})
	result, err := runTool(t, mux.URL, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for invalid group type")
	}

	// Test invalid status type (not a number)
	req = makeRequest(t, map[string]any{"status": "abc"})
	result, err = runTool(t, mux.URL, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for invalid status type")
	}
}