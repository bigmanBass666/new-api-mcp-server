//go:build e2e

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── Infrastructure ──────────────────────────────────────────

// buildBinary compiles the MCP server binary to a fixed path so that
// Windows Defender firewall does not prompt on every test run.
func buildBinary(t *testing.T) string {
	t.Helper()
	t.Log("Building server binary...")
	bin := filepath.Join("..", "..", "bin", "new-api-mcp-server-e2e.exe")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

// startMCPServer starts the server binary. Returns the MCP URL.
func startMCPServer(t *testing.T, bin, mockURL, port string) string {
	t.Helper()
	addr := fmt.Sprintf("127.0.0.1:%s", port)
	mcpPath := fmt.Sprintf("http://%s/mcp", addr)

	cmd := exec.Command(bin)
	cmd.Env = append(cmd.Env,
		"MCP_TRANSPORT=http",
		"MCP_HTTP_ADDR="+addr,
		"MCP_HTTP_AUTH_TOKEN=mcp-e2e-token",
		"NEW_API_BASE_URL="+mockURL,
		"NEW_API_SYSTEM_KEY=sk-e2e-system-key",
		"MCP_API_TOOLS_ENABLED=true",
		"MCP_LOG_CONSOLE_ENABLED=false",
		"MCP_LOG_LEVEL=error",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("server start: %v", err)
	}
	if !waitForHealth(fmt.Sprintf("http://%s/healthz", addr), 5*time.Second) {
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatalf("server not healthy\nstderr:\n%s", stderr.String())
	}
	t.Log("   ✓ server ready")

	t.Cleanup(func() {
		exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", cmd.Process.Pid)).Run()
		cmd.Wait()
		if stderr.Len() > 0 {
			t.Logf("   stderr:\n%s", stderr.String())
		}
	})
	return mcpPath
}

func waitForHealth(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// ── MCP Client ──────────────────────────────────────────────

// mcpClient holds session state for E2E test requests.
type mcpClient struct {
	url       string
	token     string
	sessionID string
}

func newMCPClient(url, token string) *mcpClient {
	return &mcpClient{url: url, token: token}
}

// sendJSON sends a JSON body and returns the response body + session ID.
func (c *mcpClient) sendJSON(t *testing.T, body []byte) ([]byte, string) {
	t.Helper()
	req, err := http.NewRequest("POST", c.url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http: %v", err)
	}
	defer resp.Body.Close()

	newSess := resp.Header.Get("Mcp-Session-Id")

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return raw, newSess
}

// call sends a JSON-RPC request and returns the parsed response.
func (c *mcpClient) call(t *testing.T, method string, params any) map[string]any {
	t.Helper()
	body := map[string]any{"jsonrpc": "2.0", "id": 1, "method": method}
	if params != nil {
		body["params"] = params
	}
	data, _ := json.Marshal(body)
	raw, newSess := c.sendJSON(t, data)
	if newSess != "" {
		c.sessionID = newSess
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// Try SSE format
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data: ") {
				s := strings.TrimPrefix(line, "data: ")
				if err := json.Unmarshal([]byte(s), &parsed); err == nil {
					return parsed
				}
			}
		}
		t.Fatalf("parse: %v\nraw: %s", err, raw)
	}
	return parsed
}

// notify sends a JSON-RPC notification (no id) to the MCP server.
func (c *mcpClient) notify(t *testing.T, method string, params any) {
	t.Helper()
	body := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		body["params"] = params
	}
	data, _ := json.Marshal(body)
	c.sendJSON(t, data)
}

// result extracts the "result" field from a JSON-RPC response.
func resultOrFail(t *testing.T, resp map[string]any) map[string]any {
	t.Helper()
	r, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result field: %v", resp)
	}
	return r
}

