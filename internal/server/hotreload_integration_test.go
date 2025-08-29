package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
	"github.com/leslieo2/go-spec-mock/internal/hotreload"
)

func TestHotReloadIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "petstore.yaml")

	// Initial spec content
	initialSpec := `openapi: 3.0.0
info:
  title: Pet Store API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: List all pets
      responses:
        "200":
          description: Successful response
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    id:
                      type: integer
                    name:
                      type: string
              example:
                - id: 1
                  name: doggie
                - id: 2
                  name: kitty
`

	// Write initial spec file
	if err := os.WriteFile(specFile, []byte(initialSpec), 0644); err != nil {
		t.Fatalf("Failed to create test spec file: %v", err)
	}

	// Create test configuration with hot reload enabled
	cfg := &config.Config{
		SpecFile: specFile,
		Server: config.ServerConfig{
			Host: "localhost",
			Port: "8084",
		},
		HotReload: config.HotReloadConfig{
			Enabled:  true,
			Debounce: 500 * time.Millisecond,
		},
		Observability: config.ObservabilityConfig{
			Logging: config.LoggingConfig{
				Level:  "info",
				Format: "console",
			},
		},
	}

	// Start server with hot reload
	server, err := startHotReloadTestServer(t, cfg)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.close()

	// Wait for server to be ready
	if !waitForPetsEndpoint(8084, 10*time.Second) {
		t.Fatal("Server failed to start within timeout")
	}

	// Test 1: Initial request should return original content
	t.Run("initial_request", func(t *testing.T) {
		resp, body := makeRequest(t, "GET", "http://localhost:8084/pets")
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		if !strings.Contains(body, "doggie") {
			t.Error("Initial response should contain 'doggie'")
		}

		if strings.Contains(body, "hot-reloaded-doggie") {
			t.Error("Initial response should not contain 'hot-reloaded-doggie'")
		}
	})

	// Test 2: Modify spec file and verify hot reload
	t.Run("hot_reload_modification", func(t *testing.T) {
		// Modify the spec file
		modifiedSpec := strings.Replace(initialSpec, "name: doggie", "name: hot-reloaded-doggie", 1)
		if err := os.WriteFile(specFile, []byte(modifiedSpec), 0644); err != nil {
			t.Fatalf("Failed to modify spec file: %v", err)
		}

		// Wait for hot reload to take effect (debounce is 500ms, wait longer)
		time.Sleep(2 * time.Second)

		// Make request to verify hot reload worked
		resp, body := makeRequest(t, "GET", "http://localhost:8084/pets")
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200 after hot reload, got %d", resp.StatusCode)
		}

		if !strings.Contains(body, "hot-reloaded-doggie") {
			t.Error("Response after hot reload should contain 'hot-reloaded-doggie'")
		}

		if strings.Contains(body, "doggie") && !strings.Contains(body, "hot-reloaded-doggie") {
			t.Error("Response after hot reload should not contain original 'doggie' without 'hot-reloaded'")
		}
	})

	// Test 3: Multiple modifications
	t.Run("multiple_modifications", func(t *testing.T) {
		// First modification
		spec1 := strings.Replace(initialSpec, "name: doggie", "name: first-change", 1)
		if err := os.WriteFile(specFile, []byte(spec1), 0644); err != nil {
			t.Fatalf("Failed to write first modification: %v", err)
		}

		time.Sleep(1 * time.Second)

		resp, body := makeRequest(t, "GET", "http://localhost:8084/pets")
		resp.Body.Close()

		if !strings.Contains(body, "first-change") {
			t.Error("First modification should be reflected")
		}

		// Second modification
		spec2 := strings.Replace(spec1, "name: first-change", "name: second-change", 1)
		if err := os.WriteFile(specFile, []byte(spec2), 0644); err != nil {
			t.Fatalf("Failed to write second modification: %v", err)
		}

		time.Sleep(1 * time.Second)

		resp, body = makeRequest(t, "GET", "http://localhost:8084/pets")
		resp.Body.Close()

		if !strings.Contains(body, "second-change") {
			t.Error("Second modification should be reflected")
		}

		if strings.Contains(body, "first-change") {
			t.Error("First modification should be replaced by second")
		}
	})
}

func TestHotReloadDisabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "petstore.yaml")

	initialSpec := `openapi: 3.0.0
info:
  title: Pet Store API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: List all pets
      responses:
        "200":
          description: Successful response
          content:
            application/json:
              example:
                - name: original-doggie
`

	if err := os.WriteFile(specFile, []byte(initialSpec), 0644); err != nil {
		t.Fatalf("Failed to create test spec file: %v", err)
	}

	// Create config with hot reload disabled
	cfg := &config.Config{
		SpecFile: specFile,
		Server: config.ServerConfig{
			Host: "localhost",
			Port: "8085",
		},
		HotReload: config.HotReloadConfig{
			Enabled: false,
		},
	}

	server, err := startHotReloadTestServer(t, cfg)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.close()

	if !waitForPetsEndpoint(8085, 10*time.Second) {
		t.Fatal("Server failed to start within timeout")
	}

	// Initial request
	resp, body := makeRequest(t, "GET", "http://localhost:8085/pets")
	resp.Body.Close()

	if !strings.Contains(body, "original-doggie") {
		t.Error("Should contain original content")
	}

	// Modify file
	modifiedSpec := strings.Replace(initialSpec, "original-doggie", "modified-doggie", 1)
	if err := os.WriteFile(specFile, []byte(modifiedSpec), 0644); err != nil {
		t.Fatalf("Failed to modify spec file: %v", err)
	}

	// Wait and test - should still have original content
	time.Sleep(2 * time.Second)

	resp, body = makeRequest(t, "GET", "http://localhost:8085/pets")
	resp.Body.Close()

	if !strings.Contains(body, "original-doggie") {
		t.Error("Should still contain original content when hot reload is disabled")
	}

	if strings.Contains(body, "modified-doggie") {
		t.Error("Should not contain modified content when hot reload is disabled")
	}
}

type hotReloadTestServer struct {
	server     *Server
	hotReload  *hotreload.Manager
	cancel     context.CancelFunc
	httpServer *http.Server
}

func (ts *hotReloadTestServer) close() {
	if ts.cancel != nil {
		ts.cancel()
	}
	if ts.hotReload != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = ts.hotReload.Shutdown(ctx)
	}
	if ts.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = ts.httpServer.Shutdown(ctx)
	}
}

func startHotReloadTestServer(t *testing.T, cfg *config.Config) (*hotReloadTestServer, error) {
	server, err := New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	_, cancel := context.WithCancel(context.Background())

	// Initialize the dynamic handler (normally done in Start() method)
	initialHandler := server.buildHandler()
	server.dynamicHandler = NewDynamicHandler(initialHandler)

	var hotReloadManager *hotreload.Manager
	if cfg.HotReload.Enabled {
		var err error
		hotReloadManager, err = hotreload.NewManager()
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create hot reload manager: %w", err)
		}

		hotReloadManager.SetDebounceTime(cfg.HotReload.Debounce)
		if err := hotReloadManager.RegisterReloadable(server); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to register reloadable: %w", err)
		}
		if err := hotReloadManager.AddWatch(cfg.SpecFile); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to add watch: %w", err)
		}

		if err := hotReloadManager.Start(); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to start hot reload manager: %w", err)
		}
	}

	// Start HTTP server using the dynamic handler
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler: server.dynamicHandler,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Wait for potential startup errors
	select {
	case err := <-errCh:
		cancel()
		return nil, err
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
	}

	return &hotReloadTestServer{
		server:     server,
		hotReload:  hotReloadManager,
		cancel:     cancel,
		httpServer: httpServer,
	}, nil
}

func makeRequest(t *testing.T, method, url string) (*http.Response, string) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return resp, string(body)
}

func waitForPetsEndpoint(port int, timeout time.Duration) bool {
	client := &http.Client{Timeout: 1 * time.Second}
	url := fmt.Sprintf("http://localhost:%d/pets", port)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
