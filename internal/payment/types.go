package payment

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Batch status constants
const (
	StatusDraft     = "draft"
	StatusValidated = "validated"
	StatusGenerated = "generated"
	StatusSent      = "sent"
	StatusProcessed = "processed"
	StatusFailed    = "failed"
)

// Batch type constants
const (
	TypeCreditTransfer = "pain.001"
	TypeDirectDebit    = "pain.008"
)

// Batch represents a payment batch (pain.001 or pain.008)
type Batch struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	Name             string          `json:"name"`
	Type             string          `json:"type"` // pain.001 or pain.008
	DebtorName       string          `json:"debtor_name"`
	DebtorIBAN       string          `json:"debtor_iban"`
	DebtorBIC        *string         `json:"debtor_bic,omitempty"`
	CreditorID       *string         `json:"creditor_id,omitempty"` // For pain.008
	ExecutionDate    *time.Time      `json:"execution_date,omitempty"`
	ItemCount        int             `json:"item_count"`
	TotalAmount      int64           `json:"total_amount"` // In cents
	Status           string          `json:"status"`
	ValidationErrors json.RawMessage `json:"validation_errors,omitempty"`
	XMLContent       []byte          `json:"xml_content,omitempty"`
	GeneratedAt      *time.Time      `json:"generated_at,omitempty"`
	SentAt           *time.Time      `json:"sent_at,omitempty"`
	CreatedBy        *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// Item represents a single payment in a batch
type Item struct {
	ID              uuid.UUID  `json:"id"`
	BatchID         uuid.UUID  `json:"batch_id"`
	EndToEndID      string     `json:"end_to_end_id"`
	Amount          int64      `json:"amount"` // In cents
	Currency        string     `json:"currency"`
	CreditorName    string     `json:"creditor_name"`
	CreditorIBAN    string     `json:"creditor_iban"`
	CreditorBIC     *string    `json:"creditor_bic,omitempty"`
	RemittanceInfo  *string    `json:"remittance_info,omitempty"`
	Purpose         *string    `json:"purpose,omitempty"`
	MandateID       *string    `json:"mandate_id,omitempty"`       // For pain.008
	MandateDate     *time.Time `json:"mandate_date,omitempty"`     // For pain.008
	SequenceType    *string    `json:"sequence_type,omitempty"`    // For pain.008
	Status          string     `json:"status"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// CreateBatchInput represents input for creating a payment batch
type CreateBatchInput struct {
	Name          string      `json:"name"`
	Type          string      `json:"type"` // pain.001 or pain.008
	DebtorName    string      `json:"debtor_name"`
	DebtorIBAN    string      `json:"debtor_iban"`
	DebtorBIC     *string     `json:"debtor_bic,omitempty"`
	CreditorID    *string     `json:"creditor_id,omitempty"`
	ExecutionDate *string     `json:"execution_date,omitempty"`
	Items         []ItemInput `json:"items"`
}

// ItemInput represents input for creating a payment item
type ItemInput struct {
	EndToEndID     string  `json:"end_to_end_id"`
	Amount         int64   `json:"amount"` // In cents
	Currency       string  `json:"currency"`
	CreditorName   string  `json:"creditor_name"`
	CreditorIBAN   string  `json:"creditor_iban"`
	CreditorBIC    *string `json:"creditor_bic,omitempty"`
	RemittanceInfo *string `json:"remittance_info,omitempty"`
	Purpose        *string `json:"purpose,omitempty"`
	MandateID      *string `json:"mandate_id,omitempty"`
	MandateDate    *string `json:"mandate_date,omitempty"`
	SequenceType   *string `json:"sequence_type,omitempty"`
}

// ListFilter represents filtering options
type ListFilter struct {
	TenantID uuid.UUID
	Type     *string
	Status   *string
	DateFrom *time.Time
	DateTo   *time.Time
	Limit    int
	Offset   int
}

// BatchResponse is the API response format
type BatchResponse struct {
	ID               uuid.UUID       `json:"id"`
	Name             string          `json:"name"`
	Type             string          `json:"type"`
	DebtorName       string          `json:"debtor_name"`
	DebtorIBAN       string          `json:"debtor_iban"`
	ExecutionDate    *string         `json:"execution_date,omitempty"`
	ItemCount        int             `json:"item_count"`
	TotalAmount      float64         `json:"total_amount"`
	Status           string          `json:"status"`
	ValidationErrors json.RawMessage `json:"validation_errors,omitempty"`
	HasXML           bool            `json:"has_xml"`
	GeneratedAt      *string         `json:"generated_at,omitempty"`
	SentAt           *string         `json:"sent_at,omitempty"`
	Items            []ItemResponse  `json:"items,omitempty"`
	CreatedAt        string          `json:"created_at"`
}

// ItemResponse is the API response format for items
type ItemResponse struct {
	ID             uuid.UUID `json:"id"`
	EndToEndID     string    `json:"end_to_end_id"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	CreditorName   string    `json:"creditor_name"`
	CreditorIBAN   string    `json:"creditor_iban"`
	RemittanceInfo *string   `json:"remittance_info,omitempty"`
	Status         string    `json:"status"`
	ErrorMessage   *string   `json:"error_message,omitempty"`
}

// BankStatement represents an imported bank statement (camt.053)
type BankStatement struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	IBAN           string    `json:"iban"`
	StatementID    string    `json:"statement_id"`
	StatementDate  time.Time `json:"statement_date"`
	OpeningBalance int64     `json:"opening_balance"` // In cents
	ClosingBalance int64     `json:"closing_balance"` // In cents
	EntryCount     int       `json:"entry_count"`
	ImportedAt     time.Time `json:"imported_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// Transaction represents a bank statement transaction
type Transaction struct {
	ID               uuid.UUID  `json:"id"`
	StatementID      uuid.UUID  `json:"statement_id"`
	Amount           int64      `json:"amount"` // In cents
	Currency         string     `json:"currency"`
	CreditDebit      string     `json:"credit_debit"` // CRDT or DBIT
	BookingDate      time.Time  `json:"booking_date"`
	ValueDate        *time.Time `json:"value_date,omitempty"`
	Reference        *string    `json:"reference,omitempty"`
	EndToEndID       *string    `json:"end_to_end_id,omitempty"`
	RemittanceInfo   *string    `json:"remittance_info,omitempty"`
	CounterpartyName *string    `json:"counterparty_name,omitempty"`
	CounterpartyIBAN *string    `json:"counterparty_iban,omitempty"`
	MatchedPaymentID *uuid.UUID `json:"matched_payment_id,omitempty"`
	MatchedInvoiceID *uuid.UUID `json:"matched_invoice_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}
