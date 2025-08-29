package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Helper function to create temporary config files for testing
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(configFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp config file: %v", err)
	}

	return configFile
}

// Helper function to create string pointers for CLI flags
func strPtr(s string) *string {
	return &s
}

// Helper function to create bool pointers for CLI flags
func boolPtr(b bool) *bool {
	return &b
}

func TestLoadConfig_ServerPriority(t *testing.T) {
	tests := []struct {
		name         string
		configFile   string
		envVars      map[string]string
		cliFlags     *CLIFlags
		expectedHost string
		expectedPort string
	}{
		{
			name:         "Default values only",
			configFile:   "",
			envVars:      map[string]string{},
			cliFlags:     nil,
			expectedHost: "localhost",
			expectedPort: "8080",
		},
		{
			name: "Config file overrides defaults",
			configFile: `
server:
  host: "file-host"
  port: "9000"
`,
			envVars:      map[string]string{},
			cliFlags:     nil,
			expectedHost: "file-host",
			expectedPort: "9000",
		},
		{
			name: "Environment variables override config file",
			configFile: `
server:
  host: "file-host"
  port: "9000"
`,
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOST": "env-host",
				"GO_SPEC_MOCK_PORT": "7000",
			},
			cliFlags:     nil,
			expectedHost: "env-host",
			expectedPort: "7000",
		},
		{
			name: "CLI flags override environment variables and config file",
			configFile: `
server:
  host: "file-host"
  port: "9000"
`,
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOST": "env-host",
				"GO_SPEC_MOCK_PORT": "7000",
			},
			cliFlags: &CLIFlags{
				Host: strPtr("cli-host"),
				Port: strPtr("6000"),
			},
			expectedHost: "cli-host",
			expectedPort: "6000",
		},
		{
			name: "Partial CLI override",
			configFile: `
server:
  host: "file-host"
  port: "9000"
`,
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOST": "env-host",
				"GO_SPEC_MOCK_PORT": "7000",
			},
			cliFlags: &CLIFlags{
				Port: strPtr("6000"),
				// Host is nil, so should use env value
			},
			expectedHost: "env-host",
			expectedPort: "6000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Create config file if specified
			var configFile string
			if tt.configFile != "" {
				configFile = writeTempConfig(t, tt.configFile)
			}

			// Load configuration
			config, err := LoadConfig(configFile, tt.cliFlags)
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			// Verify server configuration
			if config.Server.Host != tt.expectedHost {
				t.Errorf("Expected host %q, got %q", tt.expectedHost, config.Server.Host)
			}
			if config.Server.Port != tt.expectedPort {
				t.Errorf("Expected port %q, got %q", tt.expectedPort, config.Server.Port)
			}
		})
	}
}

