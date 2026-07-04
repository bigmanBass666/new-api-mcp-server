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

func TestSetUserQuota_Success(t *testing.T) {
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
	tool := NewSetUserQuotaTool(c, nil)

	// Simulate a call with JSON arguments
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 1, "quota": 100000}`),
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
	if gotPath != "/api/user/1" {
		t.Errorf("expected /api/user/1, got %s", gotPath)
	}
	q, ok := gotBody["quota"]
	if !ok {
		t.Fatal("expected quota in request body")
	}
	if q != float64(100000) { // JSON numbers decode as float64
		t.Errorf("expected quota 100000, got %v", q)
	}
}

func TestSetUserQuota_ZeroQuota(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewSetUserQuotaTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 1, "quota": 0}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler returned IsError=true for valid zero quota: %s", result.Content[0].(*mcp.TextContent).Text)
	}
}

func TestSetUserQuota_NonIntegerID(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSetUserQuotaTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": "abc", "quota": 100000}`),
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

func TestSetUserQuota_NegativeQuota(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSetUserQuotaTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 1, "quota": -100}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for negative quota")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "quota must be a non-negative integer, got -100" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestSetUserQuota_NonIntegerQuota(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSetUserQuotaTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 1, "quota": 1.5}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for non-integer quota (float)")
	}
}

func TestSetUserQuota_StringQuota(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSetUserQuotaTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 1, "quota": "100000"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for string quota")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "quota must be an integer, got string" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestSetUserQuota_UpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewSetUserQuotaTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"id": 1, "quota": 5}`),
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

func TestSetUserQuota_MissingID(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSetUserQuotaTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"quota": 5}`),
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

func TestSetUserQuota_MissingQuota(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewSetUserQuotaTool(c, nil)

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
		t.Fatal("expected IsError=true for missing quota")
	}
}