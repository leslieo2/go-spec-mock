package middleware

import (
	"fmt"
	"net/http"

	"github.com/leslieo2/go-spec-mock/internal/constants"
)

// SecurityHeadersConfig defines configuration for security headers
type SecurityHeadersConfig struct {
	Enabled    bool
	HSTSMaxAge int
}

// SecurityHeadersMiddleware creates a security headers middleware
func SecurityHeadersMiddleware(config SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Security headers
			w.Header().Set(constants.HeaderXContentTypeOptions, constants.XContentTypeOptionsNoSniff)
			w.Header().Set(constants.HeaderXFrameOptions, constants.XFrameOptionsDeny)
			w.Header().Set(constants.HeaderXXSSProtection, constants.XXSSProtectionBlock)
			w.Header().Set(constants.HeaderStrictTransportSecurity, fmt.Sprintf("max-age=%d; includeSubDomains", config.HSTSMaxAge))

			next.ServeHTTP(w, r)
		})
	}
}
