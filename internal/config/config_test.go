package config

import (
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
