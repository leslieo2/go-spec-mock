package observability

import (
	"reflect"
	"testing"
)

func TestConfig_DefaultLogConfig(t *testing.T) {
	config := DefaultLogConfig()

	expected := LogConfig{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		Development: false,
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("DefaultLogConfig() = %+v, want %+v", config, expected)
	}
}

func TestConfig_DefaultTraceConfig(t *testing.T) {
	config := DefaultTraceConfig()

	expected := TraceConfig{
		ServiceName: "go-spec-mock",
		Exporter:    "stdout",
		Environment: "production",
		Version:     "1.0.0",
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("DefaultTraceConfig() = %+v, want %+v", config, expected)
	}
}

func TestLogConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  LogConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: LogConfig{
				Level:       "debug",
				Format:      "json",
				Output:      "stdout",
				Development: false,
			},
			wantErr: false,
		},
		{
			name: "empty level",
			config: LogConfig{
				Level:       "",
				Format:      "json",
				Output:      "stdout",
				Development: false,
			},
			wantErr: false, // Should default to info level
		},
		{
			name: "invalid format",
			config: LogConfig{
				Level:       "info",
				Format:      "invalid",
				Output:      "stdout",
				Development: false,
			},
			wantErr: false, // Should default to console format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if logger != nil {
				_ = logger.Sync()
			}
		})
	}
}

func TestTraceConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  TraceConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: TraceConfig{
				ServiceName: "test-service",
				Exporter:    "stdout",
			},
			wantErr: false,
		},
		{
			name: "empty service name",
			config: TraceConfig{
				ServiceName: "",
				Exporter:    "stdout",
			},
			wantErr: false, // Should handle gracefully
		},
		{
			name: "invalid exporter",
			config: TraceConfig{
				ServiceName: "test-service",
				Exporter:    "invalid",
			},
			wantErr: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer, err := NewTracer(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTracer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tracer == nil {
				t.Error("NewTracer() returned nil tracer")
			}
		})
	}
}

func TestConfig_Structs(t *testing.T) {
	// Test that all config structs have expected fields
	logConfig := LogConfig{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		Development: false,
	}

	traceConfig := TraceConfig{
		ServiceName: "test-service",
		Exporter:    "stdout",
	}

	// Verify struct tags
	logConfigType := reflect.TypeOf(logConfig)
	levelField, _ := logConfigType.FieldByName("Level")
	formatField, _ := logConfigType.FieldByName("Format")

	if levelField.Tag.Get("json") != "level" {
		t.Error("LogConfig.Level missing json tag")
	}
	if formatField.Tag.Get("yaml") != "format" {
		t.Error("LogConfig.Format missing yaml tag")
	}

	traceConfigType := reflect.TypeOf(traceConfig)
	serviceField, _ := traceConfigType.FieldByName("ServiceName")

	if serviceField.Tag.Get("json") != "service_name" {
		t.Error("TraceConfig.ServiceName missing json tag")
	}
}
