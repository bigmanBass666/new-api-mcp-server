package hightools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewTasksCancelTool returns a ToolDef for cancelling a running or
// input_required task.
func NewTasksCancelTool(tm *TaskManager) ToolDef {
	return ToolDef{
		Name:        "tasks_cancel",
		Description: "Cancel a running or input_required task by ID. Once cancelled, the task cannot be resumed.",
		InputSchema: inputSchemaTasksCancel(),
		Handler:     handleTasksCancel(tm),
	}
}

func inputSchemaTasksCancel() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "Task ID to cancel",
			},
		},
		"required": []any{"task_id"},
	}
}

func handleTasksCancel(tm *TaskManager) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultTasksCancel(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return errorResultTasksCancel("missing required argument: task_id"), nil
		}

		slog.DebugContext(ctx, "tasks_cancel called", "task_id", taskID)

		if err := tm.CancelTask(taskID); err != nil {
			return errorResultTasksCancel(fmt.Sprintf("failed to cancel task: %v", err)), nil
		}

		slog.InfoContext(ctx, "task cancelled successfully", "task_id", taskID)

		result := fmt.Sprintf(`{"success":true,"task_id":"%s","state":"cancelled"}`, taskID)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil
	}
}

func errorResultTasksCancel(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}