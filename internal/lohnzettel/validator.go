package lohnzettel

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/elda"
	"austrian-business-infrastructure/internal/mbgm"
)

// ValidationResult contains the result of validating a Lohnzettel
type ValidationResult struct {
	Valid    bool                    `json:"valid"`
	Errors   []FieldValidationError  `json:"errors,omitempty"`
	Warnings []string                `json:"warnings,omitempty"`
}

// FieldValidationError represents a single validation error
type FieldValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Validator validates L16 Lohnzettel data
type Validator struct{}

// NewValidator creates a new L16 validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateLohnzettel validates an L16 Lohnzettel
func (v *Validator) ValidateLohnzettel(l *elda.Lohnzettel) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Validate header fields
	v.validateHeader(l, result)

	// Validate L16 data fields
	v.validateL16Data(&l.L16Data, result)

	// Add deadline warning
	deadline := elda.GetL16Deadline(l.Year)
	daysUntil := elda.DaysUntilL16Deadline(l.Year)
	if daysUntil == 0 && time.Now().After(deadline) {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("L16-Frist bereits abgelaufen (war am %s)", deadline.Format("02.01.2006")))
	} else if daysUntil <= 7 && daysUntil > 0 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("L16-Frist endet in %d Tagen (%s)", daysUntil, deadline.Format("02.01.2006")))
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// validateHeader validates the Lohnzettel header fields
func (v *Validator) validateHeader(l *elda.Lohnzettel, result *ValidationResult) {
	// Validate year
	currentYear := time.Now().Year()
	if l.Year < 2020 || l.Year > currentYear {
		result.addError("year", fmt.Sprintf("Ungültiges Jahr: %d (erlaubt: 2020-%d)", l.Year, currentYear))
	}

	// Validate ELDA account ID
	if l.ELDAAccountID == uuid.Nil {
		result.addError("elda_account_id", "ELDA-Konto ID erforderlich")
	}

	// Validate SV-Nummer
	if l.SVNummer == "" {
		result.addError("sv_nummer", "SV-Nummer erforderlich")
	} else if err := mbgm.ValidateSVNummer(l.SVNummer); err != nil {
		result.addError("sv_nummer", err.Error())
	}

	// Validate name
	if l.Familienname == "" {
		result.addError("familienname", "Familienname erforderlich")
	}
	if l.Vorname == "" {
		result.addError("vorname", "Vorname erforderlich")
	}
}

