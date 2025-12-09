package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/austrian-business-infrastructure/fo/pkg/cache"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	redis      *cache.Client
	requests   int
	window     time.Duration
	keyPrefix  string
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redis *cache.Client, requests int, window time.Duration, keyPrefix string) *RateLimiter {
	return &RateLimiter{
		redis:     redis,
		requests:  requests,
		window:    window,
		keyPrefix: keyPrefix,
	}
}

// RateLimitConfig contains rate limit configuration
type RateLimitConfig struct {
	RequestsPerMinute int
	LoginPerMinute    int
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerMinute: 100,
		LoginPerMinute:    5,
	}
}

// Limit returns middleware that applies rate limiting
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get identifier (IP or tenant ID)
		identifier := rl.getIdentifier(r)

		// Check rate limit
		key := rl.keyPrefix + ":" + identifier + ":" + currentWindow(rl.window)

		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()

		count, err := rl.redis.IncrementRateLimit(ctx, key, rl.window)
		if err != nil {
			// On error, allow request (fail open)
			next.ServeHTTP(w, r)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.requests))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, rl.requests-int(count))))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(nextWindow(rl.window).Unix(), 10))

		if count > int64(rl.requests) {
			retryAfter := int(time.Until(nextWindow(rl.window)).Seconds())
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			JSONError(w, http.StatusTooManyRequests, "Rate limit exceeded", ErrCodeRateLimited)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LimitByKey returns middleware that applies rate limiting by a custom key
func (rl *RateLimiter) LimitByKey(keyFunc func(*http.Request) string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identifier := keyFunc(r)
			key := rl.keyPrefix + ":" + identifier + ":" + currentWindow(rl.window)

			ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
			defer cancel()

			count, err := rl.redis.IncrementRateLimit(ctx, key, rl.window)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.requests))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, rl.requests-int(count))))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(nextWindow(rl.window).Unix(), 10))

			if count > int64(rl.requests) {
				retryAfter := int(time.Until(nextWindow(rl.window)).Seconds())
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				JSONError(w, http.StatusTooManyRequests, "Rate limit exceeded", ErrCodeRateLimited)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) getIdentifier(r *http.Request) string {
	// Prefer tenant ID if authenticated
	if tenantID := GetTenantID(r.Context()); tenantID != "" {
		return "tenant:" + tenantID
	}

	// Fall back to IP address
	return "ip:" + getClientIP(r)
}

func currentWindow(window time.Duration) string {
	return strconv.FormatInt(time.Now().Truncate(window).Unix(), 10)
}

func nextWindow(window time.Duration) time.Time {
	return time.Now().Truncate(window).Add(window)
}

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	return r.RemoteAddr
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
