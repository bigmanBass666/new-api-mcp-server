package hightools

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTaskManager_CreateAndGet(t *testing.T) {
	tm := NewTaskManager()

	task, err := tm.CreateTask("test_type", map[string]any{"key": "val"})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if task.ID == "" {
		t.Error("expected non-empty ID")
	}
	if task.Type != "test_type" {
		t.Errorf("expected type 'test_type', got %s", task.Type)
	}
	if task.State != TaskStatePending {
		t.Errorf("expected state 'pending', got %s", task.State)
	}
	if task.Metadata["key"] != "val" {
		t.Errorf("expected metadata key=val, got %v", task.Metadata["key"])
	}

	// Get a copy
	got, err := tm.GetTask(task.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if got.ID != task.ID {
		t.Errorf("expected ID %s, got %s", task.ID, got.ID)
	}
}

func TestTaskManager_GetNonExistent(t *testing.T) {
	tm := NewTaskManager()
	_, err := tm.GetTask("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestTaskManager_UpdateState(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("test", nil)

	// pending → running
	if err := tm.UpdateTask(task.ID, TaskStateRunning); err != nil {
		t.Fatalf("pending -> running should be valid: %v", err)
	}

	// running → succeeded
	if err := tm.UpdateTask(task.ID, TaskStateSucceeded, WithResult("done")); err != nil {
		t.Fatalf("running -> succeeded should be valid: %v", err)
	}

	got, _ := tm.GetTask(task.ID)
	if got.State != TaskStateSucceeded {
		t.Errorf("expected succeeded, got %s", got.State)
	}
	if got.Result != "done" {
		t.Errorf("expected result 'done', got %v", got.Result)
	}
}

func TestTaskManager_InvalidTransition(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("test", nil)

	// pending → succeeded (invalid, must go through running)
	err := tm.UpdateTask(task.ID, TaskStateSucceeded)
	if err == nil {
		t.Fatal("expected error for invalid transition pending -> succeeded")
	}
}

func TestTaskManager_TerminalStateRejectsUpdates(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("test", nil)

	tm.UpdateTask(task.ID, TaskStateRunning)
	tm.UpdateTask(task.ID, TaskStateSucceeded)

	// Terminal state rejects further updates
	if err := tm.UpdateTask(task.ID, TaskStateFailed); err == nil {
		t.Fatal("expected error updating terminal task")
	}
}

func TestTaskManager_CancelTask(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)

	if err := tm.CancelTask(task.ID); err != nil {
		t.Fatalf("CancelTask failed: %v", err)
	}

	got, _ := tm.GetTask(task.ID)
	if got.State != TaskStateCancelled {
		t.Errorf("expected cancelled, got %s", got.State)
	}

	// Cancel channel should be closed
	select {
	case <-tm.GetCancelCh(task.ID):
		// closed as expected
	default:
		t.Error("expected cancel channel to be closed")
	}
}

func TestTaskManager_CancelTerminalTask(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)
	tm.UpdateTask(task.ID, TaskStateSucceeded)

	if err := tm.CancelTask(task.ID); err == nil {
		t.Fatal("expected error cancelling terminal task")
	}
}

func TestTaskManager_CancelChForNonExistentTask(t *testing.T) {
	tm := NewTaskManager()
	ch := tm.GetCancelCh("nonexistent")
	select {
	case <-ch:
		// already closed (safe to read from)
	default:
		t.Error("expected closed channel for non-existent task")
	}
}

func TestTaskManager_InputRequiredCycle(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)
	tm.UpdateTask(task.ID, TaskStateInputRequired, WithError("channel timeout"))

	got, _ := tm.GetTask(task.ID)
	if got.State != TaskStateInputRequired {
		t.Errorf("expected input_required, got %s", got.State)
	}

	// Signal resume
	signal := ResumeSignal{Action: "resume", Payload: map[string]any{"skip": true}}
	if err := tm.SignalResume(task.ID, signal); err != nil {
		t.Fatalf("SignalResume failed: %v", err)
	}

	// Worker would call WaitForResume in a separate goroutine.
	// For this test we just verify SignalResume works without error.
}

func TestTaskManager_SignalResumeNonInputRequired(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)

	err := tm.SignalResume(task.ID, ResumeSignal{Action: "resume"})
	if err == nil {
		t.Fatal("expected error signalling non-input_required task")
	}
}

func TestTaskManager_ReapExpiredTasks(t *testing.T) {
	// Use a very short TTL for testing
	tm := NewTaskManagerWithOptions(50*time.Millisecond, 100)
	task1, _ := tm.CreateTask("test", nil)
	task2, _ := tm.CreateTask("test", nil)

	tm.UpdateTask(task1.ID, TaskStateRunning)
	tm.UpdateTask(task1.ID, TaskStateSucceeded)

	tm.UpdateTask(task2.ID, TaskStateRunning)
	tm.UpdateTask(task2.ID, TaskStateFailed)

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	count := tm.Reap()
	if count < 1 {
		t.Errorf("expected at least 1 reaped task, got %d", count)
	}

	// Terminal tasks should be removed
	if _, err := tm.GetTask(task1.ID); err == nil {
		t.Error("expected task1 to be reaped")
	}
}

