// Package hightools provides a container for high-level MCP tools.
//
// High-level tools differ from the auto-generated API proxy tools in
// internal/openapi/:
//   - They are constructed programmatically, not parsed from OpenAPI specs.
//   - A ToolDef carries its Handler directly, rather than relying on the
//     generic HTTP proxy in internal/handler.
//   - They can aggregate multiple upstream calls, perform logic, or present
//     a simplified interface over the raw New API endpoints.
//
// Each tool has its own constructor function (see register.go for the list).
package hightools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// ToolDef describes a high-level MCP tool.
//
// Unlike openapi.ToolDef (which maps to a single upstream HTTP endpoint),
// ToolDef bundles a ready-to-use mcp.ToolHandler. The Handler field
// implements the tool's logic directly.
type ToolDef struct {
	// Name is the MCP tool identifier, matching [a-zA-Z0-9_\-.].
	Name string

	// Description is a human-readable description shown to the LLM.
	Description string

	// InputSchema is a JSON Schema (2020-12 draft) defining the expected
	// parameters. Must be non-nil with type "object".
	InputSchema any

	// Handler implements the tool's logic.
	Handler mcp.ToolHandler
}