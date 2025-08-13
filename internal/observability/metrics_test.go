package observability

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()
	
	if metrics == nil {
		t.Fatal("NewMetrics() returned nil")
	}

	// Verify all metrics are initialized
	if metrics.RequestCount == nil {
		t.Error("RequestCount metric is nil")
	}
	if metrics.RequestDuration == nil {
		t.Error("RequestDuration metric is nil")
	}
	if metrics.RequestSize == nil {
		t.Error("RequestSize metric is nil")
	}
	if metrics.ResponseSize == nil {
		t.Error("ResponseSize metric is nil")
	}
	if metrics.ActiveConnections == nil {
		t.Error("ActiveConnections metric is nil")
	}
	if metrics.HealthStatus == nil {
		t.Error("HealthStatus metric is nil")
	}
}

func TestMetrics_RecordRequest(t *testing.T) {
	metrics := NewMetrics()
	
	method := "GET"
	endpoint := "/test"
	statusCode := 200
	duration := 100 * time.Millisecond
	requestSize := int64(1234)
	responseSize := int64(5678)

	metrics.RecordRequest(method, endpoint, statusCode, duration, requestSize, responseSize)

	// Verify metrics are initialized
	if metrics.RequestCount == nil {
		t.Error("Expected RequestCount metric to be initialized")
	}

	// Verify RequestDuration metric
	if metrics.RequestDuration == nil {
		t.Error("Expected RequestDuration metric to be initialized")
	}

	// Verify RequestSize metric
	if metrics.RequestSize == nil {
		t.Error("Expected RequestSize metric to be initialized")
	}

	// Verify ResponseSize metric
	if metrics.ResponseSize == nil {
		t.Error("Expected ResponseSize metric to be initialized")
	}
}

func TestMetrics_SetHealthStatus(t *testing.T) {
	metrics := NewMetrics()

	// Test setting health status to true
	metrics.SetHealthStatus(true)
	
	// Verify health status gauge is initialized
	if metrics.HealthStatus == nil {
		t.Error("Expected HealthStatus metric to be initialized")
	}

	// Test setting health status to false
	metrics.SetHealthStatus(false)
	
	// Just verify no panic occurs
	_ = metrics.HealthStatus
}

func TestMetrics_Handler(t *testing.T) {
	metrics := NewMetrics()
	
	handler := metrics.Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil")
	}

	// Test the handler serves metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected Content-Type to contain 'text/plain', got %s", contentType)
	}
}

func TestMetrics_Labels(t *testing.T) {
	metrics := NewMetrics()
	
	// Test with various status codes
	testCases := []struct {
		statusCode int
		expected   string
	}{
		{200, "200"},
		{404, "404"},
		{500, "500"},
	}

	for _, tc := range testCases {
		metrics.RecordRequest("GET", "/test", tc.statusCode, 100*time.Millisecond, 0, 0)
		
		// Just verify no panic occurs
		_ = metrics.RequestCount
	}
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
	metrics := NewMetrics()
	
	done := make(chan bool)
	
	// Run concurrent metric updates
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				metrics.RecordRequest(
					"GET",
					"/endpoint-"+strconv.Itoa(id),
					200,
					10*time.Millisecond,
					int64(j*100),
					int64(j*200),
				)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Just verify no panic occurs
	_ = metrics.RequestCount
}