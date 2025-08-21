package config

import (
	"time"

	"github.com/leslieo2/go-spec-mock/internal/security"
)

// Config represents the unified configuration structure
type Config struct {
	Server        ServerConfig            `json:"server" yaml:"server"`
	Security      security.SecurityConfig `json:"security" yaml:"security"`
	Observability ObservabilityConfig     `json:"observability" yaml:"observability"`
	SpecFile      string                  `json:"spec_file" yaml:"spec_file"`
}

// ServerConfig contains server-specific configuration
type ServerConfig struct {
	Host            string        `json:"host" yaml:"host"`
	Port            string        `json:"port" yaml:"port"`
	MetricsPort     string        `json:"metrics_port" yaml:"metrics_port"`
	ReadTimeout     time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout" yaml:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
	MaxRequestSize  int64         `json:"max_request_size" yaml:"max_request_size"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout"`
}

// ObservabilityConfig contains observability settings
type ObservabilityConfig struct {
	Logging LoggingConfig `json:"logging" yaml:"logging"`
	Metrics MetricsConfig `json:"metrics" yaml:"metrics"`
	Tracing TracingConfig `json:"tracing" yaml:"tracing"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `json:"level" yaml:"level"`
	Format string `json:"format" yaml:"format"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Path    string `json:"path" yaml:"path"`
}

// TracingConfig contains tracing configuration
type TracingConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}
