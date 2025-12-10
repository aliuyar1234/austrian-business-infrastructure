package sigfield

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Embedder handles embedding signatures into PDF documents
type Embedder struct {
	conf *model.Configuration
}

// NewEmbedder creates a new signature embedder
func NewEmbedder() *Embedder {
	return &Embedder{
		conf: model.NewDefaultConfiguration(),
	}
}

// GetDocumentInfo retrieves information about a PDF document
func (e *Embedder) GetDocumentInfo(content []byte) (*DocumentInfo, error) {
	reader := bytes.NewReader(content)

	ctx, err := api.ReadContext(reader, e.conf)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	info := &DocumentInfo{
		PageCount: ctx.PageCount,
		Pages:     make([]PageInfo, ctx.PageCount),
	}

	// Get page dimensions
	for i := 1; i <= ctx.PageCount; i++ {
		dim, err := ctx.PageDims()
		if err == nil && i <= len(dim) {
			info.Pages[i-1] = PageInfo{
				PageNumber: i,
				Width:      dim[i-1].Width,
				Height:     dim[i-1].Height,
			}
		}
	}

	// Check for existing signatures
	// This is a simplified check - actual implementation would parse signature dictionaries
	info.IsSigned = e.hasSignatures(ctx)

	// Check PDF/A compliance (simplified)
	info.IsPDFA = e.isPDFA(ctx)

	return info, nil
}

// hasSignatures checks if the PDF has existing signatures
func (e *Embedder) hasSignatures(ctx *model.Context) bool {
	// Look for signature fields in the AcroForm
	// This is a simplified check
	if ctx.RootDict == nil {
		return false
	}

	acroForm := ctx.RootDict.DictEntry("AcroForm")
	if acroForm == nil {
		return false
	}

	fields := acroForm.ArrayEntry("Fields")
	if fields == nil {
		return false
	}

	// Check if any field is a signature field
	// In a full implementation, we'd traverse the field tree
	return false
}

// isPDFA checks if the PDF is PDF/A compliant
func (e *Embedder) isPDFA(ctx *model.Context) bool {
	// Check for PDF/A identification in metadata
	// This is a simplified check
	return false
}

// EmbedVisualSignature embeds a visual signature representation into a PDF
// This creates the visual appearance of the signature without cryptographic signing
func (e *Embedder) EmbedVisualSignature(content []byte, opts *EmbedOptions) (*EmbedResult, error) {
	if opts.Appearance == nil {
		return nil, fmt.Errorf("appearance is required")
	}

	reader := bytes.NewReader(content)
	var output bytes.Buffer

	// Create watermark text for the signature appearance
	watermarkText := e.buildSignatureText(opts.Appearance)

	// Calculate position
	var x, y float64
	if opts.Field != nil {
		x = opts.Field.X
		y = opts.Field.Y
	} else {
		// Default position at bottom left
		x = 50
		y = 50
	}

	// Create watermark configuration
	wm, err := pdfcpu.ParseTextWatermarkDetails(watermarkText, fmt.Sprintf(
		"pos:bl, off:%f %f, sc:1 abs, rot:0, fillc:#ffffff, strokec:#000000, font:Helvetica, points:%d, rtl:off",
		x, y, int(opts.Appearance.FontSize),
	), true, types.POINTS)
	if err != nil {
		return nil, fmt.Errorf("failed to create watermark: %w", err)
	}

	// Determine which page to add the signature
	pageNr := 1
	if opts.Field != nil {
		pageNr = opts.Field.Page
	}

	// Add watermark to the PDF
	selectedPages := []string{fmt.Sprintf("%d", pageNr)}
	if err := api.AddWatermarks(reader, &output, selectedPages, wm, e.conf); err != nil {
		return nil, fmt.Errorf("failed to add signature: %w", err)
	}

	// Calculate hash of the signed document
	hash := sha256.Sum256(output.Bytes())

	return &EmbedResult{
		DocumentHash:   hex.EncodeToString(hash[:]),
		SignedDocument: output.Bytes(),
		SignatureID:    fmt.Sprintf("sig-%d", time.Now().UnixNano()),
	}, nil
}

// buildSignatureText builds the text representation of the signature
func (e *Embedder) buildSignatureText(app *SignatureAppearance) string {
	var parts []string

	// Add signer name
	if app.SignerName != "" {
		parts = append(parts, fmt.Sprintf("Signiert von: %s", app.SignerName))
	}

	// Add date
	dateStr := app.SignedAt.Format(app.DateFormat)
	if app.DateFormat == "" {
		dateStr = app.SignedAt.Format("02.01.2006 15:04")
	}
	parts = append(parts, fmt.Sprintf("Datum: %s", dateStr))

	// Add reason if present
	if app.Reason != "" {
		parts = append(parts, fmt.Sprintf("Grund: %s", app.Reason))
	}

	// Add location if present
	if app.Location != "" {
		parts = append(parts, fmt.Sprintf("Ort: %s", app.Location))
	}

	return strings.Join(parts, "\n")
}

