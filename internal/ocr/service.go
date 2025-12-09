package ocr

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

// Provider represents an OCR provider
type Provider string

const (
	ProviderAuto      Provider = "auto"
	ProviderHunyuan   Provider = "hunyuan"
	ProviderTesseract Provider = "tesseract"
	ProviderNone      Provider = "none"
)

// Result contains the OCR processing result
type Result struct {
	Text       string   // Extracted text
	Provider   Provider // Provider that was used
	Confidence float64  // Overall confidence score
	PageTexts  []string // Text per page
	Error      error    // Any error during processing
}

// Service orchestrates OCR processing with fallback
type Service struct {
	hunyuan      *HunyuanClient
	tesseract    *TesseractClient
	provider     Provider
	minConfidence float64
}

// ServiceConfig holds OCR service configuration
type ServiceConfig struct {
	Provider         Provider
	HunyuanURL       string
	TesseractPath    string
	MinConfidence    float64
}

// NewService creates a new OCR service
func NewService(cfg ServiceConfig) (*Service, error) {
	s := &Service{
		provider:      cfg.Provider,
		minConfidence: cfg.MinConfidence,
	}

	if s.minConfidence == 0 {
		s.minConfidence = 0.7
	}

	// Initialize Hunyuan client if URL provided
	if cfg.HunyuanURL != "" {
		s.hunyuan = NewHunyuanClient(cfg.HunyuanURL)
	}

	// Initialize Tesseract client
	tesseractPath := cfg.TesseractPath
	if tesseractPath == "" {
		tesseractPath = "tesseract"
	}
	s.tesseract = NewTesseractClient(tesseractPath)

	return s, nil
}

// Process performs OCR on a PDF document
func (s *Service) Process(ctx context.Context, reader io.ReadSeeker) (*Result, error) {
	// First, detect if OCR is needed
	detection, err := Detect(reader)
	if err != nil {
		return nil, fmt.Errorf("detect scan: %w", err)
	}

	// Reset reader position
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek: %w", err)
	}

	// If not scanned, extract text directly
	if !detection.ShouldOCR() {
		return s.extractNativeText(ctx, reader)
	}

	// Perform OCR based on configured provider
	switch s.provider {
	case ProviderHunyuan:
		return s.processWithHunyuan(ctx, reader)
	case ProviderTesseract:
		return s.processWithTesseract(ctx, reader)
	case ProviderAuto:
		return s.processWithFallback(ctx, reader)
	default:
		// Return empty result if OCR disabled
		return &Result{
			Provider: ProviderNone,
			Text:     "",
		}, nil
	}
}

// extractNativeText extracts text from native PDFs
func (s *Service) extractNativeText(ctx context.Context, reader io.ReadSeeker) (*Result, error) {
	text, err := ExtractPDFText(reader)
	if err != nil {
		return nil, fmt.Errorf("extract PDF text: %w", err)
	}

	return &Result{
		Text:       text,
		Provider:   ProviderNone,
		Confidence: 1.0, // Native text is 100% confidence
	}, nil
}

// processWithHunyuan uses HunyuanOCR for text extraction
func (s *Service) processWithHunyuan(ctx context.Context, reader io.ReadSeeker) (*Result, error) {
	if s.hunyuan == nil {
		return nil, fmt.Errorf("hunyuan client not configured")
	}

	// Read PDF data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read PDF: %w", err)
	}

	result, err := s.hunyuan.ProcessPDF(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("hunyuan OCR: %w", err)
	}

	return &Result{
		Text:       result.Text,
		Provider:   ProviderHunyuan,
		Confidence: result.Confidence,
		PageTexts:  result.Pages,
	}, nil
}

// processWithTesseract uses Tesseract for text extraction
func (s *Service) processWithTesseract(ctx context.Context, reader io.ReadSeeker) (*Result, error) {
	// Read PDF data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read PDF: %w", err)
	}

	result, err := s.tesseract.ProcessPDF(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("tesseract OCR: %w", err)
	}

	return &Result{
		Text:       result.Text,
		Provider:   ProviderTesseract,
		Confidence: result.Confidence,
		PageTexts:  result.Pages,
	}, nil
}

// processWithFallback tries HunyuanOCR first, falls back to Tesseract
func (s *Service) processWithFallback(ctx context.Context, reader io.ReadSeeker) (*Result, error) {
	// Read PDF data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read PDF: %w", err)
	}

	// Try HunyuanOCR first if available
	if s.hunyuan != nil {
		result, err := s.hunyuan.ProcessPDF(ctx, data)
		if err == nil && result.Confidence >= s.minConfidence {
			return &Result{
				Text:       result.Text,
				Provider:   ProviderHunyuan,
				Confidence: result.Confidence,
				PageTexts:  result.Pages,
			}, nil
		}
		// Log error but continue to fallback
		if err != nil {
			fmt.Printf("hunyuan OCR failed, falling back: %v\n", err)
		}
	}

	// Fall back to Tesseract
	result, err := s.tesseract.ProcessPDF(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("tesseract OCR (fallback): %w", err)
	}

	return &Result{
		Text:       result.Text,
		Provider:   ProviderTesseract,
		Confidence: result.Confidence,
		PageTexts:  result.Pages,
	}, nil
}

// ProcessBytes is a convenience wrapper that takes bytes directly
func (s *Service) ProcessBytes(ctx context.Context, data []byte) (*Result, error) {
	return s.Process(ctx, bytes.NewReader(data))
}

// IsAvailable checks if the service is properly configured
func (s *Service) IsAvailable() bool {
	// At minimum, tesseract should be available
	return s.tesseract != nil
}

// AvailableProviders returns a list of configured providers
func (s *Service) AvailableProviders() []Provider {
	providers := []Provider{}
	if s.hunyuan != nil {
		providers = append(providers, ProviderHunyuan)
	}
	if s.tesseract != nil {
		providers = append(providers, ProviderTesseract)
	}
	return providers
}
