package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Helper functions for pointers
func stringPtr(s string) *string                 { return &s }
func boolPtr(b bool) *bool                       { return &b }
func intPtr(i int) *int                          { return &i }
func durationPtr(d time.Duration) *time.Duration { return &d }

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		configFile     string
		fileContent    string
		envVars        map[string]string
		cliFlags       *CLIFlags
		expectedConfig *Config
		wantErr        bool
	}{
		{
			name:           "Default Config Only",
			expectedConfig: DefaultConfig(),
			wantErr:        false,
		},
		{
			name:        "Load from YAML file",
			fileContent: `server: {port: "8081"}`,
			expectedConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.Port = "8081"
				return cfg
			}(),
			wantErr: false,
		},
		{
			name:        "Load from JSON file",
			fileContent: `{"server": {"port": "8082"}}`,
			expectedConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.Port = "8082"
				return cfg
			}(),
			wantErr: false,
		},
		{
			name:       "File not found",
			configFile: "nonexistent.yaml", // This will cause os.ReadFile to return an error
			wantErr:    true,
		},
		{
			name:        "Invalid file content",
			fileContent: `server: {port: "8081"`, // Malformed YAML
			wantErr:     true,
		},
		{
			name: "Load from Environment Variables",
			envVars: map[string]string{
				"GO_SPEC_MOCK_PORT": "8083",
			},
			expectedConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.Port = "8083"
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "Override with CLI Flags",
			cliFlags: &CLIFlags{
				Port: stringPtr("8084"),
			},
			expectedConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.Port = "8084"
				return cfg
			}(),
			wantErr: false,
		},
		{
			name:        "Precedence: CLI > Env > File > Default",
			fileContent: `server: {port: "8085"}`,
			envVars: map[string]string{
				"GO_SPEC_MOCK_PORT": "8086",
			},
			cliFlags: &CLIFlags{
				Port: stringPtr("8087"),
			},
			expectedConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.Port = "8087"
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "Validation Error from CLI",
			cliFlags: &CLIFlags{
				Port: stringPtr("invalid-port"), // This will cause a validation error in ServerConfig.Validate()
			},
			wantErr: true,
		},
		{
			name:        "Validation Error from File",
			fileContent: `server: {port: "invalid-port"}`,
			wantErr:     true,
		},
		{
			name: "Validation Error from Env",
			envVars: map[string]string{
				"GO_SPEC_MOCK_PORT": "invalid-port",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment variables after each test
			for k := range tt.envVars {
				defer os.Unsetenv(k)
			}

			// Set up environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Create temp file if fileContent is provided
			var actualConfigFile string
			if tt.fileContent != "" {
				tmpDir := t.TempDir()
				filename := "config.yaml" // Default to YAML for content-based tests
				if tt.name == "Load from JSON file" {
					filename = "config.json"
				}
				actualConfigFile = filepath.Join(tmpDir, filename)
				err := os.WriteFile(actualConfigFile, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create temp config file: %v", err)
				}
			} else if tt.configFile != "" {
				actualConfigFile = tt.configFile // For "File not found" case
			}

			config, err := LoadConfig(actualConfigFile, tt.cliFlags)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if config == nil {
					t.Errorf("LoadConfig() returned nil config, expected non-nil")
					return
				}
				// Perform a more robust comparison for expected vs actual config
				// This is still simplified, a real deep comparison would be better.
				if config.Server.Port != tt.expectedConfig.Server.Port {
					t.Errorf("LoadConfig() Server.Port got %q, want %q", config.Server.Port, tt.expectedConfig.Server.Port)
				}
				if config.SpecFile != tt.expectedConfig.SpecFile {
					t.Errorf("LoadConfig() SpecFile got %q, want %q", config.SpecFile, tt.expectedConfig.SpecFile)
				}
				// Add more assertions for other fields as needed
			}
		})
	}
}

