package uva

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Status constants for UVA submissions
const (
	StatusDraft     = "draft"
	StatusValidated = "validated"
	StatusSubmitted = "submitted"
	StatusAccepted  = "accepted"
	StatusRejected  = "rejected"
	StatusError     = "error"
)

// Period type constants
const (
	PeriodTypeMonthly   = "monthly"
	PeriodTypeQuarterly = "quarterly"
)

// Submission represents a UVA submission record
type Submission struct {
	ID               uuid.UUID        `json:"id"`
	TenantID         uuid.UUID        `json:"tenant_id"`
	AccountID        uuid.UUID        `json:"account_id"`
	PeriodYear       int              `json:"period_year"`
	PeriodMonth      *int             `json:"period_month,omitempty"`
	PeriodQuarter    *int             `json:"period_quarter,omitempty"`
	PeriodType       string           `json:"period_type"`
	Data             json.RawMessage  `json:"data"`
	ValidationStatus string           `json:"validation_status"`
	ValidationErrors json.RawMessage  `json:"validation_errors,omitempty"`
	Status           string           `json:"status"`
	FOReference      *string          `json:"fo_reference,omitempty"`
	XMLContent       []byte           `json:"xml_content,omitempty"`
	SubmittedAt      *time.Time       `json:"submitted_at,omitempty"`
	SubmittedBy      *uuid.UUID       `json:"submitted_by,omitempty"`
	ResponseCode     *int             `json:"response_code,omitempty"`
	ResponseMessage  *string          `json:"response_message,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// UVAData represents the form data for a UVA (all amounts in cents)
type UVAData struct {
	// Tax amounts (in cents to avoid floating point issues)
	KZ000 int64 `json:"kz000"` // Gesamtbetrag der Lieferungen (total deliveries)
	KZ001 int64 `json:"kz001"` // Innergemeinschaftliche Lieferungen
	KZ011 int64 `json:"kz011"` // Steuerfrei ohne Vorsteuerabzug
	KZ017 int64 `json:"kz017"` // Normalsteuersatz 20%
	KZ018 int64 `json:"kz018"` // Ermäßigter Steuersatz 10%
	KZ019 int64 `json:"kz019"` // Ermäßigter Steuersatz 13%
	KZ020 int64 `json:"kz020"` // Sonstige Steuersätze
	KZ022 int64 `json:"kz022"` // Einfuhrumsatzsteuer
	KZ029 int64 `json:"kz029"` // Innergemeinschaftliche Erwerbe
	KZ060 int64 `json:"kz060"` // Vorsteuer
	KZ065 int64 `json:"kz065"` // Einfuhrumsatzsteuer als Vorsteuer
	KZ066 int64 `json:"kz066"` // Vorsteuern aus IG Erwerben
	KZ070 int64 `json:"kz070"` // Sonstige Berichtigungen
	KZ095 int64 `json:"kz095"` // Zahllast/Gutschrift (calculated)
}

// Batch represents a batch UVA submission
type Batch struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	Name          string     `json:"name"`
	PeriodYear    int        `json:"period_year"`
	PeriodMonth   *int       `json:"period_month,omitempty"`
	PeriodQuarter *int       `json:"period_quarter,omitempty"`
	PeriodType    string     `json:"period_type"`
	TotalCount    int        `json:"total_count"`
	SuccessCount  int        `json:"success_count"`
	FailedCount   int        `json:"failed_count"`
	Status        string     `json:"status"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	CreatedBy     uuid.UUID  `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// BatchItem represents a single submission within a batch
type BatchItem struct {
	BatchID      uuid.UUID `json:"batch_id"`
	SubmissionID uuid.UUID `json:"submission_id"`
	AccountID    uuid.UUID `json:"account_id"`
	Status       string    `json:"status"`
	ErrorMessage *string   `json:"error_message,omitempty"`
}

// CreateSubmissionInput represents input for creating a new UVA submission
type CreateSubmissionInput struct {
	AccountID     uuid.UUID `json:"account_id"`
	PeriodYear    int       `json:"period_year"`
	PeriodMonth   *int      `json:"period_month,omitempty"`
	PeriodQuarter *int      `json:"period_quarter,omitempty"`
	PeriodType    string    `json:"period_type"`
	Data          UVAData   `json:"data"`
}

// UpdateSubmissionInput represents input for updating a UVA submission
type UpdateSubmissionInput struct {
	Data UVAData `json:"data"`
}

// SubmitInput represents input for submitting a UVA to FinanzOnline
type SubmitInput struct {
	DryRun bool `json:"dry_run"` // Validate only, don't submit
}

// ListFilter represents filtering options for listing submissions
type ListFilter struct {
	TenantID   uuid.UUID
	AccountID  *uuid.UUID
	PeriodYear *int
	PeriodType *string
	Status     *string
	Limit      int
	Offset     int
}

// SubmissionResponse is the API response format
type SubmissionResponse struct {
	ID               uuid.UUID       `json:"id"`
	AccountID        uuid.UUID       `json:"account_id"`
	PeriodYear       int             `json:"period_year"`
	PeriodMonth      *int            `json:"period_month,omitempty"`
	PeriodQuarter    *int            `json:"period_quarter,omitempty"`
	PeriodType       string          `json:"period_type"`
	Data             UVAData         `json:"data"`
	ValidationStatus string          `json:"validation_status"`
	ValidationErrors json.RawMessage `json:"validation_errors,omitempty"`
	Status           string          `json:"status"`
	FOReference      *string         `json:"fo_reference,omitempty"`
	SubmittedAt      *string         `json:"submitted_at,omitempty"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
}

// BatchResponse is the API response format for batches
type BatchResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	PeriodYear    int       `json:"period_year"`
	PeriodMonth   *int      `json:"period_month,omitempty"`
	PeriodQuarter *int      `json:"period_quarter,omitempty"`
	PeriodType    string    `json:"period_type"`
	TotalCount    int       `json:"total_count"`
	SuccessCount  int       `json:"success_count"`
	FailedCount   int       `json:"failed_count"`
	Status        string    `json:"status"`
	StartedAt     *string   `json:"started_at,omitempty"`
	CompletedAt   *string   `json:"completed_at,omitempty"`
	CreatedAt     string    `json:"created_at"`
}
