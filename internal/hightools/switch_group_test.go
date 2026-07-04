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

func TestSwitchGroup_Success(t *testing.T) {
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
	tool := NewSwitchGroupTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"token_id": 5, "group": "vip"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler returned IsError=true for valid input: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	if gotMethod != "PUT" {
		t.Errorf("expected PUT, got %s", gotMethod)
	}
	if gotPath != "/api/token/" {
		t.Errorf("expected /api/token/, got %s", gotPath)
	}

	idVal, ok := gotBody["id"]
	if !ok {
		t.Fatal("expected id in request body")
	}
	if idVal != float64(5) {
		t.Errorf("expected id=5, got %v", idVal)
	}
	group, ok := gotBody["group"]
	if !ok {
		t.Fatal("expected group in request body")
	}
	if group != "vip" {
		t.Errorf("expected group=vip, got %v", group)
	}
}

func TestSwitchGroup_MissingTokenID(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSwitchGroupTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"group": "default"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for missing token_id")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "missing required argument: token_id" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestSwitchGroup_MissingGroup(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSwitchGroupTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"token_id": 5}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for missing group")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "missing required argument: group" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestSwitchGroup_NonIntegerTokenID(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSwitchGroupTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"token_id": "abc", "group": "default"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for non-integer token_id")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "token_id must be an integer, got string" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestSwitchGroup_EmptyGroup(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSwitchGroupTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"token_id": 5, "group": ""}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for empty group")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "group must not be empty" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestSwitchGroup_UpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "token error"}`))
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewSwitchGroupTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"token_id": 5, "group": "vip"}`),
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
	if text != `{"error": "token error"}` {
		t.Errorf("expected upstream error body, got: %s", text)
	}
}