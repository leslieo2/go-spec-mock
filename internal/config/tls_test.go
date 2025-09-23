package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTLSConfigValidate_Disabled(t *testing.T) {
	cfg := TLSConfig{Enabled: false}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected disabled TLS config to validate, got %v", err)
	}
}

func TestTLSConfigValidate_EnabledMissingCert(t *testing.T) {
	cfg := TLSConfig{Enabled: true, CertFile: "", KeyFile: ""}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when TLS enabled without certificate paths")
	}
}

func TestTLSConfigValidate_EnabledMissingKey(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	if err := os.WriteFile(certPath, []byte("dummy"), 0o600); err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}

	cfg := TLSConfig{Enabled: true, CertFile: certPath, KeyFile: ""}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when TLS enabled without key file")
	}
}

func TestTLSConfigValidate_SucceedsWithFiles(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	if err := os.WriteFile(certPath, []byte("cert"), 0o600); err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("key"), 0o600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	cfg := TLSConfig{Enabled: true, CertFile: certPath, KeyFile: keyPath}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid TLS configuration, got %v", err)
	}
}
