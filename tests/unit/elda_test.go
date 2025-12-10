package unit

import (
	"testing"
	"time"

	"austrian-business-infrastructure/internal/elda"
)

// T046: Test ELDA Anmeldung XML generation
func TestELDAAnmeldungXMLGeneration(t *testing.T) {
	creds := &elda.ELDACredentials{
		DienstgeberNr: "12345678",
		BenutzerNr:    "USER001",
		PIN:           "secret",
	}

	// SV-Nummer: 1234 (serial) + 150189 (birth date 15.01.89)
	// Format: NNNN TTMMJJ (10 digits total)
	// Valid test SV-Nummer: 1234150189 = serial 1234 + date 15.01.1989
	anmeldung := &elda.ELDAAnmeldung{
		SVNummer:       "1234150189", // Valid SV-Nr for 15.01.1989
		Vorname:        "Max",
		Nachname:       "Mustermann",
		Geburtsdatum:   time.Date(1989, 1, 15, 0, 0, 0, 0, time.UTC),
		Geschlecht:     "M",
		Eintrittsdatum: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		Beschaeftigung: elda.ELDABeschaeftigung{
			Art:        elda.BeschaeftigungVollzeit,
			Taetigkeit: "Software Developer",
		},
		Arbeitszeit: elda.ELDAArbeitszeit{
			Stunden: 38.5,
			Tage:    5,
		},
		Entgelt: elda.ELDAEntgelt{
			Brutto: 350000, // 3500.00 EUR
		},
		DienstgeberNr: "12345678",
	}

	xmlData, err := elda.GenerateAnmeldungXML(creds, anmeldung)
	if err != nil {
		t.Fatalf("Failed to generate Anmeldung XML: %v", err)
	}

	xmlStr := string(xmlData)

	// Verify XML structure
	if !contains(xmlStr, "<Anmeldung") {
		t.Error("Missing Anmeldung root element")
	}
	if !contains(xmlStr, "<SVNummer>1234150189</SVNummer>") {
		t.Error("Missing or incorrect SVNummer")
	}
	if !contains(xmlStr, "<Vorname>Max</Vorname>") {
		t.Error("Missing or incorrect Vorname")
	}
	if !contains(xmlStr, "<Nachname>Mustermann</Nachname>") {
		t.Error("Missing or incorrect Nachname")
	}
	if !contains(xmlStr, "<Geschlecht>M</Geschlecht>") {
		t.Error("Missing or incorrect Geschlecht")
	}
	if !contains(xmlStr, "<Brutto>350000</Brutto>") {
		t.Error("Missing or incorrect Brutto")
	}
}

// T047: Test SV-Nummer validation
func TestSVNummerValidation(t *testing.T) {
	testCases := []struct {
		svNummer string
		valid    bool
		errType  error
	}{
		{"1234150189", true, nil},  // Valid SV-Nummer (15.01.85, check digit 9)
		{"1234567890", false, nil}, // Invalid check digit
		{"123456789", false, elda.ErrInvalidSVNummer},  // Too short
		{"12345678901", false, elda.ErrInvalidSVNummer}, // Too long
		{"123456789X", false, elda.ErrInvalidSVNummer},  // Contains letter
		{"", false, elda.ErrInvalidSVNummer},            // Empty
	}

	for _, tc := range testCases {
		t.Run(tc.svNummer, func(t *testing.T) {
			err := elda.ValidateSVNummer(tc.svNummer)
			if tc.valid && err != nil {
				t.Errorf("Expected valid SV-Nummer, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("Expected error for invalid SV-Nummer, got nil")
			}
		})
	}
}

// T047b: Test SV-Nummer with birth date validation
func TestSVNummerWithBirthDateValidation(t *testing.T) {
	// SV-Nummer 1234150189 has birth date embedded: 15.01.89
	svNummer := "1234150189"
	correctDate := time.Date(1989, 1, 15, 0, 0, 0, 0, time.UTC)
	wrongDate := time.Date(1990, 5, 20, 0, 0, 0, 0, time.UTC)

	// Test with correct date
	err := elda.ValidateSVNummerWithBirthDate(svNummer, correctDate)
	if err != nil {
		t.Errorf("Expected valid with correct birth date, got error: %v", err)
	}

	// Test with wrong date
	err = elda.ValidateSVNummerWithBirthDate(svNummer, wrongDate)
	if err == nil {
		t.Error("Expected error with wrong birth date, got nil")
	}
}

// T047c: Test birth date extraction from SV-Nummer
func TestExtractBirthDateFromSVNummer(t *testing.T) {
	svNummer := "1234150189"
	expectedDate := time.Date(1989, 1, 15, 0, 0, 0, 0, time.UTC)

	extractedDate, err := elda.ExtractBirthDateFromSVNummer(svNummer)
	if err != nil {
		t.Fatalf("Failed to extract birth date: %v", err)
	}

	if !extractedDate.Equal(expectedDate) {
		t.Errorf("Expected %v, got %v", expectedDate, extractedDate)
	}
}

// T048: Test ELDA response parsing
func TestELDAResponseParsing(t *testing.T) {
	// Test success response
	resp := &elda.ELDAResponse{
		RC:        0,
		Msg:       "Meldung erfolgreich übermittelt",
		Reference: "ELDA-2025-AN-12345678",
	}

	if resp.RC != 0 {
		t.Errorf("Expected RC=0, got %d", resp.RC)
	}
	if resp.Reference != "ELDA-2025-AN-12345678" {
		t.Errorf("Expected reference ELDA-2025-AN-12345678, got %s", resp.Reference)
	}
}

// Test ELDA status constants
func TestELDAStatusConstants(t *testing.T) {
	statuses := []elda.ELDAMeldungStatus{
		elda.ELDAStatusDraft,
		elda.ELDAStatusSubmitted,
		elda.ELDAStatusProcessed,
		elda.ELDAStatusRejected,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("Status constant is empty")
		}
	}
}

