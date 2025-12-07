package elda

import (
	"errors"
	"regexp"
	"strconv"
	"time"
)

var (
	ErrInvalidSVNummer       = errors.New("invalid SV-Nummer format")
	ErrSVNummerCheckDigit    = errors.New("SV-Nummer check digit does not match")
	ErrSVNummerBirthDateMismatch = errors.New("SV-Nummer birth date does not match provided date")

	svNummerPattern = regexp.MustCompile(`^\d{10}$`)
)

// ValidateSVNummer validates an Austrian social security number (SV-Nummer)
// Format: NNNN TTMMJJ P where:
//   - NNNN = 4-digit serial number
//   - TTMMJJ = birth date (day, month, year)
//   - P = check digit
func ValidateSVNummer(svNummer string) error {
	if !svNummerPattern.MatchString(svNummer) {
		return ErrInvalidSVNummer
	}

	// Calculate check digit using modulo 11 algorithm
	// Weights for positions 0-8: 3, 7, 9, 5, 8, 4, 2, 1, 6
	weights := []int{3, 7, 9, 5, 8, 4, 2, 1, 6}
	sum := 0

	for i := 0; i < 9; i++ {
		digit, _ := strconv.Atoi(string(svNummer[i]))
		sum += digit * weights[i]
	}

	expectedCheckDigit := sum % 11
	actualCheckDigit, _ := strconv.Atoi(string(svNummer[9]))

	if expectedCheckDigit != actualCheckDigit {
		return ErrSVNummerCheckDigit
	}

	return nil
}

// ValidateSVNummerWithBirthDate validates an SV-Nummer and checks that
// the embedded birth date matches the provided date
func ValidateSVNummerWithBirthDate(svNummer string, birthDate time.Time) error {
	if err := ValidateSVNummer(svNummer); err != nil {
		return err
	}

	// Extract birth date from SV-Nummer (positions 4-9: TTMMJJ)
	dayStr := svNummer[4:6]
	monthStr := svNummer[6:8]
	yearStr := svNummer[8:10]

	day, _ := strconv.Atoi(dayStr)
	month, _ := strconv.Atoi(monthStr)
	yearShort, _ := strconv.Atoi(yearStr)

	// Determine century based on comparison with current year
	currentYear := time.Now().Year()
	century := (currentYear / 100) * 100
	year := century + yearShort

	// If the resulting year is in the future, use previous century
	if year > currentYear {
		year -= 100
	}

	// Compare dates
	embeddedDay := day
	embeddedMonth := time.Month(month)
	embeddedYear := year

	if birthDate.Day() != embeddedDay ||
		birthDate.Month() != embeddedMonth ||
		birthDate.Year() != embeddedYear {
		return ErrSVNummerBirthDateMismatch
	}

	return nil
}

// ExtractBirthDateFromSVNummer extracts the birth date from an SV-Nummer
// Returns the extracted date or error if SV-Nummer is invalid
func ExtractBirthDateFromSVNummer(svNummer string) (time.Time, error) {
	if err := ValidateSVNummer(svNummer); err != nil {
		return time.Time{}, err
	}

	dayStr := svNummer[4:6]
	monthStr := svNummer[6:8]
	yearStr := svNummer[8:10]

	day, _ := strconv.Atoi(dayStr)
	month, _ := strconv.Atoi(monthStr)
	yearShort, _ := strconv.Atoi(yearStr)

	// Determine century
	currentYear := time.Now().Year()
	century := (currentYear / 100) * 100
	year := century + yearShort

	if year > currentYear {
		year -= 100
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

// FormatSVNummer formats an SV-Nummer for display (NNNN TTMMJJ P)
func FormatSVNummer(svNummer string) string {
	if len(svNummer) != 10 {
		return svNummer
	}
	return svNummer[:4] + " " + svNummer[4:10]
}
