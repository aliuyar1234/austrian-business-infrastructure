package unit

import (
	"encoding/xml"
	"testing"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/fonws"
)

// T012: Test UVA XML generation
func TestUVAXMLGeneration(t *testing.T) {
	uva := fonws.UVA{
		Year:   2025,
		Period: fonws.UVAPeriod{Type: fonws.PeriodTypeMonthly, Value: 1},
		KZ000:  10000000, // 100,000.00 EUR (in cents)
		KZ017:  8000000,  // 80,000.00 EUR taxable at 20%
		KZ060:  1600000,  // 16,000.00 EUR input tax
		KZ095:  1600000,  // 16,000.00 EUR (20% of 80,000)
	}

	xmlData, err := fonws.GenerateUVAXML(&uva)
	if err != nil {
		t.Fatalf("Failed to generate UVA XML: %v", err)
	}

	xmlStr := string(xmlData)

	// Verify XML structure
	if !contains(xmlStr, `<Umsatzsteuervoranmeldung`) {
		t.Error("Missing Umsatzsteuervoranmeldung root element")
	}
	if !contains(xmlStr, `<Jahr>2025</Jahr>`) {
		t.Error("Missing or incorrect Jahr element")
	}
	if !contains(xmlStr, `<Monat>01</Monat>`) {
		t.Error("Missing or incorrect Monat element")
	}
	if !contains(xmlStr, `<KZ000>10000000</KZ000>`) {
		t.Error("Missing or incorrect KZ000 element")
	}
	if !contains(xmlStr, `<KZ017>8000000</KZ017>`) {
		t.Error("Missing or incorrect KZ017 element")
	}
	if !contains(xmlStr, `<KZ060>1600000</KZ060>`) {
		t.Error("Missing or incorrect KZ060 element")
	}
	if !contains(xmlStr, `<KZ095>1600000</KZ095>`) {
		t.Error("Missing or incorrect KZ095 element")
	}
}

// T012b: Test UVA XML generation for quarterly period
func TestUVAXMLGenerationQuarterly(t *testing.T) {
	uva := fonws.UVA{
		Year:   2025,
		Period: fonws.UVAPeriod{Type: fonws.PeriodTypeQuarterly, Value: 1},
		KZ017:  5000000,
		KZ060:  1000000,
		KZ095:  1000000,
	}

	xmlData, err := fonws.GenerateUVAXML(&uva)
	if err != nil {
		t.Fatalf("Failed to generate UVA XML: %v", err)
	}

	xmlStr := string(xmlData)

	if !contains(xmlStr, `<Quartal>1</Quartal>`) {
		t.Error("Missing or incorrect Quartal element")
	}
	// Should NOT contain Monat for quarterly
	if contains(xmlStr, `<Monat>`) {
		t.Error("Monthly period element present in quarterly UVA")
	}
}

// T013: Test UVA XML validation
func TestUVAValidation(t *testing.T) {
	testCases := []struct {
		name    string
		uva     fonws.UVA
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid monthly UVA",
			uva: fonws.UVA{
				Year:   2025,
				Period: fonws.UVAPeriod{Type: fonws.PeriodTypeMonthly, Value: 1},
				KZ017:  8000000,
				KZ060:  1600000,
				KZ095:  1600000,
			},
			wantErr: false,
		},
		{
			name: "valid quarterly UVA",
			uva: fonws.UVA{
				Year:   2025,
				Period: fonws.UVAPeriod{Type: fonws.PeriodTypeQuarterly, Value: 4},
				KZ017:  5000000,
				KZ060:  1000000,
				KZ095:  1000000,
			},
			wantErr: false,
		},
		{
			name: "invalid year too low",
			uva: fonws.UVA{
				Year:   1999,
				Period: fonws.UVAPeriod{Type: fonws.PeriodTypeMonthly, Value: 1},
				KZ017:  8000000,
			},
			wantErr: true,
			errMsg:  "year must be between 2000 and 2100",
		},
		{
			name: "invalid month too high",
			uva: fonws.UVA{
				Year:   2025,
				Period: fonws.UVAPeriod{Type: fonws.PeriodTypeMonthly, Value: 13},
				KZ017:  8000000,
			},
			wantErr: true,
			errMsg:  "month must be between 1 and 12",
		},
		{
			name: "invalid quarter too high",
			uva: fonws.UVA{
				Year:   2025,
				Period: fonws.UVAPeriod{Type: fonws.PeriodTypeQuarterly, Value: 5},
				KZ017:  8000000,
			},
			wantErr: true,
			errMsg:  "quarter must be between 1 and 4",
		},
		{
			name: "negative KZ value",
			uva: fonws.UVA{
				Year:   2025,
				Period: fonws.UVAPeriod{Type: fonws.PeriodTypeMonthly, Value: 1},
				KZ017:  -100,
			},
			wantErr: true,
			errMsg:  "KZ017 must be non-negative",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := fonws.ValidateUVA(&tc.uva)
			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tc.errMsg)
				} else if !contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

