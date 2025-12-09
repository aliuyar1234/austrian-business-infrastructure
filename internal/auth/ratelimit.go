package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrRateLimited indicates the request was rate limited
	ErrRateLimited = errors.New("rate limited: too many requests")
)

const (
	// RateLimitPrefix is the Redis key prefix for rate limiting
	RateLimitPrefix = "ratelimit:"
	// LoginRateLimitKey is the specific prefix for login rate limiting
	LoginRateLimitKey = RateLimitPrefix + "login:"
	// LoginRateLimitMax is the maximum login attempts per window (FR-106: 10/min per IP)
	LoginRateLimitMax = 10
	// LoginRateLimitWindow is the rate limit window duration
	LoginRateLimitWindow = time.Minute

	// UserRateLimitKey is the prefix for per-user rate limiting
	UserRateLimitKey = RateLimitPrefix + "user:"
	// APIRateLimitKey is the prefix for API endpoint rate limiting
	APIRateLimitKey = RateLimitPrefix + "api:"
)

// RateLimiter provides rate limiting functionality using Redis
type RateLimiter struct {
	client *redis.Client
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// RateLimitConfig contains configuration for a rate limit
type RateLimitConfig struct {
	// MaxRequests is the maximum number of requests allowed in the window
	MaxRequests int
	// Window is the time window for rate limiting
	Window time.Duration
	// KeyPrefix is the prefix for Redis keys
	KeyPrefix string
	// FailClosed if true, rejects requests when Redis is unavailable.
	// Default (false) allows requests through on Redis failure (fail-open).
	// For security-sensitive endpoints like login, set to true.
	FailClosed bool
}

// DefaultLoginRateLimitConfig returns the default config for login rate limiting.
// FailClosed is true by default for login to prevent brute-force during outages.
func DefaultLoginRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		MaxRequests: LoginRateLimitMax,
		Window:      LoginRateLimitWindow,
		KeyPrefix:   LoginRateLimitKey,
		FailClosed:  true, // Security-sensitive endpoint - fail closed
	}
}

// CheckLogin checks if a login attempt is allowed for the given IP.
// Returns nil if allowed, ErrRateLimited if blocked.
// Automatically increments the counter.
func (rl *RateLimiter) CheckLogin(ctx context.Context, ip string) error {
	return rl.Check(ctx, DefaultLoginRateLimitConfig(), ip)
}

// Check checks if a request is allowed based on the config and identifier.
// Returns nil if allowed, ErrRateLimited if blocked.
// Automatically increments the counter.
func (rl *RateLimiter) Check(ctx context.Context, config *RateLimitConfig, identifier string) error {
	key := config.KeyPrefix + identifier

	// Use a Lua script for atomic increment and check
	script := redis.NewScript(`
		local current = redis.call("INCR", KEYS[1])
		if current == 1 then
			redis.call("EXPIRE", KEYS[1], ARGV[1])
		end
		return current
	`)

	result, err := script.Run(ctx, rl.client, []string{key}, int(config.Window.Seconds())).Int64()
	if err != nil {
		return fmt.Errorf("rate limit check failed: %w", err)
	}

	if result > int64(config.MaxRequests) {
		return ErrRateLimited
	}

	return nil
}

// GetRemaining returns the remaining requests for an identifier
func (rl *RateLimiter) GetRemaining(ctx context.Context, config *RateLimitConfig, identifier string) (int, error) {
	key := config.KeyPrefix + identifier

	count, err := rl.client.Get(ctx, key).Int()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return config.MaxRequests, nil
		}
		return 0, err
	}

	remaining := config.MaxRequests - count
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// Reset resets the rate limit for an identifier
func (rl *RateLimiter) Reset(ctx context.Context, config *RateLimitConfig, identifier string) error {
	key := config.KeyPrefix + identifier
	return rl.client.Del(ctx, key).Err()
}

