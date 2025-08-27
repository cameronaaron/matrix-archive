# Matrix Archive - Professional Go Application Makefile

.PHONY: all build test test-coverage test-coverage-html clean lint fmt vet deps help enforce-coverage install

# Default target
all: build

# Build the application using the new structure
build:
	@echo "Building matrix-archive..."
	@mkdir -p bin
	go build -o bin/matrix-archive ./cmd/matrix-archive

# Install globally
install:
	@echo "Installing matrix-archive..."
	go install ./cmd/matrix-archive

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
	go test -v -race ./tests/

# Run tests with coverage
test-coverage: fmt vet
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./tests/
	@echo "Coverage summary:"
	@go tool cover -func=coverage.out | tail -1

# Generate HTML coverage report
test-coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Enforce test coverage (adjust threshold as needed)
enforce-coverage: test-coverage
	@echo "Checking test coverage..."
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Current coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < 80" | bc -l) -eq 1 ]; then \
		echo "❌ FAILED: Test coverage is $$COVERAGE%, which is below 80%"; \
		echo ""; \
		echo "Files with low coverage:"; \
		go tool cover -func=coverage.out | awk '$$3 < 80.0 && $$3 != "0.0" { print $$1 ": " $$3 }'; \
		echo ""; \
		echo "Please add tests to improve coverage."; \
		exit 1; \
	else \
		echo "✅ SUCCESS: Test coverage is $$COVERAGE%"; \
	fi

# Run tests quickly (no race detection, useful for development)
test-quick:
	@echo "Running quick tests..."
	go test ./tests/

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	go test -v ./tests/

# Run specific test
test-specific:
	@if [ -z "$(TEST)" ]; then \
		echo "Usage: make test-specific TEST=TestName"; \
		exit 1; \
	fi
	@echo "Running specific test: $(TEST)"
	go test -v -run "$(TEST)" ./tests/

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./tests/

# Clean build artifacts and test files
clean:
	@echo "Cleaning up..."
	rm -f bin/matrix-archive
	rm -f matrix-archive
	rm -f coverage.out
	rm -f coverage.html
	rm -f *.test
	rm -rf images/ thumbnails/
	go clean -testcache

# Check for race conditions
race:
	@echo "Checking for race conditions..."
	go test -race ./tests/

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run all quality checks
quality: fmt vet lint test-coverage enforce-coverage
	@echo "✅ All quality checks passed!"

# Application commands (for convenience)
run-list: build
	./bin/matrix-archive list

run-import: build
	./bin/matrix-archive import

run-export: build
	./bin/matrix-archive export archive.html

run-download: build
	./bin/matrix-archive download-images

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
	@echo "Matrix Archive - Professional Go Application"
	@echo "============================================="
	@echo ""
	@echo "Building:"
	@echo "  build              Build the application"
	@echo "  install            Install globally to GOPATH/bin"
	@echo "  clean              Clean build artifacts and test files"
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
	@echo "  quality            Run all quality checks"
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
	@echo "  enforce-coverage   Enforce minimum test coverage"
	@echo ""
	@echo "Application Commands:"
	@echo "  run-list           Run list command"
	@echo "  run-import         Run import command"
	@echo "  run-export         Run export command"
	@echo "  run-download       Run download-images command"
	@echo ""
	@echo "Development:"
	@echo "  watch              Watch for changes and run tests"
	@echo ""
	@echo "Example usage:"
	@echo "  make setup         # First time setup"
	@echo "  make build         # Build the application"
	@echo "  make test          # Run tests"
	@echo "  make quality       # Run all quality checks"

# Default goal
.DEFAULT_GOAL := help