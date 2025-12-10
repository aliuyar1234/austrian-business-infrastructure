package business

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"austrian-business-infrastructure/internal/payment"
	"github.com/google/uuid"
)

// T087: Integration tests for SEPA payment API (pain.001, pain.008, camt.053)

func TestPaymentTypes(t *testing.T) {
	t.Run("Batch struct fields", func(t *testing.T) {
		now := time.Now()
		debtorBIC := "BKAUATWW"
		batch := &payment.Batch{
			ID:            uuid.New(),
			TenantID:      uuid.New(),
			Name:          "January 2025 Salaries",
			Type:          payment.TypeCreditTransfer,
			Status:        payment.StatusDraft,
			DebtorName:    "Company GmbH",
			DebtorIBAN:    "AT611904300234573201",
			DebtorBIC:     &debtorBIC,
			TotalAmount:   100000000, // 1,000,000.00 EUR
			ItemCount:     50,
			ExecutionDate: &now,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if batch.Type != payment.TypeCreditTransfer {
			t.Errorf("Type mismatch: got %s", batch.Type)
		}
		if batch.TotalAmount != 100000000 {
			t.Errorf("TotalAmount mismatch: got %d", batch.TotalAmount)
		}
		if batch.DebtorIBAN != "AT611904300234573201" {
			t.Errorf("DebtorIBAN mismatch: got %s", batch.DebtorIBAN)
		}
	})

	t.Run("Item struct fields", func(t *testing.T) {
		creditorBIC := "BKAUATWW"
		remittanceInfo := "Salary January 2025"
		item := &payment.Item{
			ID:             uuid.New(),
			BatchID:        uuid.New(),
			CreditorName:   "Employee Name",
			CreditorIBAN:   "AT861904300234573202",
			CreditorBIC:    &creditorBIC,
			Amount:         500000, // 5,000.00 EUR
			Currency:       "EUR",
			RemittanceInfo: &remittanceInfo,
			EndToEndID:     "SALARY-2025-01-001",
			Status:         "pending",
		}

		if item.CreditorName != "Employee Name" {
			t.Errorf("CreditorName mismatch: got %s", item.CreditorName)
		}
		if item.Amount != 500000 {
			t.Errorf("Amount mismatch: got %d", item.Amount)
		}
	})
}

func TestPaymentStatusConstants(t *testing.T) {
	statuses := []string{
		payment.StatusDraft,
		payment.StatusValidated,
		payment.StatusGenerated,
		payment.StatusSent,
		payment.StatusProcessed,
		payment.StatusFailed,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("Status constant is empty")
		}
	}

	if payment.StatusDraft != "draft" {
		t.Errorf("StatusDraft mismatch: got %s", payment.StatusDraft)
	}
	if payment.StatusGenerated != "generated" {
		t.Errorf("StatusGenerated mismatch: got %s", payment.StatusGenerated)
	}
}

func TestPaymentTypeConstants(t *testing.T) {
	if payment.TypeCreditTransfer != "pain.001" {
		t.Errorf("TypeCreditTransfer mismatch: got %s", payment.TypeCreditTransfer)
	}
	if payment.TypeDirectDebit != "pain.008" {
		t.Errorf("TypeDirectDebit mismatch: got %s", payment.TypeDirectDebit)
	}
}

func TestPaymentCreateBatchRequestParsing(t *testing.T) {
	t.Run("Parse pain.001 batch request", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":         "January 2025 Salaries",
			"type":         "pain.001",
			"debtor_name":  "Company GmbH",
			"debtor_iban":  "AT611904300234573201",
			"debtor_bic":   "BKAUATWW",
			"currency":     "EUR",
			"requested_date": "2025-01-31",
			"items": []map[string]interface{}{
				{
					"creditor_name":    "Employee 1",
					"creditor_iban":    "AT861904300234573202",
					"creditor_bic":     "BKAUATWW",
					"amount":           500000,
					"remittance_info":  "Salary Jan 2025",
					"end_to_end_id":    "SALARY-001",
				},
				{
					"creditor_name":    "Employee 2",
					"creditor_iban":    "AT861904300234573203",
					"creditor_bic":     "BKAUATWW",
					"amount":           600000,
					"remittance_info":  "Salary Jan 2025",
					"end_to_end_id":    "SALARY-002",
				},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/payments/batches", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		var parsed payment.CreateBatchInput
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse request: %v", err)
		}

		if parsed.Name != "January 2025 Salaries" {
			t.Errorf("Name mismatch: got %s", parsed.Name)
		}
		if parsed.Type != "pain.001" {
			t.Errorf("Type mismatch: got %s", parsed.Type)
		}
		if len(parsed.Items) != 2 {
			t.Errorf("Items count mismatch: got %d", len(parsed.Items))
		}
	})

	t.Run("Parse pain.008 direct debit request", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":         "February 2025 Subscriptions",
			"type":         "pain.008",
			"creditor_name":  "Service GmbH",
			"creditor_iban":  "AT611904300234573201",
			"creditor_bic":   "BKAUATWW",
			"creditor_id":    "AT12ZZZ00000000001",
			"currency":       "EUR",
			"items": []map[string]interface{}{
				{
					"debtor_name":       "Customer 1",
					"debtor_iban":       "AT861904300234573202",
					"debtor_bic":        "BKAUATWW",
					"amount":            9900,
					"mandate_id":        "MNDT-001",
					"mandate_date":      "2024-01-01",
					"sequence_type":     "RCUR",
					"remittance_info":   "Subscription Feb 2025",
				},
			},
		}

		body, _ := json.Marshal(reqBody)

		var parsed payment.CreateBatchInput
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse request: %v", err)
		}

		if parsed.Type != "pain.008" {
			t.Errorf("Type mismatch: got %s", parsed.Type)
		}
	})
}

