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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
	"github.com/leslieo2/go-spec-mock/internal/constants"
	"github.com/leslieo2/go-spec-mock/internal/observability"
	"github.com/leslieo2/go-spec-mock/internal/parser"
	"github.com/leslieo2/go-spec-mock/internal/security"
	"github.com/leslieo2/go-spec-mock/internal/server/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// DynamicHandler wraps an http.Handler and allows atomic updates
type DynamicHandler struct {
	handler atomic.Value // stores http.Handler
}

// NewDynamicHandler creates a new DynamicHandler with the given initial handler
func NewDynamicHandler(handler http.Handler) *DynamicHandler {
	d := &DynamicHandler{}
	d.handler.Store(handler)
	return d
}

// ServeHTTP implements http.Handler by delegating to the current handler
func (d *DynamicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.handler.Load().(http.Handler).ServeHTTP(w, r)
}

// UpdateHandler atomically updates the handler
func (d *DynamicHandler) UpdateHandler(h http.Handler) {
	d.handler.Store(h)
}

type Server struct {
	parser   *parser.Parser
	config   *config.Config
	server   *http.Server
	cache    *sync.Map
	routes   []parser.Route
	routeMap map[string][]parser.Route
	mu       sync.RWMutex // Protects routes, routeMap, and parser

	// Dynamic handler for hot reload
	dynamicHandler *DynamicHandler

	// Security
	authManager *security.AuthManager
	rateLimiter *security.RateLimiter

	// Observability
	logger    *observability.Logger
	metrics   *observability.Metrics
	tracer    *observability.Tracer
	startTime time.Time

	// Proxy
	proxy *middleware.Proxy
}

func New(cfg *config.Config) (*Server, error) {
	p, err := parser.New(cfg.SpecFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	// Initialize observability
	logger, err := observability.NewLogger(cfg.Observability.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	metrics := observability.NewMetrics()
	if err := metrics.Register(); err != nil {
		return nil, fmt.Errorf("failed to register metrics: %w", err)
	}
	tracer, err := observability.NewTracer(cfg.Observability.Tracing)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	// Initialize security
	authManager := security.NewAuthManager(&cfg.Security.Auth)
	rateLimiter := security.NewRateLimiter(&cfg.Security)

	// Pre-build routes and route map
	routes := p.GetRoutes()
	routeMap := make(map[string][]parser.Route)
	for _, route := range routes {
		routeMap[route.Path] = append(routeMap[route.Path], route)
	}

	return &Server{
		parser:      p,
		config:      cfg,
		cache:       &sync.Map{},
		routes:      routes,
		routeMap:    routeMap,
		authManager: authManager,
		rateLimiter: rateLimiter,
		logger:      logger,
		metrics:     metrics,
		tracer:      tracer,
		startTime:   time.Now(),
	}, nil
}

// buildHandler creates a new http.Handler with current routes and middleware
func (s *Server) buildHandler() http.Handler {
	mux := http.NewServeMux()

	// Add observability endpoints
	mux.HandleFunc(constants.PathHealth, s.healthHandler)
	mux.HandleFunc(constants.PathMetrics, s.metricsHandler)
	mux.HandleFunc(constants.PathReady, s.readinessHandler)

	// Register OpenAPI routes first with proper synchronization
	s.mu.RLock()
	routeMapCopy := make(map[string][]parser.Route, len(s.routeMap))
	for path, routes := range s.routeMap {
		routeMapCopy[path] = routes
	}
	_, rootExists := s.routeMap["/"]
	s.mu.RUnlock()

	for path, routes := range routeMapCopy {
		s.registerRoute(mux, path, routes)
	}

	// Add documentation endpoint at /docs
	mux.HandleFunc(constants.PathDocumentation, s.serveDocumentation)

	// Handle root path: if no OpenAPI route exists, serve documentation or proxy
	if !rootExists {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				// Redirect to documentation or handle based on configuration
				http.Redirect(w, r, constants.PathDocumentation, http.StatusFound)
			} else if s.config.Proxy.Enabled {
				s.handleProxyRequest(w, r)
			} else {
				http.NotFound(w, r)
			}
		})
	}

	// Apply middleware chain and return
	return s.applyMiddleware(mux)
}

