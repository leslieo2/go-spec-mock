.PHONY: build build-all build-version test test-quick test-coverage clean run-example install deps fmt lint vet security docker dev watch curl-test curl-interactive help

# Default target
.DEFAULT_GOAL := help

# Binary name
BINARY := go-spec-mock

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)"

# Source files
SOURCES := $(shell find . -name '*.go' -type f)

# Output directory
BIN_DIR := bin

# Build the binary
build: deps fmt
	@echo "Building $(BINARY)..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) .

# Run tests with coverage
test: deps
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run tests without coverage
test-quick:
	@echo "Running tests (quick)..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY)
	rm -rf $(BIN_DIR)/*
	rm -f coverage.out coverage.html
	@echo "Clean complete"

# Run with example petstore
run-example: build
	@echo "Starting mock server with petstore example..."
	$(BIN_DIR)/$(BINARY) ./examples/petstore.yaml

# Install binary to GOPATH/bin
install: deps fmt
	@echo "Installing $(BINARY)..."
	go install $(LDFLAGS) .

# Install dependencies
deps:
	@echo "Checking dependencies..."
	go mod download
	go mod tidy
	go mod verify

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "goimports not found, using go fmt only"; \
	fi

# Lint code (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not found, install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Quick lint with go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Security check (requires gosec)
security:
	@echo "Running security check..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -quiet ./...; \
	else \
		echo "gosec not found, install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Build for multiple platforms
build-all: clean deps fmt
	@echo "Building for multiple platforms..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY)-windows-amd64.exe .
	@echo "Build complete! Binaries are in $(BIN_DIR)/"

# Build with version info
build-version: deps fmt
	@echo "Building version: $(VERSION) (commit: $(COMMIT), date: $(DATE))"
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) .
	@echo "Binary: $(BIN_DIR)/$(BINARY)"

# Full CI pipeline
ci: deps fmt vet lint test build

# Release build (optimized)
release: clean deps fmt vet lint security test build-all
	@echo "Release build complete for version $(VERSION)"

# Docker build
docker:
	@echo "Building Docker image..."
	docker build -t $(BINARY):$(VERSION) .
	docker tag $(BINARY):$(VERSION) $(BINARY):latest
	@echo "Docker image built: $(BINARY):$(VERSION)"

# Quick development server
dev:
	@echo "Starting development server..."
	go run . ./examples/petstore.yaml -port 8080

# Watch for changes and rebuild (requires entr)
watch:
	@echo "Starting file watcher..."
	@if command -v entr >/dev/null 2>&1; then \
		find . -name "*.go" | entr -r make dev; \
	else \
		echo "entr not found, install with: brew install entr (macOS) or apt-get install entr (Ubuntu)"; \
	fi

# Quick curl test commands
curl-test: build
	@echo "Starting server for curl testing..."
	$(BIN_DIR)/$(BINARY) ./examples/petstore.yaml -port 8085 & 
	SERVER_PID=$$!; \
	sleep 2; \
	echo "=== Testing endpoints ==="; \
	curl -s http://localhost:8085/ | jq -r '.message || .'; \
	curl -s http://localhost:8085/health | jq -r '.status || .'; \
	curl -s http://localhost:8085/pets | jq -r '.[0].name // .[0].Name // empty'; \
	curl -s http://localhost:8085/pets/123 | jq -r '.name // .Name // empty'; \
	curl -s "http://localhost:8085/pets/123?__statusCode=404" | jq -r '.error // .Error // empty'; \
	curl -s -X DELETE http://localhost:8085/pets/123 | jq -r '.error // .Error // empty'; \
	echo "=== All curl tests completed ==="; \
	kill $$SERVER_PID 2>/dev/null || true

# Interactive curl testing
curl-interactive: build
	@echo "Starting server for interactive testing..."
	@echo "Server available at http://localhost:8080"
	@echo "Try these commands:"
	@echo "  curl http://localhost:8080/"
	@echo "  curl http://localhost:8080/health"
	@echo "  curl http://localhost:8080/pets"
	@echo "  curl http://localhost:8080/pets/123"
	@echo "  curl 'http://localhost:8080/pets/123?__statusCode=404'"
	@echo "  curl -X POST http://localhost:8080/pets"
	@echo "  curl -X DELETE http://localhost:8080/pets/123"
	$(BIN_DIR)/$(BINARY) ./examples/petstore.yaml -port 8080

# Show help
help:
	@echo "$(BINARY) - Go API Mock Server"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  dev         - Start development server"
	@echo "  watch       - Watch for file changes and rebuild"
	@echo "  run-example - Run with petstore example"
	@echo ""
	@echo "Testing:"
	@echo "  test        - Run tests with coverage report"
	@echo "  test-quick  - Run tests without coverage"
	@echo "  curl-test   - Quick automated curl tests"
	@echo "  curl-interactive - Interactive curl testing"
	@echo ""
	@echo "Build:"
	@echo "  build       - Build the binary"
	@echo "  build-all   - Build for all platforms (Linux, macOS, Windows)"
	@echo "  build-version - Build with version info"
	@echo "  release     - Full release build (optimized)"
	@echo ""
	@echo "Quality:"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  lint        - Run golangci-lint"
	@echo "  security    - Run security check"
	@echo ""
	@echo "Utilities:"
	@echo "  install     - Install to GOPATH/bin"
	@echo "  clean       - Clean build artifacts"
	@echo "  deps        - Install/update dependencies"
	@echo "  ci          - Full CI pipeline"
	@echo "  docker      - Build Docker image"
	@echo ""
	@echo "Version: $(VERSION) ($(COMMIT))"
	@echo "Build date: $(DATE)"