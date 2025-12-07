package unit

import (
	"encoding/xml"
	"testing"

	"github.com/austrian-business-infrastructure/fo/internal/fonws"
)

// T030: Test UID validation request serialization
func TestUIDValidationRequestSerialization(t *testing.T) {
	req := fonws.UIDAbfrageRequest{
		TID:   "123456789012",
		BenID: "WSUSER001",
		ID:    "SESSION_TOKEN",
		UIDTN: "ATU12345678",
		Stufe: 1,
	}

	data, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal UIDAbfrageRequest: %v", err)
	}

	xmlStr := string(data)

	if !contains(xmlStr, "<tid>123456789012</tid>") {
		t.Error("Missing or incorrect tid field")
	}
	if !contains(xmlStr, "<benid>WSUSER001</benid>") {
		t.Error("Missing or incorrect benid field")
	}
	if !contains(xmlStr, "<uid_tn>ATU12345678</uid_tn>") {
		t.Error("Missing or incorrect uid_tn field")
	}
	if !contains(xmlStr, "<stufe>1</stufe>") {
		t.Error("Missing or incorrect stufe field")
	}
}

// T031: Test UID validation response parsing
func TestUIDValidationResponseParsing(t *testing.T) {
	// Test valid UID response
	validXML := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <uidAbfrageResponse>
      <rc>0</rc>
      <msg>OK</msg>
      <uid_tn>ATU12345678</uid_tn>
      <gueltig>true</gueltig>
      <name>Musterfirma GmbH</name>
      <adr_strasse>Musterstraße 1</adr_strasse>
      <adr_plz>1010</adr_plz>
      <adr_ort>Wien</adr_ort>
    </uidAbfrageResponse>
  </soap:Body>
</soap:Envelope>`

	var resp fonws.UIDAbfrageResponse
	err := fonws.ParseResponse([]byte(validXML), &resp)
	if err != nil {
		t.Fatalf("Failed to parse valid response: %v", err)
	}

	if resp.RC != 0 {
		t.Errorf("Expected RC=0, got %d", resp.RC)
	}
	if resp.Gueltig != "true" {
		t.Errorf("Expected gueltig=true, got %s", resp.Gueltig)
	}
	if resp.Name != "Musterfirma GmbH" {
		t.Errorf("Expected name 'Musterfirma GmbH', got '%s'", resp.Name)
	}
	if resp.AdrStrasse != "Musterstraße 1" {
		t.Errorf("Expected street 'Musterstraße 1', got '%s'", resp.AdrStrasse)
	}
	if resp.AdrPLZ != "1010" {
		t.Errorf("Expected PLZ '1010', got '%s'", resp.AdrPLZ)
	}
	if resp.AdrOrt != "Wien" {
		t.Errorf("Expected city 'Wien', got '%s'", resp.AdrOrt)
	}
}

// T031b: Test UID validation error responses
func TestUIDValidationResponseErrors(t *testing.T) {
	testCases := []struct {
		name     string
		xml      string
		expRC    int
		expError string
	}{
		{
			name: "daily limit exceeded",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <uidAbfrageResponse>
      <rc>1513</rc>
      <msg>Tageslimit für diese UID überschritten</msg>
      <uid_tn>ATU12345678</uid_tn>
    </uidAbfrageResponse>
  </soap:Body>
</soap:Envelope>`,
			expRC:    1513,
			expError: "Tageslimit",
		},
		{
			name: "invalid UID",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <uidAbfrageResponse>
      <rc>1514</rc>
      <msg>UID ungültig</msg>
      <uid_tn>ATU99999999</uid_tn>
    </uidAbfrageResponse>
  </soap:Body>
</soap:Envelope>`,
			expRC:    1514,
			expError: "ungültig",
		},
		{
			name: "no session",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <uidAbfrageResponse>
      <rc>-2</rc>
      <msg>Keine gültige Session</msg>
    </uidAbfrageResponse>
  </soap:Body>
</soap:Envelope>`,
			expRC:    -2,
			expError: "Session",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var resp fonws.UIDAbfrageResponse
			err := fonws.ParseResponse([]byte(tc.xml), &resp)
			if err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if resp.RC != tc.expRC {
				t.Errorf("Expected RC=%d, got %d", tc.expRC, resp.RC)
			}
			if !contains(resp.Msg, tc.expError) {
				t.Errorf("Expected message containing '%s', got '%s'", tc.expError, resp.Msg)
			}
		})
	}
}