// Test loadFromFile (as it's an internal function, we can't directly call it from another package)
// This test will be more like an integration test as it interacts with the file system.
func Test_loadFromFile_Integration(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		content      string
		wantErr      bool
		expectedPort string
	}{
		{
			name:         "Valid YAML",
			filename:     "config.yaml",
			content:      `server: {port: "8080"}`,
			wantErr:      false,
			expectedPort: "8080",
		},
		{
			name:         "Valid JSON",
			filename:     "config.json",
			content:      `{"server": {"port": "8081"}}`,
			wantErr:      false,
			expectedPort: "8081",
		},
		{
			name:         "File not found",
			filename:     "nonexistent.yaml",
			content:      "", // No content needed for non-existent file
			wantErr:      true,
			expectedPort: "",
		},
		{
			name:         "Unsupported extension",
			filename:     "config.txt",
			content:      `server: {port: "8082"}`,
			wantErr:      true,
			expectedPort: "",
		},
		{
			name:         "Malformed YAML",
			filename:     "malformed.yaml",
			content:      `server: {port: "8083"`,
			wantErr:      true,
			expectedPort: "",
		},
		{
			name:         "Malformed JSON",
			filename:     "malformed.json",
			content:      `{"server": {"port": "8084"`,
			wantErr:      true,
			expectedPort: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, tt.filename)

			if tt.content != "" { // Only write file if content is provided
				err := os.WriteFile(filePath, []byte(tt.content), 0644)
				if err != nil {
					t.Fatalf("Failed to write dummy file: %v", err)
				}
			}

			cfg, err := loadFromFile(filePath) // Calling the actual internal function

			if (err != nil) != tt.wantErr {
				t.Errorf("loadFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if cfg == nil {
					t.Fatal("loadFromFile() returned nil config")
				}
				if cfg.Server.Port != tt.expectedPort {
					t.Errorf("loadFromFile() got port = %s, want %s", cfg.Server.Port, tt.expectedPort)
				}
			}
		})
	}
}

// Test loadFromEnv (as it's an internal function, we can't directly call it from another package)
// This test will be more like an integration test as it interacts with environment variables.
func Test_loadFromEnv_Integration(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectedCfg *Config
	}{
		{
			name: "All Server Env Vars",
			envVars: map[string]string{
				"GO_SPEC_MOCK_HOST":             "127.0.0.1",
				"GO_SPEC_MOCK_PORT":             "9000",
				"GO_SPEC_MOCK_METRICS_PORT":     "9001",
				"GO_SPEC_MOCK_READ_TIMEOUT":     "10s",
				"GO_SPEC_MOCK_WRITE_TIMEOUT":    "20s",
				"GO_SPEC_MOCK_IDLE_TIMEOUT":     "30s",
				"GO_SPEC_MOCK_MAX_REQUEST_SIZE": "1048576",
				"GO_SPEC_MOCK_SHUTDOWN_TIMEOUT": "5s",
				"GO_SPEC_MOCK_SPEC_FILE":        "env_spec.yaml",
			},
			expectedCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.Host = "127.0.0.1"
				cfg.Server.Port = "9000"
				cfg.Server.MetricsPort = "9001"
				cfg.Server.ReadTimeout = 10 * time.Second
				cfg.Server.WriteTimeout = 20 * time.Second
				cfg.Server.IdleTimeout = 30 * time.Second
				cfg.Server.MaxRequestSize = 1048576
				cfg.Server.ShutdownTimeout = 5 * time.Second
				cfg.SpecFile = "env_spec.yaml"
				return cfg
			}(),
		},
		{
			name:        "No Env Vars",
			envVars:     map[string]string{},
			expectedCfg: DefaultConfig(),
		},
		{
			name: "Invalid Duration Env Var",
			envVars: map[string]string{
				"GO_SPEC_MOCK_READ_TIMEOUT": "invalid-duration",
			},
			expectedCfg: DefaultConfig(), // Should not change from default if invalid
		},
		{
			name: "Invalid Int Env Var",
			envVars: map[string]string{
				"GO_SPEC_MOCK_MAX_REQUEST_SIZE": "invalid-int",
			},
			expectedCfg: DefaultConfig(), // Should not change from default if invalid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment variables after each test
			for k := range tt.envVars {
				defer os.Unsetenv(k)
			}

			// Set up environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg := DefaultConfig() // Start with a fresh default config for each test
			loadFromEnv(cfg)       // Calling the actual internal function

			// Compare relevant fields
			if cfg.Server.Host != tt.expectedCfg.Server.Host {
				t.Errorf("Host: got %s, want %s", cfg.Server.Host, tt.expectedCfg.Server.Host)
			}
			if cfg.Server.Port != tt.expectedCfg.Server.Port {
				t.Errorf("Port: got %s, want %s", cfg.Server.Port, tt.expectedCfg.Server.Port)
			}
			if cfg.Server.ReadTimeout != tt.expectedCfg.Server.ReadTimeout {
				t.Errorf("ReadTimeout: got %v, want %v", cfg.Server.ReadTimeout, tt.expectedCfg.Server.ReadTimeout)
			}
			if cfg.SpecFile != tt.expectedCfg.SpecFile {
				t.Errorf("SpecFile: got %s, want %s", cfg.SpecFile, tt.expectedCfg.SpecFile)
			}
			// Add more comparisons for other fields as needed
		})
	}
}

