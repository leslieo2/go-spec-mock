package security

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters *cache.Cache
	config   *RateLimitConfig
	clock    Clock
}

type RateLimitConfig struct {
	Enabled         bool                  `json:"enabled" yaml:"enabled"`
	Strategy        string                `json:"strategy" yaml:"strategy"` // "api_key", "ip", "both"
	Global          *GlobalRateLimit      `json:"global" yaml:"global"`
	ByAPIKey        map[string]*RateLimit `json:"by_api_key" yaml:"by_api_key"`
	ByIP            *RateLimit            `json:"by_ip" yaml:"by_ip"`
	CleanupInterval time.Duration         `json:"cleanup_interval" yaml:"cleanup_interval"`
	MaxCacheSize    int                   `json:"max_cache_size" yaml:"max_cache_size"`
}

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now()
}

func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}

	// Set a maximum cache size to prevent memory exhaustion during DDoS attacks
	maxCacheSize := config.MaxCacheSize
	if maxCacheSize == 0 {
		maxCacheSize = 10000 // Default to 10,000 unique identifiers
	}

	rl := &RateLimiter{
		limiters: cache.New(config.CleanupInterval, config.CleanupInterval*2),
		config:   config,
		clock:    RealClock{},
	}

	// Set up periodic cleanup to enforce max cache size
	go rl.periodicCleanup(maxCacheSize)

	return rl
}

// periodicCleanup periodically cleans up the cache to prevent memory exhaustion
func (rl *RateLimiter) periodicCleanup(maxSize int) {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		currentSize := rl.limiters.ItemCount()
		if currentSize <= maxSize {
			continue // No cleanup needed
		}

		// Calculate how many entries to remove
		toRemove := currentSize - maxSize + int(float64(maxSize)*0.1) // Remove extra 10% to avoid frequent cleanup

		// Since go-cache doesn't provide access timestamps, we'll use a simple eviction strategy
		// Remove entries randomly (simple but effective for DDoS protection)
		keys := make([]string, 0, currentSize)
		for key := range rl.limiters.Items() {
			keys = append(keys, key)
		}

		// Shuffle and remove
		for i := range keys {
			j := i + int(time.Now().UnixNano()%int64(len(keys)-i))
			if j < len(keys) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}

		// Remove the calculated number of keys
		for i := 0; i < toRemove && i < len(keys); i++ {
			rl.limiters.Delete(keys[i])
		}
	}
}

func (rl *RateLimiter) Allow(identifier string, limit *RateLimit) bool {
	if !rl.config.Enabled {
		return true
	}

	// Use identifier as cache key, no time window suffix needed
	key := identifier

	var limiter *rate.Limiter

	// Try to get existing limiter from cache
	if item, found := rl.limiters.Get(key); found {
		limiter = item.(*rate.Limiter)
	} else {
		// Create new limiter with proper rate and burst
		limiter = rate.NewLimiter(rate.Limit(limit.RequestsPerSecond), limit.BurstSize)
		rl.limiters.Set(key, limiter, cache.DefaultExpiration)
	}

	return limiter.Allow()
}

func (rl *RateLimiter) GetRateLimitStatus(identifier string, limit *RateLimit) (*RateLimitStatus, error) {
	if !rl.config.Enabled {
		return &RateLimitStatus{
			Limit:      limit.RequestsPerSecond,
			Remaining:  limit.BurstSize,
			Reset:      time.Now().Add(time.Minute),
			RetryAfter: 0,
		}, nil
	}

	key := identifier

	var limiter *rate.Limiter

	// Get or create the limiter
	if item, found := rl.limiters.Get(key); found {
		limiter = item.(*rate.Limiter)
	} else {
		limiter = rate.NewLimiter(rate.Limit(limit.RequestsPerSecond), limit.BurstSize)
		rl.limiters.Set(key, limiter, cache.DefaultExpiration)
	}

	// Get current limit status
	remaining := float64(limit.BurstSize) - float64(limiter.Burst()) + limiter.Tokens()
	if remaining < 0 {
		remaining = 0
	}

	// Calculate reset time and retry after
	now := time.Now()
	resetTime := now.Add(time.Minute)
	retryAfter := time.Duration(0)

	// For rate.Limiter, we can't get exact reset time, so we use a reasonable estimate
	if !limiter.Allow() {
		// If we're rate limited, estimate retry after based on rate
		retryAfter = time.Duration(float64(time.Second) * float64(limit.BurstSize) / float64(limit.RequestsPerSecond))
	}

	return &RateLimitStatus{
		Limit:      limit.BurstSize,
		Remaining:  int(remaining),
		Reset:      resetTime,
		RetryAfter: retryAfter,
	}, nil
}