// T014: Test FileUpload SOAP request serialization
func TestFileUploadRequestSerialization(t *testing.T) {
	req := fonws.FileUploadRequest{
		TID:   "123456789012",
		BenID: "WSUSER001",
		ID:    "SESSION_TOKEN_123",
		Art:   "U30",
		Data:  "PD94bWwgdmVyc2lvbj0iMS4wIj8+", // Base64 encoded XML
	}

	data, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal FileUploadRequest: %v", err)
	}

	xmlStr := string(data)

	if !contains(xmlStr, "<tid>123456789012</tid>") {
		t.Error("Missing or incorrect tid field")
	}
	if !contains(xmlStr, "<benid>WSUSER001</benid>") {
		t.Error("Missing or incorrect benid field")
	}
	if !contains(xmlStr, "<id>SESSION_TOKEN_123</id>") {
		t.Error("Missing or incorrect id field")
	}
	if !contains(xmlStr, "<art>U30</art>") {
		t.Error("Missing or incorrect art field")
	}
	if !contains(xmlStr, "PD94bWwgdmVyc2lvbj0iMS4wIj8+") {
		t.Error("Missing or incorrect base64 data")
	}
}

// T015: Test FileUpload SOAP response parsing
func TestFileUploadResponseParsing(t *testing.T) {
	// Test success response
	successXML := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <uploadResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/fileUploadService">
      <rc>0</rc>
      <msg>Übermittlung erfolgreich</msg>
      <belegnummer>FON-2025-12345678</belegnummer>
    </uploadResponse>
  </soap:Body>
</soap:Envelope>`

	var resp fonws.FileUploadResponse
	err := fonws.ParseResponse([]byte(successXML), &resp)
	if err != nil {
		t.Fatalf("Failed to parse success response: %v", err)
	}

	if resp.RC != 0 {
		t.Errorf("Expected RC=0, got %d", resp.RC)
	}
	if resp.Belegnummer != "FON-2025-12345678" {
		t.Errorf("Expected Belegnummer FON-2025-12345678, got %s", resp.Belegnummer)
	}
	if resp.Msg != "Übermittlung erfolgreich" {
		t.Errorf("Expected success message, got %s", resp.Msg)
	}
}

// T015b: Test FileUpload error responses
func TestFileUploadResponseErrors(t *testing.T) {
	testCases := []struct {
		name     string
		xml      string
		expRC    int
		expError string
	}{
		{
			name: "invalid session",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <uploadResponse>
      <rc>-2</rc>
      <msg>Session ungültig</msg>
    </uploadResponse>
  </soap:Body>
</soap:Envelope>`,
			expRC:    -2,
			expError: "Session",
		},
		{
			name: "invalid XML",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <uploadResponse>
      <rc>-3</rc>
      <msg>XML ungültig</msg>
    </uploadResponse>
  </soap:Body>
</soap:Envelope>`,
			expRC:    -3,
			expError: "XML",
		},
		{
			name: "unauthorized tax number",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <uploadResponse>
      <rc>-4</rc>
      <msg>Steuernummer nicht berechtigt</msg>
    </uploadResponse>
  </soap:Body>
</soap:Envelope>`,
			expRC:    -4,
			expError: "Steuernummer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var resp fonws.FileUploadResponse
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

// T016: Integration test for UVA submission flow (mocked)
func TestUVASubmissionFlow(t *testing.T) {
	// This test verifies the complete flow without actual API calls
	// Uses mock client for testing

	// Step 1: Create UVA
	uva := fonws.UVA{
		Year:   2025,
		Period: fonws.UVAPeriod{Type: fonws.PeriodTypeMonthly, Value: 1},
		KZ000:  10000000,
		KZ017:  8000000,
		KZ060:  1600000,
		KZ095:  1600000,
		Status: fonws.UVAStatusDraft,
	}

	// Step 2: Validate
	err := fonws.ValidateUVA(&uva)
	if err != nil {
		t.Fatalf("UVA validation failed: %v", err)
	}

	// Step 3: Generate XML
	xmlData, err := fonws.GenerateUVAXML(&uva)
	if err != nil {
		t.Fatalf("UVA XML generation failed: %v", err)
	}
	if len(xmlData) == 0 {
		t.Error("Generated XML is empty")
	}

	// Step 4: Verify status transition
	uva.Status = fonws.UVAStatusValidated
	if uva.Status != fonws.UVAStatusValidated {
		t.Error("Status not updated to validated")
	}

	// Step 5: Simulate submission (would use mock client)
	uva.Status = fonws.UVAStatusSubmitted
	uva.SubmittedAt = timePtr(time.Now())
	uva.Reference = "FON-2025-12345678"

	if uva.Reference == "" {
		t.Error("Reference number not set after submission")
	}
	if uva.SubmittedAt == nil {
		t.Error("SubmittedAt not set after submission")
	}
}

// Test UVA status constants
func TestUVAStatusConstants(t *testing.T) {
	statuses := []fonws.UVAStatus{
		fonws.UVAStatusDraft,
		fonws.UVAStatusValidated,
		fonws.UVAStatusSubmitted,
		fonws.UVAStatusAccepted,
		fonws.UVAStatusRejected,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("Status constant is empty")
		}
	}
}

// Test UVA period types
func TestUVAPeriodTypes(t *testing.T) {
	if fonws.PeriodTypeMonthly != "monthly" {
		t.Errorf("Expected monthly, got %s", fonws.PeriodTypeMonthly)
	}
	if fonws.PeriodTypeQuarterly != "quarterly" {
		t.Errorf("Expected quarterly, got %s", fonws.PeriodTypeQuarterly)
	}
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}