// toolText extracts the text content from a tools/call result.
func toolText(t *testing.T, data map[string]any) string {
	t.Helper()
	r := resultOrFail(t, data)
	c, ok := r["content"].([]any)
	if !ok {
		b, _ := json.MarshalIndent(r, "", "  ")
		return string(b)
	}
	if len(c) == 0 {
		return "(empty)"
	}
	m, ok := c[0].(map[string]any)
	if !ok {
		return fmt.Sprintf("%v", c[0])
	}
	if s, ok := m["text"].(string); ok {
		return s
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return string(b)
}

// extractTaskID finds "<uuid>" in "任务 ID: <uuid>..."
func extractTaskID(t *testing.T, msg string) string {
	t.Helper()
	p := "任务 ID: "
	i := strings.Index(msg, p)
	if i < 0 {
		t.Fatalf("no task ID in: %s", msg)
	}
	s := msg[i+len(p):]
	for _, sep := range []string{"。", ".", " ", "\n"} {
		if j := strings.Index(s, sep); j > 0 {
			return s[:j]
		}
	}
	return strings.TrimSpace(s)
}

const testToken = "mcp-e2e-token"

// ── Test 1: Basic Tasks Extension Verification ──────────────

func TestE2E_TasksExtension(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	// Setup
	bin := buildBinary(t)
	mock := newE2EMockUpstream("e2e-mock-task", 2)
	mockSrv := httptest.NewServer(mock)
	defer mockSrv.Close()
	t.Logf("   mock upstream: %s", mockSrv.URL)

	mcpURL := startMCPServer(t, bin, mockSrv.URL, "18687")
	cl := newMCPClient(mcpURL, testToken)
	t.Logf("   MCP server: %s", mcpURL)

	// Step 1: Initialize
	t.Log("=== Step 1: initialize ===")
	initResp := cl.call(t, "initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "e2e-test", "version": "1.0"},
	})
	initResult := resultOrFail(t, initResp)
	si, _ := initResult["serverInfo"].(map[string]any)
	t.Logf("   server: %s %s", si["name"], si["version"])

	caps := initResult["capabilities"].(map[string]any)
	exts := caps["extensions"].(map[string]any)
	if _, ok := exts["io.modelcontextprotocol/tasks"]; !ok {
		t.Error("tasks extension not declared")
	} else {
		t.Log("   ✓ tasks extension declared")
	}

	// Step 2: Initialize notification
	t.Log("=== Step 2: notifications/initialized ===")
	cl.notify(t, "notifications/initialized", nil)

	// Step 3: tools/list (with pagination)
	t.Log("=== Step 3: tools/list ===")
	names := map[string]bool{}
	cursor := ""
	for {
		params := map[string]any{}
		if cursor != "" {
			params["cursor"] = cursor
		}
		pageResp := cl.call(t, "tools/list", params)
		pageResult := resultOrFail(t, pageResp)
		tools, _ := pageResult["tools"].([]any)
		for _, item := range tools {
			if m, ok := item.(map[string]any); ok {
				if n, ok := m["name"].(string); ok {
					names[n] = true
				}
			}
		}
		c, hasMore := pageResult["nextCursor"].(string)
		if !hasMore || c == "" {
			break
		}
		cursor = c
	}
	t.Logf("   total unique tools: %d", len(names))
	for _, name := range []string{"tasks_get", "tasks_update", "tasks_cancel", "test_and_report"} {
		if names[name] {
			t.Logf("   ✓ %q registered", name)
		} else {
			t.Errorf("%q not registered", name)
		}
	}

	// Step 4: test_and_report (async)
	t.Log("=== Step 4: test_and_report ===")
	trResp := cl.call(t, "tools/call", map[string]any{
		"name": "test_and_report", "arguments": map[string]any{},
	})
	trText := toolText(t, trResp)
	t.Logf("   response: %s", trText)

	if !strings.Contains(trText, "渠道测试已启动") {
		t.Error("expected async confirmation")
	} else {
		t.Log("   ✓ async return")
	}

	taskID := extractTaskID(t, trText)
	t.Logf("   task_id: %s", taskID)

	// Step 5: tasks_get (existing task)
	t.Log("=== Step 5: tasks_get ===")
	time.Sleep(800 * time.Millisecond)
	tgResp := cl.call(t, "tools/call", map[string]any{
		"name": "tasks_get", "arguments": map[string]any{"task_id": taskID},
	})
	tgText := toolText(t, tgResp)
	t.Logf("   got:\n%s", tgText)
	if strings.Contains(tgText, taskID) {
		t.Log("   ✓ tasks_get works")
	} else {
		t.Error("tasks_get output missing task ID")
	}

	// Step 6: tasks_cancel (non-existent)
	t.Log("=== Step 6: tasks_cancel (non-existent) ===")
	tcResp := cl.call(t, "tools/call", map[string]any{
		"name": "tasks_cancel", "arguments": map[string]any{"task_id": "no-such-task"},
	})
	tcText := toolText(t, tcResp)
	t.Logf("   got: %s", tcText)
	if strings.Contains(tcText, "not found") {
		t.Log("   ✓ correct error for non-existent task")
	} else {
		t.Error("expected 'not found' error")
	}

	t.Log("--- All E2E base checks passed ---")
}

