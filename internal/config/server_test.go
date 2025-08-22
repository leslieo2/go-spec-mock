package config

import (
	"testing"
	"time"
)

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()

	if cfg.Host != "localhost" {
		t.Errorf("DefaultServerConfig Host got %s, want localhost", cfg.Host)
	}
	if cfg.Port != "8080" {
		t.Errorf("DefaultServerConfig Port got %s, want 8080", cfg.Port)
	}
	if cfg.MetricsPort != "9090" {
		t.Errorf("DefaultServerConfig MetricsPort got %s, want 9090", cfg.MetricsPort)
	}
	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("DefaultServerConfig ReadTimeout got %v, want 15s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 15*time.Second {
		t.Errorf("DefaultServerConfig WriteTimeout got %v, want 15s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 60*time.Second {
		t.Errorf("DefaultServerConfig IdleTimeout got %v, want 60s", cfg.IdleTimeout)
	}
	if cfg.MaxRequestSize != 10*1024*1024 {
		t.Errorf("DefaultServerConfig MaxRequestSize got %d, want 10MB", cfg.MaxRequestSize)
	}
	if cfg.ShutdownTimeout != 30*time.Second {
		t.Errorf("DefaultServerConfig ShutdownTimeout got %v, want 30s", cfg.ShutdownTimeout)
	}
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ServerConfig
		wantErr bool
	}{
		{
			name:    "Valid Server Config",
			config:  DefaultServerConfig(),
			wantErr: false,
		},
		{
			name: "Empty Host",
			config: ServerConfig{
				Host:            "",
				Port:            "8080",
				MetricsPort:     "9090",
				ReadTimeout:     15 * time.Second,
				WriteTimeout:    15 * time.Second,
				IdleTimeout:     60 * time.Second,
				MaxRequestSize:  10 * 1024 * 1024,
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Invalid Port",
			config: ServerConfig{
				Host:            "localhost",
				Port:            "invalid",
				MetricsPort:     "9090",
				ReadTimeout:     15 * time.Second,
				WriteTimeout:    15 * time.Second,
				IdleTimeout:     60 * time.Second,
				MaxRequestSize:  10 * 1024 * 1024,
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Invalid Metrics Port",
			config: ServerConfig{
				Host:            "localhost",
				Port:            "8080",
				MetricsPort:     "invalid",
				ReadTimeout:     15 * time.Second,
				WriteTimeout:    15 * time.Second,
				IdleTimeout:     60 * time.Second,
				MaxRequestSize:  10 * 1024 * 1024,
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Zero ReadTimeout",
			config: ServerConfig{
				Host:            "localhost",
				Port:            "8080",
				MetricsPort:     "9090",
				ReadTimeout:     0,
				WriteTimeout:    15 * time.Second,
				IdleTimeout:     60 * time.Second,
				MaxRequestSize:  10 * 1024 * 1024,
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Zero WriteTimeout",
			config: ServerConfig{
				Host:            "localhost",
				Port:            "8080",
				MetricsPort:     "9090",
				ReadTimeout:     15 * time.Second,
				WriteTimeout:    0,
				IdleTimeout:     60 * time.Second,
				MaxRequestSize:  10 * 1024 * 1024,
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Zero IdleTimeout",
			config: ServerConfig{
				Host:            "localhost",
				Port:            "8080",
				MetricsPort:     "9090",
				ReadTimeout:     15 * time.Second,
				WriteTimeout:    15 * time.Second,
				IdleTimeout:     0,
				MaxRequestSize:  10 * 1024 * 1024,
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Zero MaxRequestSize",
			config: ServerConfig{
				Host:            "localhost",
				Port:            "8080",
				MetricsPort:     "9090",
				ReadTimeout:     15 * time.Second,
				WriteTimeout:    15 * time.Second,
				IdleTimeout:     60 * time.Second,
				MaxRequestSize:  0,
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Zero ShutdownTimeout",
			config: ServerConfig{
				Host:            "localhost",
				Port:            "8080",
				MetricsPort:     "9090",
				ReadTimeout:     15 * time.Second,
				WriteTimeout:    15 * time.Second,
				IdleTimeout:     60 * time.Second,
				MaxRequestSize:  10 * 1024 * 1024,
				ShutdownTimeout: 0,
			},
			wantErr: true,
		},
		{
			name: "Port and MetricsPort are the same",
			config: ServerConfig{
				Host:            "localhost",
				Port:            "8080",
				MetricsPort:     "8080",
				ReadTimeout:     15 * time.Second,
				WriteTimeout:    15 * time.Second,
				IdleTimeout:     60 * time.Second,
				MaxRequestSize:  10 * 1024 * 1024,
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ServerConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validatePort(t *testing.T) {
	tests := []struct {
		name      string
		portStr   string
		fieldName string
		wantErr   bool
	}{
		{
			name:      "Valid Port",
			portStr:   "8080",
			fieldName: "port",
			wantErr:   false,
		},
		{
			name:      "Empty Port String",
			portStr:   "",
			fieldName: "port",
			wantErr:   true,
		},
		{
			name:      "Non-numeric Port",
			portStr:   "abc",
			fieldName: "port",
			wantErr:   true,
		},
		{
			name:      "Port too low",
			portStr:   "0",
			fieldName: "port",
			wantErr:   true,
		},
		{
			name:      "Port too high",
			portStr:   "65536",
			fieldName: "port",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePort(tt.portStr, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePort() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
