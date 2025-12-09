package elda

import (
	"encoding/xml"
	"time"

	"github.com/google/uuid"
)

// MeldungType represents the type of ELDA notification
type MeldungType string

const (
	MeldungTypeAnmeldung      MeldungType = "ANMELDUNG"
	MeldungTypeAbmeldung      MeldungType = "ABMELDUNG"
	MeldungTypeAenderung      MeldungType = "AENDERUNG"
	MeldungTypeKorrektur      MeldungType = "KORREKTUR"
)

// MeldungStatus represents the status of a meldung
type MeldungStatus string

const (
	MeldungStatusDraft      MeldungStatus = "draft"
	MeldungStatusValidated  MeldungStatus = "validated"
	MeldungStatusSubmitted  MeldungStatus = "submitted"
	MeldungStatusAccepted   MeldungStatus = "accepted"
	MeldungStatusRejected   MeldungStatus = "rejected"
)

// ELDAMeldung represents a generic ELDA notification (An-/Abmeldung/Änderung)
type ELDAMeldung struct {
	ID             uuid.UUID     `json:"id" db:"id"`
	ELDAAccountID  uuid.UUID     `json:"elda_account_id" db:"elda_account_id"`
	Type           MeldungType   `json:"type" db:"type"`
	Status         MeldungStatus `json:"status" db:"status"`

	// Employee data
	SVNummer     string     `json:"sv_nummer" db:"sv_nummer"`
	Vorname      string     `json:"vorname" db:"vorname"`
	Nachname     string     `json:"nachname" db:"nachname"`
	Geburtsdatum *time.Time `json:"geburtsdatum,omitempty" db:"geburtsdatum"`
	Geschlecht   string     `json:"geschlecht" db:"geschlecht"` // M, W, D

	// Anmeldung specific
	Eintrittsdatum *time.Time `json:"eintrittsdatum,omitempty" db:"eintrittsdatum"`

	// Abmeldung specific
	Austrittsdatum *time.Time         `json:"austrittsdatum,omitempty" db:"austrittsdatum"`
	AustrittGrund  ELDAAustrittGrund `json:"austritt_grund,omitempty" db:"austritt_grund"`

	// Extended employment data
	Beschaeftigung    *ExtendedBeschaeftigung `json:"beschaeftigung,omitempty" db:"beschaeftigung"`
	Arbeitszeit       *ExtendedArbeitszeit    `json:"arbeitszeit,omitempty" db:"arbeitszeit"`
	Entgelt           *ExtendedEntgelt        `json:"entgelt,omitempty" db:"entgelt"`
	Adresse           *DienstnehmerAdresse    `json:"adresse,omitempty" db:"adresse"`
	Bankverbindung    *Bankverbindung         `json:"bankverbindung,omitempty" db:"bankverbindung"`

	// Abmeldung settlement
	Abfertigung       *int64 `json:"abfertigung,omitempty" db:"abfertigung"`
	Urlaubsersatz     *int64 `json:"urlaubsersatz,omitempty" db:"urlaubsersatz"`
	URLTage           *int   `json:"url_tage,omitempty" db:"url_tage"` // Urlaubsersatzleistung Tage

	// Änderung specific
	AenderungArt      string     `json:"aenderung_art,omitempty" db:"aenderung_art"`
	AenderungDatum    *time.Time `json:"aenderung_datum,omitempty" db:"aenderung_datum"`
	OriginalMeldungID *uuid.UUID `json:"original_meldung_id,omitempty" db:"original_meldung_id"`

	// ELDA response
	Protokollnummer string     `json:"protokollnummer,omitempty" db:"protokollnummer"`
	SubmittedAt     *time.Time `json:"submitted_at,omitempty" db:"submitted_at"`
	RequestXML      string     `json:"-" db:"request_xml"`
	ResponseXML     string     `json:"-" db:"response_xml"`
	ErrorCode       string     `json:"error_code,omitempty" db:"error_code"`
	ErrorMessage    string     `json:"error_message,omitempty" db:"error_message"`

	// Audit
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// ExtendedBeschaeftigung includes all optional ELDA employment fields
type ExtendedBeschaeftigung struct {
	Art             string `json:"art"`                        // vollzeit, teilzeit, geringfuegig
	Taetigkeit      string `json:"taetigkeit,omitempty"`       // Job description/title
	KollektivCode   string `json:"kollektiv_code,omitempty"`   // Collective agreement code
	Einstufung      string `json:"einstufung,omitempty"`       // Grading/classification
	Verwendungsgruppe string `json:"verwendungsgruppe,omitempty"` // Usage group
	Dienstort       string `json:"dienstort,omitempty"`        // Work location
	Befristet       bool   `json:"befristet,omitempty"`        // Fixed-term contract
	BefristetBis    string `json:"befristet_bis,omitempty"`    // End date if fixed-term
	Lehrling        bool   `json:"lehrling,omitempty"`         // Apprentice
	Praktikant      bool   `json:"praktikant,omitempty"`       // Intern
	Werkvertrag     bool   `json:"werkvertrag,omitempty"`      // Service contract
	FreierDV        bool   `json:"freier_dv,omitempty"`        // Freelance service contract
	Beitragsgruppe  string `json:"beitragsgruppe,omitempty"`   // Contribution group
}

// ExtendedArbeitszeit includes all ELDA working time fields
type ExtendedArbeitszeit struct {
	WochenStunden     float64 `json:"wochen_stunden"`               // Weekly hours
	TageProWoche      int     `json:"tage_pro_woche,omitempty"`     // Days per week
	ArbeitszeitCode   string  `json:"arbeitszeit_code,omitempty"`   // Working time code
	Schichtarbeit     bool    `json:"schichtarbeit,omitempty"`      // Shift work
	Nachtarbeit       bool    `json:"nachtarbeit,omitempty"`        // Night work
	Wechselschicht    bool    `json:"wechselschicht,omitempty"`     // Rotating shift
	KollektivStunden  float64 `json:"kollektiv_stunden,omitempty"`  // Collective agreement hours
}

// ExtendedEntgelt includes all ELDA salary fields
type ExtendedEntgelt struct {
	BruttoMonatlich   int64  `json:"brutto_monatlich"`             // Monthly gross in cents
	NettoMonatlich    int64  `json:"netto_monatlich,omitempty"`    // Monthly net in cents
	Sonderzahlungen   int64  `json:"sonderzahlungen,omitempty"`    // Annual special payments in cents
	Ueberstunden      int64  `json:"ueberstunden,omitempty"`       // Overtime in cents
	Zulagen           int64  `json:"zulagen,omitempty"`            // Allowances in cents
	Praemien          int64  `json:"praemien,omitempty"`           // Bonuses in cents
	Sachbezuege       int64  `json:"sachbezuege,omitempty"`        // Benefits in kind in cents
	EntgeltArt        string `json:"entgelt_art,omitempty"`        // Salary type (monatlich, stuendlich)
	StundenSatz       int64  `json:"stunden_satz,omitempty"`       // Hourly rate in cents
}

// DienstnehmerAdresse represents employee address for ELDA
type DienstnehmerAdresse struct {
	Strasse     string `json:"strasse"`
	Hausnummer  string `json:"hausnummer,omitempty"`
	Stiege      string `json:"stiege,omitempty"`
	Tuer        string `json:"tuer,omitempty"`
	PLZ         string `json:"plz"`
	Ort         string `json:"ort"`
	Land        string `json:"land,omitempty"` // Default: AT
}

// Bankverbindung represents bank account details
type Bankverbindung struct {
	IBAN          string `json:"iban"`
	BIC           string `json:"bic,omitempty"`
	Kontoinhaber  string `json:"kontoinhaber,omitempty"`
}

// MeldungCreateRequest is the request to create a new ELDA meldung
type MeldungCreateRequest struct {
	ELDAAccountID     uuid.UUID               `json:"elda_account_id" validate:"required"`
	Type              MeldungType             `json:"type" validate:"required"`
	SVNummer          string                  `json:"sv_nummer" validate:"required,len=10"`
	Vorname           string                  `json:"vorname" validate:"required,max=100"`
	Nachname          string                  `json:"nachname" validate:"required,max=100"`
	Geburtsdatum      string                  `json:"geburtsdatum,omitempty"` // YYYY-MM-DD
	Geschlecht        string                  `json:"geschlecht,omitempty"`   // M, W, D

	// Anmeldung
	Eintrittsdatum    string                  `json:"eintrittsdatum,omitempty"` // YYYY-MM-DD
	Beschaeftigung    *ExtendedBeschaeftigung `json:"beschaeftigung,omitempty"`
	Arbeitszeit       *ExtendedArbeitszeit    `json:"arbeitszeit,omitempty"`
	Entgelt           *ExtendedEntgelt        `json:"entgelt,omitempty"`
	Adresse           *DienstnehmerAdresse    `json:"adresse,omitempty"`
	Bankverbindung    *Bankverbindung         `json:"bankverbindung,omitempty"`

	// Abmeldung
	Austrittsdatum    string                  `json:"austrittsdatum,omitempty"` // YYYY-MM-DD
	AustrittGrund     ELDAAustrittGrund       `json:"austritt_grund,omitempty"`
	Abfertigung       *int64                  `json:"abfertigung,omitempty"`
	Urlaubsersatz     *int64                  `json:"urlaubsersatz,omitempty"`
	URLTage           *int                    `json:"url_tage,omitempty"`

	// Änderung
	AenderungArt      string                  `json:"aenderung_art,omitempty"`
	AenderungDatum    string                  `json:"aenderung_datum,omitempty"`
	OriginalMeldungID *uuid.UUID              `json:"original_meldung_id,omitempty"`
}

// Extended XML types for ELDA submission

// ExtendedAnmeldungDocument is the full ELDA Anmeldung XML
type ExtendedAnmeldungDocument struct {
	XMLName           xml.Name                 `xml:"Anmeldung"`
	XMLNS             string                   `xml:"xmlns,attr"`
	Kopf              ELDAKopf                 `xml:"Kopf"`
	SVNummer          string                   `xml:"SVNummer"`
	Vorname           string                   `xml:"Vorname"`
	Nachname          string                   `xml:"Nachname"`
	Geburtsdatum      string                   `xml:"Geburtsdatum,omitempty"`
	Geschlecht        string                   `xml:"Geschlecht,omitempty"`
	Eintrittsdatum    string                   `xml:"Eintrittsdatum"`
	Beschaeftigung    *XMLBeschaeftigung       `xml:"Beschaeftigung,omitempty"`
	Arbeitszeit       *XMLArbeitszeit          `xml:"Arbeitszeit,omitempty"`
	Entgelt           *XMLEntgelt              `xml:"Entgelt,omitempty"`
	Adresse           *XMLAdresse              `xml:"Adresse,omitempty"`
	Bankverbindung    *XMLBankverbindung       `xml:"Bankverbindung,omitempty"`
}

// ExtendedAbmeldungDocument is the full ELDA Abmeldung XML
type ExtendedAbmeldungDocument struct {
	XMLName           xml.Name                 `xml:"Abmeldung"`
	XMLNS             string                   `xml:"xmlns,attr"`
	Kopf              ELDAKopf                 `xml:"Kopf"`
	SVNummer          string                   `xml:"SVNummer"`
	Austrittsdatum    string                   `xml:"Austrittsdatum"`
	Grund             ELDAAustrittGrund        `xml:"Grund"`
	Abfertigung       *int64                   `xml:"Abfertigung,omitempty"`
	Urlaubsersatz     *int64                   `xml:"Urlaubsersatz,omitempty"`
	URLTage           *int                     `xml:"URLTage,omitempty"`
}

// XML helper types
type XMLBeschaeftigung struct {
	Art              string `xml:"Art"`
	Taetigkeit       string `xml:"Taetigkeit,omitempty"`
	KollektivCode    string `xml:"KollektivCode,omitempty"`
	Einstufung       string `xml:"Einstufung,omitempty"`
	Verwendungsgruppe string `xml:"Verwendungsgruppe,omitempty"`
	Dienstort        string `xml:"Dienstort,omitempty"`
	Befristet        string `xml:"Befristet,omitempty"`
	BefristetBis     string `xml:"BefristetBis,omitempty"`
	Beitragsgruppe   string `xml:"Beitragsgruppe,omitempty"`
}

type XMLArbeitszeit struct {
	WochenStunden    string `xml:"WochenStunden"`
	TageProWoche     int    `xml:"TageProWoche,omitempty"`
	ArbeitszeitCode  string `xml:"ArbeitszeitCode,omitempty"`
	Schichtarbeit    string `xml:"Schichtarbeit,omitempty"`
	KollektivStunden string `xml:"KollektivStunden,omitempty"`
}

type XMLEntgelt struct {
	BruttoMonatlich  string `xml:"BruttoMonatlich"`
	NettoMonatlich   string `xml:"NettoMonatlich,omitempty"`
	Sonderzahlungen  string `xml:"Sonderzahlungen,omitempty"`
	Ueberstunden     string `xml:"Ueberstunden,omitempty"`
	Zulagen          string `xml:"Zulagen,omitempty"`
	Sachbezuege      string `xml:"Sachbezuege,omitempty"`
	EntgeltArt       string `xml:"EntgeltArt,omitempty"`
}

type XMLAdresse struct {
	Strasse     string `xml:"Strasse"`
	Hausnummer  string `xml:"Hausnummer,omitempty"`
	Stiege      string `xml:"Stiege,omitempty"`
	Tuer        string `xml:"Tuer,omitempty"`
	PLZ         string `xml:"PLZ"`
	Ort         string `xml:"Ort"`
	Land        string `xml:"Land,omitempty"`
}

type XMLBankverbindung struct {
	IBAN         string `xml:"IBAN"`
	BIC          string `xml:"BIC,omitempty"`
	Kontoinhaber string `xml:"Kontoinhaber,omitempty"`
}

// MeldungResponse is the response from ELDA for a meldung
type MeldungResponse struct {
	XMLName         xml.Name `xml:"MeldungResponse"`
	Erfolg          bool     `xml:"Erfolg"`
	Protokollnummer string   `xml:"Protokollnummer,omitempty"`
	ErrorCode       string   `xml:"FehlerCode,omitempty"`
	ErrorMessage    string   `xml:"FehlerMeldung,omitempty"`
	Warnungen       []string `xml:"Warnungen>Warnung,omitempty"`
}
