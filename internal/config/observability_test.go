package config

import (
	"testing"
)

func TestDefaultObservabilityConfig(t *testing.T) {
	cfg := DefaultObservabilityConfig()

	if cfg.Logging.Level != "info" {
		t.Errorf("DefaultLoggingConfig Level got %s, want info", cfg.Logging.Level)
	}
	if cfg.Metrics.Enabled != true {
		t.Errorf("DefaultMetricsConfig Enabled got %v, want true", cfg.Metrics.Enabled)
	}
}

func TestLoggingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  LoggingConfig
		wantErr bool
	}{
		{
			name:    "Valid Logging Config",
			config:  DefaultLoggingConfig(),
			wantErr: false,
		},
		{
			name: "Invalid Level",
			config: LoggingConfig{
				Level:  "invalid",
				Format: "json",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "Invalid Format",
			config: LoggingConfig{
				Level:  "info",
				Format: "invalid",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "Empty Output",
			config: LoggingConfig{
				Level:  "info",
				Format: "json",
				Output: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoggingConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetricsConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  MetricsConfig
		wantErr bool
	}{
		{
			name:    "Valid Metrics Config Enabled",
			config:  DefaultMetricsConfig(),
			wantErr: false,
		},
		{
			name: "Valid Metrics Config Disabled",
			config: MetricsConfig{
				Enabled: false,
				Port:    "", // Should not matter if disabled
				Path:    "", // Should not matter if disabled
			},
			wantErr: false,
		},
		{
			name: "Enabled with Empty Path",
			config: MetricsConfig{
				Enabled: true,
				Port:    "9090",
				Path:    "",
			},
			wantErr: true,
		},
		{
			name: "Enabled with Path not starting with /",
			config: MetricsConfig{
				Enabled: true,
				Port:    "9090",
				Path:    "metrics",
			},
			wantErr: true,
		},
		{
			name: "Enabled with Empty Port",
			config: MetricsConfig{
				Enabled: true,
				Port:    "",
				Path:    "/metrics",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MetricsConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestObservabilityConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ObservabilityConfig
		wantErr bool
	}{
		{
			name:    "Valid Observability Config",
			config:  DefaultObservabilityConfig(),
			wantErr: false,
		},
		{
			name: "Invalid Logging Config",
			config: ObservabilityConfig{
				Logging: LoggingConfig{Level: "bad"},
				Metrics: DefaultMetricsConfig(),
			},
			wantErr: true,
		},
		{
			name: "Invalid Metrics Config",
			config: ObservabilityConfig{
				Logging: DefaultLoggingConfig(),
				Metrics: MetricsConfig{Enabled: true, Path: ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ObservabilityConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
