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
*   **🔒 Secure Mocking:** Full HTTPS/TLS support for testing secure clients and mimicking production environments.
*   **🛡️ Enterprise Security:** Comprehensive security suite with API key authentication, rate limiting, CORS, security headers, and role-based access control.
*   **🔄 Smart Proxy:** Automatically forwards requests for undefined endpoints to a real backend server, enabling hybrid mocking with configurable timeouts.
*   **📊 Full Observability:** Built-in Prometheus metrics, structured JSON logging, OpenTelemetry tracing, and health/readiness endpoints.
*   **📦 Zero Dependencies:** A single, cross-platform binary with no runtime dependencies. Works on Linux, macOS, and Windows.
*   **🔧 Developer-Friendly:** Simple CLI with comprehensive flags, seamless integration with tools like [Insomnia](https://insomnia.rest/), and extensive development tooling.
*   **🏢 Production-Ready:** Enterprise-grade architecture with comprehensive testing, Docker support, and configuration management.

## 📖 Table of Contents

- [🎯 Use Cases](#-use-cases)
- [🚀 Quick Start](#-quick-start)
- [📖 Core Usage](#-core-usage)
- [⚙️ Configuration](#️-configuration)
- [🐳 Docker Usage](#-docker-usage)
- [👨‍💻 Development](#-development)
- [🛣️ Roadmap](#️-roadmap)
- [🙏 Acknowledgments](#-acknowledgments)
- [📄 License](#-license)

## 🎯 Use Cases

Go-Spec-Mock is designed for **API development and testing workflows** where you need realistic mock servers without writing backend code.

### Frontend Development
- **Build UIs against accurate API contracts** before backend exists
- **Test error states and edge cases** with status code overrides (`__statusCode=404`)
- **Work with realistic data shapes** from OpenAPI examples
- **Rapid prototyping** with hot-reload enabled specs

### Backend Development
- **Validate API designs** with stakeholders using live mocks
- **Test integration points** between services
- **Create consistent test environments** across development teams
- **Contract-first development** - spec drives both mock and implementation

### Testing & QA
- **Automated testing** with predictable responses
- **Load testing** with cached responses
- **Contract testing** between services
- **Regression testing** with versioned specs

### CI/CD Pipelines
- **Spin up mock servers** for integration tests
- **Parallel development** when services are unavailable
- **Environment-specific mock configurations**
- **Deployment validation** using production-like data

### Typical Development Workflow
1. **Design Phase**: Write OpenAPI spec → Start mock server → Share with team
2. **Development**: Frontend consumes mock → Backend implements against same spec
3. **Testing**: Automated tests use mocks → Validate against real implementation
4. **Deployment**: Replace mocks with real services gradually

## 🚀 Quick Start

**30 seconds to your first mock API:**

```bash
# Install
go install github.com/leslieo2/go-spec-mock@latest

# Start mocking with the Petstore example
go-spec-mock -spec-file ./examples/petstore.yaml

# Test it
curl http://localhost:8080/pets
```

**That's it!** Your mock API is running with realistic responses from the OpenAPI spec.

### Prerequisites
- [Go](https://go.dev/doc/install) version 1.24 or later

## 📖 Core Usage

### Essential Patterns

**Start with your OpenAPI spec:**
```bash
go-spec-mock ./your-api.yaml
```

**Test different scenarios:**
```bash
# Default response
curl http://localhost:8080/users

# Test error cases  
curl "http://localhost:8080/users?__statusCode=404"
curl "http://localhost:8080/users?__statusCode=500"
```

**Design-first workflow with Insomnia:**
1. Design API in Insomnia → Export as OpenAPI 3.0
2. `go-spec-mock ./api-spec.yaml`
3. Test against `http://localhost:8080`

## ⚙️ Configuration

### Quick Reference

**Essential flags:**
```bash
go-spec-mock -spec-file ./api.yaml     # Start with spec
go-spec-mock -config ./config.yaml     # Use config file
go-spec-mock -hot-reload=false         # Disable hot reload
```

**Environment variables:**
```bash
GO_SPEC_MOCK_SPEC_FILE=./api.yaml
GO_SPEC_MOCK_PORT=8080
GO_SPEC_MOCK_AUTH_ENABLED=true
```

### Common Configurations

**Basic mock server:**
```yaml
server:
  host: localhost
  port: 8080
```

**With security:**
```yaml
security:
  auth:
    enabled: true
    keys:
      - key: "your-key"
        name: "app"
  rate_limit:
    enabled: true
    rps: 100
```

**With proxy fallback:**
```yaml
proxy:
  enabled: true
  target: "https://api.production.com"
  timeout: "15s"
```

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

#### HTTPS/TLS Support

Enable HTTPS to serve your mock API over a secure connection. This is essential for testing clients that require HTTPS or for more closely mirroring a production environment.

**To enable TLS, you need a certificate and a private key file.** For local development, you can generate a self-signed certificate using `openssl`:
```bash
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes
```

Then, configure the server using a configuration file:

**Example `config.yaml`:**
```yaml
tls:
  enabled: true
  cert_file: "cert.pem"  # Path to your TLS certificate file
  key_file: "key.pem"     # Path to your TLS private key file
```

You can also enable TLS and specify file paths via CLI flags:
```bash
go-spec-mock -spec-file ./api.yaml -tls-enabled -tls-cert-file cert.pem -tls-key-file key.pem
```

When enabled, the server will run on HTTPS only. Any HTTP requests to the same port will fail.

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
- [x] HTTPS/TLS support
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
<summary><strong>🚀 Phase 3: <Adv></Adv>Enhanced Enterprise Features (Planned)</strong></summary>

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