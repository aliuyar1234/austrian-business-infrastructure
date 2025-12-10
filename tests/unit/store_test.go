package unit

import (
	"testing"

	"austrian-business-infrastructure/internal/store"
)

// T031: Test Account struct validation (TID format, unique name)
func TestAccountValidation(t *testing.T) {
	tests := []struct {
		name        string
		account     store.Account
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid account",
			account: store.Account{
				Name:  "Test GmbH",
				TID:   "123456789012",
				BenID: "WSUSER001",
				PIN:   "secret123",
			},
			expectError: false,
		},
		{
			name: "empty name",
			account: store.Account{
				Name:  "",
				TID:   "123456789012",
				BenID: "WSUSER001",
				PIN:   "secret123",
			},
			expectError: true,
			errorMsg:    "name",
		},
		{
			name: "name too long",
			account: store.Account{
				Name:  string(make([]byte, 101)),
				TID:   "123456789012",
				BenID: "WSUSER001",
				PIN:   "secret123",
			},
			expectError: true,
			errorMsg:    "100",
		},
		{
			name: "TID too short",
			account: store.Account{
				Name:  "Test",
				TID:   "12345678901",
				BenID: "WSUSER001",
				PIN:   "secret123",
			},
			expectError: true,
			errorMsg:    "12 digits",
		},
		{
			name: "TID too long",
			account: store.Account{
				Name:  "Test",
				TID:   "1234567890123",
				BenID: "WSUSER001",
				PIN:   "secret123",
			},
			expectError: true,
			errorMsg:    "12 digits",
		},
		{
			name: "TID with letters",
			account: store.Account{
				Name:  "Test",
				TID:   "12345678901a",
				BenID: "WSUSER001",
				PIN:   "secret123",
			},
			expectError: true,
			errorMsg:    "12 digits",
		},
		{
			name: "empty BenID",
			account: store.Account{
				Name:  "Test",
				TID:   "123456789012",
				BenID: "",
				PIN:   "secret123",
			},
			expectError: true,
			errorMsg:    "benid",
		},
		{
			name: "empty PIN",
			account: store.Account{
				Name:  "Test",
				TID:   "123456789012",
				BenID: "WSUSER001",
				PIN:   "",
			},
			expectError: true,
			errorMsg:    "pin",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.account.Validate()
			if tc.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if tc.expectError && err != nil && tc.errorMsg != "" {
				if !containsStr(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %s", tc.errorMsg, err.Error())
				}
			}
		})
	}
}

// T032: Test CredentialStore JSON serialization
func TestCredentialStoreJSONSerialization(t *testing.T) {
	original := &store.CredentialStore{
		Version: 1,
		Accounts: []store.Account{
			{
				Name:  "Test GmbH",
				TID:   "123456789012",
				BenID: "WSUSER001",
				PIN:   "secret123",
			},
			{
				Name:  "Another Company",
				TID:   "987654321098",
				BenID: "WSUSER002",
				PIN:   "password456",
			},
		},
	}

	// Serialize to JSON
	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Deserialize from JSON
	restored, err := store.FromJSON(data)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Verify
	if restored.Version != original.Version {
		t.Errorf("Version mismatch: got %d, want %d", restored.Version, original.Version)
	}
	if len(restored.Accounts) != len(original.Accounts) {
		t.Errorf("Account count mismatch: got %d, want %d", len(restored.Accounts), len(original.Accounts))
	}
	for i, acc := range original.Accounts {
		if restored.Accounts[i].Name != acc.Name {
			t.Errorf("Account %d name mismatch", i)
		}
		if restored.Accounts[i].TID != acc.TID {
			t.Errorf("Account %d TID mismatch", i)
		}
		if restored.Accounts[i].BenID != acc.BenID {
			t.Errorf("Account %d BenID mismatch", i)
		}
		if restored.Accounts[i].PIN != acc.PIN {
			t.Errorf("Account %d PIN mismatch", i)
		}
	}
}

// T033: Test encrypt/decrypt roundtrip with master password
func TestEncryptDecryptRoundtrip(t *testing.T) {
	masterPassword := "correct-horse-battery-staple"

	cs := &store.CredentialStore{
		Version: 1,
		Accounts: []store.Account{
			{
				Name:  "Test Account",
				TID:   "123456789012",
				BenID: "USER01",
				PIN:   "mysecretpin",
			},
		},
	}

	// Encrypt
	encrypted, err := cs.EncryptStore(masterPassword)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Decrypt
	decrypted, err := store.DecryptStore(encrypted, masterPassword)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	// Verify
	if decrypted.Version != cs.Version {
		t.Errorf("Version mismatch after roundtrip")
	}
	if len(decrypted.Accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(decrypted.Accounts))
	}
	if decrypted.Accounts[0].PIN != "mysecretpin" {
		t.Errorf("PIN mismatch after roundtrip")
	}
}

// T034: Test wrong master password error
func TestWrongMasterPassword(t *testing.T) {
	correctPassword := "correct-password"
	wrongPassword := "wrong-password"

	cs := &store.CredentialStore{
		Version: 1,
		Accounts: []store.Account{
			{
				Name:  "Test",
				TID:   "123456789012",
				BenID: "USER01",
				PIN:   "secret",
			},
		},
	}

	// Encrypt with correct password
	encrypted, err := cs.EncryptStore(correctPassword)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Try to decrypt with wrong password
	_, err = store.DecryptStore(encrypted, wrongPassword)
	if err == nil {
		t.Error("Expected error for wrong password, got nil")
	}
}

// Helper
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
