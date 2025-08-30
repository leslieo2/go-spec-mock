package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/constants"
	"go.uber.org/zap"
)

// DelayMiddleware creates a middleware that simulates network latency
func DelayMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for delay parameter
			if delayParam := r.URL.Query().Get(constants.QueryParamDelay); delayParam != "" {
				// Parse delay duration
				delayDuration, err := parseDelay(delayParam)
				if err != nil {
					logger.Warn("Invalid delay parameter",
						zap.String("delay", delayParam),
						zap.String("path", r.URL.Path),
						zap.Error(err),
					)
				} else if delayDuration > 0 {
					// Apply the delay
					select {
					case <-time.After(delayDuration):
						// Delay completed, continue with request
					case <-r.Context().Done():
						// Request was cancelled during delay
						logger.Debug("Request cancelled during delay",
							zap.String("path", r.URL.Path),
							zap.Duration("delay", delayDuration),
						)
						return
					}

					logger.Debug("Applied response delay",
						zap.String("path", r.URL.Path),
						zap.Duration("delay", delayDuration),
					)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseDelay parses a delay string into a time.Duration
func parseDelay(delayStr string) (time.Duration, error) {
	// Remove any whitespace
	delayStr = strings.TrimSpace(delayStr)

	// Check if it's a simple number (milliseconds by default)
	if ms, err := strconv.Atoi(delayStr); err == nil {
		delay := time.Duration(ms) * time.Millisecond
		return validateDelay(delay)
	}

	// Parse as duration string (e.g., "500ms", "2s", "1m")
	delay, err := time.ParseDuration(delayStr)
	if err != nil {
		return 0, err
	}

	return validateDelay(delay)
}

// validateDelay ensures the delay is within acceptable limits
func validateDelay(delay time.Duration) (time.Duration, error) {
	if delay < 0 {
		return 0, nil // Negative delays are treated as no delay
	}

	if delay > constants.MaxDelayDuration {
		return constants.MaxDelayDuration, nil // Cap at maximum allowed delay
	}

	return delay, nil
}