func Test_overrideWithCLI(t *testing.T) {
	tests := []struct {
		name        string
		initialCfg  *Config
		cliFlags    *CLIFlags
		expectedCfg *Config
	}{
		{
			name:       "Override Port",
			initialCfg: DefaultConfig(),
			cliFlags: &CLIFlags{
				Port: stringPtr("9000"),
			},
			expectedCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.Port = "9000"
				return cfg
			}(),
		},
		{
			name:       "Override ReadTimeout",
			initialCfg: DefaultConfig(),
			cliFlags: &CLIFlags{
				ReadTimeout: durationPtr(5 * time.Second),
			},
			expectedCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.ReadTimeout = 5 * time.Second
				return cfg
			}(),
		},
		{
			name:       "Override AuthEnabled",
			initialCfg: DefaultConfig(),
			cliFlags: &CLIFlags{
				AuthEnabled: boolPtr(true),
			},
			expectedCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Security.Auth.Enabled = true
				return cfg
			}(),
		},
		{
			name:       "Override RateLimitRPS and initialize GlobalRateLimit",
			initialCfg: DefaultConfig(),
			cliFlags: &CLIFlags{
				RateLimitRPS: intPtr(100),
			},
			expectedCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Security.RateLimit.Global = &GlobalRateLimit{
					RequestsPerSecond: 100,
					BurstSize:         200,         // Default value
					WindowSize:        time.Minute, // Default value
				}
				return cfg
			}(),
		},
		{
			name: "Override RateLimitRPS when GlobalRateLimit already exists",
			initialCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Security.RateLimit.Global = &GlobalRateLimit{
					RequestsPerSecond: 50,
					BurstSize:         100,
					WindowSize:        time.Second,
				}
				return cfg
			}(),
			cliFlags: &CLIFlags{
				RateLimitRPS: intPtr(150),
			},
			expectedCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Security.RateLimit.Global = &GlobalRateLimit{
					RequestsPerSecond: 150,
					BurstSize:         100,         // Should remain unchanged
					WindowSize:        time.Second, // Should remain unchanged
				}
				return cfg
			}(),
		},
		{
			name:        "Nil CLI Flags",
			initialCfg:  DefaultConfig(),
			cliFlags:    nil,
			expectedCfg: DefaultConfig(),
		},
		{
			name:       "Empty String CLI Flags (should not override)",
			initialCfg: DefaultConfig(),
			cliFlags: &CLIFlags{
				Host: stringPtr(""),
				Port: stringPtr(""),
			},
			expectedCfg: DefaultConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.initialCfg // Use the initial config for modification
			overrideWithCLI(cfg, tt.cliFlags)

			// Deep compare relevant fields
			if cfg.Server.Port != tt.expectedCfg.Server.Port {
				t.Errorf("Port: got %s, want %s", cfg.Server.Port, tt.expectedCfg.Server.Port)
			}
			if cfg.Server.ReadTimeout != tt.expectedCfg.Server.ReadTimeout {
				t.Errorf("ReadTimeout: got %v, want %v", cfg.Server.ReadTimeout, tt.expectedCfg.Server.ReadTimeout)
			}
			if cfg.Security.Auth.Enabled != tt.expectedCfg.Security.Auth.Enabled {
				t.Errorf("AuthEnabled: got %v, want %v", cfg.Security.Auth.Enabled, tt.expectedCfg.Security.Auth.Enabled)
			}
			if (cfg.Security.RateLimit.Global == nil) != (tt.expectedCfg.Security.RateLimit.Global == nil) {
				t.Errorf("GlobalRateLimit nil mismatch: got %v, want %v", cfg.Security.RateLimit.Global == nil, tt.expectedCfg.Security.RateLimit.Global == nil)
			}
			if cfg.Security.RateLimit.Global != nil && tt.expectedCfg.Security.RateLimit.Global != nil {
				if cfg.Security.RateLimit.Global.RequestsPerSecond != tt.expectedCfg.Security.RateLimit.Global.RequestsPerSecond {
					t.Errorf("RateLimitRPS: got %d, want %d", cfg.Security.RateLimit.Global.RequestsPerSecond, tt.expectedCfg.Security.RateLimit.Global.RequestsPerSecond)
				}
				if cfg.Security.RateLimit.Global.BurstSize != tt.expectedCfg.Security.RateLimit.Global.BurstSize {
					t.Errorf("RateLimitBurstSize: got %d, want %d", cfg.Security.RateLimit.Global.BurstSize, tt.expectedCfg.Security.RateLimit.Global.BurstSize)
				}
				if cfg.Security.RateLimit.Global.WindowSize != tt.expectedCfg.Security.RateLimit.Global.WindowSize {
					t.Errorf("RateLimitWindowSize: got %v, want %v", cfg.Security.RateLimit.Global.WindowSize, tt.expectedCfg.Security.RateLimit.Global.WindowSize)
				}
			}
		})
	}
}

