package platform

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestClient wraps HTTP test requests
type TestClient struct {
	t       *testing.T
	handler http.Handler
	token   string
}

// NewTestClient creates a new test client
func NewTestClient(t *testing.T, handler http.Handler) *TestClient {
	return &TestClient{
		t:       t,
		handler: handler,
	}
}

// SetToken sets the authentication token for subsequent requests
func (c *TestClient) SetToken(token string) {
	c.token = token
}

// Request performs an HTTP request and returns the response
func (c *TestClient) Request(method, path string, body interface{}) *httptest.ResponseRecorder {
	c.t.Helper()

	var bodyReader *bytes.Buffer
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewBuffer(data)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)

	return rec
}

// Get performs a GET request
func (c *TestClient) Get(path string) *httptest.ResponseRecorder {
	return c.Request(http.MethodGet, path, nil)
}

// Post performs a POST request
func (c *TestClient) Post(path string, body interface{}) *httptest.ResponseRecorder {
	return c.Request(http.MethodPost, path, body)
}

// Put performs a PUT request
func (c *TestClient) Put(path string, body interface{}) *httptest.ResponseRecorder {
	return c.Request(http.MethodPut, path, body)
}

// Patch performs a PATCH request
func (c *TestClient) Patch(path string, body interface{}) *httptest.ResponseRecorder {
	return c.Request(http.MethodPatch, path, body)
}

// Delete performs a DELETE request
func (c *TestClient) Delete(path string) *httptest.ResponseRecorder {
	return c.Request(http.MethodDelete, path, nil)
}

// ParseResponse parses a JSON response into the provided struct
func ParseResponse(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("Failed to parse response: %v\nBody: %s", err, rec.Body.String())
	}
}

// AssertStatus checks the response status code
func AssertStatus(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()

	if rec.Code != expected {
		t.Errorf("Expected status %d, got %d\nBody: %s", expected, rec.Code, rec.Body.String())
	}
}

// AssertJSON checks that response is valid JSON with specific field
func AssertJSON(t *testing.T, rec *httptest.ResponseRecorder, field string) {
	t.Helper()

	var data map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil {
		t.Fatalf("Response is not valid JSON: %v\nBody: %s", err, rec.Body.String())
	}

	if _, ok := data[field]; !ok {
		t.Errorf("Expected field '%s' in response, got: %v", field, data)
	}
}

// getEnv is a helper that uses os.Getenv
func getEnvFromOS(key string) string {
	return os.Getenv(key)
}
