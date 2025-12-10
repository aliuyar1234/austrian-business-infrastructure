package signature

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RequestStatus represents the status of a signature request
type RequestStatus string

const (
	RequestStatusPending    RequestStatus = "pending"
	RequestStatusInProgress RequestStatus = "in_progress"
	RequestStatusCompleted  RequestStatus = "completed"
	RequestStatusExpired    RequestStatus = "expired"
	RequestStatusCancelled  RequestStatus = "cancelled"
)

// SignerStatus represents the status of a signer
type SignerStatus string

const (
	SignerStatusPending  SignerStatus = "pending"
	SignerStatusNotified SignerStatus = "notified"
	SignerStatusSigning  SignerStatus = "signing"
	SignerStatusSigned   SignerStatus = "signed"
	SignerStatusExpired  SignerStatus = "expired"
)

// BatchStatus represents the status of a batch signing
type BatchStatus string

const (
	BatchStatusPending        BatchStatus = "pending"
	BatchStatusSigning        BatchStatus = "signing"
	BatchStatusCompleted      BatchStatus = "completed"
	BatchStatusPartialFailure BatchStatus = "partial_failure"
	BatchStatusCancelled      BatchStatus = "cancelled"
)

// BatchItemStatus represents the status of a batch item
type BatchItemStatus string

const (
	BatchItemStatusPending BatchItemStatus = "pending"
	BatchItemStatusSigning BatchItemStatus = "signing"
	BatchItemStatusSigned  BatchItemStatus = "signed"
	BatchItemStatusFailed  BatchItemStatus = "failed"
)

// VerificationStatus represents the result of signature verification
type VerificationStatus string

const (
	VerificationStatusValid         VerificationStatus = "valid"
	VerificationStatusInvalid       VerificationStatus = "invalid"
	VerificationStatusIndeterminate VerificationStatus = "indeterminate"
	VerificationStatusUnknown       VerificationStatus = "unknown"
)

// SignatureRequest represents a request to sign a document
type SignatureRequest struct {
	ID                uuid.UUID     `json:"id"`
	TenantID          uuid.UUID     `json:"tenant_id"`
	DocumentID        uuid.UUID     `json:"document_id"`
	Name              *string       `json:"name,omitempty"`
	Message           *string       `json:"message,omitempty"`
	ExpiresAt         time.Time     `json:"expires_at"`
	Status            RequestStatus `json:"status"`
	CompletedAt       *time.Time    `json:"completed_at,omitempty"`
	IsSequential      bool          `json:"is_sequential"`
	CurrentSignerIdx  int           `json:"current_signer_index"`
	SignedDocumentID  *uuid.UUID    `json:"signed_document_id,omitempty"`
	CreatedBy         uuid.UUID     `json:"created_by"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`

	// Joined fields (populated by some queries)
	DocumentTitle string    `json:"document_title,omitempty"`
	Signers       []*Signer `json:"signers,omitempty"`
	Fields        []*Field  `json:"fields,omitempty"`
}

// Signer represents a person who needs to sign a document
type Signer struct {
	ID                  uuid.UUID    `json:"id"`
	SignatureRequestID  uuid.UUID    `json:"signature_request_id"`
	Email               string       `json:"email"`
	Name                string       `json:"name"`
	OrderIndex          int          `json:"order_index"`
	SigningToken        string       `json:"-"` // Never expose token in JSON
	TokenExpiresAt      time.Time    `json:"-"`
	TokenUsed           bool         `json:"-"`
	Status              SignerStatus `json:"status"`
	NotifiedAt          *time.Time   `json:"notified_at,omitempty"`
	SignedAt            *time.Time   `json:"signed_at,omitempty"`
	CertificateSubject  *string      `json:"certificate_subject,omitempty"`
	CertificateSerial   *string      `json:"certificate_serial,omitempty"`
	CertificateIssuer   *string      `json:"certificate_issuer,omitempty"`
	SignatureValue      *string      `json:"-"` // Don't expose in API
	IDAustriaSubject    *string      `json:"-"`
	IDAustriaBPK        *string      `json:"-"` // Never expose BPK
	ReminderCount       int          `json:"reminder_count"`
	LastReminderAt      *time.Time   `json:"last_reminder_at,omitempty"`
	CreatedAt           time.Time    `json:"created_at"`
}

// Field represents a visual signature field placement on a PDF
type Field struct {
	ID                 uuid.UUID  `json:"id"`
	SignatureRequestID uuid.UUID  `json:"signature_request_id"`
	SignerID           *uuid.UUID `json:"signer_id,omitempty"`
	Page               int        `json:"page"`
	X                  float64    `json:"x"`
	Y                  float64    `json:"y"`
	Width              float64    `json:"width"`
	Height             float64    `json:"height"`
	ShowName           bool       `json:"show_name"`
	ShowDate           bool       `json:"show_date"`
	ShowReason         bool       `json:"show_reason"`
	Reason             *string    `json:"reason,omitempty"`
	BackgroundImageURL *string    `json:"background_image_url,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// Batch represents a batch signing operation
type Batch struct {
	ID                 uuid.UUID   `json:"id"`
	TenantID           uuid.UUID   `json:"tenant_id"`
	Name               *string     `json:"name,omitempty"`
	TotalDocuments     int         `json:"total_documents"`
	Status             BatchStatus `json:"status"`
	StartedAt          *time.Time  `json:"started_at,omitempty"`
	CompletedAt        *time.Time  `json:"completed_at,omitempty"`
	SignedCount        int         `json:"signed_count"`
	FailedCount        int         `json:"failed_count"`
	SignerUserID       *uuid.UUID  `json:"signer_user_id,omitempty"`
	IDAustriaSessionID *uuid.UUID  `json:"idaustria_session_id,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`

	// Joined fields
	Items []*BatchItem `json:"items,omitempty"`
}

