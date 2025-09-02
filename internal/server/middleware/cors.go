package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/leslieo2/go-spec-mock/internal/constants"
	"go.uber.org/zap"
)

// CORSMiddleware creates a CORS middleware with the given configuration

type CORSMiddleware struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
	logger           *zap.Logger
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(allowedOrigins, allowedMethods, allowedHeaders []string, allowCredentials bool, maxAge int, logger *zap.Logger) *CORSMiddleware {
	return &CORSMiddleware{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   allowedMethods,
		AllowedHeaders:   allowedHeaders,
		AllowCredentials: allowCredentials,
		MaxAge:           maxAge,
		logger:           logger,
	}
}

// Handler returns the CORS middleware handler
func (c *CORSMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get(constants.HeaderOrigin)

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range c.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set(constants.HeaderAccessControlAllowOrigin, origin)
			if len(c.AllowedMethods) > 0 {
				w.Header().Set(constants.HeaderAccessControlAllowMethods, strings.Join(c.AllowedMethods, ", "))
			}
			if len(c.AllowedHeaders) > 0 {
				w.Header().Set(constants.HeaderAccessControlAllowHeaders, strings.Join(c.AllowedHeaders, ", "))
			}
			if c.AllowCredentials {
				w.Header().Set(constants.HeaderAccessControlAllowCredentials, "true")
			}
			if c.MaxAge > 0 {
				w.Header().Set(constants.HeaderAccessControlMaxAge, fmt.Sprintf("%d", c.MaxAge))
			}

			c.logger.Debug("CORS headers applied",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("origin", origin),
				zap.String("remote_addr", r.RemoteAddr),
			)
		} else if origin != "" {
			c.logger.Warn("CORS origin not allowed",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("origin", origin),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Strings("allowed_origins", c.AllowedOrigins),
			)
		}

		// Handle preflight requests
		if r.Method == constants.MethodOPTIONS {
			c.logger.Debug("CORS preflight request handled",
				zap.String("path", r.URL.Path),
				zap.String("origin", origin),
				zap.String("remote_addr", r.RemoteAddr),
			)
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
