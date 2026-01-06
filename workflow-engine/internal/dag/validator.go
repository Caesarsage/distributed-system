package dag

// wouldCreateCycle checks if adding edge from -> to would create a cycle
func (d *DAG) wouldCreateCycle(from, to string) bool {
	// DFS to check if path exists from 'to' to 'from'
	visited := make(map[string]bool)
	return d.hasCycleDFS(to, from, visited)
}

func (d *DAG) hasCycleDFS(current, target string, visited map[string]bool) bool {
	if current == target {
		return true
	}

	// if you have visited here (current) before
	if visited[current] {
		return false
	}

	visited[current] = true

	for _, neighbor := range d.Edges[current] {
		if d.hasCycleDFS(neighbor, target, visited) {
			return true
		}
	}

	return false
}

// Validate performs full DAG validation
func (d *DAG) Validate() error {
	// Check for cycles using topological sort
	_, err := d.TopologicalSort()
	return err
}
