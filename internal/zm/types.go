package zm

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Status constants for ZM submissions
const (
	StatusDraft     = "draft"
	StatusValidated = "validated"
	StatusSubmitted = "submitted"
	StatusAccepted  = "accepted"
	StatusRejected  = "rejected"
	StatusError     = "error"
)

// DeliveryType represents the type of delivery
const (
	DeliveryTypeGoods      = "L" // Lieferungen (goods)
	DeliveryTypeTriangular = "D" // Dreiecksgeschäfte (triangular transactions)
	DeliveryTypeServices   = "S" // Sonstige Leistungen (services)
)

// Submission represents a ZM submission record
type Submission struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	AccountID        uuid.UUID       `json:"account_id"`
	PeriodYear       int             `json:"period_year"`
	PeriodQuarter    int             `json:"period_quarter"`
	Entries          json.RawMessage `json:"entries"`
	EntryCount       int             `json:"entry_count"`
	TotalAmount      int64           `json:"total_amount"` // In cents
	ValidationStatus string          `json:"validation_status"`
	ValidationErrors json.RawMessage `json:"validation_errors,omitempty"`
	Status           string          `json:"status"`
	FOReference      *string         `json:"fo_reference,omitempty"`
	XMLContent       []byte          `json:"xml_content,omitempty"`
	SubmittedAt      *time.Time      `json:"submitted_at,omitempty"`
	SubmittedBy      *uuid.UUID      `json:"submitted_by,omitempty"`
	ResponseCode     *int            `json:"response_code,omitempty"`
	ResponseMessage  *string         `json:"response_message,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// Entry represents a single ZM entry
type Entry struct {
	PartnerUID   string `json:"partner_uid"`
	CountryCode  string `json:"country_code"`
	DeliveryType string `json:"delivery_type"` // L, D, or S
	Amount       int64  `json:"amount"`        // In cents
}

// CreateSubmissionInput represents input for creating a new ZM submission
type CreateSubmissionInput struct {
	AccountID     uuid.UUID `json:"account_id"`
	PeriodYear    int       `json:"period_year"`
	PeriodQuarter int       `json:"period_quarter"`
	Entries       []Entry   `json:"entries"`
}

// UpdateSubmissionInput represents input for updating a ZM submission
type UpdateSubmissionInput struct {
	Entries []Entry `json:"entries"`
}

// ListFilter represents filtering options for listing submissions
type ListFilter struct {
	TenantID   uuid.UUID
	AccountID  *uuid.UUID
	PeriodYear *int
	Status     *string
	Limit      int
	Offset     int
}

// SubmissionResponse is the API response format
type SubmissionResponse struct {
	ID               uuid.UUID       `json:"id"`
	AccountID        uuid.UUID       `json:"account_id"`
	PeriodYear       int             `json:"period_year"`
	PeriodQuarter    int             `json:"period_quarter"`
	Entries          []Entry         `json:"entries"`
	EntryCount       int             `json:"entry_count"`
	TotalAmount      int64           `json:"total_amount"`
	TotalAmountEUR   float64         `json:"total_amount_eur"`
	ValidationStatus string          `json:"validation_status"`
	ValidationErrors json.RawMessage `json:"validation_errors,omitempty"`
	Status           string          `json:"status"`
	FOReference      *string         `json:"fo_reference,omitempty"`
	SubmittedAt      *string         `json:"submitted_at,omitempty"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
}

// PeriodString returns the period in format "Q1/2025"
func (s *Submission) PeriodString() string {
	return "Q" + string(rune('0'+s.PeriodQuarter)) + "/" + string(rune('0'+s.PeriodYear/1000)) + string(rune('0'+(s.PeriodYear/100)%10)) + string(rune('0'+(s.PeriodYear/10)%10)) + string(rune('0'+s.PeriodYear%10))
}
