package hightools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewTasksGetTool returns a ToolDef for querying the status of an async task.
func NewTasksGetTool(tm *TaskManager) ToolDef {
	return ToolDef{
		Name:        "tasks_get",
		Description: "Query the status of an async task by ID. Returns the full task state including progress, result, error, and metadata.",
		InputSchema: inputSchemaTasksGet(),
		Handler:     handleTasksGet(tm),
	}
}

func inputSchemaTasksGet() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "Task ID to query",
			},
		},
		"required": []any{"task_id"},
	}
}

func handleTasksGet(tm *TaskManager) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultTasksGet(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return errorResultTasksGet("missing required argument: task_id"), nil
		}

		slog.DebugContext(ctx, "tasks_get called", "task_id", taskID)

		task, err := tm.GetTask(taskID)
		if err != nil {
			return errorResultTasksGet(fmt.Sprintf("task not found: %s", taskID)), nil
		}

		output := formatTaskOutput(task)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: output},
			},
		}, nil
	}
}

// formatTaskOutput builds a structured JSON representation of a task.
func formatTaskOutput(task *Task) string {
	output := map[string]any{
		"id":         task.ID,
		"type":       task.Type,
		"state":      task.State,
		"progress":   task.Progress,
		"created_at": task.CreatedAt.Format(time.RFC3339),
		"updated_at": task.UpdatedAt.Format(time.RFC3339),
	}

	if task.Result != nil {
		output["result"] = task.Result
	}
	if task.Error != "" {
		output["error"] = task.Error
	}
	if len(task.Metadata) > 0 {
		output["metadata"] = task.Metadata
	}

	b, _ := json.MarshalIndent(output, "", "  ")
	return string(b)
}

func errorResultTasksGet(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}