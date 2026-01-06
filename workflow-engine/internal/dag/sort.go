package dag

import (
	"fmt"

	"github.com/Caesarsage/workflow-engine/internal/task"
)


func (d *DAG) TopologicalSort() ([]*task.Task, error) {
	// Calculate in-degree for each node
	inDegree := make(map[string]int)
	for id := range d.Tasks {
		inDegree[id] = 0
	}

	for _, deps := range d.Edges {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	// Queue of nodes with no dependencies
	queue := []string{}
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	// Process queue
	sorted := []*task.Task{}

	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]

		sorted = append(sorted, d.Tasks[current])

		// Reduce in-degree of neighbors
		for _, neighbor := range d.Edges[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// If not all tasks processed, there's a cycle
	if len(sorted) != len(d.Tasks) {
		return nil, fmt.Errorf("cycle detected in DAG")
	}

	return sorted, nil
}
