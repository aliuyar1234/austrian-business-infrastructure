package mbgm

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/austrian-business-infrastructure/fo/internal/elda"
)

// ValidationResult contains the result of validating an mBGM
type ValidationResult struct {
	Valid    bool                    `json:"valid"`
	Errors   []FieldValidationError  `json:"errors,omitempty"`
	Warnings []string                `json:"warnings,omitempty"`
}

// FieldValidationError represents a single validation error for a field
type FieldValidationError struct {
	Field      string `json:"field"`
	Message    string `json:"message"`
	PositionID string `json:"position_id,omitempty"`
	Index      int    `json:"index,omitempty"`
}

// Validator validates mBGM data
type Validator struct {
	repo *Repository
}

// NewValidator creates a new mBGM validator
func NewValidator(repo *Repository) *Validator {
	return &Validator{repo: repo}
}

// ValidateMBGM validates an entire mBGM with all positions
func (v *Validator) ValidateMBGM(mbgm *elda.MBGM) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Validate header
	v.validateHeader(mbgm, result)

	// Validate positions
	if len(mbgm.Positionen) == 0 {
		result.addError("positionen", "Mindestens eine Position erforderlich", "", 0)
	}

	svNummern := make(map[string]int) // Track duplicate SV-Nummern
	for i, pos := range mbgm.Positionen {
		v.validatePosition(pos, i, result)

		// Check for duplicates
		if existing, ok := svNummern[pos.SVNummer]; ok {
			result.addError("sv_nummer", fmt.Sprintf("SV-Nummer %s bereits in Position %d verwendet", pos.SVNummer, existing+1), pos.ID.String(), i)
		}
		svNummern[pos.SVNummer] = i
	}

	// Add deadline warning
	deadline := elda.GetMBGMDeadline(mbgm.Year, mbgm.Month)
	daysUntil := int(time.Until(deadline).Hours() / 24)
	if daysUntil < 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Meldefrist bereits abgelaufen (war am %s)", deadline.Format("02.01.2006")))
	} else if daysUntil <= 3 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Meldefrist endet in %d Tagen (%s)", daysUntil, deadline.Format("02.01.2006")))
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// ValidatePosition validates a single mBGM position
func (v *Validator) ValidatePosition(pos *elda.MBGMPosition, year, month int) *ValidationResult {
	result := &ValidationResult{Valid: true}
	v.validatePosition(pos, 0, result)
	result.Valid = len(result.Errors) == 0
	return result
}

// validateHeader validates the mBGM header fields
func (v *Validator) validateHeader(mbgm *elda.MBGM, result *ValidationResult) {
	// Validate year
	currentYear := time.Now().Year()
	if mbgm.Year < 2020 || mbgm.Year > currentYear+1 {
		result.addError("year", fmt.Sprintf("Ungültiges Jahr: %d (erlaubt: 2020-%d)", mbgm.Year, currentYear+1), "", 0)
	}

	// Validate month
	if mbgm.Month < 1 || mbgm.Month > 12 {
		result.addError("month", fmt.Sprintf("Ungültiger Monat: %d (erlaubt: 1-12)", mbgm.Month), "", 0)
	}

	// Validate ELDA account ID
	if mbgm.ELDAAccountID == uuid.Nil {
		result.addError("elda_account_id", "ELDA-Konto ID erforderlich", "", 0)
	}
}

