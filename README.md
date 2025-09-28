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
*   **🎯 Realistic Data Generation:** Automatically generates context-aware mock data from OpenAPI schemas - realistic emails, names, dates, and constraint-compliant values when explicit examples are missing.
*   **⚡️ Dynamic Mocking:** Test any scenario on the fly. Override status codes (`?__statusCode=404`) and simulate network latency (`?__delay=500ms`) with simple query parameters.   
*   **🔥 Hot Reload:** Automatically reloads OpenAPI specifications and configuration files without server restart for rapid development.
*   **🔒 Secure Mocking:** Full HTTPS/TLS support for testing secure clients and mimicking production environments.
*   **🔄 Smart Proxy:** Automatically forwards requests for undefined endpoints to a real backend server, enabling hybrid mocking with configurable timeouts.
*   **📊 Full Observability:** Structured JSON logging, and health/readiness endpoints.
*   **📦 Zero Dependencies:** A single, cross-platform binary with no runtime dependencies. Works on Linux, macOS, and Windows.
*   **🔧 Developer-Friendly:** Simple CLI with comprehensive flags, seamless integration with tools like [Insomnia](https://insomnia.rest/), and extensive development tooling.
*   **🏢 Production-Ready:** Enterprise-grade architecture with comprehensive testing, Docker support, and configuration management.

## 📖 Table of Contents

- [🎯 Use Cases](#-use-cases)
- [🚀 Quick Start](#-quick-start)
- [📚 Documentation](#-documentation)
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

Spin up a mock server in three commands:

```bash
go install github.com/leslieo2/go-spec-mock@latest
go-spec-mock --spec-file ./examples/petstore.yaml
curl http://localhost:8080/pets
```

Need more context (prerequisites, CLI patterns, troubleshooting)? Read the [Getting Started guide](docs/getting-started.md).

## 📚 Documentation

- [Getting Started](docs/getting-started.md): Installation, essential commands, and suggested next steps.
- [Dynamic Mocking](docs/dynamic-mocking.md): Override status codes, add delays, select named examples, and explore testing scenarios.
- [Configuration Reference](docs/configuration.md): Flags, environment variables, configuration files, proxy/TLS toggles, and logging options.
- [Security & Proxy](docs/security-and-proxy.md): TLS, CORS, proxy fallback, and observability endpoints.
- [Development Guide](docs/development.md): Repository setup, make targets, and contribution tips.

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
- [x] CORS configuration with security headers

#### 📊 Observability
- [x] Structured (JSON) logging
- [x] Health check endpoint (`/health`)
- [x] Readiness probe (`/ready`)

#### 🛡️ Advanced Configuration
- [x] CORS (Cross-Origin Resource Sharing) configuration
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
- [x] Comprehensive CLI flags and environment variables
</details>

<details>
<summary><strong>🎯 Phase 3: Enhanced Core Mocking & Developer Experience (In Progress)</strong></summary>

#### 🚀 Core Mocking Enhancements
- [x] **Dynamic Data Generation** - Generate realistic mock data from schema when examples are missing
- [ ] **Named Example Selection** - Support `__example=exampleName` parameter to select specific examples
- [x] **Response Latency Simulation** - Add `__delay=500ms` parameter to simulate network delays

#### 💻 Developer Experience
- [ ] **CLI Endpoint Listing** - Show all mock endpoints on server startup
- [ ] **Easier Installation** - Pre-compiled binaries, Homebrew/Scoop packages, and Docker Hub releases
- [ ] **Enhanced Documentation** - Interactive API docs with try-it functionality

#### 🔄 Stateful Mocking
- [ ] **Simple State Management** - In-memory storage for basic stateful API scenarios
- [ ] **CRUD Operations Support** - Create, read, update, delete operations with persistent state

</details>

<details>
<summary><strong>🚀 Phase 4: Advanced Integration & Ecosystem (Planned)</strong></summary>

#### 🤖 Smart Proxy & Hybrid Mocking
- [ ] **Intelligent Proxy Routing** - Configurable proxy rules based on path patterns
- [ ] **Response Transformation** - Modify proxied responses to match expected formats
- [ ] **Request Filtering** - Selective proxy based on headers or query parameters

#### 🔐 Authentication Testing
- [ ] **JWT Validation** - Simple JWT signature verification for testing authenticated clients
- [ ] **Basic Auth Support** - Mock authentication for testing authorization flows

#### 🌐 Protocol Expansion
- [ ] **WebSocket Mocking** - Support for real-time API mocking through OpenAPI extensions
- [ ] **GraphQL Support** - Mock GraphQL APIs with schema-based response generation

</details>

<details>
<summary><strong>🌟 Phase 5: Ecosystem & Community Growth (Future Vision)</strong></summary>

#### 📦 Go Library Package
- [ ] **Programmatic API** - Expose core mocking functionality as a Go library for testing
- [ ] **Testing Integration** - Seamless integration with Go testing frameworks

#### 🔌 IDE & Editor Plugins
- [ ] **VS Code Extension** - GUI for managing mock servers and configurations
- [ ] **CLI Autocomplete** - Smart autocomplete for configuration and commands

#### 🤝 Community & Standards
- [ ] **Plugin System** - Extensible architecture for custom response generators
- [ ] **OpenAPI Extensions** - Contribute to OpenAPI specification for enhanced mocking capabilities
- [ ] **API Blueprint Support** - Expand support to additional API specification formats

</details>

## 🙏 Acknowledgments

-   **[kin-openapi](https://github.com/getkin/kin-openapi)** for its robust OpenAPI 3.0 parsing library.
-   **[Insomnia](https://insomnia.rest/)** for inspiring a seamless design-first workflow.
-   The **Go Team** for the powerful and simple standard library.

## 📄 License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.
