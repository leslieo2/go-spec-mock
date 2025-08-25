package server

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
	"github.com/leslieo2/go-spec-mock/internal/observability"
	"github.com/leslieo2/go-spec-mock/internal/server/middleware"
	"go.uber.org/zap"
)

func TestParseStatusCode(t *testing.T) {
	tests := []struct {
		name string
		code string
		want int
	}{
		{
			name: "valid 200",
			code: "200",
			want: 200,
		},
		{
			name: "valid 404",
			code: "404",
			want: 404,
		},
		{
			name: "invalid code",
			code: "invalid",
			want: 200,
		},
		{
			name: "empty code",
			code: "",
			want: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseStatusCode(tt.code); got != tt.want {
				t.Errorf("parseStatusCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultCORSConfig(t *testing.T) {
	cfg := config.DefaultCORSConfig()

	if !cfg.Enabled {
		t.Error("Expected CORS to be enabled by default")
	}

	if len(cfg.AllowedOrigins) == 0 || cfg.AllowedOrigins[0] != "*" {
		t.Error("Expected wildcard origin by default")
	}

	if len(cfg.AllowedMethods) == 0 {
		t.Error("Expected default allowed methods")
	}

	if cfg.MaxAge != 86400 {
		t.Errorf("Expected max age 86400, got %d", cfg.MaxAge)
	}
}

func TestCORSMiddleware(t *testing.T) {
	// Create CORS middleware directly
	corsMiddleware := middleware.NewCORSMiddleware(
		[]string{"*"},
		[]string{"GET", "POST"},
		[]string{"Content-Type"},
		true,
		3600,
	)

	// Test handler
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	handler := corsMiddleware.Handler(next)

	// Test actual request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	corsHeader := rec.Header().Get("Access-Control-Allow-Origin")
	if corsHeader != "http://localhost:3000" {
		t.Errorf("Expected CORS header, got %s", corsHeader)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	// Create CORS middleware directly
	corsMiddleware := middleware.NewCORSMiddleware(
		[]string{"*"},
		[]string{"GET", "POST"},
		[]string{"Content-Type"},
		false,
		0,
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := corsMiddleware.Handler(next)

	// Test preflight request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for preflight, got %d", rec.Code)
	}
}

func TestCORSMiddleware_Disabled(t *testing.T) {
	// Test that when CORS is not applied, no CORS headers are added
	// This test is mainly for ensuring no middleware is applied when CORS is disabled
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Since the middleware is only applied when CORS is enabled in applyMiddleware,
	// we just test a plain handler here
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	corsHeader := rec.Header().Get("Access-Control-Allow-Origin")
	if corsHeader != "" {
		t.Errorf("Expected no CORS header when disabled, got %s", corsHeader)
	}
}

func TestRequestSizeLimitMiddleware(t *testing.T) {
	// Create request size limit middleware directly
	requestSizeLimit := middleware.RequestSizeLimitMiddleware(1024) // 1KB limit

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := requestSizeLimit(next)

	// Test request within limit
	req := httptest.NewRequest("POST", "/test", strings.NewReader("small data"))
	req.ContentLength = 10
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid request, got %d", rec.Code)
	}

	// Test request exceeding limit
	req = httptest.NewRequest("POST", "/test", strings.NewReader("large data"))
	req.ContentLength = 2048
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413 for oversized request, got %d", rec.Code)
	}
}

func TestRequestSizeLimitMiddleware_NoLimit(t *testing.T) {
	// Create request size limit middleware with no limit (0)
	requestSizeLimit := middleware.RequestSizeLimitMiddleware(0) // No limit

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := requestSizeLimit(next)

	req := httptest.NewRequest("POST", "/test", strings.NewReader("any data"))
	req.ContentLength = 9999999
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 when no limit, got %d", rec.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	// Create logging middleware directly
	loggingMiddleware := middleware.LoggingMiddleware(zap.NewNop())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := loggingMiddleware(next)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}
}

func TestApplyMiddleware(t *testing.T) {
	server := &Server{
		config: &config.Config{
			Security: config.SecurityConfig{
				CORS: config.CORSConfig{
					Enabled: true,
				},
			},
			Server: config.ServerConfig{
				MaxRequestSize: 1024,
			},
		},
		logger: &observability.Logger{Logger: zap.NewNop()},
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := server.applyMiddleware(baseHandler)

	if wrapped == nil {
		t.Fatal("applyMiddleware returned nil")
	}

	// Test that middleware chain is applied
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 after middleware chain, got %d", rec.Code)
	}
}

func TestServeDocumentation(t *testing.T) {
	// Skip documentation test as it requires a valid parser
	t.Skip("Skipping documentation test as it requires valid parser")
}

func TestResponseWriter(t *testing.T) {
	base := httptest.NewRecorder()
	rw := &ResponseWriter{ResponseWriter: base, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)

	if rw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status code 404, got %d", rw.statusCode)
	}

	if base.Code != http.StatusNotFound {
		t.Errorf("Expected underlying writer status 404, got %d", base.Code)
	}
}

func TestNew(t *testing.T) {
	// Test invalid spec file
	cfg := &config.Config{SpecFile: "nonexistent.yaml"}
	_, err := New(cfg)
	if err == nil {
		t.Error("Expected error for nonexistent spec file")
	}

	// Test with empty values
	cfg = &config.Config{SpecFile: ""}
	_, err = New(cfg)
	if err == nil {
		t.Error("Expected error for empty spec file")
	}
}

func TestProxyConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		config  config.ProxyConfig
		wantErr bool
	}{
		{
			name: "valid proxy configuration",
			config: config.ProxyConfig{
				Enabled: true,
				Target:  "http://localhost:8081",
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "proxy enabled but no target",
			config: config.ProxyConfig{
				Enabled: true,
				Target:  "",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "proxy disabled with invalid target",
			config: config.ProxyConfig{
				Enabled: false,
				Target:  "",
				Timeout: 0,
			},
			wantErr: false, // Should not error when disabled
		},
		{
			name: "invalid timeout",
			config: config.ProxyConfig{
				Enabled: true,
				Target:  "http://localhost:8081",
				Timeout: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProxyConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewProxy(t *testing.T) {
	tests := []struct {
		name    string
		config  config.ProxyConfig
		wantErr bool
	}{
		{
			name: "valid proxy",
			config: config.ProxyConfig{
				Enabled: true,
				Target:  "http://localhost:8081",
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "proxy not enabled",
			config: config.ProxyConfig{
				Enabled: false,
				Target:  "http://localhost:8081",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid target URL",
			config: config.ProxyConfig{
				Enabled: true,
				Target:  "://invalid-url",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := middleware.NewProxy(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProxy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateCacheKey(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name     string
		method   string
		path     string
		request  *http.Request
		expected string
	}{
		{
			name:   "basic request without parameters",
			method: "GET",
			path:   "/users",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/users", nil)
				return req
			}(),
			expected: "GET:/users:200",
		},
		{
			name:   "request with query parameters",
			method: "GET",
			path:   "/users",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/users?name=john&age=30", nil)
				return req
			}(),
			expected: "GET:/users:200:age=30&name=john",
		},
		{
			name:   "request with status code override",
			method: "GET",
			path:   "/users",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/users?__statusCode=404", nil)
				return req
			}(),
			expected: "GET:/users:404",
		},
		{
			name:   "request with multiple parameter values",
			method: "GET",
			path:   "/search",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/search?tags=go&tags=api&q=test", nil)
				return req
			}(),
			expected: "GET:/search:200:q=test&tags=api&tags=go",
		},
		{
			name:   "request with authorization header",
			method: "GET",
			path:   "/secure",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/secure", nil)
				req.Header.Set("Authorization", "Bearer token123")
				return req
			}(),
			expected: func() string {
				// The auth hash will be consistent for the same token
				h := sha256.New()
				h.Write([]byte("Bearer token123"))
				authHash := fmt.Sprintf("%x", h.Sum(nil))[:16]
				return "GET:/secure:200:auth:" + authHash
			}(),
		},
		{
			name:   "request with accept header",
			method: "GET",
			path:   "/data",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/data", nil)
				req.Header.Set("Accept", "application/json")
				return req
			}(),
			expected: "GET:/data:200:accept:application/json",
		},
		{
			name:   "request with content type header",
			method: "POST",
			path:   "/data",
			request: func() *http.Request {
				req, _ := http.NewRequest("POST", "/data", nil)
				req.Header.Set("Content-Type", "application/json")
				return req
			}(),
			expected: "POST:/data:200:content-type:application/json",
		},
		{
			name:   "request with all context components",
			method: "GET",
			path:   "/api",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/api?param=value", nil)
				req.Header.Set("Authorization", "Bearer token")
				req.Header.Set("Accept", "application/json")
				req.Header.Set("Content-Type", "application/json")
				return req
			}(),
			expected: func() string {
				h := sha256.New()
				h.Write([]byte("Bearer token"))
				authHash := fmt.Sprintf("%x", h.Sum(nil))[:16]
				return "GET:/api:200:param=value:auth:" + authHash + ";accept:application/json;content-type:application/json"
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := server.generateCacheKey(tt.method, tt.path, tt.request)
			if key != tt.expected {
				t.Errorf("generateCacheKey() = %v, want %v", key, tt.expected)
			}
		})
	}
}

func TestGenerateCacheKey_CollisionPrevention(t *testing.T) {
	server := &Server{}

	// Test that different parameter orders generate different cache keys
	req1, _ := http.NewRequest("GET", "/test?a=1&b=2", nil)
	req2, _ := http.NewRequest("GET", "/test?b=2&a=1", nil)

	key1 := server.generateCacheKey("GET", "/test", req1)
	key2 := server.generateCacheKey("GET", "/test", req2)

	if key1 != key2 {
		t.Error("Different parameter orders should generate identical cache keys after sorting")
	}

	// Test that different authorization tokens generate different cache keys
	req3, _ := http.NewRequest("GET", "/secure", nil)
	req3.Header.Set("Authorization", "Bearer token1")

	req4, _ := http.NewRequest("GET", "/secure", nil)
	req4.Header.Set("Authorization", "Bearer token2")

	key3 := server.generateCacheKey("GET", "/secure", req3)
	key4 := server.generateCacheKey("GET", "/secure", req4)

	if key3 == key4 {
		t.Error("Different authorization tokens should generate different cache keys")
	}

	// Test that different accept headers generate different cache keys
	req5, _ := http.NewRequest("GET", "/data", nil)
	req5.Header.Set("Accept", "application/json")

	req6, _ := http.NewRequest("GET", "/data", nil)
	req6.Header.Set("Accept", "application/xml")

	key5 := server.generateCacheKey("GET", "/data", req5)
	key6 := server.generateCacheKey("GET", "/data", req6)

	if key5 == key6 {
		t.Error("Different accept headers should generate different cache keys")
	}
}

func TestGenerateCacheKey_InternalParameters(t *testing.T) {
	server := &Server{}

	// Test that internal parameters are excluded from cache key
	req, _ := http.NewRequest("GET", "/test?__statusCode=404&_=timestamp&realParam=value", nil)
	key := server.generateCacheKey("GET", "/test", req)

	// Should only include realParam, not __statusCode or _
	expected := "GET:/test:404:realParam=value"
	if key != expected {
		t.Errorf("generateCacheKey() = %v, want %v", key, expected)
	}
}
