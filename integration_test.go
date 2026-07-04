//go:build integration

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
	"github.com/QuantumNous/new-api-mcp-server/internal/handler"
	"github.com/QuantumNous/new-api-mcp-server/internal/hightools"
	openapipkg "github.com/QuantumNous/new-api-mcp-server/internal/openapi"
	"github.com/QuantumNous/new-api-mcp-server/internal/registry"
	embeddedSpecs "github.com/QuantumNous/new-api-mcp-server/openapi"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestIntegration_FullPipeline(t *testing.T) {
	// Mock upstream API
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"path":   r.URL.Path,
			"method": r.Method,
			"auth":   r.Header.Get("Authorization"),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer upstream.Close()

	// Parse relay spec
	defs, err := openapipkg.Parse(embeddedSpecs.RelaySpec)
	if err != nil {
		t.Fatalf("parse relay spec: %v", err)
	}
	t.Logf("Parsed %d relay tool definitions", len(defs))

	// Create MCP server with tools
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1"}, nil)
	c := client.New(upstream.URL, "sk-test", "", "", 5*time.Second)
	h := handler.New(c, client.SourceRelay, nil)

	count := registry.RegisterTools(server, defs, registry.Options{
		AllGroups: true,
	}, h.MakeHandler)
	t.Logf("Registered %d relay tools", count)

	if count == 0 {
		t.Fatal("expected at least 1 registered tool")
	}

	// Connect an in-memory client and call a tool
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	go server.Run(context.Background(), serverTransport)

	session, err := mcpClient.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("connect error: %v", err)
	}

	// List tools
	toolsResult, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	t.Logf("Listed %d tools", len(toolsResult.Tools))

	if len(toolsResult.Tools) == 0 {
		t.Fatal("no tools listed")
	}

	// Call the first available tool
	firstTool := toolsResult.Tools[0]
	callResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      firstTool.Name,
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(%s) error: %v", firstTool.Name, err)
	}
	t.Logf("Called tool %s, isError=%v", firstTool.Name, callResult.IsError)

	if len(callResult.Content) == 0 {
		t.Error("expected content in result")
	}
}

func TestIntegration_HighLevelTools(t *testing.T) {
	// Mock upstream API that handles endpoints used by high-level tools
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && r.URL.Path == "/api/channel/":
			// Mock response for list_providers
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"message": "",
				"data": map[string]any{
					"items": []map[string]any{
						{
							"id": 1, "name": "channel-01", "type": 1, "status": 1,
							"models": "gpt-4,gpt-3.5-turbo", "groups": "default,vip",
							"priority": 10, "weight": 0, "base_url": "http://example.com", "tag": "",
						},
						{
							"id": 2, "name": "channel-02", "type": 2, "status": 0,
							"models": "claude-3", "groups": "default",
							"priority": 5, "weight": 0, "base_url": "http://example2.com", "tag": "",
						},
					},
					"total":      2,
					"page":       1,
					"page_size":  100,
					"type_counts": map[string]any{},
				},
			})

		default:
			// Generic echo response for other endpoints
			json.NewEncoder(w).Encode(map[string]any{
				"path":   r.URL.Path,
				"method": r.Method,
				"auth":   r.Header.Get("Authorization"),
			})
		}
	}))
	defer upstream.Close()

	// Create client with both relay key and system key (system key is needed for SourceAPI)
	c := client.New(upstream.URL, "sk-test-relay", "sk-test-system", "", 5*time.Second)

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1"}, nil)

	// Parse and register relay tools (base layer)
	relayDefs, err := openapipkg.Parse(embeddedSpecs.RelaySpec)
	if err != nil {
		t.Fatalf("parse relay spec: %v", err)
	}
	relayHandler := handler.New(c, client.SourceRelay, nil)
	relayCount := registry.RegisterTools(server, relayDefs, registry.Options{
		AllGroups: true,
	}, relayHandler.MakeHandler)
	t.Logf("Registered %d relay tools", relayCount)

	// Parse and register API tools (base layer)
	apiDefs, err := openapipkg.Parse(embeddedSpecs.APISpec)
	if err != nil {
		t.Fatalf("parse api spec: %v", err)
	}
	apiHandler := handler.New(c, client.SourceAPI, nil)
	apiCount := registry.RegisterTools(server, apiDefs, registry.Options{
		AllGroups:  true,
		NamePrefix: "api_",
	}, apiHandler.MakeHandler)
	t.Logf("Registered %d API tools", apiCount)

	// Register high-level tools
	highDefs := hightools.RegisterAll(c, nil)
	for _, def := range highDefs {
		tool := &mcp.Tool{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: def.InputSchema,
		}
		server.AddTool(tool, def.Handler)
	}
	t.Logf("Registered %d high-level tools", len(highDefs))

	if len(highDefs) == 0 {
		t.Fatal("expected at least 1 high-level tool")
	}

	// Connect MCP client via in-memory transport
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	go server.Run(context.Background(), serverTransport)

	session, err := mcpClient.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("connect error: %v", err)
	}

	// List tools and verify high-level tools are present
	toolsResult, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	t.Logf("Listed %d tools total", len(toolsResult.Tools))

	if len(toolsResult.Tools) == 0 {
		t.Fatal("no tools listed")
	}

	// Build a set of tool names for easy lookup
	toolNames := make(map[string]bool)
	for _, tt := range toolsResult.Tools {
		toolNames[tt.Name] = true
	}

	// Verify every high-level tool is in the listed tools
	for _, def := range highDefs {
		if !toolNames[def.Name] {
			t.Errorf("expected high-level tool %q not found in ListTools results", def.Name)
		} else {
			t.Logf("Found high-level tool: %s", def.Name)
		}
	}

	// Call list_providers tool to verify end-to-end invocation works
	callResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_providers",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(list_providers) error: %v", err)
	}
	t.Logf("Called list_providers, isError=%v", callResult.IsError)

	if callResult.IsError {
		errText := "(no content)"
		if len(callResult.Content) > 0 {
			errText = callResult.Content[0].(*mcp.TextContent).Text
		}
		t.Fatalf("list_providers returned IsError=true: %s", errText)
	}
	if len(callResult.Content) == 0 {
		t.Fatal("expected content in list_providers result")
	}

	// Verify output contains expected channel info from the mock upstream
	text := callResult.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "channel-01") {
		t.Errorf("expected output to contain 'channel-01', got:\n%s", text)
	}
	if !strings.Contains(text, "channel-02") {
		t.Errorf("expected output to contain 'channel-02', got:\n%s", text)
	}
}
