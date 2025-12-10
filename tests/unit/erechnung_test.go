package unit

import (
	"testing"
	"time"

	"austrian-business-infrastructure/internal/erechnung"
)

// T087: Test Invoice struct to XRechnung XML
func TestInvoiceToXRechnungXML(t *testing.T) {
	invoice := createTestInvoice()

	xmlData, err := erechnung.GenerateXRechnung(invoice)
	if err != nil {
		t.Fatalf("Failed to generate XRechnung: %v", err)
	}

	xmlStr := string(xmlData)

	// Verify UBL structure
	if !contains(xmlStr, "<Invoice") {
		t.Error("Missing Invoice root element")
	}
	if !contains(xmlStr, "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2") {
		t.Error("Missing UBL namespace")
	}
	if !contains(xmlStr, "<cbc:ID>INV-2025-001</cbc:ID>") {
		t.Error("Missing or incorrect invoice ID")
	}
	if !contains(xmlStr, "<cbc:InvoiceTypeCode>380</cbc:InvoiceTypeCode>") {
		t.Error("Missing or incorrect InvoiceTypeCode")
	}
	if !contains(xmlStr, "Acme GmbH") {
		t.Error("Missing seller name")
	}
}

// T088: Test Invoice struct to ZUGFeRD XML
func TestInvoiceToZUGFeRDXML(t *testing.T) {
	invoice := createTestInvoice()

	xmlData, err := erechnung.GenerateZUGFeRD(invoice)
	if err != nil {
		t.Fatalf("Failed to generate ZUGFeRD: %v", err)
	}

	xmlStr := string(xmlData)

	// Verify CII structure
	if !contains(xmlStr, "CrossIndustryInvoice") {
		t.Error("Missing CrossIndustryInvoice root element")
	}
	if !contains(xmlStr, "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice") {
		t.Error("Missing CII namespace")
	}
	if !contains(xmlStr, "INV-2025-001") {
		t.Error("Missing invoice ID")
	}
}

// T089: Test EN16931 validation rules
func TestEN16931Validation(t *testing.T) {
	// Valid invoice
	invoice := createTestInvoice()
	result := erechnung.ValidateEN16931(invoice)
	if !result.Valid {
		t.Errorf("Expected valid invoice, got errors: %v", result.Errors)
	}

	// Invalid: missing seller
	invalidInvoice := createTestInvoice()
	invalidInvoice.Seller = nil
	result = erechnung.ValidateEN16931(invalidInvoice)
	if result.Valid {
		t.Error("Expected validation error for missing seller")
	}
	if !containsError(result.Errors, "BR-06") {
		t.Error("Expected BR-06 error for missing seller")
	}

	// Invalid: missing buyer
	invalidInvoice = createTestInvoice()
	invalidInvoice.Buyer = nil
	result = erechnung.ValidateEN16931(invalidInvoice)
	if result.Valid {
		t.Error("Expected validation error for missing buyer")
	}

	// Invalid: no invoice lines
	invalidInvoice = createTestInvoice()
	invalidInvoice.Lines = nil
	result = erechnung.ValidateEN16931(invalidInvoice)
	if result.Valid {
		t.Error("Expected validation error for missing invoice lines")
	}

	// Invalid: missing tax
	invalidInvoice = createTestInvoice()
	invalidInvoice.Lines[0].TaxPercent = 0
	invalidInvoice.Lines[0].TaxCategory = ""
	result = erechnung.ValidateEN16931(invalidInvoice)
	if result.Valid {
		t.Error("Expected validation error for missing tax info")
	}
}