// ── Test 2: Async Completion ────────────────────────────────

func TestE2E_AsyncCompletion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	bin := buildBinary(t)

	// Mock that returns "all pass" result after 2 polls
	mock := newE2EMockUpstream("e2e-success-task", 2)
	mockSrv := httptest.NewServer(mock)
	defer mockSrv.Close()
	t.Logf("   mock upstream: %s", mockSrv.URL)

	mcpURL := startMCPServer(t, bin, mockSrv.URL, "18687")
	cl := newMCPClient(mcpURL, testToken)

	// Initialize and notify
	cl.call(t, "initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "e2e-test", "version": "1.0"},
	})
	cl.notify(t, "notifications/initialized", nil)

	// Trigger test_and_report
	trResp := cl.call(t, "tools/call", map[string]any{
		"name": "test_and_report", "arguments": map[string]any{},
	})
	taskID := extractTaskID(t, toolText(t, trResp))
	t.Logf("   task_id: %s", taskID)

	// Wait for background worker to complete (poll tasks_get)
	t.Log("Waiting for background worker to complete...")
	var finalState string
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		tgResp := cl.call(t, "tools/call", map[string]any{
			"name": "tasks_get", "arguments": map[string]any{"task_id": taskID},
		})
		tgText := toolText(t, tgResp)
		if strings.Contains(tgText, `"succeeded"`) {
			t.Logf("   ✓ task succeeded\n%s", tgText)
			finalState = "succeeded"
			break
		}
		if strings.Contains(tgText, `"failed"`) || strings.Contains(tgText, `"cancelled"`) {
			t.Logf("   unexpected terminal state:\n%s", tgText)
			finalState = "unexpected"
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if finalState != "succeeded" {
		t.Fatal("task did not reach succeeded state within timeout")
	}
}

// ── Test 3: Cancel Running Task ─────────────────────────────

