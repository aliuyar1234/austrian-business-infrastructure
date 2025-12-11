// Package ai provides Claude API client for document intelligence
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"austrian-business-infrastructure/internal/constants"
)

const (
	claudeAPIURL = "https://api.anthropic.com/v1/messages"
	apiVersion   = "2023-06-01"
)

// Client is a Claude API client with rate limiting and retry logic
type Client struct {
	apiKey      string
	model       string
	maxTokens   int
	httpClient  *http.Client
	rateLimiter *RateLimiter
	mu          sync.Mutex
}

// ClientConfig holds Claude client configuration
type ClientConfig struct {
	APIKey         string
	Model          string
	MaxTokens      int
	RateLimitPerMin int
	Timeout        time.Duration
}

// Message represents a Claude API message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request represents a Claude API request
type Request struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	System      string    `json:"system,omitempty"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Response represents a Claude API response
type Response struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Role         string        `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string        `json:"model"`
	StopReason   string        `json:"stop_reason"`
	StopSequence string        `json:"stop_sequence,omitempty"`
	Usage        Usage         `json:"usage"`
}

// ContentBlock represents a content block in the response
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Usage represents token usage
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ErrorResponse represents a Claude API error
type ErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewClient creates a new Claude API client
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if cfg.Model == "" {
		cfg.Model = "claude-sonnet-4-20250514"
	}

	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 4096
	}

	if cfg.RateLimitPerMin == 0 {
		cfg.RateLimitPerMin = 60
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = constants.AIClientTimeout
	}

	return &Client{
		apiKey:    cfg.APIKey,
		model:     cfg.Model,
		maxTokens: cfg.MaxTokens,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		rateLimiter: NewRateLimiter(cfg.RateLimitPerMin),
	}, nil
}

// Complete sends a completion request to Claude API
func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string, temperature float64) (*Response, error) {
	return c.CompleteWithRetry(ctx, systemPrompt, userPrompt, temperature, 3)
}

// CompleteWithRetry sends a completion request with retry logic
func (c *Client) CompleteWithRetry(ctx context.Context, systemPrompt, userPrompt string, temperature float64, maxRetries int) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Wait for rate limiter
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		resp, err := c.doRequest(ctx, systemPrompt, userPrompt, temperature)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, err
		}

		// Exponential backoff
		backoff := time.Duration(1<<attempt) * time.Second
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *Client) doRequest(ctx context.Context, systemPrompt, userPrompt string, temperature float64) (*Response, error) {
	req := Request{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System:    systemPrompt,
		Messages: []Message{
			{Role: "user", Content: userPrompt},
		},
		Temperature: temperature,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", apiVersion)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Type:       errResp.Error.Type,
				Message:    errResp.Error.Message,
			}
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	var claudeResp Response
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &claudeResp, nil
}

// GetText extracts the text content from the response
func (r *Response) GetText() string {
	if len(r.Content) == 0 {
		return ""
	}
	return r.Content[0].Text
}

// TotalTokens returns the total token count
func (r *Response) TotalTokens() int {
	return r.Usage.InputTokens + r.Usage.OutputTokens
}

// APIError represents a Claude API error
type APIError struct {
	StatusCode int
	Type       string
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("claude API error (status %d, type %s): %s", e.StatusCode, e.Type, e.Message)
}

func isRetryableError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		// Retry on rate limit or server errors
		return apiErr.StatusCode == 429 || apiErr.StatusCode >= 500
	}
	return false
}

// EstimateCost estimates the cost in cents based on token usage
// Using approximate pricing for Claude Sonnet
func EstimateCost(inputTokens, outputTokens int) int {
	// Claude Sonnet pricing (approximate):
	// Input: $3 per 1M tokens = $0.000003 per token
	// Output: $15 per 1M tokens = $0.000015 per token
	inputCost := float64(inputTokens) * 0.000003
	outputCost := float64(outputTokens) * 0.000015
	totalCents := (inputCost + outputCost) * 100

	// Round up to nearest cent
	if totalCents < 1 {
		return 1
	}
	return int(totalCents + 0.5)
}
