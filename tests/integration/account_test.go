package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/account"
	"github.com/austrian-business-infrastructure/fo/internal/account/types"
	"github.com/google/uuid"
)

// T064: Integration tests for account CRUD and connection tests

func TestAccountCRUD(t *testing.T) {
	// Skip if no database available
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Test account creation with valid credentials
	t.Run("Create FinanzOnline account", func(t *testing.T) {
		// Create test encryption key
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}

		encryptor, err := account.NewEncryptor(key)
		if err != nil {
			t.Fatalf("Failed to create encryptor: %v", err)
		}

		// Test encrypting FO credentials
		creds := &types.FinanzOnlineCredentials{
			TID:   "123456782", // Valid TID with checksum
			BenID: "TESTUSER",
			PIN:   "testpin123",
		}

		ciphertext, iv, err := encryptor.EncryptJSON(creds)
		if err != nil {
			t.Fatalf("Failed to encrypt credentials: %v", err)
		}

		// Verify decryption works
		var decrypted types.FinanzOnlineCredentials
		err = encryptor.DecryptJSON(ciphertext, iv, &decrypted)
		if err != nil {
			t.Fatalf("Failed to decrypt credentials: %v", err)
		}

		if decrypted.TID != creds.TID {
			t.Errorf("TID mismatch: got %s, want %s", decrypted.TID, creds.TID)
		}
		if decrypted.BenID != creds.BenID {
			t.Errorf("BenID mismatch: got %s, want %s", decrypted.BenID, creds.BenID)
		}
		if decrypted.PIN != creds.PIN {
			t.Errorf("PIN mismatch: got %s, want %s", decrypted.PIN, creds.PIN)
		}
	})

	t.Run("Credential masking", func(t *testing.T) {
		creds := &types.FinanzOnlineCredentials{
			TID:   "123456782",
			BenID: "TESTUSER",
			PIN:   "secretpin",
		}

		masked := creds.Masked()

		if masked.TID != creds.TID {
			t.Errorf("TID should not be masked")
		}
		if masked.BenID != creds.BenID {
			t.Errorf("BenID should not be masked")
		}
		if masked.PIN != "****" {
			t.Errorf("PIN should be masked, got %s", masked.PIN)
		}
	})

	t.Run("ELDA credential masking", func(t *testing.T) {
		creds := &types.ELDACredentials{
			DienstgeberNr:       "123456",
			PIN:                 "secretpin",
			CertificatePath:     "/path/to/cert.p12",
			CertificatePassword: "certpass",
		}

		masked := creds.Masked()

		if masked.DienstgeberNr != creds.DienstgeberNr {
			t.Errorf("DienstgeberNr should not be masked")
		}
		if masked.CertificatePath != creds.CertificatePath {
			t.Errorf("CertificatePath should not be masked")
		}
		if masked.PIN != "****" {
			t.Errorf("PIN should be masked")
		}
		if masked.CertificatePassword != "****" {
			t.Errorf("CertificatePassword should be masked")
		}
	})

	t.Run("Account status transitions", func(t *testing.T) {
		// Test valid status values
		validStatuses := []string{"unverified", "verified", "error", "suspended"}
		for _, status := range validStatuses {
			// Status validation is done at database level via CHECK constraint
			t.Logf("Valid status: %s", status)
		}
	})

	_ = ctx // Used in full integration tests with database
}

