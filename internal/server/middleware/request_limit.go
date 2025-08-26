package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/leslieo2/go-spec-mock/internal/constants"
)

// RequestSizeLimitMiddleware creates a middleware that limits request body size
func RequestSizeLimitMiddleware(maxRequestSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if maxRequestSize > 0 && r.ContentLength > maxRequestSize {
				w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": fmt.Sprintf("Request body too large, max size: %d bytes", maxRequestSize),
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
