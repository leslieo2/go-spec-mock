package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/observability"
	"github.com/leslieo2/go-spec-mock/internal/parser"
	"github.com/leslieo2/go-spec-mock/internal/security"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type Server struct {
	parser     *parser.Parser
	host       string
	port       string
	server     *http.Server
	corsCfg    *CORSConfig
	maxReqSize int64
	cache      *sync.Map
	routes     []parser.Route
	routeMap   map[string][]parser.Route

	// Security
	authManager    *security.AuthManager
	rateLimiter    *security.RateLimiter
	securityConfig *security.SecurityConfig

	// Observability
	logger    *observability.Logger
	metrics   *observability.Metrics
	tracer    *observability.Tracer
	startTime time.Time
}

type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func New(specFile, host, port string, securityConfig *security.SecurityConfig) (*Server, error) {
	p, err := parser.New(specFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	// Initialize observability
	logger, err := observability.NewLogger(observability.DefaultLogConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	metrics := observability.NewMetrics()
	tracer, err := observability.NewTracer(observability.DefaultTraceConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	// Initialize security
	if securityConfig == nil {
		securityConfig = security.DefaultSecurityConfig()
	}

	authManager := security.NewAuthManager(&securityConfig.Auth)
	rateLimiter := security.NewRateLimiter(&securityConfig.RateLimit)

	// Pre-build routes and route map
	routes := p.GetRoutes()
	routeMap := make(map[string][]parser.Route)
	for _, route := range routes {
		routeMap[route.Path] = append(routeMap[route.Path], route)
	}

	return &Server{
		parser:         p,
		host:           host,
		port:           port,
		corsCfg:        defaultCORSConfig(),
		maxReqSize:     10 * 1024 * 1024, // 10MB default limit
		cache:          &sync.Map{},
		routes:         routes,
		routeMap:       routeMap,
		authManager:    authManager,
		rateLimiter:    rateLimiter,
		securityConfig: securityConfig,
		logger:         logger,
		metrics:        metrics,
		tracer:         tracer,
		startTime:      time.Now(),
	}, nil
}

func defaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		Enabled:          true,
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Accept", "X-Requested-With"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Add observability endpoints
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/metrics", s.metricsHandler)
	mux.HandleFunc("/ready", s.readinessHandler)

	// Register OpenAPI routes first
	for path, routes := range s.routeMap {
		s.registerRoute(mux, path, routes)
	}

	// Add documentation endpoint only if no root route is defined in OpenAPI
	if _, exists := s.routeMap["/"]; !exists {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				s.serveDocumentation(w, r)
			} else {
				http.NotFound(w, r)
			}
		})
	}

	// Apply middleware chain
	handler := s.applyMiddleware(mux)

	s.server = &http.Server{
		Addr:           fmt.Sprintf("%s:%s", s.host, s.port),
		Handler:        handler,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB max header size
	}

	s.logger.Logger.Info("Starting server",
		zap.String("host", s.host),
		zap.String("port", s.port),
		zap.Int("routes", len(s.routes)),
	)

	s.metrics.SetHealthStatus(true)

	// Start metrics server in background
	if s.metrics != nil {
		go func() {
			metricsMux := http.NewServeMux()
			metricsMux.Handle("/metrics", s.metrics.Handler())
			metricsServer := &http.Server{
				Addr:              ":9090",
				Handler:           metricsMux,
				ReadHeaderTimeout: 5 * time.Second,
			}
			if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.logger.Logger.Error("Metrics server failed", zap.Error(err))
			}
		}()
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Logger.Info("Shutting down server...")
	s.metrics.SetHealthStatus(false)

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

