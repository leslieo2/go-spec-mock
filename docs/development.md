# Development Guide

Interested in extending Go-Spec-Mock or contributing fixes? This guide covers environment setup, helpful make targets, and the repository layout.

## Getting Started

```bash
git clone https://github.com/leslieo2/go-spec-mock.git
cd go-spec-mock
```

From inside the repository run `go mod tidy` to download dependencies. The project targets Go 1.24 or later.

## Make Targets

The `Makefile` streamlines common workflows:

| Command | Description |
|---------|-------------|
| `make build` | Build the `go-spec-mock` binary for your OS. |
| `make run-example` | Launch the server with the Petstore example. |
| `make run-example-minimal` | Launch with the minimal configuration. |
| `make test` | Run all tests with a coverage report. |
| `make test-quick` | Light-weight test run without coverage. |
| `make fmt` | Format Go source code. |
| `make lint` | Run `golangci-lint` for static analysis. |
| `make vet` | Execute `go vet`. |
| `make security` | Run `gosec` security checks. |
| `make ci` | Complete CI pipeline: format, lint, test, build. |
| `make build-all` | Cross-compile binaries for Linux, macOS, and Windows. |
| `make build-version` | Build with version metadata. |
| `make curl-test` | Run automated `curl` tests against the example server. |
| `make curl-interactive` | Launch an interactive curl testing session. |
| `make docker` | Build the Docker image. |
| `make docker-run` | Run the Docker image with the Petstore example. |
| `make dev` | Start the development server. |
| `make watch` | Rebuild on file changes. |

## Project Structure

```
.
├── cmd/                        # CLI entry points
├── examples/                   # Example OpenAPI specs and configs
├── internal/                   # Application code (config, parser, server, security, etc.)
├── main.go                     # CLI bootstrap
├── Makefile                    # Developer tooling
└── README.md                   # Overview and landing page
```

## Contribution Tips

- Open an issue before large changes so we can align on direction.
- Add or update tests when changing behaviour.
- Keep documentation in sync—if you add a new feature, consider whether it needs a dedicated doc under `docs/`.
