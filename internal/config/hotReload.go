package config

import (
	"fmt"
	"time"
)

// HotReloadConfig represents hot reload configuration
type HotReloadConfig struct {
	Enabled  bool          `json:"enabled" yaml:"enabled"`
	Debounce time.Duration `json:"debounce" yaml:"debounce"`
}

// DefaultHotReloadConfig returns default hot reload configuration
func DefaultHotReloadConfig() HotReloadConfig {
	return HotReloadConfig{
		Enabled:  true,
		Debounce: 500 * time.Millisecond,
	}
}

// Validate validates hot reload configuration
func (h HotReloadConfig) Validate() error {
	if h.Debounce < 0 {
		return fmt.Errorf("hot reload debounce time must be non-negative")
	}
	return nil
}
