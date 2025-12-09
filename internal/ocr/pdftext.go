package ocr

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ExtractPDFText extracts text content from a native PDF
func ExtractPDFText(reader io.ReadSeeker) (string, error) {
	// Read all data
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read PDF: %w", err)
	}

	// Create temp file for pdfcpu
	tmpFile, err := os.CreateTemp("", "pdf-extract-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	// Create output directory for extraction
	outDir, err := os.MkdirTemp("", "pdf-content-*")
	if err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}
	defer os.RemoveAll(outDir)

	// Extract content using pdfcpu
	conf := model.NewDefaultConfiguration()
	if err := api.ExtractContentFile(tmpFile.Name(), outDir, nil, conf); err != nil {
		// If extraction fails, return empty - PDF might not have extractable text
		return "", nil
	}

	// Read extracted content files
	var text strings.Builder
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return "", nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			content, err := os.ReadFile(outDir + "/" + entry.Name())
			if err == nil {
				text.Write(content)
				text.WriteString("\n")
			}
		}
	}

	// Clean up extracted text
	return cleanText(text.String()), nil
}

// ExtractPDFTextFromBytes extracts text from PDF bytes
func ExtractPDFTextFromBytes(data []byte) (string, error) {
	return ExtractPDFText(bytes.NewReader(data))
}

// ExtractPDFPages extracts text page by page
func ExtractPDFPages(reader io.ReadSeeker) ([]string, error) {
	// For now, extract all text as single page
	// Page-by-page extraction requires more complex handling
	text, err := ExtractPDFText(reader)
	if err != nil {
		return nil, err
	}
	return []string{text}, nil
}

// cleanText cleans up extracted PDF text
func cleanText(text string) string {
	// Remove excessive whitespace
	lines := strings.Split(text, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	text = strings.Join(cleanLines, "\n")

	// Remove multiple consecutive newlines
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	// Remove control characters
	text = removeControlChars(text)

	return strings.TrimSpace(text)
}

// removeControlChars removes non-printable control characters
func removeControlChars(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r >= 32 || r == '\n' || r == '\t' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// TruncateText truncates text to a maximum length while preserving words
func TruncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	// Find the last space before maxLen
	truncated := text[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLen/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// CountWords returns the word count of text
func CountWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

// EstimateTokens estimates the number of tokens (rough approximation)
// Claude typically uses ~1 token per 4 characters for English
// German text may use slightly more tokens
func EstimateTokens(text string) int {
	// Use ~3.5 characters per token for German text
	return len(text) / 3
}
