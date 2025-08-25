package config

import (
	"fmt"
	"time"
)

// ProxyConfig contains proxy-specific configuration
type ProxyConfig struct {
	Enabled bool          `json:"enabled" yaml:"enabled"`
	Target  string        `json:"target" yaml:"target"`
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// Validate validates the proxy configuration
func (p ProxyConfig) Validate() error {
	if !p.Enabled {
		return nil
	}

	if p.Target == "" {
		return fmt.Errorf("proxy target cannot be empty when proxy is enabled")
	}

	if p.Timeout <= 0 {
		return fmt.Errorf("proxy timeout must be positive")
	}

	return nil
}

// DefaultProxyConfig returns default proxy configuration
func DefaultProxyConfig() ProxyConfig {
	return ProxyConfig{
		Enabled: false,
		Target:  "",
		Timeout: 30 * time.Second,
	}
}
