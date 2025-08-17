package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/observability"
	"go.uber.org/zap"
)

// HealthHandler handles health check requests
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	_, span := s.tracer.StartSpan(r.Context(), "health_check")
	defer span.End()

	uptime := time.Since(s.startTime)
	health := observability.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    uptime.String(),
		Checks: map[string]bool{
			"parser": s.parser != nil,
			"routes": len(s.routes) > 0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(health)

	s.logger.Logger.Debug("Health check completed",
		zap.String("path", r.URL.Path),
		zap.String("remote_addr", r.RemoteAddr),
	)
}

// ReadinessHandler handles readiness check requests
func (s *Server) readinessHandler(w http.ResponseWriter, r *http.Request) {
	_, span := s.tracer.StartSpan(r.Context(), "readiness_check")
	defer span.End()

	ready := len(s.routes) > 0 && s.parser != nil

	if ready {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
	}

	s.logger.Logger.Debug("Readiness check completed",
		zap.String("path", r.URL.Path),
		zap.Bool("ready", ready),
	)
}

// MetricsHandler serves Prometheus metrics
func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	s.metrics.Handler().ServeHTTP(w, r)
}

// DocumentationHandler serves API documentation
func (s *Server) serveDocumentation(w http.ResponseWriter, r *http.Request) {
	_, span := s.tracer.StartSpan(r.Context(), "documentation")
	defer span.End()

	type RouteInfo struct {
		Method      string `json:"method"`
		Path        string `json:"path"`
		Description string `json:"description,omitempty"`
	}

	doc := struct {
		Message       string      `json:"message"`
		Version       string      `json:"version"`
		Environment   string      `json:"environment"`
		Endpoints     []RouteInfo `json:"endpoints"`
		Observability struct {
			Health    string `json:"health"`
			Metrics   string `json:"metrics"`
			Readiness string `json:"readiness"`
		} `json:"observability"`
	}{
		Message:     "Go-Spec-Mock Enterprise API Server",
		Version:     "1.0.0",
		Environment: "production",
		Endpoints:   make([]RouteInfo, 0, len(s.routes)),
	}

	for _, route := range s.routes {
		doc.Endpoints = append(doc.Endpoints, RouteInfo{
			Method: route.Method,
			Path:   route.Path,
		})
	}

	doc.Observability.Health = "/health"
	doc.Observability.Metrics = "/metrics"
	doc.Observability.Readiness = "/ready"

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)

	s.logger.Logger.Debug("Documentation served",
		zap.String("path", r.URL.Path),
		zap.Int("routes", len(s.routes)),
	)
}
