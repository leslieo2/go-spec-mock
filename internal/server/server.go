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
	parser, err := parser.New(specFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	return &Server{
		parser:     parser,
		host:       host,
		port:       port,
		corsCfg:    defaultCORSConfig(),
		maxReqSize: 10 * 1024 * 1024, // 10MB default limit
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
	routes := s.parser.GetRoutes()

	// Group routes by path to handle method-based routing
	routeMap := make(map[string][]parser.Route)
	for _, route := range routes {
		routeMap[route.Path] = append(routeMap[route.Path], route)
	}

	for path, routes := range routeMap {
		s.registerRoute(mux, path, routes)
	}

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
	}

	log.Printf("Registered %d routes", len(routes))

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

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Find the route that matches the request method
		var matchedRoute *parser.Route
		for _, route := range routes {
			if route.Method == r.Method {
				matchedRoute = &route
				break
			}
		}

		if matchedRoute == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)

			// List supported methods
			methods := make([]string, 0, len(routes))
			for _, route := range routes {
				methods = append(methods, route.Method)
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   fmt.Sprintf("Method %s not allowed", r.Method),
				"methods": methods,
			})
			return
		}

		// Check for status code override
		statusCode := "200"
		if override := r.URL.Query().Get("__statusCode"); override != "" {
			statusCode = override
		}

		example, err := s.parser.GetExampleResponse(matchedRoute.Operation, statusCode)
		if err != nil {
			// Try to find a 2xx response if the requested status code doesn't have an example
			found := false
			for code := range matchedRoute.Operation.Responses.Map() {
				if strings.HasPrefix(code, "2") {
					example, err = s.parser.GetExampleResponse(matchedRoute.Operation, code)
					if err == nil {
						statusCode = code
						found = true
						break
					}
				}
			}

			if !found {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{
					"error": fmt.Sprintf("No example found for status code %s", statusCode),
				})
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(parseStatusCode(statusCode))
		json.NewEncoder(w).Encode(example)
	}

	methods := make([]string, 0, len(routes))
	for _, route := range routes {
		methods = append(methods, route.Method)
	}

	log.Printf("Registered route: %s %s", strings.Join(methods, ","), path)
	mux.HandleFunc(muxPath, handler)
}

func (s *Server) serveDocumentation(w http.ResponseWriter, r *http.Request) {
	routes := s.parser.GetRoutes()

	doc := struct {
		Message string              `json:"message"`
		Routes  []map[string]string `json:"routes"`
	}{
		Message: "Go-Spec-Mock is running!",
		Routes:  make([]map[string]string, 0, len(routes)),
	}

	for _, route := range routes {
		doc.Routes = append(doc.Routes, map[string]string{
			"method": route.Method,
			"path":   route.Path,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
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
			json.NewEncoder(w).Encode(map[string]string{
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
