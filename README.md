<div align="center">

# Go-Spec-Mock

**A lightweight, specification-first Go API mock server.**

*Generate realistic mock responses directly from your OpenAPI 3.0 specifications. No code generation, no complex setup—just a single command.*

</div>

<p align="center">
  <a href="https://github.com/leslieo2/go-spec-mock/actions/workflows/ci.yml"><img src="https://github.com/leslieo2/go-spec-mock/actions/workflows/ci.yml/badge.svg" alt="Build Status"></a>
  <a href="https://goreportcard.com/report/github.com/leslieo2/go-spec-mock"><img src="https://goreportcard.com/badge/github.com/leslieo2/go-spec-mock" alt="Go Report Card"></a>
  <a href="https://github.com/leslieo2/go-spec-mock/releases"><img src="https://img.shields.io/github/v/release/leslieo2/go-spec-mock" alt="Latest Release"></a>
  <a href="https://github.com/leslieo2/go-spec-mock/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
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
    - [Basic Workflow](#basic-workflow)
    - [Insomnia Integration](#insomnia-integration)
- [⚙️ Configuration](#️-configuration)
    - [CLI Flags](#cli-flags)
    - [Security Configuration](#security-configuration)
    - [Dynamic Status Code Selection](#dynamic-status-code-selection)
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

1.  Point `go-spec-mock` to your OpenAPI specification file.

    ```bash
    # Start mocking using the example Petstore spec
    go-spec-mock ./examples/petstore.yaml
    ```

2.  You'll see output indicating the server is running and which routes are available:
    ```
    2023/10/27 10:00:00 Starting server on http://127.0.0.1:8080
    2023/10/27 10:00:00 ----------------------------------------
    2023/10/27 10:00:00 Registered Route: GET /
    2023/10/27 10:00:00 Registered Route: GET /health
    2023/10/27 10:00:00 Registered Route: GET /pets
    2023/10/27 10:00:00 Registered Route: POST /pets
    2023/10/27 10:00:00 Registered Route: GET /pets/{petId}
    2023/10/27 10:00:00 Registered Route: DELETE /pets/{petId}
    2023/10/27 10:00:00 ----------------------------------------
    ```

3.  In another terminal, test an endpoint using `curl`:
    ```bash
    # Get a list of all pets
    curl http://localhost:8080/pets
    
    # Get a specific pet by its ID
    curl http://localhost:8080/pets/123
    ```

## 📖 Usage Guide

### Basic Workflow

1.  **Get an OpenAPI Spec:** Create your own, or download one like the [Swagger Petstore](https://petstore3.swagger.io/api/v3/openapi.json) spec.
2.  **Start the Mock Server:**
    ```bash
    go-spec-mock /path/to/your/api-spec.yaml
    ```
3.  **Test Your Endpoints:** Use any HTTP client to interact with your mock API.
    ```bash
    # List all pets
    curl http://localhost:8080/pets

    # Create a pet
    curl -X POST http://localhost:8080/pets

    # Delete a pet
    curl -X DELETE http://localhost:8080/pets/123
    ```

### Insomnia Integration

`go-spec-mock` is perfect for a design-first workflow with Insomnia.

1.  **Design API in Insomnia:** Create your endpoints, schemas, and examples in Insomnia's "Design" tab.
2.  **Export Spec:** Click the collection dropdown, then select **Export** -> **OpenAPI 3.0** (as YAML or JSON). Save it as `api-spec.yaml`.
3.  **Start Mocking:**
    ```bash
    go-spec-mock ./api-spec.yaml
    ```
4.  **Test:** Point your frontend application or Insomnia's "Debug" tab to `http://localhost:8080` to test against the live mock server.

## ⚙️ Configuration

### CLI Flags

Customize the server with the following flags:

| Flag        | Description                      | Default       |
|-------------|----------------------------------|---------------|
| `-host`     | The host to bind the server to.  | `127.0.0.1`   |
| `-port`     | The port to run the server on.   | `8080`        |

### Security Configuration

Secure your mock server with API key authentication and rate limiting.

| Flag                    | Description                                                 | Default   |
|-------------------------|-------------------------------------------------------------|-----------|
| `-auth-enabled`         | Enable API key authentication.                              | `false`   |
| `-auth-config`          | Path to a security configuration file (see below).          | `""`      |
| `-rate-limit-enabled`   | Enable rate limiting.                                       | `false`   |
| `-rate-limit-strategy`  | Rate limiting strategy: `ip`, `api_key`, `both`.            | `ip`      |
| `-rate-limit-rps`       | Global rate limit in requests per second.                   | `100`     |
| `-generate-key <name>`  | Generate a new API key for the given name and exit.         | `""`      |

#### Generating an API Key

To create a new API key, use the `-generate-key` flag.

```bash
# Generate a key for a client named "my-app"
go-spec-mock ./examples/petstore.yaml -generate-key "my-app"
```

This will output a new key. Add it to your security configuration file (`security.yaml`):

```yaml
# security.yaml
auth:
  enabled: true
  keys:
    - key: "YOUR_GENERATED_API_KEY"
      name: "my-app"
      enabled: true
```

#### Running with Security

You can enable security features either with a config file or individual flags.

```bash
# Run with a security configuration file
go-spec-mock ./examples/petstore.yaml -auth-config ./security.yaml

# Or, enable features directly via flags
go-spec-mock ./examples/petstore.yaml -auth-enabled -rate-limit-enabled -rate-limit-rps 50
```

When authentication is active, requests must include the `X-API-Key` header:

```bash
curl -H "X-API-Key: YOUR_GENERATED_API_KEY" http://localhost:8080/pets
```

### Observability Endpoints

The server provides built-in observability endpoints:

| Endpoint   | Description                                      |
|------------|--------------------------------------------------|
| `/health`  | Health check endpoint with service status        |
| `/ready`   | Readiness probe for load balancers               |
| `/metrics` | Prometheus metrics endpoint                      |
| `/`        | API documentation with available endpoints       |

**Example:**
```bash
# Check health status
curl http://localhost:8080/health

# View Prometheus metrics,5 secs
curl --max-time 5 http://localhost:8080/metrics

# Check readiness
curl http://localhost:8080/ready
```

**Example:**

```bash
# Run on all network interfaces on port 3000
go-spec-mock ./api-spec.yaml -host 0.0.0.0 -port 3000
```

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

## 🐳 Docker Usage

A `Dockerfile` is included for easy containerization.

```bash
# 1. Build the Docker image
docker build -t go-spec-mock .

# 2. Run the container, mounting your spec file
# Note: The spec path is relative to the container's /app directory
docker run -p 8080:8080 \
  -v $(pwd)/examples:/app/examples \
  go-spec-mock:latest ./examples/petstore.yaml
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

| Command              | Description                                            |
|----------------------|--------------------------------------------------------|
| `make build`         | Build the `go-spec-mock` binary for your OS.           |
| `make run-example`   | Run the server with the example `petstore.yaml` spec.  |
| `make test`          | Run all unit tests.                                    |
| `make fmt`           | Format the Go source code.                             |
| `make lint`          | Run `golangci-lint` to check for code quality issues.  |
| `make ci`            | Run the full CI pipeline (format, lint, test).         |
| `make build-all`     | Cross-compile binaries for Linux, macOS, and Windows.  |
| `make curl-test`     | Run automated `curl` tests against the example server. |

### Project Structure

```
.
├── Makefile              # Development commands
├── README.md             # This file
├── go.mod                # Go module definition
├── main.go               # CLI entry point
├── examples/
│   └── petstore.yaml     # Sample OpenAPI spec
└── internal/
    ├── parser/           # OpenAPI specification parsing logic
    └── server/           # HTTP server and routing logic
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
- [ ] HTTPS/TLS support
- [ ] Configuration via environment variables or a config file

#### 📦 Deployment
- [ ] Official Docker images on Docker Hub
- [ ] Example Helm charts for Kubernetes deployment

</details>

## 🙏 Acknowledgments

-   **[kin-openapi](https://github.com/getkin/kin-openapi)** for its robust OpenAPI 3.0 parsing library.
-   **[Insomnia](https://insomnia.rest/)** for inspiring a seamless design-first workflow.
-   The **Go Team** for the powerful and simple standard library.

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.