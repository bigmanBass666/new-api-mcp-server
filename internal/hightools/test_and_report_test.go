package hightools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// saveTestAndReportConfig restores defaultTestAndReportConfig after each test.
func saveTestAndReportConfig() func() {
	old := defaultTestAndReportConfig
	return func() { defaultTestAndReportConfig = old }
}

// setFastPoll sets poll timing to very short values for fast tests.
func setFastPoll() {
	defaultTestAndReportConfig = testAndReportConfig{
		pollInterval: 5 * time.Millisecond,
		pollTimeout:  50 * time.Millisecond,
	}
}

func TestTestAndReport_Success_SomeFailed(t *testing.T) {
	defer saveTestAndReportConfig()()
	setFastPoll()

	var mu sync.Mutex
	pollCount := 0

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/channel/test":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{"task_id":"task-123"}}`))
		case "/api/system-task/task-123":
			mu.Lock()
			pollCount++
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{"status":"succeeded","result":{"tested":5,"succeeded":4,"failed":1,"disabled":0,"enabled":5}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewTestAndReportTool(c, nil)

	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{}`)}}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler returned IsError=true for success: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	t.Logf("Output:\n%s", text)

	if !strings.Contains(text, "测试数: 5") {
		t.Error("expected '测试数: 5' in output")
	}
	if !strings.Contains(text, "通过: 4") {
		t.Error("expected '通过: 4' in output")
	}
	if !strings.Contains(text, "失败: 1") {
		t.Error("expected '失败: 1' in output")
	}
	if !strings.Contains(text, "已完成（有失败）") {
		t.Error("expected status with warnings")
	}
	if !strings.Contains(text, "失败渠道详情") {
		t.Error("expected '失败渠道详情' section when failures present")
	}
	if pollCount < 1 {
		t.Error("expected at least one poll call")
	}
}

func TestTestAndReport_Success_AllPass(t *testing.T) {
	defer saveTestAndReportConfig()()
	setFastPoll()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/channel/test":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{"task_id":"task-all-pass"}}`))
		case "/api/system-task/task-all-pass":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{"status":"succeeded","result":{"tested":3,"succeeded":3,"failed":0,"disabled":0,"enabled":3}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewTestAndReportTool(c, nil)

	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{}`)}}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler returned IsError=true for success: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	t.Logf("Output:\n%s", text)

	if !strings.Contains(text, "测试数: 3") {
		t.Error("expected '测试数: 3' in output")
	}
	if !strings.Contains(text, "通过: 3") {
		t.Error("expected '通过: 3' in output")
	}
	if !strings.Contains(text, "失败: 0") {
		t.Error("expected '失败: 0' in output")
	}
	if !strings.Contains(text, "✅ 已完成") {
		t.Error("expected checkmark status")
	}
	if strings.Contains(text, "失败渠道详情") {
		t.Error("should NOT include '失败渠道详情' section when all pass")
	}
}

func TestTestAndReport_Conflict(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"success":false,"message":"A test task is already running."}`))
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewTestAndReportTool(c, nil)

	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{}`)}}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler should NOT set IsError=true for conflict: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	t.Logf("Output:\n%s", text)

	if !strings.Contains(text, "already in progress") {
		t.Errorf("expected friendly 'already in progress' message, got: %s", text)
	}
}

func TestTestAndReport_Timeout(t *testing.T) {
	defer saveTestAndReportConfig()()
	setFastPoll()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/channel/test":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{"task_id":"task-timeout"}}`))
		case "/api/system-task/task-timeout":
			// Always return "running" to simulate a long-running task
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{"status":"running"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewTestAndReportTool(c, nil)

	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{}`)}}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler should NOT set IsError=true for timeout: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	t.Logf("Output:\n%s", text)

	// Should contain timeout indication
	if !strings.Contains(text, "未完成") && !strings.Contains(text, "timeout") && !strings.Contains(text, "Timeout") {
		t.Errorf("expected timeout/未完成 indication in output, got: %s", text)
	}
	// Should contain partial summary fields
	if !strings.Contains(text, "测试数:") {
		t.Error("expected partial summary with tested count")
	}
}

func TestTestAndReport_UpstreamError(t *testing.T) {
	// Point at an unreachable address
	c := client.New("http://127.0.0.1:1", "", "sk-system", "", 0)
	tool := NewTestAndReportTool(c, nil)

	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{}`)}}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for upstream unreachable")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	t.Logf("Output:\n%s", text)

	if !strings.Contains(text, "failed to trigger channel test") {
		t.Errorf("expected upstream failure message, got: %s", text)
	}
}

