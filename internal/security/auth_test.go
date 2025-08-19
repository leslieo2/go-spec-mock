package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_Success(t *testing.T) {
	apiKey, _ := generateRandomKey(32)
	config := &AuthConfig{
		Enabled:    true,
		HeaderName: "X-API-Key",
		Keys: []*APIKey{
			{Key: apiKey, Enabled: true},
		},
	}
	am := NewAuthManager(config)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", apiKey)
	rr := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if API key is in context
		ctxKey, ok := APIKeyFromContext(r.Context())
		assert.True(t, ok, "API key should be in context")
		assert.Equal(t, apiKey, ctxKey.Key, "Context should contain the correct API key")
		w.WriteHeader(http.StatusOK)
	})

	middleware := am.Middleware(testHandler)
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Handler should be called for valid key")
}

func TestAuthMiddleware_Failure_InvalidKey(t *testing.T) {
	config := &AuthConfig{
		Enabled:    true,
		HeaderName: "X-API-Key",
		Keys: []*APIKey{
			{Key: "valid-key", Enabled: true},
		},
	}
	am := NewAuthManager(config)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	rr := httptest.NewRecorder()

	middleware := am.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called")
	}))
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code, "Should return 401 for invalid key")
	assert.Contains(t, rr.Body.String(), "invalid API key", "Response body should contain error message")
}

func TestAuthMiddleware_Failure_NoKey(t *testing.T) {
	config := &AuthConfig{
		Enabled:    true,
		HeaderName: "X-API-Key",
	}
	am := NewAuthManager(config)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	middleware := am.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called")
	}))
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code, "Should return 401 for missing key")
	assert.Contains(t, rr.Body.String(), "Authentication required", "Response body should contain error message")
}

func TestAuthMiddleware_Disabled(t *testing.T) {
	config := &AuthConfig{
		Enabled: false, // Auth is disabled
	}
	am := NewAuthManager(config)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	middleware := am.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Handler should be called when auth is disabled")
}

func TestAuthMiddleware_ExpiredKey(t *testing.T) {
	expiredTime := time.Now().Add(-1 * time.Hour)
	config := &AuthConfig{
		Enabled:    true,
		HeaderName: "X-API-Key",
		Keys: []*APIKey{
			{Key: "expired-key", Enabled: true, ExpiresAt: &expiredTime},
		},
	}
	am := NewAuthManager(config)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "expired-key")
	rr := httptest.NewRecorder()

	middleware := am.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called")
	}))
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code, "Should return 401 for expired key")
	assert.Contains(t, rr.Body.String(), "API key has expired", "Response body should contain error message")
}

func TestExtractAPIKey(t *testing.T) {
	config := &AuthConfig{
		HeaderName:     "X-Custom-API-Key",
		QueryParamName: "token",
	}
	am := NewAuthManager(config)

	testCases := []struct {
		name    string
		request *http.Request
		wantKey string
	}{
		{
			name:    "From Custom Header",
			request: httptest.NewRequest("GET", "/", nil),
			wantKey: "key-from-header",
		},
		{
			name:    "From Query Parameter",
			request: httptest.NewRequest("GET", "/?token=key-from-query", nil),
			wantKey: "key-from-query",
		},
		{
			name:    "From Authorization Bearer Header",
			request: httptest.NewRequest("GET", "/", nil),
			wantKey: "key-from-bearer",
		},
		{
			name:    "Header takes precedence over Query",
			request: httptest.NewRequest("GET", "/?token=key-from-query", nil),
			wantKey: "key-from-header",
		},
		{
			name:    "No Key",
			request: httptest.NewRequest("GET", "/", nil),
			wantKey: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup request for each case
			if tc.name == "From Custom Header" || tc.name == "Header takes precedence over Query" {
				tc.request.Header.Set("X-Custom-API-Key", "key-from-header")
			}
			if tc.name == "From Authorization Bearer Header" {
				tc.request.Header.Set("Authorization", "Bearer key-from-bearer")
			}

			key := am.ExtractAPIKey(tc.request)
			assert.Equal(t, tc.wantKey, key)
		})
	}
}

func TestAPIKeyContext(t *testing.T) {
	ctx := context.Background()
	apiKey := &APIKey{Key: "test-key"}

	// Test setting and getting
	ctxWithKey := WithAPIKey(ctx, apiKey)
	retrievedKey, ok := APIKeyFromContext(ctxWithKey)

	assert.True(t, ok, "Should be able to retrieve key from context")
	assert.Equal(t, apiKey, retrievedKey, "Retrieved key should match the one set")

	// Test getting from a context without the key
	_, ok = APIKeyFromContext(ctx)
	assert.False(t, ok, "Should not be able to retrieve key from empty context")
}
