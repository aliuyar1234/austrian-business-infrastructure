package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HunyuanClient is an HTTP client for the HunyuanOCR bridge service
type HunyuanClient struct {
	baseURL    string
	httpClient *http.Client
}

// HunyuanResult contains the OCR result from Hunyuan
type HunyuanResult struct {
	Text       string   `json:"text"`
	Pages      []string `json:"pages"`
	Confidence float64  `json:"confidence"`
}

// HunyuanRequest is the request format for the bridge service
type hunyuanRequest struct {
	PDFBase64 string `json:"pdf_base64"`
	Language  string `json:"language,omitempty"`
}

// hunyuanResponse is the response format from the bridge service
type hunyuanResponse struct {
	Success    bool     `json:"success"`
	Text       string   `json:"text"`
	Pages      []string `json:"pages,omitempty"`
	Confidence float64  `json:"confidence"`
	Error      string   `json:"error,omitempty"`
}

// NewHunyuanClient creates a new HunyuanOCR client
func NewHunyuanClient(baseURL string) *HunyuanClient {
	return &HunyuanClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // OCR can take a while
		},
	}
}

// ProcessPDF sends a PDF to the HunyuanOCR bridge service
func (c *HunyuanClient) ProcessPDF(ctx context.Context, pdfData []byte) (*HunyuanResult, error) {
	// Encode PDF as base64
	encoded := base64.StdEncoding.EncodeToString(pdfData)

	req := hunyuanRequest{
		PDFBase64: encoded,
		Language:  "deu", // German
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/ocr", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("hunyuan OCR error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result hunyuanResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("hunyuan OCR failed: %s", result.Error)
	}

	return &HunyuanResult{
		Text:       result.Text,
		Pages:      result.Pages,
		Confidence: result.Confidence,
	}, nil
}

// IsAvailable checks if the HunyuanOCR service is available
func (c *HunyuanClient) IsAvailable(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