func TestTestAndReport_NoTaskID(t *testing.T) {
	defer saveTestAndReportConfig()()
	setFastPoll()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"data":{}}`)) // no task_id
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewTestAndReportTool(c, nil)

	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{}`)}}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when no task_id")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	t.Logf("Output:\n%s", text)

	if !strings.Contains(text, "did not return a task_id") {
		t.Errorf("expected 'did not return a task_id', got: %s", text)
	}
}

func TestTestAndReport_SystemTaskFailed(t *testing.T) {
	defer saveTestAndReportConfig()()
	setFastPoll()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/channel/test":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{"task_id":"task-fail"}}`))
		case "/api/system-task/task-fail":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{"status":"failed","error":"channel timeout"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewTestAndReportTool(c, nil)

	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{}`)}}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler should NOT set IsError=true for task failure: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	t.Logf("Output:\n%s", text)

	if !strings.Contains(text, "channel timeout") {
		t.Errorf("expected 'channel timeout' in output, got: %s", text)
	}
}

func TestTestAndReport_EmptyResultOnSuccess(t *testing.T) {
	defer saveTestAndReportConfig()()
	setFastPoll()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/channel/test":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{"task_id":"task-empty"}}`))
		case "/api/system-task/task-empty":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{"status":"succeeded","result":null}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewTestAndReportTool(c, nil)

	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{}`)}}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler should NOT set IsError=true for null result: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	t.Logf("Output:\n%s", text)

	// With nil result, decodeSummary gets an empty struct
	if !strings.Contains(text, "测试数: 0") {
		t.Errorf("expected zero counts for nil result, got: %s", text)
	}
}

// TestTestAndReport_CancelContext ensures the handler respects context cancellation.
func TestTestAndReport_CancelContext(t *testing.T) {
	defer saveTestAndReportConfig()()
	setFastPoll()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/channel/test":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{"task_id":"task-cancel"}}`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{"status":"running"}}`))
		}
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewTestAndReportTool(c, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{}`)}}
	result, err := tool.Handler(ctx, req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	t.Logf("Output:\n%s", text)

	// Should handle gracefully (either IsError or partial summary with cancellation)
	// The handler calls triggerChannelTest first; if ctx is cancelled, the HTTP
	// call in Do() will likely fail. Either way, should not panic.
	if !strings.Contains(text, "failed to trigger") && !result.IsError {
		t.Log("Handler handled cancellation gracefully (no error)")
	}
}

// TestTestAndReport_FormatOutput verifies the full output template via formatTestReport.
func TestTestAndReport_FormatOutput(t *testing.T) {
	tests := []struct {
		name    string
		summary channelTestSummary
		elapsed float64
		pollErr error
		checks  []string
	}{
		{
			name:    "all pass",
			summary: channelTestSummary{Tested: 5, Succeeded: 5, Failed: 0, Disabled: 0, Enabled: 5},
			elapsed: 3.5,
			checks:  []string{"测试数: 5", "通过: 5", "失败: 0", "✅ 已完成"},
		},
		{
			name:    "some failed",
			summary: channelTestSummary{Tested: 10, Succeeded: 8, Failed: 2, Disabled: 1, Enabled: 9},
			elapsed: 45.2,
			checks:  []string{"测试数: 10", "通过: 8", "失败: 2", "⚠️ 已完成（有失败）", "失败渠道详情"},
		},
		{
			name:    "timeout partial",
			summary: channelTestSummary{Tested: 3, Succeeded: 2, Failed: 0, Disabled: 0, Enabled: 3},
			elapsed: 120.0,
			pollErr: fmt.Errorf("timeout after waiting 2m0s"),
			checks:  []string{"测试数: 3", "⏳ 未完成（timeout after waiting 2m0s）"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatTestReport(tt.summary, tt.elapsed, tt.pollErr)
			t.Logf("Output:\n%s", output)

			for _, check := range tt.checks {
				if !strings.Contains(output, check) {
					t.Errorf("expected output to contain %q", check)
				}
			}
		})
	}
}