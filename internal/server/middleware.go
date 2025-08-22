package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ApplyMiddleware applies the complete middleware chain to the handler
func (s *Server) applyMiddleware(handler http.Handler) http.Handler {
	// Apply middleware chain in reverse order

	// CORS middleware
	if s.config.Security.CORS.Enabled {
		handler = s.corsMiddleware(handler)
	}

	// Security headers middleware
	if s.config.Security.Headers.Enabled {
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

// corsMiddleware handles Cross-Origin Resource Sharing
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range s.config.Security.CORS.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if len(s.config.Security.CORS.AllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(s.config.Security.CORS.AllowedMethods, ", "))
			}
			if len(s.config.Security.CORS.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(s.config.Security.CORS.AllowedHeaders, ", "))
			}
			if s.config.Security.CORS.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if s.config.Security.CORS.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", s.config.Security.CORS.MaxAge))
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

// SecurityHeadersMiddleware adds security headers to responses
func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config == nil || !s.config.Security.Headers.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", s.config.Security.Headers.HSTSMaxAge))

		if s.config.Security.Headers.ContentSecurityPolicy != "" {
			w.Header().Set("Content-Security-Policy", s.config.Security.Headers.ContentSecurityPolicy)
		}

		// Allowed hosts check
		if len(s.config.Security.Headers.AllowedHosts) > 0 {
			host := r.Host
			allowed := false
			for _, allowedHost := range s.config.Security.Headers.AllowedHosts {
				if host == allowedHost {
					allowed = true
					break
				}
			}
			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				response := map[string]interface{}{
					"error":   "FORBIDDEN",
					"message": "Host not allowed",
					"code":    "HOST_NOT_ALLOWED",
				}
				_ = json.NewEncoder(w).Encode(response)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// RequestSizeLimitMiddleware limits the size of incoming requests
func (s *Server) requestSizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.Server.MaxRequestSize > 0 && r.ContentLength > s.config.Server.MaxRequestSize {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("Request body too large, max size: %d bytes", s.config.Server.MaxRequestSize),
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs HTTP requests with detailed information
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer that captures the status code
		wrapped := &ResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

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
