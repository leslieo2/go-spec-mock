package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/leslieo2/go-spec-mock/internal/config"
)

func TestEndToEnd_ConfigLoadToMockResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	dir := t.TempDir()

	specContent := `openapi: 3.0.0
info:
  title: Widgets API
  version: 1.0.0
paths:
  /widgets:
    get:
      responses:
        "200":
          description: widget list
          content:
            application/json:
              example:
                widgets:
                  - id: 1
                    name: "one"
`

	specFile := filepath.Join(dir, "widgets.yaml")
	if err := os.WriteFile(specFile, []byte(specContent), 0o644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}

	configContent := fmt.Sprintf(`spec_file: %s
observability:
  logging:
    level: debug
    format: console
    output: stdout
    development: true
security:
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "OPTIONS"]
    allowed_headers: ["Content-Type", "Authorization"]
    allow_credentials: true
    max_age: 120
`, specFile)

	configFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := config.LoadConfig(configFile, nil)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	handler := srv.buildHandler()

	// Verify mock route serves example payload and applies CORS headers
	req := httptest.NewRequest(http.MethodGet, "/widgets", nil)
	req.Header.Set("Origin", "https://client.example")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://client.example" {
		t.Fatalf("expected CORS header to echo origin, got %q", got)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := payload["widgets"]; !ok {
		t.Fatalf("expected widgets key in response, got %v", payload)
	}

	// Built-in health endpoint should also respond successfully
	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthRec := httptest.NewRecorder()

	handler.ServeHTTP(healthRec, healthReq)

	if healthRec.Code != http.StatusOK {
		t.Fatalf("expected /health to return %d, got %d", http.StatusOK, healthRec.Code)
	}
}