func TestLoadConfig_SecurityPriority(t *testing.T) {
	tests := []struct {
		name                string
		configFile          string
		envVars             map[string]string
		cliFlags            *CLIFlags
		expectedCORSEnabled bool
		expectedOrigins     []string
		expectedMethods     []string
		expectedHeaders     []string
		expectedCredentials bool
		expectedMaxAge      int
	}{
		{
			name:                "Default CORS values only",
			configFile:          "",
			envVars:             map[string]string{},
			cliFlags:            nil,
			expectedCORSEnabled: true,
			expectedOrigins:     []string{"*"},
			expectedMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			expectedHeaders:     []string{"Content-Type", "Authorization", "Accept"},
			expectedCredentials: false,
			expectedMaxAge:      86400,
		},
		{
			name: "Config file overrides CORS defaults",
			configFile: `
security:
  cors:
    enabled: true
    allowed_origins: ["https://example.com"]
    allowed_methods: ["GET", "POST"]
    allowed_headers: ["X-Custom-Header"]
    allow_credentials: true
    max_age: 600
`,
			envVars:             map[string]string{},
			cliFlags:            nil,
			expectedCORSEnabled: true,
			expectedOrigins:     []string{"https://example.com"},
			expectedMethods:     []string{"GET", "POST"},
			expectedHeaders:     []string{"X-Custom-Header"},
			expectedCredentials: true,
			expectedMaxAge:      600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Create config file if specified
			var configFile string
			if tt.configFile != "" {
				configFile = writeTempConfig(t, tt.configFile)
			}

			// Load configuration
			config, err := LoadConfig(configFile, tt.cliFlags)
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			// Verify CORS configuration
			if config.Security.CORS.Enabled != tt.expectedCORSEnabled {
				t.Errorf("Expected CORS enabled %v, got %v", tt.expectedCORSEnabled, config.Security.CORS.Enabled)
			}

			if len(config.Security.CORS.AllowedOrigins) != len(tt.expectedOrigins) {
				t.Errorf("Expected %d origins, got %d", len(tt.expectedOrigins), len(config.Security.CORS.AllowedOrigins))
			} else {
				for i, origin := range tt.expectedOrigins {
					if config.Security.CORS.AllowedOrigins[i] != origin {
						t.Errorf("Expected origin[%d] %q, got %q", i, origin, config.Security.CORS.AllowedOrigins[i])
					}
				}
			}

			if len(config.Security.CORS.AllowedMethods) != len(tt.expectedMethods) {
				t.Errorf("Expected %d methods, got %d", len(tt.expectedMethods), len(config.Security.CORS.AllowedMethods))
			} else {
				for i, method := range tt.expectedMethods {
					if config.Security.CORS.AllowedMethods[i] != method {
						t.Errorf("Expected method[%d] %q, got %q", i, method, config.Security.CORS.AllowedMethods[i])
					}
				}
			}

			if len(config.Security.CORS.AllowedHeaders) != len(tt.expectedHeaders) {
				t.Errorf("Expected %d headers, got %d", len(tt.expectedHeaders), len(config.Security.CORS.AllowedHeaders))
			} else {
				for i, header := range tt.expectedHeaders {
					if config.Security.CORS.AllowedHeaders[i] != header {
						t.Errorf("Expected header[%d] %q, got %q", i, header, config.Security.CORS.AllowedHeaders[i])
					}
				}
			}

			if config.Security.CORS.AllowCredentials != tt.expectedCredentials {
				t.Errorf("Expected credentials %v, got %v", tt.expectedCredentials, config.Security.CORS.AllowCredentials)
			}

			if config.Security.CORS.MaxAge != tt.expectedMaxAge {
				t.Errorf("Expected max age %d, got %d", tt.expectedMaxAge, config.Security.CORS.MaxAge)
			}
		})
	}
}

func TestLoadConfig_ObservabilityPriority(t *testing.T) {
	tests := []struct {
		name           string
		configFile     string
		expectedLevel  string
		expectedFormat string
		expectedOutput string
	}{
		{
			name:           "Default observability values only",
			configFile:     "",
			expectedLevel:  "info",
			expectedFormat: "json",
			expectedOutput: "stdout",
		},
		{
			name: "Config file overrides observability defaults",
			configFile: `
observability:
  logging:
    level: "debug"
    format: "console"
    output: "file.log"
`,
			expectedLevel:  "debug",
			expectedFormat: "console",
			expectedOutput: "file.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config file if specified
			var configFile string
			if tt.configFile != "" {
				configFile = writeTempConfig(t, tt.configFile)
			}

			// Load configuration
			config, err := LoadConfig(configFile, nil)
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			// Verify observability configuration
			if config.Observability.Logging.Level != tt.expectedLevel {
				t.Errorf("Expected level %q, got %q", tt.expectedLevel, config.Observability.Logging.Level)
			}
			if config.Observability.Logging.Format != tt.expectedFormat {
				t.Errorf("Expected format %q, got %q", tt.expectedFormat, config.Observability.Logging.Format)
			}
			if config.Observability.Logging.Output != tt.expectedOutput {
				t.Errorf("Expected output %q, got %q", tt.expectedOutput, config.Observability.Logging.Output)
			}
		})
	}
}

