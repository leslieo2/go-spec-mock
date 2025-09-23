package config

import "testing"

func TestSecurityConfigValidate_Default(t *testing.T) {
	s := DefaultSecurityConfig()
	if err := s.Validate(); err != nil {
		t.Fatalf("DefaultSecurityConfig should be valid, got error: %v", err)
	}
}

func TestSecurityConfigValidate_DisabledIgnoresEmptyValues(t *testing.T) {
	s := SecurityConfig{
		CORS: CORSConfig{Enabled: false},
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("Disabled CORS should validate, got error: %v", err)
	}
}

func TestCORSConfigValidate_FailsWhenEnabledWithoutOrigins(t *testing.T) {
	s := SecurityConfig{
		CORS: CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{},
			AllowedMethods: []string{"GET"},
		},
	}
	if err := s.Validate(); err == nil {
		t.Fatal("Expected error when CORS enabled without allowed origins")
	}
}

func TestCORSConfigValidate_FailsWhenEnabledWithoutMethods(t *testing.T) {
	s := SecurityConfig{
		CORS: CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"https://example.com"},
			AllowedMethods: []string{},
		},
	}
	if err := s.Validate(); err == nil {
		t.Fatal("Expected error when CORS enabled without allowed methods")
	}
}

func TestCORSConfigValidate_PassesWithExplicitValues(t *testing.T) {
	s := SecurityConfig{
		CORS: CORSConfig{
			Enabled:          true,
			AllowedOrigins:   []string{"https://example.com"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: true,
			MaxAge:           600,
		},
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("Expected valid configuration, got error: %v", err)
	}
}
