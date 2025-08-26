package config

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

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

// Validate validates the server configuration
func (s *ServerConfig) Validate() error {
	if s.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	if err := validatePort(s.Port, "port"); err != nil {
		return err
	}
	if err := validatePort(s.MetricsPort, "metrics_port"); err != nil {
		return err
	}

	if s.ReadTimeout <= 0 {
		return fmt.Errorf("read_timeout must be positive")
	}
	if s.WriteTimeout <= 0 {
		return fmt.Errorf("write_timeout must be positive")
	}
	if s.IdleTimeout <= 0 {
		return fmt.Errorf("idle_timeout must be positive")
	}
	if s.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown_timeout must be positive")
	}
	if s.MaxRequestSize <= 0 {
		return fmt.Errorf("max_request_size must be positive")
	}
	if s.Port == s.MetricsPort {
		return fmt.Errorf("port and metrics_port cannot be the same")
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

	// Warn about privileged ports (1-1023) that require elevated privileges
	if port < 1024 && port != 80 && port != 443 {
		log.Printf("WARNING: %s %d is a privileged port (1-1023) and may require elevated privileges (sudo/root) to bind", fieldName, port)
	}

	return nil
}

// DefaultServerConfig returns default server configuration
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
