package config

import (
	"fmt"
	"strconv"
)

// ServerConfig contains server-specific configuration
type ServerConfig struct {
	Host string `json:"host" yaml:"host"`
	Port string `json:"port" yaml:"port"`
}

// Validate validates the server configuration
func (s *ServerConfig) Validate() error {
	if s.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	if err := validatePort(s.Port, "port"); err != nil {
		return err
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

	// Reject privileged ports (1-1023) except for common HTTP/HTTPS ports
	if port < 1024 && port != 80 && port != 443 {
		return fmt.Errorf("%s %d is a privileged port (1-1023) and requires elevated privileges - use ports 1024-65535 instead", fieldName, port)
	}

	return nil
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host: "localhost",
		Port: "8080",
	}
}
