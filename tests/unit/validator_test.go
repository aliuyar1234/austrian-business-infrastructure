package unit

import (
	"testing"

	"austrian-business-infrastructure/internal/account"
)

// T067: Unit tests for validators

func TestValidateTID(t *testing.T) {
	// Note: We need to calculate actual valid TIDs
	// Using Modulus 11 with weights 1,2,1,2,1,2,1,2
	// Let's calculate: 12345678X where X is checksum
	// 1*1=1, 2*2=4, 3*1=3, 4*2=8, 5*1=5, 6*2=12->1+2=3, 7*1=7, 8*2=16->1+6=7
	// Sum = 1+4+3+8+5+3+7+7 = 38
	// Check = (10 - 38%10) % 10 = (10-8) % 10 = 2
	// So 123456782 is valid

	t.Run("Valid TID passes", func(t *testing.T) {
		err := account.ValidateTID("123456782")
		if err != nil {
			t.Errorf("Valid TID '123456782' rejected: %v", err)
		}
	})

	t.Run("Valid TID with leading zeros", func(t *testing.T) {
		// 000000000: all zeros, sum=0, check=(10-0)%10=0
		err := account.ValidateTID("000000000")
		if err != nil {
			t.Errorf("Valid TID '000000000' rejected: %v", err)
		}
	})

	t.Run("Valid TID with whitespace trimmed", func(t *testing.T) {
		err := account.ValidateTID("  123456782  ")
		if err != nil {
			t.Errorf("Valid TID with whitespace rejected: %v", err)
		}
	})

	// Invalid TIDs
	invalidTIDs := []struct {
		tid     string
		comment string
	}{
		{"123456789", "Wrong checksum"},
		{"12345678", "Too short (8 digits)"},
		{"1234567890", "Too long (10 digits)"},
		{"12345678a", "Contains letter"},
		{"12345 789", "Contains space"},
		{"", "Empty string"},
		{"abcdefghi", "All letters"},
		{"-12345678", "Contains minus"},
	}

	for _, tc := range invalidTIDs {
		t.Run("Invalid TID: "+tc.comment, func(t *testing.T) {
			err := account.ValidateTID(tc.tid)
			if err == nil {
				t.Errorf("Invalid TID '%s' should fail: %s", tc.tid, tc.comment)
			}
			if err != account.ErrInvalidTID {
				t.Errorf("Expected ErrInvalidTID, got %v", err)
			}
		})
	}

	t.Run("Checksum validation", func(t *testing.T) {
		// Test that checksum actually matters
		// 123456782 is valid, 123456783 should be invalid
		err := account.ValidateTID("123456783")
		if err == nil {
			t.Error("TID with wrong checksum should fail")
		}
	})

	// Test specific checksum calculations
	checksumTests := []struct {
		base     string
		checksum int
	}{
		// Base "12345678" -> checksum 2
		{"12345678", 2},
	}

	for _, tc := range checksumTests {
		t.Run("Checksum calculation for "+tc.base, func(t *testing.T) {
			validTID := tc.base + string(rune('0'+tc.checksum))
			err := account.ValidateTID(validTID)
			if err != nil {
				t.Errorf("Calculated checksum %d should be valid for %s: %v", tc.checksum, tc.base, err)
			}

			// Wrong checksum should fail
			wrongChecksum := (tc.checksum + 1) % 10
			invalidTID := tc.base + string(rune('0'+wrongChecksum))
			err = account.ValidateTID(invalidTID)
			if err == nil {
				t.Errorf("Wrong checksum %d should fail for %s", wrongChecksum, tc.base)
			}
		})
	}
}

