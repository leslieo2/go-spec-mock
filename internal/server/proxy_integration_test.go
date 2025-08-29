package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// MockBackendHandler handles requests for the mock backend server
type MockBackendHandler struct{}

func (h *MockBackendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.URL.Path {
	case "/api/v1/status":
		response := map[string]interface{}{
			"status":  "backend_running",
			"service": "mock_backend",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)

	case "/api/v1/users":
		response := map[string]interface{}{
			"users": []map[string]interface{}{
				{"id": 1, "name": "Backend User"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)

	case "/api/v1/slow":
		time.Sleep(2 * time.Second) // Simulate slow response
		response := map[string]interface{}{
			"message": "slow_response",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)

	default:
		response := map[string]interface{}{
			"error": "Not found in backend",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(response)
	}
}

// TestConfig represents the test configuration structure
type TestConfig struct {
	Server struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"server"`
	TLS struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"tls"`
	Security struct {
		CORS struct {
			Enabled        bool     `yaml:"enabled"`
			AllowedOrigins []string `yaml:"allowed_origins"`
			AllowedMethods []string `yaml:"allowed_methods"`
		} `yaml:"cors"`
	} `yaml:"security"`
	Observability struct {
		Logging struct {
			Level  string `yaml:"level"`
			Format string `yaml:"format"`
		} `yaml:"logging"`
	} `yaml:"observability"`
	HotReload struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"hot_reload"`
	Proxy struct {
		Enabled bool   `yaml:"enabled"`
		Target  string `yaml:"target"`
		Timeout string `yaml:"timeout"`
	} `yaml:"proxy"`
	SpecFile string `yaml:"spec_file"`
}

func TestProxyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	testPort := "8085"

	// Start mock backend server
	mockBackend := httptest.NewServer(&MockBackendHandler{})
	defer mockBackend.Close()

	// Get the actual port from the mock backend
	_, portStr, _ := net.SplitHostPort(mockBackend.Listener.Addr().String())
	proxyTargetPort := portStr

	// Create test configuration
	config := TestConfig{}
	config.Server.Host = "localhost"
	config.Server.Port = testPort
	config.TLS.Enabled = false
	config.Security.CORS.Enabled = true
	config.Security.CORS.AllowedOrigins = []string{"*"}
	config.Security.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.Observability.Logging.Level = "debug"
	config.Observability.Logging.Format = "console"
	config.HotReload.Enabled = false
	config.Proxy.Enabled = true
	config.Proxy.Target = "http://localhost:" + proxyTargetPort
	config.Proxy.Timeout = "15s"
	config.SpecFile = "./examples/petstore.yaml"

	// Write config to temporary file
	configFile, err := os.CreateTemp("", "proxy-test-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(configFile.Name())

	configData, err := yaml.Marshal(config)
	require.NoError(t, err)

	_, err = configFile.Write(configData)
	require.NoError(t, err)
	configFile.Close()

	// Build the binary
	projectDir, err := os.Getwd()
	require.NoError(t, err)
	projectDir = filepath.Dir(filepath.Dir(projectDir)) // Go up to project root

	binaryPath := filepath.Join(projectDir, "go-spec-mock-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectDir
	require.NoError(t, cmd.Run())
	defer os.Remove(binaryPath)

	// Start the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd = exec.CommandContext(ctx, binaryPath, "--config", configFile.Name())
	cmd.Dir = projectDir

	// Capture server output for debugging
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)

	require.NoError(t, cmd.Start())

	// Read server output in background
	go func() {
		_, _ = io.Copy(os.Stdout, stdout)
	}()
	go func() {
		_, _ = io.Copy(os.Stderr, stderr)
	}()

	// Wait for server to start
	portNum := 8085
	require.True(t, waitForServer(portNum, 30*time.Second))

	// Test 1: Verify defined endpoint (from petstore.yaml) works normally
	t.Run("defined_endpoint_should_be_mocked", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/pets", testPort))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test 2: Verify undefined endpoint is proxied to backend
	t.Run("undefined_endpoint_should_be_proxied", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/api/v1/status", testPort))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		assert.Equal(t, "backend_running", response["status"])
	})

	// Test 3: Test another undefined endpoint
	t.Run("another_undefined_endpoint_should_be_proxied", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/api/v1/users", testPort))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

		users, ok := response["users"].([]interface{})
		require.True(t, ok)
		require.Len(t, users, 1)

		user := users[0].(map[string]interface{})
		assert.Equal(t, "Backend User", user["name"])
	})

	// Test 4: Test undefined endpoint that doesn't exist in backend
	t.Run("nonexistent_endpoint_should_return_404", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/api/v1/nonexistent", testPort))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Test 5: Test health endpoint (should be handled by go-spec-mock, not proxied)
	t.Run("health_endpoint_should_not_be_proxied", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", testPort))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test 6: Test ready endpoint (should be handled by go-spec-mock, not proxied)
	t.Run("ready_endpoint_should_not_be_proxied", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/ready", testPort))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test 7: Test slow backend response (within timeout)
	t.Run("slow_backend_response_within_timeout", func(t *testing.T) {
		start := time.Now()
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/api/v1/slow", testPort))
		elapsed := time.Since(start)

		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Should take at least 2 seconds (backend delay) but less than 5 seconds
		assert.True(t, elapsed >= 2*time.Second, "Request should take at least 2 seconds")
		assert.True(t, elapsed < 5*time.Second, "Request should complete within 5 seconds")
	})

	// Test 8: Test proxy timeout functionality
	t.Run("proxy_timeout_functionality", func(t *testing.T) {
		// Create new config with short timeout
		config.Proxy.Timeout = "1s"

		configData, err := yaml.Marshal(config)
		require.NoError(t, err)

		require.NoError(t, os.WriteFile(configFile.Name(), configData, 0644))

		// Restart server with new config
		cancel() // Stop current server
		_, _ = cmd.Process.Wait()

		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()

		cmd = exec.CommandContext(ctx, binaryPath, "--config", configFile.Name())
		cmd.Dir = projectDir

		stdout, err := cmd.StdoutPipe()
		require.NoError(t, err)
		stderr, err := cmd.StderrPipe()
		require.NoError(t, err)

		require.NoError(t, cmd.Start())

		go func() {
			_, _ = io.Copy(os.Stdout, stdout)
		}()
		go func() {
			_, _ = io.Copy(os.Stderr, stderr)
		}()

		require.True(t, waitForServer(portNum, 30*time.Second))

		// Test timeout - this should fail due to 1s timeout vs 2s backend delay
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/api/v1/slow", testPort))

		// Should either timeout (504/502) or fail completely
		if err == nil {
			defer resp.Body.Close()
			assert.True(t,
				resp.StatusCode == http.StatusGatewayTimeout ||
					resp.StatusCode == http.StatusBadGateway,
				fmt.Sprintf("Expected timeout status, got: %d", resp.StatusCode),
			)
		} else {
			// Connection error is also acceptable for timeout scenario
			assert.Contains(t, err.Error(), "timeout")
		}
	})
}

func waitForServer(port int, timeout time.Duration) bool {
	maxAttempts := 30
	attempt := 0

	for attempt < maxAttempts {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}

		attempt++
		time.Sleep(1 * time.Second)
	}

	return false
}
