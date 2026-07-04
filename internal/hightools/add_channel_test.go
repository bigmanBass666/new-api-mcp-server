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

func TestAddChannel_Success(t *testing.T) {
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
		w.Write([]byte(`{"id": 1, "success": true}`))
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewAddChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"name":"test-channel","type":1,"key":"sk-test","models":"gpt-4","group":"default","priority":5}`),
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
	if gotPath != "/api/channel/" {
		t.Errorf("expected /api/channel/, got %s", gotPath)
	}

	// Verify mode wrapper
	mode, ok := gotBody["mode"]
	if !ok {
		t.Fatal("expected mode in request body")
	}
	if mode != "single" {
		t.Errorf("expected mode=single, got %v", mode)
	}

	// Verify channel object
	chRaw, ok := gotBody["channel"]
	if !ok {
		t.Fatal("expected channel in request body")
	}
	ch, ok := chRaw.(map[string]any)
	if !ok {
		t.Fatalf("channel is not a map, got %T", chRaw)
	}
	if ch["name"] != "test-channel" {
		t.Errorf("expected name=test-channel, got %v", ch["name"])
	}
	if ch["type"] != float64(1) {
		t.Errorf("expected type=1, got %v", ch["type"])
	}
	if ch["key"] != "sk-test" {
		t.Errorf("expected key=sk-test, got %v", ch["key"])
	}
	if ch["models"] != "gpt-4" {
		t.Errorf("expected models=gpt-4, got %v", ch["models"])
	}
	if ch["group"] != "default" {
		t.Errorf("expected group=default, got %v", ch["group"])
	}
	if ch["priority"] != float64(5) {
		t.Errorf("expected priority=5, got %v", ch["priority"])
	}
}

func TestAddChannel_RequiredFieldsOnly(t *testing.T) {
	var (
		gotMethod string
		gotBody   map[string]any
	)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&gotBody)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 2, "success": true}`))
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewAddChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"name":"minimal","type":3,"key":"sk-minimal"}`),
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

	chRaw, ok := gotBody["channel"]
	if !ok {
		t.Fatal("expected channel in request body")
	}
	ch, ok := chRaw.(map[string]any)
	if !ok {
		t.Fatalf("channel is not a map, got %T", chRaw)
	}
	if ch["name"] != "minimal" {
		t.Errorf("expected name=minimal, got %v", ch["name"])
	}
	if ch["type"] != float64(3) {
		t.Errorf("expected type=3, got %v", ch["type"])
	}
	if ch["key"] != "sk-minimal" {
		t.Errorf("expected key=sk-minimal, got %v", ch["key"])
	}
	// Optional fields should not be present
	if _, ok := ch["models"]; ok {
		t.Error("unexpected models field in minimal request")
	}
	if _, ok := ch["group"]; ok {
		t.Error("unexpected group field in minimal request")
	}
}

func TestAddChannel_MissingName(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewAddChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"type":1,"key":"sk-test"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for missing name")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "required parameter 'name' is missing" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestAddChannel_MissingType(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewAddChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"name":"test","key":"sk-test"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for missing type")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "required parameter 'type' is missing" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestAddChannel_MissingKey(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewAddChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"name":"test","type":1}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for missing key")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "required parameter 'key' is missing" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestAddChannel_TypeNotInteger(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewAddChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"name":"test","type":"openai","key":"sk-test"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for non-integer type")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "type must be an integer, got string" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestAddChannel_PriorityNotInteger(t *testing.T) {
	c := client.New("http://dummy", "", "sk-system", "", 0)
	tool := NewAddChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"name":"test","type":1,"key":"sk-test","priority":"high"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for non-integer priority")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "priority must be an integer, got string" {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestAddChannel_UpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer upstream.Close()

	c := client.New(upstream.URL, "", "sk-system", "", 0)
	tool := NewAddChannelTool(c, nil)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"name":"test","type":1,"key":"sk-test"}`),
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