package sepa

import (
	"errors"
	"strings"
)

// AustrianBank represents a bank in the Austrian banking system
type AustrianBank struct {
	BankCode string `json:"bank_code"`
	BIC      string `json:"bic"`
	Name     string `json:"name"`
}

// austrianBankRegistry maps bank codes (BLZ) to BIC and bank names
// This is a subset of common Austrian banks. A complete implementation
// would load from an external data source (e.g., OeNB bank directory)
var austrianBankRegistry = map[string]AustrianBank{
	// Major Austrian Banks
	"11000": {BankCode: "11000", BIC: "BKAUATWW", Name: "Bank Austria"},
	"12000": {BankCode: "12000", BIC: "GIBAATWW", Name: "Erste Bank"},
	"14000": {BankCode: "14000", BIC: "BAWAATWW", Name: "BAWAG PSK"},
	"14900": {BankCode: "14900", BIC: "BAWAATWW", Name: "BAWAG PSK"},
	"15000": {BankCode: "15000", BIC: "OBKLAT2L", Name: "Oberbank"},
	"16000": {BankCode: "16000", BIC: "BTVAAT22", Name: "BTV - Bank für Tirol und Vorarlberg"},
	"17000": {BankCode: "17000", BIC: "BFKKAT2K", Name: "BKS Bank"},
	"18000": {BankCode: "18000", BIC: "VABORXX", Name: "Volksbank"},
	"19043": {BankCode: "19043", BIC: "BKAUATWW", Name: "Bank Austria (Landesdirektion)"},

	// Raiffeisen Banks (selection of major regional codes)
	"32000": {BankCode: "32000", BIC: "RLNWATWW", Name: "Raiffeisen NÖ-Wien"},
	"33000": {BankCode: "33000", BIC: "RLNWATWW", Name: "Raiffeisenlandesbank NÖ-Wien"},
	"34000": {BankCode: "34000", BIC: "RZOOAT2L", Name: "Raiffeisenlandesbank OÖ"},
	"35000": {BankCode: "35000", BIC: "RVSAAT2S", Name: "Raiffeisenlandesbank Salzburg"},
	"36000": {BankCode: "36000", BIC: "RZTIAT22", Name: "Raiffeisenlandesbank Tirol"},
	"37000": {BankCode: "37000", BIC: "RVVGAT2B", Name: "Raiffeisenlandesbank Vorarlberg"},
	"38000": {BankCode: "38000", BIC: "RZSTAT2G", Name: "Raiffeisenlandesbank Steiermark"},
	"39000": {BankCode: "39000", BIC: "RZKTAT2K", Name: "Raiffeisenlandesbank Kärnten"},

	// Sparkassen (selection)
	"20111": {BankCode: "20111", BIC: "GIBAATWWXXX", Name: "Erste Bank der oesterreichischen Sparkassen"},
	"20205": {BankCode: "20205", BIC: "ABORATWWXXX", Name: "Allgemeine Sparkasse OÖ"},
	"20315": {BankCode: "20315", BIC: "STSPAT2GXXX", Name: "Steiermärkische Sparkasse"},
	"20404": {BankCode: "20404", BIC: "SBGSAT2SXXX", Name: "Salzburger Sparkasse"},
	"20502": {BankCode: "20502", BIC: "SPIHAT22XXX", Name: "Tiroler Sparkasse"},

	// Other Banks
	"19200": {BankCode: "19200", BIC: "INGBATWW", Name: "ING-DiBa Austria"},
	"19500": {BankCode: "19500", BIC: "EABORWW", Name: "easybank"},
	"19600": {BankCode: "19600", BIC: "RZBAATWW", Name: "RZB - Raiffeisen Zentralbank"},
	"60000": {BankCode: "60000", BIC: "OPSKATWW", Name: "Österreichische Postsparkasse (legacy)"},
}

// LookupAustrianBank looks up a bank by its Austrian bank code (BLZ)
// Returns BIC and bank name, or empty strings if not found
func LookupAustrianBank(bankCode string) (bic, name string) {
	bankCode = strings.TrimSpace(bankCode)

	// Direct lookup
	if bank, ok := austrianBankRegistry[bankCode]; ok {
		return bank.BIC, bank.Name
	}

	// Try with leading zeros normalized to 5 digits
	normalized := padBankCode(bankCode)
	if bank, ok := austrianBankRegistry[normalized]; ok {
		return bank.BIC, bank.Name
	}

	return "", ""
}

// LookupBICByIBAN extracts the bank code from an Austrian IBAN and looks up the BIC
func LookupBICByIBAN(iban string) (bic, name string) {
	iban = strings.ReplaceAll(strings.ToUpper(iban), " ", "")

	// Check if it's an Austrian IBAN
	if len(iban) < 9 || iban[:2] != "AT" {
		return "", ""
	}

	// Extract bank code (positions 4-8, 5 digits)
	bankCode := iban[4:9]
	return LookupAustrianBank(bankCode)
}

// GetAllAustrianBanks returns all banks in the registry
func GetAllAustrianBanks() []AustrianBank {
	banks := make([]AustrianBank, 0, len(austrianBankRegistry))
	for _, bank := range austrianBankRegistry {
		banks = append(banks, bank)
	}
	return banks
}

// ErrInvalidBIC is returned when a BIC is invalid
var ErrInvalidBIC = errors.New("invalid BIC format")

// ValidateBIC validates a BIC/SWIFT code format and returns an error if invalid
// BIC format: 4 bank code + 2 country code + 2 location + (3 branch optional)
func ValidateBIC(bic string) error {
	if !validateBICFormat(bic) {
		return ErrInvalidBIC
	}
	return nil
}

// validateBICFormat validates a BIC/SWIFT code format
// BIC format: 4 bank code + 2 country code + 2 location + (3 branch optional)
func validateBICFormat(bic string) bool {
	bic = strings.ToUpper(strings.TrimSpace(bic))

	// BIC can be 8 or 11 characters
	if len(bic) != 8 && len(bic) != 11 {
		return false
	}

	// First 4 characters: bank code (letters only)
	for _, c := range bic[:4] {
		if c < 'A' || c > 'Z' {
			return false
		}
	}

	// Characters 5-6: country code (letters only)
	for _, c := range bic[4:6] {
		if c < 'A' || c > 'Z' {
			return false
		}
	}

	// Characters 7-8: location code (alphanumeric)
	for _, c := range bic[6:8] {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}

	// Characters 9-11 (if present): branch code (alphanumeric)
	if len(bic) == 11 {
		for _, c := range bic[8:11] {
			if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				return false
			}
		}
	}

	return true
}

// DeriveBICFromIBAN attempts to derive the BIC from an IBAN
// Currently only supports Austrian IBANs
func DeriveBICFromIBAN(iban string) string {
	iban = strings.ReplaceAll(strings.ToUpper(iban), " ", "")

	// Only Austrian IBANs supported for now
	if len(iban) >= 2 && iban[:2] == "AT" {
		bic, _ := LookupBICByIBAN(iban)
		return bic
	}

	return ""
}

// padBankCode normalizes a bank code to 5 digits with leading zeros
func padBankCode(code string) string {
	code = strings.TrimSpace(code)
	for len(code) < 5 {
		code = "0" + code
	}
	return code
}
