
This is a Single Master Implementation (No Leader Election Needed)
---

# Map reduce

Map reduce is used for computing large data in parrallel where it is broken into two phase, map and reduce phase. The data is first broken into chunks in the map phase and combine together in the reduce phase to give a final result.


## counting word example

1. Master creates map tasks (one per input file)
2. Workers request tasks → get map tasks
3. Workers process data with map function
4. Workers report map results back
5. Once ALL maps are done, reduce phase begins
6. Workers request tasks → get reduce tasks
7. Workers process with reduce function
8. Workers report reduce results
9. Job complete!

## What problem does MapReduce solves ?

Imagine you have a library with a million books, and you want to count how many times each work appears accross ALL books.

### The naive approach

- Read book 1, count words
- Read book 2, count words
- Read book 3, count words ...
- This would take FOREVER

### The MapReduce approach
- Get 100 people (workers)
- Each person reads 10,000 books (map phase)
- They all count words in parallel
- Then combine everyone's counts (reduce phase)
- Done in a fration of the time

# Core concepts

1. KeyValue Pair

```go
type KeyValue struc {
  Key   string
  Vlaue string
}

```
Everythin in MapReduce is a key-value pair. Think of it like a dictionary.

- Key: "hello"
- Value: 1 (meaning we saw this word onces)

2. Map Function

```go

MapFunc func(docId string, contents string) []KeyValue
```

This map function takes input and break it into key-value pairs.

**Example - Word Count:**
```
Input: "hello world hello humans"

Map function does:
- See "hello" -> emit {Key: "hello", Value: "1"}
- See "world" -> emit {Kye: "world", Value: "1"}
- See "hello" -> emit {Key: "hello", Value: "1"}
- See "humans" -> emit {Key: "humans", Value: "1"}

Output: [
  {Key: "hello", Value: "1"},
  {Key: "world", Value: "1"},
  {Key: "hello", Value: "1"},
  {Key: "humans", Value: "1"}
]
```

3. Reduce Function

```go
ReduceFunc func(key string, values []string) []KeyValue
```

The reduce function takes all values for the same key and combines them.

**Example - Word Count:**
```
Input: key="hello", values=["1", "1", "1"]

Reduce function does:
- Count how many "1"s we have
- There are 3 of them

Output: [{Key: "hello", Value: "3"}]
```

# Architecture Components

## The Master (Coordinator)

```go
type Master struct {
  numMap    int              // How many map tasks total
  NumReduce int              // How many reduce tasks total
  inputs    []string         // The actual input data

  mapStatus    []TaskState   // Track each map task: idle, in-progress, or done
  reduceStatus []TaskState   // Track each reduce task

  intermediate map[int]map[string][]string  // Store map outputs
  Outputs      map[int]string                // Store final results

  mu sync.Mutex  // Lock to prevent race conditions
}
```

The Master is like a project manager:
- Keeps track of all tasks
- Assign work to workers
- Collect results
- Handles failures (if a worker takes too long, reassings the task)

## The Worker (Processor)

```go
type Worker struct {
  ID       int        // Worker's ID number
  Master   *Master    // Reference to the master
  mapFn    MapFunc    // The map function to execute
  reduceFn ReduceFunc // The reduce function to execute
}
```

Workers are like employees:
- Ask the master for work
- Do the work (map or reduce)
- Report results back
- Repeat until all work is done

## Tasks

```go
type Task struct {
  Type       TaskType  // MapTask, ReduceTask, or NoTask
  TaskID     int       // Unique task identifier
  ChunkIndex int       // Which chunk of data
  Data       string    // The actual data to process
}
```

A task is a **work assignment**: "Hey worker, process this piece of data!"

# The Complete Workflow

Let me walk through a word count example step by step:
## Initial Setup
```go
inputs := []string{
  "hello world",      // Document 0
  "hello mapreduce",  // Document 1
  "world of go"       // Document 2
}

// Create master with 3 map tasks (one per document) and 2 reduce tasks
master := NewMaster(inputs, 2)
```

