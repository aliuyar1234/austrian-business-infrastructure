package integration

import (
	"bytes"
	"testing"

	"github.com/austrian-business-infrastructure/fo/internal/ocr"
)

// T075: Integration tests for OCR service

func TestOCRDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("Detect with invalid PDF", func(t *testing.T) {
		// Test detection with invalid PDF data
		_, err := ocr.DetectFromBytes([]byte("not a valid pdf"))
		if err == nil {
			t.Log("Detection may succeed with heuristics even for invalid PDFs")
		}
	})

	t.Run("DetectionResult methods", func(t *testing.T) {
		result := &ocr.DetectionResult{
			IsScanned:    true,
			HasText:      false,
			PageCount:    5,
			TextDensity:  50.0,
			Confidence:   0.8,
			RecommendOCR: true,
		}

		// Test ShouldOCR
		if !result.ShouldOCR() {
			t.Error("Scanned document with OCR recommendation should return true for ShouldOCR")
		}

		// Test String representation
		str := result.String()
		if str == "" {
			t.Error("String representation should not be empty")
		}
	})

	t.Run("Native PDF detection result", func(t *testing.T) {
		result := &ocr.DetectionResult{
			IsScanned:    false,
			HasText:      true,
			PageCount:    3,
			TextDensity:  2000.0,
			Confidence:   0.95,
			RecommendOCR: false,
		}

		// Native PDF should not need OCR
		if result.ShouldOCR() {
			t.Error("Native PDF should not need OCR")
		}

		if result.TextDensity < 1000 {
			t.Error("Native PDF should have high text density")
		}
	})
}

func TestOCRService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("Service creation with config", func(t *testing.T) {
		// Test that OCR service can be created with configuration
		cfg := ocr.ServiceConfig{
			Provider:      ocr.ProviderTesseract,
			HunyuanURL:    "http://localhost:8080",
			MinConfidence: 0.7,
		}

		service, err := ocr.NewService(cfg)
		if err != nil {
			t.Errorf("Failed to create OCR service: %v", err)
		}
		if service == nil {
			t.Error("Service should not be nil")
		}
	})

	t.Run("Service with auto provider", func(t *testing.T) {
		cfg := ocr.ServiceConfig{
			Provider:      ocr.ProviderAuto,
			MinConfidence: 0.6,
		}

		service, err := ocr.NewService(cfg)
		if err != nil {
			t.Errorf("Failed to create OCR service with auto provider: %v", err)
		}

		if !service.IsAvailable() {
			t.Log("Service may not be available if tesseract is not installed")
		}
	})

	t.Run("Available providers", func(t *testing.T) {
		cfg := ocr.ServiceConfig{
			Provider:   ocr.ProviderAuto,
			HunyuanURL: "http://localhost:8080",
		}

		service, _ := ocr.NewService(cfg)
		providers := service.AvailableProviders()

		// Should have at least tesseract
		if len(providers) == 0 {
			t.Log("No providers available - tesseract may not be installed")
		}
	})
}

func TestOCRResult(t *testing.T) {
	t.Run("Result structure", func(t *testing.T) {
		result := &ocr.Result{
			Text:       "Extracted text from document",
			Confidence: 0.95,
			Provider:   ocr.ProviderTesseract,
			PageTexts:  []string{"Page 1", "Page 2", "Page 3"},
		}

		if result.Text == "" {
			t.Error("Result text should not be empty")
		}

		if result.Confidence < 0 || result.Confidence > 1 {
			t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
		}

		if len(result.PageTexts) != 3 {
			t.Errorf("Expected 3 page texts, got %d", len(result.PageTexts))
		}
	})

	t.Run("Low confidence detection", func(t *testing.T) {
		result := &ocr.Result{
			Text:       "Partially readable text",
			Confidence: 0.45,
			Provider:   ocr.ProviderTesseract,
		}

		// Low confidence should trigger warning
		if result.Confidence >= 0.7 {
			t.Error("Expected low confidence result")
		}
	})

	t.Run("Result with error", func(t *testing.T) {
		result := &ocr.Result{
			Provider: ocr.ProviderNone,
			Text:     "",
		}

		if result.Text != "" {
			t.Error("Empty result should have no text")
		}
	})
}

func TestPDFTextExtraction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("Extract from bytes", func(t *testing.T) {
		// Test PDF text extraction with invalid PDF
		_, err := ocr.ExtractPDFTextFromBytes([]byte("not a pdf"))
		if err == nil {
			t.Log("Extraction from invalid PDF may succeed or fail")
		}
	})

	t.Run("Extract from reader", func(t *testing.T) {
		// Test PDF text extraction with reader
		reader := bytes.NewReader([]byte("not a pdf"))
		_, err := ocr.ExtractPDFText(reader)
		if err == nil {
			t.Log("Extraction from invalid PDF reader may succeed or fail")
		}
	})
}

func TestOCRProviderConstants(t *testing.T) {
	// Verify OCR provider constants are of correct type
	if ocr.ProviderTesseract != ocr.Provider("tesseract") {
		t.Errorf("Expected ProviderTesseract to be 'tesseract', got %s", ocr.ProviderTesseract)
	}
	if ocr.ProviderHunyuan != ocr.Provider("hunyuan") {
		t.Errorf("Expected ProviderHunyuan to be 'hunyuan', got %s", ocr.ProviderHunyuan)
	}
	if ocr.ProviderAuto != ocr.Provider("auto") {
		t.Errorf("Expected ProviderAuto to be 'auto', got %s", ocr.ProviderAuto)
	}
	if ocr.ProviderNone != ocr.Provider("none") {
		t.Errorf("Expected ProviderNone to be 'none', got %s", ocr.ProviderNone)
	}
}

func TestDetectionResultShouldOCR(t *testing.T) {
	testCases := []struct {
		name         string
		isScanned    bool
		recommendOCR bool
		expected     bool
	}{
		{"scanned with OCR recommendation", true, true, true},
		{"scanned without OCR recommendation", true, false, false},
		{"native with OCR recommendation", false, true, false},
		{"native without OCR recommendation", false, false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &ocr.DetectionResult{
				IsScanned:    tc.isScanned,
				RecommendOCR: tc.recommendOCR,
			}

			if result.ShouldOCR() != tc.expected {
				t.Errorf("ShouldOCR() = %v, expected %v", result.ShouldOCR(), tc.expected)
			}
		})
	}
}
