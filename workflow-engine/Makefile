.PHONY: build clean run-coordinator run-worker test

# Build all binaries
build:
	@echo "Building workflow engine..."
	@go build -o bin/coordinator ./cmd/coordinator
	@go build -o bin/worker ./cmd/worker
	@go build -o bin/workflow-cli ./cmd/cli
	@echo "Build complete"

# Run coordinator
run-coordinator:
	@go run cmd/coordinator/main.go --port 8080 --data-dir ./data/coordinator

# Run worker
run-worker:
	@go run cmd/worker/main.go --coordinator localhost:8080 --id worker-1

# Run multiple workers
run-workers:
	@go run cmd/worker/main.go --coordinator localhost:8080 --id worker-1 &
	@go run cmd/worker/main.go --coordinator localhost:8080 --id worker-2 &
	@go run cmd/worker/main.go --coordinator localhost:8080 --id worker-3 &

# Clean build artifacts
clean:
	@rm -rf bin/
	@rm -rf data/
	@echo "Cleaned"

# Run tests
test:
	@go test -v ./...

# Run with race detector
test-race:
	@go test -race -v ./...

# Install dependencies
deps:
	@go mod download
	@go mod tidy
