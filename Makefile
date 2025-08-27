# Matrix Archive Go - Makefile for testing and building

.PHONY: all build test test-coverage test-coverage-html clean lint fmt vet deps help run-tests enforce-coverage

# Default target
all: build

# Build the application
build:
	@echo "Building matrix-archive-go..."
	go build -o matrix-archive-go .

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "Running go vet..."
	go vet ./...

# Lint code (requires golangci-lint to be installed)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.54.2"; \
	fi

# Run all tests
test: fmt vet
	@echo "Running tests..."
	go test -v -race ./...

# Run tests with coverage
test-coverage: fmt vet
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Coverage summary:"
	@go tool cover -func=coverage.out | tail -1

# Generate HTML coverage report
test-coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Enforce 100% test coverage
enforce-coverage: test-coverage
	@echo "Enforcing 100% test coverage..."
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Current coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < 100" | bc -l) -eq 1 ]; then \
		echo "❌ FAILED: Test coverage is $$COVERAGE%, which is below 100%"; \
		echo ""; \
		echo "Files with incomplete coverage:"; \
		go tool cover -func=coverage.out | grep -v "100.0%" | grep -v "total:"; \
		echo ""; \
		echo "Please add tests to achieve 100% coverage."; \
		exit 1; \
	else \
		echo "✅ SUCCESS: Test coverage is 100%"; \
	fi

# Run tests quickly (no race detection, useful for development)
test-quick:
	@echo "Running quick tests..."
	go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	go test -v ./...

# Run specific test
test-specific:
	@if [ -z "$(TEST)" ]; then \
		echo "Usage: make test-specific TEST=TestName"; \
		exit 1; \
	fi
	@echo "Running specific test: $(TEST)"
	go test -v -run "$(TEST)" ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Clean build artifacts and test files
clean:
	@echo "Cleaning up..."
	rm -f matrix-archive-go
	rm -f matrix-archive
	rm -f coverage.out
	rm -f coverage.html
	rm -f *.test
	go clean -testcache

# Check for race conditions
race:
	@echo "Checking for race conditions..."
	go test -race ./...

# Generate test coverage badge (requires gopherbadger)
badge:
	@echo "Generating coverage badge..."
	@if command -v gopherbadger > /dev/null; then \
		gopherbadger -fmt=png -o=coverage_badge.png; \
	else \
		echo "gopherbadger not installed. Install with: go install github.com/jpoles1/gopherbadger@latest"; \
	fi

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/jpoles1/gopherbadger@latest

# Run all quality checks
quality: fmt vet lint test-coverage enforce-coverage
	@echo "✅ All quality checks passed!"

# Legacy build target for compatibility
build-legacy:
	go build -o matrix-archive .

# Legacy run targets for compatibility
run-list: build-legacy
	./matrix-archive list

run-import: build-legacy
	./matrix-archive import

run-export: build-legacy
	./matrix-archive export archive.html

run-download: build-legacy
	./matrix-archive download-images

# Install globally
install:
	go install .

# Setup development environment
setup: deps
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env file from template. Please edit it with your credentials."; \
	fi
	@echo "Run 'make build' to build the application"

# Watch for file changes and run tests (requires entr)
watch:
	@echo "Watching for changes... (requires 'entr' to be installed)"
	@if command -v entr > /dev/null; then \
		find . -name "*.go" | entr -c make test; \
	else \
		echo "entr not installed. Install with your package manager (e.g., brew install entr)"; \
	fi

# Show help
help:
	@echo "Matrix Archive Go - Available Make Targets:"
	@echo ""
	@echo "Building:"
	@echo "  build              Build the application (matrix-archive-go)"
	@echo "  build-legacy       Build with legacy name (matrix-archive)"
	@echo "  clean              Clean build artifacts and test files"
	@echo "  install            Install globally"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps               Install/update dependencies"
	@echo "  install-tools      Install development tools"
	@echo "  setup              Setup development environment"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt                Format code"
	@echo "  vet                Run go vet"
	@echo "  lint               Run linter"
	@echo "  race               Check for race conditions"
	@echo ""
	@echo "Testing:"
	@echo "  test               Run all tests with race detection"
	@echo "  test-quick         Run tests without race detection"
	@echo "  test-verbose       Run tests with verbose output"
	@echo "  test-specific      Run specific test (use TEST=TestName)"
	@echo "  bench              Run benchmarks"
	@echo ""
	@echo "Coverage:"
	@echo "  test-coverage      Run tests with coverage"
	@echo "  test-coverage-html Generate HTML coverage report"
	@echo "  enforce-coverage   Enforce 100% test coverage"
	@echo "  badge              Generate coverage badge"
	@echo ""
	@echo "Legacy Commands:"
	@echo "  run-list           Run list command"
	@echo "  run-import         Run import command"
	@echo "  run-export         Run export command"
	@echo "  run-download       Run download-images command"
	@echo ""
	@echo "Development:"
	@echo "  watch              Watch for changes and run tests"
	@echo "  quality            Run all quality checks"
	@echo ""
	@echo "Example usage:"
	@echo "  make test"
	@echo "  make enforce-coverage"
	@echo "  make test-specific TEST=TestBeeperAuth"
	@echo "  make quality"

# Default goal
.DEFAULT_GOAL := help