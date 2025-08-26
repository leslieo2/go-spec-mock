package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/leslieo2/go-spec-mock/internal/constants"
)

// SecurityHeadersConfig defines configuration for security headers
type SecurityHeadersConfig struct {
	Enabled               bool
	HSTSMaxAge            int
	ContentSecurityPolicy string
	AllowedHosts          []string
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

			if config.ContentSecurityPolicy != "" {
				w.Header().Set(constants.HeaderContentSecurityPolicy, config.ContentSecurityPolicy)
			}

			// Allowed hosts check
			if len(config.AllowedHosts) > 0 {
				host := r.Host
				allowed := false
				for _, allowedHost := range config.AllowedHosts {
					if host == allowedHost {
						allowed = true
						break
					}
				}
				if !allowed {
					w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
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
}
