package unit

import (
	"testing"
	"time"

	"austrian-business-infrastructure/internal/sepa"
)

// T105: Test pain.001 XML generation
func TestPain001XMLGeneration(t *testing.T) {
	transfer := createTestCreditTransfer()

	xmlData, err := sepa.GeneratePain001(transfer)
	if err != nil {
		t.Fatalf("Failed to generate pain.001: %v", err)
	}

	xmlStr := string(xmlData)

	// Verify XML structure
	if !contains(xmlStr, "<Document") {
		t.Error("Missing Document root element")
	}
	if !contains(xmlStr, "urn:iso:std:iso:20022:tech:xsd:pain.001") {
		t.Error("Missing pain.001 namespace")
	}
	if !contains(xmlStr, "<MsgId>CT-2025-001</MsgId>") {
		t.Error("Missing or incorrect MsgId")
	}
	if !contains(xmlStr, "<NbOfTxs>2</NbOfTxs>") {
		t.Error("Missing or incorrect NbOfTxs")
	}
	if !contains(xmlStr, "AT611904300234573201") {
		t.Error("Missing debtor IBAN")
	}
}

// T106: Test camt.053 XML parsing
func TestCamt053XMLParsing(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <Stmt>
      <Id>STMT-2025-001</Id>
      <CreDtTm>2025-01-15T10:00:00</CreDtTm>
      <Acct>
        <Id>
          <IBAN>AT611904300234573201</IBAN>
        </Id>
        <Ccy>EUR</Ccy>
      </Acct>
      <Bal>
        <Tp>
          <CdOrPrtry>
            <Cd>OPBD</Cd>
          </CdOrPrtry>
        </Tp>
        <Amt Ccy="EUR">10000.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <Dt>
          <Dt>2025-01-15</Dt>
        </Dt>
      </Bal>
      <Ntry>
        <Amt Ccy="EUR">1500.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <BookgDt>
          <Dt>2025-01-14</Dt>
        </BookgDt>
        <NtryDtls>
          <TxDtls>
            <Refs>
              <EndToEndId>PAY-001</EndToEndId>
            </Refs>
            <RmtInf>
              <Ustrd>Invoice payment INV-2025-001</Ustrd>
            </RmtInf>
          </TxDtls>
        </NtryDtls>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`

	stmt, err := sepa.ParseCamt053([]byte(xmlData))
	if err != nil {
		t.Fatalf("Failed to parse camt.053: %v", err)
	}

	if stmt.ID != "STMT-2025-001" {
		t.Errorf("Expected ID STMT-2025-001, got %s", stmt.ID)
	}
	if stmt.Account.IBAN != "AT611904300234573201" {
		t.Errorf("Expected IBAN AT611904300234573201, got %s", stmt.Account.IBAN)
	}
	if len(stmt.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(stmt.Entries))
	}
	if stmt.Entries[0].Amount != 150000 { // 1500.00 in cents
		t.Errorf("Expected amount 150000, got %d", stmt.Entries[0].Amount)
	}
}

// T107: Test IBAN validation
func TestIBANValidation(t *testing.T) {
	testCases := []struct {
		iban  string
		valid bool
	}{
		{"AT611904300234573201", true},   // Valid Austrian IBAN
		{"DE89370400440532013000", true}, // Valid German IBAN
		{"FR7630006000011234567890189", true}, // Valid French IBAN
		{"AT611904300234573200", false},  // Invalid check digit
		{"AT61190430023457320", false},   // Too short
		{"AT6119043002345732011", false}, // Too long
		{"XX611904300234573201", false},  // Invalid country code
		{"", false},                       // Empty
		{"NOTANIBAN", false},              // Invalid format
	}

	for _, tc := range testCases {
		t.Run(tc.iban, func(t *testing.T) {
			err := sepa.ValidateIBAN(tc.iban)
			if tc.valid && err != nil {
				t.Errorf("Expected valid IBAN, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("Expected error for invalid IBAN, got nil")
			}
		})
	}
}

// T108: Test BIC lookup
func TestBICLookup(t *testing.T) {
	// Test known Austrian bank
	bic, name := sepa.LookupAustrianBank("19043")
	if bic == "" {
		t.Fatal("Expected to find bank for code 19043")
	}
	if bic != "BKAUATWW" {
		t.Errorf("Expected BIC BKAUATWW, got %s", bic)
	}
	if name != "Bank Austria (Landesdirektion)" {
		t.Errorf("Expected name Bank Austria (Landesdirektion), got %s", name)
	}

	// Test unknown bank code
	bic, _ = sepa.LookupAustrianBank("99999")
	if bic != "" {
		t.Error("Expected empty string for unknown bank code")
	}

	// Test BIC lookup from IBAN
	bic, _ = sepa.LookupBICByIBAN("AT611904300234573201")
	if bic != "BKAUATWW" {
		t.Errorf("Expected BIC BKAUATWW, got %s", bic)
	}
}

// Test SEPA credit transfer structures
func TestSEPACreditTransfer(t *testing.T) {
	transfer := &sepa.SEPACreditTransfer{
		MessageID:      "CT-2025-001",
		CreationTime:   time.Now(),
		NumberOfTxs:    1,
		ControlSum:     100000,
		InitiatingParty: sepa.SEPAParty{
			Name: "Acme GmbH",
		},
		Debtor: sepa.SEPAParty{
			Name: "Acme GmbH",
		},
		DebtorAccount: sepa.SEPAAccount{
			IBAN: "AT611904300234573201",
			BIC:  "BKAUATWW",
		},
		Transactions: []*sepa.SEPACreditTransaction{
			{
				InstructionID: "TXN-001",
				EndToEndID:    "E2E-001",
				Amount:        100000,
				Currency:      "EUR",
				Creditor: sepa.SEPAParty{
					Name: "Supplier AG",
				},
				CreditorAccount: sepa.SEPAAccount{
					IBAN: "DE89370400440532013000",
				},
				RemittanceInfo: "Invoice payment",
			},
		},
	}

	if transfer.MessageID != "CT-2025-001" {
		t.Errorf("Expected MessageID CT-2025-001, got %s", transfer.MessageID)
	}
	if len(transfer.Transactions) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(transfer.Transactions))
	}
}

// Test SEPA address
func TestSEPAAddress(t *testing.T) {
	addr := &sepa.SEPAAddress{
		StreetName:  "Hauptstraße 1",
		PostCode:    "1010",
		TownName:    "Wien",
		Country:     "AT",
	}

	if addr.StreetName != "Hauptstraße 1" {
		t.Errorf("Expected StreetName Hauptstraße 1, got %s", addr.StreetName)
	}
	if addr.Country != "AT" {
		t.Errorf("Expected Country AT, got %s", addr.Country)
	}
}

// Test SEPA statement entry
func TestSEPAStatementEntry(t *testing.T) {
	entry := &sepa.SEPAStatementEntry{
		Amount:        150000,
		Currency:      "EUR",
		CreditDebit:   sepa.CreditDebitCredit,
		BookingDate:   time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
		EndToEndID:    "PAY-001",
		RemittanceInfo: "Invoice payment",
	}

	if entry.Amount != 150000 {
		t.Errorf("Expected Amount 150000, got %d", entry.Amount)
	}
	if entry.CreditDebit != sepa.CreditDebitCredit {
		t.Errorf("Expected CreditDebit CRDT, got %s", entry.CreditDebit)
	}
	if entry.AmountEUR() != 1500.0 {
		t.Errorf("Expected AmountEUR 1500.0, got %.2f", entry.AmountEUR())
	}
}

// Test NewCreditTransfer
func TestNewCreditTransfer(t *testing.T) {
	ct := sepa.NewCreditTransfer("CT-TEST-001", "Initiator GmbH")

	if ct.MessageID != "CT-TEST-001" {
		t.Errorf("Expected MessageID CT-TEST-001, got %s", ct.MessageID)
	}
	if ct.InitiatingParty.Name != "Initiator GmbH" {
		t.Errorf("Expected InitiatingParty Initiator GmbH, got %s", ct.InitiatingParty.Name)
	}
	if ct.CreationTime.IsZero() {
		t.Error("CreationTime should be set")
	}
}

// Test pain.008 direct debit generation
func TestPain008XMLGeneration(t *testing.T) {
	debit := createTestDirectDebit()

	xmlData, err := sepa.GeneratePain008(debit)
	if err != nil {
		t.Fatalf("Failed to generate pain.008: %v", err)
	}

	xmlStr := string(xmlData)

	// Verify XML structure
	if !contains(xmlStr, "<Document") {
		t.Error("Missing Document root element")
	}
	if !contains(xmlStr, "urn:iso:std:iso:20022:tech:xsd:pain.008") {
		t.Error("Missing pain.008 namespace")
	}
	if !contains(xmlStr, "<MsgId>DD-2025-001</MsgId>") {
		t.Error("Missing or incorrect MsgId")
	}
}

// Test ParseCSV for credit transfers
func TestParseCSVForCreditTransfer(t *testing.T) {
	csvData := `creditor_name,creditor_iban,amount,currency,reference
Supplier AG,DE89370400440532013000,1500.00,EUR,INV-001
Vendor GmbH,AT611904300234573201,750.50,EUR,INV-002`

	txns, err := sepa.ParseCreditTransferCSV([]byte(csvData))
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(txns) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(txns))
	}

	if txns[0].Creditor.Name != "Supplier AG" {
		t.Errorf("Expected creditor name Supplier AG, got %s", txns[0].Creditor.Name)
	}
	if txns[0].CreditorAccount.IBAN != "DE89370400440532013000" {
		t.Errorf("Expected IBAN DE89370400440532013000, got %s", txns[0].CreditorAccount.IBAN)
	}
	if txns[0].Amount != 150000 { // 1500.00 in cents
		t.Errorf("Expected amount 150000, got %d", txns[0].Amount)
	}
}

// Test IBAN validation with details
func TestIBANValidationWithDetails(t *testing.T) {
	result := sepa.ValidateIBANWithDetails("AT611904300234573201")

	if !result.Valid {
		t.Error("Expected valid IBAN")
	}
	if result.CountryCode != "AT" {
		t.Errorf("Expected country AT, got %s", result.CountryCode)
	}
	if result.BankCode != "19043" {
		t.Errorf("Expected bank code 19043, got %s", result.BankCode)
	}
	if result.BIC != "BKAUATWW" {
		t.Errorf("Expected BIC BKAUATWW, got %s", result.BIC)
	}
}

// Test BIC validation
func TestBICValidation(t *testing.T) {
	testCases := []struct {
		bic   string
		valid bool
	}{
		{"BKAUATWW", true},     // Valid 8-char BIC
		{"BKAUATWWXXX", true},  // Valid 11-char BIC
		{"COBADEFF", true},     // Another valid BIC
		{"INVALID", false},     // Too short
		{"BKAUATWW1234", false}, // Too long
		{"", false},             // Empty
	}

	for _, tc := range testCases {
		t.Run(tc.bic, func(t *testing.T) {
			err := sepa.ValidateBIC(tc.bic)
			if tc.valid && err != nil {
				t.Errorf("Expected valid BIC, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("Expected error for invalid BIC, got nil")
			}
		})
	}
}

// Helper function to create a test credit transfer
func createTestCreditTransfer() *sepa.SEPACreditTransfer {
	return &sepa.SEPACreditTransfer{
		MessageID:    "CT-2025-001",
		CreationTime: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		NumberOfTxs:  2,
		ControlSum:   225000, // 2250.00 EUR
		InitiatingParty: sepa.SEPAParty{
			Name: "Acme GmbH",
		},
		Debtor: sepa.SEPAParty{
			Name: "Acme GmbH",
			Address: &sepa.SEPAAddress{
				StreetName: "Hauptstraße 1",
				PostCode:   "1010",
				TownName:   "Wien",
				Country:    "AT",
			},
		},
		DebtorAccount: sepa.SEPAAccount{
			IBAN: "AT611904300234573201",
			BIC:  "BKAUATWW",
		},
		Transactions: []*sepa.SEPACreditTransaction{
			{
				InstructionID:     "TXN-001",
				EndToEndID:        "E2E-001",
				Amount:            150000, // 1500.00 EUR
				Currency:          "EUR",
				Creditor:          sepa.SEPAParty{Name: "Supplier AG"},
				CreditorAccount:   sepa.SEPAAccount{IBAN: "DE89370400440532013000", BIC: "COBADEFF"},
				RemittanceInfo:    "Invoice INV-001",
			},
			{
				InstructionID:     "TXN-002",
				EndToEndID:        "E2E-002",
				Amount:            75000, // 750.00 EUR
				Currency:          "EUR",
				Creditor:          sepa.SEPAParty{Name: "Vendor GmbH"},
				CreditorAccount:   sepa.SEPAAccount{IBAN: "AT482011100000123456", BIC: "GIBAATWW"},
				RemittanceInfo:    "Invoice INV-002",
			},
		},
	}
}

// Helper function to create a test direct debit
func createTestDirectDebit() *sepa.SEPADirectDebit {
	return &sepa.SEPADirectDebit{
		MessageID:    "DD-2025-001",
		CreationTime: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		NumberOfTxs:  1,
		ControlSum:   50000,
		Creditor:     sepa.SEPAParty{Name: "Creditor GmbH"},
		CreditorAccount: sepa.SEPAAccount{
			IBAN: "AT611904300234573201",
			BIC:  "BKAUATWW",
		},
		CreditorID: "AT12ZZZ00000000001",
		Transactions: []*sepa.SEPADirectDebitTransaction{
			{
				InstructionID:   "DD-TXN-001",
				EndToEndID:      "DD-E2E-001",
				Amount:          50000,
				Currency:        "EUR",
				Debtor:          sepa.SEPAParty{Name: "Customer AG"},
				DebtorAccount:   sepa.SEPAAccount{IBAN: "DE89370400440532013000"},
				MandateID:       "MANDATE-001",
				MandateDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				SequenceType:    sepa.SequenceTypeRecurrent,
				RemittanceInfo:  "Subscription fee",
			},
		},
	}
}