// validatePosition validates a single position
func (v *Validator) validatePosition(pos *elda.MBGMPosition, index int, result *ValidationResult) {
	posID := pos.ID.String()
	if pos.ID == uuid.Nil {
		posID = ""
	}

	// Validate SV-Nummer
	if pos.SVNummer == "" {
		result.addError("sv_nummer", "SV-Nummer erforderlich", posID, index)
	} else if err := ValidateSVNummer(pos.SVNummer); err != nil {
		result.addError("sv_nummer", err.Error(), posID, index)
	}

	// Validate name
	if pos.Familienname == "" {
		result.addError("familienname", "Familienname erforderlich", posID, index)
	} else if len(pos.Familienname) > 100 {
		result.addError("familienname", "Familienname zu lang (max. 100 Zeichen)", posID, index)
	}

	if pos.Vorname == "" {
		result.addError("vorname", "Vorname erforderlich", posID, index)
	} else if len(pos.Vorname) > 100 {
		result.addError("vorname", "Vorname zu lang (max. 100 Zeichen)", posID, index)
	}

	// Validate Beitragsgruppe
	if pos.Beitragsgruppe == "" {
		result.addError("beitragsgruppe", "Beitragsgruppe erforderlich", posID, index)
	} else if !v.isValidBeitragsgruppe(pos.Beitragsgruppe) {
		result.addError("beitragsgruppe", fmt.Sprintf("Ungültige Beitragsgruppe: %s", pos.Beitragsgruppe), posID, index)
	}

	// Validate Beitragsgrundlage
	if pos.Beitragsgrundlage < 0 {
		result.addError("beitragsgrundlage", "Beitragsgrundlage darf nicht negativ sein", posID, index)
	} else {
		// Check against limits
		v.validateBeitragsgrundlage(pos, index, result)
	}

	// Validate Sonderzahlung
	if pos.Sonderzahlung < 0 {
		result.addError("sonderzahlung", "Sonderzahlung darf nicht negativ sein", posID, index)
	}

	// Validate Wochenstunden
	if pos.Wochenstunden != nil {
		if *pos.Wochenstunden < 0 || *pos.Wochenstunden > 60 {
			result.addError("wochenstunden", "Wochenstunden müssen zwischen 0 und 60 liegen", posID, index)
		}
	}

	// Validate date range if provided
	if pos.VonDatum != nil && pos.BisDatum != nil {
		if pos.BisDatum.Before(*pos.VonDatum) {
			result.addError("bis_datum", "Bis-Datum muss nach Von-Datum liegen", posID, index)
		}
	}
}

// validateBeitragsgrundlage checks the amount against SV limits
func (v *Validator) validateBeitragsgrundlage(pos *elda.MBGMPosition, index int, result *ValidationResult) {
	// Check Geringfügigkeitsgrenze
	if elda.IsGeringfuegig(pos.Beitragsgrundlage, time.Now().Year()) {
		// This is a warning, not an error - geringfügig is valid
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Position %d (%s): Beitragsgrundlage %.2f€ liegt unter der Geringfügigkeitsgrenze",
				index+1, pos.Familienname, pos.Beitragsgrundlage))
	}

	// Check Höchstbeitragsgrundlage
	if elda.ExceedsHoechstbeitrag(pos.Beitragsgrundlage, time.Now().Year()) {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Position %d (%s): Beitragsgrundlage %.2f€ übersteigt die Höchstbeitragsgrundlage - wird auf %.2f€ gekappt",
				index+1, pos.Familienname, pos.Beitragsgrundlage, elda.HoechstbeitragsGrundlage2025))
	}
}

// isValidBeitragsgruppe checks if a Beitragsgruppe code is valid
func (v *Validator) isValidBeitragsgruppe(code string) bool {
	// If we have a repo, check against the database
	if v.repo != nil {
		valid, _ := v.repo.IsBeitragsgruppValid(context.Background(), code)
		return valid
	}

	// Fallback to known codes
	return isKnownBeitragsgruppe(code)
}

