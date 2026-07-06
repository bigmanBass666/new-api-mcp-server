package hightools

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TaskState represents the lifecycle state of an async task.
type TaskState string

const (
	TaskStatePending       TaskState = "pending"
	TaskStateRunning       TaskState = "running"
	TaskStateSucceeded     TaskState = "succeeded"
	TaskStateFailed        TaskState = "failed"
	TaskStateInputRequired TaskState = "input_required"
	TaskStateCancelled     TaskState = "cancelled"
)

// validTransitions defines allowed state transitions.
// Keys are from-states; values are sets of reachable to-states.
var validTransitions = map[TaskState]map[TaskState]bool{
	TaskStatePending: {
		TaskStateRunning: true,
	},
	TaskStateRunning: {
		TaskStateSucceeded:     true,
		TaskStateFailed:        true,
		TaskStateInputRequired: true,
		TaskStateCancelled:     true,
	},
	TaskStateInputRequired: {
		TaskStateRunning:  true,
		TaskStateCancelled: true,
	},
	// Terminal states: no outgoing transitions.
	TaskStateSucceeded: {},
	TaskStateFailed:    {},
	TaskStateCancelled: {},
}

// terminalStates contains states that cannot transition further.
var terminalStates = map[TaskState]bool{
	TaskStateSucceeded: true,
	TaskStateFailed:    true,
	TaskStateCancelled: true,
}

// activeStates contains states for in-progress tasks.
var activeStates = map[TaskState]bool{
	TaskStatePending:       true,
	TaskStateRunning:       true,
	TaskStateInputRequired: true,
}

