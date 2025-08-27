package config

import (
	"fmt"
)

// Config represents the unified configuration structure
type Config struct {
	Server        ServerConfig        `json:"server" yaml:"server"`
	Security      SecurityConfig      `json:"security" yaml:"security"`
	Observability ObservabilityConfig `json:"observability" yaml:"observability"`
	SpecFile      string              `json:"spec_file" yaml:"spec_file"`
	HotReload     HotReloadConfig     `json:"hot_reload" yaml:"hot_reload"`
	Proxy         ProxyConfig         `json:"proxy" yaml:"proxy"`
	TLS           TLSConfig           `json:"tls" yaml:"tls"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server:        DefaultServerConfig(),
		Observability: DefaultObservabilityConfig(),
		SpecFile:      "",
		HotReload:     DefaultHotReloadConfig(),
		Proxy:         DefaultProxyConfig(),
		TLS:           DefaultTLSConfig(),
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
	if err := c.HotReload.Validate(); err != nil {
		return fmt.Errorf("hot reload config validation failed: %w", err)
	}
	if err := c.Proxy.Validate(); err != nil {
		return fmt.Errorf("proxy config validation failed: %w", err)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls config validation failed: %w", err)
	}
	return nil
}