// isKnownBeitragsgruppe checks against hardcoded known codes
func isKnownBeitragsgruppe(code string) bool {
	knownCodes := map[string]bool{
		"A1":  true, // Arbeiter vollversichert
		"A2":  true, // Arbeiter teilversichert
		"A3":  true, // Arbeiter geringfügig
		"A4":  true, // Arbeiter fallweise
		"D1":  true, // Angestellte vollversichert
		"D2":  true, // Angestellte teilversichert
		"D3":  true, // Angestellte geringfügig
		"D4":  true, // Angestellte fallweise
		"L1":  true, // Lehrlinge 1. Lehrjahr
		"L2":  true, // Lehrlinge 2. Lehrjahr
		"L3":  true, // Lehrlinge 3./4. Lehrjahr
		"N1":  true, // Freie Dienstnehmer vollversichert
		"N2":  true, // Freie Dienstnehmer geringfügig
		"F1":  true, // Freiwillig versichert
		"P1":  true, // Pensionist
		"GF":  true, // Geschäftsführer
		"PR":  true, // Praktikant
		"FER": true, // Ferialpraktikant
	}
	return knownCodes[strings.ToUpper(code)]
}

// ValidateSVNummer validates an Austrian social security number
func ValidateSVNummer(svNummer string) error {
	// Remove spaces
	svNummer = strings.ReplaceAll(svNummer, " ", "")

	// Check length
	if len(svNummer) != 10 {
		return fmt.Errorf("SV-Nummer muss 10 Stellen haben (hat %d)", len(svNummer))
	}

	// Check format: only digits
	if !regexp.MustCompile(`^\d{10}$`).MatchString(svNummer) {
		return fmt.Errorf("SV-Nummer darf nur Ziffern enthalten")
	}

	// Check serial number (positions 1-3, must not be 000)
	serialNumber := svNummer[0:3]
	if serialNumber == "000" {
		return fmt.Errorf("SV-Nummer ungültig: Laufnummer 000 nicht erlaubt")
	}

	// Validate check digit (position 4) using Modulo 11
	if !validateSVNummerCheckDigit(svNummer) {
		return fmt.Errorf("SV-Nummer ungültig: Prüfziffer falsch")
	}

	// Validate birthdate portion (positions 5-10: DDMMYY)
	birthDate := svNummer[4:10]
	if err := validateSVNummerBirthDate(birthDate); err != nil {
		return fmt.Errorf("SV-Nummer ungültig: %v", err)
	}

	return nil
}

// validateSVNummerCheckDigit validates the check digit using Modulo 11
func validateSVNummerCheckDigit(svNummer string) bool {
	// Weights for Modulo 11 calculation
	// Positions: 1  2  3  4  5  6  7  8  9  10
	// Weights:   3  7  9  -  5  8  4  2  1  6
	weights := []int{3, 7, 9, 0, 5, 8, 4, 2, 1, 6}

	sum := 0
	for i, w := range weights {
		if w == 0 {
			continue // Skip check digit position
		}
		digit, _ := strconv.Atoi(string(svNummer[i]))
		sum += digit * w
	}

	// Check digit is at position 4 (index 3)
	checkDigit, _ := strconv.Atoi(string(svNummer[3]))
	calculated := sum % 11

	return checkDigit == calculated
}

// validateSVNummerBirthDate validates the birthdate portion
func validateSVNummerBirthDate(birthDate string) error {
	// Format: DDMMYY
	day, _ := strconv.Atoi(birthDate[0:2])
	month, _ := strconv.Atoi(birthDate[2:4])
	// year is YY - we don't validate the year part strictly

	if month < 1 || month > 12 {
		return fmt.Errorf("ungültiger Monat im Geburtsdatum")
	}

	// Max days per month (simplified)
	maxDays := []int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if day < 1 || day > maxDays[month] {
		return fmt.Errorf("ungültiger Tag im Geburtsdatum")
	}

	return nil
}

