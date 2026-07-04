package hightools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToggleChannel_Enable(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]any
	)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&gotBody)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewToggleChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 42, "enabled": true}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler returned IsError=true for valid input: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	if gotMethod != "POST" {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/api/channel/42/status" {
		t.Errorf("expected /api/channel/42/status, got %s", gotPath)
	}

	statusVal, ok := gotBody["status"]
	if !ok {
		t.Fatal("expected status in request body")
	}
	if statusVal != float64(1) {
		t.Errorf("expected status=1 for enable, got %v", statusVal)
	}
}

func TestToggleChannel_Disable(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]any
	)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&gotBody)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewToggleChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 7, "enabled": false}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler returned IsError=true for valid input: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	if gotMethod != "POST" {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/api/channel/7/status" {
		t.Errorf("expected /api/channel/7/status, got %s", gotPath)
	}

	statusVal, ok := gotBody["status"]
	if !ok {
		t.Fatal("expected status in request body")
	}
	if statusVal != float64(2) {
		t.Errorf("expected status=2 for disable, got %v", statusVal)
	}
}

func TestToggleChannel_MissingID(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewToggleChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"enabled": true}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for missing id")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "missing required argument: id" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestToggleChannel_MissingEnabled(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewToggleChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 1}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for missing enabled")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "missing required argument: enabled" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestToggleChannel_NonIntegerID(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewToggleChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": "abc", "enabled": true}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for non-integer id")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "id must be an integer, got string" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestToggleChannel_UpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "channel error"}`))
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewToggleChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 1, "enabled": true}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for upstream 500 error")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != `{"error": "channel error"}` {
		t.Errorf("expected upstream error body, got: %s", text)
	}
}