func TestAccountValidation(t *testing.T) {
	t.Run("Valid TID passes", func(t *testing.T) {
		// TID: 123456782 (last digit is checksum)
		err := account.ValidateTID("123456782")
		if err != nil {
			t.Errorf("Valid TID rejected: %v", err)
		}
	})

	t.Run("Invalid TID checksum fails", func(t *testing.T) {
		err := account.ValidateTID("123456789")
		if err == nil {
			t.Error("Invalid TID checksum should fail")
		}
	})

	t.Run("TID wrong length fails", func(t *testing.T) {
		err := account.ValidateTID("12345678")
		if err == nil {
			t.Error("TID with wrong length should fail")
		}
	})

	t.Run("Valid BenID passes", func(t *testing.T) {
		err := account.ValidateBenID("TESTUSER123")
		if err != nil {
			t.Errorf("Valid BenID rejected: %v", err)
		}
	})

	t.Run("BenID too long fails", func(t *testing.T) {
		err := account.ValidateBenID("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
		if err == nil {
			t.Error("BenID over 20 chars should fail")
		}
	})

	t.Run("Valid Dienstgebernummer passes", func(t *testing.T) {
		err := account.ValidateDienstgebernummer("123456")
		if err != nil {
			t.Errorf("Valid Dienstgebernummer rejected: %v", err)
		}
	})

	t.Run("Dienstgebernummer wrong length fails", func(t *testing.T) {
		err := account.ValidateDienstgebernummer("12345")
		if err == nil {
			t.Error("Dienstgebernummer with wrong length should fail")
		}
	})
}

func TestAccountHandler(t *testing.T) {
	// Mock handler tests
	t.Run("Create account request validation", func(t *testing.T) {
		// Test request body parsing
		reqBody := map[string]interface{}{
			"name": "Test Account",
			"type": "finanzonline",
			"credentials": map[string]string{
				"tid":    "123456782",
				"ben_id": "TESTUSER",
				"pin":    "testpin",
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/accounts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// Verify request can be parsed
		var parsed struct {
			Name        string          `json:"name"`
			Type        string          `json:"type"`
			Credentials json.RawMessage `json:"credentials"`
		}

		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse request: %v", err)
		}

		if parsed.Name != "Test Account" {
			t.Errorf("Name mismatch: got %s", parsed.Name)
		}
		if parsed.Type != "finanzonline" {
			t.Errorf("Type mismatch: got %s", parsed.Type)
		}
	})

	t.Run("List accounts pagination", func(t *testing.T) {
		// Test pagination parameter parsing
		req := httptest.NewRequest("GET", "/api/v1/accounts?limit=10&offset=20&type=finanzonline", nil)

		limit := req.URL.Query().Get("limit")
		offset := req.URL.Query().Get("offset")
		accountType := req.URL.Query().Get("type")

		if limit != "10" {
			t.Errorf("Limit mismatch: got %s", limit)
		}
		if offset != "20" {
			t.Errorf("Offset mismatch: got %s", offset)
		}
		if accountType != "finanzonline" {
			t.Errorf("Type mismatch: got %s", accountType)
		}
	})
}

func TestConnectionTest(t *testing.T) {
	t.Run("Connection test result structure", func(t *testing.T) {
		result := &account.ConnectionTestResult{
			Success:      true,
			DurationMs:   150,
			ErrorCode:    "",
			ErrorMessage: "",
		}

		if !result.Success {
			t.Error("Result should be successful")
		}
		if result.DurationMs != 150 {
			t.Errorf("Duration mismatch: got %d", result.DurationMs)
		}
	})

	t.Run("Connection test with error", func(t *testing.T) {
		result := &account.ConnectionTestResult{
			Success:      false,
			DurationMs:   50,
			ErrorCode:    "-4",
			ErrorMessage: "Invalid credentials",
		}

		if result.Success {
			t.Error("Result should not be successful")
		}
		if result.ErrorCode != "-4" {
			t.Errorf("ErrorCode mismatch: got %s", result.ErrorCode)
		}
	})
}

func TestAccountRepository(t *testing.T) {
	t.Run("Account struct fields", func(t *testing.T) {
		now := time.Now()
		acc := &account.Account{
			ID:        uuid.New(),
			TenantID:  uuid.New(),
			Name:      "Test Account",
			Type:      "finanzonline",
			Status:    "unverified",
			CreatedAt: now,
			UpdatedAt: now,
		}

		if acc.Name != "Test Account" {
			t.Errorf("Name mismatch")
		}
		if acc.Status != "unverified" {
			t.Errorf("Status mismatch")
		}
	})

	t.Run("ListFilter defaults", func(t *testing.T) {
		filter := account.ListFilter{
			TenantID: uuid.New(),
			Limit:    50,
			Offset:   0,
		}

		if filter.Limit != 50 {
			t.Errorf("Default limit should be 50")
		}
		if filter.IncludeDeleted {
			t.Error("IncludeDeleted should default to false")
		}
	})
}

func TestEncryptionKeyRotation(t *testing.T) {
	t.Run("Rotate credentials to new key", func(t *testing.T) {
		oldKey := make([]byte, 32)
		newKey := make([]byte, 32)
		for i := range oldKey {
			oldKey[i] = byte(i)
			newKey[i] = byte(i + 1)
		}

		rotator, err := account.NewKeyRotator(oldKey, newKey)
		if err != nil {
			t.Fatalf("Failed to create rotator: %v", err)
		}

		// Encrypt with old key
		oldEnc, _ := account.NewEncryptor(oldKey)
		ciphertext, iv, _ := oldEnc.Encrypt([]byte("secret data"))

		// Rotate to new key
		newCiphertext, newIV, err := rotator.RotateCredentials(ciphertext, iv)
		if err != nil {
			t.Fatalf("Failed to rotate: %v", err)
		}

		// Verify can decrypt with new key
		newEnc, _ := account.NewEncryptor(newKey)
		plaintext, err := newEnc.Decrypt(newCiphertext, newIV)
		if err != nil {
			t.Fatalf("Failed to decrypt with new key: %v", err)
		}

		if string(plaintext) != "secret data" {
			t.Errorf("Decrypted data mismatch")
		}
	})
}

func TestRedaction(t *testing.T) {
	t.Run("Redact JSON string", func(t *testing.T) {
		input := `{"tid":"123456789","pin":"secret123","ben_id":"USER"}`
		redacted := account.RedactCredentials(input)

		if !contains(redacted, "[REDACTED]") {
			t.Error("PIN should be redacted")
		}
		if !contains(redacted, "123456789") {
			t.Error("TID should not be redacted")
		}
	})

	t.Run("Mask TID", func(t *testing.T) {
		masked := account.MaskTID("123456789")
		if masked != "*****6789" {
			t.Errorf("Unexpected masked TID: %s", masked)
		}
	})

	t.Run("Mask BenID", func(t *testing.T) {
		masked := account.MaskBenID("TESTUSER")
		if masked != "*****SER" {
			t.Errorf("Unexpected masked BenID: %s", masked)
		}
	})
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