func TestValidateBenID(t *testing.T) {
	validBenIDs := []struct {
		benID   string
		comment string
	}{
		{"USER1", "Short alphanumeric"},
		{"A", "Single character"},
		{"ABCDEFGHIJKLMNOPQRST", "Max length (20)"},
		{"user123", "Lowercase allowed"},
		{"USER", "All uppercase"},
		{"12345", "All numbers"},
		{"Ab1Cd2Ef3Gh4Ij5Kl6Mn", "Mixed 20 chars"},
	}

	for _, tc := range validBenIDs {
		t.Run("Valid BenID: "+tc.comment, func(t *testing.T) {
			err := account.ValidateBenID(tc.benID)
			if err != nil {
				t.Errorf("Valid BenID '%s' rejected: %v", tc.benID, err)
			}
		})
	}

	t.Run("Valid BenID with whitespace trimmed", func(t *testing.T) {
		err := account.ValidateBenID("  USER123  ")
		if err != nil {
			t.Errorf("BenID with whitespace should be trimmed and valid: %v", err)
		}
	})

	invalidBenIDs := []struct {
		benID   string
		comment string
	}{
		{"", "Empty string"},
		{"   ", "Only whitespace"},
		{"ABCDEFGHIJKLMNOPQRSTU", "Too long (21 chars)"},
		{"USER-1", "Contains hyphen"},
		{"USER_1", "Contains underscore"},
		{"USER.1", "Contains dot"},
		{"USER 1", "Contains space"},
		{"USER@1", "Contains @"},
		{"ÜSER1", "Contains umlaut"},
	}

	for _, tc := range invalidBenIDs {
		t.Run("Invalid BenID: "+tc.comment, func(t *testing.T) {
			err := account.ValidateBenID(tc.benID)
			if err == nil {
				t.Errorf("Invalid BenID '%s' should fail: %s", tc.benID, tc.comment)
			}
			if err != account.ErrInvalidBenID {
				t.Errorf("Expected ErrInvalidBenID, got %v", err)
			}
		})
	}
}

func TestValidateDienstgebernummer(t *testing.T) {
	validNumbers := []struct {
		nr      string
		comment string
	}{
		{"123456", "Standard 6 digits"},
		{"000000", "All zeros"},
		{"999999", "All nines"},
		{"000001", "Leading zeros"},
	}

	for _, tc := range validNumbers {
		t.Run("Valid Dienstgebernummer: "+tc.comment, func(t *testing.T) {
			err := account.ValidateDienstgebernummer(tc.nr)
			if err != nil {
				t.Errorf("Valid Dienstgebernummer '%s' rejected: %v", tc.nr, err)
			}
		})
	}

	t.Run("Valid with whitespace trimmed", func(t *testing.T) {
		err := account.ValidateDienstgebernummer("  123456  ")
		if err != nil {
			t.Errorf("Dienstgebernummer with whitespace should be valid: %v", err)
		}
	})

	invalidNumbers := []struct {
		nr      string
		comment string
	}{
		{"12345", "Too short (5 digits)"},
		{"1234567", "Too long (7 digits)"},
		{"12345a", "Contains letter"},
		{"", "Empty string"},
		{"12 456", "Contains space"},
		{"-12345", "Contains minus"},
	}

	for _, tc := range invalidNumbers {
		t.Run("Invalid Dienstgebernummer: "+tc.comment, func(t *testing.T) {
			err := account.ValidateDienstgebernummer(tc.nr)
			if err == nil {
				t.Errorf("Invalid Dienstgebernummer '%s' should fail: %s", tc.nr, tc.comment)
			}
			if err != account.ErrInvalidDienstgeberNr {
				t.Errorf("Expected ErrInvalidDienstgeberNr, got %v", err)
			}
		})
	}
}

func TestValidateAccountType(t *testing.T) {
	validTypes := []string{"finanzonline", "elda", "firmenbuch"}

	for _, accountType := range validTypes {
		t.Run("Valid type: "+accountType, func(t *testing.T) {
			err := account.ValidateAccountType(accountType)
			if err != nil {
				t.Errorf("Valid account type '%s' rejected: %v", accountType, err)
			}
		})
	}

	invalidTypes := []string{
		"",
		"invalid",
		"FINANZONLINE", // Case sensitive
		"Elda",
		"finanz-online",
		"finanzonline ",
	}

	for _, accountType := range invalidTypes {
		t.Run("Invalid type: "+accountType, func(t *testing.T) {
			err := account.ValidateAccountType(accountType)
			if err == nil {
				t.Errorf("Invalid account type '%s' should fail", accountType)
			}
			if err != account.ErrInvalidAccountType {
				t.Errorf("Expected ErrInvalidAccountType, got %v", err)
			}
		})
	}
}

