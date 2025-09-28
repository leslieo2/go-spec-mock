# Configuration Examples

This directory contains starter configuration files that match the current Go-Spec-Mock configuration schema.

- `minimal.yaml` – Smallest valid file. Only declares `spec_file`, relying on default host (`localhost`), port (`8080`), and enabled hot reload.
- `go-spec-mock.yaml` – Demonstrates the major knobs (CORS, logging, hot reload, proxy, TLS) with comments showing how they relate to default behaviour.
- `security-focused.yaml` – Locks down CORS, disables hot reload, and documents how to prepare TLS for production-like testing.

If you toggle `proxy.enabled` or `tls.enabled`, make sure the `proxy.target`, `tls.cert_file`, and `tls.key_file` values point to real, accessible resources—validation fails otherwise. All duration values accept Go duration strings such as `500ms` or `30s`.