// RateLimitMiddleware creates HTTP middleware for rate limiting
func RateLimitMiddleware(limiter *RateLimiter, config *RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetClientIP(r)

			if err := limiter.Check(r.Context(), config, ip); err != nil {
				if errors.Is(err, ErrRateLimited) {
					w.Header().Set("Retry-After", fmt.Sprintf("%d", int(config.Window.Seconds())))
					http.Error(w, "Too many requests", http.StatusTooManyRequests)
					return
				}
				// Redis error occurred
				if config.FailClosed {
					// For security-sensitive endpoints, reject on backend failure
					http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
					return
				}
				// Fail-open: allow request through on Redis failure (default for non-sensitive endpoints)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClientIP extracts the client IP from the request.
// Checks X-Forwarded-For and X-Real-IP headers first.
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (may contain comma-separated list)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP (original client)
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return normalizeIP(xff[:i])
			}
		}
		return normalizeIP(xff)
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return normalizeIP(xri)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return normalizeIP(r.RemoteAddr)
	}
	return normalizeIP(ip)
}

// normalizeIP normalizes an IP address for rate limiting.
// Removes zone identifiers and normalizes IPv6.
func normalizeIP(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}

	// For IPv6, use /64 subnet for rate limiting
	// This prevents users from bypassing limits with IPv6 rotation
	if parsed.To4() == nil && parsed.To16() != nil {
		// Zero out the last 64 bits
		for i := 8; i < 16; i++ {
			parsed[i] = 0
		}
	}

	return parsed.String()
}

// LoginRateLimitInfo contains rate limit information for the login endpoint
type LoginRateLimitInfo struct {
	IP            string `json:"ip"`
	Remaining     int    `json:"remaining"`
	ResetInSec    int    `json:"reset_in_seconds"`
	IsRateLimited bool   `json:"is_rate_limited"`
}

// GetLoginRateLimitInfo returns rate limit information for an IP
func (rl *RateLimiter) GetLoginRateLimitInfo(ctx context.Context, ip string) (*LoginRateLimitInfo, error) {
	config := DefaultLoginRateLimitConfig()
	key := config.KeyPrefix + ip

	// Get current count
	count, err := rl.client.Get(ctx, key).Int()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	// Get TTL
	ttl, err := rl.client.TTL(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	remaining := config.MaxRequests - count
	if remaining < 0 {
		remaining = 0
	}

	resetIn := int(ttl.Seconds())
	if resetIn < 0 {
		resetIn = 0
	}

	return &LoginRateLimitInfo{
		IP:            ip,
		Remaining:     remaining,
		ResetInSec:    resetIn,
		IsRateLimited: remaining == 0,
	}, nil
}

// UserRateLimitConfig returns rate limit config for per-user API limits
func UserRateLimitConfig(maxRequests int, window time.Duration) *RateLimitConfig {
	return &RateLimitConfig{
		MaxRequests: maxRequests,
		Window:      window,
		KeyPrefix:   UserRateLimitKey,
		FailClosed:  false, // User-level limits can fail open
	}
}

// APIEndpointRateLimitConfig returns rate limit config for specific API endpoints
func APIEndpointRateLimitConfig(endpoint string, maxRequests int, window time.Duration, failClosed bool) *RateLimitConfig {
	return &RateLimitConfig{
		MaxRequests: maxRequests,
		Window:      window,
		KeyPrefix:   APIRateLimitKey + endpoint + ":",
		FailClosed:  failClosed,
	}
}

// CheckUserRate checks rate limit for a specific user
func (rl *RateLimiter) CheckUserRate(ctx context.Context, userID string, maxRequests int, window time.Duration) error {
	config := UserRateLimitConfig(maxRequests, window)
	return rl.Check(ctx, config, userID)
}

// CheckCombined checks both IP-based and user-based rate limits
// Returns error if either limit is exceeded
func (rl *RateLimiter) CheckCombined(ctx context.Context, ip, userID string, ipConfig, userConfig *RateLimitConfig) error {
	// Check IP-based limit first
	if ipConfig != nil {
		if err := rl.Check(ctx, ipConfig, ip); err != nil {
			return fmt.Errorf("ip rate limit: %w", err)
		}
	}

	// Check user-based limit
	if userConfig != nil && userID != "" {
		if err := rl.Check(ctx, userConfig, userID); err != nil {
			return fmt.Errorf("user rate limit: %w", err)
		}
	}

	return nil
}

// MultiLevelRateLimiter provides hierarchical rate limiting
type MultiLevelRateLimiter struct {
	limiter *RateLimiter
	levels  []RateLimitLevel
}

// RateLimitLevel defines a level in the rate limit hierarchy
type RateLimitLevel struct {
	Name       string
	Config     *RateLimitConfig
	KeyBuilder func(r *http.Request) string
}

// NewMultiLevelRateLimiter creates a rate limiter with multiple levels
func NewMultiLevelRateLimiter(limiter *RateLimiter, levels []RateLimitLevel) *MultiLevelRateLimiter {
	return &MultiLevelRateLimiter{
		limiter: limiter,
		levels:  levels,
	}
}

// Check checks all rate limit levels
func (m *MultiLevelRateLimiter) Check(ctx context.Context, r *http.Request) (string, error) {
	for _, level := range m.levels {
		key := level.KeyBuilder(r)
		if key == "" {
			continue
		}

		if err := m.limiter.Check(ctx, level.Config, key); err != nil {
			return level.Name, err
		}
	}
	return "", nil
}

// Middleware creates HTTP middleware for multi-level rate limiting
func (m *MultiLevelRateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			levelName, err := m.Check(r.Context(), r)
			if err != nil {
				if errors.Is(err, ErrRateLimited) {
					w.Header().Set("X-RateLimit-Exceeded", levelName)
					w.Header().Set("Retry-After", "60")
					http.Error(w, "Too many requests", http.StatusTooManyRequests)
					return
				}
				// Backend error - check if any level requires fail-closed
				for _, level := range m.levels {
					if level.Name == levelName && level.Config.FailClosed {
						http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
						return
					}
				}
				// Otherwise fail open
			}

			next.ServeHTTP(w, r)
		})
	}
}

