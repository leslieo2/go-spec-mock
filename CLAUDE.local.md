# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Go-Spec-Mock** is a lightweight, specification-first Go API mock server that generates realistic mock responses directly from OpenAPI 3.0 specifications. It's a single binary with zero runtime dependencies.

## Core Architecture

The codebase follows a clean, modular architecture with these key components:

- **main.go**: CLI entry point with flag parsing and configuration loading
- **internal/config/**: Unified configuration management with YAML/JSON support
- **internal/parser/**: OpenAPI 3.0 specification parsing using kin-openapi
- **internal/server/**: HTTP server with dynamic routing from OpenAPI paths
- **internal/security/**: API key authentication and rate limiting
- **internal/observability/**: Logging, metrics, tracing, and health checks
- **internal/hotreload/**: Hot reload functionality for specs and config files
- **internal/server/middleware/**: Proxy mode for forwarding undefined endpoints to target servers

### Key Design Patterns
- **Configuration Precedence**: CLI > Env > File > Defaults
- **Middleware Chain**: Request processing pipeline with security, logging, and rate limiting
- **Response Caching**: Pre-generated examples with sync.Map for concurrent access
- **Hot Reload**: File watching with debouncing and atomic configuration updates
- **Proxy Mode**: Mock-first, proxy-fallback strategy for undefined endpoints

## Development Commands

### Build & Run
```bash
make build              # Build binary with version info
make build-all          # Cross-compile for Linux, macOS, Windows
make run-example        # Run with petstore example
make run-example-secure # Run with security features enabled
make dev               # Development server with hot reload (go run)
make watch             # Watch for file changes (requires entr)
```

### Testing & Quality
```bash
make test              # Run all tests with coverage and HTML report
make test-quick        # Run tests without coverage
make vet               # Run go vet for static analysis
make lint              # Run golangci-lint (requires installation)
make security          # Run gosec security scan (requires installation)
make ci                # Full CI pipeline: fmt, vet, lint, test, build
```

### Integration Testing
```bash
make curl-test         # Automated curl tests against running server
make curl-interactive  # Interactive curl testing session
```

### Docker & Deployment
```bash
make docker            # Build Docker image with version tagging
make docker-run        # Run with petstore example in container
make docker-run-config # Run with mounted config file
```

### Utilities
```bash
make generate-key      # Interactive API key generation
make install           # Install to GOPATH/bin
make clean             # Clean build artifacts
make deps              # Install/update dependencies
```

## Key Files & Locations

### Core Implementation
- **main.go:50-67**: CLI flag parsing and configuration loading
- **internal/config/config.go**: Unified configuration structure
- **internal/config/loader.go:30-50**: Configuration precedence logic
- **internal/server/server.go:42-84**: Server initialization with all components
- **internal/server/server.go:208-314**: Dynamic route registration and handling
- **internal/parser/parser.go:40-70**: Route extraction from OpenAPI spec

### Security & Observability
- **internal/security/auth.go:34-50**: API key management initialization
- **internal/security/rate_limiter.go**: Rate limiting strategies
- **internal/observability/**: Structured logging, metrics, tracing, health checks
- **internal/server/middleware/**: Security, CORS, logging, request limiting

### Hot Reload System
- **internal/hotreload/hotreload.go**: Main hot reload manager
- **internal/hotreload/watcher.go**: File system watching
- **internal/server/server.go:316-343**: Server reload implementation

## Configuration Files

- **examples/config/go-spec-mock.yaml**: Complete configuration with all options
- **examples/config/minimal.yaml**: Minimal required configuration
- **examples/config/security-focused.yaml**: Security-first configuration
- **examples/petstore.yaml**: Sample OpenAPI specification for testing

## Environment Variables

All environment variables are prefixed with `GO_SPEC_MOCK_`:
- Server: `HOST`, `PORT`, `METRICS_PORT`, `READ_TIMEOUT`, `WRITE_TIMEOUT`
- Configuration: `SPEC_FILE`, `CONFIG`, `HOT_RELOAD`, `HOT_RELOAD_DEBOUNCE`
- Security: `AUTH_ENABLED`, `RATE_LIMIT_ENABLED`, `RATE_LIMIT_STRATEGY`, `RATE_LIMIT_RPS`
- Proxy: `PROXY_ENABLED`, `PROXY_TARGET`, `PROXY_TIMEOUT`

## Testing Patterns

The codebase uses standard Go testing patterns:
- Table-driven tests with sub-tests (e.g., `internal/parser/parser_test.go:7-30`)
- Test coverage with HTML reports (`make test` generates coverage.html)
- Integration testing with actual HTTP servers (`make curl-test`)
- Mock-free testing where possible, using real OpenAPI specs

## Development Workflow

1. **Start Development**: `make dev` or `make watch` for hot reload
2. **Run Tests**: `make test` for coverage or `make test-quick` for fast feedback
3. **Quality Checks**: `make ci` runs full pipeline before commits
4. **Integration Testing**: `make curl-test` verifies server functionality
5. **Build & Release**: `make release` creates optimized multi-platform binaries

## Common Patterns

- **Error Handling**: Consistent error wrapping with `fmt.Errorf("%w", err)`
- **Logging**: Structured logging with zap throughout the codebase
- **Concurrency**: sync.Map for response caching, sync primitives for coordination
- **Configuration**: Validation methods on all config structs
- **Middleware**: Chainable middleware pattern for HTTP request processing

## Hot Reload Features

The hot reload system provides:
- File watching with configurable debounce (default: 500ms)
- Atomic server reload without downtime
- Only support for both spec files
- Dry-run validation before applying changes

## Usage Examples

Basic usage:
```bash
./bin/go-spec-mock -spec-file ./examples/petstore.yaml
```

With security features:
```bash
./bin/go-spec-mock -spec-file ./examples/petstore.yaml -auth-enabled -rate-limit-enabled
```

With configuration file:
```bash
./bin/go-spec-mock -config ./examples/config/security-focused.yaml -spec-file ./examples/petstore.yaml
```

Generate API key:
```bash
./bin/go-spec-mock -spec-file ./examples/petstore.yaml -generate-key my-app
```

With proxy mode:
```bash
./bin/go-spec-mock -spec-file ./examples/petstore.yaml -proxy-enabled -proxy-target http://localhost:8081
```

With proxy mode and timeout:
```bash
./bin/go-spec-mock -spec-file ./examples/petstore.yaml -proxy-enabled -proxy-target http://localhost:8081 -proxy-timeout 10s
```