// AddSignatureField adds a signature field placeholder to a PDF
func (e *Embedder) AddSignatureField(content []byte, field *SignatureField) ([]byte, error) {
	reader := bytes.NewReader(content)
	var output bytes.Buffer

	// Create a placeholder annotation for the signature field
	// This is a simplified implementation - full implementation would create
	// proper AcroForm signature fields
	placeholderText := fmt.Sprintf("[%s]", field.FieldName)

	wm, err := pdfcpu.ParseTextWatermarkDetails(placeholderText, fmt.Sprintf(
		"pos:bl, off:%f %f, sc:1 abs, rot:0, fillc:#f0f0f0, strokec:#cccccc, font:Helvetica, points:%d, rtl:off, border:1",
		field.X, field.Y, int(field.FontSize),
	), true, types.POINTS)
	if err != nil {
		return nil, fmt.Errorf("failed to create field placeholder: %w", err)
	}

	selectedPages := []string{fmt.Sprintf("%d", field.Page)}
	if err := api.AddWatermarks(reader, &output, selectedPages, wm, e.conf); err != nil {
		return nil, fmt.Errorf("failed to add field placeholder: %w", err)
	}

	return output.Bytes(), nil
}

// GeneratePreview generates a preview of the document with signature field placeholders
func (e *Embedder) GeneratePreview(content []byte, fields []*SignatureField) ([]byte, error) {
	result := content

	for _, field := range fields {
		var err error
		result, err = e.AddSignatureField(result, field)
		if err != nil {
			return nil, fmt.Errorf("failed to add field %s: %w", field.FieldName, err)
		}
	}

	return result, nil
}

// ValidateFieldPosition validates that a signature field position is within page bounds
func (e *Embedder) ValidateFieldPosition(content []byte, field *SignatureField) error {
	info, err := e.GetDocumentInfo(content)
	if err != nil {
		return fmt.Errorf("failed to get document info: %w", err)
	}

	if field.Page < 1 || field.Page > info.PageCount {
		return fmt.Errorf("page %d is out of range (document has %d pages)", field.Page, info.PageCount)
	}

	pageInfo := info.Pages[field.Page-1]

	if field.X < 0 || field.X+field.Width > pageInfo.Width {
		return fmt.Errorf("field x position is out of page bounds")
	}

	if field.Y < 0 || field.Y+field.Height > pageInfo.Height {
		return fmt.Errorf("field y position is out of page bounds")
	}

	return nil
}

// HashDocument calculates the SHA-256 hash of a PDF document
func (e *Embedder) HashDocument(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// HashDocumentReader calculates hash from a reader
func (e *Embedder) HashDocumentReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ConvertToPDFA converts a PDF to PDF/A format for long-term archiving
func (e *Embedder) ConvertToPDFA(content []byte) ([]byte, error) {
	// Note: pdfcpu doesn't directly support PDF/A conversion
	// In a production system, you might use a different library or service
	// For now, we'll return the original document with a note

	// TODO: Implement actual PDF/A conversion
	// Options:
	// 1. Use a dedicated PDF/A library
	// 2. Use Ghostscript via exec
	// 3. Use a cloud service

	return content, nil
}

// ExtractTextFromPage extracts text from a specific page (for positioning help)
func (e *Embedder) ExtractTextFromPage(content []byte, pageNum int) (string, error) {
	reader := bytes.NewReader(content)

	// Create a temporary directory for extraction
	tmpDir, err := os.MkdirTemp("", "pdf-extract-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// ExtractContent writes files to the output directory
	if err := api.ExtractContent(reader, tmpDir, "content", []string{fmt.Sprintf("%d", pageNum)}, e.conf); err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	// Read the extracted content file
	outputFile := filepath.Join(tmpDir, fmt.Sprintf("content_page_%d.txt", pageNum))
	data, err := os.ReadFile(outputFile)
	if err != nil {
		// Try alternate naming convention
		outputFile = filepath.Join(tmpDir, "content.txt")
		data, err = os.ReadFile(outputFile)
		if err != nil {
			return "", fmt.Errorf("failed to read extracted content: %w", err)
		}
	}

	return string(data), nil
}