// Task represents an async task with lifecycle tracking.
type Task struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	State     TaskState      `json:"state"`
	Progress  float64        `json:"progress,omitempty"`
	Result    any            `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// ResumeSignal carries user-provided input to unblock a task waiting
// in input_required state.
type ResumeSignal struct {
	Action  string         `json:"action"`
	Payload map[string]any `json:"payload,omitempty"`
}

// TaskOption is a functional option applied during UpdateTask.
type TaskOption func(*Task)

// WithResult sets the task result value.
func WithResult(result any) TaskOption {
	return func(t *Task) {
		t.Result = result
	}
}

// WithError sets the task error message.
func WithError(err string) TaskOption {
	return func(t *Task) {
		t.Error = err
	}
}

// WithProgress sets the task progress, clamped to [0, 1].
func WithProgress(progress float64) TaskOption {
	return func(t *Task) {
		if progress < 0 {
			progress = 0
		}
		if progress > 1 {
			progress = 1
		}
		t.Progress = progress
	}
}

// WithMetadata sets a metadata key-value pair on the task.
func WithMetadata(key string, value any) TaskOption {
	return func(t *Task) {
		if t.Metadata == nil {
			t.Metadata = make(map[string]any)
		}
		t.Metadata[key] = value
	}
}

// TaskManager manages async task lifecycle with state machine enforcement,
// TTL-based reaping, and resume/cancel channel support.
type TaskManager struct {
	mu        sync.RWMutex
	tasks     map[string]*Task
	ttl       time.Duration
	maxTasks  int
	cancelChs map[string]chan struct{}
	resumeChs map[string]chan ResumeSignal
}

// NewTaskManager creates a TaskManager with default settings:
// TTL of 5 minutes and max 100 tasks.
func NewTaskManager() *TaskManager {
	return NewTaskManagerWithOptions(5*time.Minute, 100)
}

// NewTaskManagerWithOptions creates a TaskManager with custom TTL and capacity.
func NewTaskManagerWithOptions(ttl time.Duration, maxTasks int) *TaskManager {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if maxTasks <= 0 {
		maxTasks = 100
	}
	return &TaskManager{
		tasks:     make(map[string]*Task),
		ttl:       ttl,
		maxTasks:  maxTasks,
		cancelChs: make(map[string]chan struct{}),
		resumeChs: make(map[string]chan ResumeSignal),
	}
}

// CreateTask creates a new task in pending state with the given type
// and optional metadata. Returns the created task or an error if the
// manager is at capacity.
func (tm *TaskManager) CreateTask(taskType string, metadata map[string]any) (*Task, error) {
	task := &Task{
		ID:        uuid.New().String(),
		Type:      taskType,
		State:     TaskStatePending,
		Metadata:  metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Try to free space before checking capacity.
	tm.reapLocked()
	if len(tm.tasks) >= tm.maxTasks {
		tm.reapOldestLocked()
	}
	if len(tm.tasks) >= tm.maxTasks {
		return nil, fmt.Errorf("task manager at capacity (%d)", tm.maxTasks)
	}

	tm.tasks[task.ID] = task
	tm.cancelChs[task.ID] = make(chan struct{})
	tm.resumeChs[task.ID] = make(chan ResumeSignal, 1)

	slog.Debug("task created", "task_id", task.ID, "type", taskType)

	return task, nil
}

// GetTask retrieves a task by ID. Returns a copy to prevent concurrent access
// races between readers and background worker goroutines. Returns an error if
// the task is not found.
func (tm *TaskManager) GetTask(id string) (*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, ok := tm.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	// Return a shallow copy so the caller can safely read value fields
	// without racing with background goroutines that update the task via
	// UpdateTask.
	taskCopy := *task
	return &taskCopy, nil
}

// UpdateTask transitions a task to a new state with optional mutations.
// Returns an error if the transition is invalid or the task does not exist.
func (tm *TaskManager) UpdateTask(id string, newState TaskState, opts ...TaskOption) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	if err := validateTransition(task.State, newState); err != nil {
		return err
	}

	task.State = newState
	task.UpdatedAt = time.Now()
	for _, opt := range opts {
		opt(task)
	}

	if terminalStates[newState] {
		tm.cleanupChannelsLocked(id)
	}

	slog.Debug("task updated", "task_id", id, "state", newState)

	return nil
}

// validateTransition checks whether moving from one state to another is allowed.
func validateTransition(from, to TaskState) error {
	allowed, ok := validTransitions[from]
	if !ok {
		return fmt.Errorf("unknown source state: %s", from)
	}
	if !allowed[to] {
		return fmt.Errorf("invalid state transition: %s -> %s", from, to)
	}
	return nil
}

// CancelTask cancels a task and signals the worker goroutine.
// Returns an error if the task is already in a terminal state.
func (tm *TaskManager) CancelTask(id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}
	if terminalStates[task.State] {
		return fmt.Errorf("task %s is already in terminal state: %s", id, task.State)
	}

	// Signal the worker goroutine.
	if ch, ok := tm.cancelChs[id]; ok {
		close(ch)
		delete(tm.cancelChs, id)
	}

	task.State = TaskStateCancelled
	task.UpdatedAt = time.Now()
	tm.cleanupChannelsLocked(id)

	slog.Debug("task cancelled", "task_id", id)

	return nil
}

// ListTasks returns a snapshot of all tasks currently tracked.
// Each task is a copy safe for concurrent access.
func (tm *TaskManager) ListTasks() []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make([]*Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		taskCopy := *task
		result = append(result, &taskCopy)
	}
	return result
}

// Reap removes expired terminal tasks. Returns the count removed.
// Active tasks (pending, running, input_required) are never removed.
func (tm *TaskManager) Reap() int {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.reapLocked()
}

// reapLocked removes terminal tasks whose TTL has expired.
// Caller must hold tm.mu.
func (tm *TaskManager) reapLocked() int {
	cutoff := time.Now().Add(-tm.ttl)
	removed := 0

	for id, task := range tm.tasks {
		if terminalStates[task.State] && task.UpdatedAt.Before(cutoff) {
			delete(tm.tasks, id)
			tm.cleanupChannelsLocked(id)
			removed++
		}
	}
	return removed
}

// reapOldestLocked removes the oldest terminal tasks until the map is
// below maxTasks. Caller must hold tm.mu.
func (tm *TaskManager) reapOldestLocked() {
	if len(tm.tasks) < tm.maxTasks {
		return
	}

	// Collect terminal tasks with their update timestamps.
	type agedEntry struct {
		id string
		t  time.Time
	}
	var candidates []agedEntry

	for id, task := range tm.tasks {
		if terminalStates[task.State] {
			candidates = append(candidates, agedEntry{id: id, t: task.UpdatedAt})
		}
	}

	// Sort by timestamp ascending (oldest first).
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].t.Before(candidates[i].t) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Remove at least one entry to get below capacity.
	toRemove := len(tm.tasks) - tm.maxTasks + 1
	if toRemove > len(candidates) {
		toRemove = len(candidates)
	}

	for i := 0; i < toRemove; i++ {
		delete(tm.tasks, candidates[i].id)
		tm.cleanupChannelsLocked(candidates[i].id)
	}
}

// WaitForResume blocks until a resume signal is received for the task
// or the underlying channel is closed (e.g. on task cancellation).
func (tm *TaskManager) WaitForResume(id string) (*ResumeSignal, error) {
	tm.mu.RLock()
	ch, ok := tm.resumeChs[id]
	tm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("task not found or no resume channel: %s", id)
	}

	signal, ok := <-ch
	if !ok {
		return nil, fmt.Errorf("task %s resume channel closed", id)
	}
	return &signal, nil
}

// SignalResume sends a resume signal to a task in input_required state.
// Returns an error if the task is not in input_required state.
func (tm *TaskManager) SignalResume(id string, signal ResumeSignal) error {
	tm.mu.Lock()
	task, ok := tm.tasks[id]
	if !ok {
		tm.mu.Unlock()
		return fmt.Errorf("task not found: %s", id)
	}
	if task.State != TaskStateInputRequired {
		tm.mu.Unlock()
		return fmt.Errorf("task %s is not in input_required state (current: %s)", id, task.State)
	}
	ch, hasCh := tm.resumeChs[id]
	tm.mu.Unlock()

	if !hasCh {
		return fmt.Errorf("task %s has no resume channel", id)
	}

	// Non-blocking send; buffer of 1 so this should succeed.
	select {
	case ch <- signal:
	default:
		slog.Warn("resume channel buffer full, dropping signal", "task_id", id)
	}
	return nil
}

// GetCancelCh returns a read-only channel that is closed when the task
// is cancelled. For nonexistent or terminal tasks, returns an already-closed
// channel so callers never block.
func (tm *TaskManager) GetCancelCh(id string) <-chan struct{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	ch, ok := tm.cancelChs[id]
	if !ok {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return ch
}

// cleanupChannelsLocked removes both cancel and resume channels for a task.
// Caller must hold tm.mu.
func (tm *TaskManager) cleanupChannelsLocked(id string) {
	if ch, ok := tm.cancelChs[id]; ok {
		select {
		case <-ch:
		default:
			close(ch)
		}
		delete(tm.cancelChs, id)
	}
	if ch, ok := tm.resumeChs[id]; ok {
		select {
		case <-ch:
		default:
			close(ch)
		}
		delete(tm.resumeChs, id)
	}
}