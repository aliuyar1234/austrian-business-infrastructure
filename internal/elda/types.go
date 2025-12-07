package elda

import (
	"encoding/xml"
	"time"
)

// ELDA status constants
type ELDAMeldungStatus string

const (
	ELDAStatusDraft     ELDAMeldungStatus = "draft"
	ELDAStatusSubmitted ELDAMeldungStatus = "submitted"
	ELDAStatusProcessed ELDAMeldungStatus = "processed"
	ELDAStatusRejected  ELDAMeldungStatus = "rejected"
)

// Employment types
const (
	BeschaeftigungVollzeit    = "vollzeit"
	BeschaeftigungTeilzeit    = "teilzeit"
	BeschaeftigungGeringfuegig = "geringfuegig"
)

// Exit reason constants
type ELDAAustrittGrund string

const (
	ELDAGrundKuendigung     ELDAAustrittGrund = "K"  // Kündigung
	ELDAGrundEinvernehmlich ELDAAustrittGrund = "E"  // Einvernehmlich
	ELDAGrundEntlassung     ELDAAustrittGrund = "EN" // Entlassung
	ELDAGrundAustritt       ELDAAustrittGrund = "A"  // Vorzeitiger Austritt
	ELDAGrundBefristet      ELDAAustrittGrund = "B"  // Befristung
)

// ELDAKopf represents the header section of an ELDA message
type ELDAKopf struct {
	XMLName       xml.Name `xml:"Kopf"`
	DienstgeberNr string   `xml:"DienstgeberNr"`
	Datum         string   `xml:"Datum"` // Format: YYYY-MM-DD
	MeldungsArt   string   `xml:"MeldungsArt"`
}