func (s *Server) Start() error {
	// Create initial handler and dynamic wrapper
	initialHandler := s.buildHandler()
	s.dynamicHandler = NewDynamicHandler(initialHandler)

	// Apply middleware chain to the dynamic handler
	handler := s.dynamicHandler

	s.server = &http.Server{
		Addr:           fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port),
		Handler:        handler,
		ReadTimeout:    s.config.Server.ReadTimeout,
		WriteTimeout:   s.config.Server.WriteTimeout,
		IdleTimeout:    s.config.Server.IdleTimeout,
		MaxHeaderBytes: 1 << 20, // 1MB max header size
	}

	s.logger.Logger.Info("Starting server",
		zap.String("host", s.config.Server.Host),
		zap.String("port", s.config.Server.Port),
		zap.Int("routes", len(s.routes)),
	)

	s.metrics.SetHealthStatus(true)

	// Start metrics server in background
	var metricsServer *http.Server
	if s.metrics != nil {
		metricsMux := http.NewServeMux()
		metricsMux.Handle(constants.PathMetrics, s.metrics.Handler())
		metricsServer = &http.Server{
			Addr:              fmt.Sprintf(":%s", s.config.Server.MetricsPort),
			Handler:           metricsMux,
			ReadHeaderTimeout: 5 * time.Second,
		}
		s.logger.Logger.Info("Starting metrics server",
			zap.String("port", s.config.Server.MetricsPort),
		)
		go func() {
			if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.logger.Logger.Error("Metrics server failed", zap.Error(err))
			}
		}()
	}

	go func() {
		var err error
		if s.config.TLS.Enabled {
			s.logger.Logger.Info("Starting server with HTTPS/TLS",
				zap.String("host", s.config.Server.Host),
				zap.String("port", s.config.Server.Port),
			)
			err = s.server.ListenAndServeTLS(s.config.TLS.CertFile, s.config.TLS.KeyFile)
		} else {
			s.logger.Logger.Info("Starting server with HTTP",
				zap.String("host", s.config.Server.Host),
				zap.String("port", s.config.Server.Port),
			)
			err = s.server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Logger.Info("Shutting down server...")
	s.metrics.SetHealthStatus(false)

	// Graceful shutdown with parallel server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Server.ShutdownTimeout)
	defer cancel()

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Shutdown metrics server in parallel
	if metricsServer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Logger.Info("Shutting down metrics server...")
			if err := metricsServer.Shutdown(ctx); err != nil {
				s.logger.Logger.Error("Failed to shutdown metrics server", zap.Error(err))
				errChan <- fmt.Errorf("metrics server shutdown: %w", err)
			}
		}()
	}

	// Shutdown main server in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.logger.Logger.Info("Shutting down main server...")
		if err := s.server.Shutdown(ctx); err != nil {
			s.logger.Logger.Error("Failed to shutdown main server", zap.Error(err))
			errChan <- fmt.Errorf("main server shutdown: %w", err)
		}
	}()

	// Wait for both shutdowns to complete
	wg.Wait()
	close(errChan)

	// Return the first error encountered, if any
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
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

// Reload implements the hotreload.Reloadable interface
func (s *Server) Reload(ctx context.Context) error {
	s.logger.Logger.Info("Reloading server configuration")

	// Parse the updated OpenAPI spec
	newParser, err := parser.New(s.config.SpecFile)
	if err != nil {
		return fmt.Errorf("failed to parse updated OpenAPI spec: %w", err)
	}

	// Update routes by re-initializing the parser and routes
	newRoutes := newParser.GetRoutes()
	newRouteMap := make(map[string][]parser.Route)

	for _, route := range newRoutes {
		newRouteMap[route.Path] = append(newRouteMap[route.Path], route)
	}

	// Update server state atomically with proper synchronization
	s.mu.Lock()
	s.routes = newRoutes
	s.routeMap = newRouteMap
	s.parser = newParser
	s.mu.Unlock()

	// Rebuild and swap the handler atomically
	newHandler := s.buildHandler()
	s.dynamicHandler.UpdateHandler(newHandler)

	s.logger.Logger.Info("Server configuration reloaded successfully",
		zap.Int("routes", len(newRoutes)))
	return nil
}

// handleProxyRequest handles requests by forwarding them to the configured proxy target
func (s *Server) handleProxyRequest(w http.ResponseWriter, r *http.Request) {
	if s.proxy == nil {
		// Lazy initialization of proxy
		proxy, err := middleware.NewProxy(s.config.Proxy)
		if err != nil {
			s.logger.Logger.Error("Failed to initialize proxy", zap.Error(err))
			http.Error(w, "Proxy configuration error", http.StatusInternalServerError)
			return
		}
		s.proxy = proxy
	}

	// Forward the request to the target server
	s.proxy.ServeHTTP(w, r)
}

// Name returns the name of this reloadable component
func (s *Server) Name() string {
	return "mock-server"
}
