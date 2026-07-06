package hightools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewTasksUpdateTool returns a ToolDef for updating an async task with user input.
//
// This is used to submit decisions for tasks in input_required state. The action
// can be "resume" (provide input and continue) or "retry" (retry the task).
func NewTasksUpdateTool(tm *TaskManager) ToolDef {
	return ToolDef{
		Name:        "tasks_update",
		Description: "Update an async task with user input (e.g., submitting a decision for input_required tasks). Use 'resume' to provide input and continue, or 'retry' to retry the task.",
		InputSchema: inputSchemaTasksUpdate(),
		Handler:     handleTasksUpdate(tm),
	}
}

func inputSchemaTasksUpdate() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "Task ID to update",
			},
			"action": map[string]any{
				"type":        "string",
				"enum":        []any{"resume", "retry"},
				"description": "Action to perform on the task",
			},
			"payload": map[string]any{
				"type":        "object",
				"description": "Optional payload to pass to the task on resume",
			},
		},
		"required": []any{"task_id", "action"},
	}
}

func handleTasksUpdate(tm *TaskManager) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args map[string]any
		if req.Params != nil && len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResultTasksUpdate(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return errorResultTasksUpdate("missing required argument: task_id"), nil
		}

		action, ok := args["action"].(string)
		if !ok || action == "" {
			return errorResultTasksUpdate("missing required argument: action"), nil
		}

		payload, _ := args["payload"].(map[string]any)

		slog.DebugContext(ctx, "tasks_update called", "task_id", taskID, "action", action)

		// Check task exists and is in input_required state
		task, err := tm.GetTask(taskID)
		if err != nil {
			return errorResultTasksUpdate(fmt.Sprintf("task not found: %s", taskID)), nil
		}

		if task.State != TaskStateInputRequired {
			return errorResultTasksUpdate(fmt.Sprintf("task %s is not waiting for input (current state: %s)", taskID, task.State)), nil
		}

		// Signal resume with action and payload
		signal := ResumeSignal{
			Action:  action,
			Payload: payload,
		}
		if err := tm.SignalResume(taskID, signal); err != nil {
			return errorResultTasksUpdate(fmt.Sprintf("failed to signal resume: %v", err)), nil
		}

		// Transition back to running
		if err := tm.UpdateTask(taskID, TaskStateRunning); err != nil {
			return errorResultTasksUpdate(fmt.Sprintf("failed to update task state: %v", err)), nil
		}

		slog.InfoContext(ctx, "task updated successfully",
			"task_id", taskID,
			"action", action,
		)

		result := fmt.Sprintf(`{"success":true,"task_id":"%s","state":"running","action":"%s"}`, taskID, action)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil
	}
}

func errorResultTasksUpdate(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}