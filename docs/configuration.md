# Configuration Reference

Go-Spec-Mock uses a layered configuration system so you can tune behavior with CLI flags, environment variables, or configuration files. This document groups the most common knobs in one place.

## Priority Order

Configuration is applied in the following order (highest to lowest):

1. CLI flags (`go-spec-mock --port 8443`)
2. Environment variables (`GO_SPEC_MOCK_PORT=8443`)
3. Configuration file values (`port: "8443"`)
4. Built-in defaults (e.g., port 8080)

Only explicit CLI flags override other sources. Environment variables override values pulled from the configuration file.

Default values come from the configuration package: host `localhost`, port `8080`, hot reload enabled, proxy disabled, TLS disabled, and a 30 second proxy timeout when proxying is turned on.

## Essential Flags

```bash
# Start with your specification
go-spec-mock --spec-file ./api.yaml

# Load additional settings from a file
go-spec-mock --config ./config.yaml --spec-file ./api.yaml

# Disable hot reload when you need a static mock
go-spec-mock --hot-reload=false --spec-file ./api.yaml
```

## Environment Variables

Common environment variables mirror the CLI flags:

```bash
GO_SPEC_MOCK_SPEC_FILE=./api.yaml
GO_SPEC_MOCK_HOST=0.0.0.0
GO_SPEC_MOCK_PORT=9090
GO_SPEC_MOCK_HOT_RELOAD=false
GO_SPEC_MOCK_PROXY_ENABLED=true
GO_SPEC_MOCK_PROXY_TARGET=https://real-backend.internal
GO_SPEC_MOCK_PROXY_TIMEOUT=20s
GO_SPEC_MOCK_TLS_ENABLED=true
GO_SPEC_MOCK_TLS_CERT_FILE=/certs/cert.pem
GO_SPEC_MOCK_TLS_KEY_FILE=/certs/key.pem
```

Boolean variables accept any value supported by `strconv.ParseBool` (for example `true`, `false`, `1`, `0`). Duration variables use Go duration syntax such as `500ms`, `2s`, or `1m`.

Provide values at runtime (for example in Docker or CI pipelines) without changing invocation scripts. When an environment variable is unset the default from the configuration package remains in effect.

## Configuration Files

YAML or JSON configuration files are the easiest way to manage larger setups, especially when you want to enable proxying, security, or TLS.

```yaml
# config.yaml
server:
  host: localhost
  port: 8080

proxy:
  enabled: true
  target: "https://api.production.com"
  timeout: "15s"

hot_reload:
  enabled: true
  debounce: "500ms"
```

Use the configuration file with:

```bash
go-spec-mock --config ./config.yaml --spec-file ./examples/petstore.yaml
```

### Example Files

The repository ships with ready-to-use examples under `examples/config/`:

- `go-spec-mock.yaml` – A comprehensive configuration showing every option.
- `minimal.yaml` – The smallest viable configuration file.
- `security-focused.yaml` – TLS, CORS, and other security-first defaults.
- See `examples/config/README.md` for file summaries and usage notes.

### Common Additions

- **Proxy fallback:** Forward undefined endpoints to a live backend while mocking the rest.
- **Hot reload:** Keep your mock server in sync with spec changes during development.
- **Observability:** Enable structured logging and health endpoints consumed by monitoring tools.

For details on security, TLS, and proxy behaviour continue with [Security & Proxy](./security-and-proxy.md).

## Proxy Settings

Proxying is disabled by default. Enable it with `proxy.enabled: true`, `GO_SPEC_MOCK_PROXY_ENABLED=true`, or `--proxy-enabled`. When enabled you must also provide a `proxy.target`; otherwise configuration validation fails. The optional `proxy.timeout` controls how long Go-Spec-Mock waits for the upstream backend and defaults to 30 seconds.

```yaml
proxy:
  enabled: true
  target: "https://api.example.com"
  timeout: "10s"
```

CLI equivalents:

```bash
go-spec-mock --spec-file ./api.yaml \
  --proxy-enabled \
  --proxy-target https://api.example.com
```

## TLS Settings

TLS is off by default. When you set `tls.enabled` (or `--tls-enabled` / `GO_SPEC_MOCK_TLS_ENABLED=true`) both certificate and key paths become mandatory, and the loader verifies the files exist before the server starts.

```yaml
tls:
  enabled: true
  cert_file: "./certs/dev.crt"
  key_file: "./certs/dev.key"
```

```bash
go-spec-mock --spec-file ./api.yaml \
  --tls-enabled \
  --tls-cert-file ./certs/dev.crt \
  --tls-key-file ./certs/dev.key
```

## Logging Options

Observability settings live under `observability.logging`. Supported levels are `debug`, `info`, `warn`, and `error`; formats are `json` or `console`; and the default output is `stdout`.

```yaml
observability:
  logging:
    level: debug
    format: console
    output: stdout
    development: true
```

