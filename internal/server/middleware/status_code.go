package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/leslieo2/go-spec-mock/internal/constants"
	"go.uber.org/zap"
)

// StatusCodeMiddleware creates a middleware that extracts and validates status code from query parameters
func StatusCodeMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for status code parameter
			if statusCodeParam := r.URL.Query().Get(constants.QueryParamStatusCode); statusCodeParam != "" {
				// Parse and validate status code
				statusCode, err := parseStatusCode(statusCodeParam)
				if err != nil {
					logger.Warn("Invalid status code parameter",
						zap.String("status_code", statusCodeParam),
						zap.String("path", r.URL.Path),
						zap.Error(err),
					)
					// Continue with default status code
				} else {
					// Store validated status code in request context for downstream handlers
					ctx := context.WithValue(r.Context(), constants.ContextKeyStatusCode, statusCode)
					r = r.WithContext(ctx)

					logger.Debug("Status code override applied",
						zap.String("path", r.URL.Path),
						zap.Int("status_code", statusCode),
					)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseStatusCode converts string status code to int with validation
func parseStatusCode(code string) (int, error) {
	statusCode, err := strconv.Atoi(code)
	if err != nil {
		return constants.StatusOK, err
	}

	// Validate status code range (100-599)
	if statusCode < 100 || statusCode >= 600 {
		return constants.StatusOK, &InvalidStatusCodeError{Code: statusCode}
	}

	return statusCode, nil
}

// InvalidStatusCodeError represents an invalid HTTP status code error
type InvalidStatusCodeError struct {
	Code int
}

func (e *InvalidStatusCodeError) Error() string {
	return strconv.Itoa(e.Code) + " is not a valid HTTP status code"
}
