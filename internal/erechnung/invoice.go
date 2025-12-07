package erechnung

import (
	"encoding/json"
	"strings"
	"time"
)

// DateOnly is a custom type for parsing dates in YYYY-MM-DD format
type DateOnly time.Time

// UnmarshalJSON implements json.Unmarshaler for DateOnly
func (d *DateOnly) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "" || s == "null" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	*d = DateOnly(t)
	return nil
}

// MarshalJSON implements json.Marshaler for DateOnly
func (d DateOnly) MarshalJSON() ([]byte, error) {
	return []byte("\"" + time.Time(d).Format("2006-01-02") + "\""), nil
}

// Time returns the underlying time.Time
func (d DateOnly) Time() time.Time {
	return time.Time(d)
}

// InvoiceType represents the type of invoice (UNTDID 1001)
type InvoiceType string

const (
	InvoiceTypeCommercial InvoiceType = "380" // Commercial invoice
	InvoiceTypeCreditNote InvoiceType = "381" // Credit note
	InvoiceTypeSelfBilled InvoiceType = "389" // Self-billed invoice
)

// TaxCategory codes (UNCL5305)
const (
	TaxCategoryStandard      = "S"  // Standard rate
	TaxCategoryReduced       = "AA" // Lower rate (10% in Austria)
	TaxCategoryZero          = "Z"  // Zero rated goods
	TaxCategoryExempt        = "E"  // Exempt from tax
	TaxCategoryReverseCharge = "AE" // VAT Reverse Charge
)

// PaymentMeansCode (UNCL4461)
type PaymentMeansCode string

const (
	PaymentBankTransfer PaymentMeansCode = "30" // Credit transfer
	PaymentDirectDebit  PaymentMeansCode = "49" // Direct debit
	PaymentCreditCard   PaymentMeansCode = "48" // Bank card
	PaymentCash         PaymentMeansCode = "10" // Cash
)

// Invoice represents an EN16931-compliant electronic invoice
type Invoice struct {
	// BG-1: Invoice number
	ID string `json:"id"`

	// BT-3: Invoice type code
	InvoiceType InvoiceType `json:"invoice_type"`

	// BT-2: Issue date
	IssueDate time.Time `json:"issue_date"`

	// BT-9: Due date
	DueDate time.Time `json:"due_date,omitempty"`

	// BT-5: Invoice currency code
	Currency string `json:"currency"`

	// BT-10: Buyer reference
	BuyerReference string `json:"buyer_reference,omitempty"`

	// BT-13: Purchase order reference
	OrderReference string `json:"order_reference,omitempty"`

	// BG-4: Seller (Supplier)
	Seller *InvoiceParty `json:"seller"`

	// BG-7: Buyer
	Buyer *InvoiceParty `json:"buyer"`

	// BG-25: Invoice lines
	Lines []*InvoiceLine `json:"lines"`

	// BT-81: Payment means type code
	PaymentMeans PaymentMeansCode `json:"payment_means,omitempty"`

	// BT-20: Payment terms
	PaymentTerms string `json:"payment_terms,omitempty"`

	// BG-17: Payment account
	BankAccount *BankAccount `json:"bank_account,omitempty"`

	// BG-23: Tax breakdown
	TaxSubtotals []*TaxSubtotal `json:"tax_subtotals,omitempty"`

	// BT-106: Sum of invoice line net amounts
	TaxExclusiveAmount int64 `json:"tax_exclusive_amount"`

	// BT-110: Invoice total VAT amount
	TaxAmount int64 `json:"tax_amount"`

	// BT-112: Invoice total with VAT
	TaxInclusiveAmount int64 `json:"tax_inclusive_amount"`

	// BT-115: Amount due for payment
	PayableAmount int64 `json:"payable_amount"`

	// BT-22: Notes
	Notes string `json:"notes,omitempty"`
}

// InvoiceParty represents a party (seller or buyer) in an invoice
type InvoiceParty struct {
	// BT-27/BT-44: Party name
	Name string `json:"name"`

	// BT-29/BT-46: Party identifier
	ID string `json:"id,omitempty"`

	// BT-35/BT-50: Address line 1
	Street string `json:"street,omitempty"`

	// BT-36/BT-51: Address line 2
	AdditionalStreet string `json:"additional_street,omitempty"`

	// BT-37/BT-52: City
	City string `json:"city,omitempty"`

	// BT-38/BT-53: Postal code
	PostalCode string `json:"postal_code,omitempty"`

	// BT-40/BT-55: Country code
	Country string `json:"country"`

	// BT-31/BT-48: VAT identifier
	VATNumber string `json:"vat_number,omitempty"`

	// BT-32/BT-47: Tax registration identifier
	TaxID string `json:"tax_id,omitempty"`

	// BT-34: Electronic address
	Email string `json:"email,omitempty"`

	// BT-42: Seller contact name
	ContactName string `json:"contact_name,omitempty"`

	// BT-43: Seller contact phone
	ContactPhone string `json:"contact_phone,omitempty"`
}

// InvoiceLine represents a single line item in an invoice
type InvoiceLine struct {
	// BT-126: Invoice line identifier
	ID string `json:"id"`

	// BT-153: Item name
	Description string `json:"description"`

	// BT-154: Item description (detailed)
	DetailedDescription string `json:"detailed_description,omitempty"`

	// BT-129: Invoiced quantity
	Quantity float64 `json:"quantity"`

	// BT-130: Invoiced quantity unit of measure
	UnitCode string `json:"unit_code"`

	// BT-146: Item net price
	UnitPrice int64 `json:"unit_price"` // In cents

	// BT-131: Invoice line net amount
	LineTotal int64 `json:"line_total"` // In cents (calculated)

	// BT-151: VAT category code
	TaxCategory string `json:"tax_category"`

	// BT-152: VAT rate
	TaxPercent float64 `json:"tax_percent"`

	// BT-155: Item seller identifier
	ItemID string `json:"item_id,omitempty"`

	// BT-157: Item standard identifier (GTIN/EAN)
	GTIN string `json:"gtin,omitempty"`
}

