package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/austrian-business-infrastructure/fo/internal/fonws"
)

// T020: Integration test for session login flow with mocked SOAP
func TestSessionLoginFlowMocked(t *testing.T) {
	// Create mock SOAP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and content type
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "text/xml; charset=utf-8" {
			t.Errorf("Expected Content-Type text/xml, got %s", r.Header.Get("Content-Type"))
		}

		// Return mock success response
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <LoginResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/sessionService">
      <rc>0</rc>
      <msg></msg>
      <id>MOCK_SESSION_TOKEN</id>
    </LoginResponse>
  </soap:Body>
</soap:Envelope>`))
	}))
	defer server.Close()

	// Create client and make login request
	client := fonws.NewClient()

	req := fonws.LoginRequest{
		Xmlns:  fonws.SessionNS,
		TID:    "123456789012",
		BenID:  "TESTUSER",
		PIN:    "testpin",
		Herst:  "false",
	}

	var resp fonws.LoginResponse
	err := client.Call(server.URL, req, &resp)
	if err != nil {
		t.Fatalf("Login call failed: %v", err)
	}

	// Verify response
	if resp.RC != 0 {
		t.Errorf("Expected RC=0, got %d", resp.RC)
	}
	if resp.ID != "MOCK_SESSION_TOKEN" {
		t.Errorf("Expected MOCK_SESSION_TOKEN, got %s", resp.ID)
	}
}

func TestSessionLoginInvalidCredentials(t *testing.T) {
	// Create mock SOAP server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <LoginResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/sessionService">
      <rc>-4</rc>
      <msg>Ungültige Zugangsdaten</msg>
      <id></id>
    </LoginResponse>
  </soap:Body>
</soap:Envelope>`))
	}))
	defer server.Close()

	client := fonws.NewClient()

	req := fonws.LoginRequest{
		Xmlns:  fonws.SessionNS,
		TID:    "000000000000",
		BenID:  "INVALID",
		PIN:    "wrong",
		Herst:  "false",
	}

	var resp fonws.LoginResponse
	err := client.Call(server.URL, req, &resp)
	if err != nil {
		t.Fatalf("Call failed unexpectedly: %v", err)
	}

	// Check error code
	if resp.RC != -4 {
		t.Errorf("Expected RC=-4, got %d", resp.RC)
	}

	// Verify error message is correct
	foErr := fonws.CheckResponse(resp.RC, resp.Msg)
	if foErr == nil {
		t.Error("Expected error for RC=-4")
	}
}
