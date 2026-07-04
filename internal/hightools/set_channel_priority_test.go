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

func TestSetChannelPriority_Success(t *testing.T) {
	// Mock upstream that captures the PUT request
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
	tool := NewSetChannelPriorityTool(c, nil)

	// Simulate a call with JSON arguments
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 42, "priority": 10}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler returned IsError=true for valid input: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	// Verify upstream was called correctly
	if gotMethod != "PUT" {
		t.Errorf("expected PUT, got %s", gotMethod)
	}
	if gotPath != "/api/channel/" {
		t.Errorf("expected /api/channel/, got %s", gotPath)
	}
	idVal, ok := gotBody["id"]
	if !ok {
		t.Fatal("expected id in request body")
	}
	if idVal != float64(42) {
		t.Errorf("expected id 42, got %v", idVal)
	}
	prio, ok := gotBody["priority"]
	if !ok {
		t.Fatal("expected priority in request body")
	}
	if prio != float64(10) { // JSON numbers decode as float64
		t.Errorf("expected priority 10, got %v", prio)
	}
}

func TestSetChannelPriority_NonIntegerString(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSetChannelPriorityTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 1, "priority": "high"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for non-integer priority (string)")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "priority must be an integer, got string" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestSetChannelPriority_NonIntegerFloat(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSetChannelPriorityTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 1, "priority": 1.5}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for non-integer priority (float)")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "priority must be an integer, got float64" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestSetChannelPriority_UpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewSetChannelPriorityTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 1, "priority": 5}`),
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
	if text != `{"error": "internal error"}` {
		t.Errorf("expected upstream error body, got: %s", text)
	}
}

func TestSetChannelPriority_MissingID(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSetChannelPriorityTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"priority": 5}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for missing id")
	}
}

func TestSetChannelPriority_MissingPriority(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSetChannelPriorityTool(c, nil)

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
		t.Fatal("expected IsError=true for missing priority")
	}
}

func TestSetChannelPriority_NonIntegerID(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSetChannelPriorityTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": "abc", "priority": 5}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for non-integer id")
	}
}