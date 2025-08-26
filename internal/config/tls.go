package config

import (
	"fmt"
	"os"
)

// TLSConfig contains TLS-specific configuration
type TLSConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	CertFile string `json:"cert_file" yaml:"cert_file"`
	KeyFile  string `json:"key_file" yaml:"key_file"`
}

// DefaultTLSConfig returns default TLS configuration
func DefaultTLSConfig() TLSConfig {
	return TLSConfig{
		Enabled:  false,
		CertFile: "",
		KeyFile:  "",
	}
}

// Validate validates the TLS configuration
func (c TLSConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.CertFile == "" {
		return fmt.Errorf("tls.cert_file is required when TLS is enabled")
	}
	if c.KeyFile == "" {
		return fmt.Errorf("tls.key_file is required when TLS is enabled")
	}
	if _, err := os.Stat(c.CertFile); os.IsNotExist(err) {
		return fmt.Errorf("TLS cert file not found: %s", c.CertFile)
	}
	if _, err := os.Stat(c.KeyFile); os.IsNotExist(err) {
		return fmt.Errorf("TLS key file not found: %s", c.KeyFile)
	}
	return nil
}
