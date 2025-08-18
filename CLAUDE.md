# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go-Spec-Mock is a lightweight, specification-first Go API mock server that generates realistic mock responses directly from OpenAPI 3.0 specifications. It's designed for zero-configuration mocking with enterprise-grade security and observability features.

## Key Architecture

### Core Components

- **main.go**: CLI entry point with flag parsing and server initialization
- **internal/parser/**: OpenAPI 3.0 specification parsing using kin-openapi
- **internal/server/**: HTTP server with routing, middleware, and response generation
- **internal/security/**: API key authentication and rate limiting
- **internal/observability/**: Logging, metrics, and distributed tracing

### Data Flow

1. **Specification Loading**: Parser loads and validates OpenAPI specs
2. **Route Registration**: Server registers HTTP routes from OpenAPI paths
3. **Request Handling**: Middleware chain (auth, rate limiting, CORS, observability)
4. **Response Generation**: Dynamic response generation from OpenAPI examples/schemas
5. **Caching**: In-memory response caching with sync.Map for performance

## Development Commands

### Build & Run
```bash
make build              # Build binary to bin/go-spec-mock
make run-example        # Run with petstore example
make dev                # Start development server on :8080
make watch              # Auto-rebuild on file changes (requires entr)
```

### Testing
```bash
make test               # Run tests with coverage report
make test-quick         # Run tests without coverage
make curl-test          # Automated endpoint testing
make curl-interactive   # Interactive testing server
```

### Code Quality
```bash
make fmt                # Format Go code with goimports
make lint               # Run golangci-lint
make vet                # Run go vet
make security           # Run gosec security scan
make ci                 # Full CI pipeline
```

### Cross-Platform Builds
```bash
make build-all          # Build for Linux, macOS, Windows
make build-version      # Build with version info
make release            # Full release build
```

### Docker
```bash
make docker             # Build Docker image
make docker-run         # Run with petstore example
make docker-run-dev     # Interactive Docker development
```

## Quick Start

```bash
# Install dependencies
go mod tidy

# Build and run with example
make build
./bin/go-spec-mock ./examples/petstore.yaml

# Or install globally
go install .
go-spec-mock ./examples/petstore.yaml
```

## Testing Endpoints

```bash
# Basic endpoints
curl http://localhost:8080/health
curl http://localhost:8080/metrics
curl http://localhost:8080/pets
curl http://localhost:8080/pets/123

# Dynamic status codes
curl "http://localhost:8080/pets/123?__statusCode=404"
```

## Security Features

- **API Key Authentication**: Configurable via CLI flags or security.yaml
- **Rate Limiting**: IP-based, API key-based, or both strategies
- **Security Headers**: CORS, CSP, HSTS configuration
- **Request Size Limiting**: 10MB default limit

## Configuration Files

- **security.yaml**: Authentication and rate limiting configuration
- **examples/petstore.yaml**: Sample OpenAPI 3.0 specification
- **go-spec-mock-api.yaml**: Main API specification

## Key Patterns

- **Caching**: Uses sync.Map for thread-safe response caching
- **Middleware Chain**: Request logging, auth, rate limiting, CORS
- **Observability**: Structured JSON logging, Prometheus metrics, OpenTelemetry tracing
- **Graceful Shutdown**: 30-second timeout for clean shutdown
- **Error Handling**: Structured error responses with proper HTTP status codes