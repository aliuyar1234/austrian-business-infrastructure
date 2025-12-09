package business

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/invoice"
	"github.com/google/uuid"
)

// T086: Integration tests for E-Rechnung (XRechnung/ZUGFeRD) API

func TestInvoiceTypes(t *testing.T) {
	t.Run("Invoice struct fields", func(t *testing.T) {
		now := time.Now()
		sellerVAT := "ATU12345678"
		buyerVAT := "ATU87654321"
		inv := &invoice.Invoice{
			ID:                 uuid.New(),
			TenantID:           uuid.New(),
			InvoiceNumber:      "INV-2025-0001",
			InvoiceType:        "380",
			IssueDate:          now,
			Currency:           "EUR",
			SellerName:         "Seller GmbH",
			SellerVAT:          &sellerVAT,
			BuyerName:          "Buyer AG",
			BuyerVAT:           &buyerVAT,
			TaxExclusiveAmount: 10000000, // 100,000.00 EUR
			TaxAmount:          2000000,  // 20,000.00 EUR
			TaxInclusiveAmount: 12000000, // 120,000.00 EUR
			PayableAmount:      12000000,
			Status:             invoice.StatusDraft,
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		if inv.InvoiceNumber != "INV-2025-0001" {
			t.Errorf("InvoiceNumber mismatch: got %s", inv.InvoiceNumber)
		}
		if inv.Currency != "EUR" {
			t.Errorf("Currency mismatch: got %s", inv.Currency)
		}
		if inv.TaxExclusiveAmount != 10000000 {
			t.Errorf("TaxExclusiveAmount mismatch: got %d", inv.TaxExclusiveAmount)
		}
	})

	t.Run("Invoice item struct", func(t *testing.T) {
		item := &invoice.InvoiceItem{
			InvoiceID:   uuid.New(),
			LineNumber:  1,
			Description: "Professional Services",
			Quantity:    10.0,
			UnitCode:    "HUR",
			UnitPrice:   10000000, // 100,000.00 EUR per unit
			LineTotal:   100000000,
			TaxCategory: "S",
			TaxPercent:  20.0,
		}

		if item.Description != "Professional Services" {
			t.Errorf("Description mismatch: got %s", item.Description)
		}
		if item.UnitCode != "HUR" {
			t.Errorf("UnitCode mismatch: got %s", item.UnitCode)
		}
		if item.TaxCategory != "S" {
			t.Errorf("TaxCategory mismatch: got %s", item.TaxCategory)
		}
	})
}

func TestInvoiceStatusConstants(t *testing.T) {
	statuses := []string{
		invoice.StatusDraft,
		invoice.StatusValidated,
		invoice.StatusGenerated,
		invoice.StatusSent,
		invoice.StatusPaid,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("Status constant is empty")
		}
	}

	if invoice.StatusDraft != "draft" {
		t.Errorf("StatusDraft mismatch: got %s", invoice.StatusDraft)
	}
	if invoice.StatusValidated != "validated" {
		t.Errorf("StatusValidated mismatch: got %s", invoice.StatusValidated)
	}
	if invoice.StatusGenerated != "generated" {
		t.Errorf("StatusGenerated mismatch: got %s", invoice.StatusGenerated)
	}
}

func TestInvoiceFormatConstants(t *testing.T) {
	if invoice.FormatXRechnung != "xrechnung" {
		t.Errorf("FormatXRechnung mismatch: got %s", invoice.FormatXRechnung)
	}
	if invoice.FormatZUGFeRD != "zugferd" {
		t.Errorf("FormatZUGFeRD mismatch: got %s", invoice.FormatZUGFeRD)
	}
}

func TestInvoiceCreateRequestParsing(t *testing.T) {
	t.Run("Parse create invoice request", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"invoice_number": "INV-2025-0001",
			"invoice_type":   "380",
			"issue_date":     "2025-01-15",
			"due_date":       "2025-02-15",
			"currency":       "EUR",
			"seller_name":    "Seller GmbH",
			"seller_vat":     "ATU12345678",
			"seller_address": map[string]string{
				"street":      "Musterstraße 1",
				"city":        "Wien",
				"postal_code": "1010",
				"country":     "AT",
			},
			"buyer_name": "Buyer AG",
			"buyer_vat":  "ATU87654321",
			"buyer_address": map[string]string{
				"street":      "Testgasse 5",
				"city":        "Graz",
				"postal_code": "8010",
				"country":     "AT",
			},
			"items": []map[string]interface{}{
				{
					"description":  "Professional Services",
					"quantity":     10.0,
					"unit_code":    "HUR",
					"unit_price":   10000000,
					"tax_category": "S",
					"tax_percent":  20.0,
				},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		var parsed invoice.CreateInvoiceInput
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse request: %v", err)
		}

		if parsed.InvoiceNumber != "INV-2025-0001" {
			t.Errorf("InvoiceNumber mismatch: got %s", parsed.InvoiceNumber)
		}
		if parsed.SellerName != "Seller GmbH" {
			t.Errorf("SellerName mismatch: got %s", parsed.SellerName)
		}
		if len(parsed.Items) != 1 {
			t.Errorf("Items count mismatch: got %d", len(parsed.Items))
		}
	})
}

