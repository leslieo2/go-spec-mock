package server

import (
	"net/http"

	"github.com/leslieo2/go-spec-mock/internal/constants"
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

	// Request size limit middleware
	handler = middleware.RequestSizeLimitMiddleware(constants.ServerMaxRequestSize)(handler)

	// Logging middleware
	handler = middleware.LoggingMiddleware(s.logger.Logger)(handler)

	return handler
}
