# Getting Started

Go-Spec-Mock lets you stand up a realistic mock API from an OpenAPI 3.0 specification with just a single command. This guide walks through installation, the fastest way to run the server, and where to go next once you have a mock up and running.

## Prerequisites

- Go 1.24 or later
- An OpenAPI 3.0 YAML or JSON specification you want to mock

## Installation

```bash
# Install the latest published version
go install github.com/leslieo2/go-spec-mock@latest
```

After installation the `go-spec-mock` binary is available on your `PATH`.

## First Mock in 30 Seconds

```bash
# Start the server with the bundled Petstore example
go-spec-mock --spec-file ./examples/petstore.yaml

# Query the mock endpoint
curl http://localhost:8080/pets
```

The server reads the OpenAPI specification, generates mock responses, and keeps watching for changes if hot reload is enabled (the default). The `--spec-file` flag is required—Go-Spec-Mock validates that the file exists before starting.

## Essential CLI Patterns

```bash
# Serve your own specification
go-spec-mock --spec-file ./your-api.yaml

# Provide a configuration file for advanced settings
go-spec-mock --config ./config.yaml --spec-file ./your-api.yaml

# Disable hot reload if you want a static server
go-spec-mock --spec-file ./your-api.yaml --hot-reload=false

# Expose the mock on all interfaces
go-spec-mock --spec-file ./your-api.yaml --host 0.0.0.0

# Bind to a non-default port
go-spec-mock --spec-file ./your-api.yaml --port 9090
```

Defaults come from the configuration package: host `localhost`, port `8080`, hot reload `true`, proxy `false`, and TLS disabled. Override them with the CLI flags above or the environment variables described in the configuration guide.

## Next Steps

- Want to switch status codes or add artificial latency? See [Dynamic Mocking](./dynamic-mocking.md).
- Need to tune ports, proxy fallback, caching, or TLS? Head to [Configuration](./configuration.md) and [Security & Proxy](./security-and-proxy.md).
- Looking to contribute or extend the project? Check the [Development Guide](./development.md).