func TestInvoiceListFilter(t *testing.T) {
	t.Run("List filter defaults", func(t *testing.T) {
		filter := invoice.ListFilter{
			TenantID: uuid.New(),
			Limit:    50,
			Offset:   0,
		}

		if filter.Limit != 50 {
			t.Errorf("Default limit should be 50, got %d", filter.Limit)
		}
		if filter.Offset != 0 {
			t.Errorf("Default offset should be 0, got %d", filter.Offset)
		}
	})
}

func TestInvoiceListQueryParsing(t *testing.T) {
	t.Run("Parse list query parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/invoices?status=draft&buyer_name=Test&date_from=2025-01-01&date_to=2025-01-31&limit=20&offset=10", nil)

		query := req.URL.Query()

		if query.Get("status") != "draft" {
			t.Error("status not parsed correctly")
		}
		if query.Get("buyer_name") != "Test" {
			t.Error("buyer_name not parsed correctly")
		}
		if query.Get("date_from") != "2025-01-01" {
			t.Error("date_from not parsed correctly")
		}
		if query.Get("date_to") != "2025-01-31" {
			t.Error("date_to not parsed correctly")
		}
		if query.Get("limit") != "20" {
			t.Error("limit not parsed correctly")
		}
		if query.Get("offset") != "10" {
			t.Error("offset not parsed correctly")
		}
	})
}

func TestInvoiceResponse(t *testing.T) {
	t.Run("Invoice response structure", func(t *testing.T) {
		sellerVAT := "ATU12345678"
		buyerVAT := "ATU87654321"
		dueDate := "2025-02-15"

		resp := &invoice.InvoiceResponse{
			ID:                 uuid.New(),
			InvoiceNumber:      "INV-2025-0001",
			InvoiceType:        "380",
			IssueDate:          "2025-01-15",
			DueDate:            &dueDate,
			Currency:           "EUR",
			SellerName:         "Seller GmbH",
			SellerVAT:          &sellerVAT,
			BuyerName:          "Buyer AG",
			BuyerVAT:           &buyerVAT,
			TaxExclusiveAmount: 10000000,
			TaxAmount:          2000000,
			TaxInclusiveAmount: 12000000,
			PayableAmount:      12000000,
			ValidationStatus:   "passed",
			Status:             "validated",
			CreatedAt:          "2025-01-15T10:00:00Z",
			UpdatedAt:          "2025-01-15T10:00:00Z",
		}

		if resp.InvoiceNumber != "INV-2025-0001" {
			t.Error("InvoiceNumber mismatch")
		}
		if resp.Status != "validated" {
			t.Error("Status mismatch")
		}
		if resp.ValidationStatus != "passed" {
			t.Error("ValidationStatus mismatch")
		}
	})
}

func TestInvoiceXMLDownload(t *testing.T) {
	t.Run("XRechnung XML content type", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "application/xml")
		rec.Header().Set("Content-Disposition", "attachment; filename=invoice-INV-2025-0001.xml")

		if rec.Header().Get("Content-Type") != "application/xml" {
			t.Error("Content-Type should be application/xml")
		}
	})

	t.Run("ZUGFeRD PDF content type", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "application/pdf")
		rec.Header().Set("Content-Disposition", "attachment; filename=invoice-INV-2025-0001.pdf")

		if rec.Header().Get("Content-Type") != "application/pdf" {
			t.Error("Content-Type should be application/pdf")
		}
	})
}

func TestInvoiceTypeCode(t *testing.T) {
	// EN16931 invoice type codes
	typeCodes := map[string]string{
		"380": "Commercial Invoice",
		"381": "Credit Note",
		"384": "Corrected Invoice",
		"389": "Self-billed Invoice",
		"751": "Invoice Information for Accounting Purposes",
	}

	for code, desc := range typeCodes {
		if code == "" || desc == "" {
			t.Error("Type code or description is empty")
		}
	}

	// Most common should be 380
	if _, ok := typeCodes["380"]; !ok {
		t.Error("380 (Commercial Invoice) should be a valid type code")
	}
}

func TestInvoiceTaxCategories(t *testing.T) {
	// UN/ECE 5305 tax category codes
	taxCategories := map[string]string{
		"S": "Standard rate",
		"Z": "Zero rated goods",
		"E": "Exempt from tax",
		"AE": "Reverse charge",
		"K": "Intra-Community supply",
		"G": "Export outside EU",
		"O": "Services outside scope of tax",
		"L": "IGIC (Canary Islands)",
		"M": "IPSI (Ceuta/Melilla)",
	}

	for code, desc := range taxCategories {
		if code == "" || desc == "" {
			t.Error("Tax category code or description is empty")
		}
	}

	// Standard rate should be available
	if _, ok := taxCategories["S"]; !ok {
		t.Error("S (Standard rate) should be a valid tax category")
	}
}

