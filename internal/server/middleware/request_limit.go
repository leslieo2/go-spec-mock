package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/leslieo2/go-spec-mock/internal/constants"
	"go.uber.org/zap"
)

// RequestSizeLimitMiddleware creates a middleware that limits request body size
func RequestSizeLimitMiddleware(maxRequestSize int64, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if maxRequestSize > 0 && r.ContentLength > maxRequestSize {
				logger.Warn("Request size limit exceeded",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr),
					zap.Int64("content_length", r.ContentLength),
					zap.Int64("max_request_size", maxRequestSize),
				)

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
