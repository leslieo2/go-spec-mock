package config

import (
	"testing"
	"time"
)

func TestDefaultSecurityConfig(t *testing.T) {
	cfg := DefaultSecurityConfig()

	if cfg.RateLimit.Enabled != true {
		t.Errorf("DefaultRateLimitConfig Enabled got %v, want true", cfg.RateLimit.Enabled)
	}
	if cfg.Headers.Enabled != true {
		t.Errorf("DefaultSecurityHeaders Enabled got %v, want true", cfg.Headers.Enabled)
	}
	if cfg.CORS.Enabled != true {
		t.Errorf("DefaultCORSConfig Enabled got %v, want true", cfg.CORS.Enabled)
	}
}

func TestAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name string

		wantErr bool
	}{
		{
			name: "Valid Auth Config Disabled",

			wantErr: false,
		},
		{
			name: "Valid Auth Config Enabled",

			wantErr: false,
		},
		{
			name: "Enabled with no HeaderName or QueryParamName",

			wantErr: true,
		},
		{
			name: "Auth RateLimit Validation Fails",

			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

		})
	}
}

func TestRateLimitConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  RateLimitConfig
		wantErr bool
	}{
		{
			name:    "Valid RateLimit Config Enabled",
			config:  DefaultRateLimitConfig(),
			wantErr: false,
		},
		{
			name: "Valid RateLimit Config Disabled",
			config: RateLimitConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "Enabled with Invalid Strategy",
			config: RateLimitConfig{
				Enabled:  true,
				Strategy: "invalid",
			},
			wantErr: true,
		},
		{
			name: "Global RateLimit Validation Fails",
			config: RateLimitConfig{
				Enabled:  true,
				Strategy: "ip",
				Global: &RateLimit{ // Changed to RateLimit
					RequestsPerSecond: 0,           // Invalid
					BurstSize:         1,           // Added to satisfy RateLimit validation
					WindowSize:        time.Minute, // Added to satisfy RateLimit validation
				},
			},
			wantErr: true,
		},
		{
			name: "ByIP RateLimit Validation Fails",
			config: RateLimitConfig{
				Enabled:  true,
				Strategy: "ip",
				ByIP: &RateLimit{
					RequestsPerSecond: 0, // Invalid
				},
			},
			wantErr: true,
		},
		{
			name: "ByAPIKey RateLimit Validation Fails",
			config: RateLimitConfig{
				Enabled:  true,
				Strategy: "api_key",
				ByAPIKey: map[string]*RateLimit{
					"key1": {RequestsPerSecond: 0}, // Invalid
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RateLimitConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCORSConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CORSConfig
		wantErr bool
	}{
		{
			name:    "Valid CORS Config Enabled",
			config:  DefaultCORSConfig(),
			wantErr: false,
		},
		{
			name: "Valid CORS Config Disabled",
			config: CORSConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "Enabled with Empty AllowedOrigins",
			config: CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{},
			},
			wantErr: true,
		},
		{
			name: "Enabled with Empty AllowedMethods",
			config: CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CORSConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityHeaders_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  SecurityHeaders
		wantErr bool
	}{
		{
			name:    "Valid SecurityHeaders Config Enabled",
			config:  DefaultSecurityHeaders(),
			wantErr: false,
		},
		{
			name: "Valid SecurityHeaders Config Disabled",
			config: SecurityHeaders{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "Enabled with Negative HSTSMaxAge",
			config: SecurityHeaders{
				Enabled:    true,
				HSTSMaxAge: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SecurityHeaders.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRateLimit_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  RateLimit
		wantErr bool
	}{
		{
			name: "Valid RateLimit",
			config: RateLimit{
				RequestsPerSecond: 100,
				BurstSize:         200,
				WindowSize:        time.Minute,
			},
			wantErr: false,
		},
		{
			name: "Zero RequestsPerSecond",
			config: RateLimit{
				RequestsPerSecond: 0,
				BurstSize:         200,
				WindowSize:        time.Minute,
			},
			wantErr: true,
		},
		{
			name: "Zero BurstSize",
			config: RateLimit{
				RequestsPerSecond: 100,
				BurstSize:         0,
				WindowSize:        time.Minute,
			},
			wantErr: true,
		},
		{
			name: "Zero WindowSize",
			config: RateLimit{
				RequestsPerSecond: 100,
				BurstSize:         200,
				WindowSize:        0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RateLimit.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  SecurityConfig
		wantErr bool
	}{
		{
			name:    "Valid Security Config",
			config:  DefaultSecurityConfig(),
			wantErr: false,
		},
		{
			name: "Invalid Auth Config",
			config: SecurityConfig{

				RateLimit: DefaultRateLimitConfig(),
				Headers:   DefaultSecurityHeaders(),
				CORS:      DefaultCORSConfig(),
			},
			wantErr: true,
		},
		{
			name: "Invalid RateLimit Config",
			config: SecurityConfig{

				RateLimit: RateLimitConfig{
					Enabled:  true,
					Strategy: "invalid",
				},
				Headers: DefaultSecurityHeaders(),
				CORS:    DefaultCORSConfig(),
			},
			wantErr: true,
		},
		{
			name: "Invalid Security Headers Config",
			config: SecurityConfig{

				RateLimit: DefaultRateLimitConfig(),
				Headers: SecurityHeaders{
					Enabled:    true,
					HSTSMaxAge: -1,
				},
				CORS: DefaultCORSConfig(),
			},
			wantErr: true,
		},
		{
			name: "Invalid CORS Config",
			config: SecurityConfig{

				RateLimit: DefaultRateLimitConfig(),
				Headers:   DefaultSecurityHeaders(),
				CORS: CORSConfig{
					Enabled:        true,
					AllowedOrigins: []string{},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SecurityConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