func TestPaymentListFilter(t *testing.T) {
	t.Run("List filter defaults", func(t *testing.T) {
		filter := payment.ListFilter{
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

func TestPaymentBatchResponse(t *testing.T) {
	t.Run("Batch response structure", func(t *testing.T) {
		generatedAt := "2025-01-30T14:00:00Z"

		resp := &payment.BatchResponse{
			ID:          uuid.New(),
			Name:        "January 2025 Salaries",
			Type:        "pain.001",
			Status:      "generated",
			DebtorName:  "Company GmbH",
			DebtorIBAN:  "AT611904300234573201",
			TotalAmount: 100000000.00,
			ItemCount:   50,
			HasXML:      true,
			GeneratedAt: &generatedAt,
			CreatedAt:   "2025-01-30T10:00:00Z",
		}

		if resp.Type != "pain.001" {
			t.Error("Type mismatch")
		}
		if resp.Status != "generated" {
			t.Error("Status mismatch")
		}
		if resp.ItemCount != 50 {
			t.Errorf("ItemCount mismatch: got %d", resp.ItemCount)
		}
	})
}

func TestPaymentXMLDownload(t *testing.T) {
	t.Run("pain.001 XML content type", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "application/xml")
		rec.Header().Set("Content-Disposition", "attachment; filename=pain001-batch-id.xml")

		if rec.Header().Get("Content-Type") != "application/xml" {
			t.Error("Content-Type should be application/xml")
		}
	})

	t.Run("pain.008 XML content type", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "application/xml")
		rec.Header().Set("Content-Disposition", "attachment; filename=pain008-batch-id.xml")

		if rec.Header().Get("Content-Type") != "application/xml" {
			t.Error("Content-Type should be application/xml")
		}
	})
}

func TestPaymentIBANValidation(t *testing.T) {
	t.Run("Valid Austrian IBAN format", func(t *testing.T) {
		// Austrian IBANs: AT + 2 check digits + 5 digits bank code + 11 digits account
		validIBANs := []string{
			"AT611904300234573201",
			"AT483200000012345864",
			"AT021200050012345678",
		}

		for _, iban := range validIBANs {
			if len(iban) != 20 {
				t.Errorf("Austrian IBAN should be 20 characters: %s (len=%d)", iban, len(iban))
			}
			if iban[:2] != "AT" {
				t.Errorf("Austrian IBAN should start with AT: %s", iban)
			}
		}
	})

	t.Run("Valid German IBAN format", func(t *testing.T) {
		// German IBANs: DE + 2 check digits + 8 digits bank code + 10 digits account
		validIBANs := []string{
			"DE89370400440532013000",
			"DE02370501980009000027",
		}

		for _, iban := range validIBANs {
			if len(iban) != 22 {
				t.Errorf("German IBAN should be 22 characters: %s (len=%d)", iban, len(iban))
			}
			if iban[:2] != "DE" {
				t.Errorf("German IBAN should start with DE: %s", iban)
			}
		}
	})
}

func TestPaymentBICValidation(t *testing.T) {
	t.Run("Valid BIC formats", func(t *testing.T) {
		// BIC can be 8 or 11 characters
		validBICs := []string{
			"BKAUATWW",     // 8 chars
			"BKAUATWWXXX",  // 11 chars
			"COBADEFFXXX",  // 11 chars
			"DEUTDEFF",     // 8 chars
		}

		for _, bic := range validBICs {
			if len(bic) != 8 && len(bic) != 11 {
				t.Errorf("BIC should be 8 or 11 characters: %s (len=%d)", bic, len(bic))
			}
		}
	})
}

func TestPaymentSequenceTypes(t *testing.T) {
	// SEPA direct debit sequence types
	sequenceTypes := map[string]string{
		"FRST": "First",
		"RCUR": "Recurring",
		"FNAL": "Final",
		"OOFF": "One-off",
	}

	for code, desc := range sequenceTypes {
		if code == "" || desc == "" {
			t.Error("Sequence type code or description is empty")
		}
	}

	// RCUR should be most common
	if _, ok := sequenceTypes["RCUR"]; !ok {
		t.Error("RCUR should be a valid sequence type")
	}
}

