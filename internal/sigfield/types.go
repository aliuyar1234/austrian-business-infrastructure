package sigfield

import (
	"time"

	"github.com/google/uuid"
)

// SignatureField represents a signature field position on a PDF page
type SignatureField struct {
	ID           uuid.UUID  `json:"id"`
	DocumentID   uuid.UUID  `json:"document_id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	SignerID     *uuid.UUID `json:"signer_id,omitempty"`
	Page         int        `json:"page"`
	X            float64    `json:"x"`
	Y            float64    `json:"y"`
	Width        float64    `json:"width"`
	Height       float64    `json:"height"`
	FieldName    string     `json:"field_name"`
	Required     bool       `json:"required"`
	ShowName     bool       `json:"show_name"`
	ShowDate     bool       `json:"show_date"`
	ShowReason   bool       `json:"show_reason"`
	CustomText   string     `json:"custom_text,omitempty"`
	FontSize     float64    `json:"font_size"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// SignatureAppearance defines the visual appearance of a signature
type SignatureAppearance struct {
	SignerName    string    `json:"signer_name"`
	SignedAt      time.Time `json:"signed_at"`
	Reason        string    `json:"reason,omitempty"`
	Location      string    `json:"location,omitempty"`
	ContactInfo   string    `json:"contact_info,omitempty"`
	ShowLogo      bool      `json:"show_logo"`
	ShowBorder    bool      `json:"show_border"`
	BackgroundRGB [3]int    `json:"background_rgb"`
	TextRGB       [3]int    `json:"text_rgb"`
	FontSize      float64   `json:"font_size"`
	DateFormat    string    `json:"date_format"`
}

// EmbedOptions specifies options for embedding a signature
type EmbedOptions struct {
	Field           *SignatureField      `json:"field,omitempty"`
	Appearance      *SignatureAppearance `json:"appearance"`
	SignatureData   []byte               `json:"-"` // PKCS#7/CMS signature data
	CertificateData []byte               `json:"-"` // X.509 certificate
	Timestamp       []byte               `json:"-"` // RFC 3161 timestamp token
}

// EmbedResult contains the result of embedding a signature
type EmbedResult struct {
	DocumentHash   string `json:"document_hash"`
	SignedDocument []byte `json:"-"`
	SignatureID    string `json:"signature_id"`
}

// PageInfo contains information about a PDF page
type PageInfo struct {
	PageNumber int     `json:"page_number"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
}

// DocumentInfo contains information about a PDF document
type DocumentInfo struct {
	PageCount    int        `json:"page_count"`
	Pages        []PageInfo `json:"pages"`
	Title        string     `json:"title,omitempty"`
	Author       string     `json:"author,omitempty"`
	IsSigned     bool       `json:"is_signed"`
	IsEncrypted  bool       `json:"is_encrypted"`
	IsPDFA       bool       `json:"is_pdfa"`
}

// DefaultAppearance returns a default signature appearance
func DefaultAppearance(signerName string, signedAt time.Time) *SignatureAppearance {
	return &SignatureAppearance{
		SignerName:    signerName,
		SignedAt:      signedAt,
		ShowLogo:      false,
		ShowBorder:    true,
		BackgroundRGB: [3]int{255, 255, 255},
		TextRGB:       [3]int{0, 0, 0},
		FontSize:      10,
		DateFormat:    "02.01.2006 15:04",
	}
}

// DefaultField returns a default signature field at the bottom of page 1
func DefaultField(documentID, tenantID uuid.UUID) *SignatureField {
	return &SignatureField{
		ID:         uuid.New(),
		DocumentID: documentID,
		TenantID:   tenantID,
		Page:       1,
		X:          50,
		Y:          50,
		Width:      200,
		Height:     60,
		FieldName:  "Signature",
		Required:   true,
		ShowName:   true,
		ShowDate:   true,
		ShowReason: true,
		FontSize:   10,
	}
}