When the master is created:

```
Map tasks:
- Task 0: "hello world" → state: idle
- Task 1: "hello mapreduce" → state: idle
- Task 2: "world of go" → state: idle

Reduce tasks:
- Task 0: state: idle
- Task 1: state: idle
```

## Phase 1: Map Phase

### Worker 1 starts:
```go

worker1.Run()  // Worker asks: "Master, got work for me?"
```

**Master responds:**

```go

// Master checks mapStatus, finds Task 0 is idle
// Assigns it to Worker 1
task := Task{
    Type: MapTask,
    TaskID: 0,
    Data: "hello world"
}
// Changes Task 0 state to "in-progress"
```

**Worker 1 executes map:**

```go
// Worker 1 runs the map function on "hello world"
kvs := mapFn("hello world", "hello world")

// Map function produces:
kvs = [
    {Key: "hello", Value: "1"},
    {Key: "world", Value: "1"}
]

// Worker partitions these by key using hash function
// ihash("hello") % 2 = 1  → goes to reducer 1
// ihash("world") % 2 = 0  → goes to reducer 0

partitions = {
    0: [{Key: "world", Value: "1"}],
    1: [{Key: "hello", Value: "1"}]
}
```

**Worker 1 reports back:**

```go
master.ReportMapDone(0, partitions)

// Master stores this in intermediate storage:
intermediate[0]["world"] = ["1"]
intermediate[1]["hello"] = ["1"]

// Master marks Task 0 as "done"
```

**Meanwhile, Workers 2 and 3 do the same for Tasks 1 and 2!**

After all maps complete:
```
intermediate[0] = {
    "world": ["1", "1"],  // From docs 0 and 2
    "of": ["1"]           // From doc 2
}

intermediate[1] = {
    "hello": ["1", "1", "1"],  // From docs 0, 1, and 2
    "mapreduce": ["1"],        // From doc 1
    "go": ["1"]                // From doc 2
}
```

## Phase 2: Reduce Phase
### Worker 1 asks for more work:

```go

// Master checks: all maps done? YES!
// Master assigns reduce task 0
task := Task{
    Type: ReduceTask,
    TaskID: 0
}
```

**Worker 1 executes reduce:**

```go
// Worker gets partition 0 data from master
partition = {
    "world": ["1", "1"],
    "of": ["1"]
}

// Worker runs reduce function on each key
results = []
for each key in partition:
    reduced := reduceFn("world", ["1", "1"])
    // reduceFn counts: 2 values, so "world" appears 2 times
    results.append({Key: "world", Value: "2"})

    reduced := reduceFn("of", ["1"])
    results.append({Key: "of", Value: "1"})

// Format output
output = "world:2\nof:1\n"
Worker 1 reports back:
gomaster.ReportReduceDone(0, output)

// Master stores final output
Outputs[0] = "world:2\nof:1\n"
```

**Worker 2 does the same for partition 1:**
```
Outputs[1] = "hello:3\nmapreduce:1\ngo:1\n"
Final Results
```

```go
// All tasks complete!
master.Done() // returns true

// Final word counts:
Partition 0: world:2, of:1
Partition 1: hello:3, mapreduce:1, go:1
```

# Key Implementation Details

## 1. Task Assignment (Master.RequestTask)

