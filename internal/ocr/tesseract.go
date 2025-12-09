package ocr

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gen2brain/go-fitz"
)

// TesseractClient wraps Tesseract OCR
type TesseractClient struct {
	tesseractPath string
	language      string
}

// TesseractResult contains the OCR result from Tesseract
type TesseractResult struct {
	Text       string
	Pages      []string
	Confidence float64
}

// NewTesseractClient creates a new Tesseract client
func NewTesseractClient(tesseractPath string) *TesseractClient {
	if tesseractPath == "" {
		tesseractPath = "tesseract"
	}
	return &TesseractClient{
		tesseractPath: tesseractPath,
		language:      "deu", // German
	}
}

// ProcessPDF converts PDF to images and runs OCR on each page
func (c *TesseractClient) ProcessPDF(ctx context.Context, pdfData []byte) (*TesseractResult, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "ocr-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write PDF to temp file
	pdfPath := filepath.Join(tempDir, "input.pdf")
	if err := os.WriteFile(pdfPath, pdfData, 0644); err != nil {
		return nil, fmt.Errorf("write PDF: %w", err)
	}

	// Convert PDF pages to images using go-fitz
	images, err := c.pdfToImages(pdfPath, tempDir)
	if err != nil {
		return nil, fmt.Errorf("convert PDF to images: %w", err)
	}

	// OCR each image
	var pages []string
	var allText strings.Builder
	totalConfidence := 0.0

	for i, imgPath := range images {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		text, conf, err := c.ocrImage(ctx, imgPath)
		if err != nil {
			return nil, fmt.Errorf("OCR page %d: %w", i+1, err)
		}

		pages = append(pages, text)
		allText.WriteString(text)
		allText.WriteString("\n\n")
		totalConfidence += conf
	}

	avgConfidence := 0.0
	if len(pages) > 0 {
		avgConfidence = totalConfidence / float64(len(pages))
	}

	return &TesseractResult{
		Text:       strings.TrimSpace(allText.String()),
		Pages:      pages,
		Confidence: avgConfidence / 100.0, // Convert to 0-1 range
	}, nil
}

// pdfToImages converts PDF pages to PNG images
func (c *TesseractClient) pdfToImages(pdfPath, outputDir string) ([]string, error) {
	doc, err := fitz.New(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("open PDF: %w", err)
	}
	defer doc.Close()

	var images []string

	for i := 0; i < doc.NumPage(); i++ {
		// Render page to image (300 DPI for good OCR quality)
		img, err := doc.Image(i)
		if err != nil {
			return nil, fmt.Errorf("render page %d: %w", i+1, err)
		}

		// Save as PNG
		imgPath := filepath.Join(outputDir, fmt.Sprintf("page_%04d.png", i+1))
		if err := saveImage(img, imgPath); err != nil {
			return nil, fmt.Errorf("save page %d: %w", i+1, err)
		}

		images = append(images, imgPath)
	}

	return images, nil
}

// saveImage saves an image to a PNG file
func saveImage(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// ocrImage runs Tesseract on a single image
func (c *TesseractClient) ocrImage(ctx context.Context, imagePath string) (string, float64, error) {
	// Create output base path (tesseract adds .txt)
	outputBase := strings.TrimSuffix(imagePath, filepath.Ext(imagePath))

	// Run tesseract with confidence output
	args := []string{
		imagePath,
		outputBase,
		"-l", c.language,
		"--psm", "1", // Automatic page segmentation with OSD
		"--oem", "1", // LSTM only
	}

	cmd := exec.CommandContext(ctx, c.tesseractPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", 0, fmt.Errorf("tesseract failed: %w, stderr: %s", err, stderr.String())
	}

	// Read output text
	outputPath := outputBase + ".txt"
	text, err := os.ReadFile(outputPath)
	if err != nil {
		return "", 0, fmt.Errorf("read output: %w", err)
	}

	// Estimate confidence based on text quality
	// (Full confidence extraction would require parsing hOCR output)
	confidence := estimateConfidence(string(text))

	return strings.TrimSpace(string(text)), confidence, nil
}

// estimateConfidence provides a rough confidence estimate based on text quality
func estimateConfidence(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	// Count "garbage" characters (non-printable, weird symbols)
	garbageCount := 0
	totalCount := 0

	for _, r := range text {
		totalCount++
		// Count characters that are likely OCR errors
		if r > 127 || (r < 32 && r != '\n' && r != '\t' && r != ' ') {
			garbageCount++
		}
	}

	if totalCount == 0 {
		return 0
	}

	// Higher garbage ratio = lower confidence
	garbageRatio := float64(garbageCount) / float64(totalCount)
	confidence := 100 * (1 - garbageRatio*2) // Scale garbage impact

	if confidence < 0 {
		confidence = 0
	}
	if confidence > 100 {
		confidence = 100
	}

	return confidence
}

// IsAvailable checks if Tesseract is installed and available
func (c *TesseractClient) IsAvailable() bool {
	cmd := exec.Command(c.tesseractPath, "--version")
	return cmd.Run() == nil
}

// GetLanguages returns available Tesseract languages
func (c *TesseractClient) GetLanguages() ([]string, error) {
	cmd := exec.Command(c.tesseractPath, "--list-langs")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list languages: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var languages []string

	// Skip first line (header)
	for i := 1; i < len(lines); i++ {
		lang := strings.TrimSpace(lines[i])
		if lang != "" {
			languages = append(languages, lang)
		}
	}

	return languages, nil
}