func TestE2E_CancelRunningTask(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	bin := buildBinary(t)

	// Mock that never completes (hang behavior)
	mock := newE2EMockUpstream("e2e-hang-task", 0)
	mock.AddTask("e2e-hang-task", behaviorHang, 0)
	mockSrv := httptest.NewServer(mock)
	defer mockSrv.Close()
	t.Logf("   mock upstream: %s", mockSrv.URL)

	mcpURL := startMCPServer(t, bin, mockSrv.URL, "18688")
	cl := newMCPClient(mcpURL, testToken)

	// Initialize and notify
	cl.call(t, "initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "e2e-test", "version": "1.0"},
	})
	cl.notify(t, "notifications/initialized", nil)

	// Trigger test_and_report
	trResp := cl.call(t, "tools/call", map[string]any{
		"name": "test_and_report", "arguments": map[string]any{},
	})
	taskID := extractTaskID(t, toolText(t, trResp))
	t.Logf("   task_id: %s", taskID)

	// Give the background worker time to start
	time.Sleep(500 * time.Millisecond)

	// Cancel the running task
	t.Log("Cancelling running task...")
	tcResp := cl.call(t, "tools/call", map[string]any{
		"name": "tasks_cancel", "arguments": map[string]any{"task_id": taskID},
	})
	tcText := toolText(t, tcResp)
	t.Logf("   cancel response: %s", tcText)

	if strings.Contains(tcText, "cancelled") || strings.Contains(tcText, "已取消") {
		t.Log("   ✓ task cancelled successfully")
	} else if strings.Contains(tcText, "already in terminal") {
		// Rare race: worker completed before cancel was processed
		t.Log("   ⚠ task already completed before cancel (acceptable race)")
	} else {
		t.Errorf("unexpected cancel response: %s", tcText)
	}

	// Verify via tasks_get
	time.Sleep(200 * time.Millisecond)
	tgResp := cl.call(t, "tools/call", map[string]any{
		"name": "tasks_get", "arguments": map[string]any{"task_id": taskID},
	})
	tgText := toolText(t, tgResp)
	t.Logf("   tasks_get after cancel:\n%s", tgText)
	if strings.Contains(tgText, `"cancelled"`) {
		t.Log("   ✓ tasks_get confirms cancelled state")
	}
}

// ── Test 4: Input Required → Resume ─────────────────────────

func TestE2E_InputRequiredResume(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	bin := buildBinary(t)

	// Mock that returns "failed" after 1 poll, then "succeeded" after resume
	// We use a hack: the mock has a task that fails, but the test doesn't
	// actually wait for the worker to poll again — we just verify the
	// input_required state is reachable.
	mock := newE2EMockUpstream("e2e-fail-task", 1)
	mock.AddTask("e2e-fail-task", behaviorFail, 1)
	mockSrv := httptest.NewServer(mock)
	defer mockSrv.Close()
	t.Logf("   mock upstream: %s", mockSrv.URL)

	mcpURL := startMCPServer(t, bin, mockSrv.URL, "18689")
	cl := newMCPClient(mcpURL, testToken)

	// Initialize and notify
	cl.call(t, "initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "e2e-test", "version": "1.0"},
	})
	cl.notify(t, "notifications/initialized", nil)

	// Trigger test_and_report
	trResp := cl.call(t, "tools/call", map[string]any{
		"name": "test_and_report", "arguments": map[string]any{},
	})
	taskID := extractTaskID(t, toolText(t, trResp))
	t.Logf("   task_id: %s", taskID)

	// Wait for the background worker to detect the upstream task failure
	// and enter input_required state
	t.Log("Waiting for input_required state...")
	deadline := time.Now().Add(8 * time.Second)
	var enteredInputRequired bool
	for time.Now().Before(deadline) {
		tgResp := cl.call(t, "tools/call", map[string]any{
			"name": "tasks_get", "arguments": map[string]any{"task_id": taskID},
		})
		tgText := toolText(t, tgResp)
		if strings.Contains(tgText, `"input_required"`) {
			t.Logf("   ✓ task entered input_required\n%s", tgText)
			enteredInputRequired = true
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !enteredInputRequired {
		// Check if the task somehow succeeded or failed
		tgResp := cl.call(t, "tools/call", map[string]any{
			"name": "tasks_get", "arguments": map[string]any{"task_id": taskID},
		})
		tgText := toolText(t, tgResp)
		t.Fatalf("task did not enter input_required within timeout\n%s", tgText)
	}

	// Now resume via tasks_update
	t.Log("Resuming task via tasks_update...")
	tuResp := cl.call(t, "tools/call", map[string]any{
		"name": "tasks_update", "arguments": map[string]any{
			"task_id": taskID,
			"action":  "resume",
		},
	})
	tuText := toolText(t, tuResp)
	t.Logf("   update response: %s", tuText)
	if strings.Contains(tuText, "running") {
		t.Log("   ✓ task resumed to running")
	} else {
		t.Errorf("expected 'running' after resume, got: %s", tuText)
	}

	t.Log("--- InputRequired/Resume flow verified ---")
}