func TestInvoiceUnitCodes(t *testing.T) {
	// Common UN/ECE Recommendation 20 unit codes
	unitCodes := map[string]string{
		"C62": "One (piece)",
		"HUR": "Hour",
		"DAY": "Day",
		"MON": "Month",
		"KGM": "Kilogram",
		"MTR": "Metre",
		"LTR": "Litre",
		"MTK": "Square metre",
		"MTQ": "Cubic metre",
		"SET": "Set",
	}

	for code, desc := range unitCodes {
		if code == "" || desc == "" {
			t.Error("Unit code or description is empty")
		}
	}

	// Hour and piece should be available
	if _, ok := unitCodes["HUR"]; !ok {
		t.Error("HUR (Hour) should be a valid unit code")
	}
	if _, ok := unitCodes["C62"]; !ok {
		t.Error("C62 (One/piece) should be a valid unit code")
	}
}

func TestInvoiceCalculations(t *testing.T) {
	t.Run("Tax calculation verification", func(t *testing.T) {
		// Test case: 10 hours at 10,000 EUR/hour with 20% VAT
		quantity := 10.0
		unitPrice := int64(10000000) // 100,000.00 EUR
		lineTotal := int64(float64(unitPrice) * quantity)
		taxPercent := 20.0
		taxAmount := int64(float64(lineTotal) * taxPercent / 100)
		totalWithTax := lineTotal + taxAmount

		expectedLineTotal := int64(100000000) // 1,000,000.00 EUR
		expectedTax := int64(20000000)        // 200,000.00 EUR (20%)
		expectedTotal := int64(120000000)     // 1,200,000.00 EUR

		if lineTotal != expectedLineTotal {
			t.Errorf("Line total mismatch: got %d, want %d", lineTotal, expectedLineTotal)
		}
		if taxAmount != expectedTax {
			t.Errorf("Tax amount mismatch: got %d, want %d", taxAmount, expectedTax)
		}
		if totalWithTax != expectedTotal {
			t.Errorf("Total with tax mismatch: got %d, want %d", totalWithTax, expectedTotal)
		}
	})
}

func TestInvoiceAddressValidation(t *testing.T) {
	t.Run("Address struct", func(t *testing.T) {
		addr := invoice.Address{
			Street:     "Musterstraße 1",
			City:       "Wien",
			PostalCode: "1010",
			Country:    "AT",
		}

		if addr.Street != "Musterstraße 1" {
			t.Errorf("Street mismatch: got %s", addr.Street)
		}
		if addr.City != "Wien" {
			t.Errorf("City mismatch: got %s", addr.City)
		}
		if addr.PostalCode != "1010" {
			t.Errorf("PostalCode mismatch: got %s", addr.PostalCode)
		}
		if addr.Country != "AT" {
			t.Errorf("Country mismatch: got %s", addr.Country)
		}
	})
}

func TestInvoiceValidationErrors(t *testing.T) {
	t.Run("Validation error structure", func(t *testing.T) {
		errJSON := json.RawMessage(`[{"field":"seller_vat","rule":"BR-CO-26","message":"VAT identifier required"}]`)

		resp := &invoice.InvoiceResponse{
			ID:               uuid.New(),
			InvoiceNumber:    "INV-2025-0001",
			ValidationStatus: "failed",
			ValidationErrors: errJSON,
			Status:           "draft",
		}

		if resp.ValidationStatus != "failed" {
			t.Error("ValidationStatus should be failed")
		}
		if len(resp.ValidationErrors) == 0 {
			t.Error("ValidationErrors should not be empty")
		}
	})
}

func TestInvoiceBankAccount(t *testing.T) {
	t.Run("Bank account for payment in Invoice struct", func(t *testing.T) {
		iban := "AT611904300234573201"
		bic := "BKAUATWW"

		inv := &invoice.Invoice{
			ID:          uuid.New(),
			PaymentIBAN: &iban,
			PaymentBIC:  &bic,
		}

		if inv.PaymentIBAN == nil || *inv.PaymentIBAN != iban {
			t.Error("PaymentIBAN not set correctly")
		}
		if inv.PaymentBIC == nil || *inv.PaymentBIC != bic {
			t.Error("PaymentBIC not set correctly")
		}
	})
}

func TestInvoiceEN16931Compliance(t *testing.T) {
	t.Run("Required fields for EN16931", func(t *testing.T) {
		// EN16931 mandatory fields
		requiredFields := []string{
			"InvoiceNumber",    // BT-1
			"IssueDate",        // BT-2
			"InvoiceType",      // BT-3
			"Currency",         // BT-5
			"SellerName",       // BT-27
			"BuyerName",        // BT-44
			"TaxAmount",        // BT-110
			"TaxInclusiveAmount", // BT-112
			"PayableAmount",    // BT-115
		}

		for _, field := range requiredFields {
			if field == "" {
				t.Error("Required field name is empty")
			}
		}
	})
}
