package hightools

import (
	"github.com/QuantumNous/new-api-mcp-server/internal/client"
	"github.com/QuantumNous/new-api-mcp-server/internal/observability"
)

// RegisterAll returns the full list of high-level tool definitions.
//
// Each entry is produced by an independent constructor function, making
// them easy to add, remove, or reorder. Callers iterate the returned slice
// and register each ToolDef on the MCP server.
func RegisterAll(c *client.Client, metrics *observability.Metrics) []ToolDef {
	return []ToolDef{
		NewSetChannelPriorityTool(c, metrics),
		NewToggleChannelTool(c, metrics),
		NewToggleUserStatusTool(c, metrics),
		NewTestAndReportTool(c, metrics),
		NewListProvidersTool(c, metrics),
		NewSetUserQuotaTool(c, metrics),
		NewShowBalanceTool(c, metrics),
		NewListUsersTool(c, metrics),
		NewSwitchGroupTool(c, metrics),
		NewAddChannelTool(c, metrics),
	}
}