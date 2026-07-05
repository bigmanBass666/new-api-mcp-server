package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
	"github.com/QuantumNous/new-api-mcp-server/internal/openapi"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHandle_SimpleGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/items" {
			t.Errorf("path = %s, want /api/items", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[]}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "sk-key", "", "", 5*time.Second)
	def := openapi.ToolDef{
		Name:   "listItems",
		Method: "GET",
		Path:   "/api/items",
	}

	h := New(c, client.SourceRelay, nil)
	handler := h.MakeHandler(def)

	req := &mcp.CallToolRequest{}
	req.Params = &mcp.CallToolParamsRaw{
		Name:      "listItems",
		Arguments: json.RawMessage(`{}`),
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatal("result.IsError = true")
	}
	if len(result.Content) == 0 {
		t.Fatal("no content in result")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != `{"items":[]}` {
		t.Errorf("result text = %q, want %q", text, `{"items":[]}`)
	}
}

func TestHandle_PathParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/items/42" {
			t.Errorf("path = %s, want /api/items/42", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":42}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "sk-key", "", "", 5*time.Second)
	def := openapi.ToolDef{
		Name:   "getItem",
		Method: "GET",
		Path:   "/api/items/{id}",
		PathParams: []openapi.ParamDef{
			{Name: "id", In: "path", Required: true},
		},
	}

	h := New(c, client.SourceRelay, nil)
	handler := h.MakeHandler(def)

	req := &mcp.CallToolRequest{}
	req.Params = &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{"id": 42}`)}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != `{"id":42}` {
		t.Errorf("result text = %q", text)
	}
}

func TestHandle_QueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("query limit = %q, want 10", r.URL.Query().Get("limit"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "sk-key", "", "", 5*time.Second)
	def := openapi.ToolDef{
		Name:   "listItems",
		Method: "GET",
		Path:   "/api/items",
		QueryParams: []openapi.ParamDef{
			{Name: "limit", In: "query"},
		},
	}

	h := New(c, client.SourceRelay, nil)
	handler := h.MakeHandler(def)

	req := &mcp.CallToolRequest{}
	req.Params = &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{"limit": 10}`)}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatal("result.IsError = true")
	}
}

func TestHandle_RequestBody(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = json.Marshal(nil) // init
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		gotBody = buf[:n]
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"created":true}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "sk-key", "", "", 5*time.Second)
	def := openapi.ToolDef{
		Name:    "createItem",
		Method:  "POST",
		Path:    "/api/items",
		HasBody: true,
	}

	h := New(c, client.SourceRelay, nil)
	handler := h.MakeHandler(def)

	req := &mcp.CallToolRequest{}
	req.Params = &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{"body": {"name": "test"}}`)}

	_, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if len(gotBody) == 0 {
		t.Error("expected request body, got empty")
	}
}

func TestHandle_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "sk-key", "", "", 5*time.Second)
	def := openapi.ToolDef{
		Name:   "getItem",
		Method: "GET",
		Path:   "/api/items/999",
	}

	h := New(c, client.SourceRelay, nil)
	handler := h.MakeHandler(def)

	req := &mcp.CallToolRequest{}
	req.Params = &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{}`)}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError = true for non-2xx response")
	}
	// Verify structured error contains code field
	if len(result.Content) > 0 {
		text := result.Content[0].(*mcp.TextContent).Text
		var errData map[string]any
		if err := json.Unmarshal([]byte(text), &errData); err == nil {
			if code, ok := errData["code"]; !ok || code == "" {
				t.Error("expected error to contain 'code' field")
			}
		}
	}
}

func TestHandle_NonJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47}) // PNG header bytes
	}))
	defer srv.Close()

	c := client.New(srv.URL, "sk-key", "", "", 5*time.Second)
	def := openapi.ToolDef{
		Name:   "getImage",
		Method: "GET",
		Path:   "/image",
	}

	h := New(c, client.SourceRelay, nil)
	handler := h.MakeHandler(def)

	req := &mcp.CallToolRequest{}
	req.Params = &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{}`)}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	// Non-JSON should be base64 encoded
	text := result.Content[0].(*mcp.TextContent).Text
	if text == string([]byte{0x89, 0x50, 0x4E, 0x47}) {
		t.Error("expected base64 encoded content for non-JSON response")
	}
}
func TestHandle_InvalidJSONParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach upstream with invalid args")
	}))
	defer srv.Close()

	c := client.New(srv.URL, "sk-key", "", "", 5*time.Second)
	def := openapi.ToolDef{
		Name:   "testTool",
		Method: "GET",
		Path:   "/api/test",
	}

	h := New(c, client.SourceRelay, nil)
	handler := h.MakeHandler(def)

	// Arguments is malformed JSON (not a valid JSON object)
	req := &mcp.CallToolRequest{}
	req.Params = &mcp.CallToolParamsRaw{
		Name:      "testTool",
		Arguments: json.RawMessage("{invalid}"),
	}

	result, err := handler(context.Background(), req)
	if err == nil {
		t.Fatal("expected protocol-level error for invalid JSON args")
	}
	if result != nil {
		t.Fatal("expected nil result for protocol-level error")
	}

	// Verify it's a jsonrpc.Error with code -32000
	var rpcErr *jsonrpc.Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *jsonrpc.Error, got %T", err)
	}
	if rpcErr.Code != -32000 {
		t.Errorf("expected code -32000, got %d", rpcErr.Code)
	}
	if !strings.Contains(rpcErr.Message, "invalid arguments") {
		t.Errorf("message should contain 'invalid arguments', got: %s", rpcErr.Message)
	}
}

func TestHandle_InvalidPathParamType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach upstream with invalid path param")
	}))
	defer srv.Close()

	c := client.New(srv.URL, "sk-key", "", "", 5*time.Second)
	def := openapi.ToolDef{
		Name:   "getItem",
		Method: "GET",
		Path:   "/api/items/{id}",
		PathParams: []openapi.ParamDef{
			{Name: "id", In: "path", Required: true, Schema: map[string]any{"type": "integer"}},
		},
	}

	h := New(c, client.SourceRelay, nil)
	handler := h.MakeHandler(def)

	// Send a non-integer path param
	req := &mcp.CallToolRequest{}
	req.Params = &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{"id": "abc"}`)}

	result, err := handler(context.Background(), req)
	if err == nil {
		t.Fatal("expected protocol-level error for invalid path param type")
	}
	if result != nil {
		t.Fatal("expected nil result for protocol-level error")
	}

	// Verify it's a jsonrpc.Error with code -32000
	var rpcErr *jsonrpc.Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *jsonrpc.Error, got %T", err)
	}
	if rpcErr.Code != -32000 {
		t.Errorf("expected code -32000, got %d", rpcErr.Code)
	}
	if !strings.Contains(rpcErr.Message, `path param "id" must be an integer`) {
		t.Errorf("message should mention path param validation, got: %s", rpcErr.Message)
	}
}
