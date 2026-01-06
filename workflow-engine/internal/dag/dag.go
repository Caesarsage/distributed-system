package dag

import (
	"fmt"

	"github.com/Caesarsage/workflow-engine/internal/task"
)

// DAG -> a Directed Acyclic Graph of tasks
type DAG struct {
	Tasks map[string]*task.Task
	Edges map[string][]string // task ID -> dependent task IDs
}

// Creates a new DAG
func New() *DAG {
	return &DAG{
		Tasks: make(map[string]*task.Task),
		Edges: make(map[string][]string),
	}
}

// AddTask adds a task to the DAG
func (d *DAG) AddTask(t *task.Task) error {
	if _, exists := d.Tasks[t.ID]; exists {
		return fmt.Errorf("task %s already exists", t.ID)
	}

	d.Tasks[t.ID] = t
	d.Edges[t.ID] = []string{}

	return nil
}

// AddDependency adds an edge from -> to
func (d *DAG) AddDependency(from, to string) error {
	// check if task for the edges from -> exists
	if _, exists := d.Tasks[from]; !exists {
		return fmt.Errorf("task %s not found", from)
	}
	if _, exists := d.Tasks[to]; !exists {
		return fmt.Errorf("task %s not found", to)
	}

	// Check if would create cycle
	if d.wouldCreateCycle(from, to) {
		return fmt.Errorf("adding edge %s -> %s would create cycle", from, to)
	}

	d.Edges[from] = append(d.Edges[from], to)
	d.Tasks[to].Dependencies = append(d.Tasks[to].Dependencies, from)

	return nil
}

// GetReadyTasks returns tasks with all dependencies satisfied
func (d *DAG) GetReadyTasks() []*task.Task {
	ready := []*task.Task{}

	for _, t := range d.Tasks {
		if !t.CanExecute() {
			continue
		}

		// Check if all dependencies are complete
		allDepsComplete := true
		for _, depID := range t.Dependencies {
			dep := d.Tasks[depID]
			if dep.Status != task.StatusSuccess {
				allDepsComplete = false
				break
			}
		}

		if allDepsComplete {
			ready = append(ready, t)
		}
	}

	return ready
}

// IsComplete returns true if all tasks are complete
func (d *DAG) IsComplete() bool {
	for _, t := range d.Tasks {
		if !t.IsComplete() {
			return false
		}
	}
	return true
}

// HasFailures returns true if any task failed
func (d *DAG) HasFailures() bool {
	for _, t := range d.Tasks {
		if t.Status == task.StatusFailed {
			return true
		}
	}
	return false
}
