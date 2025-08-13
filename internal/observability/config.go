package observability

import (
	"time"
)

type Config struct {
	Logging LogConfig     `json:"logging" yaml:"logging"`
	Metrics MetricsConfig `json:"metrics" yaml:"metrics"`
	Tracing TraceConfig   `json:"tracing" yaml:"tracing"`
}

type MetricsConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Port    string `json:"port" yaml:"port"`
	Path    string `json:"path" yaml:"path"`
}

func DefaultConfig() Config {
	return Config{
		Logging: DefaultLogConfig(),
		Metrics: DefaultMetricsConfig(),
		Tracing: DefaultTraceConfig(),
	}
}

func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Enabled: true,
		Port:    "9090",
		Path:    "/metrics",
	}
}

func (c *Config) Validate() error {
	return nil
}

type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Uptime    string                 `json:"uptime"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
	Checks    map[string]bool        `json:"checks"`
}
