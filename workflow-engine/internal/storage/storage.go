package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Caesarsage/workflow-engine/internal/dag"
	"github.com/Caesarsage/workflow-engine/internal/task"
)

// Storage provides persistence for workflows and tasks
type Storage interface {
	// Workflow operations
	SaveWorkflow(workflowID string, workflow *WorkflowState) error
	LoadWorkflow(workflowID string) (*WorkflowState, error)
	DeleteWorkflow(workflowID string) error
	ListWorkflows() ([]string, error)

	// Task operations
	SaveTask(workflowID string, task *task.Task) error
	LoadTask(workflowID string, taskID string) (*task.Task, error)
	LoadAllTasks(workflowID string) (map[string]*task.Task, error)
}

// WorkflowState represents the persisted state of a workflow
type WorkflowState struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	DAG       *dag.DAG               `json:"dag"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// FileStorage implements Storage using file-based JSON persistence
type FileStorage struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFileStorage creates a new file-based storage instance
func NewFileStorage(baseDir string) (*FileStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create subdirectories
	workflowsDir := filepath.Join(baseDir, "workflows")
	tasksDir := filepath.Join(baseDir, "tasks")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workflows directory: %w", err)
	}
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create tasks directory: %w", err)
	}

	return &FileStorage{
		baseDir: baseDir,
	}, nil
}

// SaveWorkflow persists a workflow to disk
func (fs *FileStorage) SaveWorkflow(workflowID string, workflow *WorkflowState) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	workflowsDir := filepath.Join(fs.baseDir, "workflows")
	filePath := filepath.Join(workflowsDir, workflowID+".json")

	data, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	return nil
}

// LoadWorkflow loads a workflow from disk
func (fs *FileStorage) LoadWorkflow(workflowID string) (*WorkflowState, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	workflowsDir := filepath.Join(fs.baseDir, "workflows")
	filePath := filepath.Join(workflowsDir, workflowID+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("workflow %s not found", workflowID)
		}
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	var workflow WorkflowState
	if err := json.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow: %w", err)
	}

	return &workflow, nil
}

// DeleteWorkflow removes a workflow from disk
func (fs *FileStorage) DeleteWorkflow(workflowID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	workflowsDir := filepath.Join(fs.baseDir, "workflows")
	filePath := filepath.Join(workflowsDir, workflowID+".json")

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete workflow file: %w", err)
	}

	// Also delete associated tasks directory
	tasksDir := filepath.Join(fs.baseDir, "tasks", workflowID)
	if err := os.RemoveAll(tasksDir); err != nil {
		// Log but don't fail if tasks dir doesn't exist
		_ = err
	}

	return nil
}

// ListWorkflows returns all workflow IDs
func (fs *FileStorage) ListWorkflows() ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	workflowsDir := filepath.Join(fs.baseDir, "workflows")
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	var workflowIDs []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			workflowID := entry.Name()[:len(entry.Name())-5] // Remove .json extension
			workflowIDs = append(workflowIDs, workflowID)
		}
	}

	return workflowIDs, nil
}

// SaveTask persists a task to disk
func (fs *FileStorage) SaveTask(workflowID string, t *task.Task) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	tasksDir := filepath.Join(fs.baseDir, "tasks", workflowID)
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	filePath := filepath.Join(tasksDir, t.ID+".json")

	// Create a serializable version of the task (without Execute function)
	taskData := map[string]interface{}{
		"id":           t.ID,
		"name":         t.Name,
		"dependencies": t.Dependencies,
		"retries":      t.Retries,
		"timeout":      t.Timeout.String(),
		"metadata":     t.Metadata,
		"status":       string(t.Status),
		"attempts":     t.Attempts,
	}

	if t.Result != nil {
		taskData["result"] = map[string]interface{}{
			"output": t.Result.Output,
			"error":  t.Result.Error,
		}
	}

	if !t.StartTime.IsZero() {
		taskData["start_time"] = t.StartTime.Format("2006-01-02T15:04:05Z07:00")
	}

	if !t.EndTime.IsZero() {
		taskData["end_time"] = t.EndTime.Format("2006-01-02T15:04:05Z07:00")
	}

	data, err := json.MarshalIndent(taskData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	return nil
}

// LoadTask loads a task from disk
func (fs *FileStorage) LoadTask(workflowID string, taskID string) (*task.Task, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	tasksDir := filepath.Join(fs.baseDir, "tasks", workflowID)
	filePath := filepath.Join(tasksDir, taskID+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("task %s not found", taskID)
		}
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	var taskData map[string]interface{}
	if err := json.Unmarshal(data, &taskData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	// Reconstruct task (without Execute function - that needs to be provided by workflow definition)
	t := &task.Task{
		ID:           taskData["id"].(string),
		Name:         taskData["name"].(string),
		Dependencies: []string{},
		Retries:      int(taskData["retries"].(float64)),
		Metadata:     make(map[string]interface{}),
		Status:       task.Status(taskData["status"].(string)),
		Attempts:     int(taskData["attempts"].(float64)),
	}

	// Parse dependencies
	if deps, ok := taskData["dependencies"].([]interface{}); ok {
		for _, dep := range deps {
			t.Dependencies = append(t.Dependencies, dep.(string))
		}
	}

	// Parse timeout
	if timeoutStr, ok := taskData["timeout"].(string); ok {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			t.Timeout = timeout
		}
	}

	// Parse metadata
	if metadata, ok := taskData["metadata"].(map[string]interface{}); ok {
		t.Metadata = metadata
	}

	// Parse result
	if resultData, ok := taskData["result"].(map[string]interface{}); ok {
		t.Result = &task.Result{
			Output: make(map[string]interface{}),
		}
		if output, ok := resultData["output"].(map[string]interface{}); ok {
			t.Result.Output = output
		}
		if errStr, ok := resultData["error"].(string); ok {
			t.Result.Error = errStr
		}
	}

	// Parse timestamps
	if startTimeStr, ok := taskData["start_time"].(string); ok {
		if startTime, err := time.Parse("2006-01-02T15:04:05Z07:00", startTimeStr); err == nil {
			t.StartTime = startTime
		}
	}

	if endTimeStr, ok := taskData["end_time"].(string); ok {
		if endTime, err := time.Parse("2006-01-02T15:04:05Z07:00", endTimeStr); err == nil {
			t.EndTime = endTime
		}
	}

	return t, nil
}

// LoadAllTasks loads all tasks for a workflow
func (fs *FileStorage) LoadAllTasks(workflowID string) (map[string]*task.Task, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	tasksDir := filepath.Join(fs.baseDir, "tasks", workflowID)
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*task.Task), nil
		}
		return nil, fmt.Errorf("failed to read tasks directory: %w", err)
	}

	tasks := make(map[string]*task.Task)
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			taskID := entry.Name()[:len(entry.Name())-5] // Remove .json extension
			t, err := fs.LoadTask(workflowID, taskID)
			if err != nil {
				return nil, err
			}
			tasks[taskID] = t
		}
	}

	return tasks, nil
}
