package server

import (
	"context"
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

	// Server configuration
	metricsPort     string
	readTimeout     time.Duration
	writeTimeout    time.Duration
	idleTimeout     time.Duration
	shutdownTimeout time.Duration

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

func New(specFile, host, port string, securityConfig *security.SecurityConfig, metricsPort string, readTimeout, writeTimeout, idleTimeout, shutdownTimeout time.Duration, maxRequestSize int64) (*Server, error) {
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
		parser:          p,
		host:            host,
		port:            port,
		corsCfg:         corsConfigFromSecurity(securityConfig.CORS),
		maxReqSize:      maxRequestSize,
		metricsPort:     metricsPort,
		readTimeout:     readTimeout,
		writeTimeout:    writeTimeout,
		idleTimeout:     idleTimeout,
		shutdownTimeout: shutdownTimeout,
		cache:           &sync.Map{},
		routes:          routes,
		routeMap:        routeMap,
		authManager:     authManager,
		rateLimiter:     rateLimiter,
		securityConfig:  securityConfig,
		logger:          logger,
		metrics:         metrics,
		tracer:          tracer,
		startTime:       time.Now(),
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

func corsConfigFromSecurity(corsCfg security.CORSConfig) *CORSConfig {
	return &CORSConfig{
		Enabled:          corsCfg.Enabled,
		AllowedOrigins:   corsCfg.AllowedOrigins,
		AllowedMethods:   corsCfg.AllowedMethods,
		AllowedHeaders:   corsCfg.AllowedHeaders,
		AllowCredentials: corsCfg.AllowCredentials,
		MaxAge:           corsCfg.MaxAge,
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
		ReadTimeout:    s.readTimeout,
		WriteTimeout:   s.writeTimeout,
		IdleTimeout:    s.idleTimeout,
		MaxHeaderBytes: 1 << 20, // 1MB max header size
	}

	s.logger.Logger.Info("Starting server",
		zap.String("host", s.host),
		zap.String("port", s.port),
		zap.Int("routes", len(s.routes)),
	)

	s.metrics.SetHealthStatus(true)

	// Start metrics server in background
	var metricsServer *http.Server
	if s.metrics != nil {
		metricsMux := http.NewServeMux()
		metricsMux.Handle("/metrics", s.metrics.Handler())
		metricsServer = &http.Server{
			Addr:              fmt.Sprintf(":%s", s.metricsPort),
			Handler:           metricsMux,
			ReadHeaderTimeout: 5 * time.Second,
		}
		s.logger.Logger.Info("Starting metrics server",
			zap.String("port", s.metricsPort),
		)
		go func() {
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
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	// Shutdown metrics server first, then main server
	if metricsServer != nil {
		s.logger.Logger.Info("Shutting down metrics server...")
		if err := metricsServer.Shutdown(ctx); err != nil {
			s.logger.Logger.Error("Failed to shutdown metrics server", zap.Error(err))
		}
	}

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
			s.metrics.RecordRequest(r.Method, path, http.StatusMethodNotAllowed, time.Since(start), requestSize, 0)
			s.logger.Logger.Warn("Method not allowed",
				zap.String("method", r.Method),
				zap.String("path", path),
				zap.String("remote_addr", r.RemoteAddr),
			)
			return
		}

		// Cache key for response
		cacheKey := s.generateCacheKey(r.Method, path, r)

		// Try to get from cache
		if cached, ok := s.getCachedResponse(cacheKey); ok {
			s.sendJSONResponse(w, cached.StatusCode, cached.Body)
			s.metrics.RecordRequest(r.Method, path, cached.StatusCode, time.Since(start), requestSize, int64(len(cached.Body)))
			s.logger.Logger.Debug("Served from cache",
				zap.String("method", r.Method),
				zap.String("path", path),
				zap.Int("status_code", cached.StatusCode),
			)
			return
		}

		// Generate response
		statusCode := s.getStatusCodeFromRequest(r)
		buf, status, err := s.generateResponse(matchedRoute, statusCode)
		if err != nil {
			if strings.Contains(err.Error(), "no example found") {
				s.sendErrorResponse(w, http.StatusNotFound, err.Error())
				s.metrics.RecordRequest(r.Method, path, http.StatusNotFound, time.Since(start), requestSize, 0)
				s.logger.Logger.Warn("No example found",
					zap.String("status_code", statusCode),
					zap.String("path", path),
				)
			} else {
				s.sendErrorResponse(w, http.StatusInternalServerError, err.Error())
				s.metrics.RecordRequest(r.Method, path, http.StatusInternalServerError, time.Since(start), requestSize, 0)
				s.logger.Logger.Error("Failed to serialize response",
					zap.Error(err),
					zap.String("path", path),
				)
			}
			return
		}

		responseSize := int64(len(buf))

		// Cache the response
		s.cacheResponse(cacheKey, status, buf)

		// Send response
		s.sendJSONResponse(w, status, buf)
		s.metrics.RecordRequest(r.Method, path, status, time.Since(start), requestSize, responseSize)
		s.logger.Logger.Debug("Request processed",
			zap.String("method", r.Method),
			zap.String("path", path),
			zap.Int("status_code", status),
			zap.Duration("duration", time.Since(start)),
			zap.Int64("request_size", requestSize),
			zap.Int64("response_size", responseSize),
		)
	}

	s.logger.Logger.Info("Registered route",
		zap.String("methods", strings.Join(methods, ",")),
		zap.String("path", path),
	)
	mux.HandleFunc(muxPath, handler)
}
