package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	URL          string
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultRedisConfig returns sensible defaults for Redis connection
func DefaultRedisConfig(url string) *RedisConfig {
	return &RedisConfig{
		URL:          url,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// Client wraps redis.Client with additional functionality
type Client struct {
	*redis.Client
}

// NewClient creates a new Redis client
func NewClient(ctx context.Context, cfg *RedisConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("redis URL is required")
	}

	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	// Apply pool settings
	opt.PoolSize = cfg.PoolSize
	opt.MinIdleConns = cfg.MinIdleConns
	opt.DialTimeout = cfg.DialTimeout
	opt.ReadTimeout = cfg.ReadTimeout
	opt.WriteTimeout = cfg.WriteTimeout

	client := redis.NewClient(opt)

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &Client{Client: client}, nil
}

// Close closes the Redis client
func (c *Client) Close() error {
	if c.Client != nil {
		return c.Client.Close()
	}
	return nil
}

// Health checks if the Redis connection is healthy
func (c *Client) Health(ctx context.Context) error {
	return c.Ping(ctx).Err()
}

// Session operations for authentication

// SetSession stores a session with TTL
func (c *Client) SetSession(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.Set(ctx, "session:"+key, value, ttl).Err()
}

// GetSession retrieves a session
func (c *Client) GetSession(ctx context.Context, key string) (string, error) {
	return c.Get(ctx, "session:"+key).Result()
}

// DeleteSession removes a session
func (c *Client) DeleteSession(ctx context.Context, key string) error {
	return c.Del(ctx, "session:"+key).Err()
}

// RefreshSession extends session TTL
func (c *Client) RefreshSession(ctx context.Context, key string, ttl time.Duration) error {
	return c.Expire(ctx, "session:"+key, ttl).Err()
}

// Rate limiting operations

// RateLimitKey generates a rate limit key for a given identifier
func RateLimitKey(identifier string, window string) string {
	return fmt.Sprintf("ratelimit:%s:%s", identifier, window)
}

// IncrementRateLimit increments a rate limit counter
func (c *Client) IncrementRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := c.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

// GetRateLimit gets current rate limit count
func (c *Client) GetRateLimit(ctx context.Context, key string) (int64, error) {
	val, err := c.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}
