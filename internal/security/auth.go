package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type APIKey struct {
	Key       string            `json:"key" yaml:"key"`
	Name      string            `json:"name" yaml:"name"`
	Enabled   bool              `json:"enabled" yaml:"enabled"`
	CreatedAt time.Time         `json:"created_at" yaml:"created_at"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	LastUsed  *time.Time        `json:"last_used,omitempty" yaml:"last_used,omitempty"`
	RateLimit *RateLimit        `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type RateLimit struct {
	RequestsPerSecond int           `json:"requests_per_second" yaml:"requests_per_second"`
	BurstSize         int           `json:"burst_size" yaml:"burst_size"`
	WindowSize        time.Duration `json:"window_size" yaml:"window_size"`
}

type AuthManager struct {
	mu     sync.RWMutex
	keys   map[string]*APIKey
	cache  *sync.Map
	config *AuthConfig
}

type AuthConfig struct {
	Enabled        bool             `json:"enabled" yaml:"enabled"`
	HeaderName     string           `json:"header_name" yaml:"header_name"`
	QueryParamName string           `json:"query_param_name" yaml:"query_param_name"`
	Keys           []*APIKey        `json:"keys" yaml:"keys"`
	RateLimit      *GlobalRateLimit `json:"rate_limit" yaml:"rate_limit"`
}

type GlobalRateLimit struct {
	RequestsPerSecond int           `json:"requests_per_second" yaml:"requests_per_second"`
	BurstSize         int           `json:"burst_size" yaml:"burst_size"`
	WindowSize        time.Duration `json:"window_size" yaml:"window_size"`
	Enabled           bool          `json:"enabled" yaml:"enabled"`
}

func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		Enabled:        false,
		HeaderName:     "X-API-Key",
		QueryParamName: "api_key",
		Keys:           []*APIKey{},
		RateLimit: &GlobalRateLimit{
			Enabled:           true,
			RequestsPerSecond: 100,
			BurstSize:         200,
			WindowSize:        time.Minute,
		},
	}
}

func NewAuthManager(config *AuthConfig) *AuthManager {
	am := &AuthManager{
		keys:   make(map[string]*APIKey),
		cache:  &sync.Map{},
		config: config,
	}

	if config != nil {
		for _, key := range config.Keys {
			am.keys[key.Key] = key
		}
	}

	return am
}

func (am *AuthManager) GenerateAPIKey(name string) (*APIKey, error) {
	key, err := generateRandomKey(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	apiKey := &APIKey{
		Key:       key,
		Name:      name,
		Enabled:   true,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	am.keys[key] = apiKey
	return apiKey, nil
}

func (am *AuthManager) ValidateAPIKey(providedKey string) (*APIKey, error) {
	if !am.config.Enabled {
		return nil, nil // Auth disabled
	}

	am.mu.RLock()
	defer am.mu.RUnlock()

	// To mitigate timing attacks, iterate through all keys and use a constant-time comparison.
	var foundKey *APIKey
	for _, apiKey := range am.keys {
		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(apiKey.Key), []byte(providedKey)) == 1 {
			foundKey = apiKey
			break
		}
	}

	if foundKey == nil {
		return nil, fmt.Errorf("invalid API key")
	}

	// Key exists, now check status
	if !foundKey.Enabled {
		return nil, fmt.Errorf("API key is disabled")
	}

	if foundKey.ExpiresAt != nil && time.Now().After(*foundKey.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	// Update last used timestamp in a separate goroutine to avoid lock contention
	go func(k *APIKey) {
		am.mu.Lock()
		defer am.mu.Unlock()
		now := time.Now()
		k.LastUsed = &now
	}(foundKey)

	return foundKey, nil
}

func (am *AuthManager) RevokeAPIKey(key string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if apiKey, exists := am.keys[key]; exists {
		apiKey.Enabled = false
		return nil
	}

	return fmt.Errorf("API key not found")
}

func (am *AuthManager) ListAPIKeys() []*APIKey {
	am.mu.RLock()
	defer am.mu.RUnlock()

	keys := make([]*APIKey, 0, len(am.keys))
	for _, key := range am.keys {
		keys = append(keys, key)
	}
	return keys
}

func (am *AuthManager) ExtractAPIKey(r *http.Request) string {
	// Check header first
	headerKey := r.Header.Get(am.config.HeaderName)
	if headerKey != "" {
		return strings.TrimSpace(headerKey)
	}

	// Check query parameter
	queryKey := r.URL.Query().Get(am.config.QueryParamName)
	if queryKey != "" {
		return strings.TrimSpace(queryKey)
	}

	// Check Authorization header (Bearer token)
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}

	return ""
}

func generateRandomKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// Middleware returns an HTTP middleware for API key authentication
func (am *AuthManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !am.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for health and metrics endpoints
		if am.shouldSkipAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		key := am.ExtractAPIKey(r)
		if key == "" {
			http.Error(w, "API key required", http.StatusUnauthorized)
			return
		}

		apiKey, err := am.ValidateAPIKey(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Add API key info to request context
		ctx := WithAPIKey(r.Context(), apiKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (am *AuthManager) shouldSkipAuth(path string) bool {
	skippedPaths := []string{
		"/health",
		"/ready",
		"/metrics",
	}

	for _, skipped := range skippedPaths {
		if path == skipped {
			return true
		}
	}

	return false
}

type contextKey string

const apiKeyContextKey contextKey = "api_key"

func WithAPIKey(ctx context.Context, key *APIKey) context.Context {
	return context.WithValue(ctx, apiKeyContextKey, key)
}

func APIKeyFromContext(ctx context.Context) (*APIKey, bool) {
	key, ok := ctx.Value(apiKeyContextKey).(*APIKey)
	return key, ok
}