```go

func (m *Master) RequestTask() Task {
    m.mu.Lock()  // Lock to prevent race conditions so no one task is assign to same worker, and other (see number 4)
    defer m.mu.Unlock()

    // Priority 1: Assign idle map tasks
    for i := 0; i < m.numMap; i++ {
        if m.mapStatus[i].State == "idle" {
            m.mapStatus[i].State = "in-progress"
            m.mapStatus[i].AssignedAt = time.Now()  // Record when assigned
            return Task{Type: MapTask, TaskID: i, Data: m.inputs[i]}
        }
    }

    // Priority 2: Reassign timed-out map tasks (fault tolerance!)
    for i := 0; i < m.numMap; i++ {
        if m.mapStatus[i].State == "in-progress" {
            if time.Since(m.mapStatus[i].AssignedAt) > 5*time.Second {
                // Worker is taking too long, reassign!
                m.mapStatus[i].AssignedAt = time.Now()
                return Task{Type: MapTask, TaskID: i, Data: m.inputs[i]}
            }
        }
    }

    // Priority 3: Check if we can start reduce phase
    allMapDone := true
    for i := 0; i < m.numMap; i++ {
        if m.mapStatus[i].State != "done" {
            allMapDone = false
            break
        }
    }

    if !allMapDone {
        return Task{Type: NoTask}  // Maps not done, wait
    }

    // Priority 4: Assign reduce tasks (only if all maps done!)
    for i := 0; i < m.NumReduce; i++ {
        if m.reduceStatus[i].State == "idle" {
            m.reduceStatus[i].State = "in-progress"
            m.reduceStatus[i].AssignedAt = time.Now()
            return Task{Type: ReduceTask, TaskID: i}
        }
    }

    // Priority 5: Reassign timed-out reduce tasks
    // ... similar to map timeout handling

    return Task{Type: NoTask}  // No work available
}
```

### Why this order?

- Maps must complete BEFORE reduces can start
- Timeout handling provides fault tolerance - if a worker crashes or is slow, another worker can take over
NoTask tells workers to wait

## 2. Hash Partitioning (ihash)

```go
func ihash(key string) int {
    h := fnv.New32a()
    h.Write([]byte(key))
    return int(h.Sum32() & 0x7fffffff)
}
```

**Why do we need this?**

We need to ensure that ALL occurrences of the same key go to the SAME reducer!
```
"hello" from doc 0 → ihash("hello") % 2 = 1 → reducer 1
"hello" from doc 1 → ihash("hello") % 2 = 1 → reducer 1
"hello" from doc 2 → ihash("hello") % 2 = 1 → reducer 1
This way, reducer 1 gets ALL "hello" values and can count them correctly!
```

## 3. Worker Loop

```go
func (w *Worker) Run() {
    for {
        if w.Master.Done() {
            return  // Job complete, stop working
        }

        task := w.Master.RequestTask()

        switch task.Type {
        case MapTask:
            w.doMap(task)
        case ReduceTask:
            w.doReduce(task)
        case NoTask:
            time.Sleep(100 * time.Millisecond)  // Wait before asking again
        }
    }
}
```

Workers continuously:

- Check if job is done
- Ask for work
- Do the work
- Repeat

## 4. Thread Safety (Mutexes)

```go
m.mu.Lock()
defer m.mu.Unlock()
```

**Why?** Multiple workers run in parallel (goroutines). Without locks, they could:
- Read/write the same data simultaneously
- Corrupt the data
- Assign the same task twice

Locks ensure only ONE worker accesses shared data at a time.

---

## Putting It All Together

**Simple Mental Model:**
```
1. Master creates a TODO list of map tasks
2. Workers grab tasks from the list
3. Workers process data and produce intermediate results
4. When ALL maps are done, master creates reduce TODO list
5. Workers grab reduce tasks
6. Workers combine intermediate results into final output
7. Done!
```

**Visualization:**
```
INPUT: ["hello world", "hello go", "world go"]

MAP PHASE:
Worker 1: "hello world" → {hello:1, world:1}
Worker 2: "hello go" → {hello:1, go:1}
Worker 3: "world go" → {world:1, go:1}

SHUFFLE (hash partitioning):
Partition 0: {world:[1,1], go:[1,1]}
Partition 1: {hello:[1,1]}

REDUCE PHASE:
Worker 1: Partition 0 → {world:2, go:2}
Worker 2: Partition 1 → {hello:2}

FINAL OUTPUT:
Partition 0: "world:2\ngo:2"
Partition 1: "hello:2"