// DefaultAPIRateLimits returns standard rate limit levels for API endpoints
func DefaultAPIRateLimits(limiter *RateLimiter) *MultiLevelRateLimiter {
	return NewMultiLevelRateLimiter(limiter, []RateLimitLevel{
		{
			Name: "global_ip",
			Config: &RateLimitConfig{
				MaxRequests: 1000,
				Window:      time.Minute,
				KeyPrefix:   APIRateLimitKey + "global:ip:",
				FailClosed:  false,
			},
			KeyBuilder: func(r *http.Request) string {
				return GetClientIP(r)
			},
		},
		{
			Name: "user",
			Config: &RateLimitConfig{
				MaxRequests: 100,
				Window:      time.Minute,
				KeyPrefix:   UserRateLimitKey,
				FailClosed:  false,
			},
			KeyBuilder: func(r *http.Request) string {
				// Extract user ID from context (set by auth middleware)
				if userID, ok := r.Context().Value("user_id").(string); ok {
					return userID
				}
				return ""
			},
		},
	})
}

// SensitiveEndpointRateLimits returns stricter rate limits for sensitive operations
func SensitiveEndpointRateLimits(limiter *RateLimiter, endpoint string) *MultiLevelRateLimiter {
	return NewMultiLevelRateLimiter(limiter, []RateLimitLevel{
		{
			Name: "sensitive_ip",
			Config: &RateLimitConfig{
				MaxRequests: 10,
				Window:      time.Minute,
				KeyPrefix:   APIRateLimitKey + endpoint + ":ip:",
				FailClosed:  true, // Fail closed for sensitive endpoints
			},
			KeyBuilder: func(r *http.Request) string {
				return GetClientIP(r)
			},
		},
		{
			Name: "sensitive_user",
			Config: &RateLimitConfig{
				MaxRequests: 5,
				Window:      time.Minute,
				KeyPrefix:   APIRateLimitKey + endpoint + ":user:",
				FailClosed:  true,
			},
			KeyBuilder: func(r *http.Request) string {
				if userID, ok := r.Context().Value("user_id").(string); ok {
					return userID
				}
				return ""
			},
		},
	})
}