// T090: Test invoice calculation (totals, tax)
func TestInvoiceCalculation(t *testing.T) {
	invoice := createTestInvoice()

	// Calculate totals
	if err := invoice.CalculateTotals(); err != nil {
		t.Fatalf("Failed to calculate totals: %v", err)
	}

	// Line 1: 10 * 100.00 = 1000.00 EUR
	if invoice.Lines[0].LineTotal != 100000 {
		t.Errorf("Expected line total 100000 (1000.00), got %d", invoice.Lines[0].LineTotal)
	}

	// Line 2: 5 * 50.00 = 250.00 EUR
	if invoice.Lines[1].LineTotal != 25000 {
		t.Errorf("Expected line total 25000 (250.00), got %d", invoice.Lines[1].LineTotal)
	}

	// Net total: 1000.00 + 250.00 = 1250.00
	if invoice.TaxExclusiveAmount != 125000 {
		t.Errorf("Expected TaxExclusiveAmount 125000 (1250.00), got %d", invoice.TaxExclusiveAmount)
	}

	// Tax: 1000.00 * 20% + 250.00 * 10% = 200.00 + 25.00 = 225.00
	if invoice.TaxAmount != 22500 {
		t.Errorf("Expected TaxAmount 22500 (225.00), got %d", invoice.TaxAmount)
	}

	// Gross total: 1250.00 + 225.00 = 1475.00
	if invoice.TaxInclusiveAmount != 147500 {
		t.Errorf("Expected TaxInclusiveAmount 147500 (1475.00), got %d", invoice.TaxInclusiveAmount)
	}

	// Payable: 1475.00
	if invoice.PayableAmount != 147500 {
		t.Errorf("Expected PayableAmount 147500 (1475.00), got %d", invoice.PayableAmount)
	}
}

// Test Invoice type constants
func TestInvoiceTypeConstants(t *testing.T) {
	types := []erechnung.InvoiceType{
		erechnung.InvoiceTypeCommercial,
		erechnung.InvoiceTypeCreditNote,
		erechnung.InvoiceTypeSelfBilled,
	}

	for _, it := range types {
		if it == "" {
			t.Error("InvoiceType constant is empty")
		}
	}
}

// Test TaxCategory constants
func TestTaxCategoryConstants(t *testing.T) {
	categories := []string{
		erechnung.TaxCategoryStandard,
		erechnung.TaxCategoryReduced,
		erechnung.TaxCategoryZero,
		erechnung.TaxCategoryExempt,
		erechnung.TaxCategoryReverseCharge,
	}

	for _, cat := range categories {
		if cat == "" {
			t.Error("TaxCategory constant is empty")
		}
	}
}

// Test PaymentMeans constants
func TestPaymentMeansConstants(t *testing.T) {
	means := []erechnung.PaymentMeansCode{
		erechnung.PaymentBankTransfer,
		erechnung.PaymentDirectDebit,
		erechnung.PaymentCreditCard,
		erechnung.PaymentCash,
	}

	for _, m := range means {
		if m == "" {
			t.Error("PaymentMeansCode constant is empty")
		}
	}
}

// Test InvoiceParty
func TestInvoiceParty(t *testing.T) {
	party := &erechnung.InvoiceParty{
		Name:       "Test GmbH",
		Street:     "Hauptstraße 1",
		City:       "Wien",
		PostalCode: "1010",
		Country:    "AT",
		VATNumber:  "ATU12345678",
		Email:      "info@test.at",
	}

	if party.Name != "Test GmbH" {
		t.Errorf("Expected Name Test GmbH, got %s", party.Name)
	}
	if party.VATNumber != "ATU12345678" {
		t.Errorf("Expected VATNumber ATU12345678, got %s", party.VATNumber)
	}
}

// Test InvoiceLine
func TestInvoiceLine(t *testing.T) {
	line := &erechnung.InvoiceLine{
		ID:          "1",
		Description: "Consulting services",
		Quantity:    10,
		UnitCode:    "HUR", // Hour
		UnitPrice:   10000, // 100.00 EUR
		TaxPercent:  20,
		TaxCategory: erechnung.TaxCategoryStandard,
	}

	// Calculate line total
	line.CalculateTotal()

	if line.LineTotal != 100000 {
		t.Errorf("Expected LineTotal 100000, got %d", line.LineTotal)
	}
}

