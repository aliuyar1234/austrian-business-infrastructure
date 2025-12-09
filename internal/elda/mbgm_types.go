package elda

import (
	"encoding/xml"
	"time"

	"github.com/google/uuid"
)

// mBGM Status constants
type MBGMStatus string

const (
	MBGMStatusDraft     MBGMStatus = "draft"
	MBGMStatusValidated MBGMStatus = "validated"
	MBGMStatusSubmitted MBGMStatus = "submitted"
	MBGMStatusAccepted  MBGMStatus = "accepted"
	MBGMStatusRejected  MBGMStatus = "rejected"
)

// MBGM represents a monthly contribution report
type MBGM struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	ELDAAccountID   uuid.UUID  `json:"elda_account_id" db:"elda_account_id"`
	Year            int        `json:"year" db:"year"`
	Month           int        `json:"month" db:"month"`
	Status          MBGMStatus `json:"status" db:"status"`
	Protokollnummer string     `json:"protokollnummer,omitempty" db:"protokollnummer"`

	// Statistics
	TotalDienstnehmer      int     `json:"total_dienstnehmer" db:"total_dienstnehmer"`
	TotalBeitragsgrundlage float64 `json:"total_beitragsgrundlage" db:"total_beitragsgrundlage"`

	// ELDA Response
	SubmittedAt        *time.Time `json:"submitted_at,omitempty" db:"submitted_at"`
	ResponseReceivedAt *time.Time `json:"response_received_at,omitempty" db:"response_received_at"`
	RequestXML         string     `json:"-" db:"request_xml"`
	ResponseXML        string     `json:"-" db:"response_xml"`
	ErrorMessage       string     `json:"error_message,omitempty" db:"error_message"`
	ErrorCode          string     `json:"error_code,omitempty" db:"error_code"`

	// Correction
	IsCorrection bool       `json:"is_correction" db:"is_correction"`
	CorrectsID   *uuid.UUID `json:"corrects_id,omitempty" db:"corrects_id"`

	// Audit
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`

	// Relationships (not stored in DB)
	Positionen []*MBGMPosition `json:"positionen,omitempty" db:"-"`
}

// MBGMPosition represents a single employee entry in mBGM
type MBGMPosition struct {
	ID     uuid.UUID `json:"id" db:"id"`
	MBGMID uuid.UUID `json:"mbgm_id" db:"mbgm_id"`

	// Employee
	SVNummer     string     `json:"sv_nummer" db:"sv_nummer"`
	Familienname string     `json:"familienname" db:"familienname"`
	Vorname      string     `json:"vorname" db:"vorname"`
	Geburtsdatum *time.Time `json:"geburtsdatum,omitempty" db:"geburtsdatum"`

	// Employment
	Beitragsgruppe      string  `json:"beitragsgruppe" db:"beitragsgruppe"`
	Beitragsgrundlage   float64 `json:"beitragsgrundlage" db:"beitragsgrundlage"`
	Sonderzahlung       float64 `json:"sonderzahlung" db:"sonderzahlung"`

	// Period (if different from month)
	VonDatum *time.Time `json:"von_datum,omitempty" db:"von_datum"`
	BisDatum *time.Time `json:"bis_datum,omitempty" db:"bis_datum"`

	// Hours (optional)
	Wochenstunden *float64 `json:"wochenstunden,omitempty" db:"wochenstunden"`

	// Validation
	IsValid          bool     `json:"is_valid" db:"is_valid"`
	ValidationErrors []string `json:"validation_errors,omitempty" db:"validation_errors"`

	PositionIndex int       `json:"position_index" db:"position_index"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// MBGMCreateRequest is the request to create a new mBGM
type MBGMCreateRequest struct {
	ELDAAccountID uuid.UUID                  `json:"elda_account_id" validate:"required"`
	Year          int                        `json:"year" validate:"required,min=2020,max=2100"`
	Month         int                        `json:"month" validate:"required,min=1,max=12"`
	Positionen    []MBGMPositionCreateRequest `json:"positionen" validate:"required,min=1"`
}

// MBGMPositionCreateRequest is the request to create a position
type MBGMPositionCreateRequest struct {
	SVNummer          string   `json:"sv_nummer" validate:"required,len=10"`
	Familienname      string   `json:"familienname" validate:"required,max=100"`
	Vorname           string   `json:"vorname" validate:"required,max=100"`
	Geburtsdatum      string   `json:"geburtsdatum,omitempty"` // YYYY-MM-DD
	Beitragsgruppe    string   `json:"beitragsgruppe" validate:"required,max=10"`
	Beitragsgrundlage float64  `json:"beitragsgrundlage" validate:"required,min=0"`
	Sonderzahlung     float64  `json:"sonderzahlung"`
	VonDatum          string   `json:"von_datum,omitempty"`
	BisDatum          string   `json:"bis_datum,omitempty"`
	Wochenstunden     *float64 `json:"wochenstunden,omitempty"`
}

