package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"austrian-business-infrastructure/pkg/cache"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	redis          *cache.Client
	requests       int
	window         time.Duration
	keyPrefix      string
	trustedProxies map[string]bool // Trusted proxy IPs for X-Forwarded-For validation
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
			// Fail-closed: reject requests when Redis is unavailable to prevent abuse during outages
			JSONError(w, http.StatusServiceUnavailable, "Service temporarily unavailable", ErrCodeServiceUnavailable)
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
				// Fail-closed: reject requests when Redis is unavailable to prevent abuse during outages
				JSONError(w, http.StatusServiceUnavailable, "Service temporarily unavailable", ErrCodeServiceUnavailable)
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

	// Fall back to IP address with trusted proxy validation
	return "ip:" + getClientIPWithTrustedProxies(r, rl.trustedProxies)
}

func currentWindow(window time.Duration) string {
	return strconv.FormatInt(time.Now().Truncate(window).Unix(), 10)
}

func nextWindow(window time.Duration) time.Time {
	return time.Now().Truncate(window).Add(window)
}

// getClientIP extracts client IP from request with trusted proxy validation
// This prevents IP spoofing attacks (CWE-290) by only trusting X-Forwarded-For
// from known proxy IPs (e.g., Caddy, Traefik, load balancers)
func getClientIP(r *http.Request) string {
	// Without trusted proxy configuration, always use RemoteAddr (secure default)
	return getClientIPWithTrustedProxies(r, nil)
}

// getClientIPWithTrustedProxies extracts client IP with trusted proxy validation
func getClientIPWithTrustedProxies(r *http.Request, trustedProxies map[string]bool) string {
	// Extract remote IP (strip port if present)
	remoteAddr := r.RemoteAddr
	if idx := lastIndexByte(remoteAddr, ':'); idx != -1 {
		// Check if this is IPv6 [::1]:port format
		if len(remoteAddr) > 0 && remoteAddr[0] == '[' {
			if bracketIdx := lastIndexByte(remoteAddr, ']'); bracketIdx != -1 {
				remoteAddr = remoteAddr[1:bracketIdx]
			}
		} else {
			remoteAddr = remoteAddr[:idx]
		}
	}

	// Only trust forwarded headers if request comes from trusted proxy
	// If no trusted proxies configured, always use RemoteAddr (secure default)
	if len(trustedProxies) == 0 || !trustedProxies[remoteAddr] {
		return remoteAddr
	}

	// Request is from trusted proxy - check X-Forwarded-For
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Parse X-Forwarded-For: client, proxy1, proxy2
		// Get the rightmost non-trusted IP (real client)
		ips := splitXFF(xff)
		for i := len(ips) - 1; i >= 0; i-- {
			ip := ips[i]
			if !trustedProxies[ip] {
				return ip
			}
		}
	}

	// Check X-Real-IP (set by some proxies like nginx)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return trimSpace(xri)
	}

	return remoteAddr
}

// lastIndexByte returns the index of the last instance of c in s, or -1
func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// splitXFF splits X-Forwarded-For header and trims whitespace
func splitXFF(xff string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(xff); i++ {
		if i == len(xff) || xff[i] == ',' {
			ip := trimSpace(xff[start:i])
			if ip != "" {
				result = append(result, ip)
			}
			start = i + 1
		}
	}
	return result
}

// trimSpace removes leading/trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// SetTrustedProxies configures which proxy IPs are trusted for X-Forwarded-For
// Common values: "127.0.0.1", "::1", "10.0.0.0/8", Docker network IPs
func (rl *RateLimiter) SetTrustedProxies(proxies []string) {
	rl.trustedProxies = make(map[string]bool)
	for _, p := range proxies {
		rl.trustedProxies[p] = true
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
