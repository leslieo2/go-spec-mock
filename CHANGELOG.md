# Changelog

All notable changes to the Go-Spec-Mock project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.6.0] - 2025-08-26

### ✨ Added
- **HTTPS/TLS Support**: The server can now be configured to run in HTTPS mode, providing secure mocking for clients that require it.
  - New top-level `tls` configuration block in `config.yaml`.
  - New CLI flags: `-tls-enabled`, `-tls-cert-file`, `-tls-key-file`.
  - New environment variables: `GO_SPEC_MOCK_TLS_ENABLED`, `GO_SPEC_MOCK_TLS_CERT_FILE`, `GO_SPEC_MOCK_TLS_KEY_FILE`.
- **TLS Configuration Tests**: Added unit tests to validate the new TLS configuration options.
- **TLS Server Integration Test**: Added an integration test to verify that the server runs correctly in HTTPS mode.

### 🔧 Changed
- The server now operates in an exclusive mode (either HTTP or HTTPS) on its main port, providing clear and predictable behavior.

### 📚 Documentation
- Updated `README.md` to include HTTPS/TLS in the Key Features list and added a comprehensive section on how to configure and use it.
- Updated the project roadmap to reflect the completion of TLS support.

## [1.5.1] - 2025-08-25

### 🐛 Fixed
- **Hot Reload Stability**: Fixed panic in coordinator timer handling during hot reload operations.
- **Cache Key Generation**: Improved cache key generation to prevent collisions and ensure consistent response caching.
- **Graceful Shutdown**: Enhanced server shutdown process for better resource cleanup.

## [1.5.0] - 2025-08-25

### ✨ Added
- **Proxy Mode**: Mock-first, proxy-fallback strategy for undefined endpoints
  - Automatically forwards requests to target servers when no mock route matches
  - Configurable via CLI flags: `-proxy-enabled`, `-proxy-target`, `-proxy-timeout`
  - Environment variables: `GO_SPEC_MOCK_PROXY_ENABLED`, `GO_SPEC_MOCK_PROXY_TARGET`, `GO_SPEC_MOCK_PROXY_TIMEOUT`
- **Enhanced Error Handling**: Better error messages and status codes for proxy failures

## [1.4.0] - 2025-08-24

### ✨ Added
- **Hot Reload System**: Zero-downtime configuration and specification updates
  - File watching for both OpenAPI specs and configuration files
  - Configurable debounce timing to prevent excessive reloads
  - Atomic server reload without dropping requests
  - CLI flags: `-hot-reload`, `-hot-reload-debounce`
  - Environment variables: `GO_SPEC_MOCK_HOT_RELOAD`, `GO_SPEC_MOCK_HOT_RELOAD_DEBOUNCE`

### 🔧 Changed
- **Middleware Refactoring**: Modularized middleware system for better maintainability
- **Server Architecture**: Improved internal server structure for hot reload support

## [1.3.0] - 2025-08-22

### ✨ Added
- **Unified Configuration System**: Complete configuration management overhaul
  - Support for YAML and JSON configuration files
  - CLI flags override config files override environment variables override defaults
  - New configuration file templates and examples
  - All server settings now configurable via CLI, env vars, or config files
- **Enhanced Configuration Templates**: Added comprehensive YAML configuration examples
- **License Change**: Updated from MIT to Apache 2.0 for better enterprise compatibility

### 🔧 Changed
- **Configuration Architecture**: Centralized configuration management for consistency
- **OpenAPI Validation**: Improved validation script and petstore example
- **Documentation**: Enhanced security documentation and configuration guides

### ✅ Added
- **Comprehensive Unit Tests**: Added extensive test coverage for all new features

## [1.2.0] - 2025-08-15

### ✨ Added
- **API Key Authentication**: Secure endpoints with `X-API-Key` header validation.
- **Rate Limiting**: Added configurable rate limiting by IP, API key, or both.
- **Security Configuration**: Manage security features via CLI flags (`-auth-enabled`, `-rate-limit-enabled`, etc.) or a YAML configuration file.
- **API Key Generation**: New `-generate-key <name>` flag to easily create new API keys.
- Added `make run-example-secure` and `make generate-key` to the `Makefile`.

### 📚 Changed
- Updated `README.md` with comprehensive documentation for all new security features.
- Updated `CHANGELOG.md` to reflect all past and present versions based on git history.

## [1.1.1] - 2025-08-14

### ✨ Added
- Added more comprehensive OpenAPI examples for `uspto.yml` and `petstore.yaml`.

### 🚀 Changed
- Improved server routing logic for better handling of complex specifications.
- Refined the observability stack for more accurate metrics and logging.

## [1.1.0] - 2025-08-13

### ✨ Added
- **Comprehensive Observability Stack**:
  - **Logging**: Added structured (JSON) and configurable logging.
  - **Metrics**: Integrated Prometheus metrics endpoint (`/metrics`).
  - **Tracing**: Added support for distributed tracing via OpenTelemetry.
  - **Health Checks**: Added `/health` and `/ready` endpoints.