func (s *Server) registerRoute(mux *http.ServeMux, path string, routes []parser.Route) {
	// With Go 1.22+, http.ServeMux supports path parameters in the {name} format,
	// which is the same as the OpenAPI spec. We can use the path as is.
	muxPath := path

	// Pre-compute supported methods for this path
	methods := make([]string, 0, len(routes))
	for _, route := range routes {
		methods = append(methods, route.Method)
	}

	// Create a fast lookup map for routes
	routeLookup := make(map[string]*parser.Route, len(routes))
	for i := range routes {
		routeLookup[routes[i].Method] = &routes[i]
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		_, span := s.tracer.StartSpan(r.Context(), "handle_request",
			attribute.String("http.method", r.Method),
			attribute.String("http.path", path),
			attribute.String("http.user_agent", r.UserAgent()),
		)
		defer span.End()

		// Get request size
		requestSize := r.ContentLength
		if requestSize < 0 {
			requestSize = 0
		}

		// Fast path: check if method exists
		matchedRoute, exists := routeLookup[r.Method]
		if !exists {
			s.sendMethodNotAllowedResponse(w, methods, r.Method)
			s.recordRequestMetrics(r.Method, path, http.StatusMethodNotAllowed, time.Since(start), requestSize, 0)
			s.logMethodNotAllowed(r.Method, path, r.RemoteAddr)
			return
		}

		// Cache key for response
		cacheKey := s.generateCacheKey(r.Method, path, r)

		// Try to get from cache
		if cached, ok := s.getCachedResponse(cacheKey); ok {
			s.sendJSONResponse(w, cached.StatusCode, cached.Body)
			s.recordRequestMetrics(r.Method, path, cached.StatusCode, time.Since(start), requestSize, int64(len(cached.Body)))
			s.logServedFromCache(r.Method, path, cached.StatusCode)
			return
		}

		// Generate response
		statusCode := s.getStatusCodeFromRequest(r)
		buf, status, err := s.generateResponse(matchedRoute, statusCode)
		if err != nil {
			if strings.Contains(err.Error(), "no example found") {
				s.sendErrorResponse(w, http.StatusNotFound, err.Error())
				s.recordRequestMetrics(r.Method, path, http.StatusNotFound, time.Since(start), requestSize, 0)
				s.logNoExampleFound(statusCode, path)
			} else {
				s.sendErrorResponse(w, http.StatusInternalServerError, err.Error())
				s.recordRequestMetrics(r.Method, path, http.StatusInternalServerError, time.Since(start), requestSize, 0)
				s.logSerializationError(err, path)
			}
			return
		}

		responseSize := int64(len(buf))

		// Cache the response
		s.cacheResponse(cacheKey, status, buf)

		// Send response
		s.sendJSONResponse(w, status, buf)
		s.recordRequestMetrics(r.Method, path, status, time.Since(start), requestSize, responseSize)
		s.logRequestProcessed(r.Method, path, status, time.Since(start), requestSize, responseSize)
	}

	s.logger.Logger.Info("Registered route",
		zap.String("methods", strings.Join(methods, ",")),
		zap.String("path", path),
	)
	mux.HandleFunc(muxPath, handler)
}

type cachedResponse struct {
	StatusCode int
	Body       []byte
}

// CacheManager handles response caching operations
func (s *Server) getCachedResponse(cacheKey string) (*cachedResponse, bool) {
	if cached, ok := s.cache.Load(cacheKey); ok {
		if response, ok := cached.(cachedResponse); ok {
			return &response, true
		}
	}
	return nil, false
}

func (s *Server) cacheResponse(cacheKey string, statusCode int, body []byte) {
	s.cache.Store(cacheKey, cachedResponse{
		StatusCode: statusCode,
		Body:       body,
	})
}

func (s *Server) generateCacheKey(method, path string, r *http.Request) string {
	return method + ":" + path + ":" + r.URL.Query().Get("__statusCode")
}

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

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	s.metrics.Handler().ServeHTTP(w, r)
}

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

func (s *Server) applyMiddleware(handler http.Handler) http.Handler {
	// Apply middleware chain in reverse order

	// CORS middleware
	if s.corsCfg != nil && s.corsCfg.Enabled {
		handler = s.corsMiddleware(handler)
	}

	// Security headers middleware
	if s.securityConfig != nil {
		handler = s.securityHeadersMiddleware(handler)
	}

	// Rate limiting middleware
	if s.rateLimiter != nil {
		handler = s.rateLimiter.Middleware(handler)
	}

	// API key authentication middleware
	if s.authManager != nil {
		handler = s.authManager.Middleware(handler)
	}

	// Request size limit middleware
	handler = s.requestSizeLimitMiddleware(handler)

	// Logging middleware
	handler = s.loggingMiddleware(handler)

	return handler
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range s.corsCfg.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if len(s.corsCfg.AllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(s.corsCfg.AllowedMethods, ", "))
			}
			if len(s.corsCfg.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(s.corsCfg.AllowedHeaders, ", "))
			}
			if s.corsCfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if s.corsCfg.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", s.corsCfg.MaxAge))
			}
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestSizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.maxReqSize > 0 && r.ContentLength > s.maxReqSize {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("Request body too large, max size: %d bytes", s.maxReqSize),
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer that captures the status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		s.logger.Logger.Info("HTTP request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.Int("status_code", wrapped.statusCode),
			zap.Duration("duration", duration),
			zap.String("user_agent", r.UserAgent()),
		)
	})
}

