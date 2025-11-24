package mapreduce

import (
	"sync"
	"time"
)

type KeyValue struct {
	Key   string
	Value string
}

type (
	MapFunc    func(docID string, contents string) []KeyValue
	ReduceFunc func(key string, values []string) []KeyValue
)

type TaskType int

const (
	MapTask TaskType = iota
	ReduceTask
	NoTask
)

type Master struct {
	numMap    int
	NumReduce int

	inputs []string

	mapStatus    []TaskState
	reduceStatus []TaskState

	intermediate map[int]map[string][]string
	midMu        sync.Mutex

	Outputs map[int]string
	outMu   sync.Mutex

	mu sync.Mutex
}

type Worker struct {
	ID       int
	Master   *Master
	mapFn    MapFunc
	reduceFn ReduceFunc
}

type Task struct {
	Type       TaskType
	ChunkIndex int
	TaskID     int
	Data       string
}

type TaskState struct {
	State      string
	AssignedAt time.Time
}

func NewMaster(inputs []string, numReduce int) *Master {
	m := &Master{
		numMap:       len(inputs),
		NumReduce:    numReduce,
		inputs:       inputs,
		mapStatus:    make([]TaskState, len(inputs)),
		reduceStatus: make([]TaskState, numReduce),
		intermediate: make(map[int]map[string][]string),
		Outputs:      map[int]string{},
	}

	// 1. Initialize states to idle
	for i := range m.mapStatus {
		m.mapStatus[i].State = "idle"
	}
	for i := range m.reduceStatus {
		m.reduceStatus[i].State = "idle"
	}

	for i := 0; i < numReduce; i++ {
		m.intermediate[i] = make(map[string][]string)
	}

	return m
}

func (m *Master) RequestTask() Task {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Assign idle map tasks
	for i := 0; i < m.numMap; i++ {
		if m.mapStatus[i].State == "idle" {
			m.mapStatus[i].State = "in-progress"
			m.mapStatus[i].AssignedAt = time.Now()
			//  Include the actual data
			return Task{
				Type:       MapTask,
				TaskID:     i,
				ChunkIndex: i,
				Data:       m.inputs[i],
			}
		}
	}

	// 2. Reassigned timed-out map tasks
	for i := 0; i < m.numMap; i++ {
		if m.mapStatus[i].State == "in-progress" {
			if time.Since(m.mapStatus[i].AssignedAt) > 5*time.Second {
				m.mapStatus[i].AssignedAt = time.Now()
				return Task{
					Type:       MapTask,
					TaskID:     i,
					ChunkIndex: i,
					Data:       m.inputs[i],
				}
			}
		}
	}

	// 3. Check if all map tasks are donw
	allMapDone := true
	for i := 0; i < m.numMap; i++ {
		if m.mapStatus[i].State != "done" {
			allMapDone = false
			break
		}
	}
	if !allMapDone {
		return Task{Type: NoTask}
	}

	// 4. assign idle redice task
	for i := 0; i < m.NumReduce; i++ {
		if m.reduceStatus[i].State == "idle" {
			m.reduceStatus[i].State = "in-progress"
			m.reduceStatus[i].AssignedAt = time.Now()
			// Include the actual data
			return Task{
				Type:       ReduceTask,
				TaskID:     i,
				ChunkIndex: i,
				Data:       m.inputs[i],
			}
		}
	}

	// 5. Reassign idle reduce task
	for i := 0; i < m.NumReduce; i++ {
		if m.reduceStatus[i].State == "in-progress" {
			if time.Since(m.reduceStatus[i].AssignedAt) > 10*time.Second {
				m.reduceStatus[i].AssignedAt = time.Now()
				return Task{
					Type:       MapTask,
					TaskID:     i,
					ChunkIndex: i,
					Data:       m.inputs[i],
				}
			}
		}
	}

	return Task{Type: NoTask}
}

func (m *Master) ReportMapDone(mapID int, partitions map[int][]KeyValue) {
	m.midMu.Lock()
	defer m.midMu.Unlock()

	for r, kvs := range partitions {
		for _, kv := range kvs {
			m.intermediate[r][kv.Key] = append(m.intermediate[r][kv.Key], kv.Value)
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.mapStatus[mapID].State = "done"
}

func (m *Master) ReportReduceDone(reduceID int, out string) {
	m.outMu.Lock()
	defer m.outMu.Unlock()

	m.Outputs[reduceID] = out

	m.mu.Lock()
	m.reduceStatus[reduceID].State = "done"
	m.mu.Unlock()
}

func (m *Master) GetReducePartition(reduceIdx int) map[string][]string {
 	m.midMu.Lock()
	defer m.midMu.Unlock()

	copyMap := make(map[string][]string)
	for k, v := range m.intermediate[reduceIdx] {
		copyv := make([]string, len(v))
		copy(copyv, v)
		copyMap[k] = copyv
	}

	return copyMap
}

func (m *Master) Done() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := 0; i < m.numMap; i++ {
		if m.mapStatus[i].State != "done" {
			return false
		}
	}

	for i := 0; i < m.NumReduce; i++ {
		if m.reduceStatus[i].State != "done" {
			return false
		}
	}

	return true
}
