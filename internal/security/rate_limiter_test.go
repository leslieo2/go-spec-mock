package security

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

// MockClock allows controlling time in tests
type MockClock struct {
	mu  sync.Mutex
	now time.Time
}

func (mc *MockClock) Now() time.Time {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.now
}

func (mc *MockClock) Advance(d time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.now = mc.now.Add(d)
}

func newTestRateLimiter(config *RateLimitConfig) (*RateLimiter, *MockClock) {
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}

	mockClock := &MockClock{now: time.Now()}
	rl := &RateLimiter{
		limiters: cache.New(config.CleanupInterval, config.CleanupInterval*2),
		config:   config,
		clock:    mockClock,
	}
	// Do not start cleanup goroutine in tests
	return rl, mockClock
}

func TestRateLimiter_Success(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:  true,
		Strategy: "ip",
		ByIP: &RateLimit{
			RequestsPerSecond: 5,
			BurstSize:         10,
			WindowSize:        time.Second,
		},
	}
	rl, _ := newTestRateLimiter(config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	rr := httptest.NewRecorder()

	middleware := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Request within limit should succeed")
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Limit"), "Should have X-RateLimit-Limit header")
	assert.Equal(t, "9", rr.Header().Get("X-RateLimit-Remaining"), "Remaining requests should be correct")
}

func TestRateLimiter_Failure_Exceeded(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:  true,
		Strategy: "ip",
		ByIP: &RateLimit{
			RequestsPerSecond: 2,
			BurstSize:         2,
			WindowSize:        time.Minute, // Use a long window to prevent reset during test
		},
	}
	rl, _ := newTestRateLimiter(config)

	middleware := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request should succeed
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.2:12345"
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code, "Request #1 should succeed")

	// Second request should fail (exceeds burst)
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.2:12345"
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "Request exceeding limit should fail")
	assert.NotEmpty(t, rr.Header().Get("Retry-After"), "Should have Retry-After header")
	assert.Contains(t, rr.Body.String(), "Rate limit exceeded", "Response body should contain error message")
}

func TestRateLimiter_PerIP(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:  true,
		Strategy: "ip",
		ByIP: &RateLimit{
			RequestsPerSecond: 1,
			BurstSize:         1,
			WindowSize:        time.Minute,
		},
	}
	rl, _ := newTestRateLimiter(config)

	middleware := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request from IP 1 - should succeed
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.0.2.3:12345"
	rr1 := httptest.NewRecorder()
	middleware.ServeHTTP(rr1, req1)
	assert.Equal(t, http.StatusOK, rr1.Code)

	// Second request from IP 1 - should fail
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.0.2.3:12345"
	rr2 := httptest.NewRecorder()
	middleware.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rr2.Code)

	// First request from IP 2 - should succeed
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "192.0.2.4:12345"
	rr3 := httptest.NewRecorder()
	middleware.ServeHTTP(rr3, req3)
	assert.Equal(t, http.StatusOK, rr3.Code)
}

func TestRateLimiter_Disabled(t *testing.T) {
	config := &RateLimitConfig{
		Enabled: false, // Rate limiting is disabled
	}
	rl, _ := newTestRateLimiter(config)

	middleware := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make multiple requests, all should pass
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.0.2.5:12345"
		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Empty(t, rr.Header().Get("X-RateLimit-Limit"), "Should not have rate limit headers when disabled")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:  true,
		Strategy: "ip",
		ByIP: &RateLimit{
			RequestsPerSecond: 1,
			BurstSize:         1,
			WindowSize:        time.Second, // 1 second window
		},
	}
	rl, _ := newTestRateLimiter(config)

	middleware := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request - should succeed
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.6:12345"
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Second request immediately - should fail
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// Wait for tokens to refill
	time.Sleep(1 * time.Second)

	// Third request after rate allows - should succeed
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "0", rr.Header().Get("X-RateLimit-Remaining"))
}

func TestGetClientIP(t *testing.T) {
	rl := &RateLimiter{}

	testCases := []struct {
		name    string
		headers map[string]string
		remote  string
		wantIP  string
	}{
		{
			name:    "From X-Forwarded-For",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.1, 198.51.100.2"},
			remote:  "192.0.2.1:12345",
			wantIP:  "203.0.113.1",
		},
		{
			name:    "From X-Real-IP",
			headers: map[string]string{"X-Real-IP": "203.0.113.2"},
			remote:  "192.0.2.1:12345",
			wantIP:  "203.0.113.2",
		},
		{
			name:    "From RemoteAddr",
			headers: map[string]string{},
			remote:  "203.0.113.3:12345",
			wantIP:  "203.0.113.3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tc.remote

			ip := rl.getClientIP(req)
			assert.Equal(t, tc.wantIP, ip)
		})
	}
}

func TestRateLimitStatusHeaders(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:  true,
		Strategy: "ip",
		ByIP: &RateLimit{
			RequestsPerSecond: 10,
			BurstSize:         10,
			WindowSize:        time.Minute,
		},
	}
	rl, clock := newTestRateLimiter(config)

	middleware := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.7:12345"
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	limit, _ := strconv.Atoi(rr.Header().Get("X-RateLimit-Limit"))
	remaining, _ := strconv.Atoi(rr.Header().Get("X-RateLimit-Remaining"))
	reset, _ := strconv.ParseInt(rr.Header().Get("X-RateLimit-Reset"), 10, 64)

	assert.Equal(t, 10, limit)
	assert.Equal(t, 9, remaining)
	assert.True(t, time.Unix(reset, 0).After(clock.Now()))
}
