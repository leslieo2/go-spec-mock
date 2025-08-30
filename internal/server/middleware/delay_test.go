package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/constants"
	"go.uber.org/zap"
)

func TestDelayMiddleware(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	middleware := DelayMiddleware(logger)

	tests := []struct {
		name           string
		delayParam     string
		expectedMin    time.Duration
		expectedMax    time.Duration
		shouldDelay    bool
		expectErrorLog bool
	}{
		{
			name:        "valid milliseconds delay",
			delayParam:  "500ms",
			expectedMin: 450 * time.Millisecond,
			expectedMax: 550 * time.Millisecond,
			shouldDelay: true,
		},
		{
			name:        "valid seconds delay",
			delayParam:  "2s",
			expectedMin: 1900 * time.Millisecond,
			expectedMax: 2100 * time.Millisecond,
			shouldDelay: true,
		},
		{
			name:        "valid numeric delay (milliseconds)",
			delayParam:  "1000",
			expectedMin: 950 * time.Millisecond,
			expectedMax: 1050 * time.Millisecond,
			shouldDelay: true,
		},
		{
			name:           "invalid delay format",
			delayParam:     "invalid",
			shouldDelay:    false,
			expectErrorLog: true,
		},
		{
			name:        "no delay parameter",
			delayParam:  "",
			shouldDelay: false,
		},
		{
			name:        "delay exceeds maximum",
			delayParam:  "1m", // 1 minute > 30 second max
			expectedMin: 29 * time.Second,
			expectedMax: 31 * time.Second,
			shouldDelay: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.delayParam != "" {
				q := req.URL.Query()
				q.Add(constants.QueryParamDelay, tt.delayParam)
				req.URL.RawQuery = q.Encode()
			}

			// Create test response writer
			w := httptest.NewRecorder()

			// Create a simple handler that records when it's called
			handlerCalled := false
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			// Apply middleware and measure time
			start := time.Now()
			middleware(handler).ServeHTTP(w, req)
			elapsed := time.Since(start)

			// Verify results
			if tt.shouldDelay {
				if elapsed < tt.expectedMin || elapsed > tt.expectedMax {
					t.Errorf("Expected delay between %v and %v, got %v", tt.expectedMin, tt.expectedMax, elapsed)
				}
				if !handlerCalled {
					t.Error("Handler should have been called after delay")
				}
			} else {
				if elapsed > 100*time.Millisecond {
					t.Errorf("Expected no significant delay, got %v", elapsed)
				}
				if !handlerCalled {
					t.Error("Handler should have been called")
				}
			}

			// Check response status
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		})
	}
}

func TestParseDelay(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		hasError bool
	}{
		{"valid milliseconds", "500ms", 500 * time.Millisecond, false},
		{"valid seconds", "2s", 2 * time.Second, false},
		{"valid numeric", "1000", 1000 * time.Millisecond, false},
		{"negative delay", "-100ms", 0, false},
		{"invalid format", "invalid", 0, true},
		{"empty string", "", 0, true},
		{"whitespace", "  500ms  ", 500 * time.Millisecond, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDelay(tt.input)

			if tt.hasError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestValidateDelay(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected time.Duration
	}{
		{"within limit", 5 * time.Second, 5 * time.Second},
		{"exceeds limit", 40 * time.Second, constants.MaxDelayDuration},
		{"negative", -1 * time.Second, 0},
		{"zero", 0, 0},
		{"exactly max", constants.MaxDelayDuration, constants.MaxDelayDuration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateDelay(tt.input)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