func TestTaskManager_ReapPreservesActiveTasks(t *testing.T) {
	tm := NewTaskManagerWithOptions(50*time.Millisecond, 100)
	task, _ := tm.CreateTask("test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)

	time.Sleep(100 * time.Millisecond)
	count := tm.Reap()

	if count != 0 {
		t.Errorf("expected 0 reaped (active task preserved), got %d", count)
	}

	if _, err := tm.GetTask(task.ID); err != nil {
		t.Errorf("active task should not be reaped: %v", err)
	}
}

func TestTaskManager_CapacityLimit(t *testing.T) {
	tm := NewTaskManagerWithOptions(5*time.Minute, 3)

	// Create 3 tasks
	t1, _ := tm.CreateTask("test", nil)
	t2, _ := tm.CreateTask("test", nil)
	t3, _ := tm.CreateTask("test", nil)

	tm.UpdateTask(t1.ID, TaskStateRunning)
	tm.UpdateTask(t1.ID, TaskStateSucceeded)
	tm.UpdateTask(t2.ID, TaskStateRunning)
	tm.UpdateTask(t2.ID, TaskStateSucceeded)
	tm.UpdateTask(t3.ID, TaskStateRunning)
	tm.UpdateTask(t3.ID, TaskStateSucceeded)

	// Create a 4th - should succeed (reapOldest makes room)
	t4, err := tm.CreateTask("test", nil)
	if err != nil {
		t.Fatalf("CreateTask should succeed with reaping: %v", err)
	}

	if t4 == nil {
		t.Fatal("expected non-nil task")
	}
}

func TestTaskManager_ListTasks(t *testing.T) {
	tm := NewTaskManager()
	t1, _ := tm.CreateTask("type_a", nil)
	t2, _ := tm.CreateTask("type_b", nil)

	tasks := tm.ListTasks()
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}

	ids := map[string]bool{}
	for _, task := range tasks {
		ids[task.ID] = true
	}
	if !ids[t1.ID] || !ids[t2.ID] {
		t.Error("ListTasks missing some tasks")
	}
}

func TestTaskManager_ConcurrentCreateAndGet(t *testing.T) {
	tm := NewTaskManager()
	done := make(chan bool, 20)

	for i := 0; i < 10; i++ {
		go func() {
			task, err := tm.CreateTask("concurrent", nil)
			if err == nil && task != nil {
				tm.GetTask(task.ID)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	tasks := tm.ListTasks()
	if len(tasks) != 10 {
		t.Errorf("expected 10 tasks, got %d", len(tasks))
	}
}

func TestTaskManager_TaskOptionWithMetadata(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)
	tm.UpdateTask(task.ID, TaskStateSucceeded, WithMetadata("version", "1.0"))

	got, _ := tm.GetTask(task.ID)
	if got.Metadata["version"] != "1.0" {
		t.Errorf("expected metadata version=1.0, got %v", got.Metadata["version"])
	}
}

func TestTaskManager_TaskOptionWithProgress(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning, WithProgress(0.5))

	got, _ := tm.GetTask(task.ID)
	if got.Progress != 0.5 {
		t.Errorf("expected progress 0.5, got %f", got.Progress)
	}

	// Clamped to [0, 1]
	tm.UpdateTask(task.ID, TaskStateRunning, WithProgress(1.5))
	got, _ = tm.GetTask(task.ID)
	if got.Progress > 1.0 {
		t.Errorf("expected progress clamped to 1.0, got %f", got.Progress)
	}
}

func TestTaskManager_UpdateTaskWithError(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("test", nil)
	tm.UpdateTask(task.ID, TaskStateRunning)
	tm.UpdateTask(task.ID, TaskStateFailed, WithError("something went wrong"))

	got, _ := tm.GetTask(task.ID)
	if got.Error != "something went wrong" {
		t.Errorf("expected error 'something went wrong', got %s", got.Error)
	}
}

// TestTaskManager_GetReturnsCopy ensures GetTask returns a copy, not the internal pointer.
func TestTaskManager_GetReturnsCopy(t *testing.T) {
	tm := NewTaskManager()
	task, _ := tm.CreateTask("test", nil)

	got1, _ := tm.GetTask(task.ID)
	got2, _ := tm.GetTask(task.ID)

	// Different pointers means it's a copy
	if got1 == got2 {
		t.Error("expected GetTask to return different pointers (should be copies)")
	}
}

func TestNewTaskManagerWithOptions_DefaultValues(t *testing.T) {
	tm := NewTaskManagerWithOptions(0, 0)
	if tm.ttl != 5*time.Minute {
		t.Errorf("expected default TTL 5m, got %v", tm.ttl)
	}
	if tm.maxTasks != 100 {
		t.Errorf("expected default maxTasks 100, got %d", tm.maxTasks)
	}
}

func TestUUIDGeneration(t *testing.T) {
	tm := NewTaskManager()
	t1, _ := tm.CreateTask("test", nil)
	t2, _ := tm.CreateTask("test", nil)

	if t1.ID == t2.ID {
		t.Error("expected unique task IDs")
	}

	// Verify UUID format
	if _, err := uuid.Parse(t1.ID); err != nil {
		t.Errorf("expected valid UUID, got %s: %v", t1.ID, err)
	}
}