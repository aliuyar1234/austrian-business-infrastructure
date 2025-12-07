package unit

import (
	"encoding/xml"
	"testing"

	"github.com/austrian-business-infrastructure/fo/internal/fonws"
)

// T016: Test Login SOAP request serialization
func TestLoginRequestSerialization(t *testing.T) {
	req := fonws.LoginRequest{
		Xmlns:  fonws.SessionNS,
		TID:    "123456789012",
		BenID:  "WSUSER001",
		PIN:    "testpin",
		Herst:  "false",
	}

	data, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal LoginRequest: %v", err)
	}

	// Verify essential fields are present
	xmlStr := string(data)
	if !contains(xmlStr, "<tid>123456789012</tid>") {
		t.Error("Missing or incorrect tid field")
	}
	if !contains(xmlStr, "<benid>WSUSER001</benid>") {
		t.Error("Missing or incorrect benid field")
	}
	if !contains(xmlStr, "<pin>testpin</pin>") {
		t.Error("Missing or incorrect pin field")
	}
}

// T017: Test Login SOAP response parsing (success)
func TestLoginResponseParsingSuccess(t *testing.T) {
	responseXML := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <LoginResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/sessionService">
      <rc>0</rc>
      <msg></msg>
      <id>SESSION_TOKEN_12345</id>
    </LoginResponse>
  </soap:Body>
</soap:Envelope>`

	var resp fonws.LoginResponse
	err := fonws.ParseResponse([]byte(responseXML), &resp)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.RC != 0 {
		t.Errorf("Expected RC=0, got %d", resp.RC)
	}
	if resp.ID != "SESSION_TOKEN_12345" {
		t.Errorf("Expected ID=SESSION_TOKEN_12345, got %s", resp.ID)
	}
}

// T018: Test Login SOAP response parsing (all error codes)
func TestLoginResponseParsingErrors(t *testing.T) {
	testCases := []struct {
		code        int
		expectError bool
		errorMsg    string
	}{
		{0, false, ""},
		{-1, true, "Session expired"},
		{-2, true, "maintenance"},
		{-3, true, "Technical error"},
		{-4, true, "Invalid credentials"},
		{-5, true, "temporarily locked"},
		{-6, true, "permanently locked"},
		{-7, true, "WebService user"},
		{-8, true, "Participant locked"},
	}

	for _, tc := range testCases {
		t.Run(fonws.GetErrorMessage(tc.code), func(t *testing.T) {
			err := fonws.CheckResponse(tc.code, "")
			if tc.expectError && err == nil {
				t.Errorf("Expected error for code %d, got nil", tc.code)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for code %d, got %v", tc.code, err)
			}
			if tc.expectError && err != nil {
				if !contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.errorMsg, err.Error())
				}
			}
		})
	}
}

// T019: Test Logout SOAP request/response
func TestLogoutRequestResponse(t *testing.T) {
	req := fonws.LogoutRequest{
		Xmlns:  fonws.SessionNS,
		ID:     "SESSION_TOKEN_12345",
		TID:    "123456789012",
		BenID:  "WSUSER001",
	}

	data, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal LogoutRequest: %v", err)
	}

	xmlStr := string(data)
	if !contains(xmlStr, "<id>SESSION_TOKEN_12345</id>") {
		t.Error("Missing or incorrect id field")
	}

	// Test response parsing
	responseXML := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <LogoutResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/sessionService">
      <rc>0</rc>
      <msg></msg>
    </LogoutResponse>
  </soap:Body>
</soap:Envelope>`

	var resp fonws.LogoutResponse
	err = fonws.ParseResponse([]byte(responseXML), &resp)
	if err != nil {
		t.Fatalf("Failed to parse logout response: %v", err)
	}

	if resp.RC != 0 {
		t.Errorf("Expected RC=0, got %d", resp.RC)
	}
}

// T049: Test GetDataboxInfo SOAP request serialization
func TestGetDataboxInfoRequestSerialization(t *testing.T) {
	req := fonws.GetDataboxInfoRequest{
		Xmlns:     fonws.DataboxNS,
		ID:        "SESSION_TOKEN",
		TID:       "123456789012",
		BenID:     "WSUSER001",
		TsZustVon: "2025-01-01",
		TsZustBis: "2025-12-31",
	}

	data, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal GetDataboxInfoRequest: %v", err)
	}

	xmlStr := string(data)
	if !contains(xmlStr, "<id>SESSION_TOKEN</id>") {
		t.Error("Missing or incorrect id field")
	}
	if !contains(xmlStr, "<tid>123456789012</tid>") {
		t.Error("Missing or incorrect tid field")
	}
	if !contains(xmlStr, "<ts_zust_von>2025-01-01</ts_zust_von>") {
		t.Error("Missing or incorrect ts_zust_von field")
	}
}

