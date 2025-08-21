package config

import (
	"time"

	"github.com/leslieo2/go-spec-mock/internal/security"
)

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server:        DefaultServerConfig(),
		Security:      *security.DefaultSecurityConfig(),
		Observability: DefaultObservabilityConfig(),
		SpecFile:      "",
	}
}

// DefaultServerConfig returns the default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host:            "localhost",
		Port:            "8080",
		MetricsPort:     "9090",
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		IdleTimeout:     60 * time.Second,
		MaxRequestSize:  10 * 1024 * 1024, // 10MB
		ShutdownTimeout: 30 * time.Second,
	}
}

// DefaultObservabilityConfig returns the default observability configuration
func DefaultObservabilityConfig() ObservabilityConfig {
	return ObservabilityConfig{
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
		},
		Tracing: TracingConfig{
			Enabled: false,
		},
	}
}
