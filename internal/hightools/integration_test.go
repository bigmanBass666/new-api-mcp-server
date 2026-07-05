//go:build integration

package hightools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	mcpServerURL = "http://localhost:4051"
	authToken    = "mcp-test-token-2026"
)

// mcpRequest sends a JSON-RPC request to the MCP server and returns the SSE data.
func mcpRequest(t *testing.T, method string, params any, sessionID string) (string, string) {
	t.Helper()

	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		body["params"] = params
	}

	b, _ := json.Marshal(body)
	url := mcpServerURL + "/mcp"

	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	raw := buf.String()

	// Extract session ID from headers
	newSession := resp.Header.Get("Mcp-Session-Id")

	// Parse SSE data
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			return strings.TrimPrefix(line, "data: "), newSession
		}
	}
	t.Fatalf("no SSE data in response: %s", raw)
	return "", newSession
}

// initializeSession creates a new MCP session and returns the session ID.
func initializeSession(t *testing.T) string {
	t.Helper()

	initParams := map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "integration-test",
			"version": "1.0",
		},
	}

	data, sessionID := mcpRequest(t, "initialize", initParams, "")
	if sessionID == "" {
		t.Fatalf("no session ID returned: %s", data)
	}
	return sessionID
}

// callTool sends a tools/call request and returns the result.
func callTool(t *testing.T, sessionID, toolName string, args map[string]any) string {
	t.Helper()

	params := map[string]any{
		"name":      toolName,
		"arguments": args,
	}
	data, _ := mcpRequest(t, "tools/call", params, sessionID)

	// Parse the response to check for errors
	var resp struct {
		Result *struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("parse response: %v, raw: %s", err, data)
	}
	if resp.Result == nil {
		t.Fatalf("no result in response: %s", data)
	}
	if len(resp.Result.Content) == 0 {
		t.Fatalf("no content in result: %s", data)
	}
	return resp.Result.Content[0].Text
}

func startMCPServer(t *testing.T) func() {
	t.Helper()
	// The MCP server should already be running.
	// If not, the test will fail fast with connection refused.
	resp, err := http.Get(mcpServerURL + "/healthz")
	if err != nil {
		t.Skipf("MCP server not running at %s: %v. Start it with: MCP_TRANSPORT=http go run ./cmd/server", mcpServerURL, err)
	}
	resp.Body.Close()

	return func() {} // no cleanup needed
}

func TestIntegration_ListProviders(t *testing.T) {
	cleanup := startMCPServer(t)
	defer cleanup()

	sessionID := initializeSession(t)
	result := callTool(t, sessionID, "list_providers", map[string]any{})

	t.Logf("list_providers result length: %d", len(result))
	if len(result) == 0 {
		t.Error("empty result from list_providers")
	}
}

func TestIntegration_ListUsers(t *testing.T) {
	cleanup := startMCPServer(t)
	defer cleanup()

	sessionID := initializeSession(t)
	result := callTool(t, sessionID, "list_users", map[string]any{})

	t.Logf("list_users result length: %d", len(result))
	if len(result) == 0 {
		t.Error("empty result from list_users")
	}
}

func TestIntegration_ListTokens(t *testing.T) {
	cleanup := startMCPServer(t)
	defer cleanup()

	sessionID := initializeSession(t)
	result := callTool(t, sessionID, "list_tokens", map[string]any{"page": 1, "page_size": 10})

	t.Logf("list_tokens result: %s", result)
	if len(result) == 0 {
		t.Error("empty result from list_tokens")
	}
}

func TestIntegration_AddChannel_SingleKey(t *testing.T) {
	cleanup := startMCPServer(t)
	defer cleanup()

	sessionID := initializeSession(t)

	name := fmt.Sprintf("int-test-single-%d", time.Now().Unix())
	result := callTool(t, sessionID, "add_channel", map[string]any{
		"name":   name,
		"type":   1,
		"key":    "sk-int-test-key",
		"models": "gpt-4,gpt-3.5-turbo",
		"group":  "default",
	})

	t.Logf("add_channel result: %s", result)
	if strings.Contains(result, `"success":false`) {
		t.Errorf("add_channel failed: %s", result)
	}
}

func TestIntegration_AddChannelKeys(t *testing.T) {
	cleanup := startMCPServer(t)
	defer cleanup()

	sessionID := initializeSession(t)

	// First, list channels to get an existing channel ID
	result := callTool(t, sessionID, "list_providers", map[string]any{})
	t.Logf("list_providers: %s", result[:min(len(result), 200)])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}