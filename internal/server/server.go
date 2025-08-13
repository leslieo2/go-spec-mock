package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/parser"
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
}

type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func New(specFile, host, port string) (*Server, error) {
	p, err := parser.New(specFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	// Pre-build routes and route map
	routes := p.GetRoutes()
	routeMap := make(map[string][]parser.Route)
	for _, route := range routes {
		routeMap[route.Path] = append(routeMap[route.Path], route)
	}

	return &Server{
		parser:     p,
		host:       host,
		port:       port,
		corsCfg:    defaultCORSConfig(),
		maxReqSize: 10 * 1024 * 1024, // 10MB default limit
		cache:      &sync.Map{},
		routes:     routes,
		routeMap:   routeMap,
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

	for path, routes := range s.routeMap {
		s.registerRoute(mux, path, routes)
	}

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			log.Printf("Error encoding health response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	// Add documentation endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			s.serveDocumentation(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	// Apply middleware chain
	handler := s.applyMiddleware(mux)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", s.host, s.port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		// Optimized settings for better performance
		MaxHeaderBytes: 1 << 20, // 1MB max header size
	}

	log.Printf("Registered %d routes", len(s.routes))

	// Graceful shutdown
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

func (s *Server) registerRoute(mux *http.ServeMux, path string, routes []parser.Route) {
	// Convert OpenAPI path parameters to Go http.ServeMux format
	// Replace {petId} with * for wildcard matching
	muxPath := strings.ReplaceAll(path, "{", "")
	muxPath = strings.ReplaceAll(muxPath, "}", "")

	// Handle path parameters - convert to wildcard for Go's ServeMux
	// This is a simple approach for path parameters
	if strings.Contains(muxPath, "petId") {
		muxPath = "/pets/"
	}

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
		// Fast path: check if method exists
		matchedRoute, exists := routeLookup[r.Method]
		if !exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   fmt.Sprintf("Method %s not allowed", r.Method),
				"methods": methods,
			})
			return
		}

		// Cache key for response
		cacheKey := r.Method + ":" + path + ":" + r.URL.Query().Get("__statusCode")

		// Try to get from cache
		if cached, ok := s.cache.Load(cacheKey); ok {
			if response, ok := cached.(cachedResponse); ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(response.StatusCode)
				if _, err := w.Write(response.Body); err != nil {
					log.Printf("Error writing cached response: %v", err)
					// Optionally, handle this error more gracefully, e.g., by returning a 500
				}
				return
			}
		}

		statusCode := "200"
		if override := r.URL.Query().Get("__statusCode"); override != "" {
			statusCode = override
		}

		example, err := s.parser.GetExampleResponse(matchedRoute.Operation, statusCode)
		if err != nil {
			// Fast path: check for 2xx responses
			found := false
			for code := range matchedRoute.Operation.Responses.Map() {
				if strings.HasPrefix(code, "2") {
					if example, err = s.parser.GetExampleResponse(matchedRoute.Operation, code); err == nil {
						statusCode = code
						found = true
						break
					}
				}
			}
			if !found {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": fmt.Sprintf("No example found for status code %s", statusCode),
				})
				return
			}
		}

		// Serialize response
		buf, err := json.Marshal(example)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "Failed to serialize response",
			})
			return
		}

		status := parseStatusCode(statusCode)

		// Cache the response
		s.cache.Store(cacheKey, cachedResponse{
			StatusCode: status,
			Body:       buf,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if _, err := w.Write(buf); err != nil {
			log.Printf("Error writing response: %v", err)
			// Optionally, handle this error more gracefully, e.g., by returning a 500
		}
	}

	log.Printf("Registered route: %s %s", strings.Join(methods, ","), path)
	mux.HandleFunc(muxPath, handler)
}

type cachedResponse struct {
	StatusCode int
	Body       []byte
}

func (s *Server) serveDocumentation(w http.ResponseWriter, _ *http.Request) {
	doc := struct {
		Message string              `json:"message"`
		Routes  []map[string]string `json:"routes"`
	}{
		Message: "Go-Spec-Mock is running!",
		Routes:  make([]map[string]string, 0, len(s.routes)),
	}

	for _, route := range s.routes {
		doc.Routes = append(doc.Routes, map[string]string{
			"method": route.Method,
			"path":   route.Path,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

func (s *Server) applyMiddleware(handler http.Handler) http.Handler {
	// Apply middleware chain in reverse order

	// CORS middleware
	if s.corsCfg != nil && s.corsCfg.Enabled {
		handler = s.corsMiddleware(handler)
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
			if err := json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("Request body too large, max size: %d bytes", s.maxReqSize),
			}); err != nil {
				log.Printf("Error encoding request size limit response: %v", err)
			}
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
		log.Printf("[%s] %s %s - %d (%s)", r.Method, r.URL.Path, r.RemoteAddr, wrapped.statusCode, duration)
	})
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
