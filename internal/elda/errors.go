package elda

import (
	"errors"
	"fmt"
	"time"
)

// ELDA error codes and their meanings
const (
	// Validation errors (E0xx)
	ErrCodeSVNummerInvalid      = "E001"
	ErrCodeBeitragsgruppeFehlt  = "E002"
	ErrCodeZeitraumUngueltig    = "E003"
	ErrCodeBetragUngueltig      = "E004"
	ErrCodeDatumUngueltig       = "E005"
	ErrCodePflichtfeldFehlt     = "E006"
	ErrCodeFormatFehler         = "E007"
	ErrCodeDuplicate            = "E008"

	// Authentication errors (E1xx)
	ErrCodeZertifikatAbgelaufen = "E101"
	ErrCodeKeineBerechtigung    = "E102"
	ErrCodeZertifikatUngueltig  = "E103"
	ErrCodeSessionAbgelaufen    = "E104"

	// Business errors (E2xx)
	ErrCodeDNNichtGefunden      = "E201"
	ErrCodeDNBereitsAngemeldet  = "E202"
	ErrCodeDNNichtAngemeldet    = "E203"
	ErrCodeMeldungBereitsGesendet = "E204"
	ErrCodeKorrekturNichtMoeglich = "E205"

	// System errors (E9xx)
	ErrCodeServerFehler         = "E901"
	ErrCodeWartung              = "E902"
	ErrCodeTimeout              = "E903"

	// Warnings (W0xx) - submission accepted but with warnings
	WarnCodeGeringfuegig        = "W001"
	WarnCodeHoechstbeitrag      = "W002"
	WarnCodeRueckwirkend        = "W003"
)

// ELDAError represents an error from the ELDA system
type ELDAError struct {
	Code       string
	Message    string
	Field      string // optional: which field caused the error
	Details    string // optional: additional details
	Retryable  bool
	StatusCode int // HTTP status if applicable
}

func (e *ELDAError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("ELDA error %s (%s): %s", e.Code, e.Field, e.Message)
	}
	return fmt.Sprintf("ELDA error %s: %s", e.Code, e.Message)
}

