package main

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============= TYPE DEFINITIONS =============

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

// ============= MASTER IMPLEMENTATION =============

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
			return Task{
				Type:       MapTask,
				TaskID:     i,
				ChunkIndex: i,
				Data:       m.inputs[i],
			}
		}
	}

	// 2. Reassign timed-out map tasks
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

	// 3. Check if all map tasks are done
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

	// 4. Assign idle reduce tasks
	for i := 0; i < m.NumReduce; i++ {
		if m.reduceStatus[i].State == "idle" {
			m.reduceStatus[i].State = "in-progress"
			m.reduceStatus[i].AssignedAt = time.Now()
			return Task{
				Type:       ReduceTask,
				TaskID:     i,
				ChunkIndex: i,
				Data:       "",
			}
		}
	}

	// 5. Reassign timed-out reduce tasks
	for i := 0; i < m.NumReduce; i++ {
		if m.reduceStatus[i].State == "in-progress" {
			if time.Since(m.reduceStatus[i].AssignedAt) > 10*time.Second {
				m.reduceStatus[i].AssignedAt = time.Now()
				return Task{
					Type:       ReduceTask,
					TaskID:     i,
					ChunkIndex: i,
					Data:       "",
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

// ============= WORKER IMPLEMENTATION =============

func NewWorker(id int, master *Master, mapFn MapFunc, reduceFn ReduceFunc) *Worker {
	return &Worker{
		ID:       id,
		Master:   master,
		mapFn:    mapFn,
		reduceFn: reduceFn,
	}
}

func (w *Worker) Run() {
	for {
		if w.Master.Done() {
			return
		}

		task := w.Master.RequestTask()

		switch task.Type {
		case MapTask:
			w.doMap(task)
		case ReduceTask:
			w.doReduce(task)
		case NoTask:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (w *Worker) doMap(task Task) {
	kvs := w.mapFn(task.Data, task.Data)

	partitions := make(map[int][]KeyValue)
	for _, kv := range kvs {
		reduceIdx := ihash(kv.Key) % w.Master.NumReduce
		partitions[reduceIdx] = append(partitions[reduceIdx], kv)
	}

	w.Master.ReportMapDone(task.TaskID, partitions)
}

func (w *Worker) doReduce(task Task) {
	partition := w.Master.GetReducePartition(task.TaskID)

	var results []KeyValue
	for key, values := range partition {
		reduced := w.reduceFn(key, values)
		results = append(results, reduced...)
	}

	output := ""
	for _, kv := range results {
		output += kv.Key + ":" + kv.Value + "\n"
	}

	w.Master.ReportReduceDone(task.TaskID, output)
}

func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// ============= HELPER FUNCTION =============

func RunMapReduce(inputs []string, numReduce int, mapFn MapFunc, reduceFn ReduceFunc, numWorkers int) map[int]string {
	master := NewMaster(inputs, numReduce)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker := NewWorker(workerID, master, mapFn, reduceFn)
			worker.Run()
		}(i)
	}

	wg.Wait()

	return master.Outputs
}

// ============= MAIN FUNCTION WITH EXAMPLES =============

func main() {
	fmt.Println("========================================")
	fmt.Println("MapReduce Implementation Demo")
	fmt.Println("========================================\n")

	// Example 1: Word Count
	fmt.Println("Example 1: Word Count")
	fmt.Println("----------------------")

	mapFn := func(docID, contents string) []KeyValue {
		words := strings.Fields(contents)
		var kvs []KeyValue
		for _, word := range words {
			kvs = append(kvs, KeyValue{Key: word, Value: "1"})
		}
		return kvs
	}

	reduceFn := func(key string, values []string) []KeyValue {
		count := len(values)
		return []KeyValue{{Key: key, Value: strconv.Itoa(count)}}
	}

	inputs := []string{
		"hello world",
		"hello mapreduce",
		"world of go hello",
	}

	fmt.Println("Input documents:")
	for i, doc := range inputs {
		fmt.Printf("  Doc %d: %s\n", i+1, doc)
	}

	fmt.Println("\nRunning MapReduce with 3 workers and 2 reducers...")
	results := RunMapReduce(inputs, 2, mapFn, reduceFn, 3)

	fmt.Println("\nWord Counts:")
	for partitionID, output := range results {
		fmt.Printf("Partition %d:\n", partitionID)
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if line != "" {
				fmt.Printf("  %s\n", line)
			}
		}
	}

	// Example 2: Character Frequency
	fmt.Println("\n========================================")
	fmt.Println("Example 2: Character Frequency")
	fmt.Println("----------------------")

	charMapFn := func(docID, contents string) []KeyValue {
		var kvs []KeyValue
		for _, char := range contents {
			if char != ' ' {
				kvs = append(kvs, KeyValue{Key: string(char), Value: "1"})
			}
		}
		return kvs
	}

	charReduceFn := func(key string, values []string) []KeyValue {
		return []KeyValue{{Key: key, Value: strconv.Itoa(len(values))}}
	}

	charInputs := []string{"aaa", "abb", "abc"}

	fmt.Println("Input documents:")
	for i, doc := range charInputs {
		fmt.Printf("  Doc %d: %s\n", i+1, doc)
	}

	fmt.Println("\nRunning MapReduce...")
	charResults := RunMapReduce(charInputs, 2, charMapFn, charReduceFn, 2)

	fmt.Println("\nCharacter Frequencies:")
	for partitionID, output := range charResults {
		fmt.Printf("Partition %d:\n", partitionID)
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if line != "" {
				fmt.Printf("  %s\n", line)
			}
		}
	}

	// Example 3: Inverted Index
	fmt.Println("\n========================================")
	fmt.Println("Example 3: Inverted Index")
	fmt.Println("----------------------")

	indexMapFn := func(docID, contents string) []KeyValue {
		words := strings.Fields(contents)
		var kvs []KeyValue
		for _, word := range words {
			// Extract doc ID from content (format: "docX: text")
			parts := strings.SplitN(contents, ":", 2)
			if len(parts) > 0 {
				kvs = append(kvs, KeyValue{Key: word, Value: parts[0]})
			}
		}
		return kvs
	}

	indexReduceFn := func(key string, values []string) []KeyValue {
		seen := make(map[string]bool)
		var docs []string
		for _, v := range values {
			if !seen[v] {
				seen[v] = true
				docs = append(docs, v)
			}
		}
		return []KeyValue{{Key: key, Value: strings.Join(docs, ",")}}
	}

	indexInputs := []string{
		"doc1: the quick brown fox",
		"doc2: the lazy dog",
		"doc3: quick brown dog",
	}

	fmt.Println("Input documents:")
	for _, doc := range indexInputs {
		fmt.Printf("  %s\n", doc)
	}

	fmt.Println("\nRunning MapReduce...")
	indexResults := RunMapReduce(indexInputs, 3, indexMapFn, indexReduceFn, 2)

	fmt.Println("\nInverted Index (word -> documents):")
	for partitionID, output := range indexResults {
		fmt.Printf("Partition %d:\n", partitionID)
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if line != "" {
				fmt.Printf("  %s\n", line)
			}
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("All examples completed successfully!")
	fmt.Println("========================================")
}
