package fonws

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// FinanzOnline WebService base URL
	BaseURL = "https://finanzonline.bmf.gv.at/fonws/ws"

	// Service endpoints
	SessionServiceURL = BaseURL + "/sessionService"
	DataboxServiceURL = BaseURL + "/databoxService"

	// XML namespaces
	SOAPEnvNS    = "http://schemas.xmlsoap.org/soap/envelope/"
	SessionNS    = "https://finanzonline.bmf.gv.at/fonws/ws/sessionService"
	DataboxNS    = "https://finanzonline.bmf.gv.at/fonws/ws/databoxService"

	// HTTP settings
	DefaultTimeout = 30 * time.Second

	// Retry settings
	DefaultMaxRetries   = 3
	DefaultRetryBackoff = 1 * time.Second
)

// SOAPEnvelope represents a SOAP envelope for requests
type SOAPEnvelope struct {
	XMLName xml.Name    `xml:"soap:Envelope"`
	SoapNS  string      `xml:"xmlns:soap,attr"`
	Body    SOAPBody    `xml:"soap:Body"`
}

// SOAPBody represents the SOAP body
type SOAPBody struct {
	Content interface{} `xml:",any"`
}

// SOAPResponseEnvelope represents a SOAP envelope for responses
type SOAPResponseEnvelope struct {
	XMLName xml.Name         `xml:"Envelope"`
	Body    SOAPResponseBody `xml:"Body"`
}

// SOAPResponseBody represents the response SOAP body
type SOAPResponseBody struct {
	Content []byte `xml:",innerxml"`
}

// Client is the SOAP HTTP client for FinanzOnline WebService
type Client struct {
	httpClient   *http.Client
	verbose      bool
	maxRetries   int
	retryBackoff time.Duration
}

// NewClient creates a new SOAP client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		maxRetries:   DefaultMaxRetries,
		retryBackoff: DefaultRetryBackoff,
	}
}

// SetRetry configures retry behavior for network operations
func (c *Client) SetRetry(maxRetries int, backoff time.Duration) {
	c.maxRetries = maxRetries
	c.retryBackoff = backoff
}

// SetVerbose enables verbose logging
func (c *Client) SetVerbose(v bool) {
	c.verbose = v
}

// BuildEnvelope creates a SOAP envelope containing the given request body
func BuildEnvelope(body interface{}) ([]byte, error) {
	envelope := SOAPEnvelope{
		SoapNS: SOAPEnvNS,
		Body: SOAPBody{
			Content: body,
		},
	}

	data, err := xml.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SOAP envelope: %w", err)
	}

	// Add XML declaration
	result := append([]byte(xml.Header), data...)
	return result, nil
}

// Post sends a SOAP request and returns the raw response body content
func (c *Client) Post(url string, requestBody interface{}) ([]byte, error) {
	// Build SOAP envelope
	envelope, err := BuildEnvelope(requestBody)
	if err != nil {
		return nil, err
	}

	return c.postWithRetry(url, envelope)
}

// postWithRetry executes an HTTP POST with exponential backoff retry
func (c *Client) postWithRetry(url string, envelope []byte) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, ...
			backoff := c.retryBackoff * time.Duration(1<<(attempt-1))
			time.Sleep(backoff)
		}

		body, err := c.doPost(url, envelope)
		if err == nil {
			return body, nil
		}

		lastErr = err

		// Only retry on transient network errors, not on HTTP 4xx errors
		if !isRetryableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
}

// doPost performs a single HTTP POST request
func (c *Client) doPost(url string, envelope []byte) ([]byte, error) {
	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewReader(envelope))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("Accept", "text/xml")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	return body, nil
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP error %d: %s", e.StatusCode, e.Body)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	// Retry on network errors (connection refused, timeout, etc.)
	if _, ok := err.(*HTTPError); ok {
		httpErr := err.(*HTTPError)
		// Retry on 5xx server errors and 429 (rate limit)
		return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429
	}
	// Retry on other errors (likely network issues)
	return true
}

// ParseResponse extracts the inner content from a SOAP response and unmarshals it
func ParseResponse(responseBody []byte, result interface{}) error {
	// Parse the SOAP envelope
	var envelope SOAPResponseEnvelope
	if err := xml.Unmarshal(responseBody, &envelope); err != nil {
		return fmt.Errorf("failed to parse SOAP envelope: %w", err)
	}

	// Unmarshal the body content into the result
	if err := xml.Unmarshal(envelope.Body.Content, result); err != nil {
		return fmt.Errorf("failed to parse response body: %w", err)
	}

	return nil
}

// Call makes a SOAP call and parses the response into the result
func (c *Client) Call(url string, request interface{}, response interface{}) error {
	body, err := c.Post(url, request)
	if err != nil {
		return err
	}

	return ParseResponse(body, response)
}