// T050: Test GetDataboxInfo SOAP response parsing
func TestGetDataboxInfoResponseParsing(t *testing.T) {
	responseXML := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetDataboxInfoResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/databoxService">
      <rc>0</rc>
      <msg></msg>
      <result>
        <databox>
          <applkey>APP123456</applkey>
          <filebez>Bescheid_2025</filebez>
          <ts_zust>2025-12-01T10:30:00</ts_zust>
          <erlession>B</erlession>
          <veression>N</veression>
        </databox>
        <databox>
          <applkey>APP789012</applkey>
          <filebez>Ergaenzungsersuchen</filebez>
          <ts_zust>2025-12-05T14:15:00</ts_zust>
          <erlession>E</erlession>
          <veression>N</veression>
        </databox>
      </result>
    </GetDataboxInfoResponse>
  </soap:Body>
</soap:Envelope>`

	var resp fonws.GetDataboxInfoResponse
	err := fonws.ParseResponse([]byte(responseXML), &resp)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.RC != 0 {
		t.Errorf("Expected RC=0, got %d", resp.RC)
	}
	if len(resp.Result.Entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(resp.Result.Entries))
	}
	if resp.Result.Entries[0].Applkey != "APP123456" {
		t.Errorf("Expected applkey APP123456, got %s", resp.Result.Entries[0].Applkey)
	}
	if resp.Result.Entries[1].Erlession != "E" {
		t.Errorf("Expected erlession E, got %s", resp.Result.Entries[1].Erlession)
	}
}

// T051: Test DataboxEntry.ActionRequired() helper
func TestDataboxEntryActionRequired(t *testing.T) {
	testCases := []struct {
		erlession string
		expected  bool
	}{
		{"B", false}, // Bescheid - no action
		{"E", true},  // Ergänzungsersuchen - action required
		{"M", false}, // Mitteilung - no action
		{"V", true},  // Vorhalt - action required
	}

	for _, tc := range testCases {
		entry := fonws.DataboxEntry{Erlession: tc.erlession}
		if entry.ActionRequired() != tc.expected {
			t.Errorf("ActionRequired() for %s: expected %v, got %v", tc.erlession, tc.expected, entry.ActionRequired())
		}
	}
}

// T052: Test GetDatabox (download) request/response
func TestGetDataboxRequestResponse(t *testing.T) {
	req := fonws.GetDataboxRequest{
		Xmlns:   fonws.DataboxNS,
		ID:      "SESSION_TOKEN",
		TID:     "123456789012",
		BenID:   "WSUSER001",
		Applkey: "APP123456",
	}

	data, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal GetDataboxRequest: %v", err)
	}

	xmlStr := string(data)
	if !contains(xmlStr, "<applkey>APP123456</applkey>") {
		t.Error("Missing or incorrect applkey field")
	}

	// Test response parsing
	responseXML := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetDataboxResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/databoxService">
      <rc>0</rc>
      <msg></msg>
      <result>
        <filename>Bescheid_ESt_2024.pdf</filename>
        <content>SGVsbG8gV29ybGQ=</content>
      </result>
    </GetDataboxResponse>
  </soap:Body>
</soap:Envelope>`

	var resp fonws.GetDataboxResponse
	err = fonws.ParseResponse([]byte(responseXML), &resp)
	if err != nil {
		t.Fatalf("Failed to parse GetDatabox response: %v", err)
	}

	if resp.RC != 0 {
		t.Errorf("Expected RC=0, got %d", resp.RC)
	}
	if resp.Result.Filename != "Bescheid_ESt_2024.pdf" {
		t.Errorf("Expected filename Bescheid_ESt_2024.pdf, got %s", resp.Result.Filename)
	}
	if resp.Result.Content != "SGVsbG8gV29ybGQ=" {
		t.Errorf("Expected base64 content SGVsbG8gV29ybGQ=, got %s", resp.Result.Content)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
