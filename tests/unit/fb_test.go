package unit

import (
	"encoding/xml"
	"testing"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/fb"
)

// T067: Test FBSearchRequest serialization
func TestFBSearchRequestSerialization(t *testing.T) {
	req := &fb.FBSearchRequest{
		Name:    "Acme GmbH",
		Ort:     "Wien",
		MaxHits: 10,
	}

	xmlData, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal FBSearchRequest: %v", err)
	}

	xmlStr := string(xmlData)

	if !contains(xmlStr, "<Name>Acme GmbH</Name>") {
		t.Error("Missing Name element")
	}
	if !contains(xmlStr, "<Ort>Wien</Ort>") {
		t.Error("Missing Ort element")
	}
	if !contains(xmlStr, "<MaxHits>10</MaxHits>") {
		t.Error("Missing MaxHits element")
	}
}

// T068: Test FBExtractResponse parsing
func TestFBExtractResponseParsing(t *testing.T) {
	// Test basic extract data
	extract := &fb.FBExtract{
		FN:        "FN123456a",
		Firma:     "Acme GmbH",
		Rechtsform: fb.RechtsformGmbH,
		Sitz:      "Wien",
		Adresse: fb.FBAdresse{
			Strasse: "Hauptstraße 1",
			PLZ:     "1010",
			Ort:     "Wien",
			Land:    "AT",
		},
		Stammkapital: 3500000, // 35000.00 EUR in cents
		Status:       fb.FBStatusAktiv,
	}

	if extract.FN != "FN123456a" {
		t.Errorf("Expected FN FN123456a, got %s", extract.FN)
	}
	if extract.Firma != "Acme GmbH" {
		t.Errorf("Expected Firma Acme GmbH, got %s", extract.Firma)
	}
	if extract.Rechtsform != fb.RechtsformGmbH {
		t.Errorf("Expected Rechtsform GmbH, got %s", extract.Rechtsform)
	}
	if extract.Stammkapital != 3500000 {
		t.Errorf("Expected Stammkapital 3500000, got %d", extract.Stammkapital)
	}
}

// T069: Test FN number format validation
func TestFNValidation(t *testing.T) {
	testCases := []struct {
		fn    string
		valid bool
	}{
		{"FN123456a", true},
		{"FN999999z", true},
		{"FN000001a", true},
		{"FN12345678a", true},    // 8-digit number
		{"FN1234567890a", false}, // Too many digits
		{"FN12345", false},       // Missing suffix letter
		{"FN12345aa", false},     // Two letters
		{"FN12345A", false},      // Uppercase suffix (should be lowercase)
		{"123456a", false},       // Missing FN prefix
		{"", false},              // Empty
		{"FN", false},            // Just prefix
		{"FNa", false},           // No digits
		{"FN0a", true},           // Minimum valid
	}

	for _, tc := range testCases {
		t.Run(tc.fn, func(t *testing.T) {
			err := fb.ValidateFN(tc.fn)
			if tc.valid && err != nil {
				t.Errorf("Expected valid FN, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("Expected error for invalid FN, got nil")
			}
		})
	}
}

// Test FBSearchResponse with results
func TestFBSearchResponseWithResults(t *testing.T) {
	resp := &fb.FBSearchResponse{
		Results: []fb.FBSearchResult{
			{
				FN:    "FN123456a",
				Firma: "Acme GmbH",
				Sitz:  "Wien",
			},
			{
				FN:    "FN654321b",
				Firma: "Acme AG",
				Sitz:  "Graz",
			},
		},
		TotalCount: 2,
	}

	if len(resp.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(resp.Results))
	}
	if resp.TotalCount != 2 {
		t.Errorf("Expected TotalCount 2, got %d", resp.TotalCount)
	}
	if resp.Results[0].FN != "FN123456a" {
		t.Errorf("Expected first result FN123456a, got %s", resp.Results[0].FN)
	}
}

