// Package ocr provides OCR capabilities for scanned documents
package ocr

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// DetectionResult contains the result of scan detection
type DetectionResult struct {
	IsScanned      bool    // True if the PDF appears to be scanned
	HasText        bool    // True if the PDF has embedded text
	PageCount      int     // Number of pages
	TextDensity    float64 // Characters per page (0 if no text)
	ImageCount     int     // Number of images found
	Confidence     float64 // Confidence in the detection (0-1)
	RecommendOCR   bool    // True if OCR is recommended
}

// Detect analyzes a PDF to determine if it's scanned or native
func Detect(reader io.ReadSeeker) (*DetectionResult, error) {
	result := &DetectionResult{}

	// Read PDF into buffer
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, reader); err != nil {
		return nil, fmt.Errorf("read PDF: %w", err)
	}

	// Reset reader position
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek: %w", err)
	}

	// Get PDF info using pdfcpu
	ctx, err := api.ReadContext(bytes.NewReader(buf.Bytes()), model.NewDefaultConfiguration())
	if err != nil {
		return nil, fmt.Errorf("read PDF context: %w", err)
	}

	result.PageCount = ctx.PageCount

	// Extract text to check if PDF has embedded text
	extractedText, err := ExtractPDFTextFromBytes(buf.Bytes())
	if err != nil {
		// Some PDFs may fail text extraction, treat as potentially scanned
		result.HasText = false
	} else {
		result.HasText = len(extractedText) > 100 // Minimal text threshold

		// Calculate text density
		if result.PageCount > 0 {
			result.TextDensity = float64(len(extractedText)) / float64(result.PageCount)
		}
	}

	// Analyze images
	// Note: Full image analysis would require deeper PDF parsing
	// For now, use heuristics based on text density

	// Heuristics for determining if scanned:
	// - Very low text density suggests scanned
	// - High page count with low text suggests scanned images

	if result.TextDensity < 100 && result.PageCount > 0 {
		// Very low text per page - likely scanned
		result.IsScanned = true
		result.Confidence = 0.8
		result.RecommendOCR = true
	} else if result.TextDensity < 500 {
		// Low text density - possibly scanned with some OCR
		result.IsScanned = true
		result.Confidence = 0.6
		result.RecommendOCR = true
	} else {
		// Good text density - native PDF
		result.IsScanned = false
		result.Confidence = 0.9
		result.RecommendOCR = false
	}

	// Override if we have good embedded text
	if result.HasText && result.TextDensity > 1000 {
		result.IsScanned = false
		result.Confidence = 0.95
		result.RecommendOCR = false
	}

	return result, nil
}

// DetectFromBytes is a convenience wrapper that takes bytes directly
func DetectFromBytes(data []byte) (*DetectionResult, error) {
	return Detect(bytes.NewReader(data))
}

// ShouldOCR returns true if OCR processing is recommended for this document
func (r *DetectionResult) ShouldOCR() bool {
	return r.RecommendOCR && r.IsScanned
}

// String returns a human-readable description
func (r *DetectionResult) String() string {
	scanType := "native"
	if r.IsScanned {
		scanType = "scanned"
	}
	return fmt.Sprintf("PDF: %s, %d pages, text density: %.0f chars/page, confidence: %.0f%%",
		scanType, r.PageCount, r.TextDensity, r.Confidence*100)
}