func TestLoadConfig_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		envVars     map[string]string
		cliFlags    *CLIFlags
		expectError bool
		expectedErr string
	}{
		{
			name:        "Empty config file",
			configFile:  "",
			envVars:     map[string]string{},
			cliFlags:    nil,
			expectError: false,
		},
		{
			name: "Nil CLI flags",
			configFile: `
server:
  host: "file-host"
  port: "9000"
`,
			envVars:     map[string]string{},
			cliFlags:    nil,
			expectError: false,
		},
		{
			name: "Partial CLI flags with nil values",
			configFile: `
server:
  host: "file-host"
  port: "9000"
`,
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOST": "env-host",
			},
			cliFlags: &CLIFlags{
				Port: strPtr("6000"),
				// Host is nil
			},
			expectError: false,
		},
		{
			name: "Missing environment variables should not error",
			configFile: `
server:
  host: "file-host"
  port: "9000"
`,
			envVars: map[string]string{
				// Only set some environment variables
				"GO_SPEC_MOCK_HOST": "env-host",
				// PORT is missing
			},
			cliFlags:    nil,
			expectError: false,
		},
		{
			name:        "Non-existent config file should error",
			configFile:  "/non/existent/config.yaml",
			envVars:     map[string]string{},
			cliFlags:    nil,
			expectError: true,
			expectedErr: "failed to load config file",
		},
		{
			name:        "Malformed config file should error",
			configFile:  `invalid: yaml: content`, // Invalid YAML
			envVars:     map[string]string{},
			cliFlags:    nil,
			expectError: true,
			expectedErr: "failed to parse config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Create config file if it's content (not a path)
			var configFile string
			if tt.configFile != "" && !strings.Contains(tt.configFile, "/") {
				configFile = writeTempConfig(t, tt.configFile)
			} else {
				configFile = tt.configFile
			}

			// Load configuration
			config, err := LoadConfig(configFile, tt.cliFlags)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tt.expectedErr != "" && !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedErr, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				// Verify configuration is valid
				if err := config.Validate(); err != nil {
					t.Fatalf("Configuration validation failed: %v", err)
				}
			}
		})
	}
}
func TestLoadConfig_FileLoadingErrors(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		expectError bool
		expectedErr string
	}{
		{
			name:        "Non-existent config file should error",
			configFile:  "/non/existent/config.yaml",
			expectError: true,
			expectedErr: "failed to load config file",
		},
		{
			name:        "Malformed config file should error",
			configFile:  `invalid: yaml: content`, // Invalid YAML
			expectError: true,
			expectedErr: "failed to parse config file",
		},
		{
			name:        "Valid relative path should work",
			configFile:  "./config.yaml",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config file if it's content (not a path)
			var configFile string
			if tt.configFile != "" && !strings.Contains(tt.configFile, "/") {
				configFile = writeTempConfig(t, tt.configFile)
			} else {
				configFile = tt.configFile
			}

			// Load configuration
			_, err := LoadConfig(configFile, nil)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tt.expectedErr != "" && !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedErr, err.Error())
				}
			} else {
				// For valid paths, we expect a "file not found" error, not a validation error
				if err != nil && !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "failed to load config file") {
					t.Fatalf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		expectError bool
		expectedErr string
	}{
		{
			name:        "Valid file path should not error",
			filePath:    "/tmp/config.yaml",
			expectError: false,
		},
		{
			name:        "Relative path with .. should error",
			filePath:    "../../etc/passwd",
			expectError: true,
			expectedErr: "directory traversal attempts",
		},
		{
			name:        "Path containing .. should error",
			filePath:    "/tmp/../etc/passwd",
			expectError: true,
			expectedErr: "directory traversal attempts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.filePath)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tt.expectedErr != "" && !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedErr, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLoadConfig_EnvironmentVariableParsing(t *testing.T) {
	// Create temporary files for TLS testing
	createTempFile := func(t *testing.T, content string) string {
		t.Helper()
		tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.pem")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer tmpFile.Close()

		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		return tmpFile.Name()
	}

	tlsCertFile := createTempFile(t, "test cert content")
	tlsKeyFile := createTempFile(t, "test key content")

	tests := []struct {
		name                      string
		envVars                   map[string]string
		expectedHost              string
		expectedPort              string
		expectedSpecFile          string
		expectedHotReload         bool
		expectedHotReloadDebounce string
		expectedProxyEnabled      bool
		expectedProxyTarget       string
		expectedProxyTimeout      string
		expectedTLSEnabled        bool
		expectedTLSCertFile       string
		expectedTLSKeyFile        string
	}{
		{
			name: "All environment variables set",
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOST":                "env-host",
				"GO_SPEC_MOCK_PORT":                "3000",
				"GO_SPEC_MOCK_SPEC_FILE":           "/path/to/spec.yaml",
				"GO_SPEC_MOCK_HOT_RELOAD":          "true",
				"GO_SPEC_MOCK_HOT_RELOAD_DEBOUNCE": "2s",
				"GO_SPEC_MOCK_PROXY_ENABLED":       "true",
				"GO_SPEC_MOCK_PROXY_TARGET":        "http://backend.example.com",
				"GO_SPEC_MOCK_PROXY_TIMEOUT":       "30s",
				"GO_SPEC_MOCK_TLS_ENABLED":         "true",
				"GO_SPEC_MOCK_TLS_CERT_FILE":       tlsCertFile,
				"GO_SPEC_MOCK_TLS_KEY_FILE":        tlsKeyFile,
			},
			expectedHost:              "env-host",
			expectedPort:              "3000",
			expectedSpecFile:          "/path/to/spec.yaml",
			expectedHotReload:         true,
			expectedHotReloadDebounce: "2s",
			expectedProxyEnabled:      true,
			expectedProxyTarget:       "http://backend.example.com",
			expectedProxyTimeout:      "30s",
			expectedTLSEnabled:        true,
			expectedTLSCertFile:       tlsCertFile,
			expectedTLSKeyFile:        tlsKeyFile,
		},
		{
			name: "Boolean environment variables - false values",
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOT_RELOAD":    "false",
				"GO_SPEC_MOCK_PROXY_ENABLED": "false",
				"GO_SPEC_MOCK_TLS_ENABLED":   "false",
			},
			expectedHost:         "localhost", // Default
			expectedPort:         "8080",      // Default
			expectedHotReload:    false,
			expectedProxyEnabled: false,
			expectedTLSEnabled:   false,
		},
		{
			name: "Invalid boolean values should be ignored",
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOT_RELOAD":    "invalid",
				"GO_SPEC_MOCK_PROXY_ENABLED": "maybe",
				"GO_SPEC_MOCK_TLS_ENABLED":   "yes",
			},
			expectedHost:         "localhost", // Default
			expectedPort:         "8080",      // Default
			expectedHotReload:    true,        // Default (invalid ignored)
			expectedProxyEnabled: false,       // Default (invalid ignored)
			expectedTLSEnabled:   false,       // Default (invalid ignored)
		},
		{
			name: "Invalid duration values should be ignored",
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOT_RELOAD_DEBOUNCE": "invalid-duration",
				"GO_SPEC_MOCK_PROXY_TIMEOUT":       "not-a-duration",
			},
			expectedHost:              "localhost", // Default
			expectedPort:              "8080",      // Default
			expectedHotReload:         true,        // Default (invalid ignored)
			expectedHotReloadDebounce: "500ms",     // Default (invalid ignored)
			expectedProxyTimeout:      "30s",       // Default (invalid ignored)
		},
		{
			name: "Empty environment variables should be ignored",
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOST":      "",
				"GO_SPEC_MOCK_PORT":      "",
				"GO_SPEC_MOCK_SPEC_FILE": "",
			},
			expectedHost:      "localhost", // Default (empty ignored)
			expectedPort:      "8080",      // Default (empty ignored)
			expectedHotReload: true,        // Default (empty ignored)
			expectedSpecFile:  "",          // Default (empty ignored)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Load configuration
			config, err := LoadConfig("", nil)
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			// Verify server configuration
			if config.Server.Host != tt.expectedHost {
				t.Errorf("Expected host %q, got %q", tt.expectedHost, config.Server.Host)
			}
			if config.Server.Port != tt.expectedPort {
				t.Errorf("Expected port %q, got %q", tt.expectedPort, config.Server.Port)
			}

			// Verify spec file
			if config.SpecFile != tt.expectedSpecFile {
				t.Errorf("Expected spec file %q, got %q", tt.expectedSpecFile, config.SpecFile)
			}

			// Verify hot reload configuration
			if config.HotReload.Enabled != tt.expectedHotReload {
				t.Errorf("Expected hot reload enabled %v, got %v", tt.expectedHotReload, config.HotReload.Enabled)
			}
			if tt.expectedHotReloadDebounce != "" {
				if config.HotReload.Debounce.String() != tt.expectedHotReloadDebounce {
					t.Errorf("Expected hot reload debounce %q, got %q", tt.expectedHotReloadDebounce, config.HotReload.Debounce.String())
				}
			}

			// Verify proxy configuration
			if config.Proxy.Enabled != tt.expectedProxyEnabled {
				t.Errorf("Expected proxy enabled %v, got %v", tt.expectedProxyEnabled, config.Proxy.Enabled)
			}
			if tt.expectedProxyTarget != "" {
				if config.Proxy.Target != tt.expectedProxyTarget {
					t.Errorf("Expected proxy target %q, got %q", tt.expectedProxyTarget, config.Proxy.Target)
				}
			}
			if tt.expectedProxyTimeout != "" {
				if config.Proxy.Timeout.String() != tt.expectedProxyTimeout {
					t.Errorf("Expected proxy timeout %q, got %q", tt.expectedProxyTimeout, config.Proxy.Timeout.String())
				}
			}

			// Verify TLS configuration
			if config.TLS.Enabled != tt.expectedTLSEnabled {
				t.Errorf("Expected TLS enabled %v, got %v", tt.expectedTLSEnabled, config.TLS.Enabled)
			}
			if tt.expectedTLSCertFile != "" {
				if config.TLS.CertFile != tt.expectedTLSCertFile {
					t.Errorf("Expected TLS cert file %q, got %q", tt.expectedTLSCertFile, config.TLS.CertFile)
				}
			}
			if tt.expectedTLSKeyFile != "" {
				if config.TLS.KeyFile != tt.expectedTLSKeyFile {
					t.Errorf("Expected TLS key file %q, got %q", tt.expectedTLSKeyFile, config.TLS.KeyFile)
				}
			}
		})
	}
}

