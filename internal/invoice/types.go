package invoice

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Status constants for invoices
const (
	StatusDraft     = "draft"
	StatusValidated = "validated"
	StatusGenerated = "generated"
	StatusSent      = "sent"
	StatusPaid      = "paid"
	StatusCancelled = "cancelled"
)

// Format constants
const (
	FormatXRechnung = "xrechnung"
	FormatZUGFeRD   = "zugferd"
)

// Invoice represents an invoice record in the database
type Invoice struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	InvoiceNumber      string          `json:"invoice_number"`
	InvoiceType        string          `json:"invoice_type"`
	IssueDate          time.Time       `json:"issue_date"`
	DueDate            *time.Time      `json:"due_date,omitempty"`
	Currency           string          `json:"currency"`
	SellerID           *uuid.UUID      `json:"seller_id,omitempty"`
	SellerName         string          `json:"seller_name"`
	SellerVAT          *string         `json:"seller_vat,omitempty"`
	SellerAddress      json.RawMessage `json:"seller_address,omitempty"`
	BuyerID            *uuid.UUID      `json:"buyer_id,omitempty"`
	BuyerName          string          `json:"buyer_name"`
	BuyerVAT           *string         `json:"buyer_vat,omitempty"`
	BuyerAddress       json.RawMessage `json:"buyer_address,omitempty"`
	BuyerReference     *string         `json:"buyer_reference,omitempty"`
	OrderReference     *string         `json:"order_reference,omitempty"`
	TaxExclusiveAmount int64           `json:"tax_exclusive_amount"`
	TaxAmount          int64           `json:"tax_amount"`
	TaxInclusiveAmount int64           `json:"tax_inclusive_amount"`
	PayableAmount      int64           `json:"payable_amount"`
	PaymentTerms       *string         `json:"payment_terms,omitempty"`
	PaymentIBAN        *string         `json:"payment_iban,omitempty"`
	PaymentBIC         *string         `json:"payment_bic,omitempty"`
	Notes              *string         `json:"notes,omitempty"`
	Status             string          `json:"status"`
	ValidationStatus   string          `json:"validation_status"`
	ValidationErrors   json.RawMessage `json:"validation_errors,omitempty"`
	XRechnungXML       []byte          `json:"xrechnung_xml,omitempty"`
	ZUGFeRDXML         []byte          `json:"zugferd_xml,omitempty"`
	PDFContent         []byte          `json:"pdf_content,omitempty"`
	CreatedBy          *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

// InvoiceItem represents an invoice line item
type InvoiceItem struct {
	ID          uuid.UUID `json:"id"`
	InvoiceID   uuid.UUID `json:"invoice_id"`
	LineNumber  int       `json:"line_number"`
	Description string    `json:"description"`
	Quantity    float64   `json:"quantity"`
	UnitCode    string    `json:"unit_code"`
	UnitPrice   int64     `json:"unit_price"`
	LineTotal   int64     `json:"line_total"`
	TaxCategory string    `json:"tax_category"`
	TaxPercent  float64   `json:"tax_percent"`
	ItemID      *string   `json:"item_id,omitempty"`
	GTIN        *string   `json:"gtin,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Address represents a postal address
type Address struct {
	Street         string `json:"street,omitempty"`
	AdditionalLine string `json:"additional_line,omitempty"`
	City           string `json:"city,omitempty"`
	PostalCode     string `json:"postal_code,omitempty"`
	Country        string `json:"country"`
}

// CreateInvoiceInput represents input for creating a new invoice
type CreateInvoiceInput struct {
	InvoiceNumber  string        `json:"invoice_number"`
	InvoiceType    string        `json:"invoice_type"`
	IssueDate      string        `json:"issue_date"`
	DueDate        *string       `json:"due_date,omitempty"`
	Currency       string        `json:"currency"`
	SellerID       *uuid.UUID    `json:"seller_id,omitempty"`
	SellerName     string        `json:"seller_name"`
	SellerVAT      *string       `json:"seller_vat,omitempty"`
	SellerAddress  *Address      `json:"seller_address,omitempty"`
	BuyerID        *uuid.UUID    `json:"buyer_id,omitempty"`
	BuyerName      string        `json:"buyer_name"`
	BuyerVAT       *string       `json:"buyer_vat,omitempty"`
	BuyerAddress   *Address      `json:"buyer_address,omitempty"`
	BuyerReference *string       `json:"buyer_reference,omitempty"`
	OrderReference *string       `json:"order_reference,omitempty"`
	PaymentTerms   *string       `json:"payment_terms,omitempty"`
	PaymentIBAN    *string       `json:"payment_iban,omitempty"`
	PaymentBIC     *string       `json:"payment_bic,omitempty"`
	Notes          *string       `json:"notes,omitempty"`
	Items          []ItemInput   `json:"items"`
}

// ItemInput represents input for creating an invoice item
type ItemInput struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitCode    string  `json:"unit_code"`
	UnitPrice   int64   `json:"unit_price"`
	TaxCategory string  `json:"tax_category"`
	TaxPercent  float64 `json:"tax_percent"`
	ItemID      *string `json:"item_id,omitempty"`
	GTIN        *string `json:"gtin,omitempty"`
}

// ListFilter represents filtering options for listing invoices
type ListFilter struct {
	TenantID    uuid.UUID
	Status      *string
	BuyerID     *uuid.UUID
	SellerID    *uuid.UUID
	DateFrom    *time.Time
	DateTo      *time.Time
	Search      *string
	Limit       int
	Offset      int
}

// InvoiceResponse is the API response format
type InvoiceResponse struct {
	ID                 uuid.UUID       `json:"id"`
	InvoiceNumber      string          `json:"invoice_number"`
	InvoiceType        string          `json:"invoice_type"`
	IssueDate          string          `json:"issue_date"`
	DueDate            *string         `json:"due_date,omitempty"`
	Currency           string          `json:"currency"`
	SellerName         string          `json:"seller_name"`
	SellerVAT          *string         `json:"seller_vat,omitempty"`
	BuyerName          string          `json:"buyer_name"`
	BuyerVAT           *string         `json:"buyer_vat,omitempty"`
	BuyerReference     *string         `json:"buyer_reference,omitempty"`
	TaxExclusiveAmount float64         `json:"tax_exclusive_amount"`
	TaxAmount          float64         `json:"tax_amount"`
	TaxInclusiveAmount float64         `json:"tax_inclusive_amount"`
	PayableAmount      float64         `json:"payable_amount"`
	Status             string          `json:"status"`
	ValidationStatus   string          `json:"validation_status"`
	ValidationErrors   json.RawMessage `json:"validation_errors,omitempty"`
	HasXRechnung       bool            `json:"has_xrechnung"`
	HasZUGFeRD         bool            `json:"has_zugferd"`
	HasPDF             bool            `json:"has_pdf"`
	Items              []ItemResponse  `json:"items,omitempty"`
	CreatedAt          string          `json:"created_at"`
	UpdatedAt          string          `json:"updated_at"`
}

// ItemResponse is the API response format for invoice items
type ItemResponse struct {
	ID          uuid.UUID `json:"id"`
	LineNumber  int       `json:"line_number"`
	Description string    `json:"description"`
	Quantity    float64   `json:"quantity"`
	UnitCode    string    `json:"unit_code"`
	UnitPrice   float64   `json:"unit_price"`
	LineTotal   float64   `json:"line_total"`
	TaxCategory string    `json:"tax_category"`
	TaxPercent  float64   `json:"tax_percent"`
}
