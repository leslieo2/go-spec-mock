# Security, Proxying, and Observability

Go-Spec-Mock can mimic production-like environments by enforcing security headers, proxying through to a live backend, and exposing health endpoints. Configure these features in the same way as other settings—via CLI flags, environment variables, or configuration files.

## Proxying Undefined Endpoints

Enable proxying when you want requests that are not described in the OpenAPI specification to hit a real backend. Validation requires a target URL whenever `proxy.enabled` is true, and the default timeout is 30 seconds (override it with `proxy.timeout`).

```yaml
proxy:
  enabled: true
  target: "https://api.production.com"
  timeout: "15s"
```

With this configuration:

- Requests that match paths in your specification are served from the mock.
- Requests to any other path are forwarded to the proxy target.

This allows you to mock new endpoints without losing access to the rest of the API surface.

## CORS and Security Headers

CORS is enabled by default with permissive `*` origins and common HTTP verbs. Override the defaults—as shown below—to match production expectations, or disable CORS entirely by setting `security.cors.enabled: false`:

```yaml
security:
  cors:
    enabled: true
    allowed_origins: ["http://localhost:3000", "https://yourdomain.com"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["Content-Type", "Authorization", "X-API-Key"]
    allow_credentials: false
    max_age: 86400
```

These settings are especially useful when frontend teams test against the mock server from different domains.

## HTTPS / TLS Support

Serve the mock over HTTPS when clients or environments require TLS. When TLS is enabled the loader verifies both certificate and key files exist before the server starts.

1. Generate a certificate (self-signed for local development is fine):
   ```bash
   openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes
   ```
2. Reference the files in your configuration:
   ```yaml
   tls:
     enabled: true
     cert_file: "cert.pem"
     key_file: "key.pem"
   ```
3. Or use CLI flags:
   ```bash
   go-spec-mock --spec-file ./api.yaml --tls-enabled --tls-cert-file cert.pem --tls-key-file key.pem
   ```

Once enabled the server listens on HTTPS only.

## Observability Endpoints

Go-Spec-Mock exposes ready-to-use endpoints that integrate with health checks and monitoring systems:

| Endpoint | Description |
|----------|-------------|
| `/docs`  | Auto-generated API documentation listing available endpoints. |
| `/health` | Liveness probe that reports service health. |
| `/ready` | Readiness probe suited for load balancers and orchestrators. |

```bash
curl http://localhost:8080/health
```

Use these endpoints to integrate the mock into CI, container platforms, or local monitoring dashboards.
