package middleware

import (
	"context"
	"net/http"

	"github.com/leslieo2/go-spec-mock/internal/constants"
	"go.uber.org/zap"
)

// Context key for example name
type contextKey string

const (
	ContextKeyExampleName = contextKey("exampleName")
)

// ExampleMiddleware creates a middleware that extracts example name from query parameters
func ExampleMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for example parameter
			if exampleName := r.URL.Query().Get(constants.QueryParamExample); exampleName != "" {
				// Store example name in request context for downstream handlers
				ctx := context.WithValue(r.Context(), ContextKeyExampleName, exampleName)
				r = r.WithContext(ctx)

				logger.Debug("Example name override applied",
					zap.String("path", r.URL.Path),
					zap.String("example_name", exampleName),
				)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetExampleNameFromContext retrieves the example name from the request context
func GetExampleNameFromContext(r *http.Request) string {
	if exampleName, ok := r.Context().Value(ContextKeyExampleName).(string); ok {
		return exampleName
	}
	return ""
}
