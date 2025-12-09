package ai

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	tokens         int
	maxTokens      int
	refillRate     int // tokens per minute
	lastRefill     time.Time
	mu             sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		tokens:     requestsPerMinute,
		maxTokens:  requestsPerMinute,
		refillRate: requestsPerMinute,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available or context is cancelled
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		r.mu.Lock()
		r.refill()

		if r.tokens > 0 {
			r.tokens--
			r.mu.Unlock()
			return nil
		}

		// Calculate time until next token
		tokensNeeded := 1
		waitDuration := time.Duration(float64(tokensNeeded) / float64(r.refillRate) * float64(time.Minute))
		r.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
		}
	}
}

// refill adds tokens based on time elapsed
func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)

	// Calculate tokens to add based on elapsed time
	tokensToAdd := int(elapsed.Minutes() * float64(r.refillRate))

	if tokensToAdd > 0 {
		r.tokens += tokensToAdd
		if r.tokens > r.maxTokens {
			r.tokens = r.maxTokens
		}
		r.lastRefill = now
	}
}

// Available returns the number of available tokens
func (r *RateLimiter) Available() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refill()
	return r.tokens
}

// TokenTracker tracks token usage for cost monitoring
type TokenTracker struct {
	mu           sync.Mutex
	inputTokens  int
	outputTokens int
	requests     int
	startTime    time.Time
}

// NewTokenTracker creates a new token tracker
func NewTokenTracker() *TokenTracker {
	return &TokenTracker{
		startTime: time.Now(),
	}
}

// Track records token usage
func (t *TokenTracker) Track(input, output int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.inputTokens += input
	t.outputTokens += output
	t.requests++
}

// Stats returns current usage statistics
func (t *TokenTracker) Stats() TokenStats {
	t.mu.Lock()
	defer t.mu.Unlock()
	return TokenStats{
		InputTokens:  t.inputTokens,
		OutputTokens: t.outputTokens,
		TotalTokens:  t.inputTokens + t.outputTokens,
		Requests:     t.requests,
		Duration:     time.Since(t.startTime),
		CostCents:    EstimateCost(t.inputTokens, t.outputTokens),
	}
}

// Reset resets the tracker
func (t *TokenTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.inputTokens = 0
	t.outputTokens = 0
	t.requests = 0
	t.startTime = time.Now()
}

// TokenStats holds token usage statistics
type TokenStats struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	Requests     int
	Duration     time.Duration
	CostCents    int
}
