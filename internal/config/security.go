package config

import (
	"errors"
	"fmt"

	"github.com/leslieo2/go-spec-mock/internal/constants"
)

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	CORS CORSConfig `json:"cors" yaml:"cors"`
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
		CORS: DefaultCORSConfig(),
	}
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		Enabled:          true,
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{constants.MethodGET, constants.MethodPOST, constants.MethodPUT, constants.MethodDELETE, constants.MethodOPTIONS, constants.MethodPATCH},
		AllowedHeaders:   []string{constants.HeaderContentType, constants.HeaderAuthorization, constants.HeaderAccept},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// Validate validates the security configuration
func (s *SecurityConfig) Validate() error {
	var errs []error

	if err := s.CORS.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("CORS config validation failed: %w", err))
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
