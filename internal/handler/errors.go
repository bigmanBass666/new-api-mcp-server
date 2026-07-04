package handler

import (
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ErrorCode represents a structured error category for MCP tool calls.
type ErrorCode string

const (
	// ErrInvalidParams indicates missing or invalid parameters.
	ErrInvalidParams ErrorCode = "INVALID_PARAMS"
	// ErrUpstreamError indicates the upstream API returned a non-2xx response.
	ErrUpstreamError ErrorCode = "UPSTREAM_ERROR"
	// ErrUpstreamTimeout indicates the upstream request timed out.
	ErrUpstreamTimeout ErrorCode = "UPSTREAM_TIMEOUT"
	// ErrUpstreamAuth indicates the upstream returned 401 or 403.
	ErrUpstreamAuth ErrorCode = "UPSTREAM_AUTH"
	// ErrInternal indicates an internal processing error (JSON marshal, etc.).
	ErrInternal ErrorCode = "INTERNAL_ERROR"
	// ErrUpstreamNotFound indicates the upstream returned 404.
	ErrUpstreamNotFound ErrorCode = "UPSTREAM_NOT_FOUND"
)

// ToolError is a structured error returned in MCP tool results.
type ToolError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	StatusCode int       `json:"status_code,omitempty"`
}

// NewError creates a ToolError with the given code and message.
func NewError(code ErrorCode, msg string) *ToolError {
	return &ToolError{Code: code, Message: msg}
}

// errorResult returns an MCP CallToolResult with IsError=true and structured
// JSON content containing the error code and message.
func errorResult(code ErrorCode, msg string) *mcp.CallToolResult {
	te := ToolError{Code: code, Message: msg}
	data, _ := json.Marshal(te)
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}
}