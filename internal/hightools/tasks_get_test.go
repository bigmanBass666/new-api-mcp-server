package hightools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestTasksGet_ExistingTask(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("channel_test", map[string]any{"source": "test"})
	tm.UpdateTask(task.ID, TaskStateRunning)
	tm.UpdateTask(task.ID, TaskStateSucceeded, WithResult("ok"))

	tool := NewTasksGetTool(tm)
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"task_id":"` + task.ID + `"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler returned IsError=true")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	t.Logf("Output:\n%s", text)

	if !strings.Contains(text, task.ID) {
		t.Errorf("expected task ID in output")
	}
	if !strings.Contains(text, "succeeded") {
		t.Errorf("expected 'succeeded' in output")
	}
	if !strings.Contains(text, "channel_test") {
		t.Errorf("expected 'channel_test' in output")
	}
}

func TestTasksGet_NonExistentTask(t *testing.T) {
	tm := NewTaskManager()
	tool := NewTasksGetTool(tm)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"task_id":"nonexistent"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for non-existent task")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "not found") {
		t.Errorf("expected 'not found' error, got: %s", text)
	}
}

func TestTasksGet_MissingTaskID(t *testing.T) {
	tm := NewTaskManager()
	tool := NewTasksGetTool(tm)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for missing task_id")
	}
}

func TestTasksGet_WithProgress(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("channel_test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning, WithProgress(0.5))

	tool := NewTasksGetTool(tm)
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"task_id":"` + task.ID + `"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler returned IsError=true")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "running") {
		t.Errorf("expected 'running' state, got: %s", text)
	}
}