- **Enhanced CI/CD**: Improved CI workflows with `gosec` for security scanning and better linting rules.

### 📚 Changed
- **Upgraded Go Version**: Project now uses Go 1.24.
- **Overhauled README**: Completely restructured the `README.md` for better clarity, adding sections for Docker, development, and a detailed feature list.
- **Enhanced Docker Support**: Improved `Dockerfile` and development tooling.

### 🐛 Fixed
- Corrected the `gosec` action configuration in the CI workflow.

## [1.0.1] - 2025-08-13

### ⚡️ Performance
- **Optimized Server Performance**: Implemented response caching and pre-computation of routes to significantly reduce response times.
- **Optimized Parser**: Improved the performance and memory usage of the OpenAPI specification parser.

## [1.0.0] - 2025-08-13

### 🚀 Initial Release

Go-Spec-Mock is officially released as a production-ready, enterprise-grade API mock server for Go applications. This first release delivers a complete, zero-dependency solution for generating realistic API responses from OpenAPI 3.0 specifications.

### ✨ New Features

#### Core Functionality
- **CLI Interface** - Simple command-line usage with comprehensive flag support
- **OpenAPI 3.0 Support** - Full parsing of YAML/JSON specifications using kin-openapi
- **HTTP Server** - Built with Go's standard library `net/http` for maximum compatibility
- **Dynamic Routing** - Automatic route registration with path parameter support
- **Static Responses** - Returns examples defined in OpenAPI specifications
- **Method-based Routing** - Proper HTTP method handling (GET, POST, DELETE, PUT, PATCH)

#### Advanced Features
- **Dynamic Status Code Selection** - Override response codes via `__statusCode` query parameter
- **Path Parameter Support** - Handles OpenAPI path parameters like `{petId}`, `{userId}`, etc.
- **Cross-platform Builds** - Single binary distribution for Linux, macOS, and Windows
- **Zero Runtime Dependencies** - No external dependencies required at runtime
- **Insomnia Integration** - Complete workflow integration documented

#### Developer Experience
- **Comprehensive Testing** - Full unit test coverage for parser and server components
- **Professional Documentation** - Detailed README with usage examples and best practices
- **Makefile Support** - Standard development commands (`make build`, `make test`, `make ci`)
- **Docker Support** - Containerized deployment ready

### 🎯 Key Use Cases

- **Frontend Development** - Mock backend APIs during UI development
- **API Design** - Test API specifications before implementation
- **Testing** - Create realistic test environments
- **Microservices** - Mock external service dependencies
- **Insomnia Workflow** - Design → Export → Mock → Test cycle

### 📊 Performance Characteristics

- **Startup Time**: < 1 second typical startup
- **Memory Usage**: Minimal memory footprint
- **Resource Usage**: Low CPU and memory consumption
- **Scalability**: Handles concurrent requests efficiently

### 🔧 Installation & Usage

```bash
# Install
go install github.com/leslieo2/go-spec-mock@latest

# Basic usage
go-spec-mock ./api-spec.yaml

# Custom host and port
go-spec-mock ./api-spec.yaml -host 0.0.0.0 -port 8080

# Test with curl
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/users
```

### 📋 API Endpoints (Example)

When running with the included petstore example:

- **GET** `/` - API documentation and route listing
- **GET** `/health` - Health check endpoint
- **GET** `/pets` - List all pets
- **POST** `/pets` - Create a new pet
- **GET** `/pets/{petId}` - Get pet by ID
- **DELETE** `/pets/{petId}` - Delete pet by ID

### 🛠️ Development Commands

```bash
make build        # Build the binary
make test         # Run all tests
make fmt          # Format code
make lint         # Run linting
make ci           # Full CI pipeline
make run-example  # Run with petstore example
```

### 📦 Distribution

- **Binary**: Single executable for all platforms
- **Docker**: Container images available
- **Go Install**: Direct installation via `go install`
- **Cross-compilation**: Builds for Linux, macOS, Windows

### 🔍 Quality Assurance

- ✅ All tests passing
- ✅ Code formatting verified
- ✅ Linting checks passed
- ✅ Cross-platform builds successful
- ✅ Documentation complete
- ✅ Examples provided and tested

### 🎯 Project Status

**Production Ready** - This release represents a complete, enterprise-grade implementation ready for production use.

### 📈 Future Roadmap

See the [Enterprise Enhancement Roadmap](README.md#-enterprise-implementation-roadmap) in the README for planned security, observability, and performance improvements.

---

### 🙏 Acknowledgments

- Built with [kin-openapi](https://github.com/getkin/kin-openapi) for OpenAPI 3.0 parsing
- Inspired by [Insomnia](https://insomnia.rest/) workflow integration
- Powered by Go standard library for robust HTTP server implementation

[1.0.0]: https://github.com/leslieo2/go-spec-mock/releases/tag/v1.0.0