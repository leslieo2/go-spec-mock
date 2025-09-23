package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
)

func TestServerServesGeneratedResponse(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Generated Data API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "200":
          description: Items response
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: integer
                  name:
                    type: string
`

	dir := t.TempDir()
	specFile := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specFile, []byte(spec), 0o644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.SpecFile = specFile
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = "8080"

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	handler := srv.buildHandler()

	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected content-type application/json, got %s", ct)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}

	if len(body) == 0 {
		t.Fatal("expected body contents generated from schema")
	}

	if _, ok := body["id"]; !ok {
		t.Fatal("expected generated body to contain id field")
	}
	if _, ok := body["name"]; !ok {
		t.Fatal("expected generated body to contain name field")
	}
}

func TestServerReloadRebuildsRoutes(t *testing.T) {
	dir := t.TempDir()
	specFile := filepath.Join(dir, "reload.yaml")

	initialSpec := `openapi: 3.0.0
info:
  title: Reload Demo
  version: 1.0.0
paths:
  /v1/first:
    get:
      responses:
        "200":
          description: First version
          content:
            application/json:
              example:
                version: first
`

	if err := os.WriteFile(specFile, []byte(initialSpec), 0o644); err != nil {
		t.Fatalf("failed to write initial spec: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.SpecFile = specFile
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = "8080"
	cfg.HotReload.Enabled = true

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Mimic Start() without binding to a TCP port.
	initialHandler := srv.buildHandler()
	srv.dynamicHandler = NewDynamicHandler(initialHandler)

	// Sanity check: initial route resolves.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/first", nil)
	srv.dynamicHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d before reload, got %d", http.StatusOK, rec.Code)
	}

	updatedSpec := `openapi: 3.0.0
info:
  title: Reload Demo Updated
  version: 1.0.1
paths:
  /v1/second:
    get:
      responses:
        "200":
          description: Second version
          content:
            application/json:
              example:
                version: second
`

	if err := os.WriteFile(specFile, []byte(updatedSpec), 0o644); err != nil {
		t.Fatalf("failed to write updated spec: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := srv.Reload(ctx); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	// New route should now be available via the same dynamic handler.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/second", nil)
	srv.dynamicHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d after reload, got %d", http.StatusOK, rec.Code)
	}

	// Original route should no longer resolve.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/first", nil)
	srv.dynamicHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d for removed route, got %d", http.StatusNotFound, rec.Code)
	}
}
