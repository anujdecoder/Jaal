.PHONY: build test clean lint fmt vet run-example run-basic run-character

# Build all packages
build:
	go build ./...

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-v:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Clean build cache
clean:
	go clean -cache
	go clean -testcache

# Format all Go files
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Run all checks (fmt, vet, build, test)
check: fmt vet build test

# Run the basic example (full-featured users API)
run-basic:
	go run example/basic/main.go

# Run the character example (simple character API)
run-character:
	go run example/character/main.go

# Legacy alias for basic example
run-example: run-basic

# Download dependencies
deps:
	go mod download
	go mod tidy

# Update dependencies
update-deps:
	go get -u ./...
	go mod tidy