// CalculateTotal calculates the line total
func (l *InvoiceLine) CalculateTotal() {
	l.LineTotal = int64(float64(l.UnitPrice) * l.Quantity)
}

// TaxSubtotal represents a VAT breakdown category
type TaxSubtotal struct {
	// BT-116: VAT category taxable amount
	TaxableAmount int64 `json:"taxable_amount"` // In cents

	// BT-117: VAT category tax amount
	TaxAmount int64 `json:"tax_amount"` // In cents

	// BT-118: VAT category code
	TaxCategory string `json:"tax_category"`

	// BT-119: VAT category rate
	TaxPercent float64 `json:"tax_percent"`

	// BT-120: VAT exemption reason text
	ExemptionReason string `json:"exemption_reason,omitempty"`
}

// BankAccount represents payment account information
type BankAccount struct {
	// BT-84: Payment account identifier (IBAN)
	IBAN string `json:"iban"`

	// BT-86: Payment service provider identifier (BIC)
	BIC string `json:"bic,omitempty"`

	// BT-85: Payment account name
	Name string `json:"name,omitempty"`
}

// NewInvoice creates a new invoice with default values
func NewInvoice() *Invoice {
	return &Invoice{
		InvoiceType: InvoiceTypeCommercial,
		Currency:    "EUR",
		IssueDate:   time.Now(),
		Lines:       make([]*InvoiceLine, 0),
	}
}

// CalculateTotals calculates all invoice totals
func (inv *Invoice) CalculateTotals() error {
	// Calculate line totals
	for _, line := range inv.Lines {
		line.CalculateTotal()
	}

	// Group by tax category/rate
	taxGroups := make(map[string]*TaxSubtotal)

	for _, line := range inv.Lines {
		key := line.TaxCategory + "_" + formatFloat(line.TaxPercent)

		if _, exists := taxGroups[key]; !exists {
			taxGroups[key] = &TaxSubtotal{
				TaxCategory: line.TaxCategory,
				TaxPercent:  line.TaxPercent,
			}
		}

		taxGroups[key].TaxableAmount += line.LineTotal
	}

	// Calculate tax amounts and build subtotals
	inv.TaxSubtotals = make([]*TaxSubtotal, 0, len(taxGroups))
	inv.TaxExclusiveAmount = 0
	inv.TaxAmount = 0

	for _, ts := range taxGroups {
		ts.TaxAmount = int64(float64(ts.TaxableAmount) * ts.TaxPercent / 100)
		inv.TaxExclusiveAmount += ts.TaxableAmount
		inv.TaxAmount += ts.TaxAmount
		inv.TaxSubtotals = append(inv.TaxSubtotals, ts)
	}

	// Calculate totals
	inv.TaxInclusiveAmount = inv.TaxExclusiveAmount + inv.TaxAmount
	inv.PayableAmount = inv.TaxInclusiveAmount

	return nil
}

// formatFloat formats a float for use as a map key
func formatFloat(f float64) string {
	return string(rune(int(f * 100)))
}

// invoiceJSON is used for JSON parsing with date handling
type invoiceJSON struct {
	ID             string           `json:"id"`
	InvoiceType    InvoiceType      `json:"invoice_type"`
	IssueDate      string           `json:"issue_date"`
	DueDate        string           `json:"due_date,omitempty"`
	Currency       string           `json:"currency"`
	BuyerReference string           `json:"buyer_reference,omitempty"`
	OrderReference string           `json:"order_reference,omitempty"`
	Seller         *InvoiceParty    `json:"seller"`
	Buyer          *InvoiceParty    `json:"buyer"`
	Lines          []*InvoiceLine   `json:"lines"`
	PaymentMeans   PaymentMeansCode `json:"payment_means,omitempty"`
	PaymentTerms   string           `json:"payment_terms,omitempty"`
	BankAccount    *BankAccount     `json:"bank_account,omitempty"`
	Notes          string           `json:"notes,omitempty"`
}

// ParseInvoiceJSON parses an invoice from JSON
func ParseInvoiceJSON(data []byte) (*Invoice, error) {
	var invJSON invoiceJSON
	if err := json.Unmarshal(data, &invJSON); err != nil {
		return nil, err
	}

	invoice := &Invoice{
		ID:             invJSON.ID,
		InvoiceType:    invJSON.InvoiceType,
		Currency:       invJSON.Currency,
		BuyerReference: invJSON.BuyerReference,
		OrderReference: invJSON.OrderReference,
		Seller:         invJSON.Seller,
		Buyer:          invJSON.Buyer,
		Lines:          invJSON.Lines,
		PaymentMeans:   invJSON.PaymentMeans,
		PaymentTerms:   invJSON.PaymentTerms,
		BankAccount:    invJSON.BankAccount,
		Notes:          invJSON.Notes,
	}

	// Parse dates
	if invJSON.IssueDate != "" {
		t, err := time.Parse("2006-01-02", invJSON.IssueDate)
		if err != nil {
			return nil, err
		}
		invoice.IssueDate = t
	}

	if invJSON.DueDate != "" {
		t, err := time.Parse("2006-01-02", invJSON.DueDate)
		if err != nil {
			return nil, err
		}
		invoice.DueDate = t
	}

	return invoice, nil
}

// AmountEUR returns an amount in EUR (from cents)
func AmountEUR(cents int64) float64 {
	return float64(cents) / 100
}
