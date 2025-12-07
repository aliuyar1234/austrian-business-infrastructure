package fb

import (
	"encoding/xml"
	"errors"
	"regexp"
	"time"
)

var (
	ErrInvalidFN = errors.New("invalid Firmenbuch number format")

	// FN format: FN followed by 1-9 digits followed by a single lowercase letter
	fnPattern = regexp.MustCompile(`^FN\d{1,9}[a-z]$`)
)

// ValidateFN validates an Austrian Firmenbuch number
// Format: FN + 1-9 digits + lowercase letter (e.g., FN123456a)
func ValidateFN(fn string) error {
	if !fnPattern.MatchString(fn) {
		return ErrInvalidFN
	}
	return nil
}

// Rechtsform represents the legal form of a company
type Rechtsform string

const (
	RechtsformGmbH          Rechtsform = "GmbH"           // Gesellschaft mit beschränkter Haftung
	RechtsformAG            Rechtsform = "AG"             // Aktiengesellschaft
	RechtsformKG            Rechtsform = "KG"             // Kommanditgesellschaft
	RechtsformOG            Rechtsform = "OG"             // Offene Gesellschaft
	RechtsformEU            Rechtsform = "e.U."           // Eingetragener Unternehmer
	RechtsformGenossenschaft Rechtsform = "GenmbH"        // Genossenschaft
	RechtsformVerein        Rechtsform = "Verein"         // Verein
	RechtsformStiftung      Rechtsform = "Stiftung"       // Privatstiftung
)

// FBStatus represents the status of a company in the Firmenbuch
type FBStatus string

const (
	FBStatusAktiv        FBStatus = "aktiv"
	FBStatusGeloescht    FBStatus = "geloescht"
	FBStatusInLiquidation FBStatus = "in_liquidation"
	FBStatusInsolvent    FBStatus = "insolvent"
)

// Funktion represents a function/role in a company
type Funktion string

const (
	FunktionGeschaeftsfuehrer Funktion = "GF"       // Geschäftsführer
	FunktionVorstand          Funktion = "VOR"      // Vorstandsmitglied
	FunktionProkulist         Funktion = "PRO"      // Prokurist
	FunktionAufsichtsrat      Funktion = "AR"       // Aufsichtsrat
	FunktionKomplementaer     Funktion = "KOMP"     // Komplementär
	FunktionKommanditist      Funktion = "KOMM"     // Kommanditist
)

// VertretungsArt represents how someone can represent the company
type VertretungsArt string

const (
	VertretungSelbstaendig VertretungsArt = "selbstaendig"
	VertretungGemeinsam    VertretungsArt = "gemeinsam"
)

// FBAdresse represents an address in the Firmenbuch
type FBAdresse struct {
	Strasse string `json:"strasse" xml:"Strasse"`
	PLZ     string `json:"plz" xml:"PLZ"`
	Ort     string `json:"ort" xml:"Ort"`
	Land    string `json:"land" xml:"Land"`
}

// FBSearchRequest represents a search request to the Firmenbuch
type FBSearchRequest struct {
	XMLName xml.Name `xml:"FBSuche"`
	Name    string   `xml:"Name,omitempty"`
	FN      string   `xml:"FN,omitempty"`
	Ort     string   `xml:"Ort,omitempty"`
	MaxHits int      `xml:"MaxHits,omitempty"`
}

// FBSearchResult represents a single search result
type FBSearchResult struct {
	FN        string     `json:"fn" xml:"FN"`
	Firma     string     `json:"firma" xml:"Firma"`
	Rechtsform Rechtsform `json:"rechtsform" xml:"Rechtsform"`
	Sitz      string     `json:"sitz" xml:"Sitz"`
	Status    FBStatus   `json:"status" xml:"Status"`
}

// FBSearchResponse represents the response from a Firmenbuch search
type FBSearchResponse struct {
	XMLName    xml.Name         `xml:"FBSucheAntwort"`
	Results    []FBSearchResult `json:"results" xml:"Treffer"`
	TotalCount int              `json:"total_count" xml:"Anzahl"`
}

// FBExtract represents a full company extract from the Firmenbuch
type FBExtract struct {
	XMLName         xml.Name          `xml:"FBAuszug"`
	FN              string            `json:"fn" xml:"FN"`
	Firma           string            `json:"firma" xml:"Firma"`
	Rechtsform      Rechtsform        `json:"rechtsform" xml:"Rechtsform"`
	Sitz            string            `json:"sitz" xml:"Sitz"`
	Adresse         FBAdresse         `json:"adresse" xml:"Adresse"`
	Stammkapital    int64             `json:"stammkapital" xml:"Stammkapital"`        // In cents
	Waehrung        string            `json:"waehrung" xml:"Waehrung"`
	Status          FBStatus          `json:"status" xml:"Status"`
	Gruendungsdatum time.Time         `json:"gruendungsdatum" xml:"-"`
	LetzteAenderung time.Time         `json:"letzte_aenderung" xml:"-"`
	Geschaeftsfuehrer []FBPerson      `json:"geschaeftsfuehrer" xml:"Geschaeftsfuehrer>Person"`
	Gesellschafter  []FBGesellschafter `json:"gesellschafter" xml:"Gesellschafter>Gesellschafter"`
	Gegenstand      string            `json:"gegenstand" xml:"Gegenstand"`
	UID             string            `json:"uid" xml:"UID"`                          // VAT ID
}

