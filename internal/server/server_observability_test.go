package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/observability"
	"github.com/leslieo2/go-spec-mock/internal/parser"
	"github.com/leslieo2/go-spec-mock/internal/security"
)

func TestServer_ObservabilityEndpoints(t *testing.T) {
	// Create server with test spec
	specFile := "../../examples/petstore.yaml"
	server, err := New(specFile, "localhost", "8080", security.DefaultSecurityConfig())
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() { _ = server.logger.Sync() }()

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		contentType    string
	}{
		{
			name:           "health endpoint",
			endpoint:       "/health",
			expectedStatus: http.StatusOK,
			contentType:    "application/json",
		},
		{
			name:           "ready endpoint",
			endpoint:       "/ready",
			expectedStatus: http.StatusOK,
			contentType:    "application/json",
		},
		{
			name:           "metrics endpoint",
			endpoint:       "/metrics",
			expectedStatus: http.StatusOK,
			contentType:    "text/plain; version=0.0.4; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint, nil)
			w := httptest.NewRecorder()

			var handler http.HandlerFunc
			switch tt.endpoint {
			case "/health":
				handler = server.healthHandler
			case "/ready":
				handler = server.readinessHandler
			case "/metrics":
				handler = server.metricsHandler
			}

			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

		})
	}
}

func TestHealthHandler(t *testing.T) {
	specFile := "../../examples/petstore.yaml"
	server, err := New(specFile, "localhost", "8080", security.DefaultSecurityConfig())
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() { _ = server.logger.Sync() }()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var health observability.HealthStatus
	if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}

	if health.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", health.Version)
	}

	if health.Uptime == "" {
		t.Error("Expected uptime to be set")
	}

	if len(health.Checks) == 0 {
		t.Error("Expected health checks to be present")
	}

	// Verify specific checks
	if !health.Checks["parser"] {
		t.Error("Expected parser check to be true")
	}
	if !health.Checks["routes"] {
		t.Error("Expected routes check to be true")
	}
}

func TestReadinessHandler_Ready(t *testing.T) {
	specFile := "../../examples/petstore.yaml"
	server, err := New(specFile, "localhost", "8080", security.DefaultSecurityConfig())
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() { _ = server.logger.Sync() }()

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.readinessHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode readiness response: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", response["status"])
	}
}

func TestReadinessHandler_NotReady(t *testing.T) {
	// Create server with minimal setup to test not-ready state
	logger, _ := observability.NewLogger(observability.DefaultLogConfig())
	tracer, _ := observability.NewTracer(observability.DefaultTraceConfig())
	server := &Server{
		routes:    []parser.Route{},
		parser:    nil,
		logger:    logger,
		metrics:   observability.NewMetrics(),
		tracer:    tracer,
		startTime: time.Now(),
	}

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.readinessHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode readiness response: %v", err)
	}

	if response["status"] != "not ready" {
		t.Errorf("Expected status 'not ready', got '%s'", response["status"])
	}
}

func TestMetricsHandler(t *testing.T) {
	specFile := "../../examples/petstore.yaml"
	server, err := New(specFile, "localhost", "8080", security.DefaultSecurityConfig())
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() { _ = server.logger.Sync() }()

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	server.metricsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected Content-Type to contain 'text/plain', got '%s'", contentType)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Expected metrics body to contain data")
	}
}

func TestDocumentationHandler(t *testing.T) {
	specFile := "../../examples/petstore.yaml"
	server, err := New(specFile, "localhost", "8080", security.DefaultSecurityConfig())
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() { _ = server.logger.Sync() }()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.serveDocumentation(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var doc struct {
		Message     string `json:"message"`
		Version     string `json:"version"`
		Environment string `json:"environment"`
		Endpoints   []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"endpoints"`
		Observability struct {
			Health    string `json:"health"`
			Metrics   string `json:"metrics"`
			Readiness string `json:"readiness"`
		} `json:"observability"`
	}

	if err := json.NewDecoder(w.Body).Decode(&doc); err != nil {
		t.Fatalf("Failed to decode documentation: %v", err)
	}

	if doc.Message != "Go-Spec-Mock Enterprise API Server" {
		t.Errorf("Expected message 'Go-Spec-Mock Enterprise API Server', got '%s'", doc.Message)
	}

	if doc.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", doc.Version)
	}

	if len(doc.Endpoints) == 0 {
		t.Error("Expected endpoints to be present")
	}

	if doc.Observability.Health != "/health" {
		t.Errorf("Expected health endpoint '/health', got '%s'", doc.Observability.Health)
	}

	if doc.Observability.Metrics != "/metrics" {
		t.Errorf("Expected metrics endpoint '/metrics', got '%s'", doc.Observability.Metrics)
	}

	if doc.Observability.Readiness != "/ready" {
		t.Errorf("Expected readiness endpoint '/ready', got '%s'", doc.Observability.Readiness)
	}
}

func TestServer_MetricsCollection(t *testing.T) {
	specFile := "../../examples/petstore.yaml"
	server, err := New(specFile, "localhost", "8080", security.DefaultSecurityConfig())
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() { _ = server.logger.Sync() }()

	// Verify metrics collection is initialized
	// Note: Actual metrics collection testing would require integration tests

	// This test verifies that metrics collection is initialized
	// Actual metrics collection would require integration testing
	if server.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}

	if server.logger == nil {
		t.Error("Expected logger to be initialized")
	}

	if server.tracer == nil {
		t.Error("Expected tracer to be initialized")
	}
}

func TestServer_ObservabilityIntegration(t *testing.T) {
	specFile := "../../examples/petstore.yaml"
	server, err := New(specFile, "localhost", "8080", security.DefaultSecurityConfig())
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() { _ = server.logger.Sync() }()

	// Test that all observability components are properly initialized
	if server.logger == nil {
		t.Error("Logger is not initialized")
	}

	if server.metrics == nil {
		t.Error("Metrics is not initialized")
	}

	if server.tracer == nil {
		t.Error("Tracer is not initialized")
	}

	// Test that health status is properly set
	server.metrics.SetHealthStatus(true)

	// Test start time is set
	if server.startTime.IsZero() {
		t.Error("Start time is not set")
	}
}