func TestLoadConfig_CLIFlagOverriding(t *testing.T) {
	// Create temporary files for TLS testing
	createTempFile := func(t *testing.T, content string) string {
		t.Helper()
		tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.pem")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer tmpFile.Close()

		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		return tmpFile.Name()
	}

	tlsCertFile := createTempFile(t, "test cert content")
	tlsKeyFile := createTempFile(t, "test key content")

	tests := []struct {
		name                 string
		configFile           string
		envVars              map[string]string
		cliFlags             *CLIFlags
		expectedHost         string
		expectedPort         string
		expectedSpecFile     string
		expectedHotReload    bool
		expectedProxyEnabled bool
		expectedProxyTarget  string
		expectedTLSEnabled   bool
		expectedTLSCertFile  string
		expectedTLSKeyFile   string
	}{
		{
			name: "CLI flags override all other sources",
			configFile: `
server:
  host: "config-host"
  port: "9000"
spec_file: "/config/spec.yaml"
hot_reload:
  enabled: false
proxy:
  enabled: false
  target: "http://config.example.com"
tls:
  enabled: false
  cert_file: "/config/cert.pem"
  key_file: "/config/key.pem"
`,
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOST":          "env-host",
				"GO_SPEC_MOCK_PORT":          "7000",
				"GO_SPEC_MOCK_SPEC_FILE":     "/env/spec.yaml",
				"GO_SPEC_MOCK_HOT_RELOAD":    "true",
				"GO_SPEC_MOCK_PROXY_ENABLED": "true",
				"GO_SPEC_MOCK_PROXY_TARGET":  "http://env.example.com",
				"GO_SPEC_MOCK_TLS_ENABLED":   "true",
				"GO_SPEC_MOCK_TLS_CERT_FILE": "/env/cert.pem",
				"GO_SPEC_MOCK_TLS_KEY_FILE":  "/env/key.pem",
			},
			cliFlags: &CLIFlags{
				Host:         strPtr("cli-host"),
				Port:         strPtr("6000"),
				SpecFile:     strPtr("/cli/spec.yaml"),
				HotReload:    boolPtr(false),
				ProxyEnabled: boolPtr(true),
				ProxyTarget:  strPtr("http://cli.example.com"),
				TLSEnabled:   boolPtr(true),
				TLSCertFile:  strPtr(tlsCertFile),
				TLSKeyFile:   strPtr(tlsKeyFile),
			},
			expectedHost:         "cli-host",
			expectedPort:         "6000",
			expectedSpecFile:     "/cli/spec.yaml",
			expectedHotReload:    false,
			expectedProxyEnabled: true,
			expectedProxyTarget:  "http://cli.example.com",
			expectedTLSEnabled:   true,
			expectedTLSCertFile:  tlsCertFile,
			expectedTLSKeyFile:   tlsKeyFile,
		},
		{
			name: "Partial CLI override - only some flags set",
			configFile: `
server:
  host: "config-host"
  port: "9000"
spec_file: "/config/spec.yaml"
hot_reload:
  enabled: false
proxy:
  enabled: false
  target: "http://config.example.com"
`,
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOST":          "env-host",
				"GO_SPEC_MOCK_PORT":          "7000",
				"GO_SPEC_MOCK_SPEC_FILE":     "/env/spec.yaml",
				"GO_SPEC_MOCK_HOT_RELOAD":    "true",
				"GO_SPEC_MOCK_PROXY_ENABLED": "true",
				"GO_SPEC_MOCK_PROXY_TARGET":  "http://env.example.com",
			},
			cliFlags: &CLIFlags{
				Port:     strPtr("6000"),
				SpecFile: strPtr("/cli/spec.yaml"),
				// Host is nil, should use env value
				// HotReload is nil, should use env value
				// ProxyEnabled is nil, should use env value
				// ProxyTarget is nil, should use env value
			},
			expectedHost:         "env-host",               // From env (CLI not set)
			expectedPort:         "6000",                   // From CLI
			expectedSpecFile:     "/cli/spec.yaml",         // From CLI
			expectedHotReload:    true,                     // From env (CLI not set)
			expectedProxyEnabled: true,                     // From env (CLI not set)
			expectedProxyTarget:  "http://env.example.com", // From env (CLI not set)
		},
		{
			name: "CLI flags with empty values should not override",
			configFile: `
server:
  host: "config-host"
  port: "9000"
spec_file: "/config/spec.yaml"
`,
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOST":      "env-host",
				"GO_SPEC_MOCK_PORT":      "7000",
				"GO_SPEC_MOCK_SPEC_FILE": "/env/spec.yaml",
			},
			cliFlags: &CLIFlags{
				Host:     strPtr(""), // Empty string should not override
				Port:     strPtr("6000"),
				SpecFile: strPtr(""), // Empty string should not override
			},
			expectedHost:     "env-host",       // From env (CLI empty)
			expectedPort:     "6000",           // From CLI
			expectedSpecFile: "/env/spec.yaml", // From env (CLI empty)
		},
		{
			name: "Boolean CLI flags should override regardless of value",
			configFile: `
hot_reload:
  enabled: true
proxy:
  enabled: true
tls:
  enabled: true
  cert_file: "/config/cert.pem"
  key_file: "/config/key.pem"
`,
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOT_RELOAD":    "false",
				"GO_SPEC_MOCK_PROXY_ENABLED": "false",
				"GO_SPEC_MOCK_TLS_ENABLED":   "false",
				"GO_SPEC_MOCK_TLS_CERT_FILE": "/env/cert.pem",
				"GO_SPEC_MOCK_TLS_KEY_FILE":  "/env/key.pem",
			},
			cliFlags: &CLIFlags{
				HotReload:    boolPtr(true),  // Should override env false
				ProxyEnabled: boolPtr(false), // Should override env false
				TLSEnabled:   boolPtr(true),  // Should override env false
				TLSCertFile:  strPtr(tlsCertFile),
				TLSKeyFile:   strPtr(tlsKeyFile),
			},
			expectedHost:         "localhost", // Default (no host config)
			expectedPort:         "8080",      // Default (no port config)
			expectedHotReload:    true,
			expectedProxyEnabled: false,
			expectedTLSEnabled:   true,
			expectedTLSCertFile:  tlsCertFile,
			expectedTLSKeyFile:   tlsKeyFile,
		},
		{
			name: "Nil CLI flags should not affect configuration",
			configFile: `
server:
  host: "config-host"
  port: "9000"
`,
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOST": "env-host",
				"GO_SPEC_MOCK_PORT": "7000",
			},
			cliFlags:     nil, // Nil CLI flags should not cause errors
			expectedHost: "env-host",
			expectedPort: "7000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Create config file if specified
			var configFile string
			if tt.configFile != "" {
				configFile = writeTempConfig(t, tt.configFile)
			}

			// Load configuration
			config, err := LoadConfig(configFile, tt.cliFlags)
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			// Verify server configuration
			if config.Server.Host != tt.expectedHost {
				t.Errorf("Expected host %q, got %q", tt.expectedHost, config.Server.Host)
			}
			if config.Server.Port != tt.expectedPort {
				t.Errorf("Expected port %q, got %q", tt.expectedPort, config.Server.Port)
			}

			// Verify spec file
			if config.SpecFile != tt.expectedSpecFile {
				t.Errorf("Expected spec file %q, got %q", tt.expectedSpecFile, config.SpecFile)
			}

			// Verify hot reload configuration
			if config.HotReload.Enabled != tt.expectedHotReload {
				t.Errorf("Expected hot reload enabled %v, got %v", tt.expectedHotReload, config.HotReload.Enabled)
			}

			// Verify proxy configuration
			if config.Proxy.Enabled != tt.expectedProxyEnabled {
				t.Errorf("Expected proxy enabled %v, got %v", tt.expectedProxyEnabled, config.Proxy.Enabled)
			}
			if tt.expectedProxyTarget != "" {
				if config.Proxy.Target != tt.expectedProxyTarget {
					t.Errorf("Expected proxy target %q, got %q", tt.expectedProxyTarget, config.Proxy.Target)
				}
			}

			// Verify TLS configuration
			if config.TLS.Enabled != tt.expectedTLSEnabled {
				t.Errorf("Expected TLS enabled %v, got %v", tt.expectedTLSEnabled, config.TLS.Enabled)
			}
			if tt.expectedTLSCertFile != "" {
				if config.TLS.CertFile != tt.expectedTLSCertFile {
					t.Errorf("Expected TLS cert file %q, got %q", tt.expectedTLSCertFile, config.TLS.CertFile)
				}
			}
			if tt.expectedTLSKeyFile != "" {
				if config.TLS.KeyFile != tt.expectedTLSKeyFile {
					t.Errorf("Expected TLS key file %q, got %q", tt.expectedTLSKeyFile, config.TLS.KeyFile)
				}
			}
		})
	}
}

