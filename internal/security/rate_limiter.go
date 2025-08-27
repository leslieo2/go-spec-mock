package security

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
	"github.com/leslieo2/go-spec-mock/internal/constants"
	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters       *cache.Cache
	securityConfig *config.SecurityConfig
	clock          Clock
}

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now()
}

func NewRateLimiter(securityConfig *config.SecurityConfig) *RateLimiter {
	rl := &RateLimiter{
		limiters:       cache.New(constants.RateLimitCleanupInterval, constants.RateLimitCleanupInterval*2),
		securityConfig: securityConfig,
		clock:          RealClock{},
	}

	// Set up periodic cleanup to enforce max cache size
	go rl.periodicCleanup()

	return rl
}

// periodicCleanup periodically cleans up the cache to prevent memory exhaustion
func (rl *RateLimiter) periodicCleanup() {
	ticker := time.NewTicker(constants.RateLimitCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		currentSize := rl.limiters.ItemCount()
		maxSize := constants.RateLimitMaxCacheSize // Get maxSize from constants
		if currentSize <= maxSize {
			continue // No cleanup needed
		}

		// Calculate how many entries to remove
		toRemove := currentSize - constants.RateLimitMaxCacheSize + int(float64(constants.RateLimitMaxCacheSize)*0.1) // Remove extra 10% to avoid frequent cleanup

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

func (rl *RateLimiter) Allow(identifier string, limit *config.RateLimit) bool {
	if !rl.securityConfig.RateLimit.Enabled {
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

func (rl *RateLimiter) GetRateLimitStatus(identifier string, limit *config.RateLimit) (*RateLimitStatus, error) {
	if !rl.securityConfig.RateLimit.Enabled {
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

func (rl *RateLimiter) sendRateLimitResponse(w http.ResponseWriter, _ *http.Request, identifier string, limit *config.RateLimit) {
	status, err := rl.GetRateLimitStatus(identifier, limit)
	if err != nil {
		status = &RateLimitStatus{RetryAfter: limit.WindowSize}
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.Header().Set(constants.HeaderXRateLimitLimit, strconv.Itoa(status.Limit))
	w.Header().Set(constants.HeaderXRateLimitRemaining, strconv.Itoa(status.Remaining))
	w.Header().Set(constants.HeaderXRateLimitReset, strconv.FormatInt(status.Reset.Unix(), 10))

	if status.RetryAfter > 0 {
		w.Header().Set(constants.HeaderRetryAfter, strconv.Itoa(int(status.RetryAfter.Seconds())))
	}

	w.WriteHeader(constants.StatusTooManyRequests)

	response := map[string]interface{}{
		"error":       constants.ErrorCodeRateLimitExceeded,
		"message":     fmt.Sprintf("Rate limit exceeded. Try again in %v", status.RetryAfter),
		"code":        constants.ErrorCodeRateLimitExceeded,
		"retry_after": int(status.RetryAfter.Seconds()),
	}
	jsonResponse, _ := json.Marshal(response)
	_, err = w.Write(jsonResponse)
	if err != nil {
		return
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.securityConfig.RateLimit.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Skip rate limiting for health and metrics endpoints
		if rl.shouldSkipRateLimit(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Layered Rate Limiting - Onion Model (API Key → IP → Global)
		var (
			effectiveLimit *config.RateLimit
			identifier     string
		)

		// 2. IP Rate Limit (medium priority - if strategy includes IP)
		if rl.securityConfig.RateLimit.Strategy == constants.RateLimitStrategyIP {
			ip := rl.getClientIP(r)
			if rl.securityConfig.RateLimit.ByIP != nil {
				if !rl.Allow("ip:"+ip, rl.securityConfig.RateLimit.ByIP) {
					rl.sendRateLimitResponse(w, r, "ip:"+ip, rl.securityConfig.RateLimit.ByIP)
					return
				}
				// Only set effective limit if no API key limit was applied
				if effectiveLimit == nil {
					effectiveLimit = rl.securityConfig.RateLimit.ByIP
					identifier = "ip:" + ip
				}
			}
		}

		// 3. Global Rate Limit (lowest priority - always checked as safety net)
		if rl.securityConfig.RateLimit.Global != nil {
			if !rl.Allow("global", rl.securityConfig.RateLimit.Global) {
				rl.sendRateLimitResponse(w, r, "global", rl.securityConfig.RateLimit.Global)
				return
			}
			// Only set effective limit if no more specific limit was applied
			if effectiveLimit == nil {
				effectiveLimit = rl.securityConfig.RateLimit.Global
				identifier = "global"
			}
		}

		// Set rate limit headers based on the most specific limit that was applied
		if effectiveLimit != nil {
			status, _ := rl.GetRateLimitStatus(identifier, effectiveLimit)
			w.Header().Set(constants.HeaderXRateLimitLimit, strconv.Itoa(status.Limit))
			w.Header().Set(constants.HeaderXRateLimitRemaining, strconv.Itoa(status.Remaining))
			w.Header().Set(constants.HeaderXRateLimitReset, strconv.FormatInt(status.Reset.Unix(), 10))
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get(constants.HeaderXForwardedFor)
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get(constants.HeaderXRealIP)
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

func (rl *RateLimiter) shouldSkipRateLimit(path string) bool {
	skippedPaths := []string{
		constants.PathHealth,
		constants.PathReady,
	}

	for _, skipped := range skippedPaths {
		if path == skipped {
			return true
		}
	}

	return false
}
