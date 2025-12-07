package sepa

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

var (
	ErrInvalidIBAN       = errors.New("invalid IBAN format")
	ErrIBANCheckDigit    = errors.New("IBAN check digit validation failed")
	ErrUnsupportedCountry = errors.New("unsupported IBAN country")

	// Basic IBAN pattern: 2 letters + 2 digits + up to 30 alphanumeric
	ibanPattern = regexp.MustCompile(`^[A-Z]{2}[0-9]{2}[A-Z0-9]{1,30}$`)
)

// IBANCountrySpec defines the IBAN length for each country
var ibanCountryLengths = map[string]int{
	"AT": 20, // Austria
	"DE": 22, // Germany
	"CH": 21, // Switzerland
	"LI": 21, // Liechtenstein
	"BE": 16, // Belgium
	"NL": 18, // Netherlands
	"FR": 27, // France
	"IT": 27, // Italy
	"ES": 24, // Spain
	"PT": 25, // Portugal
	"GB": 22, // United Kingdom
	"IE": 22, // Ireland
	"LU": 20, // Luxembourg
	"CZ": 24, // Czech Republic
	"SK": 24, // Slovakia
	"HU": 28, // Hungary
	"PL": 28, // Poland
	"SI": 19, // Slovenia
	"HR": 21, // Croatia
}

// IBANValidationResult contains IBAN validation results
type IBANValidationResult struct {
	IBAN         string `json:"iban"`
	Valid        bool   `json:"valid"`
	CountryCode  string `json:"country_code"`
	BankCode     string `json:"bank_code"`
	BIC          string `json:"bic,omitempty"`
	BankName     string `json:"bank_name,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ValidateIBAN validates an IBAN using the ISO 7064 Mod 97 algorithm
func ValidateIBAN(iban string) error {
	// Normalize: remove spaces and convert to uppercase
	iban = normalizeIBAN(iban)

	// Check basic format
	if !ibanPattern.MatchString(iban) {
		return ErrInvalidIBAN
	}

	// Check country-specific length
	countryCode := iban[:2]
	expectedLen, ok := ibanCountryLengths[countryCode]
	if !ok {
		return ErrUnsupportedCountry
	}

	if len(iban) != expectedLen {
		return ErrInvalidIBAN
	}

	// Validate check digit using Mod 97 algorithm
	if !validateMod97(iban) {
		return ErrIBANCheckDigit
	}

	return nil
}

// ValidateIBANWithDetails validates an IBAN and returns detailed information
func ValidateIBANWithDetails(iban string) *IBANValidationResult {
	result := &IBANValidationResult{
		IBAN: normalizeIBAN(iban),
	}

	if err := ValidateIBAN(iban); err != nil {
		result.Valid = false
		result.ErrorMessage = err.Error()
		return result
	}

	result.Valid = true
	result.CountryCode = result.IBAN[:2]

	// Extract bank code based on country format
	switch result.CountryCode {
	case "AT":
		// Austrian IBAN: AT + 2 check + 5 bank code + 11 account
		result.BankCode = result.IBAN[4:9]
		// Lookup BIC from Austrian bank registry
		if bic, name := LookupAustrianBank(result.BankCode); bic != "" {
			result.BIC = bic
			result.BankName = name
		}
	case "DE":
		// German IBAN: DE + 2 check + 8 bank code + 10 account
		result.BankCode = result.IBAN[4:12]
	case "CH", "LI":
		// Swiss/Liechtenstein IBAN: XX + 2 check + 5 bank code + rest
		result.BankCode = result.IBAN[4:9]
	}

	return result
}

// normalizeIBAN removes spaces and converts to uppercase
func normalizeIBAN(iban string) string {
	iban = strings.ReplaceAll(iban, " ", "")
	iban = strings.ToUpper(iban)
	return iban
}

// validateMod97 implements the ISO 7064 Mod 97 check
func validateMod97(iban string) bool {
	// Move first 4 characters to end
	rearranged := iban[4:] + iban[:4]

	// Convert letters to numbers (A=10, B=11, ..., Z=35)
	var numericStr strings.Builder
	for _, char := range rearranged {
		if char >= 'A' && char <= 'Z' {
			// Convert letter to two-digit number string
			numericStr.WriteString(fmt.Sprintf("%d", int(char-'A'+10)))
		} else {
			numericStr.WriteRune(char)
		}
	}

	// Calculate mod 97 using big integer (IBAN can be up to 34 chars)
	numericValue := new(big.Int)
	numericValue.SetString(numericStr.String(), 10)

	mod := new(big.Int)
	mod.Mod(numericValue, big.NewInt(97))

	return mod.Int64() == 1
}

// CalculateIBANCheckDigit calculates the check digits for a BBAN
// countryCode: 2-letter country code (e.g., "AT")
// bban: Basic Bank Account Number (without country code and check digits)
func CalculateIBANCheckDigit(countryCode, bban string) (string, error) {
	countryCode = strings.ToUpper(countryCode)
	bban = strings.ToUpper(strings.ReplaceAll(bban, " ", ""))

	// Create provisional IBAN with "00" as check digits
	provisional := bban + countryCode + "00"

	// Convert letters to numbers
	var numericStr strings.Builder
	for _, char := range provisional {
		if char >= 'A' && char <= 'Z' {
			numericStr.WriteString(fmt.Sprintf("%d", int(char-'A'+10)))
		} else {
			numericStr.WriteRune(char)
		}
	}

	// Calculate: 98 - (provisional mod 97)
	numericValue := new(big.Int)
	numericValue.SetString(numericStr.String(), 10)

	mod := new(big.Int)
	mod.Mod(numericValue, big.NewInt(97))

	checkDigits := 98 - mod.Int64()

	// Format with leading zero if needed
	return strings.ToUpper(countryCode) + padLeft(int(checkDigits), 2) + bban, nil
}

// FormatIBAN formats an IBAN for display with spaces every 4 characters
func FormatIBAN(iban string) string {
	iban = normalizeIBAN(iban)
	var result strings.Builder
	for i, char := range iban {
		if i > 0 && i%4 == 0 {
			result.WriteRune(' ')
		}
		result.WriteRune(char)
	}
	return result.String()
}

// padLeft pads a number with leading zeros
func padLeft(n, width int) string {
	format := fmt.Sprintf("%%0%dd", width)
	return fmt.Sprintf(format, n)
}