// T032: Test UID format validation
func TestUIDFormatValidation(t *testing.T) {
	testCases := []struct {
		uid     string
		valid   bool
		country string
	}{
		// Austrian UIDs
		{"ATU12345678", true, "AT"},
		{"ATU00000001", true, "AT"},
		{"ATU99999999", true, "AT"},

		// German UIDs
		{"DE123456789", true, "DE"},
		{"DE000000001", true, "DE"},

		// Italian UIDs
		{"IT12345678901", true, "IT"},

		// Invalid formats
		{"ATU1234567", false, ""},   // Too short
		{"ATU123456789", false, ""},  // Too long
		{"AT12345678", false, ""},    // Missing U
		{"atu12345678", true, "AT"},  // Lowercase (normalized to uppercase)
		{"12345678", false, ""},      // No country code
		{"ATU1234567X", false, ""},   // Contains letter in number part
		{"XX123456789", false, ""},   // Invalid country code
		{"", false, ""},              // Empty
	}

	for _, tc := range testCases {
		t.Run(tc.uid, func(t *testing.T) {
			result := fonws.ValidateUIDFormat(tc.uid)
			if result.Valid != tc.valid {
				t.Errorf("ValidateUIDFormat(%s): expected valid=%v, got %v", tc.uid, tc.valid, result.Valid)
			}
			if tc.valid && result.CountryCode != tc.country {
				t.Errorf("ValidateUIDFormat(%s): expected country=%s, got %s", tc.uid, tc.country, result.CountryCode)
			}
		})
	}
}

// T033: Test CSV batch processing
func TestUIDCSVBatchProcessing(t *testing.T) {
	csvContent := `uid
ATU12345678
DE123456789
IT12345678901
ATU99999999`

	uids, err := fonws.ParseUIDCSV([]byte(csvContent))
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(uids) != 4 {
		t.Errorf("Expected 4 UIDs, got %d", len(uids))
	}

	expected := []string{"ATU12345678", "DE123456789", "IT12345678901", "ATU99999999"}
	for i, uid := range uids {
		if uid != expected[i] {
			t.Errorf("UID %d: expected %s, got %s", i, expected[i], uid)
		}
	}
}

// T033b: Test CSV with extra columns
func TestUIDCSVBatchProcessingWithExtraColumns(t *testing.T) {
	csvContent := `uid,company,notes
ATU12345678,Firma A,Test
DE123456789,Firma B,
IT12345678901,Firma C,Another test`

	uids, err := fonws.ParseUIDCSV([]byte(csvContent))
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(uids) != 3 {
		t.Errorf("Expected 3 UIDs, got %d", len(uids))
	}
}

// T034: Integration test for UID validation flow (mocked)
func TestUIDValidationFlow(t *testing.T) {
	// Test the complete validation flow without actual API calls

	// Step 1: Validate format
	uid := "ATU12345678"
	formatResult := fonws.ValidateUIDFormat(uid)
	if !formatResult.Valid {
		t.Fatalf("Format validation failed for %s", uid)
	}
	if formatResult.CountryCode != "AT" {
		t.Errorf("Expected country AT, got %s", formatResult.CountryCode)
	}

	// Step 2: Create validation request
	req := fonws.UIDAbfrageRequest{
		TID:   "123456789012",
		BenID: "WSUSER001",
		ID:    "SESSION_TOKEN",
		UIDTN: uid,
		Stufe: 1,
	}

	// Step 3: Verify request can be serialized
	_, err := xml.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to serialize request: %v", err)
	}

	// Step 4: Simulate parsing a response
	responseXML := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <uidAbfrageResponse>
      <rc>0</rc>
      <msg>OK</msg>
      <uid_tn>ATU12345678</uid_tn>
      <gueltig>true</gueltig>
      <name>Test Company</name>
      <adr_strasse>Test Street 1</adr_strasse>
      <adr_plz>1010</adr_plz>
      <adr_ort>Wien</adr_ort>
    </uidAbfrageResponse>
  </soap:Body>
</soap:Envelope>`

	var resp fonws.UIDAbfrageResponse
	err = fonws.ParseResponse([]byte(responseXML), &resp)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Step 5: Convert to result
	result := fonws.ConvertUIDResponse(&resp)
	if !result.Valid {
		t.Error("Expected valid result")
	}
	if result.UID != uid {
		t.Errorf("Expected UID %s, got %s", uid, result.UID)
	}
	if result.CompanyName != "Test Company" {
		t.Errorf("Expected company 'Test Company', got '%s'", result.CompanyName)
	}
}

// Test UID result helper methods
func TestUIDValidationResultMethods(t *testing.T) {
	result := &fonws.UIDValidationResult{
		UID:         "ATU12345678",
		Valid:       true,
		CompanyName: "Test GmbH",
		Address: fonws.UIDAddress{
			Street:   "Teststraße 1",
			PostCode: "1010",
			City:     "Wien",
			Country:  "AT",
		},
	}

	// Test formatted address
	expected := "Teststraße 1, 1010 Wien"
	if result.FormattedAddress() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result.FormattedAddress())
	}
}
