package fb

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// FBNS is the namespace for Firmenbuch services
	FBNS = "https://www.justiz.gv.at/firmenbuch"

	// FBEndpoint is the production Firmenbuch service endpoint
	FBEndpoint = "https://www.justiz.gv.at/firmenbuch/ws/abfrage"

	// FBTestEndpoint is the test Firmenbuch service endpoint
	FBTestEndpoint = "https://test.justiz.gv.at/firmenbuch/ws/abfrage"
)

// Client handles Firmenbuch API communication
type Client struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
	timeout    time.Duration
}

// NewClient creates a new Firmenbuch client
func NewClient(apiKey string, testMode bool) *Client {
	endpoint := FBEndpoint
	if testMode {
		endpoint = FBTestEndpoint
	}

	return &Client{
		endpoint: endpoint,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}
}

// soapEnvelope wraps a request in a SOAP envelope with API key authentication
func (c *Client) soapEnvelope(body []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString(`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">`)
	buf.WriteString(`<soap:Header>`)
	buf.WriteString(`<auth:Authentication xmlns:auth="https://www.justiz.gv.at/auth">`)
	buf.WriteString(fmt.Sprintf(`<auth:APIKey>%s</auth:APIKey>`, c.apiKey))
	buf.WriteString(`</auth:Authentication>`)
	buf.WriteString(`</soap:Header>`)
	buf.WriteString(`<soap:Body>`)
	buf.Write(body)
	buf.WriteString(`</soap:Body>`)
	buf.WriteString(`</soap:Envelope>`)
	return buf.Bytes()
}

// call makes a SOAP call to Firmenbuch
func (c *Client) call(action string, request interface{}, response interface{}) error {
	// Marshal request body
	body, err := xml.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Wrap in SOAP envelope
	soapBody := c.soapEnvelope(body)

	// Create HTTP request
	req, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(soapBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", action)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("FB connection failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FB request failed: HTTP %d", resp.StatusCode)
	}

	// Parse response (extract from SOAP envelope)
	return parseSOAPResponse(respBody, response)
}

// parseSOAPResponse extracts the body from a SOAP response
func parseSOAPResponse(data []byte, v interface{}) error {
	type soapBody struct {
		Content []byte `xml:",innerxml"`
	}
	type soapEnvelope struct {
		Body soapBody `xml:"Body"`
	}

	var env soapEnvelope
	if err := xml.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("failed to parse SOAP envelope: %w", err)
	}

	if err := xml.Unmarshal(env.Body.Content, v); err != nil {
		return fmt.Errorf("failed to parse response body: %w", err)
	}

	return nil
}

// Search searches for companies by name or other criteria
func (c *Client) Search(req *FBSearchRequest) (*FBSearchResponse, error) {
	var resp FBSearchResponse
	err := c.call("Search", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Extract retrieves the full company extract for a given FN
func (c *Client) Extract(fn string) (*FBExtract, error) {
	// Validate FN format
	if err := ValidateFN(fn); err != nil {
		return nil, fmt.Errorf("invalid FN: %w", err)
	}

	req := &FBExtractRequest{
		FN: fn,
	}

	var extract FBExtract
	err := c.call("Extract", req, &extract)
	if err != nil {
		return nil, err
	}

	return &extract, nil
}

// GenerateSearchXML generates XML for a search request (for testing/debugging)
func GenerateSearchXML(req *FBSearchRequest) ([]byte, error) {
	output, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}

	result := []byte(xml.Header)
	result = append(result, output...)
	return result, nil
}