// BatchItem represents a single document in a batch
type BatchItem struct {
	ID               uuid.UUID       `json:"id"`
	BatchID          uuid.UUID       `json:"batch_id"`
	DocumentID       uuid.UUID       `json:"document_id"`
	Status           BatchItemStatus `json:"status"`
	SignedAt         *time.Time      `json:"signed_at,omitempty"`
	ErrorMessage     *string         `json:"error_message,omitempty"`
	SignedDocumentID *uuid.UUID      `json:"signed_document_id,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`

	// Joined fields
	DocumentTitle string `json:"document_title,omitempty"`
}

// Verification represents the result of verifying a signed PDF
type Verification struct {
	ID                 uuid.UUID          `json:"id"`
	TenantID           uuid.UUID          `json:"tenant_id"`
	DocumentID         *uuid.UUID         `json:"document_id,omitempty"`
	DocumentHash       string             `json:"document_hash"`
	OriginalFilename   *string            `json:"original_filename,omitempty"`
	IsValid            bool               `json:"is_valid"`
	VerificationStatus VerificationStatus `json:"verification_status"`
	Signatures         json.RawMessage    `json:"signatures"`
	SignatureCount     int                `json:"signature_count"`
	VerifiedAt         time.Time          `json:"verified_at"`
	VerifiedBy         *uuid.UUID         `json:"verified_by,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
}

// SignatureDetails contains details about a single signature (for JSONB storage)
type SignatureDetails struct {
	SignerName         string            `json:"signer_name"`
	SignerEmail        string            `json:"signer_email,omitempty"`
	Certificate        *CertificateInfo  `json:"certificate,omitempty"`
	Timestamp          *TimestampInfo    `json:"timestamp,omitempty"`
	SignatureAlgorithm string            `json:"signature_algorithm"`
	HashAlgorithm      string            `json:"hash_algorithm"`
	IsValid            bool              `json:"is_valid"`
	ValidationMessages []string          `json:"validation_messages,omitempty"`
}

// CertificateInfo contains certificate details
type CertificateInfo struct {
	Subject   string    `json:"subject"`
	Issuer    string    `json:"issuer"`
	Serial    string    `json:"serial"`
	ValidFrom time.Time `json:"valid_from"`
	ValidTo   time.Time `json:"valid_to"`
}

// TimestampInfo contains timestamp details
type TimestampInfo struct {
	Time      time.Time `json:"time"`
	Authority string    `json:"authority"`
}

// Template represents a reusable signature configuration
type Template struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	Name              string          `json:"name"`
	Description       *string         `json:"description,omitempty"`
	IsSequential      bool            `json:"is_sequential"`
	DefaultExpiryDays int             `json:"default_expiry_days"`
	DefaultMessage    *string         `json:"default_message,omitempty"`
	SignerTemplates   json.RawMessage `json:"signer_templates"`
	FieldTemplates    json.RawMessage `json:"field_templates"`
	UseCount          int             `json:"use_count"`
	LastUsedAt        *time.Time      `json:"last_used_at,omitempty"`
	CreatedBy         *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// SignerTemplate is a template for a signer (stored in JSONB)
type SignerTemplate struct {
	Role  string  `json:"role"`
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
	Order int     `json:"order"`
}

// FieldTemplate is a template for a signature field (stored in JSONB)
type FieldTemplate struct {
	Page       int     `json:"page"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	SignerRole string  `json:"signer_role"`
	ShowName   bool    `json:"show_name"`
	ShowDate   bool    `json:"show_date"`
	ShowReason bool    `json:"show_reason"`
	Reason     *string `json:"reason,omitempty"`
}

// AuditEvent represents an audit log entry
type AuditEvent struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           *uuid.UUID      `json:"tenant_id,omitempty"`
	SignatureRequestID *uuid.UUID      `json:"signature_request_id,omitempty"`
	SignerID           *uuid.UUID      `json:"signer_id,omitempty"`
	BatchID            *uuid.UUID      `json:"batch_id,omitempty"`
	VerificationID     *uuid.UUID      `json:"verification_id,omitempty"`
	EventType          string          `json:"event_type"`
	Details            json.RawMessage `json:"details,omitempty"`
	ActorType          *string         `json:"actor_type,omitempty"`
	ActorID            *string         `json:"actor_id,omitempty"`
	ActorIP            *string         `json:"actor_ip,omitempty"`
	ActorUserAgent     *string         `json:"actor_user_agent,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

// Usage represents signature usage for billing
type Usage struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	SignatureRequestID *uuid.UUID `json:"signature_request_id,omitempty"`
	BatchID            *uuid.UUID `json:"batch_id,omitempty"`
	SignatureCount     int        `json:"signature_count"`
	CostCents          *int       `json:"cost_cents,omitempty"`
	UsageDate          time.Time  `json:"usage_date"`
	CreatedAt          time.Time  `json:"created_at"`
}

// Audit event types
const (
	AuditEventRequestCreated      = "request_created"
	AuditEventRequestCancelled    = "request_cancelled"
	AuditEventRequestExpired      = "request_expired"
	AuditEventRequestCompleted    = "request_completed"
	AuditEventSignerNotified      = "signer_notified"
	AuditEventSignerReminded      = "signer_reminded"
	AuditEventSigningStarted      = "signing_started"
	AuditEventSigningCompleted    = "signing_completed"
	AuditEventSigningFailed       = "signing_failed"
	AuditEventBatchStarted        = "batch_started"
	AuditEventBatchCompleted      = "batch_completed"
	AuditEventVerificationDone    = "verification_performed"
)
