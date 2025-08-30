package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestExampleMiddleware(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name         string
		query        string
		expectedName string
		shouldBeSet  bool
	}{
		{
			name:         "no example parameter",
			query:        "",
			expectedName: "",
			shouldBeSet:  false,
		},
		{
			name:         "valid example name",
			query:        "__example=success",
			expectedName: "success",
			shouldBeSet:  true,
		},
		{
			name:         "empty example name",
			query:        "__example=",
			expectedName: "",
			shouldBeSet:  true,
		},
		{
			name:         "example name with other parameters",
			query:        "__example=premium&name=test",
			expectedName: "premium",
			shouldBeSet:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler that checks context
			handler := ExampleMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if example name was set in context
				exampleName := GetExampleNameFromContext(r)
				if tt.shouldBeSet && exampleName != tt.expectedName {
					t.Errorf("Expected example name '%s' in context, got '%s'", tt.expectedName, exampleName)
				}
				if !tt.shouldBeSet && exampleName != "" {
					t.Error("Expected no example name in context but found one")
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

func TestGetExampleNameFromContext(t *testing.T) {
	// Test with no example name in context
	req := httptest.NewRequest("GET", "/test", nil)

	if name := GetExampleNameFromContext(req); name != "" {
		t.Errorf("Expected empty example name, got '%s'", name)
	}

	// Test with example name in context
	reqWithExample := httptest.NewRequest("GET", "/test?__example=test", nil)
	handler := ExampleMiddleware(zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if name := GetExampleNameFromContext(r); name != "test" {
			t.Errorf("Expected example name 'test', got '%s'", name)
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, reqWithExample)
}
