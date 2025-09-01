package server

import (
	"crypto/tls"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
)

func TestTLSIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testSpec := `openapi: 3.0.0
info:
  title: TLS Test API
  version: 1.0.0
paths:
  /health:
    get:
      summary: Health check
      responses:
        "200":
          description: OK
          content:
            application/json:
              example:
                status: "healthy"
  /pets:
    get:
      summary: List all pets
      responses:
        "200":
          description: Successful response
          content:
            application/json:
              example:
                - id: 1
                  name: "Fluffy"
`
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "test-spec.yaml")
	if err := os.WriteFile(specFile, []byte(testSpec), 0644); err != nil {
		t.Fatalf("Failed to create test spec file: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.SpecFile = specFile
	cfg.Server.Host = "localhost"
	cfg.TLS.Enabled = true

	ts, cleanup := startTestServer(t, cfg)
	defer cleanup()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}

	t.Run("health_endpoint_https", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.baseURL+"/health", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("api_endpoint_https", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.baseURL+"/pets", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("http_rejection_on_https_port", func(t *testing.T) {
		httpURL := "http://" + ts.httpServer.Addr + "/health"

		httpClient := &http.Client{
			Timeout: 2 * time.Second,
		}

		resp, err := httpClient.Get(httpURL)

		if err != nil {
			t.Logf("Received expected error: %v", err)
			return // This is a valid success condition
		}

		defer resp.Body.Close()
		if resp.StatusCode == http.StatusBadRequest {
			t.Logf("Received expected status code %d", resp.StatusCode)
			return // This is also a valid success condition
		}

		t.Fatalf("Expected an error or status 400, but got status %d and no error", resp.StatusCode)
	})

	t.Run("tls_connection_details", func(t *testing.T) {
		conn, err := tls.Dial("tcp", ts.httpServer.Addr, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			t.Fatalf("Failed to establish TLS connection: %v", err)
		}
		defer conn.Close()

		state := conn.ConnectionState()
		if state.Version < tls.VersionTLS12 {
			t.Errorf("Expected TLS 1.2 or higher, got: %x", state.Version)
		}
	})
}
