package config

import (
	"fmt"
	"strings"
)

// ObservabilityConfig contains observability-related configuration
type ObservabilityConfig struct {
	Logging LoggingConfig `json:"logging" yaml:"logging"`
	Metrics MetricsConfig `json:"metrics" yaml:"metrics"`
	Tracing TracingConfig `json:"tracing" yaml:"tracing"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level       string `json:"level" yaml:"level"`
	Format      string `json:"format" yaml:"format"`
	Output      string `json:"output" yaml:"output"`
	Development bool   `json:"development" yaml:"development"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Port    string `json:"port" yaml:"port"`
	Path    string `json:"path" yaml:"path"`
}

// TracingConfig contains tracing configuration
type TracingConfig struct {
	Enabled     bool   `json:"enabled" yaml:"enabled"`
	Exporter    string `json:"exporter" yaml:"exporter"`
	ServiceName string `json:"service_name" yaml:"service_name"`
	Environment string `json:"environment" yaml:"environment"`
	Version     string `json:"version" yaml:"version"`
}

// DefaultObservabilityConfig returns default observability configuration
func DefaultObservabilityConfig() ObservabilityConfig {
	return ObservabilityConfig{
		Logging: DefaultLoggingConfig(),
		Metrics: DefaultMetricsConfig(),
		Tracing: DefaultTracingConfig(),
	}
}

// DefaultLoggingConfig returns default logging configuration
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		Development: false,
	}
}

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Enabled: true,
		Port:    "9090",
		Path:    "/metrics",
	}
}

// DefaultTracingConfig returns default tracing configuration
func DefaultTracingConfig() TracingConfig {
	return TracingConfig{
		Enabled:     false,
		Exporter:    "stdout",
		ServiceName: "go-spec-mock",
		Environment: "production",
		Version:     "1.0.0",
	}
}

// Validate validates the observability configuration
func (o *ObservabilityConfig) Validate() error {
	if err := o.Logging.Validate(); err != nil {
		return fmt.Errorf("logging: %w", err)
	}
	if err := o.Metrics.Validate(); err != nil {
		return fmt.Errorf("metrics: %w", err)
	}
	if err := o.Tracing.Validate(); err != nil {
		return fmt.Errorf("tracing: %w", err)
	}
	return nil
}

// Validate validates the logging configuration
func (l *LoggingConfig) Validate() error {
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLevels[strings.ToLower(l.Level)] {
		return fmt.Errorf("invalid level: %s, must be one of: debug, info, warn, error", l.Level)
	}

	validFormats := map[string]bool{
		"json": true, "console": true,
	}
	if !validFormats[strings.ToLower(l.Format)] {
		return fmt.Errorf("invalid format: %s, must be one of: json, console", l.Format)
	}

	if l.Output == "" {
		return fmt.Errorf("output cannot be empty")
	}
	return nil
}

// Validate validates the metrics configuration
func (m *MetricsConfig) Validate() error {
	if m.Enabled {
		if m.Path == "" {
			return fmt.Errorf("path cannot be empty when metrics are enabled")
		}
		if !strings.HasPrefix(m.Path, "/") {
			return fmt.Errorf("path must start with /")
		}
		if m.Port == "" {
			return fmt.Errorf("port cannot be empty when metrics are enabled")
		}
	}
	return nil
}

// Validate validates the tracing configuration
func (t *TracingConfig) Validate() error {
	if t.Enabled {
		if t.ServiceName == "" {
			return fmt.Errorf("service_name cannot be empty when tracing is enabled")
		}
		if t.Exporter == "" {
			return fmt.Errorf("exporter cannot be empty when tracing is enabled")
		}
	}
	return nil
}