func TestValidatePIN(t *testing.T) {
	validPINs := []string{
		"1234",
		"secretpin",
		"a",
		"very-long-pin-with-special-chars!@#$%",
		"äöü", // Unicode allowed
	}

	for _, pin := range validPINs {
		t.Run("Valid PIN", func(t *testing.T) {
			err := account.ValidatePIN(pin)
			if err != nil {
				t.Errorf("Valid PIN rejected: %v", err)
			}
		})
	}

	invalidPINs := []struct {
		pin     string
		comment string
	}{
		{"", "Empty string"},
		{"   ", "Only whitespace"},
		{"\t\n", "Only control characters"},
	}

	for _, tc := range invalidPINs {
		t.Run("Invalid PIN: "+tc.comment, func(t *testing.T) {
			err := account.ValidatePIN(tc.pin)
			if err == nil {
				t.Errorf("Invalid PIN should fail: %s", tc.comment)
			}
			if err != account.ErrInvalidPIN {
				t.Errorf("Expected ErrInvalidPIN, got %v", err)
			}
		})
	}
}

func TestValidateFinanzOnlineCredentials(t *testing.T) {
	t.Run("Valid credentials", func(t *testing.T) {
		err := account.ValidateFinanzOnlineCredentials("123456782", "USER1", "pin123")
		if err != nil {
			t.Errorf("Valid FO credentials rejected: %v", err)
		}
	})

	t.Run("Invalid TID", func(t *testing.T) {
		err := account.ValidateFinanzOnlineCredentials("123456789", "USER1", "pin123")
		if err != account.ErrInvalidTID {
			t.Errorf("Expected ErrInvalidTID, got %v", err)
		}
	})

	t.Run("Invalid BenID", func(t *testing.T) {
		err := account.ValidateFinanzOnlineCredentials("123456782", "", "pin123")
		if err != account.ErrInvalidBenID {
			t.Errorf("Expected ErrInvalidBenID, got %v", err)
		}
	})

	t.Run("Invalid PIN", func(t *testing.T) {
		err := account.ValidateFinanzOnlineCredentials("123456782", "USER1", "")
		if err != account.ErrInvalidPIN {
			t.Errorf("Expected ErrInvalidPIN, got %v", err)
		}
	})

	t.Run("All invalid - returns first error", func(t *testing.T) {
		// TID is checked first
		err := account.ValidateFinanzOnlineCredentials("bad", "", "")
		if err != account.ErrInvalidTID {
			t.Errorf("Expected ErrInvalidTID (first check), got %v", err)
		}
	})
}

func TestValidateELDACredentials(t *testing.T) {
	t.Run("Valid credentials", func(t *testing.T) {
		err := account.ValidateELDACredentials("123456", "pin123", "/path/to/cert.p12")
		if err != nil {
			t.Errorf("Valid ELDA credentials rejected: %v", err)
		}
	})

	t.Run("Invalid Dienstgebernummer", func(t *testing.T) {
		err := account.ValidateELDACredentials("12345", "pin123", "/path/to/cert.p12")
		if err != account.ErrInvalidDienstgeberNr {
			t.Errorf("Expected ErrInvalidDienstgeberNr, got %v", err)
		}
	})

	t.Run("Invalid PIN", func(t *testing.T) {
		err := account.ValidateELDACredentials("123456", "", "/path/to/cert.p12")
		if err != account.ErrInvalidPIN {
			t.Errorf("Expected ErrInvalidPIN, got %v", err)
		}
	})

	t.Run("Missing certificate path", func(t *testing.T) {
		err := account.ValidateELDACredentials("123456", "pin123", "")
		if err != account.ErrInvalidCertificatePath {
			t.Errorf("Expected ErrInvalidCertificatePath, got %v", err)
		}
	})

	t.Run("Certificate path with whitespace only", func(t *testing.T) {
		err := account.ValidateELDACredentials("123456", "pin123", "   ")
		if err != account.ErrInvalidCertificatePath {
			t.Errorf("Expected ErrInvalidCertificatePath for whitespace path, got %v", err)
		}
	})
}