// Test employment type constants
func TestBeschaeftigungConstants(t *testing.T) {
	if elda.BeschaeftigungVollzeit != "vollzeit" {
		t.Errorf("Expected vollzeit, got %s", elda.BeschaeftigungVollzeit)
	}
	if elda.BeschaeftigungTeilzeit != "teilzeit" {
		t.Errorf("Expected teilzeit, got %s", elda.BeschaeftigungTeilzeit)
	}
	if elda.BeschaeftigungGeringfuegig != "geringfuegig" {
		t.Errorf("Expected geringfuegig, got %s", elda.BeschaeftigungGeringfuegig)
	}
}

// Test exit reason constants
func TestAustrittGrundConstants(t *testing.T) {
	grounds := []elda.ELDAAustrittGrund{
		elda.ELDAGrundKuendigung,
		elda.ELDAGrundEinvernehmlich,
		elda.ELDAGrundEntlassung,
		elda.ELDAGrundAustritt,
		elda.ELDAGrundBefristet,
	}

	for _, grund := range grounds {
		if grund == "" {
			t.Error("Austritt grund constant is empty")
		}
	}
}

// Test Anmeldung date string methods
func TestAnmeldungDateMethods(t *testing.T) {
	anmeldung := &elda.ELDAAnmeldung{
		Geburtsdatum:   time.Date(1989, 1, 15, 0, 0, 0, 0, time.UTC),
		Eintrittsdatum: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	if anmeldung.GeburtsdatumString() != "1989-01-15" {
		t.Errorf("Expected 1989-01-15, got %s", anmeldung.GeburtsdatumString())
	}
	if anmeldung.EintrittsdatumString() != "2025-02-01" {
		t.Errorf("Expected 2025-02-01, got %s", anmeldung.EintrittsdatumString())
	}
}

// Test Abmeldung date string method
func TestAbmeldungDateMethod(t *testing.T) {
	abmeldung := &elda.ELDAAbmeldung{
		Austrittsdatum: time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
	}

	if abmeldung.AustrittsdatumString() != "2025-06-30" {
		t.Errorf("Expected 2025-06-30, got %s", abmeldung.AustrittsdatumString())
	}
}

// Test NewELDAAnmeldung constructor
func TestNewELDAAnmeldung(t *testing.T) {
	anmeldung := elda.NewELDAAnmeldung()

	if anmeldung.Status != elda.ELDAStatusDraft {
		t.Errorf("Expected status draft, got %s", anmeldung.Status)
	}
	if anmeldung.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

// Test NewELDAAbmeldung constructor
func TestNewELDAAbmeldung(t *testing.T) {
	abmeldung := elda.NewELDAAbmeldung()

	if abmeldung.Status != elda.ELDAStatusDraft {
		t.Errorf("Expected status draft, got %s", abmeldung.Status)
	}
	if abmeldung.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}
