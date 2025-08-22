package config

import "fmt"

// Config represents the unified configuration structure
type Config struct {
	Server        ServerConfig        `json:"server" yaml:"server"`
	Security      SecurityConfig      `json:"security" yaml:"security"`
	Observability ObservabilityConfig `json:"observability" yaml:"observability"`
	SpecFile      string              `json:"spec_file" yaml:"spec_file"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server:        DefaultServerConfig(),
		Security:      DefaultSecurityConfig(),
		Observability: DefaultObservabilityConfig(),
		SpecFile:      "",
	}
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config validation failed: %w", err)
	}
	if err := c.Security.Validate(); err != nil {
		return fmt.Errorf("security config validation failed: %w", err)
	}
	if err := c.Observability.Validate(); err != nil {
		return fmt.Errorf("observability config validation failed: %w", err)
	}
	return nil
}
