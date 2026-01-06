# Workflow Engine

A distributed workflow engine built from scratch in Go.

## Features

- ✅ DAG-based task execution
- ✅ Distributed workers
- ✅ Fault tolerance
- ✅ Task retries
- ✅ Parallel execution
- ✅ Persistent state

## Architecture
```
Coordinator (Leader)
    ├─ Parse & validate DAG
    ├─ Schedule tasks
    └─ Track execution

Workers (Multiple)
    ├─ Execute tasks
    └─ Report results
```

## Quick Start
```bash
# Terminal 1: Start coordinator
make run-coordinator

# Terminal 2: Start worker
make run-worker

# Terminal 3: Submit workflow
go run cmd/cli/main.go submit --file examples/simple_workflow.json
```

## Project Structure
```
workflow-engine/
├── cmd/             # Entry points
├── internal/        # Private application code
├── pkg/             # Public libraries
├── examples/        # Example workflows
└── scripts/         # Helper scripts
```

## Development
```bash
# Build all binaries
make build

# Run tests
make test

# Clean
make clean
```
