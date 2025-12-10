package atrust

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides access to the A-Trust Fernsignatur API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	sandbox    bool
	retryMax   int
}

// ClientOption is a functional option for configuring the client
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithRetryMax sets the maximum number of retries
func WithRetryMax(max int) ClientOption {
	return func(c *Client) {
		c.retryMax = max
	}
}

// WithSandbox enables sandbox mode
func WithSandbox(enabled bool) ClientOption {
	return func(c *Client) {
		c.sandbox = enabled
	}
}

// NewClient creates a new A-Trust API client
func NewClient(baseURL, apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL:  baseURL,
		apiKey:   apiKey,
		retryMax: 3,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Sign signs a document hash using the A-Trust Fernsignatur API
func (c *Client) Sign(ctx context.Context, req *SignRequest) (*SignResponse, error) {
	if req.HashAlgorithm == "" {
		req.HashAlgorithm = HashAlgoSHA256
	}

	// Validate hash format
	if _, err := hex.DecodeString(req.DocumentHash); err != nil {
		return nil, fmt.Errorf("invalid document hash: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp SignResponse
	if err := c.doRequest(ctx, "POST", "/sign", body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// BatchSign signs multiple document hashes in a single operation
func (c *Client) BatchSign(ctx context.Context, req *BatchSignRequest) (*BatchSignResponse, error) {
	if len(req.Documents) == 0 {
		return nil, fmt.Errorf("no documents to sign")
	}

	if len(req.Documents) > 100 {
		return nil, fmt.Errorf("batch size exceeds maximum of 100 documents")
	}

	// Validate all hashes
	for i, doc := range req.Documents {
		if _, err := hex.DecodeString(doc.DocumentHash); err != nil {
			return nil, fmt.Errorf("invalid hash for document %d (%s): %w", i, doc.ID, err)
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp BatchSignResponse
	if err := c.doRequest(ctx, "POST", "/batch-sign", body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetTimestamp retrieves a qualified timestamp for a hash
func (c *Client) GetTimestamp(ctx context.Context, req *TimestampRequest) (*TimestampResponse, error) {
	if req.HashAlgorithm == "" {
		req.HashAlgorithm = HashAlgoSHA256
	}

	// Validate hash format
	if _, err := hex.DecodeString(req.Hash); err != nil {
		return nil, fmt.Errorf("invalid hash: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp TimestampResponse
	if err := c.doRequest(ctx, "POST", "/timestamp", body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetCertificateInfo retrieves information about a certificate
func (c *Client) GetCertificateInfo(ctx context.Context, certID string) (*CertificateInfo, error) {
	var resp CertificateInfo
	if err := c.doRequest(ctx, "GET", "/certificates/"+certID, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// HealthCheck performs a health check on the A-Trust API
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.doRequest(ctx, "GET", "/health", nil, nil)
}

// doRequest performs an HTTP request with retry logic
func (c *Client) doRequest(ctx context.Context, method, path string, body []byte, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.retryMax; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := c.doSingleRequest(ctx, method, path, body, result)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if atrustErr, ok := err.(*ATrustError); ok {
			if !atrustErr.IsRetryable() {
				return err
			}
		}
	}

	return fmt.Errorf("request failed after %d attempts: %w", c.retryMax+1, lastErr)
}

func (c *Client) doSingleRequest(ctx context.Context, method, path string, body []byte, result interface{}) error {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.sandbox {
		req.Header.Set("X-Sandbox-Mode", "true")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return &ATrustError{
				StatusCode: resp.StatusCode,
				Code:       "UNKNOWN",
				Message:    string(respBody),
			}
		}
		return &ATrustError{
			StatusCode: resp.StatusCode,
			Code:       errResp.Code,
			Message:    errResp.Message,
			Details:    errResp.Details,
		}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// ATrustError represents an error from the A-Trust API
type ATrustError struct {
	StatusCode int
	Code       string
	Message    string
	Details    string
}

func (e *ATrustError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("A-Trust error %s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("A-Trust error %s: %s", e.Code, e.Message)
}

// IsRetryable returns true if the error is retryable
func (e *ATrustError) IsRetryable() bool {
	switch e.Code {
	case ErrCodeServiceUnavailable, ErrCodeRateLimited:
		return true
	default:
		return e.StatusCode >= 500
	}
}

// IsCertificateError returns true if the error is related to the certificate
func (e *ATrustError) IsCertificateError() bool {
	switch e.Code {
	case ErrCodeInvalidCertificate, ErrCodeCertificateExpired, ErrCodeCertificateRevoked:
		return true
	default:
		return false
	}
}

// HashDocument calculates the SHA-256 hash of a document
func HashDocument(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// HashDocumentReader calculates the SHA-256 hash of a document from a reader
func HashDocumentReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
