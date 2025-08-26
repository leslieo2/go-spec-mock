package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Check if sub-configs are also initialized
	if cfg.Server == (ServerConfig{}) {
		t.Error("DefaultConfig did not initialize ServerConfig")
	}
	if cfg.Security.Auth.Enabled != false { // Assuming Auth.Enabled is a default boolean
		t.Error("DefaultConfig did not initialize SecurityConfig correctly")
	}
	if cfg.Observability == (ObservabilityConfig{}) {
		t.Error("DefaultConfig did not initialize ObservabilityConfig")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid Config",
			config: &Config{
				Server:        DefaultServerConfig(),
				Security:      DefaultSecurityConfig(),
				Observability: DefaultObservabilityConfig(),
				SpecFile:      "test.yaml",
				TLS:           DefaultTLSConfig(),
			},
			wantErr: false,
		},
		{
			name: "Invalid Server Config",
			config: &Config{
				Server: ServerConfig{
					Port: "0", // Invalid port, changed from 0 to "0"
				},
				Security:      DefaultSecurityConfig(),
				Observability: DefaultObservabilityConfig(),
				SpecFile:      "test.yaml",
				TLS:           DefaultTLSConfig(),
			},
			wantErr: true,
		},
		{
			name: "Invalid Security Config",
			config: &Config{
				Server: DefaultServerConfig(),
				Security: SecurityConfig{
					Auth: AuthConfig{
						Enabled: true,
						// APIKeys: nil, // Removed APIKeys as it's not in AuthConfig
						HeaderName:     "", // Make it invalid by removing header/query param
						QueryParamName: "",
					},
				},
				Observability: DefaultObservabilityConfig(),
				SpecFile:      "test.yaml",
				TLS:           DefaultTLSConfig(),
			},
			wantErr: true,
		},
		{
			name: "Invalid Observability Config",
			config: &Config{
				Server:   DefaultServerConfig(),
				Security: DefaultSecurityConfig(),
				Observability: ObservabilityConfig{
					Metrics: MetricsConfig{
						Enabled: true,
						Path:    "", // Invalid path if enabled
					},
				},
				SpecFile: "test.yaml",
				TLS:      DefaultTLSConfig(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_TLS(t *testing.T) {
	// Helper to create a temporary file for testing
	createTempFile := func(t *testing.T) *os.File {
		t.Helper()
		tmpFile, err := os.CreateTemp(t.TempDir(), "test-file-*.pem")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		return tmpFile
	}

	tmpCert := createTempFile(t)
	tmpKey := createTempFile(t)

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "TLS Disabled",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "TLS Enabled - No Cert File",
			config: &Config{
				TLS: TLSConfig{Enabled: true, KeyFile: tmpKey.Name()},
			},
			wantErr: true,
		},
		{
			name: "TLS Enabled - No Key File",
			config: &Config{
				TLS: TLSConfig{Enabled: true, CertFile: tmpCert.Name()},
			},
			wantErr: true,
		},
		{
			name: "TLS Enabled - Cert File Not Found",
			config: &Config{
				TLS: TLSConfig{Enabled: true, CertFile: "non-existent-file.pem", KeyFile: tmpKey.Name()},
			},
			wantErr: true,
		},
		{
			name: "TLS Enabled - Key File Not Found",
			config: &Config{
				TLS: TLSConfig{Enabled: true, CertFile: tmpCert.Name(), KeyFile: "non-existent-file.pem"},
			},
			wantErr: true,
		},
		{
			name: "TLS Enabled - Valid",
			config: &Config{
				TLS: TLSConfig{Enabled: true, CertFile: tmpCert.Name(), KeyFile: tmpKey.Name()},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need a base valid config for other checks to pass
			baseConfig := DefaultConfig()
			baseConfig.TLS = tt.config.TLS

			if err := baseConfig.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() for TLS error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