// XML types for ELDA submission

// MBGMDocument is the XML document for mBGM submission
type MBGMDocument struct {
	XMLName    xml.Name     `xml:"mBGM"`
	XMLNS      string       `xml:"xmlns,attr"`
	Kopf       MBGMKopf     `xml:"Kopf"`
	Positionen []MBGMXMLPos `xml:"Positionen>Position"`
}

// MBGMKopf is the header section
type MBGMKopf struct {
	DienstgeberNummer string `xml:"DienstgeberNummer"`
	Jahr              int    `xml:"Meldezeitraum>Jahr"`
	Monat             int    `xml:"Meldezeitraum>Monat"`
	Erstellungsdatum  string `xml:"Erstellungsdatum"`
	IsKorrektur       bool   `xml:"IsKorrektur,omitempty"`
}

// MBGMXMLPos is a single position in XML format
type MBGMXMLPos struct {
	SVNummer          string  `xml:"SVNummer"`
	Familienname      string  `xml:"Familienname"`
	Vorname           string  `xml:"Vorname"`
	Geburtsdatum      string  `xml:"Geburtsdatum,omitempty"`
	Beitragsgruppe    string  `xml:"Beitragsgruppe"`
	Beitragsgrundlage string  `xml:"Beitragsgrundlage"` // formatted as decimal
	Sonderzahlung     string  `xml:"Sonderzahlung,omitempty"`
	VonDatum          string  `xml:"BeitragszeitraumVon,omitempty"`
	BisDatum          string  `xml:"BeitragszeitraumBis,omitempty"`
	Wochenstunden     *string `xml:"Wochenstunden,omitempty"`
}

// MBGMResponse is the response from ELDA for mBGM submission
type MBGMResponse struct {
	XMLName        xml.Name `xml:"MBGMResponse"`
	Erfolg         bool     `xml:"Erfolg"`
	Protokollnummer string  `xml:"Protokollnummer,omitempty"`
	ErrorCode      string   `xml:"FehlerCode,omitempty"`
	ErrorMessage   string   `xml:"FehlerMeldung,omitempty"`
	Warnungen      []string `xml:"Warnungen>Warnung,omitempty"`
}

// Beitragsgruppe represents an SV contribution group
type Beitragsgruppe struct {
	Code        string     `json:"code" db:"code"`
	Bezeichnung string     `json:"bezeichnung" db:"bezeichnung"`
	Beschreibung string    `json:"beschreibung,omitempty" db:"beschreibung"`
	ValidFrom   time.Time  `json:"valid_from" db:"valid_from"`
	ValidUntil  *time.Time `json:"valid_until,omitempty" db:"valid_until"`
	IsActive    bool       `json:"is_active" db:"is_active"`
}

// Kollektivvertrag represents a collective agreement
type Kollektivvertrag struct {
	Code        string    `json:"code" db:"code"`
	Bezeichnung string    `json:"bezeichnung" db:"bezeichnung"`
	Branche     string    `json:"branche,omitempty" db:"branche"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// 2025 Limits (update yearly)
const (
	// Geringfügigkeitsgrenze 2025
	GeringfuegigkeitsGrenze2025 = 518.44

	// Höchstbeitragsgrundlage 2025
	HoechstbeitragsGrundlage2025 = 6060.00

	// mBGM deadline: 15th of following month
	MBGMDeadlineDay = 15
)

// GetMBGMDeadline returns the deadline for submitting mBGM
func GetMBGMDeadline(year, month int) time.Time {
	// Deadline is 15th of the following month
	nextMonth := month + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}
	return time.Date(nextYear, time.Month(nextMonth), MBGMDeadlineDay, 23, 59, 59, 0, time.Local)
}

// IsGeringfuegig checks if the amount is below the Geringfügigkeitsgrenze
func IsGeringfuegig(monthlyAmount float64, year int) bool {
	// TODO: load from config based on year
	return monthlyAmount <= GeringfuegigkeitsGrenze2025
}

// ExceedsHoechstbeitrag checks if the amount exceeds the Höchstbeitragsgrundlage
func ExceedsHoechstbeitrag(monthlyAmount float64, year int) bool {
	// TODO: load from config based on year
	return monthlyAmount > HoechstbeitragsGrundlage2025
}
