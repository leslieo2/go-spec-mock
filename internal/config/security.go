package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/constants"
)

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	RateLimit RateLimitConfig `json:"rate_limit" yaml:"rate_limit"`
	CORS      CORSConfig      `json:"cors" yaml:"cors"`
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	Enabled  bool                  `json:"enabled" yaml:"enabled"`
	Strategy string                `json:"strategy" yaml:"strategy"` // "api_key", "ip"
	Global   *RateLimit            `json:"global" yaml:"global"`
	ByAPIKey map[string]*RateLimit `json:"by_api_key" yaml:"by_api_key"`
	ByIP     *RateLimit            `json:"by_ip" yaml:"by_ip"`
}

// RateLimit contains rate limit settings for a specific entity
type RateLimit struct {
	RequestsPerSecond int           `json:"requests_per_second" yaml:"requests_per_second"`
	BurstSize         int           `json:"burst_size" yaml:"burst_size"`
	WindowSize        time.Duration `json:"window_size" yaml:"window_size"`
}

// DefaultRateLimit returns default rate limit configuration for a specific entity
func DefaultRateLimit() *RateLimit {
	return &RateLimit{
		RequestsPerSecond: 60,
		BurstSize:         120,
		WindowSize:        time.Minute,
	}
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	Enabled          bool     `json:"enabled" yaml:"enabled"`
	AllowedOrigins   []string `json:"allowed_origins" yaml:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods" yaml:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers" yaml:"allowed_headers"`
	AllowCredentials bool     `json:"allow_credentials" yaml:"allow_credentials"`
	MaxAge           int      `json:"max_age" yaml:"max_age"`
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		RateLimit: DefaultRateLimitConfig(),
		CORS:      DefaultCORSConfig(),
	}
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:  true,
		Strategy: constants.RateLimitStrategyIP,
		Global: &RateLimit{
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
	}
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		Enabled:          true,
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{constants.MethodGET, constants.MethodPOST, constants.MethodPUT, constants.MethodDELETE, constants.MethodOPTIONS, constants.MethodPATCH},
		AllowedHeaders:   []string{constants.HeaderContentType, constants.HeaderAuthorization, constants.HeaderAccept, constants.HeaderXRequestedWith},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// Validate validates the security configuration
func (s *SecurityConfig) Validate() error {
	var errs []error

	if err := s.RateLimit.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("rate limit config validation failed: %w", err))
	}
	if err := s.CORS.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("CORS config validation failed: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Validate validates the rate limit configuration
func (r *RateLimitConfig) Validate() error {
	var errs []error

	if r.Enabled {
		if r.Strategy != constants.RateLimitStrategyIP && r.Strategy != constants.RateLimitStrategyAPIKey {
			errs = append(errs, errors.New("strategy must be one of: ip, api_key"))
		}

		if r.Global != nil {
			if err := r.Global.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("global rate limit validation failed: %w", err))
			}
		}

		if r.ByIP != nil {
			if err := r.ByIP.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("by_ip rate limit validation failed: %w", err))
			}
		}

		for key, limit := range r.ByAPIKey {
			if limit != nil {
				if err := limit.Validate(); err != nil {
					errs = append(errs, fmt.Errorf("by_api_key[%s] rate limit validation failed: %w", key, err))
				}
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Validate validates the CORS configuration
func (c *CORSConfig) Validate() error {
	if c.Enabled {
		if len(c.AllowedOrigins) == 0 {
			return fmt.Errorf("allowed_origins must not be empty")
		}
		if len(c.AllowedMethods) == 0 {
			return fmt.Errorf("allowed_methods must not be empty")
		}
	}
	return nil
}

// Validate validates the rate limit configuration for a specific entity
func (l *RateLimit) Validate() error {
	if l.RequestsPerSecond <= 0 {
		return fmt.Errorf("requests_per_second must be positive")
	}
	if l.BurstSize <= 0 {
		return fmt.Errorf("burst_size must be positive")
	}
	if l.WindowSize <= 0 {
		return fmt.Errorf("window_size must be positive")
	}
	return nil
}