// GruendungsdatumString returns the founding date as a string
func (e *FBExtract) GruendungsdatumString() string {
	return e.Gruendungsdatum.Format("2006-01-02")
}

// LetzteAenderungString returns the last change date as a string
func (e *FBExtract) LetzteAenderungString() string {
	return e.LetzteAenderung.Format("2006-01-02")
}

// StammkapitalEUR returns the share capital in EUR
func (e *FBExtract) StammkapitalEUR() float64 {
	return float64(e.Stammkapital) / 100
}

// FBPerson represents a person associated with a company
type FBPerson struct {
	Vorname        string         `json:"vorname" xml:"Vorname"`
	Nachname       string         `json:"nachname" xml:"Nachname"`
	Geburtsdatum   time.Time      `json:"geburtsdatum" xml:"-"`
	Funktion       Funktion       `json:"funktion" xml:"Funktion"`
	VertretungsArt VertretungsArt `json:"vertretungsart" xml:"VertretungsArt"`
	Seit           time.Time      `json:"seit" xml:"-"`
	Bis            *time.Time     `json:"bis,omitempty" xml:"-"`
}

// GeburtsdatumString returns the birth date as a string
func (p *FBPerson) GeburtsdatumString() string {
	return p.Geburtsdatum.Format("2006-01-02")
}

// SeitString returns the start date as a string
func (p *FBPerson) SeitString() string {
	return p.Seit.Format("2006-01-02")
}

// FullName returns the full name of the person
func (p *FBPerson) FullName() string {
	return p.Vorname + " " + p.Nachname
}

// FBGesellschafter represents a shareholder of a company
type FBGesellschafter struct {
	// Either a person or a company can be a shareholder
	Name         string    `json:"name" xml:"Name"`                   // Company name or person name
	FN           string    `json:"fn,omitempty" xml:"FN,omitempty"`   // If company
	Anteil       int       `json:"anteil" xml:"Anteil"`               // Share in basis points (1/100 of percent)
	Stammeinlage int64     `json:"stammeinlage" xml:"Stammeinlage"`   // Capital contribution in cents
	Seit         time.Time `json:"seit" xml:"-"`
}

// AnteilProzent returns the share percentage
func (g *FBGesellschafter) AnteilProzent() float64 {
	return float64(g.Anteil) / 100
}

// StammeinlageEUR returns the capital contribution in EUR
func (g *FBGesellschafter) StammeinlageEUR() float64 {
	return float64(g.Stammeinlage) / 100
}

// FBWatchlistEntry represents an entry in the company watchlist
type FBWatchlistEntry struct {
	FN         string    `json:"fn"`
	Firma      string    `json:"firma"`
	AddedAt    time.Time `json:"added_at"`
	LastCheck  time.Time `json:"last_check"`
	LastStatus FBStatus  `json:"last_status"`
	Notes      string    `json:"notes,omitempty"`
}

// FBWatchlist manages a list of companies to monitor
type FBWatchlist struct {
	Entries []FBWatchlistEntry `json:"entries"`
}

// NewWatchlist creates a new empty watchlist
func NewWatchlist() *FBWatchlist {
	return &FBWatchlist{
		Entries: make([]FBWatchlistEntry, 0),
	}
}

// Add adds an entry to the watchlist
func (w *FBWatchlist) Add(entry FBWatchlistEntry) {
	// Check if already exists
	for i, e := range w.Entries {
		if e.FN == entry.FN {
			w.Entries[i] = entry
			return
		}
	}
	w.Entries = append(w.Entries, entry)
}

// Remove removes an entry from the watchlist by FN
func (w *FBWatchlist) Remove(fn string) bool {
	for i, e := range w.Entries {
		if e.FN == fn {
			w.Entries = append(w.Entries[:i], w.Entries[i+1:]...)
			return true
		}
	}
	return false
}

// Find finds an entry by FN
func (w *FBWatchlist) Find(fn string) *FBWatchlistEntry {
	for i := range w.Entries {
		if w.Entries[i].FN == fn {
			return &w.Entries[i]
		}
	}
	return nil
}

// FBExtractRequest represents a request to get a company extract
type FBExtractRequest struct {
	XMLName xml.Name `xml:"FBAuszugAnfrage"`
	FN      string   `xml:"FN"`
}

// FBResponse represents a generic response from the Firmenbuch API
type FBResponse struct {
	XMLName xml.Name `xml:"FBAntwort"`
	RC      int      `xml:"rc"`
	Msg     string   `xml:"msg"`
}