// ValidateCreateRequest validates an mBGM create request
func ValidateCreateRequest(req *elda.MBGMCreateRequest) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Validate ELDA account
	if req.ELDAAccountID == uuid.Nil {
		result.addError("elda_account_id", "ELDA-Konto ID erforderlich", "", 0)
	}

	// Validate year
	currentYear := time.Now().Year()
	if req.Year < 2020 || req.Year > currentYear+1 {
		result.addError("year", fmt.Sprintf("Ungültiges Jahr: %d", req.Year), "", 0)
	}

	// Validate month
	if req.Month < 1 || req.Month > 12 {
		result.addError("month", fmt.Sprintf("Ungültiger Monat: %d", req.Month), "", 0)
	}

	// Validate positions
	if len(req.Positionen) == 0 {
		result.addError("positionen", "Mindestens eine Position erforderlich", "", 0)
	}

	svNummern := make(map[string]int)
	for i, posReq := range req.Positionen {
		validatePositionRequest(&posReq, i, result)

		if existing, ok := svNummern[posReq.SVNummer]; ok {
			result.addError("sv_nummer",
				fmt.Sprintf("SV-Nummer %s bereits in Position %d verwendet", posReq.SVNummer, existing+1),
				"", i)
		}
		svNummern[posReq.SVNummer] = i
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// validatePositionRequest validates a position create request
func validatePositionRequest(req *elda.MBGMPositionCreateRequest, index int, result *ValidationResult) {
	// SV-Nummer
	if req.SVNummer == "" {
		result.addError("sv_nummer", "SV-Nummer erforderlich", "", index)
	} else if err := ValidateSVNummer(req.SVNummer); err != nil {
		result.addError("sv_nummer", err.Error(), "", index)
	}

	// Name
	if req.Familienname == "" {
		result.addError("familienname", "Familienname erforderlich", "", index)
	}
	if req.Vorname == "" {
		result.addError("vorname", "Vorname erforderlich", "", index)
	}

	// Beitragsgruppe
	if req.Beitragsgruppe == "" {
		result.addError("beitragsgruppe", "Beitragsgruppe erforderlich", "", index)
	} else if !isKnownBeitragsgruppe(req.Beitragsgruppe) {
		result.addError("beitragsgruppe", fmt.Sprintf("Ungültige Beitragsgruppe: %s", req.Beitragsgruppe), "", index)
	}

	// Amounts
	if req.Beitragsgrundlage < 0 {
		result.addError("beitragsgrundlage", "Beitragsgrundlage darf nicht negativ sein", "", index)
	}
	if req.Sonderzahlung < 0 {
		result.addError("sonderzahlung", "Sonderzahlung darf nicht negativ sein", "", index)
	}

	// Wochenstunden
	if req.Wochenstunden != nil && (*req.Wochenstunden < 0 || *req.Wochenstunden > 60) {
		result.addError("wochenstunden", "Wochenstunden müssen zwischen 0 und 60 liegen", "", index)
	}

	// Geburtsdatum format
	if req.Geburtsdatum != "" {
		if _, err := time.Parse("2006-01-02", req.Geburtsdatum); err != nil {
			result.addError("geburtsdatum", "Ungültiges Datumsformat (erwartet: YYYY-MM-DD)", "", index)
		}
	}

	// Date range
	if req.VonDatum != "" && req.BisDatum != "" {
		von, errVon := time.Parse("2006-01-02", req.VonDatum)
		bis, errBis := time.Parse("2006-01-02", req.BisDatum)
		if errVon == nil && errBis == nil && bis.Before(von) {
			result.addError("bis_datum", "Bis-Datum muss nach Von-Datum liegen", "", index)
		}
	}
}

// addError adds an error to the validation result
func (r *ValidationResult) addError(field, message, positionID string, index int) {
	r.Errors = append(r.Errors, FieldValidationError{
		Field:      field,
		Message:    message,
		PositionID: positionID,
		Index:      index,
	})
}

// HasErrors returns true if there are validation errors
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// ErrorMessages returns all error messages as a slice
func (r *ValidationResult) ErrorMessages() []string {
	messages := make([]string, len(r.Errors))
	for i, fieldErr := range r.Errors {
		if fieldErr.Index > 0 {
			messages[i] = fmt.Sprintf("Position %d, %s: %s", fieldErr.Index+1, fieldErr.Field, fieldErr.Message)
		} else {
			messages[i] = fmt.Sprintf("%s: %s", fieldErr.Field, fieldErr.Message)
		}
	}
	return messages
}
