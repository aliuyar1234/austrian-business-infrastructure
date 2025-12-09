package elda

import (
	"encoding/xml"
	"time"

	"github.com/google/uuid"
)

// Lohnzettel (L16) Status constants
type L16Status string

const (
	L16StatusDraft     L16Status = "draft"
	L16StatusValidated L16Status = "validated"
	L16StatusSubmitted L16Status = "submitted"
	L16StatusAccepted  L16Status = "accepted"
	L16StatusRejected  L16Status = "rejected"
)

// LohnzettelBatchStatus constants
type LohnzettelBatchStatus string

const (
	BatchStatusDraft          LohnzettelBatchStatus = "draft"
	BatchStatusSubmitting     LohnzettelBatchStatus = "submitting"
	BatchStatusCompleted      LohnzettelBatchStatus = "completed"
	BatchStatusPartialFailure LohnzettelBatchStatus = "partial_failure"
)

// Lohnzettel represents an annual tax form (L16)
type Lohnzettel struct {
	ID            uuid.UUID `json:"id" db:"id"`
	ELDAAccountID uuid.UUID `json:"elda_account_id" db:"elda_account_id"`
	Year          int       `json:"year" db:"year"`

	// Employee
	SVNummer     string     `json:"sv_nummer" db:"sv_nummer"`
	Familienname string     `json:"familienname" db:"familienname"`
	Vorname      string     `json:"vorname" db:"vorname"`
	Geburtsdatum *time.Time `json:"geburtsdatum,omitempty" db:"geburtsdatum"`

	// L16 Data (BMF spec fields)
	L16Data L16Data `json:"l16_data" db:"l16_data"`

	// Status
	Status          L16Status `json:"status" db:"status"`
	Protokollnummer string    `json:"protokollnummer,omitempty" db:"protokollnummer"`

	// Batch reference
	BatchID *uuid.UUID `json:"batch_id,omitempty" db:"batch_id"`

	// ELDA Response
	SubmittedAt  *time.Time `json:"submitted_at,omitempty" db:"submitted_at"`
	RequestXML   string     `json:"-" db:"request_xml"`
	ResponseXML  string     `json:"-" db:"response_xml"`
	ErrorMessage string     `json:"error_message,omitempty" db:"error_message"`
	ErrorCode    string     `json:"error_code,omitempty" db:"error_code"`

	// Correction
	IsBerichtigung bool       `json:"is_berichtigung" db:"is_berichtigung"`
	BerichtigtID   *uuid.UUID `json:"berichtigt_id,omitempty" db:"berichtigt_id"`

	// Audit
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// L16Data contains all L16 form fields (BMF E-Lohnzettel)
type L16Data struct {
	// Kennzahlen (field codes)
	KZ210 float64 `json:"kz210"` // Bruttobezüge gemäß §25 EStG
	KZ215 float64 `json:"kz215"` // Davon sonstige Bezüge (13./14.)
	KZ220 float64 `json:"kz220"` // Einbehaltene Lohnsteuer
	KZ230 float64 `json:"kz230"` // Pflichtbeiträge SV
	KZ243 float64 `json:"kz243"` // Pendlerpauschale
	KZ245 float64 `json:"kz245"` // Pendlereuro

	// Additional fields
	KZ226 float64 `json:"kz226"` // Werbungskosten
	KZ231 float64 `json:"kz231"` // Betriebliche Mitarbeitervorsorge
	KZ235 float64 `json:"kz235"` // Sonderausgaben
	KZ240 float64 `json:"kz240"` // Außergewöhnliche Belastungen
	KZ250 float64 `json:"kz250"` // Sachbezüge
	KZ260 float64 `json:"kz260"` // Steuerfreie Bezüge

	// Employment details
	BeschaeftigungVon  string `json:"beschaeftigung_von"`  // Start of employment YYYY-MM-DD
	BeschaeftigungBis  string `json:"beschaeftigung_bis"`  // End of employment YYYY-MM-DD
	ArbeitsTage        int    `json:"arbeits_tage"`        // Working days
	Tarifklasse        string `json:"tarifklasse"`         // Tax class
	AVAB               bool   `json:"avab"`                // Alleinverdienerabsetzbetrag
	AEAB               bool   `json:"aeab"`                // Alleinerzieherabsetzbetrag
	KinderAnzahl       int    `json:"kinder_anzahl"`       // Number of children for tax purposes
	PendlerPauschaleKM int    `json:"pendler_pauschale_km"` // Commuter allowance km
}

// LohnzettelBatch represents a batch of L16 submissions
type LohnzettelBatch struct {
	ID            uuid.UUID             `json:"id" db:"id"`
	ELDAAccountID uuid.UUID             `json:"elda_account_id" db:"elda_account_id"`
	Year          int                   `json:"year" db:"year"`

	// Statistics
	TotalLohnzettel int `json:"total_lohnzettel" db:"total_lohnzettel"`
	SubmittedCount  int `json:"submitted_count" db:"submitted_count"`
	AcceptedCount   int `json:"accepted_count" db:"accepted_count"`
	RejectedCount   int `json:"rejected_count" db:"rejected_count"`

	// Status
	Status      LohnzettelBatchStatus `json:"status" db:"status"`
	StartedAt   *time.Time            `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time            `json:"completed_at,omitempty" db:"completed_at"`

	// Audit
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`

	// Relationships (not stored in DB)
	Lohnzettel []*Lohnzettel `json:"lohnzettel,omitempty" db:"-"`
}

// LohnzettelCreateRequest is the request to create a new L16
type LohnzettelCreateRequest struct {
	ELDAAccountID uuid.UUID `json:"elda_account_id" validate:"required"`
	Year          int       `json:"year" validate:"required,min=2020,max=2100"`
	SVNummer      string    `json:"sv_nummer" validate:"required,len=10"`
	Familienname  string    `json:"familienname" validate:"required,max=100"`
	Vorname       string    `json:"vorname" validate:"required,max=100"`
	Geburtsdatum  string    `json:"geburtsdatum,omitempty"` // YYYY-MM-DD
	L16Data       L16Data   `json:"l16_data" validate:"required"`
}

// LohnzettelBatchCreateRequest is the request to create a batch
type LohnzettelBatchCreateRequest struct {
	ELDAAccountID uuid.UUID   `json:"elda_account_id" validate:"required"`
	Year          int         `json:"year" validate:"required,min=2020,max=2100"`
	LohnzettelIDs []uuid.UUID `json:"lohnzettel_ids" validate:"required,min=1"`
}

// XML types for ELDA submission

// L16Document is the XML document for L16 submission
type L16Document struct {
	XMLName      xml.Name       `xml:"Lohnzettel"`
	XMLNS        string         `xml:"xmlns,attr"`
	Jahr         int            `xml:"Jahr"`
	Arbeitnehmer L16Arbeitnehmer `xml:"Arbeitnehmer"`
	Bezuege      L16Bezuege      `xml:"Bezuege"`
	Zeiten       L16Zeiten       `xml:"Zeiten,omitempty"`
	Abzuege      L16Abzuege      `xml:"Abzuege,omitempty"`
}

// L16Arbeitnehmer contains employee info
type L16Arbeitnehmer struct {
	SVNummer     string `xml:"SVNummer"`
	Familienname string `xml:"Familienname"`
	Vorname      string `xml:"Vorname"`
	Geburtsdatum string `xml:"Geburtsdatum,omitempty"`
}

// L16Bezuege contains income fields
type L16Bezuege struct {
	KZ210 string `xml:"Kennzahl210"` // formatted as decimal
	KZ215 string `xml:"Kennzahl215,omitempty"`
	KZ220 string `xml:"Kennzahl220,omitempty"`
	KZ230 string `xml:"Kennzahl230,omitempty"`
	KZ243 string `xml:"Kennzahl243,omitempty"`
	KZ245 string `xml:"Kennzahl245,omitempty"`
	KZ250 string `xml:"Kennzahl250,omitempty"`
	KZ260 string `xml:"Kennzahl260,omitempty"`
}

// L16Zeiten contains employment period info
type L16Zeiten struct {
	BeschaeftigungVon string `xml:"BeschaeftigungVon,omitempty"`
	BeschaeftigungBis string `xml:"BeschaeftigungBis,omitempty"`
	ArbeitsTage       int    `xml:"ArbeitsTage,omitempty"`
}

// L16Abzuege contains deduction fields
type L16Abzuege struct {
	KZ226 string `xml:"Kennzahl226,omitempty"`
	KZ231 string `xml:"Kennzahl231,omitempty"`
	KZ235 string `xml:"Kennzahl235,omitempty"`
	KZ240 string `xml:"Kennzahl240,omitempty"`
}

// L16Response is the response from ELDA for L16 submission
type L16Response struct {
	XMLName        xml.Name `xml:"LohnzettelResponse"`
	Erfolg         bool     `xml:"Erfolg"`
	Protokollnummer string  `xml:"Protokollnummer,omitempty"`
	ErrorCode      string   `xml:"FehlerCode,omitempty"`
	ErrorMessage   string   `xml:"FehlerMeldung,omitempty"`
}

// L16BatchResult contains results for a batch submission
type L16BatchResult struct {
	Total    int             `json:"total"`
	Submitted int           `json:"submitted"`
	Accepted int            `json:"accepted"`
	Rejected int            `json:"rejected"`
	Results  []L16Result     `json:"results"`
}

// L16Result contains result for a single L16 in a batch
type L16Result struct {
	LohnzettelID    uuid.UUID `json:"lohnzettel_id"`
	Success         bool      `json:"success"`
	Protokollnummer string    `json:"protokollnummer,omitempty"`
	Error           string    `json:"error,omitempty"`
}

// L16 deadline constants
const (
	// L16 deadline: End of February of the following year
	L16DeadlineMonth = 2
	L16DeadlineDay   = 28 // Will be adjusted for leap years
)

// GetL16Deadline returns the deadline for submitting L16 for a given year
func GetL16Deadline(year int) time.Time {
	// Deadline is end of February of the following year
	nextYear := year + 1

	// Check for leap year
	deadline := time.Date(nextYear, time.February, 28, 23, 59, 59, 0, time.Local)
	if isLeapYear(nextYear) {
		deadline = time.Date(nextYear, time.February, 29, 23, 59, 59, 0, time.Local)
	}

	return deadline
}

// isLeapYear checks if a year is a leap year
func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// DaysUntilL16Deadline returns the number of days until L16 deadline
func DaysUntilL16Deadline(year int) int {
	deadline := GetL16Deadline(year)
	now := time.Now()
	if now.After(deadline) {
		return 0
	}
	return int(deadline.Sub(now).Hours() / 24)
}
