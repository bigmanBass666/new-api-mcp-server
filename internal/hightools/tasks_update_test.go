package hightools

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestTasksUpdate_ResumeInputRequired(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("channel_test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)
	tm.UpdateTask(task.ID, TaskStateInputRequired, WithError("channel timeout"))

	// Start a goroutine to wait for resume (simulating the background worker)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		signal, err := tm.WaitForResume(task.ID)
		if err != nil {
			t.Logf("WaitForResume error: %v", err)
			return
		}
		if signal.Action != "resume" {
			t.Errorf("expected action 'resume', got %s", signal.Action)
		}
	}()

	// Call tasks_update to resume
	tool := NewTasksUpdateTool(tm)
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"task_id":"` + task.ID + `","action":"resume"}`),
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
		t.Errorf("expected 'running' in output, got: %s", text)
	}

	// Wait for the goroutine to receive the signal
	wg.Wait()

	// Verify task is back to running
	got, _ := tm.GetTask(task.ID)
	if got.State != TaskStateRunning {
		t.Errorf("expected running state after resume, got %s", got.State)
	}
}

func TestTasksUpdate_NotInputRequired(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("channel_test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)

	tool := NewTasksUpdateTool(tm)
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"task_id":"` + task.ID + `","action":"resume"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for non-input_required task")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "not waiting for input") {
		t.Errorf("expected 'not waiting for input' error, got: %s", text)
	}
}

func TestTasksUpdate_NonExistentTask(t *testing.T) {
	tm := NewTaskManager()
	tool := NewTasksUpdateTool(tm)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"task_id":"nonexistent","action":"resume"}`),
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

func TestTasksUpdate_MissingAction(t *testing.T) {
	tm := NewTaskManager()
	tool := NewTasksUpdateTool(tm)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"task_id":"some-id"}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for missing action")
	}
}

func TestTasksUpdate_WithPayload(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("channel_test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)
	tm.UpdateTask(task.ID, TaskStateInputRequired, WithError("timeout"))

	var receivedSignal ResumeSignal
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		signal, _ := tm.WaitForResume(task.ID)
		receivedSignal = *signal
	}()

	// Give the goroutine time to start waiting
	time.Sleep(50 * time.Millisecond)

	tool := NewTasksUpdateTool(tm)
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"task_id":"` + task.ID + `","action":"retry","payload":{"max_retries":3}}`),
		},
	}
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Handler returned IsError=true")
	}

	wg.Wait()

	if receivedSignal.Action != "retry" {
		t.Errorf("expected action 'retry', got %s", receivedSignal.Action)
	}
	payload, ok := receivedSignal.Payload["max_retries"].(float64)
	if !ok || payload != 3 {
		t.Errorf("expected payload max_retries=3, got %v", receivedSignal.Payload["max_retries"])
	}
}