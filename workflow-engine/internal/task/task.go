package task

import (
	"context"
	"time"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusRunning  Status = "running"
	StatusSuccess  Status = "success"
	StatusFailed   Status = "failed"
	StatusCanceled Status = "canceled"
	StatusRetrying Status = "retrying"
)

type Task struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Dependencies []string               `json:"dependencies"`
	Execute      ExecuteFunc            `json:"-"`
	Retries      int                    `json:"retries"`
	Timeout      time.Duration          `json:"timeout"`
	Metadata     map[string]interface{} `json:"metadata"`

	// Runtime state
	Status    Status    `json:"status"`
	Result    *Result   `json:"result,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Attempts  int       `json:"attempts"`
}

// ExecuteFunc is the function signature for task execution
type ExecuteFunc func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

// Result represents task execution result
type Result struct {
	Output map[string]interface{} `json:"output"`
	Error  string                 `json:"error,omitempty"`
}

// New creates a new task
func New(id, name string, execute ExecuteFunc) *Task {
	return &Task{
		ID:           id,
		Name:         name,
		Execute:      execute,
		Dependencies: []string{},
		Status:       StatusPending,
		Retries:      3,
		Timeout:      5 * time.Minute,
		Metadata:     make(map[string]interface{}),
	}
}

// CanExecute checks if task is ready to run
func (t *Task) CanExecute() bool {
	return t.Status == StatusPending || t.Status == StatusRetrying
}

// IsComplete checks if task has finished
func (t *Task) IsComplete() bool {
	return t.Status == StatusSuccess ||
		t.Status == StatusFailed ||
		t.Status == StatusCanceled
}

// Duration returns task execution duration
func (t *Task) Duration() time.Duration {
	if t.StartTime.IsZero() {
		return 0
	}
	if t.EndTime.IsZero() {
		return time.Since(t.StartTime)
	}
	return t.EndTime.Sub(t.StartTime)
}