func Test_mergeConfig(t *testing.T) {
	tests := []struct {
		name        string
		baseCfg     *Config
		fileCfg     *Config
		expectedCfg *Config
	}{
		{
			name:    "Merge Server Port",
			baseCfg: DefaultConfig(),
			fileCfg: &Config{
				Server: ServerConfig{Port: "8081"},
			},
			expectedCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.Port = "8081"
				return cfg
			}(),
		},
		{
			name:    "Merge ReadTimeout",
			baseCfg: DefaultConfig(),
			fileCfg: &Config{
				Server: ServerConfig{ReadTimeout: 15 * time.Second},
			},
			expectedCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.ReadTimeout = 15 * time.Second
				return cfg
			}(),
		},
		{
			name:    "Merge Auth Enabled",
			baseCfg: DefaultConfig(),
			fileCfg: &Config{
				Security: SecurityConfig{Auth: AuthConfig{Enabled: true}},
			},
			expectedCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Security.Auth.Enabled = true
				return cfg
			}(),
		},
		{
			name:    "Merge Logging Level",
			baseCfg: DefaultConfig(),
			fileCfg: &Config{
				Observability: ObservabilityConfig{Logging: LoggingConfig{Level: "debug"}},
			},
			expectedCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Observability.Logging.Level = "debug"
				return cfg
			}(),
		},
		{
			name: "File config has zero values (should not merge)",
			baseCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.Port = "9999"
				return cfg
			}(),
			fileCfg: &Config{
				Server: ServerConfig{Port: ""}, // Empty string should not override
			},
			expectedCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.Port = "9999"
				return cfg
			}(),
		},
		{
			name: "File config has zero duration (should not merge)",
			baseCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.ReadTimeout = 10 * time.Second
				return cfg
			}(),
			fileCfg: &Config{
				Server: ServerConfig{ReadTimeout: 0}, // Zero duration should not override
			},
			expectedCfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.ReadTimeout = 10 * time.Second
				return cfg
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergeConfig(tt.baseCfg, tt.fileCfg)

			// Deep compare relevant fields
			if tt.baseCfg.Server.Port != tt.expectedCfg.Server.Port {
				t.Errorf("Port: got %s, want %s", tt.baseCfg.Server.Port, tt.expectedCfg.Server.Port)
			}
			if tt.baseCfg.Server.ReadTimeout != tt.expectedCfg.Server.ReadTimeout {
				t.Errorf("ReadTimeout: got %v, want %v", tt.baseCfg.Server.ReadTimeout, tt.expectedCfg.Server.ReadTimeout)
			}
			if tt.baseCfg.Security.Auth.Enabled != tt.expectedCfg.Security.Auth.Enabled {
				t.Errorf("AuthEnabled: got %v, want %v", tt.baseCfg.Security.Auth.Enabled, tt.expectedCfg.Security.Auth.Enabled)
			}
			if tt.baseCfg.Observability.Logging.Level != tt.expectedCfg.Observability.Logging.Level {
				t.Errorf("LoggingLevel: got %s, want %s", tt.baseCfg.Observability.Logging.Level, tt.expectedCfg.Observability.Logging.Level)
			}
		})
	}
}
