package server

import (
	"net/http"

	"github.com/leslieo2/go-spec-mock/internal/server/middleware"
)

// ApplyMiddleware applies the complete middleware chain to the handler
func (s *Server) applyMiddleware(handler http.Handler) http.Handler {
	// Apply middleware chain in reverse order

	// CORS middleware
	if s.config.Security.CORS.Enabled {
		corsMiddleware := middleware.NewCORSMiddleware(
			s.config.Security.CORS.AllowedOrigins,
			s.config.Security.CORS.AllowedMethods,
			s.config.Security.CORS.AllowedHeaders,
			s.config.Security.CORS.AllowCredentials,
			s.config.Security.CORS.MaxAge,
		)
		handler = corsMiddleware.Handler(handler)
	}

	// Security headers middleware
	if s.config.Security.Headers.Enabled {
		securityConfig := middleware.SecurityHeadersConfig{
			Enabled:    s.config.Security.Headers.Enabled,
			HSTSMaxAge: s.config.Security.Headers.HSTSMaxAge,
		}
		handler = middleware.SecurityHeadersMiddleware(securityConfig)(handler)
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
	handler = middleware.RequestSizeLimitMiddleware(s.config.Server.MaxRequestSize)(handler)

	// Logging middleware
	handler = middleware.LoggingMiddleware(s.logger.Logger)(handler)

	return handler
}