// ELDAAnmeldung represents an employee registration
type ELDAAnmeldung struct {
	// Employee data
	SVNummer     string    `json:"sv_nummer" xml:"SVNummer"`
	Vorname      string    `json:"vorname" xml:"Vorname"`
	Nachname     string    `json:"nachname" xml:"Nachname"`
	Geburtsdatum time.Time `json:"geburtsdatum" xml:"-"`
	Geschlecht   string    `json:"geschlecht" xml:"Geschlecht"` // "M" or "W"

	// Employment data
	Eintrittsdatum time.Time          `json:"eintrittsdatum" xml:"-"`
	Beschaeftigung ELDABeschaeftigung `json:"beschaeftigung" xml:"Beschaeftigung"`
	Arbeitszeit    ELDAArbeitszeit    `json:"arbeitszeit" xml:"Arbeitszeit"`
	Entgelt        ELDAEntgelt        `json:"entgelt" xml:"Entgelt"`

	// Employer reference
	DienstgeberNr string `json:"dienstgeber_nr" xml:"DienstgeberNr"`

	// Metadata
	Status    ELDAMeldungStatus `json:"status"`
	Reference string            `json:"reference,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// GeburtsdatumString returns the birth date as a string for XML
func (a *ELDAAnmeldung) GeburtsdatumString() string {
	return a.Geburtsdatum.Format("2006-01-02")
}

// EintrittsdatumString returns the entry date as a string for XML
func (a *ELDAAnmeldung) EintrittsdatumString() string {
	return a.Eintrittsdatum.Format("2006-01-02")
}

// ELDABeschaeftigung describes the employment details
type ELDABeschaeftigung struct {
	Art        string `json:"art" xml:"Art"`              // vollzeit, teilzeit, geringfuegig
	Taetigkeit string `json:"taetigkeit" xml:"Taetigkeit"` // Job description
	Kollektiv  string `json:"kollektiv" xml:"Kollektiv"`   // Collective agreement code
	Einstufung string `json:"einstufung" xml:"Einstufung"` // Grading
}

// ELDAArbeitszeit describes working hours
type ELDAArbeitszeit struct {
	Stunden float64 `json:"stunden" xml:"Stunden"` // Weekly hours
	Tage    int     `json:"tage" xml:"Tage"`       // Days per week
}

// ELDAEntgelt describes salary information
type ELDAEntgelt struct {
	Brutto     int64 `json:"brutto" xml:"Brutto"`         // Monthly gross in cents
	Netto      int64 `json:"netto,omitempty" xml:"Netto"` // Monthly net in cents (optional)
	Sonderzahl int64 `json:"sonderzahl" xml:"Sonderzahl"` // Special payments per year in cents
}

// ELDAAbmeldung represents an employee deregistration
type ELDAAbmeldung struct {
	SVNummer       string            `json:"sv_nummer" xml:"SVNummer"`
	Austrittsdatum time.Time         `json:"austrittsdatum" xml:"-"`
	Grund          ELDAAustrittGrund `json:"grund" xml:"Grund"`

	// Final settlement
	Abfertigung   int64 `json:"abfertigung" xml:"Abfertigung"`     // Severance pay in cents
	Urlaubsersatz int64 `json:"urlaubsersatz" xml:"Urlaubsersatz"` // Vacation compensation in cents

	DienstgeberNr string            `json:"dienstgeber_nr" xml:"DienstgeberNr"`
	Status        ELDAMeldungStatus `json:"status"`
	Reference     string            `json:"reference,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

// AustrittsdatumString returns the exit date as a string for XML
func (a *ELDAAbmeldung) AustrittsdatumString() string {
	return a.Austrittsdatum.Format("2006-01-02")
}

// ELDAResponse represents a response from ELDA
type ELDAResponse struct {
	RC        int    `xml:"rc"`
	Msg       string `xml:"msg"`
	Reference string `xml:"referenz"`
}

// ELDAAnmeldungDocument is the XML document for an Anmeldung
type ELDAAnmeldungDocument struct {
	XMLName        xml.Name           `xml:"Anmeldung"`
	XMLNS          string             `xml:"xmlns,attr"`
	Kopf           ELDAKopf           `xml:"Kopf"`
	SVNummer       string             `xml:"SVNummer"`
	Vorname        string             `xml:"Vorname"`
	Nachname       string             `xml:"Nachname"`
	Geburtsdatum   string             `xml:"Geburtsdatum"`
	Geschlecht     string             `xml:"Geschlecht"`
	Eintrittsdatum string             `xml:"Eintrittsdatum"`
	Beschaeftigung ELDABeschaeftigung `xml:"Beschaeftigung"`
	Arbeitszeit    ELDAArbeitszeit    `xml:"Arbeitszeit"`
	Entgelt        ELDAEntgelt        `xml:"Entgelt"`
}

// ELDAAbmeldungDocument is the XML document for an Abmeldung
type ELDAAbmeldungDocument struct {
	XMLName        xml.Name          `xml:"Abmeldung"`
	XMLNS          string            `xml:"xmlns,attr"`
	Kopf           ELDAKopf          `xml:"Kopf"`
	SVNummer       string            `xml:"SVNummer"`
	Austrittsdatum string            `xml:"Austrittsdatum"`
	Grund          ELDAAustrittGrund `xml:"Grund"`
	Abfertigung    int64             `xml:"Abfertigung,omitempty"`
	Urlaubsersatz  int64             `xml:"Urlaubsersatz,omitempty"`
}

// ELDACredentials stores ELDA-specific authentication
type ELDACredentials struct {
	DienstgeberNr string `json:"dienstgeber_nr"`
	BenutzerNr    string `json:"benutzer_nr"`
	PIN           string `json:"pin"`
}

// NewELDAAnmeldung creates a new employee registration
func NewELDAAnmeldung() *ELDAAnmeldung {
	return &ELDAAnmeldung{
		Status:    ELDAStatusDraft,
		CreatedAt: time.Now(),
	}
}

// NewELDAAbmeldung creates a new employee deregistration
func NewELDAAbmeldung() *ELDAAbmeldung {
	return &ELDAAbmeldung{
		Status:    ELDAStatusDraft,
		CreatedAt: time.Now(),
	}
}
