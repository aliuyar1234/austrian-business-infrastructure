package template

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SignatureTemplate represents a reusable signature configuration
type SignatureTemplate struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Signers     json.RawMessage `json:"signers"`       // []SignerTemplate as JSON
	Fields      json.RawMessage `json:"fields"`        // []FieldTemplate as JSON
	Settings    json.RawMessage `json:"settings"`      // TemplateSettings as JSON
	IsActive    bool            `json:"is_active"`
	UsageCount  int             `json:"usage_count"`
	CreatedBy   uuid.UUID       `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// SignerTemplate represents a signer placeholder in a template
type SignerTemplate struct {
	Role       string `json:"role"`                  // e.g., "Manager", "Kunde", "Steuerberater"
	Name       string `json:"name,omitempty"`        // Pre-filled name (optional)
	Email      string `json:"email,omitempty"`       // Pre-filled email (optional)
	Reason     string `json:"reason,omitempty"`      // Default reason
	OrderIndex int    `json:"order_index"`           // Signing order
	Required   bool   `json:"required"`              // Must be filled when applying template
}

// FieldTemplate represents a signature field placeholder in a template
type FieldTemplate struct {
	SignerRole string  `json:"signer_role"` // Links to SignerTemplate.Role
	Page       int     `json:"page"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	FieldName  string  `json:"field_name"`
	ShowName   bool    `json:"show_name"`
	ShowDate   bool    `json:"show_date"`
	ShowReason bool    `json:"show_reason"`
	FontSize   float64 `json:"font_size"`
}

// TemplateSettings contains default settings for the template
type TemplateSettings struct {
	ExpiryDays      int    `json:"expiry_days"`
	DefaultMessage  string `json:"default_message,omitempty"`
	NotifyRequester bool   `json:"notify_requester"`
	AutoRemind      bool   `json:"auto_remind"`
	RemindDays      int    `json:"remind_days"`
}

// ApplyTemplateRequest is the request to create a signature request from a template
type ApplyTemplateRequest struct {
	TemplateID uuid.UUID               `json:"template_id"`
	DocumentID uuid.UUID               `json:"document_id"`
	Title      string                  `json:"title"`
	Signers    []ApplyTemplateSignerInput `json:"signers"`
	Message    string                  `json:"message,omitempty"`
	ExpiryDays *int                    `json:"expiry_days,omitempty"` // Override template default
}

// ApplyTemplateSignerInput maps template roles to actual signers
type ApplyTemplateSignerInput struct {
	Role   string `json:"role"`   // Matches SignerTemplate.Role
	Name   string `json:"name"`
	Email  string `json:"email"`
	Reason string `json:"reason,omitempty"`
}

// GetSignerTemplates parses the signers JSON field
func (t *SignatureTemplate) GetSignerTemplates() ([]SignerTemplate, error) {
	if t.Signers == nil {
		return []SignerTemplate{}, nil
	}
	var signers []SignerTemplate
	if err := json.Unmarshal(t.Signers, &signers); err != nil {
		return nil, err
	}
	return signers, nil
}

// GetFieldTemplates parses the fields JSON field
func (t *SignatureTemplate) GetFieldTemplates() ([]FieldTemplate, error) {
	if t.Fields == nil {
		return []FieldTemplate{}, nil
	}
	var fields []FieldTemplate
	if err := json.Unmarshal(t.Fields, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

// GetSettings parses the settings JSON field
func (t *SignatureTemplate) GetSettings() (*TemplateSettings, error) {
	if t.Settings == nil {
		return &TemplateSettings{
			ExpiryDays:      14,
			NotifyRequester: true,
			AutoRemind:      true,
			RemindDays:      7,
		}, nil
	}
	var settings TemplateSettings
	if err := json.Unmarshal(t.Settings, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// SetSignerTemplates sets the signers JSON field
func (t *SignatureTemplate) SetSignerTemplates(signers []SignerTemplate) error {
	data, err := json.Marshal(signers)
	if err != nil {
		return err
	}
	t.Signers = data
	return nil
}

// SetFieldTemplates sets the fields JSON field
func (t *SignatureTemplate) SetFieldTemplates(fields []FieldTemplate) error {
	data, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	t.Fields = data
	return nil
}

// SetSettings sets the settings JSON field
func (t *SignatureTemplate) SetSettings(settings *TemplateSettings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	t.Settings = data
	return nil
}
