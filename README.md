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
*   **🛡️ Security First:** Built-in support for API key authentication and rate limiting to simulate real-world security policies.
*   **📦 Zero Dependencies:** A single, cross-platform binary with no runtime dependencies. Works on Linux, macOS, and Windows.
*   **🔧 Developer-Friendly:** Simple CLI, seamless integration with tools like [Insomnia](https://insomnia.rest/), and a comprehensive set of utility endpoints.
*   **🏢 Enterprise-Ready:** Built with a clean, testable, and performant Go architecture.

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

- [Go](https://go.dev/doc/install) version 1.21 or later.

### Installation

Install the `go-spec-mock` CLI with a single command:

```bash
go install github.com/leslieo2/go-spec-mock@latest
```

### Quick Usage

1.  **Get an OpenAPI Spec:** Create your own, or download one like the [Swagger Petstore](https://petstore3.swagger.io/api/v3/openapi.json) spec.

2.  **Start the Mock Server:** Point `go-spec-mock` to your specification file.
    ```bash
    # Start mocking using the example Petstore spec
    go-spec-mock ./examples/petstore.yaml
    ```

3.  You'll see output indicating the server is running and which routes are available:
    ```
    2023/10/27 10:00:00 Starting server on http://127.0.0.1:8080
    2023/10/27 10:00:00 ----------------------------------------
    2023/10/27 10:00:00 Registered Route: GET /
    2023/10/27 10:00:00 Registered Route: GET /health
    2023/10/27 10:00:00 Registered Route: GET /pets
    ...
    ```

4.  **Test Your Endpoints:** In another terminal, use any HTTP client like `curl` to interact with your mock API.
    ```bash
    # Get a list of all pets
    curl http://localhost:8080/pets
  
    # Get a specific pet by its ID
    curl http://localhost:8080/pets/123
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
| `make generate-key` | Generate a new API key interactively. |
| `make test` | Run all unit tests. |
| `make fmt` | Format the Go source code. |
| `make lint` | Run `golangci-lint` to check for code quality issues. |
| `make security` | Run security scan with `gosec`. |
| `make ci` | Run the full CI pipeline (format, lint, test). |
| `make build-all` | Cross-compile binaries for Linux, macOS, and Windows. |
| `make curl-test` | Run automated `curl` tests against the example server. |

### Project Structure

```
.
├── Makefile                      # Development commands
├── README.md                     # This file
├── go.mod                        # Go module definition
├── main.go                       # CLI entry point
├── examples/
│   ├── petstore.yaml             # Sample OpenAPI spec
│   └── config/                   # Configuration examples
├── internal/
│   ├── config/                   # Configuration management
│   ├── parser/                   # OpenAPI specification parsing logic
│   ├── server/                   # HTTP server and routing logic
│   ├── security/                 # Authentication and rate limiting
│   └── observability/            # Logging, metrics, and tracing
```

## 🛣️ Roadmap

The project is currently at **v1.0.0** and is stable for general use. The future roadmap is focused on adding enterprise-grade features for security, observability, and configuration.

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
<summary><strong>📋 Phase 2: Enterprise Enhancements (Planned)</strong></summary>

#### 🔒 Security & Robustness
- [x] Request size limiting
- [x] Configurable log levels (DEBUG, INFO, WARN, ERROR)
- [ ] Sensitive data masking in logs

#### 📊 Observability
- [x] Structured (JSON) logging
- [x] Prometheus metrics endpoint (`/metrics`)
- [x] Distributed tracing support (OpenTelemetry)
- [x] Health check endpoint (`/health`)
- [x] Readiness probe (`/ready`)

#### 🛡️ Advanced Configuration
- [x] CORS (Cross-Origin Resource Sharing) configuration
- [x] Rate limiting
- [x] API key authentication
- [x] Configuration via CLI flags and environment variables
- [x] Customizable server timeouts and ports
- [ ] HTTPS/TLS support
- [x] Configuration file support (YAML/JSON)

#### 📦 Deployment
- [ ] Official Docker images on Docker Hub
- [ ] Example Helm charts for Kubernetes deployment

</details>

## 🙏 Acknowledgments

-   **[kin-openapi](https://github.com/getkin/kin-openapi)** for its robust OpenAPI 3.0 parsing library.
-   **[Insomnia](https://insomnia.rest/)** for inspiring a seamless design-first workflow.
-   The **Go Team** for the powerful and simple standard library.

## 📄 License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.