package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/leslieo2/go-spec-mock/internal/constants"
)

// CORSMiddleware creates a CORS middleware with the given configuration
type CORSMiddleware struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(allowedOrigins, allowedMethods, allowedHeaders []string, allowCredentials bool, maxAge int) *CORSMiddleware {
	return &CORSMiddleware{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   allowedMethods,
		AllowedHeaders:   allowedHeaders,
		AllowCredentials: allowCredentials,
		MaxAge:           maxAge,
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
		}

		// Handle preflight requests
		if r.Method == constants.MethodOPTIONS {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
