package hightools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestTasksCancel_RunningTask(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("channel_test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)

	tool := NewTasksCancelTool(tm)
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
	if !strings.Contains(text, "cancelled") {
		t.Errorf("expected 'cancelled' in output, got: %s", text)
	}

	got, _ := tm.GetTask(task.ID)
	if got.State != TaskStateCancelled {
		t.Errorf("expected cancelled state, got %s", got.State)
	}
}

func TestTasksCancel_InputRequiredTask(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("channel_test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)
	tm.UpdateTask(task.ID, TaskStateInputRequired, WithError("channel error"))

	tool := NewTasksCancelTool(tm)
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

	got, _ := tm.GetTask(task.ID)
	if got.State != TaskStateCancelled {
		t.Errorf("expected cancelled state, got %s", got.State)
	}
}

func TestTasksCancel_AlreadyTerminal(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("channel_test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)
	tm.UpdateTask(task.ID, TaskStateSucceeded)

	tool := NewTasksCancelTool(tm)
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"task_id":"` + task.ID + `"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for already terminal task")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "already in terminal") && !strings.Contains(text, "terminal") {
		t.Errorf("expected terminal state error, got: %s", text)
	}
}

func TestTasksCancel_NonExistentTask(t *testing.T) {
	tm := NewTaskManager()
	tool := NewTasksCancelTool(tm)

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
}

func TestTasksCancel_MissingTaskID(t *testing.T) {
	tm := NewTaskManager()
	tool := NewTasksCancelTool(tm)

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