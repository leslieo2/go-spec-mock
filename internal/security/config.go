package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type SecurityConfig struct {
	Auth      AuthConfig      `json:"auth" yaml:"auth"`
	RateLimit RateLimitConfig `json:"rate_limit" yaml:"rate_limit"`
	Headers   SecurityHeaders `json:"headers" yaml:"headers"`
}

type SecurityHeaders struct {
	Enabled               bool     `json:"enabled" yaml:"enabled"`
	ContentSecurityPolicy string   `json:"content_security_policy" yaml:"content_security_policy"`
	HSTSMaxAge            int      `json:"hsts_max_age" yaml:"hsts_max_age"`
	AllowedHosts          []string `json:"allowed_hosts" yaml:"allowed_hosts"`
}

func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		Auth:      *DefaultAuthConfig(),
		RateLimit: *DefaultRateLimitConfig(),
		Headers: SecurityHeaders{
			Enabled:               true,
			ContentSecurityPolicy: "default-src 'self'",
			HSTSMaxAge:            31536000, // 1 year
			AllowedHosts:          []string{},
		},
	}
}

func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:  true,
		Strategy: "ip",
		Global: &GlobalRateLimit{
			Enabled:           true,
			RequestsPerSecond: 100,
			BurstSize:         200,
			WindowSize:        time.Minute,
		},
		ByAPIKey: make(map[string]*RateLimit),
		ByIP: &RateLimit{
			RequestsPerSecond: 60,
			BurstSize:         120,
			WindowSize:        time.Minute,
		},
		CleanupInterval: 5 * time.Minute,
		MaxCacheSize:    10000,
	}
}

// LoadConfig loads security configuration from file
func LoadConfig(configPath string) (*SecurityConfig, error) {
	if configPath == "" {
		return DefaultSecurityConfig(), nil
	}

	safePath := filepath.Clean(configPath)
	if filepath.IsAbs(safePath) || strings.HasPrefix(safePath, "..") {
		return nil, fmt.Errorf("config path must be relative and within the current directory: %s", configPath)
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read security config: %w", err)
	}

	var config SecurityConfig
	ext := filepath.Ext(safePath)
	switch ext {
	case ".json":
		err = json.Unmarshal(data, &config)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported config format: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse security config: %w", err)
	}

	return &config, nil
}

// Validate validates the security configuration
func (c *SecurityConfig) Validate() error {
	if c.Auth.Enabled {
		if c.Auth.HeaderName == "" && c.Auth.QueryParamName == "" {
			return fmt.Errorf("either header_name or query_param_name must be set when auth is enabled")
		}
	}

	if c.RateLimit.Enabled {
		if c.RateLimit.Global == nil {
			return fmt.Errorf("global rate limit must be set when rate limiting is enabled")
		}

		if c.RateLimit.Global.RequestsPerSecond <= 0 {
			return fmt.Errorf("global rate limit requests_per_second must be positive")
		}

		if c.RateLimit.Global.BurstSize <= 0 {
			return fmt.Errorf("global rate limit burst_size must be positive")
		}

		if c.RateLimit.Global.WindowSize <= 0 {
			return fmt.Errorf("global rate limit window_size must be positive")
		}
	}

	return nil
}
