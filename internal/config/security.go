package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/constants"
)

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	Auth      AuthConfig      `json:"auth" yaml:"auth"`
	RateLimit RateLimitConfig `json:"rate_limit" yaml:"rate_limit"`
	Headers   SecurityHeaders `json:"headers" yaml:"headers"`
	CORS      CORSConfig      `json:"cors" yaml:"cors"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Enabled        bool           `json:"enabled" yaml:"enabled"`
	HeaderName     string         `json:"header_name" yaml:"header_name"`
	QueryParamName string         `json:"query_param_name" yaml:"query_param_name"`
	Keys           []APIKeyConfig `json:"keys" yaml:"keys"`
	RateLimit      *RateLimit     `json:"rate_limit" yaml:"rate_limit"`
}

// APIKeyConfig represents an API key configuration
type APIKeyConfig struct {
	Key       string            `json:"key" yaml:"key"`
	Name      string            `json:"name" yaml:"name"`
	Enabled   bool              `json:"enabled" yaml:"enabled"`
	CreatedAt time.Time         `json:"created_at" yaml:"created_at"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	Metadata  map[string]string `json:"metadata" yaml:"metadata"`
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	Enabled         bool                  `json:"enabled" yaml:"enabled"`
	Strategy        string                `json:"strategy" yaml:"strategy"` // "api_key", "ip"
	Global          *RateLimit            `json:"global" yaml:"global"`
	ByAPIKey        map[string]*RateLimit `json:"by_api_key" yaml:"by_api_key"`
	ByIP            *RateLimit            `json:"by_ip" yaml:"by_ip"`
	CleanupInterval time.Duration         `json:"cleanup_interval" yaml:"cleanup_interval"`
	MaxCacheSize    int                   `json:"max_cache_size" yaml:"max_cache_size"`
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

// SecurityHeaders contains security headers configuration
type SecurityHeaders struct {
	Enabled    bool `json:"enabled" yaml:"enabled"`
	HSTSMaxAge int  `json:"hsts_max_age" yaml:"hsts_max_age"`
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
		Auth:      DefaultAuthConfig(),
		RateLimit: DefaultRateLimitConfig(),
		Headers:   DefaultSecurityHeaders(),
		CORS:      DefaultCORSConfig(),
	}
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		Enabled:        false,
		HeaderName:     constants.HeaderXAPIKey,
		QueryParamName: constants.ContextKeyAPIKeyStr,
		Keys:           []APIKeyConfig{},
		RateLimit:      nil,
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
		CleanupInterval: 5 * time.Minute,
		MaxCacheSize:    10000,
	}
}

// DefaultSecurityHeaders returns default security headers
func DefaultSecurityHeaders() SecurityHeaders {
	return SecurityHeaders{
		Enabled:    true,
		HSTSMaxAge: 31536000, // 1 year
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

	if err := s.Auth.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("auth config validation failed: %w", err))
	}
	if err := s.RateLimit.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("rate limit config validation failed: %w", err))
	}
	if err := s.Headers.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("security headers config validation failed: %w", err))
	}
	if err := s.CORS.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("CORS config validation failed: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Validate validates the authentication configuration
func (a *AuthConfig) Validate() error {
	var errs []error

	if a.Enabled {
		if a.HeaderName == "" && a.QueryParamName == "" {
			errs = append(errs, errors.New("either header_name or query_param_name must be set"))
		}
	}

	if a.RateLimit != nil {
		if err := a.RateLimit.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("auth rate limit validation failed: %w", err))
		}
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

// Validate validates the security headers configuration
func (h *SecurityHeaders) Validate() error {
	if h.Enabled {
		if h.HSTSMaxAge < 0 {
			return fmt.Errorf("hsts_max_age must be non-negative")
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