// Test FBPerson struct
func TestFBPerson(t *testing.T) {
	person := &fb.FBPerson{
		Vorname:       "Max",
		Nachname:      "Mustermann",
		Geburtsdatum:  time.Date(1980, 5, 15, 0, 0, 0, 0, time.UTC),
		Funktion:      fb.FunktionGeschaeftsfuehrer,
		VertretungsArt: fb.VertretungSelbstaendig,
		Seit:          time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if person.Vorname != "Max" {
		t.Errorf("Expected Vorname Max, got %s", person.Vorname)
	}
	if person.Funktion != fb.FunktionGeschaeftsfuehrer {
		t.Errorf("Expected Funktion Geschäftsführer, got %s", person.Funktion)
	}
	if person.GeburtsdatumString() != "1980-05-15" {
		t.Errorf("Expected Geburtsdatum 1980-05-15, got %s", person.GeburtsdatumString())
	}
}

// Test FBGesellschafter struct
func TestFBGesellschafter(t *testing.T) {
	g := &fb.FBGesellschafter{
		Name:          "Holding GmbH",
		FN:            "FN999999z",
		Anteil:        5000, // 50.00% in basis points
		Stammeinlage:  1750000, // 17500.00 EUR in cents
	}

	if g.Name != "Holding GmbH" {
		t.Errorf("Expected Name Holding GmbH, got %s", g.Name)
	}
	if g.Anteil != 5000 {
		t.Errorf("Expected Anteil 5000 (50%%), got %d", g.Anteil)
	}
	if g.AnteilProzent() != 50.0 {
		t.Errorf("Expected AnteilProzent 50.0, got %.2f", g.AnteilProzent())
	}
}

// Test Rechtsform constants
func TestRechtsformConstants(t *testing.T) {
	rechtsformen := []fb.Rechtsform{
		fb.RechtsformGmbH,
		fb.RechtsformAG,
		fb.RechtsformKG,
		fb.RechtsformOG,
		fb.RechtsformEU,
		fb.RechtsformGenossenschaft,
		fb.RechtsformVerein,
		fb.RechtsformStiftung,
	}

	for _, rf := range rechtsformen {
		if rf == "" {
			t.Error("Rechtsform constant is empty")
		}
	}
}

// Test FBStatus constants
func TestFBStatusConstants(t *testing.T) {
	statuses := []fb.FBStatus{
		fb.FBStatusAktiv,
		fb.FBStatusGeloescht,
		fb.FBStatusInLiquidation,
		fb.FBStatusInsolvent,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("FBStatus constant is empty")
		}
	}
}

// Test Funktion constants
func TestFunktionConstants(t *testing.T) {
	funktionen := []fb.Funktion{
		fb.FunktionGeschaeftsfuehrer,
		fb.FunktionVorstand,
		fb.FunktionProkulist,
		fb.FunktionAufsichtsrat,
		fb.FunktionKomplementaer,
		fb.FunktionKommanditist,
	}

	for _, f := range funktionen {
		if f == "" {
			t.Error("Funktion constant is empty")
		}
	}
}

// Test NewFBClient
func TestNewFBClient(t *testing.T) {
	client := fb.NewClient("test-api-key", true)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
}

// Test FBWatchlist
func TestFBWatchlist(t *testing.T) {
	wl := fb.NewWatchlist()

	// Test adding entry
	entry := fb.FBWatchlistEntry{
		FN:       "FN123456a",
		Firma:    "Acme GmbH",
		AddedAt:  time.Now(),
		LastCheck: time.Time{},
	}

	wl.Add(entry)

	if len(wl.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(wl.Entries))
	}

	// Test finding entry
	found := wl.Find("FN123456a")
	if found == nil {
		t.Error("Expected to find entry")
	}

	// Test removing entry
	wl.Remove("FN123456a")
	if len(wl.Entries) != 0 {
		t.Errorf("Expected 0 entries after removal, got %d", len(wl.Entries))
	}
}

// Test FBExtract date methods
func TestFBExtractDateMethods(t *testing.T) {
	extract := &fb.FBExtract{
		Gruendungsdatum: time.Date(2010, 3, 15, 0, 0, 0, 0, time.UTC),
		LetzteAenderung: time.Date(2024, 11, 20, 0, 0, 0, 0, time.UTC),
	}

	if extract.GruendungsdatumString() != "2010-03-15" {
		t.Errorf("Expected Gruendungsdatum 2010-03-15, got %s", extract.GruendungsdatumString())
	}
	if extract.LetzteAenderungString() != "2024-11-20" {
		t.Errorf("Expected LetzteAenderung 2024-11-20, got %s", extract.LetzteAenderungString())
	}
}