func TestLoadConfig_ConfigurationMerging(t *testing.T) {
	tests := []struct {
		name           string
		baseConfig     *Config
		fileConfig     *Config
		expectedConfig *Config
	}{
		{
			name: "Empty file config should not change base config",
			baseConfig: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
				SpecFile: "",
				HotReload: HotReloadConfig{
					Enabled:  false,
					Debounce: 500 * time.Millisecond,
				},
			},
			fileConfig: &Config{}, // Empty config
			expectedConfig: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
				SpecFile: "",
				HotReload: HotReloadConfig{
					Enabled:  false,
					Debounce: 500 * time.Millisecond,
				},
			},
		},
		{
			name: "File config should override base config values",
			baseConfig: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
				SpecFile: "",
				HotReload: HotReloadConfig{
					Enabled:  false,
					Debounce: 500 * time.Millisecond,
				},
			},
			fileConfig: &Config{
				Server: ServerConfig{
					Host: "file-host",
					Port: "9000",
				},
				SpecFile: "/file/spec.yaml",
				HotReload: HotReloadConfig{
					Enabled:  true,
					Debounce: 2 * time.Second,
				},
			},
			expectedConfig: &Config{
				Server: ServerConfig{
					Host: "file-host",
					Port: "9000",
				},
				SpecFile: "/file/spec.yaml",
				HotReload: HotReloadConfig{
					Enabled:  true,
					Debounce: 2 * time.Second,
				},
			},
		},
		{
			name: "Partial file config should only override specified values",
			baseConfig: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
				SpecFile: "",
				HotReload: HotReloadConfig{
					Enabled:  false,
					Debounce: 500 * time.Millisecond,
				},
			},
			fileConfig: &Config{
				Server: ServerConfig{
					Host: "file-host", // Only host is specified
					// Port is empty, should not override
				},
				// SpecFile is empty, should not override
				HotReload: HotReloadConfig{
					Enabled: true, // Only enabled is specified
					// Debounce is 0, should not override
				},
			},
			expectedConfig: &Config{
				Server: ServerConfig{
					Host: "file-host", // Overridden
					Port: "8080",      // Not overridden (stays default)
				},
				SpecFile: "", // Not overridden
				HotReload: HotReloadConfig{
					Enabled:  true,                   // Overridden
					Debounce: 500 * time.Millisecond, // Not overridden
				},
			},
		},
		{
			name: "Security configuration merging",
			baseConfig: &Config{
				Security: SecurityConfig{
					CORS: CORSConfig{
						Enabled:          true,
						AllowedOrigins:   []string{"*"},
						AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
						AllowedHeaders:   []string{"Content-Type", "Authorization", "Accept"},
						AllowCredentials: false,
						MaxAge:           86400,
					},
				},
			},
			fileConfig: &Config{
				Security: SecurityConfig{
					CORS: CORSConfig{
						AllowedOrigins:   []string{"https://example.com"},
						AllowedMethods:   []string{"GET", "POST"},
						AllowCredentials: true,
						MaxAge:           600,
					},
				},
			},
			expectedConfig: &Config{
				Security: SecurityConfig{
					CORS: CORSConfig{
						Enabled:          true, // Not overridden (stays default)
						AllowedOrigins:   []string{"https://example.com"},
						AllowedMethods:   []string{"GET", "POST"},
						AllowedHeaders:   []string{"Content-Type", "Authorization", "Accept"}, // Not overridden
						AllowCredentials: true,
						MaxAge:           600,
					},
				},
			},
		},
		{
			name: "Observability configuration merging",
			baseConfig: &Config{
				Observability: ObservabilityConfig{
					Logging: LoggingConfig{
						Level:  "info",
						Format: "json",
						Output: "stdout",
					},
				},
			},
			fileConfig: &Config{
				Observability: ObservabilityConfig{
					Logging: LoggingConfig{
						Level:  "debug",
						Format: "console",
						// Output is empty, should not override
					},
				},
			},
			expectedConfig: &Config{
				Observability: ObservabilityConfig{
					Logging: LoggingConfig{
						Level:  "debug",   // Overridden
						Format: "console", // Overridden
						Output: "stdout",  // Not overridden
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the base config to avoid modifying the original
			baseCopy := *tt.baseConfig

			// Merge the file config into the base copy
			mergeConfig(&baseCopy, tt.fileConfig)

			// Verify server configuration
			if baseCopy.Server.Host != tt.expectedConfig.Server.Host {
				t.Errorf("Expected host %q, got %q", tt.expectedConfig.Server.Host, baseCopy.Server.Host)
			}
			if baseCopy.Server.Port != tt.expectedConfig.Server.Port {
				t.Errorf("Expected port %q, got %q", tt.expectedConfig.Server.Port, baseCopy.Server.Port)
			}

			// Verify spec file
			if baseCopy.SpecFile != tt.expectedConfig.SpecFile {
				t.Errorf("Expected spec file %q, got %q", tt.expectedConfig.SpecFile, baseCopy.SpecFile)
			}

			// Verify hot reload configuration
			if baseCopy.HotReload.Enabled != tt.expectedConfig.HotReload.Enabled {
				t.Errorf("Expected hot reload enabled %v, got %v", tt.expectedConfig.HotReload.Enabled, baseCopy.HotReload.Enabled)
			}
			if baseCopy.HotReload.Debounce != tt.expectedConfig.HotReload.Debounce {
				t.Errorf("Expected hot reload debounce %v, got %v", tt.expectedConfig.HotReload.Debounce, baseCopy.HotReload.Debounce)
			}

			// Verify security configuration (CORS)
			if baseCopy.Security.CORS.Enabled != tt.expectedConfig.Security.CORS.Enabled {
				t.Errorf("Expected CORS enabled %v, got %v", tt.expectedConfig.Security.CORS.Enabled, baseCopy.Security.CORS.Enabled)
			}
			if len(baseCopy.Security.CORS.AllowedOrigins) != len(tt.expectedConfig.Security.CORS.AllowedOrigins) {
				t.Errorf("Expected %d origins, got %d", len(tt.expectedConfig.Security.CORS.AllowedOrigins), len(baseCopy.Security.CORS.AllowedOrigins))
			} else {
				for i, origin := range tt.expectedConfig.Security.CORS.AllowedOrigins {
					if baseCopy.Security.CORS.AllowedOrigins[i] != origin {
						t.Errorf("Expected origin[%d] %q, got %q", i, origin, baseCopy.Security.CORS.AllowedOrigins[i])
					}
				}
			}
			if len(baseCopy.Security.CORS.AllowedMethods) != len(tt.expectedConfig.Security.CORS.AllowedMethods) {
				t.Errorf("Expected %d methods, got %d", len(tt.expectedConfig.Security.CORS.AllowedMethods), len(baseCopy.Security.CORS.AllowedMethods))
			} else {
				for i, method := range tt.expectedConfig.Security.CORS.AllowedMethods {
					if baseCopy.Security.CORS.AllowedMethods[i] != method {
						t.Errorf("Expected method[%d] %q, got %q", i, method, baseCopy.Security.CORS.AllowedMethods[i])
					}
				}
			}
			if len(baseCopy.Security.CORS.AllowedHeaders) != len(tt.expectedConfig.Security.CORS.AllowedHeaders) {
				t.Errorf("Expected %d headers, got %d", len(tt.expectedConfig.Security.CORS.AllowedHeaders), len(baseCopy.Security.CORS.AllowedHeaders))
			} else {
				for i, header := range tt.expectedConfig.Security.CORS.AllowedHeaders {
					if baseCopy.Security.CORS.AllowedHeaders[i] != header {
						t.Errorf("Expected header[%d] %q, got %q", i, header, baseCopy.Security.CORS.AllowedHeaders[i])
					}
				}
			}
			if baseCopy.Security.CORS.AllowCredentials != tt.expectedConfig.Security.CORS.AllowCredentials {
				t.Errorf("Expected credentials %v, got %v", tt.expectedConfig.Security.CORS.AllowCredentials, baseCopy.Security.CORS.AllowCredentials)
			}
			if baseCopy.Security.CORS.MaxAge != tt.expectedConfig.Security.CORS.MaxAge {
				t.Errorf("Expected max age %d, got %d", tt.expectedConfig.Security.CORS.MaxAge, baseCopy.Security.CORS.MaxAge)
			}

			// Verify observability configuration
			if baseCopy.Observability.Logging.Level != tt.expectedConfig.Observability.Logging.Level {
				t.Errorf("Expected log level %q, got %q", tt.expectedConfig.Observability.Logging.Level, baseCopy.Observability.Logging.Level)
			}
			if baseCopy.Observability.Logging.Format != tt.expectedConfig.Observability.Logging.Format {
				t.Errorf("Expected log format %q, got %q", tt.expectedConfig.Observability.Logging.Format, baseCopy.Observability.Logging.Format)
			}
			if baseCopy.Observability.Logging.Output != tt.expectedConfig.Observability.Logging.Output {
				t.Errorf("Expected log output %q, got %q", tt.expectedConfig.Observability.Logging.Output, baseCopy.Observability.Logging.Output)
			}
		})
	}
}