// Test NewInvoice
func TestNewInvoice(t *testing.T) {
	invoice := erechnung.NewInvoice()

	if invoice.InvoiceType != erechnung.InvoiceTypeCommercial {
		t.Errorf("Expected InvoiceType 380, got %s", invoice.InvoiceType)
	}
	if invoice.Currency != "EUR" {
		t.Errorf("Expected Currency EUR, got %s", invoice.Currency)
	}
	if invoice.IssueDate.IsZero() {
		t.Error("IssueDate should be set")
	}
}

// Test TaxSubtotal
func TestTaxSubtotal(t *testing.T) {
	subtotal := &erechnung.TaxSubtotal{
		TaxableAmount: 100000,
		TaxAmount:     20000,
		TaxCategory:   erechnung.TaxCategoryStandard,
		TaxPercent:    20,
	}

	if subtotal.TaxableAmount != 100000 {
		t.Errorf("Expected TaxableAmount 100000, got %d", subtotal.TaxableAmount)
	}
	if subtotal.TaxAmount != 20000 {
		t.Errorf("Expected TaxAmount 20000, got %d", subtotal.TaxAmount)
	}
}

// Test ParseInvoiceJSON
func TestParseInvoiceJSON(t *testing.T) {
	jsonData := `{
		"id": "INV-TEST-001",
		"invoice_type": "380",
		"issue_date": "2025-01-15",
		"currency": "EUR",
		"seller": {
			"name": "Seller GmbH",
			"country": "AT",
			"vat_number": "ATU12345678"
		},
		"buyer": {
			"name": "Buyer AG",
			"country": "DE"
		},
		"lines": [
			{
				"id": "1",
				"description": "Product A",
				"quantity": 5,
				"unit_price": 10000,
				"tax_percent": 20,
				"tax_category": "S"
			}
		]
	}`

	invoice, err := erechnung.ParseInvoiceJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if invoice.ID != "INV-TEST-001" {
		t.Errorf("Expected ID INV-TEST-001, got %s", invoice.ID)
	}
	if invoice.Seller.Name != "Seller GmbH" {
		t.Errorf("Expected Seller name Seller GmbH, got %s", invoice.Seller.Name)
	}
	if len(invoice.Lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(invoice.Lines))
	}
}

// Helper function to create a test invoice
func createTestInvoice() *erechnung.Invoice {
	return &erechnung.Invoice{
		ID:          "INV-2025-001",
		InvoiceType: erechnung.InvoiceTypeCommercial,
		IssueDate:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		DueDate:     time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC),
		Currency:    "EUR",
		Seller: &erechnung.InvoiceParty{
			Name:       "Acme GmbH",
			Street:     "Hauptstraße 1",
			City:       "Wien",
			PostalCode: "1010",
			Country:    "AT",
			VATNumber:  "ATU12345678",
		},
		Buyer: &erechnung.InvoiceParty{
			Name:       "Customer AG",
			Street:     "Nebenstraße 2",
			City:       "Graz",
			PostalCode: "8010",
			Country:    "AT",
			VATNumber:  "ATU87654321",
		},
		Lines: []*erechnung.InvoiceLine{
			{
				ID:          "1",
				Description: "Consulting services",
				Quantity:    10,
				UnitCode:    "HUR",
				UnitPrice:   10000, // 100.00 EUR
				TaxPercent:  20,
				TaxCategory: erechnung.TaxCategoryStandard,
			},
			{
				ID:          "2",
				Description: "Software license",
				Quantity:    5,
				UnitCode:    "C62",
				UnitPrice:   5000, // 50.00 EUR
				TaxPercent:  10,
				TaxCategory: erechnung.TaxCategoryReduced,
			},
		},
		PaymentMeans: erechnung.PaymentBankTransfer,
		PaymentTerms: "Net 30 days",
		BankAccount: &erechnung.BankAccount{
			IBAN: "AT611904300234573201",
			BIC:  "BKAUATWW",
			Name: "Acme GmbH",
		},
	}
}

// Helper function to check if errors contain a specific code
func containsError(errors []erechnung.ValidationError, code string) bool {
	for _, e := range errors {
		if e.Code == code || contains(e.Message, code) {
			return true
		}
	}
	return false
}
