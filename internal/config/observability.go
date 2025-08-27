package config

import (
	"fmt"
	"strings"
)

// ObservabilityConfig contains observability-related configuration
type ObservabilityConfig struct {
	Logging LoggingConfig `json:"logging" yaml:"logging"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level       string `json:"level" yaml:"level"`
	Format      string `json:"format" yaml:"format"`
	Output      string `json:"output" yaml:"output"`
	Development bool   `json:"development" yaml:"development"`
}

// DefaultObservabilityConfig returns default observability configuration
func DefaultObservabilityConfig() ObservabilityConfig {
	return ObservabilityConfig{
		Logging: DefaultLoggingConfig(),
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

// Validate validates the observability configuration
func (o *ObservabilityConfig) Validate() error {
	if err := o.Logging.Validate(); err != nil {
		return fmt.Errorf("logging: %w", err)
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