// Is implements error comparison
func (e *ELDAError) Is(target error) bool {
	t, ok := target.(*ELDAError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// NewELDAError creates a new ELDA error
func NewELDAError(code, message string) *ELDAError {
	return &ELDAError{
		Code:      code,
		Message:   message,
		Retryable: isRetryableCode(code),
	}
}

// NewELDAErrorWithField creates a new ELDA error with field info
func NewELDAErrorWithField(code, message, field string) *ELDAError {
	return &ELDAError{
		Code:      code,
		Message:   message,
		Field:     field,
		Retryable: isRetryableCode(code),
	}
}

// Common ELDA errors as package-level variables
var (
	ErrSVNummerInvalid      = NewELDAError(ErrCodeSVNummerInvalid, "SV-Nummer ist ungültig")
	ErrBeitragsgruppeFehlt  = NewELDAError(ErrCodeBeitragsgruppeFehlt, "Beitragsgruppe fehlt oder ist ungültig")
	ErrZeitraumUngueltig    = NewELDAError(ErrCodeZeitraumUngueltig, "Zeitraum ist ungültig")
	ErrBetragUngueltig      = NewELDAError(ErrCodeBetragUngueltig, "Betrag ist ungültig")
	ErrDatumUngueltig       = NewELDAError(ErrCodeDatumUngueltig, "Datum ist ungültig")
	ErrPflichtfeldFehlt     = NewELDAError(ErrCodePflichtfeldFehlt, "Pflichtfeld fehlt")
	ErrFormatFehler         = NewELDAError(ErrCodeFormatFehler, "Formatfehler")
	ErrDuplicate            = NewELDAError(ErrCodeDuplicate, "Meldung bereits vorhanden")

	ErrZertifikatAbgelaufen = NewELDAError(ErrCodeZertifikatAbgelaufen, "ELDA-Zertifikat ist abgelaufen")
	ErrKeineBerechtigung    = NewELDAError(ErrCodeKeineBerechtigung, "Keine Berechtigung für Dienstgeber")
	ErrZertifikatUngueltig  = NewELDAError(ErrCodeZertifikatUngueltig, "ELDA-Zertifikat ist ungültig")
	ErrSessionAbgelaufen    = NewELDAError(ErrCodeSessionAbgelaufen, "ELDA-Session ist abgelaufen")

	ErrDNNichtGefunden      = NewELDAError(ErrCodeDNNichtGefunden, "Dienstnehmer nicht gefunden")
	ErrDNBereitsAngemeldet  = NewELDAError(ErrCodeDNBereitsAngemeldet, "Dienstnehmer bereits angemeldet")
	ErrDNNichtAngemeldet    = NewELDAError(ErrCodeDNNichtAngemeldet, "Dienstnehmer nicht angemeldet")
	ErrMeldungBereitsGesendet = NewELDAError(ErrCodeMeldungBereitsGesendet, "Meldung wurde bereits gesendet")
	ErrKorrekturNichtMoeglich = NewELDAError(ErrCodeKorrekturNichtMoeglich, "Korrektur nicht möglich")

	ErrServerFehler         = NewELDAError(ErrCodeServerFehler, "ELDA-Server nicht erreichbar")
	ErrWartung              = NewELDAError(ErrCodeWartung, "ELDA-System in Wartung")
	ErrTimeout              = NewELDAError(ErrCodeTimeout, "ELDA-Anfrage Zeitüberschreitung")
)

// isRetryableCode determines if an error code indicates a retryable condition
func isRetryableCode(code string) bool {
	switch code {
	case ErrCodeServerFehler, ErrCodeWartung, ErrCodeTimeout, ErrCodeSessionAbgelaufen:
		return true
	default:
		return false
	}
}

// ErrorCodeToError maps an ELDA error code to a Go error
func ErrorCodeToError(code string, message string) error {
	switch code {
	case ErrCodeSVNummerInvalid:
		return ErrSVNummerInvalid
	case ErrCodeBeitragsgruppeFehlt:
		return ErrBeitragsgruppeFehlt
	case ErrCodeZeitraumUngueltig:
		return ErrZeitraumUngueltig
	case ErrCodeBetragUngueltig:
		return ErrBetragUngueltig
	case ErrCodeDatumUngueltig:
		return ErrDatumUngueltig
	case ErrCodePflichtfeldFehlt:
		return ErrPflichtfeldFehlt
	case ErrCodeFormatFehler:
		return ErrFormatFehler
	case ErrCodeDuplicate:
		return ErrDuplicate
	case ErrCodeZertifikatAbgelaufen:
		return ErrZertifikatAbgelaufen
	case ErrCodeKeineBerechtigung:
		return ErrKeineBerechtigung
	case ErrCodeZertifikatUngueltig:
		return ErrZertifikatUngueltig
	case ErrCodeSessionAbgelaufen:
		return ErrSessionAbgelaufen
	case ErrCodeDNNichtGefunden:
		return ErrDNNichtGefunden
	case ErrCodeDNBereitsAngemeldet:
		return ErrDNBereitsAngemeldet
	case ErrCodeDNNichtAngemeldet:
		return ErrDNNichtAngemeldet
	case ErrCodeMeldungBereitsGesendet:
		return ErrMeldungBereitsGesendet
	case ErrCodeKorrekturNichtMoeglich:
		return ErrKorrekturNichtMoeglich
	case ErrCodeServerFehler:
		return ErrServerFehler
	case ErrCodeWartung:
		return ErrWartung
	case ErrCodeTimeout:
		return ErrTimeout
	default:
		return NewELDAError(code, message)
	}
}

// IsValidationError checks if the error is a validation error
func IsValidationError(err error) bool {
	var eldaErr *ELDAError
	if errors.As(err, &eldaErr) {
		return len(eldaErr.Code) >= 1 && eldaErr.Code[0] == 'E' &&
		       len(eldaErr.Code) >= 2 && eldaErr.Code[1] == '0'
	}
	return false
}

// IsAuthError checks if the error is an authentication error
func IsAuthError(err error) bool {
	var eldaErr *ELDAError
	if errors.As(err, &eldaErr) {
		return len(eldaErr.Code) >= 2 && eldaErr.Code[0] == 'E' && eldaErr.Code[1] == '1'
	}
	return false
}

// IsBusinessError checks if the error is a business logic error
func IsBusinessError(err error) bool {
	var eldaErr *ELDAError
	if errors.As(err, &eldaErr) {
		return len(eldaErr.Code) >= 2 && eldaErr.Code[0] == 'E' && eldaErr.Code[1] == '2'
	}
	return false
}

// IsSystemError checks if the error is a system error
func IsSystemError(err error) bool {
	var eldaErr *ELDAError
	if errors.As(err, &eldaErr) {
		return len(eldaErr.Code) >= 2 && eldaErr.Code[0] == 'E' && eldaErr.Code[1] == '9'
	}
	return false
}

// IsWarning checks if the error is actually a warning (submission accepted)
func IsWarning(err error) bool {
	var eldaErr *ELDAError
	if errors.As(err, &eldaErr) {
		return len(eldaErr.Code) >= 1 && eldaErr.Code[0] == 'W'
	}
	return false
}

// IsRetryable checks if the error indicates a condition that might succeed on retry
func IsRetryable(err error) bool {
	var eldaErr *ELDAError
	if errors.As(err, &eldaErr) {
		return eldaErr.Retryable
	}
	return false
}

// ValidationErrors collects multiple validation errors
type ValidationErrors struct {
	Errors []*ELDAError
}

func (v *ValidationErrors) Error() string {
	if len(v.Errors) == 0 {
		return "no validation errors"
	}
	return fmt.Sprintf("%d validation errors: %s", len(v.Errors), v.Errors[0].Error())
}

// Add adds a validation error
func (v *ValidationErrors) Add(err *ELDAError) {
	v.Errors = append(v.Errors, err)
}

// AddField adds a validation error for a specific field
func (v *ValidationErrors) AddField(code, message, field string) {
	v.Errors = append(v.Errors, NewELDAErrorWithField(code, message, field))
}

// HasErrors returns true if there are any errors
func (v *ValidationErrors) HasErrors() bool {
	return len(v.Errors) > 0
}

// ToError returns the ValidationErrors as an error, or nil if empty
func (v *ValidationErrors) ToError() error {
	if !v.HasErrors() {
		return nil
	}
	return v
}

// IsMaintenanceError checks if the error indicates ELDA is in maintenance
func IsMaintenanceError(err error) bool {
	var eldaErr *ELDAError
	if errors.As(err, &eldaErr) {
		return eldaErr.Code == ErrCodeWartung
	}
	// Also check for specific maintenance messages in generic errors
	if err != nil {
		msg := err.Error()
		return containsAny(msg, []string{"wartung", "maintenance", "nicht verfügbar", "unavailable"})
	}
	return false
}

// MaintenanceWindow represents ELDA scheduled maintenance
type MaintenanceWindow struct {
	Start       string `json:"start"`       // "HH:MM"
	End         string `json:"end"`         // "HH:MM"
	DaysOfWeek  []int  `json:"days"`        // 0=Sunday, 1=Monday, etc.
	Description string `json:"description"`
}

// DefaultMaintenanceWindows returns known ELDA maintenance windows
// ELDA typically has nightly maintenance windows
func DefaultMaintenanceWindows() []MaintenanceWindow {
	return []MaintenanceWindow{
		{
			Start:       "00:00",
			End:         "06:00",
			DaysOfWeek:  []int{0}, // Sunday
			Description: "Sonntagswartung",
		},
		{
			Start:       "02:00",
			End:         "04:00",
			DaysOfWeek:  []int{1, 2, 3, 4, 5, 6}, // Mon-Sat
			Description: "Nächtliche Wartung",
		},
	}
}

// IsInMaintenanceWindow checks if the current time is within a maintenance window
func IsInMaintenanceWindow(windows []MaintenanceWindow) bool {
	now := timeNow()
	weekday := int(now.Weekday())
	currentTime := now.Format("15:04")

	for _, w := range windows {
		for _, day := range w.DaysOfWeek {
			if day == weekday {
				if currentTime >= w.Start && currentTime < w.End {
					return true
				}
			}
		}
	}
	return false
}

// Helper to check if string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	lowered := toLower(s)
	for _, sub := range substrs {
		if indexOf(lowered, toLower(sub)) >= 0 {
			return true
		}
	}
	return false
}

// toLower converts string to lowercase (simple ASCII implementation)
func toLower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] = b[i] + 32
		}
	}
	return string(b)
}

// indexOf returns index of substring or -1
func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// timeNow is a variable to allow testing
var timeNow = func() time.Time { return time.Now() }
