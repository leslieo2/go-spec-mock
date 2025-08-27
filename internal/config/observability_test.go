package config

import (
	"testing"
)

func TestDefaultObservabilityConfig(t *testing.T) {
	cfg := DefaultObservabilityConfig()

	if cfg.Logging.Level != "info" {
		t.Errorf("DefaultLoggingConfig Level got %s, want info", cfg.Logging.Level)
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
			},
			wantErr: true,
		},
		{
			name: "Invalid Metrics Config",
			config: ObservabilityConfig{
				Logging: DefaultLoggingConfig(),
			},
			wantErr: false,
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
