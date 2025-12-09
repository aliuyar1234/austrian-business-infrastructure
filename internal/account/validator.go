package account

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrInvalidTID            = errors.New("invalid TID: must be 9 digits with valid checksum")
	ErrInvalidBenID          = errors.New("invalid BenID: must be alphanumeric, max 20 characters")
	ErrInvalidDienstgeberNr  = errors.New("invalid Dienstgebernummer: must be 6 digits")
	ErrInvalidPIN            = errors.New("PIN is required")
	ErrInvalidAccountType    = errors.New("invalid account type")
	ErrInvalidUsername       = errors.New("invalid username")
	ErrInvalidCertificatePath = errors.New("certificate path is required for ELDA")
)

// Account types
const (
	AccountTypeFinanzOnline = "finanzonline"
	AccountTypeELDA         = "elda"
	AccountTypeFirmenbuch   = "firmenbuch"
)

// ValidAccountTypes lists all valid account types
var ValidAccountTypes = []string{
	AccountTypeFinanzOnline,
	AccountTypeELDA,
	AccountTypeFirmenbuch,
}

// Regex patterns
var (
	tidPattern         = regexp.MustCompile(`^\d{9}$`)
	benIDPattern       = regexp.MustCompile(`^[a-zA-Z0-9]{1,20}$`)
	dienstgeberPattern = regexp.MustCompile(`^\d{6}$`)
)

// ValidateTID validates a FinanzOnline TID (Steuernummer)
// TID must be 9 digits with valid Modulus 11 checksum
func ValidateTID(tid string) error {
	tid = strings.TrimSpace(tid)

	if !tidPattern.MatchString(tid) {
		return ErrInvalidTID
	}

	// Modulus 11 checksum validation
	// The checksum digit is the 9th digit
	// Weights for positions 1-8: 1, 2, 1, 2, 1, 2, 1, 2
	weights := []int{1, 2, 1, 2, 1, 2, 1, 2}
	sum := 0

	for i := 0; i < 8; i++ {
		digit, _ := strconv.Atoi(string(tid[i]))
		product := digit * weights[i]

		// If product > 9, add digits together (e.g., 14 -> 1+4 = 5)
		if product > 9 {
			product = product/10 + product%10
		}
		sum += product
	}

	// Calculate check digit
	checkDigit := (10 - (sum % 10)) % 10
	actualCheckDigit, _ := strconv.Atoi(string(tid[8]))

	if checkDigit != actualCheckDigit {
		return ErrInvalidTID
	}

	return nil
}

// ValidateBenID validates a FinanzOnline BenID (Benutzer-ID)
// BenID must be alphanumeric, max 20 characters
func ValidateBenID(benID string) error {
	benID = strings.TrimSpace(benID)

	if benID == "" || !benIDPattern.MatchString(benID) {
		return ErrInvalidBenID
	}

	return nil
}

// ValidateDienstgebernummer validates an ELDA Dienstgebernummer
// Must be exactly 6 digits
func ValidateDienstgebernummer(nr string) error {
	nr = strings.TrimSpace(nr)

	if !dienstgeberPattern.MatchString(nr) {
		return ErrInvalidDienstgeberNr
	}

	return nil
}

// ValidateAccountType checks if account type is valid
func ValidateAccountType(accountType string) error {
	for _, t := range ValidAccountTypes {
		if t == accountType {
			return nil
		}
	}
	return ErrInvalidAccountType
}

// ValidatePIN validates that PIN is not empty
func ValidatePIN(pin string) error {
	if strings.TrimSpace(pin) == "" {
		return ErrInvalidPIN
	}
	return nil
}

// ValidateFinanzOnlineCredentials validates FO credentials
func ValidateFinanzOnlineCredentials(tid, benID, pin string) error {
	if err := ValidateTID(tid); err != nil {
		return err
	}
	if err := ValidateBenID(benID); err != nil {
		return err
	}
	if err := ValidatePIN(pin); err != nil {
		return err
	}
	return nil
}

// ValidateELDACredentials validates ELDA credentials
func ValidateELDACredentials(dienstgeberNr, pin, certPath string) error {
	if err := ValidateDienstgebernummer(dienstgeberNr); err != nil {
		return err
	}
	if err := ValidatePIN(pin); err != nil {
		return err
	}
	if strings.TrimSpace(certPath) == "" {
		return ErrInvalidCertificatePath
	}
	return nil
}

// ValidateFirmenbuchCredentials validates Firmenbuch credentials
func ValidateFirmenbuchCredentials(username, password string) error {
	if strings.TrimSpace(username) == "" {
		return ErrInvalidUsername
	}
	if strings.TrimSpace(password) == "" {
		return ErrInvalidPIN // Reuse for password
	}
	return nil
}

// ValidationError holds multiple validation errors
type ValidationError struct {
	Errors map[string]string
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	var parts []string
	for field, msg := range e.Errors {
		parts = append(parts, field+": "+msg)
	}
	return strings.Join(parts, "; ")
}

func (e *ValidationError) Add(field, message string) {
	if e.Errors == nil {
		e.Errors = make(map[string]string)
	}
	e.Errors[field] = message
}

func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}
