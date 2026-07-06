//go:build e2e

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// taskBehavior describes how a mock system task behaves when polled.
type taskBehavior int

const (
	behaviorSucceed  taskBehavior = iota // returns running N times, then succeeded
	behaviorFail                         // returns running N times, then failed
	behaviorHang                         // always returns running (never terminates)
)

// e2eMockUpstream simulates the New API endpoints for E2E testing.
// It supports multiple task IDs with configurable behaviors, making it
// suitable for testing cancellation, completion, and failure scenarios.
type e2eMockUpstream struct {
	mu          sync.Mutex
	defaultTask string                       // task ID returned by /api/channel/test
	behaviors   map[string]taskBehavior      // per-task behavior config
	counts      map[string]int               // per-task poll counts
	steps       map[string]int               // per-task delay steps before terminal
}

// newE2EMockUpstream creates a mock upstream. The defaultTask is the task_id
// returned by /api/channel/test. delaySteps is how many "running" responses
// before reaching the terminal state (for succeed/fail behaviors).
func newE2EMockUpstream(defaultTask string, delaySteps int) *e2eMockUpstream {
	m := &e2eMockUpstream{
		defaultTask: defaultTask,
		behaviors:   make(map[string]taskBehavior),
		counts:      make(map[string]int),
		steps:       make(map[string]int),
	}
	m.AddTask(defaultTask, behaviorSucceed, delaySteps)
	return m
}

// AddTask registers a named task with a specific behavior and poll delay.
func (m *e2eMockUpstream) AddTask(id string, behavior taskBehavior, delaySteps int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.behaviors[id] = behavior
	m.steps[id] = delaySteps
	m.counts[id] = 0
}

func (m *e2eMockUpstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/healthz":
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	case "/api/channel/test":
		// Return the default task configuration
		taskID := m.defaultTask
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"task_id": taskID,
				"status":  "running",
			},
		})

	case "/api/system-task/" + m.defaultTask:
		m.mu.Lock()
		m.counts[m.defaultTask]++
		count := m.counts[m.defaultTask]
		behavior := m.behaviors[m.defaultTask]
		steps := m.steps[m.defaultTask]
		m.mu.Unlock()

		m.writePollResponse(w, behavior, steps, count)

	default:
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("unknown path: %s", r.URL.Path)})
	}
}

// writePollResponse writes a system-task poll response based on the task's
// behavior, steps, and current poll count.
func (m *e2eMockUpstream) writePollResponse(w http.ResponseWriter, behavior taskBehavior, steps, count int) {
	w.WriteHeader(http.StatusOK)

	switch behavior {
	case behaviorSucceed:
		if count <= steps {
			// Still running
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data":    map[string]any{"status": "running"},
			})
		} else {
			// Succeeded
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"status": "succeeded",
					"result": map[string]any{
						"tested":    3,
						"succeeded": 3,
						"failed":    0,
						"disabled":  0,
						"enabled":   3,
					},
				},
			})
		}

	case behaviorFail:
		if count <= steps {
			// Still running
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data":    map[string]any{"status": "running"},
			})
		} else {
			// Failed
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"status": "failed",
					"error":  "channel timeout: upstream unreachable",
				},
			})
		}

	case behaviorHang:
		// Always running — never terminates
		time.Sleep(50 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"status": "running"},
		})
	}
}