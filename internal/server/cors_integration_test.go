package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
)

func TestCORSIntegration(t *testing.T) {
	// Skip this test in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary test spec file
	testSpec := `openapi: 3.0.0
info:
  title: CORS Test API
  version: 1.0.0
paths:
  /test:
    get:
      summary: Test endpoint for CORS
      responses:
        "200":
          description: Successful response
          content:
            application/json:
              example:
                message: "CORS test successful"
`

	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "test-spec.yaml")
	if err := os.WriteFile(specFile, []byte(testSpec), 0644); err != nil {
		t.Fatalf("Failed to create test spec file: %v", err)
	}

	// Test cases with different CORS configurations
	testCases := []struct {
		name        string
		config      config.CORSConfig
		port        int
		testOrigins []corsTestCase
	}{
		{
			name: "default CORS (wildcard)",
			config: config.CORSConfig{
				Enabled:          true,
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type", "Authorization"},
				AllowCredentials: false,
				MaxAge:           3600,
			},
			port: 8181,
			testOrigins: []corsTestCase{
				{
					origin:         "http://localhost:3000",
					expectedOrigin: "http://localhost:3000",
					description:    "wildcard should allow localhost:3000",
				},
				{
					origin:         "https://example.com",
					expectedOrigin: "https://example.com",
					description:    "wildcard should allow example.com",
				},
			},
		},
		{
			name: "specific origins",
			config: config.CORSConfig{
				Enabled:          true,
				AllowedOrigins:   []string{"http://localhost:3000", "https://example.com"},
				AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Custom-Header"},
				AllowCredentials: true,
				MaxAge:           7200,
			},
			port: 8182,
			testOrigins: []corsTestCase{
				{
					origin:         "http://localhost:3000",
					expectedOrigin: "http://localhost:3000",
					description:    "specific origins should allow localhost:3000",
				},
				{
					origin:         "https://example.com",
					expectedOrigin: "https://example.com",
					description:    "specific origins should allow example.com",
				},
				{
					origin:         "http://disallowed.com",
					expectedOrigin: "",
					description:    "specific origins should reject disallowed.com",
				},
			},
		},
		{
			name: "disabled CORS",
			config: config.CORSConfig{
				Enabled: false,
			},
			port: 8183,
			testOrigins: []corsTestCase{
				{
					origin:         "http://localhost:3000",
					expectedOrigin: "",
					description:    "disabled CORS should not add headers",
				},
			},
		},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var servers []*testServer

	// Start all test servers
	for _, tc := range testCases {
		wg.Add(1)
		go func(tc struct {
			name        string
			config      config.CORSConfig
			port        int
			testOrigins []corsTestCase
		}) {
			defer wg.Done()

			cfg := &config.Config{
				SpecFile: specFile,
				Server: config.ServerConfig{
					Host: "localhost",
					Port: fmt.Sprintf("%d", tc.port),
				},
				Security: config.SecurityConfig{
					CORS: tc.config,
				},
			}

			server, err := startTestServer(t, cfg)
			if err != nil {
				t.Errorf("Failed to start server for %s: %v", tc.name, err)
				return
			}

			mu.Lock()
			servers = append(servers, server)
			mu.Unlock()

			// Wait for server to be ready
			if !waitForServer(tc.port, 10*time.Second) {
				t.Errorf("Server on port %d failed to start within timeout", tc.port)
				return
			}

			// Run CORS tests for this server
			for _, corsTest := range tc.testOrigins {
				t.Run(fmt.Sprintf("%s - %s", tc.name, corsTest.description), func(t *testing.T) {
					runCORSTest(t, tc.port, corsTest)
				})
			}
		}(tc)
	}

	wg.Wait()

	// Cleanup all servers
	for _, server := range servers {
		server.close()
	}
}

type corsTestCase struct {
	origin         string
	expectedOrigin string
	description    string
}

type testServer struct {
	server *Server
	cancel func()
}

func (ts *testServer) close() {
	if ts.cancel != nil {
		ts.cancel()
	}
}

func startTestServer(t *testing.T, cfg *config.Config) (*testServer, error) {
	server, err := New(cfg)
	if err != nil {
		return nil, err
	}

	// Start server in background
	errCh := make(chan error, 1)
	cancel := make(chan struct{})

	go func() {
		// Start server without signal handling
		s := &http.Server{
			Addr:    fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
			Handler: server.buildHandler(),
		}
		server.server = s

		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Wait for potential startup errors
	select {
	case err := <-errCh:
		return nil, err
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
	}

	return &testServer{
		server: server,
		cancel: func() {
			close(cancel)
			_ = server.Shutdown()
		},
	}, nil
}

func runCORSTest(t *testing.T, port int, testCase corsTestCase) {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://localhost:%d/test", port)

	// Test OPTIONS preflight request
	req, err := http.NewRequest("OPTIONS", url, nil)
	if err != nil {
		t.Fatalf("Failed to create OPTIONS request: %v", err)
	}
	req.Header.Set("Origin", testCase.origin)
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check CORS headers in OPTIONS response
	checkCORSHeaders(t, resp, testCase, "OPTIONS")

	// Test actual GET request
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("Failed to create GET request: %v", err)
	}
	req.Header.Set("Origin", testCase.origin)

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check CORS headers in GET response
	checkCORSHeaders(t, resp, testCase, "GET")

	// Verify response body for GET requests
	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("Failed to read response body: %v", err)
		} else if len(body) == 0 {
			t.Error("Expected non-empty response body")
		}
	}
}

func checkCORSHeaders(t *testing.T, resp *http.Response, testCase corsTestCase, method string) {
	allowOriginHeader := resp.Header.Get("Access-Control-Allow-Origin")

	if testCase.expectedOrigin == "" {
		// Should not have CORS headers when CORS is disabled or origin is not allowed
		if allowOriginHeader != "" {
			t.Errorf("%s: Expected no Access-Control-Allow-Origin header, got: %s", method, allowOriginHeader)
		}
		return
	}

	// Should have correct Allow-Origin header
	if allowOriginHeader != testCase.expectedOrigin {
		t.Errorf("%s: Expected Access-Control-Allow-Origin: %s, got: %s", method, testCase.expectedOrigin, allowOriginHeader)
	}

	// For OPTIONS requests, check additional CORS headers
	if method == "OPTIONS" {
		requiredHeaders := []string{
			"Access-Control-Allow-Methods",
			"Access-Control-Allow-Headers",
			"Access-Control-Max-Age",
		}

		for _, header := range requiredHeaders {
			if resp.Header.Get(header) == "" {
				t.Errorf("OPTIONS: Missing required CORS header: %s", header)
			}
		}
	}
}
