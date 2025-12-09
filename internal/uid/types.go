package uid

import (
	"time"

	"github.com/google/uuid"
)

// ValidationLevel constants
const (
	Level1 = 1 // Basic validation - just checks if UID is registered
	Level2 = 2 // Extended validation - includes company name and address
)

// Validation represents a UID validation record
type Validation struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	UID          string     `json:"uid"`
	CountryCode  string     `json:"country_code"`
	Valid        bool       `json:"valid"`
	Level        int        `json:"level"`
	CompanyName  *string    `json:"company_name,omitempty"`
	Street       *string    `json:"street,omitempty"`
	PostCode     *string    `json:"post_code,omitempty"`
	City         *string    `json:"city,omitempty"`
	Country      *string    `json:"country,omitempty"`
	ErrorCode    *int       `json:"error_code,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	Source       string     `json:"source"` // "finanzonline" or "vies"
	ValidatedAt  time.Time  `json:"validated_at"`
	ValidatedBy  *uuid.UUID `json:"validated_by,omitempty"`
	AccountID    *uuid.UUID `json:"account_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ValidateInput represents input for validating a UID
type ValidateInput struct {
	UID       string    `json:"uid"`
	Level     int       `json:"level"` // 1 or 2
	AccountID uuid.UUID `json:"account_id"`
}

// ValidateBatchInput represents input for batch validation
type ValidateBatchInput struct {
	UIDs      []string  `json:"uids"`
	Level     int       `json:"level"`
	AccountID uuid.UUID `json:"account_id"`
}

// ListFilter represents filtering options for listing validations
type ListFilter struct {
	TenantID    uuid.UUID
	AccountID   *uuid.UUID
	UID         *string
	Valid       *bool
	CountryCode *string
	DateFrom    *time.Time
	DateTo      *time.Time
	Limit       int
	Offset      int
}

// ValidationResponse is the API response format
type ValidationResponse struct {
	ID           uuid.UUID `json:"id"`
	UID          string    `json:"uid"`
	CountryCode  string    `json:"country_code"`
	Valid        bool      `json:"valid"`
	Level        int       `json:"level"`
	CompanyName  *string   `json:"company_name,omitempty"`
	Street       *string   `json:"street,omitempty"`
	PostCode     *string   `json:"post_code,omitempty"`
	City         *string   `json:"city,omitempty"`
	Country      *string   `json:"country,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	Source       string    `json:"source"`
	ValidatedAt  string    `json:"validated_at"`
	CreatedAt    string    `json:"created_at"`
}

// FormatValidationResult represents a format check result
type FormatValidationResult struct {
	UID         string `json:"uid"`
	Valid       bool   `json:"valid"`
	CountryCode string `json:"country_code,omitempty"`
	Error       string `json:"error,omitempty"`
}

// BatchValidationResponse is the API response for batch validation
type BatchValidationResponse struct {
	Total      int                   `json:"total"`
	Valid      int                   `json:"valid"`
	Invalid    int                   `json:"invalid"`
	Results    []*ValidationResponse `json:"results"`
	ProcessedAt string               `json:"processed_at"`
}