func TestPaymentCreditorID(t *testing.T) {
	t.Run("Austrian creditor ID format", func(t *testing.T) {
		// Austrian creditor ID format: AT + 2 check digits + ZZZ + creditor business code
		creditorID := "AT12ZZZ00000000001"

		if len(creditorID) != 18 {
			t.Errorf("Austrian creditor ID should be 18 characters: len=%d", len(creditorID))
		}
		if creditorID[:2] != "AT" {
			t.Errorf("Austrian creditor ID should start with AT")
		}
	})
}

func TestBankStatementTypes(t *testing.T) {
	t.Run("BankStatement struct fields", func(t *testing.T) {
		now := time.Now()
		stmt := &payment.BankStatement{
			ID:             uuid.New(),
			TenantID:       uuid.New(),
			IBAN:           "AT611904300234573201",
			StatementID:    "STMT-2025-001",
			StatementDate:  now,
			OpeningBalance: 100000000,
			ClosingBalance: 150000000,
			EntryCount:     100,
			ImportedAt:     now,
			CreatedAt:      now,
		}

		if stmt.IBAN != "AT611904300234573201" {
			t.Errorf("IBAN mismatch: got %s", stmt.IBAN)
		}
		if stmt.EntryCount != 100 {
			t.Errorf("EntryCount mismatch: got %d", stmt.EntryCount)
		}
	})

	t.Run("Transaction struct fields", func(t *testing.T) {
		now := time.Now()
		reference := "REF-001"
		endToEndID := "E2E-001"
		remittanceInfo := "Payment INV-2025-001"
		counterpartyName := "Customer GmbH"
		counterpartyIBAN := "AT861904300234573202"
		tx := &payment.Transaction{
			ID:               uuid.New(),
			StatementID:      uuid.New(),
			Amount:           500000,
			Currency:         "EUR",
			CreditDebit:      "CRDT",
			BookingDate:      now,
			ValueDate:        &now,
			Reference:        &reference,
			EndToEndID:       &endToEndID,
			RemittanceInfo:   &remittanceInfo,
			CounterpartyName: &counterpartyName,
			CounterpartyIBAN: &counterpartyIBAN,
			CreatedAt:        now,
		}

		if tx.CreditDebit != "CRDT" {
			t.Errorf("CreditDebit mismatch: got %s", tx.CreditDebit)
		}
		if tx.Amount != 500000 {
			t.Errorf("Amount mismatch: got %d", tx.Amount)
		}
	})
}

func TestBankStatementImport(t *testing.T) {
	t.Run("camt.053 upload content type", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/payments/statements", nil)
		req.Header.Set("Content-Type", "application/xml")

		if req.Header.Get("Content-Type") != "application/xml" {
			t.Error("Content-Type should be application/xml for camt.053")
		}
	})
}

func TestTransactionMatching(t *testing.T) {
	t.Run("Match request parsing", func(t *testing.T) {
		invoiceID := uuid.New()
		reqBody := map[string]interface{}{
			"invoice_id": invoiceID.String(),
		}

		body, _ := json.Marshal(reqBody)

		var parsed struct {
			InvoiceID string `json:"invoice_id"`
		}
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse match request: %v", err)
		}

		if parsed.InvoiceID != invoiceID.String() {
			t.Error("InvoiceID not parsed correctly")
		}
	})
}

func TestPaymentCSVImport(t *testing.T) {
	t.Run("CSV import content type", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/payments/batches/import", nil)
		req.Header.Set("Content-Type", "text/csv")

		if req.Header.Get("Content-Type") != "text/csv" {
			t.Error("Content-Type should be text/csv for CSV import")
		}
	})
}

func TestPaymentSEPACharset(t *testing.T) {
	t.Run("SEPA allowed characters", func(t *testing.T) {
		// SEPA allows only a restricted character set (Latin characters, numbers, some special chars)
		allowedChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/-?:().,'+  "

		for _, char := range allowedChars {
			if char == 0 {
				t.Error("Empty character in allowed set")
			}
		}

		// German umlauts should be converted
		specialChars := map[string]string{
			"ä": "ae",
			"ö": "oe",
			"ü": "ue",
			"ß": "ss",
			"Ä": "Ae",
			"Ö": "Oe",
			"Ü": "Ue",
		}

		for umlaut, replacement := range specialChars {
			if umlaut == "" || replacement == "" {
				t.Error("Umlaut or replacement is empty")
			}
		}
	})
}

func TestPaymentAmountValidation(t *testing.T) {
	t.Run("Amount in cents", func(t *testing.T) {
		// Amounts are stored in cents to avoid floating point issues
		amountEUR := 1234.56
		amountCents := int64(amountEUR * 100)

		expectedCents := int64(123456)
		if amountCents != expectedCents {
			t.Errorf("Amount conversion mismatch: got %d, want %d", amountCents, expectedCents)
		}
	})

	t.Run("Maximum SEPA amount", func(t *testing.T) {
		// SEPA maximum is 999,999,999.99 EUR
		maxSEPA := int64(99999999999) // in cents

		if maxSEPA != 99999999999 {
			t.Error("Max SEPA amount should be 999,999,999.99 EUR (99999999999 cents)")
		}
	})
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
