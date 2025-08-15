package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/leslieo2/go-spec-mock/internal/observability"
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
	cfg := defaultCORSConfig()
	if cfg == nil {
		t.Fatal("defaultCORSConfig() returned nil")
	}

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
	server := &Server{
		corsCfg: &CORSConfig{
			Enabled:          true,
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: true,
			MaxAge:           3600,
		},
	}

	// Test handler
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	middleware := server.corsMiddleware(next)

	// Test actual request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	corsHeader := rec.Header().Get("Access-Control-Allow-Origin")
	if corsHeader != "http://localhost:3000" {
		t.Errorf("Expected CORS header, got %s", corsHeader)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	server := &Server{
		corsCfg: &CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST"},
			AllowedHeaders: []string{"Content-Type"},
		},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.corsMiddleware(next)

	// Test preflight request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for preflight, got %d", rec.Code)
	}
}

func TestCORSMiddleware_Disabled(t *testing.T) {
	server := &Server{
		corsCfg: &CORSConfig{
			Enabled: false,
		},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.corsMiddleware(next)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	corsHeader := rec.Header().Get("Access-Control-Allow-Origin")
	if corsHeader != "" {
		t.Errorf("Expected no CORS header when disabled, got %s", corsHeader)
	}
}

func TestRequestSizeLimitMiddleware(t *testing.T) {
	server := &Server{
		maxReqSize: 1024, // 1KB limit
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.requestSizeLimitMiddleware(next)

	// Test request within limit
	req := httptest.NewRequest("POST", "/test", strings.NewReader("small data"))
	req.ContentLength = 10
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid request, got %d", rec.Code)
	}

	// Test request exceeding limit
	req = httptest.NewRequest("POST", "/test", strings.NewReader("large data"))
	req.ContentLength = 2048
	rec = httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413 for oversized request, got %d", rec.Code)
	}
}

func TestRequestSizeLimitMiddleware_NoLimit(t *testing.T) {
	server := &Server{
		maxReqSize: 0, // No limit
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.requestSizeLimitMiddleware(next)

	req := httptest.NewRequest("POST", "/test", strings.NewReader("any data"))
	req.ContentLength = 9999999
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 when no limit, got %d", rec.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	server := &Server{
		logger: &observability.Logger{Logger: zap.NewNop()},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	middleware := server.loggingMiddleware(next)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}
}

func TestApplyMiddleware(t *testing.T) {
	server := &Server{
		corsCfg: &CORSConfig{
			Enabled: true,
		},
		maxReqSize: 1024,
		logger:     &observability.Logger{Logger: zap.NewNop()},
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
	rw := &responseWriter{ResponseWriter: base, statusCode: http.StatusOK}

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
	_, err := New("nonexistent.yaml", "localhost", "8080", nil)
	if err == nil {
		t.Error("Expected error for nonexistent spec file")
	}

	// Test with empty values
	_, err = New("", "", "", nil)
	if err == nil {
		t.Error("Expected error for empty spec file")
	}
}
