package task

import (
	"context"
	"fmt"
	"time"
)

// Executor executes tasks with timeout and retry logic
type Executor struct {
	maxRetries      int
	retryBackoff    time.Duration
	maxRetryBackoff time.Duration
}

// NewExecutor creates a new task executor
func NewExecutor() *Executor {
	return &Executor{
		maxRetries:      3,
		retryBackoff:    time.Second,
		maxRetryBackoff: 30 * time.Second,
	}
}

// Execute executes a task with timeout and retry logic
func (e *Executor) Execute(ctx context.Context, t *Task, input map[string]interface{}) (*Result, error) {
	if t.Execute == nil {
		return &Result{
			Error: "task has no execute function",
		}, fmt.Errorf("task %s has no execute function", t.ID)
	}

	// Use task's timeout if set, otherwise use context timeout
	timeout := t.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Track attempts
	attempts := 0
	maxAttempts := t.Retries + 1
	if maxAttempts == 0 {
		maxAttempts = e.maxRetries + 1
	}

	var lastErr error
	var result *Result

	for attempts < maxAttempts {
		attempts++
		t.Attempts = attempts

		// Update status
		if attempts > 1 {
			t.Status = StatusRetrying
		} else {
			t.Status = StatusRunning
		}
		t.StartTime = time.Now()

		// Execute task
		output, err := t.Execute(ctx, input)

		if err == nil {
			// Success
			t.Status = StatusSuccess
			t.EndTime = time.Now()
			result = &Result{
				Output: output,
			}
			return result, nil
		}

		// Failure
		lastErr = err
		t.EndTime = time.Now()

		// Check if we should retry
		if attempts < maxAttempts {
			// Calculate backoff
			backoff := e.calculateBackoff(attempts)
			
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				t.Status = StatusFailed
				result = &Result{
					Error: fmt.Sprintf("task timeout: %v", err),
				}
				return result, ctx.Err()
			case <-time.After(backoff):
				// Continue to retry
			}
		}
	}

	// All retries exhausted
	t.Status = StatusFailed
	result = &Result{
		Error: fmt.Sprintf("task failed after %d attempts: %v", attempts, lastErr),
	}
	return result, lastErr
}

// calculateBackoff calculates exponential backoff duration
func (e *Executor) calculateBackoff(attempt int) time.Duration {
	backoff := e.retryBackoff * time.Duration(1<<uint(attempt-1))
	if backoff > e.maxRetryBackoff {
		backoff = e.maxRetryBackoff
	}
	return backoff
}

// ExecuteWithResult executes a task and aggregates results from dependencies
func (e *Executor) ExecuteWithResult(ctx context.Context, t *Task, dependencyResults map[string]*Result) (*Result, error) {
	// Build input from dependency results
	input := make(map[string]interface{})
	
	// Aggregate outputs from all dependencies
	for depID, depResult := range dependencyResults {
		if depResult != nil && depResult.Output != nil {
			// Use dependency ID as key, or merge all outputs
			input[depID] = depResult.Output
			
			// Also merge individual output fields
			for k, v := range depResult.Output {
				input[k] = v
			}
		}
	}

	return e.Execute(ctx, t, input)
}