type RateLimitStatus struct {
	Limit      int           `json:"limit"`
	Remaining  int           `json:"remaining"`
	Reset      time.Time     `json:"reset"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Skip rate limiting for health and metrics endpoints
		if rl.shouldSkipRateLimit(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Get identifier based on strategy
		identifier := rl.getIdentifier(r)

		// Get applicable rate limit
		limit := rl.getRateLimit(identifier)

		// Check rate limit
		if !rl.Allow(identifier, limit) {
			status, err := rl.GetRateLimitStatus(identifier, limit)
			if err != nil {
				status = &RateLimitStatus{RetryAfter: limit.WindowSize}
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(status.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(status.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(status.Reset.Unix(), 10))

			if status.RetryAfter > 0 {
				w.Header().Set("Retry-After", strconv.Itoa(int(status.RetryAfter.Seconds())))
			}

			w.WriteHeader(http.StatusTooManyRequests)

			response := map[string]interface{}{
				"error":       "RATE_LIMIT_EXCEEDED",
				"message":     fmt.Sprintf("Rate limit exceeded. Try again in %v", status.RetryAfter),
				"code":        "RATE_LIMIT_EXCEEDED",
				"retry_after": int(status.RetryAfter.Seconds()),
			}
			jsonResponse, _ := json.Marshal(response)
			_, err = w.Write(jsonResponse)
			if err != nil {
				return
			}
			return
		}

		// Add rate limit headers
		status, _ := rl.GetRateLimitStatus(identifier, limit)
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(status.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(status.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(status.Reset.Unix(), 10))

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getIdentifier(r *http.Request) string {
	switch rl.config.Strategy {
	case "api_key":
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			return "api_key:" + apiKey
		}
		if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
			return "api_key:" + apiKey
		}
	case "both":
		// Combine API key and IP
		identifier := "ip:" + rl.getClientIP(r)
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			identifier += "|api_key:" + apiKey
		} else if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
			identifier += "|api_key:" + apiKey
		}
		return identifier
	default: // "ip"
		return "ip:" + rl.getClientIP(r)
	}

	return "ip:" + rl.getClientIP(r)
}

func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (rl *RateLimiter) getRateLimit(identifier string) *RateLimit {
	// Check for API key specific limits
	if strings.HasPrefix(identifier, "api_key:") {
		key := strings.TrimPrefix(identifier, "api_key:")
		if limit, exists := rl.config.ByAPIKey[key]; exists {
			return limit
		}
	}

	// Check for IP specific limits
	if strings.HasPrefix(identifier, "ip:") {
		if rl.config.ByIP != nil {
			return rl.config.ByIP
		}
	}

	// Use global limits
	return &RateLimit{
		RequestsPerSecond: rl.config.Global.RequestsPerSecond,
		BurstSize:         rl.config.Global.BurstSize,
		WindowSize:        rl.config.Global.WindowSize,
	}
}

func (rl *RateLimiter) shouldSkipRateLimit(path string) bool {
	skippedPaths := []string{
		"/health",
		"/ready",
		"/metrics",
	}

	for _, skipped := range skippedPaths {
		if path == skipped {
			return true
		}
	}

	return false
}
