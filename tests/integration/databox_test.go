package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/austrian-business-infrastructure/fo/internal/fonws"
)

// T053: Integration test for databox list with mocked SOAP
func TestDataboxListFlowMocked(t *testing.T) {
	// Create mock SOAP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetDataboxInfoResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/databoxService">
      <rc>0</rc>
      <msg></msg>
      <result>
        <databox>
          <applkey>APP001</applkey>
          <filebez>Bescheid_ESt_2024</filebez>
          <ts_zust>2025-12-01T10:30:00</ts_zust>
          <erlession>B</erlession>
          <veression>N</veression>
        </databox>
        <databox>
          <applkey>APP002</applkey>
          <filebez>Ergaenzungsersuchen_2025</filebez>
          <ts_zust>2025-12-05T14:15:00</ts_zust>
          <erlession>E</erlession>
          <veression>N</veression>
        </databox>
        <databox>
          <applkey>APP003</applkey>
          <filebez>Vorhalt_2025</filebez>
          <ts_zust>2025-12-06T09:00:00</ts_zust>
          <erlession>V</erlession>
          <veression>N</veression>
        </databox>
      </result>
    </GetDataboxInfoResponse>
  </soap:Body>
</soap:Envelope>`))
	}))
	defer server.Close()

	// Create client and make GetDataboxInfo request
	client := fonws.NewClient()

	req := fonws.GetDataboxInfoRequest{
		Xmlns: fonws.DataboxNS,
		ID:    "MOCK_SESSION",
		TID:   "123456789012",
		BenID: "TESTUSER",
	}

	var resp fonws.GetDataboxInfoResponse
	err := client.Call(server.URL, req, &resp)
	if err != nil {
		t.Fatalf("GetDataboxInfo call failed: %v", err)
	}

	// Verify response
	if resp.RC != 0 {
		t.Errorf("Expected RC=0, got %d", resp.RC)
	}
	if len(resp.Result.Entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(resp.Result.Entries))
	}

	// Verify entries
	if resp.Result.Entries[0].Applkey != "APP001" {
		t.Errorf("Wrong applkey for entry 0")
	}

	// Verify ActionRequired helper
	if resp.Result.Entries[0].ActionRequired() {
		t.Error("Entry 0 (Bescheid) should not require action")
	}
	if !resp.Result.Entries[1].ActionRequired() {
		t.Error("Entry 1 (Ergänzungsersuchen) should require action")
	}
	if !resp.Result.Entries[2].ActionRequired() {
		t.Error("Entry 2 (Vorhalt) should require action")
	}
}

func TestDataboxDownloadMocked(t *testing.T) {
	// Create mock SOAP server for download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(http.StatusOK)
		// SGVsbG8gV29ybGQ= is base64 for "Hello World"
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetDataboxResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/databoxService">
      <rc>0</rc>
      <msg></msg>
      <result>
        <filename>Test_Document.pdf</filename>
        <content>SGVsbG8gV29ybGQ=</content>
      </result>
    </GetDataboxResponse>
  </soap:Body>
</soap:Envelope>`))
	}))
	defer server.Close()

	client := fonws.NewClient()

	req := fonws.GetDataboxRequest{
		Xmlns:   fonws.DataboxNS,
		ID:      "MOCK_SESSION",
		TID:     "123456789012",
		BenID:   "TESTUSER",
		Applkey: "APP001",
	}

	var resp fonws.GetDataboxResponse
	err := client.Call(server.URL, req, &resp)
	if err != nil {
		t.Fatalf("GetDatabox call failed: %v", err)
	}

	if resp.RC != 0 {
		t.Errorf("Expected RC=0, got %d", resp.RC)
	}
	if resp.Result.Filename != "Test_Document.pdf" {
		t.Errorf("Expected filename Test_Document.pdf, got %s", resp.Result.Filename)
	}
	if resp.Result.Content != "SGVsbG8gV29ybGQ=" {
		t.Errorf("Wrong content returned")
	}
}

// T068: Integration test for batch operation with multiple mocked accounts
func TestBatchDataboxOperation(t *testing.T) {
	// Create mock SOAP server that tracks request count
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(http.StatusOK)
		// Return different results based on request count to simulate different accounts
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetDataboxInfoResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/databoxService">
      <rc>0</rc>
      <msg></msg>
      <result>
        <databox>
          <applkey>APP001</applkey>
          <filebez>Test</filebez>
          <ts_zust>2025-12-01T10:30:00</ts_zust>
          <erlession>B</erlession>
          <veression>N</veression>
        </databox>
      </result>
    </GetDataboxInfoResponse>
  </soap:Body>
</soap:Envelope>`))
	}))
	defer server.Close()

	// Simulate processing multiple accounts
	accounts := []struct {
		name  string
		tid   string
		benid string
	}{
		{"Account1", "111111111111", "USER1"},
		{"Account2", "222222222222", "USER2"},
		{"Account3", "333333333333", "USER3"},
	}

	client := fonws.NewClient()
	results := make([]struct {
		name    string
		entries int
		err     error
	}, len(accounts))

	for i, acc := range accounts {
		req := fonws.GetDataboxInfoRequest{
			Xmlns: fonws.DataboxNS,
			ID:    "MOCK_SESSION",
			TID:   acc.tid,
			BenID: acc.benid,
		}

		var resp fonws.GetDataboxInfoResponse
		err := client.Call(server.URL, req, &resp)
		results[i].name = acc.name
		if err != nil {
			results[i].err = err
		} else {
			results[i].entries = len(resp.Result.Entries)
		}
	}

	// Verify all accounts were processed
	for i, r := range results {
		if r.err != nil {
			t.Errorf("Account %d failed: %v", i, r.err)
		}
		if r.entries != 1 {
			t.Errorf("Account %d: expected 1 entry, got %d", i, r.entries)
		}
	}

	// Verify server received requests for all accounts
	if requestCount != 3 {
		t.Errorf("Expected 3 requests, got %d", requestCount)
	}
}

func TestDataboxSessionExpired(t *testing.T) {
	// Create mock SOAP server returning session expired error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetDataboxInfoResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/databoxService">
      <rc>-1</rc>
      <msg>Session abgelaufen</msg>
      <result></result>
    </GetDataboxInfoResponse>
  </soap:Body>
</soap:Envelope>`))
	}))
	defer server.Close()

	client := fonws.NewClient()

	req := fonws.GetDataboxInfoRequest{
		Xmlns: fonws.DataboxNS,
		ID:    "EXPIRED_SESSION",
		TID:   "123456789012",
		BenID: "TESTUSER",
	}

	var resp fonws.GetDataboxInfoResponse
	err := client.Call(server.URL, req, &resp)
	if err != nil {
		t.Fatalf("Call failed unexpectedly: %v", err)
	}

	// Verify error code
	if resp.RC != -1 {
		t.Errorf("Expected RC=-1, got %d", resp.RC)
	}

	// Check using the error function
	foErr := fonws.CheckResponse(resp.RC, resp.Msg)
	if foErr == nil {
		t.Error("Expected error for session expired")
	}
	if !fonws.IsSessionExpired(foErr) {
		t.Error("Expected IsSessionExpired to return true")
	}
}