// validateL16Data validates the L16 data fields
func (v *Validator) validateL16Data(data *elda.L16Data, result *ValidationResult) {
	// KZ210 - Bruttobezüge (required, must be positive)
	if data.KZ210 <= 0 {
		result.addError("kz210", "Bruttobezüge (KZ210) müssen größer als 0 sein")
	}

	// KZ215 - Sonstige Bezüge (cannot exceed KZ210)
	if data.KZ215 > data.KZ210 {
		result.addError("kz215", "Sonstige Bezüge (KZ215) dürfen Bruttobezüge (KZ210) nicht übersteigen")
	}
	if data.KZ215 < 0 {
		result.addError("kz215", "Sonstige Bezüge (KZ215) dürfen nicht negativ sein")
	}

	// KZ220 - Einbehaltene Lohnsteuer (cannot be negative)
	if data.KZ220 < 0 {
		result.addError("kz220", "Lohnsteuer (KZ220) darf nicht negativ sein")
	}

	// KZ230 - Pflichtbeiträge SV (cannot be negative)
	if data.KZ230 < 0 {
		result.addError("kz230", "SV-Beiträge (KZ230) dürfen nicht negativ sein")
	}

	// KZ243 - Pendlerpauschale (cannot be negative)
	if data.KZ243 < 0 {
		result.addError("kz243", "Pendlerpauschale (KZ243) darf nicht negativ sein")
	}

	// KZ245 - Pendlereuro (cannot be negative)
	if data.KZ245 < 0 {
		result.addError("kz245", "Pendlereuro (KZ245) darf nicht negativ sein")
	}

	// KZ250 - Sachbezüge (cannot be negative)
	if data.KZ250 < 0 {
		result.addError("kz250", "Sachbezüge (KZ250) dürfen nicht negativ sein")
	}

	// KZ260 - Steuerfreie Bezüge (cannot be negative)
	if data.KZ260 < 0 {
		result.addError("kz260", "Steuerfreie Bezüge (KZ260) dürfen nicht negativ sein")
	}

	// Employment dates
	if data.BeschaeftigungVon != "" {
		if _, err := time.Parse("2006-01-02", data.BeschaeftigungVon); err != nil {
			result.addError("beschaeftigung_von", "Ungültiges Datumsformat (erwartet: YYYY-MM-DD)")
		}
	}
	if data.BeschaeftigungBis != "" {
		if _, err := time.Parse("2006-01-02", data.BeschaeftigungBis); err != nil {
			result.addError("beschaeftigung_bis", "Ungültiges Datumsformat (erwartet: YYYY-MM-DD)")
		}
	}

	// Validate date range
	if data.BeschaeftigungVon != "" && data.BeschaeftigungBis != "" {
		von, errVon := time.Parse("2006-01-02", data.BeschaeftigungVon)
		bis, errBis := time.Parse("2006-01-02", data.BeschaeftigungBis)
		if errVon == nil && errBis == nil && bis.Before(von) {
			result.addError("beschaeftigung_bis", "Bis-Datum muss nach Von-Datum liegen")
		}
	}

	// ArbeitsTage
	if data.ArbeitsTage < 0 || data.ArbeitsTage > 366 {
		result.addError("arbeits_tage", "Arbeitstage müssen zwischen 0 und 366 liegen")
	}

	// KinderAnzahl
	if data.KinderAnzahl < 0 {
		result.addError("kinder_anzahl", "Kinderanzahl darf nicht negativ sein")
	}

	// PendlerPauschaleKM
	if data.PendlerPauschaleKM < 0 {
		result.addError("pendler_pauschale_km", "Pendlerpauschale km darf nicht negativ sein")
	}

	// Consistency checks
	if data.KZ243 > 0 && data.PendlerPauschaleKM == 0 {
		result.Warnings = append(result.Warnings, "Pendlerpauschale (KZ243) ohne km-Angabe")
	}

	if (data.AVAB || data.AEAB) && data.KinderAnzahl == 0 {
		result.Warnings = append(result.Warnings, "AVAB/AEAB ohne Kinderanzahl")
	}
}

// ValidateCreateRequest validates a Lohnzettel create request
func ValidateCreateRequest(req *elda.LohnzettelCreateRequest) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Validate ELDA account
	if req.ELDAAccountID == uuid.Nil {
		result.addError("elda_account_id", "ELDA-Konto ID erforderlich")
	}

	// Validate year
	currentYear := time.Now().Year()
	if req.Year < 2020 || req.Year > currentYear {
		result.addError("year", fmt.Sprintf("Ungültiges Jahr: %d", req.Year))
	}

	// Validate SV-Nummer
	if req.SVNummer == "" {
		result.addError("sv_nummer", "SV-Nummer erforderlich")
	} else if err := mbgm.ValidateSVNummer(req.SVNummer); err != nil {
		result.addError("sv_nummer", err.Error())
	}

	// Validate name
	if req.Familienname == "" {
		result.addError("familienname", "Familienname erforderlich")
	}
	if req.Vorname == "" {
		result.addError("vorname", "Vorname erforderlich")
	}

	// Validate L16 data
	v := &Validator{}
	v.validateL16Data(&req.L16Data, result)

	result.Valid = len(result.Errors) == 0
	return result
}

// ValidateBatchCreateRequest validates a batch create request
func ValidateBatchCreateRequest(req *elda.LohnzettelBatchCreateRequest) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if req.ELDAAccountID == uuid.Nil {
		result.addError("elda_account_id", "ELDA-Konto ID erforderlich")
	}

	currentYear := time.Now().Year()
	if req.Year < 2020 || req.Year > currentYear {
		result.addError("year", fmt.Sprintf("Ungültiges Jahr: %d", req.Year))
	}

	if len(req.LohnzettelIDs) == 0 {
		result.addError("lohnzettel_ids", "Mindestens ein Lohnzettel erforderlich")
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// addError adds an error to the validation result
func (r *ValidationResult) addError(field, message string) {
	r.Errors = append(r.Errors, FieldValidationError{
		Field:   field,
		Message: message,
	})
}

// HasErrors returns true if there are validation errors
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// ErrorMessages returns all error messages as a slice
func (r *ValidationResult) ErrorMessages() []string {
	messages := make([]string, len(r.Errors))
	for i, err := range r.Errors {
		messages[i] = fmt.Sprintf("%s: %s", err.Field, err.Message)
	}
	return messages
}
