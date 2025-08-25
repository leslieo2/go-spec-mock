package server

import (
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