func TestValidateFirmenbuchCredentials(t *testing.T) {
	t.Run("Valid credentials", func(t *testing.T) {
		err := account.ValidateFirmenbuchCredentials("username", "password123")
		if err != nil {
			t.Errorf("Valid Firmenbuch credentials rejected: %v", err)
		}
	})

	t.Run("Empty username", func(t *testing.T) {
		err := account.ValidateFirmenbuchCredentials("", "password123")
		if err != account.ErrInvalidUsername {
			t.Errorf("Expected ErrInvalidUsername, got %v", err)
		}
	})

	t.Run("Whitespace username", func(t *testing.T) {
		err := account.ValidateFirmenbuchCredentials("   ", "password123")
		if err != account.ErrInvalidUsername {
			t.Errorf("Expected ErrInvalidUsername for whitespace username, got %v", err)
		}
	})

	t.Run("Empty password", func(t *testing.T) {
		err := account.ValidateFirmenbuchCredentials("username", "")
		if err != account.ErrInvalidPIN {
			t.Errorf("Expected ErrInvalidPIN (reused for password), got %v", err)
		}
	})
}

func TestValidationError(t *testing.T) {
	t.Run("Empty validation error", func(t *testing.T) {
		ve := &account.ValidationError{}
		if ve.HasErrors() {
			t.Error("Empty ValidationError should not have errors")
		}
		if ve.Error() != "validation failed" {
			t.Errorf("Unexpected error message: %s", ve.Error())
		}
	})

	t.Run("Add single error", func(t *testing.T) {
		ve := &account.ValidationError{}
		ve.Add("field1", "error message 1")

		if !ve.HasErrors() {
			t.Error("ValidationError should have errors after Add")
		}
		if ve.Errors["field1"] != "error message 1" {
			t.Error("Error message not stored correctly")
		}
	})

	t.Run("Add multiple errors", func(t *testing.T) {
		ve := &account.ValidationError{}
		ve.Add("tid", "invalid TID")
		ve.Add("ben_id", "invalid BenID")
		ve.Add("pin", "PIN required")

		if len(ve.Errors) != 3 {
			t.Errorf("Expected 3 errors, got %d", len(ve.Errors))
		}

		errStr := ve.Error()
		if errStr == "" {
			t.Error("Error string should not be empty")
		}
		// Check that error string contains all fields
		if !validatorContains(errStr, "tid") || !validatorContains(errStr, "ben_id") || !validatorContains(errStr, "pin") {
			t.Errorf("Error string missing fields: %s", errStr)
		}
	})

	t.Run("Overwrite error for same field", func(t *testing.T) {
		ve := &account.ValidationError{}
		ve.Add("field", "first error")
		ve.Add("field", "second error")

		if ve.Errors["field"] != "second error" {
			t.Error("Second error should overwrite first")
		}
		if len(ve.Errors) != 1 {
			t.Error("Should only have one error entry for duplicate field")
		}
	})
}

func TestAccountTypeConstants(t *testing.T) {
	t.Run("Account type constants defined", func(t *testing.T) {
		if account.AccountTypeFinanzOnline != "finanzonline" {
			t.Error("AccountTypeFinanzOnline constant incorrect")
		}
		if account.AccountTypeELDA != "elda" {
			t.Error("AccountTypeELDA constant incorrect")
		}
		if account.AccountTypeFirmenbuch != "firmenbuch" {
			t.Error("AccountTypeFirmenbuch constant incorrect")
		}
	})

	t.Run("ValidAccountTypes slice", func(t *testing.T) {
		if len(account.ValidAccountTypes) != 3 {
			t.Errorf("Expected 3 valid account types, got %d", len(account.ValidAccountTypes))
		}

		// All constants should be in the slice
		found := map[string]bool{}
		for _, t := range account.ValidAccountTypes {
			found[t] = true
		}

		if !found[account.AccountTypeFinanzOnline] {
			t.Error("ValidAccountTypes missing finanzonline")
		}
		if !found[account.AccountTypeELDA] {
			t.Error("ValidAccountTypes missing elda")
		}
		if !found[account.AccountTypeFirmenbuch] {
			t.Error("ValidAccountTypes missing firmenbuch")
		}
	})
}

// Helper function - use strings.Contains instead to avoid redeclaration
func validatorContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