// ObservabilityManager handles metrics and logging operations
func (s *Server) recordRequestMetrics(method, path string, statusCode int, duration time.Duration, requestSize, responseSize int64) {
	s.metrics.RecordRequest(method, path, statusCode, duration, requestSize, responseSize)
}

func (s *Server) logRequestProcessed(method, path string, statusCode int, duration time.Duration, requestSize, responseSize int64) {
	s.logger.Logger.Debug("Request processed",
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status_code", statusCode),
		zap.Duration("duration", duration),
		zap.Int64("request_size", requestSize),
		zap.Int64("response_size", responseSize),
	)
}

func (s *Server) logMethodNotAllowed(method, path, remoteAddr string) {
	s.logger.Logger.Warn("Method not allowed",
		zap.String("method", method),
		zap.String("path", path),
		zap.String("remote_addr", remoteAddr),
	)
}

func (s *Server) logNoExampleFound(statusCode, path string) {
	s.logger.Logger.Warn("No example found",
		zap.String("status_code", statusCode),
		zap.String("path", path),
	)
}

func (s *Server) logSerializationError(err error, path string) {
	s.logger.Logger.Error("Failed to serialize response",
		zap.Error(err),
		zap.String("path", path),
	)
}

func (s *Server) logServedFromCache(method, path string, statusCode int) {
	s.logger.Logger.Debug("Served from cache",
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status_code", statusCode),
	)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func parseStatusCode(code string) int {
	var statusCode int
	_, err := fmt.Sscanf(code, "%d", &statusCode)
	if err != nil {
		return 200
	}
	return statusCode
}

// ResponseGenerator handles response generation logic
func (s *Server) getStatusCodeFromRequest(r *http.Request) string {
	statusCode := "200"
	if override := r.URL.Query().Get("__statusCode"); override != "" {
		statusCode = override
	}
	return statusCode
}

func (s *Server) generateResponse(route *parser.Route, statusCode string) ([]byte, int, error) {
	example, err := s.parser.GetExampleResponse(route.Operation, statusCode)
	if err == nil {
		buf, err := json.Marshal(example)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to serialize response: %w", err)
		}
		return buf, parseStatusCode(statusCode), nil
	}

	// Try to find any 2xx response if requested status not found
	for code := range route.Operation.Responses.Map() {
		if strings.HasPrefix(code, "2") {
			if example, err = s.parser.GetExampleResponse(route.Operation, code); err == nil {
				buf, err := json.Marshal(example)
				if err != nil {
					return nil, 0, fmt.Errorf("failed to serialize response: %w", err)
				}
				return buf, parseStatusCode(code), nil
			}
		}
	}

	return nil, 0, fmt.Errorf("no example found for status code %s", statusCode)
}

func (s *Server) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := map[string]string{"error": message}
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) sendMethodNotAllowedResponse(w http.ResponseWriter, methods []string, requestedMethod string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	response := map[string]interface{}{
		"error":   fmt.Sprintf("Method %s not allowed", requestedMethod),
		"methods": methods,
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.securityConfig == nil || !s.securityConfig.Headers.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", s.securityConfig.Headers.HSTSMaxAge))

		if s.securityConfig.Headers.ContentSecurityPolicy != "" {
			w.Header().Set("Content-Security-Policy", s.securityConfig.Headers.ContentSecurityPolicy)
		}

		// Allowed hosts check
		if len(s.securityConfig.Headers.AllowedHosts) > 0 {
			host := r.Host
			allowed := false
			for _, allowedHost := range s.securityConfig.Headers.AllowedHosts {
				if host == allowedHost {
					allowed = true
					break
				}
			}
			if !allowed {
				http.Error(w, "Host not allowed", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) sendJSONResponse(w http.ResponseWriter, statusCode int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}
