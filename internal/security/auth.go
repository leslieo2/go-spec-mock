package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
	"github.com/leslieo2/go-spec-mock/internal/constants"
)

type APIKey struct {
	Key       string            `json:"key" yaml:"key"`
	Name      string            `json:"name" yaml:"name"`
	Enabled   bool              `json:"enabled" yaml:"enabled"`
	CreatedAt time.Time         `json:"created_at" yaml:"created_at"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	LastUsed  *time.Time        `json:"last_used,omitempty" yaml:"last_used,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type AuthManager struct {
	mu     sync.RWMutex
	keys   map[string]*APIKey
	config *config.AuthConfig
}

func NewAuthManager(config *config.AuthConfig) *AuthManager {
	am := &AuthManager{
		keys:   make(map[string]*APIKey),
		config: config,
	}

	if config != nil {
		for _, key := range config.Keys {
			am.keys[key.Key] = &APIKey{
				Key:       key.Key,
				Name:      key.Name,
				Enabled:   key.Enabled,
				CreatedAt: key.CreatedAt,
				ExpiresAt: key.ExpiresAt,
				Metadata:  key.Metadata,
			}
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
	go func(key string) {
		am.mu.Lock()
		defer am.mu.Unlock()
		if apiKey, exists := am.keys[key]; exists {
			now := time.Now()
			apiKey.LastUsed = &now
		}
	}(foundKey.Key)

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
	authHeader := r.Header.Get(constants.HeaderAuthorization)
	if strings.HasPrefix(authHeader, constants.BearerPrefix) {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, constants.BearerPrefix))
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
			w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
			w.WriteHeader(constants.StatusUnauthorized)
			response := map[string]interface{}{
				"error":   "Authentication required",
				"message": "API key is required to access this endpoint",
				"code":    constants.ErrorCodeUnauthorized,
			}
			jsonResponse, _ := json.Marshal(response)
			_, _ = w.Write(jsonResponse)
			return
		}

		apiKey, err := am.ValidateAPIKey(key)
		if err != nil {
			w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
			w.WriteHeader(constants.StatusUnauthorized)
			code := constants.ErrorCodeInvalidAPIKey
			message := err.Error()

			// Map specific error types to codes
			switch {
			case strings.Contains(message, "expired"):
				code = constants.ErrorCodeAPIKeyExpired
			case strings.Contains(message, "disabled"):
				code = constants.ErrorCodeAPIKeyDisabled
			case strings.Contains(message, "invalid"):
				code = constants.ErrorCodeInvalidAPIKey
			}

			response := map[string]interface{}{
				"error":   code,
				"message": message,
				"code":    code,
			}
			jsonResponse, _ := json.Marshal(response)
			_, _ = w.Write(jsonResponse)
			return
		}

		// Add API key info to request context
		ctx := WithAPIKey(r.Context(), apiKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (am *AuthManager) shouldSkipAuth(path string) bool {
	skippedPaths := []string{
		constants.PathHealth,
		constants.PathReady,
		constants.PathMetrics,
		constants.PathDocumentation,
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
