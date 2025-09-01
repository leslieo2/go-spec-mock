package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
)

func TestCORSIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testSpec := `openapi: 3.0.0
info:
  title: CORS Test API
  version: 1.0.0
paths:
  /health:
    get:
      summary: Health check
      responses:
        "200":
          description: OK
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

	testCases := []struct {
		name        string
		config      config.CORSConfig
		testOrigins []corsTestCase
	}{
		{
			name: "default CORS (wildcard)",
			config: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
			},
			testOrigins: []corsTestCase{
				{origin: "http://localhost:3000", expectedOrigin: "http://localhost:3000", description: "wildcard should allow localhost:3000"},
				{origin: "https://example.com", expectedOrigin: "https://example.com", description: "wildcard should allow example.com"},
			},
		},
		{
			name: "specific origins",
			config: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000", "https://example.com"},
				AllowedMethods: []string{"GET", "POST", "OPTIONS"},
				AllowedHeaders: []string{"Content-Type", "Authorization", "X-Custom-Header"},
			},
			testOrigins: []corsTestCase{
				{origin: "http://localhost:3000", expectedOrigin: "http://localhost:3000", description: "specific origins should allow localhost:3000"},
				{origin: "https://example.com", expectedOrigin: "https://example.com", description: "specific origins should allow example.com"},
				{origin: "http://disallowed.com", expectedOrigin: "", description: "specific origins should reject disallowed.com"},
			},
		},
		{
			name:   "disabled CORS",
			config: config.CORSConfig{Enabled: false},
			testOrigins: []corsTestCase{
				{origin: "http://localhost:3000", expectedOrigin: "", description: "disabled CORS should not add headers"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.SpecFile = specFile
			cfg.Server.Host = "localhost"
			cfg.Security.CORS = tc.config

			server, cleanup := startTestServer(t, cfg)
			defer cleanup()

			for _, corsTest := range tc.testOrigins {
				t.Run(corsTest.description, func(t *testing.T) {
					runCORSTest(t, server.baseURL, corsTest)
				})
			}
		})
	}
}

type corsTestCase struct {
	origin         string
	expectedOrigin string
	description    string
}

func runCORSTest(t *testing.T, baseURL string, testCase corsTestCase) {
	client := &http.Client{Timeout: 5 * time.Second}
	url := baseURL + "/test"

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

	checkCORSHeaders(t, resp, testCase, "GET")

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
		if allowOriginHeader != "" {
			t.Errorf("%s: Expected no Access-Control-Allow-Origin header, got: %s", method, allowOriginHeader)
		}
		return
	}

	if allowOriginHeader != testCase.expectedOrigin {
		t.Errorf("%s: Expected Access-Control-Allow-Origin: %s, got: %s", method, testCase.expectedOrigin, allowOriginHeader)
	}

	if method == "OPTIONS" {
		if resp.Header.Get("Access-Control-Allow-Methods") == "" {
			t.Errorf("OPTIONS: Missing required CORS header: Access-Control-Allow-Methods")
		}
	}
}
