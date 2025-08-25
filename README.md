<div align="center">

# Go-Spec-Mock

**A lightweight, specification-first Go API mock server.**

*Generate realistic mock responses directly from your OpenAPI 3.0 specifications. No code generation, no complex setup—just a single command.*

</div>

<p align="center">
  <a href="https://github.com/leslieo2/go-spec-mock/actions/workflows/ci.yml"><img src="https://github.com/leslieo2/go-spec-mock/actions/workflows/ci.yml/badge.svg" alt="Build Status"></a>
  <a href="https://goreportcard.com/report/github.com/leslieo2/go-spec-mock"><img src="https://goreportcard.com/badge/github.com/leslieo2/go-spec-mock" alt="Go Report Card"></a>
  <a href="https://github.com/leslieo2/go-spec-mock/releases"><img src="https://img.shields.io/github/v/release/leslieo2/go-spec-mock" alt="Latest Release"></a>
  <a href="https://github.com/leslieo2/go-spec-mock/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License: Apache 2.0"></a>
</p>

---

## ✨ Key Features

*   **🚀 Specification-First:** Instantly mock any API by providing an OpenAPI 3.0 (YAML/JSON) file.
*   **⚡️ Dynamic Mocking:** Serves static examples from your spec and allows dynamic status code overrides for testing different scenarios.
*   **🔥 Hot Reload:** Automatically reloads OpenAPI specifications and configuration files without server restart for rapid development.
*   **🛡️ Enterprise Security:** Comprehensive security suite with API key authentication, rate limiting, CORS, security headers, and role-based access control.
*   **🔄 Smart Proxy:** Automatically forwards requests for undefined endpoints to a real backend server, enabling hybrid mocking with configurable timeouts.
*   **📊 Full Observability:** Built-in Prometheus metrics, structured JSON logging, OpenTelemetry tracing, and health/readiness endpoints.
*   **📦 Zero Dependencies:** A single, cross-platform binary with no runtime dependencies. Works on Linux, macOS, and Windows.
*   **🔧 Developer-Friendly:** Simple CLI with comprehensive flags, seamless integration with tools like [Insomnia](https://insomnia.rest/), and extensive development tooling.
*   **🏢 Production-Ready:** Enterprise-grade architecture with comprehensive testing, Docker support, and configuration management.

## 📖 Table of Contents

- [🚀 Getting Started](#-getting-started)
    - [Prerequisites](#prerequisites)
    - [Installation](#installation)
    - [Quick Usage](#quick-usage)
- [📖 Usage Guide](#-usage-guide)
    - [Insomnia Integration](#insomnia-integration)
    - [Dynamic Status Code Selection](#dynamic-status-code-selection)
- [⚙️ Configuration](#️-configuration)
    - [Configuration Precedence](#configuration-precedence)
    - [CLI Flags & Environment Variables](#cli-flags--environment-variables)
    - [Configuration File](#configuration-file)
    - [Security Features](#security-features)
- [🐳 Docker Usage](#-docker-usage)
- [👨‍💻 Development & Contribution](#-development--contribution)
    - [Setup](#setup)
    - [Development Commands](#development-commands)
    - [Project Structure](#project-structure)
- [🛣️ Roadmap](#️-roadmap)
- [🙏 Acknowledgments](#-acknowledgments)
- [📄 License](#-license)

## 🚀 Getting Started

### Prerequisites

- [Go](https://go.dev/doc/install) version 1.24 or later.

### Installation

Install the `go-spec-mock` CLI with a single command:

```bash
go install github.com/leslieo2/go-spec-mock@latest
```

### Quick Usage

1.  **Get an OpenAPI Spec:** Create your own, or use the provided examples.

2.  **Start the Mock Server:** Point `go-spec-mock` to your specification file.
    ```bash
    # Start mocking using the example Petstore spec
    go-spec-mock -spec-file ./examples/petstore.yaml
    
    # With security features enabled
    go-spec-mock -spec-file ./examples/petstore.yaml -auth-enabled -rate-limit-enabled
    
    # With configuration file
    go-spec-mock -config ./examples/config/security-focused.yaml -spec-file ./examples/petstore.yaml
    ```

3.  You'll see structured output indicating the server is running:
    ```json
    {"level":"info","ts":"2025-08-25T10:00:00.000Z","msg":"Starting server","host":"localhost","port":"8080"}
    {"level":"info","ts":"2025-08-25T10:00:00.000Z","msg":"Registered route","method":"GET","path":"/"}
    {"level":"info","ts":"2025-08-25T10:00:00.000Z","msg":"Registered route","method":"GET","path":"/health"}
    {"level":"info","ts":"2025-08-25T10:00:00.000Z","msg":"Registered route","method":"GET","path":"/pets"}
    ```

4.  **Test Your Endpoints:** In another terminal, use any HTTP client like `curl` to interact with your mock API.
    ```bash
    # Get a list of all pets
    curl http://localhost:8080/pets
  
    # Get a specific pet by its ID
    curl http://localhost:8080/pets/123
    
    # Check health status
    curl http://localhost:8080/health
    
    # View Prometheus metrics
    curl http://localhost:9090/metrics
    ```

## 📖 Usage Guide

### Insomnia Integration

`go-spec-mock` is perfect for a design-first workflow with Insomnia.

1.  **Design API in Insomnia:** Create your endpoints, schemas, and examples in Insomnia's "Design" tab.
2.  **Export Spec:** Click the collection dropdown, then select **Export** -> **OpenAPI 3.0** (as YAML or JSON). Save it as `api-spec.yaml`.
3.  **Start Mocking:**
    ```bash
    go-spec-mock ./api-spec.yaml
    ```
4.  **Test:** Point your frontend application or Insomnia's "Debug" tab to `http://localhost:8080` to test against the live mock server.

### Dynamic Status Code Selection

Test different response scenarios by overriding the status code with the `__statusCode` query parameter. The server will look for a matching response example in your spec.

```bash
# Get the default 200 OK response
curl http://localhost:8080/pets/1

# Force a 404 Not Found response
curl "http://localhost:8080/pets/1?__statusCode=404"

# Force a 400 Bad Request response
curl "http://localhost:8080/pets?__statusCode=400"
```

## ⚙️ Configuration

### Configuration Precedence

Go-Spec-Mock supports flexible configuration with the following precedence:

1.  **CLI flags** (highest priority)
2.  **Environment variables**
3.  **Configuration file values**
4.  **Default values** (lowest priority)

### CLI Flags & Environment Variables

All settings can be configured via CLI flags or environment variables.

#### Server & General Configuration

| Flag | Environment Variable | Description | Default |
| :--- | :--- | :--- | :--- |
| `-spec-file` | `GO_SPEC_MOCK_SPEC_FILE` | Path to OpenAPI specification file. | `Required` |
| `-config` | `GO_SPEC_MOCK_CONFIG` | Path to configuration file (YAML/JSON). | `""` |
| `-host` | `GO_SPEC_MOCK_HOST` | The host to bind the server to. | `localhost` |
| `-port` | `GO_SPEC_MOCK_PORT` | The port to run the server on. | `8080` |
| `-metrics-port` | `GO_SPEC_MOCK_METRICS_PORT` | The port for the metrics server. | `9090` |
| `-read-timeout` | `GO_SPEC_MOCK_READ_TIMEOUT` | HTTP server read timeout. | `15s` |
| `-write-timeout` | `GO_SPEC_MOCK_WRITE_TIMEOUT` | HTTP server write timeout. | `15s` |
| `-idle-timeout` | `GO_SPEC_MOCK_IDLE_TIMEOUT` | HTTP server idle timeout. | `60s` |
| `-max-request-size` | `GO_SPEC_MOCK_MAX_REQUEST_SIZE` | Maximum request size in bytes. | `10485760` |
| `-shutdown-timeout` | `GO_SPEC_MOCK_SHUTDOWN_TIMEOUT` | Graceful shutdown timeout. | `30s` |

#### Hot Reload Configuration

| Flag | Environment Variable | Description | Default |
| :--- | :--- | :--- | :--- |
| `-hot-reload` | `GO_SPEC_MOCK_HOT_RELOAD` | Enable automatic hot reloading of spec/config files. | `true` |
| `-hot-reload-debounce` | `GO_SPEC_MOCK_HOT_RELOAD_DEBOUNCE` | Debounce duration for file changes. | `500ms` |

#### Proxy Configuration

| Flag | Environment Variable | Description | Default |
| :--- | :--- | :--- | :--- |
| `-proxy-enabled` | `GO_SPEC_MOCK_PROXY_ENABLED` | Enable proxy mode for undefined endpoints. | `false` |
| `-proxy-target` | `GO_SPEC_MOCK_PROXY_TARGET` | Target server URL for proxy mode. | `""` |
| `-proxy-timeout` | `GO_SPEC_MOCK_PROXY_TIMEOUT` | Timeout for proxy requests. | `30s` |

#### Security Configuration

| Flag | Environment Variable | Description | Default |
| :--- | :--- | :--- | :--- |
| `-auth-enabled` | `GO_SPEC_MOCK_AUTH_ENABLED` | Enable API key authentication. | `false` |
| `-auth-config` | `GO_SPEC_MOCK_AUTH_CONFIG` | Path to a security configuration file. | `""` |
| `-rate-limit-enabled` | `GO_SPEC_MOCK_RATE_LIMIT_ENABLED` | Enable rate limiting. | `false` |
| `-rate-limit-strategy` | `GO_SPEC_MOCK_RATE_LIMIT_STRATEGY` | Strategy: `ip`, `api_key`, `both`. | `ip` |
| `-rate-limit-rps` | `GO_SPEC_MOCK_RATE_LIMIT_RPS` | Global rate limit in requests per second. | `100` |
| `-generate-key <name>` | `N/A` | Generate a new API key and exit. | `""` |

### Configuration File

For complex setups, especially involving security, a YAML or JSON configuration file is recommended.

```bash
# Use a configuration file
go-spec-mock -config ./config.yaml -spec-file ./examples/petstore.yaml
```

**Example configuration files are provided:**
*   `examples/config/go-spec-mock.yaml` - Complete configuration with all options.
*   `examples/config/minimal.yaml` - Minimal required configuration.
*   `examples/config/security-focused.yaml` - Security-first configuration.

#### Hot Reload Configuration

Enable automatic reloading of OpenAPI specifications without server restart:

```yaml
# Hot Reload Configuration
hot_reload:
  enabled: true
  debounce: "500ms"
```

With hot reload enabled, simply save your OpenAPI spec file and the server will automatically reload the new specifications without requiring a restart.

#### 🔄 Proxying Undefined Endpoints

The proxy feature allows `go-spec-mock` to act as a pass-through for any requests that are not defined in your OpenAPI specification. This is useful when you want to mock a few endpoints of a larger, existing API without needing to define the entire API surface.

When enabled, if a request comes in for a path that is not in the spec file (e.g., `/api/v1/existing-endpoint`), it will be forwarded to the configured target server.

```yaml
# Proxy configuration
proxy:
  enabled: true
  target: "https://api.production.com" # The real backend API
  timeout: "15s"
```

With this configuration, you can make requests to `http://localhost:8080`:
-   Requests to paths defined in your `spec.yaml` (e.g., `/pets`) will be mocked by `go-spec-mock`.
-   Requests to any other path (e.g., `/users`, `/orders`) will be proxied to `https://api.production.com`.

### Security Features

Secure your mock server with API key authentication, rate limiting, security headers, and CORS. These are best configured via a configuration file.

#### API Key Authentication

Enable with `-auth-enabled` or in a config file. You can generate keys with the `-generate-key` flag.

```bash
# Generate a key named "my-app"
go-spec-mock -spec-file ./examples/petstore.yaml -generate-key "my-app"
```

**Example `config.yaml`:**
```yaml
security:
  auth:
    enabled: true
    header_name: "X-API-Key"      # or "Authorization" for Bearer tokens
    query_param_name: "api_key"
    keys:
      - key: "your-generated-key-here"
        name: "my-app"
        enabled: true
```

**Usage:**
```bash
# Header authentication
curl -H "X-API-Key: your-key-here" http://localhost:8080/pets

# Query parameter authentication
curl "http://localhost:8080/pets?api_key=your-key-here"
```

#### Rate Limiting

Enable with `-rate-limit-enabled`. The server returns standard `X-RateLimit-*` headers.

**Example `config.yaml`:**
```yaml
security:
  rate_limit:
    enabled: true
    strategy: "both" # ip, api_key, or both
    global:
      requests_per_second: 100
      burst_size: 200
      window_size: "1m"
```

#### CORS & Security Headers

Configure Cross-Origin Resource Sharing (CORS) and other security headers for enterprise-grade protection.

**Example `config.yaml`:**
```yaml
security:
  cors:
    enabled: true
    allowed_origins: ["http://localhost:3000", "https://yourdomain.com"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["Content-Type", "Authorization", "X-API-Key"]
    allow_credentials: false
    max_age: 86400
  headers:
    enabled: true
    content_security_policy: "default-src 'self'"
    hsts_max_age: 31536000
    allowed_hosts: ["localhost", "yourdomain.com"]
```

### Observability Endpoints

The server provides built-in observability endpoints:

| Endpoint | Description |
| :--- | :--- |
| `/` | API documentation with available endpoints |
| `/health` | Health check endpoint with service status |
| `/ready` | Readiness probe for load balancers |
| `/metrics` | Prometheus metrics endpoint (on metrics port) |

**Usage examples:**
```bash
# Check health status
curl http://localhost:8080/health

# View Prometheus metrics (assuming default port 9090)
curl http://localhost:9090/metrics
```

## 🐳 Docker Usage

A `Dockerfile` is included for easy containerization.

```bash
# 1. Build the Docker image
docker build -t go-spec-mock .

# 2. Run the container with a mounted config file
docker run -p 8080:8080 -p 9090:9090 \
  -v $(pwd)/examples:/app/examples \
  go-spec-mock:latest -config ./examples/config/minimal.yaml -spec-file ./examples/petstore.yaml

# 3. Run with configuration via environment variables
docker run -p 8081:8081 -p 9091:9091 \
  -e GO_SPEC_MOCK_PORT=8081 \
  -e GO_SPEC_MOCK_METRICS_PORT=9091 \
  -v $(pwd)/examples/petstore.yaml:/app/petstore.yaml \
  go-spec-mock:latest -spec-file /app/petstore.yaml
```

## 👨‍💻 Development & Contribution

Contributions are welcome! Please feel free to open an issue or submit a pull request.

### Setup

1.  Clone the repository:
    ```bash
    git clone https://github.com/leslieo2/go-spec-mock.git
    cd go-spec-mock
    ```
2.  Install dependencies:
    ```bash
    go mod tidy
    ```

### Development Commands

This project uses a `Makefile` to streamline common development tasks.

| Command | Description |
| :--- | :--- |
| `make build` | Build the `go-spec-mock` binary for your OS. |
| `make run-example` | Run the server with the example `petstore.yaml` spec. |
| `make run-example-secure` | Run with security features enabled. |
| `make run-example-secure-config` | Run with security-focused configuration. |
| `make run-example-minimal` | Run with minimal configuration. |
| `make generate-key` | Generate a new API key interactively. |
| `make test` | Run all tests with coverage report. |
| `make test-quick` | Run tests without coverage. |
| `make fmt` | Format the Go source code. |
| `make lint` | Run `golangci-lint` to check for code quality issues. |
| `make vet` | Run `go vet` for static analysis. |
| `make security` | Run security scan with `gosec`. |
| `make ci` | Run the full CI pipeline (format, lint, test, build). |
| `make build-all` | Cross-compile binaries for Linux, macOS, and Windows. |
| `make build-version` | Build with version information. |
| `make curl-test` | Run automated `curl` tests against the example server. |
| `make curl-interactive` | Interactive curl testing session. |
| `make docker` | Build Docker image. |
| `make docker-run` | Run with petstore example in container. |
| `make dev` | Start development server. |
| `make watch` | Watch for file changes and rebuild. |

### Project Structure

```
.
├── Makefile                      # Development commands
├── README.md                     # This file
├── go.mod                        # Go module definition
├── main.go                       # CLI entry point
├── examples/
│   ├── petstore.yaml             # Sample OpenAPI spec
│   ├── uspto.yml                 # USPTO API specification example
│   └── config/                   # Configuration examples
├── internal/
│   ├── config/                   # Configuration management (YAML/JSON)
│   ├── parser/                   # OpenAPI specification parsing logic
│   ├── server/                   # HTTP server and routing logic
│   │   └── middleware/           # HTTP middleware chain (CORS, security, logging, proxy)
│   ├── security/                 # Authentication and rate limiting
│   ├── observability/            # Logging, metrics, tracing, and health checks
│   ├── hotreload/                # Hot reload functionality for specs and config
│   ├── proxy/                    # Proxy functionality for undefined endpoints
├── Dockerfile                    # Multi-stage Docker build
├── CHANGELOG.md                  # Version history and features
├── LICENSE                       # Apache 2.0 license
├── coverage.html                 # Test coverage report
├── go-spec-mock-api.yaml         # Project's own OpenAPI specification
```

## 🛣️ Roadmap

The project is currently at **v1.5.1** and is production-ready with enterprise-grade security, observability, and configuration features. All core functionality is complete and battle-tested.

<details>
<summary><strong>✅ Phase 1: Core Features (Complete)</strong></summary>

- [x] OpenAPI 3.0 specification parsing
- [x] Dynamic HTTP routing from spec paths
- [x] Static example response generation
- [x] Dynamic status code override (`__statusCode`)
- [x] Cross-platform builds (Linux, macOS, Windows)
- [x] Comprehensive unit tests and documentation

</details>

<details>
<summary><strong>✅ Phase 2: Enterprise Enhancements (Complete)</strong></summary>

#### 🔒 Security & Robustness
- [x] Request size limiting
- [x] Configurable log levels (DEBUG, INFO, WARN, ERROR)
- [x] Comprehensive security configuration (YAML/JSON)
- [x] API key authentication with role-based access
- [x] Rate limiting by IP, API key, or both
- [x] CORS configuration with security headers
- [ ] Sensitive data masking in logs

#### 📊 Observability
- [x] Structured (JSON) logging
- [x] Prometheus metrics endpoint (`/metrics`)
- [x] Distributed tracing support (OpenTelemetry)
- [x] Health check endpoint (`/health`)
- [x] Readiness probe (`/ready`)

#### 🛡️ Advanced Configuration
- [x] CORS (Cross-Origin Resource Sharing) configuration
- [x] Rate limiting with granular controls
- [x] Configuration via CLI flags and environment variables
- [x] Customizable server timeouts and ports
- [ ] HTTPS/TLS support
- [x] Configuration file support (YAML/JSON)

#### 📦 Deployment
- [x] Docker support with multi-stage builds
- [ ] Official Docker images on Docker Hub
- [ ] Example Helm charts for Kubernetes deployment

#### 🔥 Developer Experience
- [x] Hot reload for specifications and configuration
- [x] Interactive API key generation
- [x] Comprehensive CLI flags and environment variables
</details>

<details>
<summary><strong>🚀 Phase 3: <Adv></Adv>anced Enterprise Features (Planned)</strong></summary>

#### 🔥 **Core Enterprise Priorities**
- [ ] **Smart Proxy Routing** - Spec-based intelligent matching for proxy requests
- [ ] **JWT/OAuth 2.0 Integration** - Modern enterprise security standards
- [ ] **OpenTelemetry Distributed Tracing** - Production debugging and monitoring
- [ ] **WebSocket Protocol Mocking** - Real-time API support

#### ⚡ **Enhanced Proxy & Security**
- [ ] **Response Transformation** - Format conversion for proxied requests
- [ ] **Stateful Mocking** - Complex business scenario testing
- [ ] **RBAC & Multi-tenancy** - Team collaboration and access control

#### 📈 **Production Observability**
- [ ] **Custom Business Metrics** - Tailored analytics for enterprise needs
- [ ] **Performance Profiling** - Real-time performance insights
- [ ] **Enhanced Health Monitoring** - Comprehensive service status

</details>

## 🙏 Acknowledgments

-   **[kin-openapi](https://github.com/getkin/kin-openapi)** for its robust OpenAPI 3.0 parsing library.
-   **[Insomnia](https://insomnia.rest/)** for inspiring a seamless design-first workflow.
-   The **Go Team** for the powerful and simple standard library.

## 📄 License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.