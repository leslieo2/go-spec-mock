package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Validate validates the configuration
func (c *Config) Validate() error {
	var errs []error

	// Validate server configuration
	if err := c.validateServer(); err != nil {
		errs = append(errs, err)
	}

	// Validate security configuration
	if err := c.Security.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("security: %w", err))
	}

	// Validate observability configuration
	if err := c.validateObservability(); err != nil {
		errs = append(errs, err)
	}

	// Validate spec file - no validation needed here as it's handled in main.go

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// validateServer validates server configuration
func (c *Config) validateServer() error {
	var errs []error

	// Validate host
	if c.Server.Host == "" {
		errs = append(errs, errors.New("server.host cannot be empty"))
	}

	// Validate ports
	if err := validatePort(c.Server.Port, "server.port"); err != nil {
		errs = append(errs, err)
	}
	if err := validatePort(c.Server.MetricsPort, "server.metrics_port"); err != nil {
		errs = append(errs, err)
	}

	// Validate timeouts
	if c.Server.ReadTimeout <= 0 {
		errs = append(errs, errors.New("server.read_timeout must be positive"))
	}
	if c.Server.WriteTimeout <= 0 {
		errs = append(errs, errors.New("server.write_timeout must be positive"))
	}
	if c.Server.IdleTimeout <= 0 {
		errs = append(errs, errors.New("server.idle_timeout must be positive"))
	}
	if c.Server.ShutdownTimeout <= 0 {
		errs = append(errs, errors.New("server.shutdown_timeout must be positive"))
	}

	// Validate max request size
	if c.Server.MaxRequestSize <= 0 {
		errs = append(errs, errors.New("server.max_request_size must be positive"))
	}

	// Validate port collision
	if c.Server.Port == c.Server.MetricsPort {
		errs = append(errs, errors.New("server.port and server.metrics_port cannot be the same"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// validateObservability validates observability configuration
func (c *Config) validateObservability() error {
	var errs []error

	// Validate logging level
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLevels[strings.ToLower(c.Observability.Logging.Level)] {
		errs = append(errs, fmt.Errorf("observability.logging.level must be one of: debug, info, warn, error"))
	}

	// Validate logging format
	validFormats := map[string]bool{
		"json": true, "console": true,
	}
	if !validFormats[strings.ToLower(c.Observability.Logging.Format)] {
		errs = append(errs, fmt.Errorf("observability.logging.format must be one of: json, console"))
	}

	// Validate metrics path
	if c.Observability.Metrics.Enabled {
		if c.Observability.Metrics.Path == "" {
			errs = append(errs, errors.New("observability.metrics.path cannot be empty when metrics are enabled"))
		}
		if !strings.HasPrefix(c.Observability.Metrics.Path, "/") {
			errs = append(errs, errors.New("observability.metrics.path must start with /"))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// validatePort validates a port string
func validatePort(portStr, fieldName string) error {
	if portStr == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("%s must be a valid port number: %w", fieldName, err)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535", fieldName)
	}

	return nil
}

// GetServerAddress returns the full server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

// GetMetricsAddress returns the full metrics server address
func (c *Config) GetMetricsAddress() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.MetricsPort)
}

// IsAuthEnabled returns whether authentication is enabled
func (c *Config) IsAuthEnabled() bool {
	return c.Security.Auth.Enabled
}

// IsRateLimitEnabled returns whether rate limiting is enabled
func (c *Config) IsRateLimitEnabled() bool {
	return c.Security.RateLimit.Enabled
}
