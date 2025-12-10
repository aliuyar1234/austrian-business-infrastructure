package integration

import (
	"os"
	"path/filepath"
	"testing"

	"austrian-business-infrastructure/internal/store"
)

// T035: Integration test for add/list/remove account flow
func TestAccountAddListRemoveFlow(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "fo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	credPath := filepath.Join(tmpDir, "accounts.enc")
	masterPassword := "test-master-password"

	// Step 1: Create new credential store and add first account
	cs := store.NewCredentialStore()

	acc1 := store.Account{
		Name:  "Company A",
		TID:   "111111111111",
		BenID: "USERA",
		PIN:   "pinA",
	}
	if err := cs.AddAccount(acc1); err != nil {
		t.Fatalf("Failed to add first account: %v", err)
	}

	// Save to file
	if err := cs.Save(credPath, masterPassword); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Step 2: Load and verify
	loaded, err := store.Load(credPath, masterPassword)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	accounts := loaded.ListAccounts()
	if len(accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(accounts))
	}
	if accounts[0] != "Company A" {
		t.Errorf("Expected 'Company A', got '%s'", accounts[0])
	}

	// Step 3: Add second account
	acc2 := store.Account{
		Name:  "Company B",
		TID:   "222222222222",
		BenID: "USERB",
		PIN:   "pinB",
	}
	if err := loaded.AddAccount(acc2); err != nil {
		t.Fatalf("Failed to add second account: %v", err)
	}

	// Try to add duplicate name - should fail
	accDup := store.Account{
		Name:  "Company A",
		TID:   "333333333333",
		BenID: "USERC",
		PIN:   "pinC",
	}
	if err := loaded.AddAccount(accDup); err == nil {
		t.Error("Expected error for duplicate account name")
	}

	// Save and reload
	if err := loaded.Save(credPath, masterPassword); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	loaded2, err := store.Load(credPath, masterPassword)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	accounts = loaded2.ListAccounts()
	if len(accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(accounts))
	}

	// Step 4: Get specific account
	retrieved, err := loaded2.GetAccount("Company A")
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}
	if retrieved.TID != "111111111111" {
		t.Errorf("Wrong TID: got %s", retrieved.TID)
	}

	// Get non-existent account
	_, err = loaded2.GetAccount("Non-existent")
	if err == nil {
		t.Error("Expected error for non-existent account")
	}

	// Step 5: Remove account
	if err := loaded2.RemoveAccount("Company A"); err != nil {
		t.Fatalf("Failed to remove account: %v", err)
	}

	accounts = loaded2.ListAccounts()
	if len(accounts) != 1 {
		t.Errorf("Expected 1 account after removal, got %d", len(accounts))
	}
	if accounts[0] != "Company B" {
		t.Errorf("Expected 'Company B' to remain, got '%s'", accounts[0])
	}

	// Remove non-existent account - should fail
	if err := loaded2.RemoveAccount("Non-existent"); err == nil {
		t.Error("Expected error removing non-existent account")
	}

	// Save final state
	if err := loaded2.Save(credPath, masterPassword); err != nil {
		t.Fatalf("Failed to save final state: %v", err)
	}

	// Verify persistence
	final, err := store.Load(credPath, masterPassword)
	if err != nil {
		t.Fatalf("Failed final load: %v", err)
	}
	if len(final.ListAccounts()) != 1 {
		t.Errorf("Expected 1 account in final state")
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	_, err := store.Load("/non/existent/path/accounts.enc", "password")
	if err == nil {
		t.Error("Expected error loading non-existent file")
	}
}

func TestLoadWithWrongPassword(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	credPath := filepath.Join(tmpDir, "accounts.enc")

	// Create and save with one password
	cs := store.NewCredentialStore()
	cs.AddAccount(store.Account{
		Name:  "Test",
		TID:   "123456789012",
		BenID: "USER",
		PIN:   "pin",
	})
	if err := cs.Save(credPath, "correct-password"); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Try to load with wrong password
	_, err = store.Load(credPath, "wrong-password")
	if err == nil {
		t.Error("Expected error with wrong password")
	}
}
