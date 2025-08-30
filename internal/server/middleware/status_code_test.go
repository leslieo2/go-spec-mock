package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leslieo2/go-spec-mock/internal/constants"
	"go.uber.org/zap"
)

func TestStatusCodeMiddleware(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		shouldLogError bool
	}{
		{
			name:           "no status code parameter",
			query:          "",
			expectedStatus: constants.StatusOK,
			shouldLogError: false,
		},
		{
			name:           "valid status code",
			query:          "__statusCode=404",
			expectedStatus: 404,
			shouldLogError: false,
		},
		{
			name:           "another valid status code",
			query:          "__statusCode=201",
			expectedStatus: 201,
			shouldLogError: false,
		},
		{
			name:           "invalid status code (too low)",
			query:          "__statusCode=99",
			expectedStatus: constants.StatusOK,
			shouldLogError: true,
		},
		{
			name:           "invalid status code (too high)",
			query:          "__statusCode=600",
			expectedStatus: constants.StatusOK,
			shouldLogError: true,
		},
		{
			name:           "non-numeric status code",
			query:          "__statusCode=invalid",
			expectedStatus: constants.StatusOK,
			shouldLogError: true,
		},
		{
			name:           "status code with other parameters",
			query:          "__statusCode=500&name=test",
			expectedStatus: 500,
			shouldLogError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler that checks context
			handler := StatusCodeMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if status code was set in context
				if statusCode, ok := r.Context().Value(constants.ContextKeyStatusCode).(int); ok {
					if statusCode != tt.expectedStatus {
						t.Errorf("Expected status code %d in context, got %d", tt.expectedStatus, statusCode)
					}
				} else if tt.expectedStatus != constants.StatusOK {
					t.Error("Expected status code to be set in context but it wasn't")
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test?"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected handler to return 200, got %d", w.Code)
			}
		})
	}
}

func TestParseStatusCode(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"200", 200, false},
		{"404", 404, false},
		{"201", 201, false},
		{"500", 500, false},
		{"99", 0, true},      // Too low
		{"600", 0, true},     // Too high
		{"invalid", 0, true}, // Non-numeric
		{"", 0, true},        // Empty
		{"0", 0, true},       // Too low
		{"599", 599, false},  // Edge case
		{"100", 100, false},  // Edge case
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseStatusCode(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %s, but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %s: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("Expected %d, got %d for input %s", tt.expected, result, tt.input)
				}
			}
		})
	}
}

func TestInvalidStatusCodeError(t *testing.T) {
	err := &InvalidStatusCodeError{Code: 999}
	expectedMsg := "999 is not a valid HTTP status code"

	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestStatusCodeMiddlewareIntegration(t *testing.T) {
	logger := zap.NewNop()

	// Test that middleware properly integrates with context
	handler := StatusCodeMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should be called after the middleware processes the request
		w.WriteHeader(http.StatusOK)
	}))

	// Test with valid status code
	req := httptest.NewRequest("GET", "/test?__statusCode=404", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}
}
