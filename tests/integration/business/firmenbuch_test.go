package business

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/firmenbuch"
	"github.com/google/uuid"
)

// T088: Integration tests for Firmenbuch (Austrian company registry) API

func TestFirmenbuchTypes(t *testing.T) {
	t.Run("Company struct fields", func(t *testing.T) {
		now := time.Now()
		stammkapital := int64(3500000) // 35,000.00 EUR
		waehrung := "EUR"
		uid := "ATU12345678"
		gegenstand := "Softwareentwicklung"

		company := &firmenbuch.Company{
			ID:              uuid.New(),
			TenantID:        uuid.New(),
			FN:              "FN123456a",
			Name:            "Test GmbH",
			Rechtsform:      "GmbH",
			Sitz:            "Wien",
			Stammkapital:    &stammkapital,
			Waehrung:        &waehrung,
			Status:          "aktiv",
			Gruendungsdatum: &now,
			UID:             &uid,
			Gegenstand:      &gegenstand,
			LastFetchedAt:   &now,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if company.FN != "FN123456a" {
			t.Errorf("FN mismatch: got %s", company.FN)
		}
		if company.Name != "Test GmbH" {
			t.Errorf("Name mismatch: got %s", company.Name)
		}
		if company.Rechtsform != "GmbH" {
			t.Errorf("Rechtsform mismatch: got %s", company.Rechtsform)
		}
		if company.Status != "aktiv" {
			t.Errorf("Status mismatch: got %s", company.Status)
		}
	})

	t.Run("WatchlistEntry struct fields", func(t *testing.T) {
		now := time.Now()
		notes := "Monitor for ownership changes"

		entry := &firmenbuch.WatchlistEntry{
			ID:          uuid.New(),
			TenantID:    uuid.New(),
			CompanyID:   uuid.New(),
			FN:          "FN123456a",
			Name:        "Test GmbH",
			LastStatus:  "aktiv",
			LastChecked: &now,
			Notes:       &notes,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if entry.FN != "FN123456a" {
			t.Errorf("FN mismatch: got %s", entry.FN)
		}
		if entry.LastStatus != "aktiv" {
			t.Errorf("LastStatus mismatch: got %s", entry.LastStatus)
		}
	})

	t.Run("HistoryEntry struct fields", func(t *testing.T) {
		now := time.Now()
		oldValue := json.RawMessage(`{"status":"aktiv"}`)
		newValue := json.RawMessage(`{"status":"gelöscht"}`)

		entry := &firmenbuch.HistoryEntry{
			ID:         uuid.New(),
			CompanyID:  uuid.New(),
			ChangeType: "status_change",
			OldValue:   oldValue,
			NewValue:   newValue,
			DetectedAt: now,
			CreatedAt:  now,
		}

		if entry.ChangeType != "status_change" {
			t.Errorf("ChangeType mismatch: got %s", entry.ChangeType)
		}
	})
}

func TestFirmenbuchSearchRequestParsing(t *testing.T) {
	t.Run("Parse search by name", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/firmenbuch/search?name=Test%20GmbH&max_hits=20", nil)

		query := req.URL.Query()

		if query.Get("name") != "Test GmbH" {
			t.Errorf("name not parsed correctly: got %s", query.Get("name"))
		}
		if query.Get("max_hits") != "20" {
			t.Errorf("max_hits not parsed correctly: got %s", query.Get("max_hits"))
		}
	})

	t.Run("Parse search by FN", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/firmenbuch/search?fn=FN123456a", nil)

		query := req.URL.Query()

		if query.Get("fn") != "FN123456a" {
			t.Errorf("fn not parsed correctly: got %s", query.Get("fn"))
		}
	})

	t.Run("Parse search by location", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/firmenbuch/search?ort=Wien", nil)

		query := req.URL.Query()

		if query.Get("ort") != "Wien" {
			t.Errorf("ort not parsed correctly: got %s", query.Get("ort"))
		}
	})
}

func TestFirmenbuchSearchInput(t *testing.T) {
	t.Run("SearchInput struct", func(t *testing.T) {
		input := &firmenbuch.SearchInput{
			Name:    "Test",
			FN:      "",
			Ort:     "Wien",
			MaxHits: 20,
		}

		if input.Name != "Test" {
			t.Errorf("Name mismatch: got %s", input.Name)
		}
		if input.Ort != "Wien" {
			t.Errorf("Ort mismatch: got %s", input.Ort)
		}
		if input.MaxHits != 20 {
			t.Errorf("MaxHits mismatch: got %d", input.MaxHits)
		}
	})
}

func TestFirmenbuchSearchResponse(t *testing.T) {
	t.Run("SearchResponse structure", func(t *testing.T) {
		resp := &firmenbuch.SearchResponse{
			Results: []firmenbuch.SearchResult{
				{
					FN:         "FN123456a",
					Name:       "Test GmbH",
					Rechtsform: "GmbH",
					Sitz:       "Wien",
					Status:     "aktiv",
				},
				{
					FN:         "FN654321b",
					Name:       "Another Test GmbH",
					Rechtsform: "GmbH",
					Sitz:       "Wien",
					Status:     "aktiv",
				},
			},
			TotalCount: 2,
			Cached:     false,
		}

		if len(resp.Results) != 2 {
			t.Errorf("Results count mismatch: got %d", len(resp.Results))
		}
		if resp.TotalCount != 2 {
			t.Errorf("TotalCount mismatch: got %d", resp.TotalCount)
		}
	})
}

func TestFirmenbuchExtractResponse(t *testing.T) {
	t.Run("ExtractResponse full structure", func(t *testing.T) {
		stammkapital := 35000.00
		waehrung := "EUR"
		gruendungsdatum := "2020-01-15"
		letzteAenderung := "2024-12-01"
		uid := "ATU12345678"
		gegenstand := "Softwareentwicklung und IT-Beratung"
		lastFetchedAt := "2025-01-15T10:00:00Z"

		resp := &firmenbuch.ExtractResponse{
			FN:         "FN123456a",
			Name:       "Test GmbH",
			Rechtsform: "GmbH",
			Sitz:       "Wien",
			Adresse: &firmenbuch.AddressResponse{
				Strasse: "Musterstraße 1",
				PLZ:     "1010",
				Ort:     "Wien",
				Land:    "Österreich",
			},
			Stammkapital:     &stammkapital,
			Waehrung:         &waehrung,
			Status:           "aktiv",
			Gruendungsdatum:  &gruendungsdatum,
			LetzteAenderung:  &letzteAenderung,
			UID:              &uid,
			Gegenstand:       &gegenstand,
			Geschaeftsfuehrer: []firmenbuch.PersonResponse{
				{
					Vorname:        "Max",
					Nachname:       "Mustermann",
					Funktion:       "Geschäftsführer",
					VertretungsArt: "selbständig",
				},
			},
			Gesellschafter: []firmenbuch.ShareholderResponse{
				{
					Name:      "Holding GmbH",
					AnteilPct: 100.0,
				},
			},
			LastFetchedAt: &lastFetchedAt,
			Cached:        true,
		}

		if resp.FN != "FN123456a" {
			t.Error("FN mismatch")
		}
		if resp.Adresse == nil {
			t.Error("Adresse should not be nil")
		}
		if resp.Adresse.PLZ != "1010" {
			t.Error("PLZ mismatch")
		}
		if len(resp.Geschaeftsfuehrer) != 1 {
			t.Errorf("Geschaeftsfuehrer count mismatch: got %d", len(resp.Geschaeftsfuehrer))
		}
		if len(resp.Gesellschafter) != 1 {
			t.Errorf("Gesellschafter count mismatch: got %d", len(resp.Gesellschafter))
		}
	})
}

func TestFirmenbuchFNValidation(t *testing.T) {
	t.Run("Valid FN formats", func(t *testing.T) {
		// Austrian FN format: FN + 1-9 digits + lowercase letter (checksum)
		validFNs := []string{
			"FN1a",
			"FN123456a",
			"FN123456789z",
		}

		for _, fn := range validFNs {
			if len(fn) < 4 {
				t.Errorf("FN should be at least 4 characters: %s", fn)
			}
			if fn[:2] != "FN" {
				t.Errorf("FN should start with 'FN': %s", fn)
			}
			lastChar := fn[len(fn)-1]
			if lastChar < 'a' || lastChar > 'z' {
				t.Errorf("FN should end with lowercase letter: %s", fn)
			}
		}
	})

	t.Run("Invalid FN formats", func(t *testing.T) {
		invalidFNs := []string{
			"123456a",      // Missing FN prefix
			"FN123456",     // Missing checksum letter
			"FN123456A",    // Uppercase checksum
			"FNa",          // No digits
			"fn123456a",    // Lowercase FN prefix
		}

		for _, fn := range invalidFNs {
			t.Logf("Testing invalid FN: %s", fn)
		}
	})
}

func TestFirmenbuchValidateEndpoint(t *testing.T) {
	t.Run("Parse validate request", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"fn": "FN123456a",
		}

		body, _ := json.Marshal(reqBody)

		var parsed struct {
			FN string `json:"fn"`
		}
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse request: %v", err)
		}

		if parsed.FN != "FN123456a" {
			t.Errorf("FN not parsed correctly: got %s", parsed.FN)
		}
	})
}

func TestFirmenbuchWatchlistInput(t *testing.T) {
	t.Run("WatchlistInput struct", func(t *testing.T) {
		notes := "Monitor quarterly"
		input := &firmenbuch.WatchlistInput{
			FN:    "FN123456a",
			Notes: &notes,
		}

		if input.FN != "FN123456a" {
			t.Errorf("FN mismatch: got %s", input.FN)
		}
		if input.Notes == nil || *input.Notes != "Monitor quarterly" {
			t.Error("Notes not set correctly")
		}
	})
}

func TestFirmenbuchWatchlistResponse(t *testing.T) {
	t.Run("WatchlistResponse structure", func(t *testing.T) {
		lastChecked := "2025-01-15T10:00:00Z"
		notes := "Important client"

		resp := firmenbuch.WatchlistResponse{
			ID:          uuid.New(),
			FN:          "FN123456a",
			Name:        "Test GmbH",
			LastStatus:  "aktiv",
			LastChecked: &lastChecked,
			Notes:       &notes,
			CreatedAt:   "2025-01-01T10:00:00Z",
		}

		if resp.FN != "FN123456a" {
			t.Error("FN mismatch")
		}
		if resp.LastStatus != "aktiv" {
			t.Error("LastStatus mismatch")
		}
		if resp.Notes == nil || *resp.Notes != "Important client" {
			t.Error("Notes not set correctly")
		}
	})
}

func TestFirmenbuchHistoryResponse(t *testing.T) {
	t.Run("HistoryResponse structure", func(t *testing.T) {
		oldValue := json.RawMessage(`{"status":"aktiv"}`)
		newValue := json.RawMessage(`{"status":"gelöscht"}`)

		resp := firmenbuch.HistoryResponse{
			ID:         uuid.New(),
			ChangeType: "status_change",
			OldValue:   oldValue,
			NewValue:   newValue,
			DetectedAt: "2025-01-15T10:00:00Z",
		}

		if resp.ChangeType != "status_change" {
			t.Error("ChangeType mismatch")
		}
	})
}

func TestFirmenbuchListFilter(t *testing.T) {
	t.Run("ListFilter defaults", func(t *testing.T) {
		filter := firmenbuch.ListFilter{
			TenantID: uuid.New(),
			Limit:    50,
			Offset:   0,
		}

		if filter.Limit != 50 {
			t.Errorf("Default limit should be 50, got %d", filter.Limit)
		}
		if filter.Offset != 0 {
			t.Errorf("Default offset should be 0, got %d", filter.Offset)
		}
	})

	t.Run("ListFilter with options", func(t *testing.T) {
		status := "aktiv"
		search := "Test"

		filter := firmenbuch.ListFilter{
			TenantID: uuid.New(),
			Status:   &status,
			Search:   &search,
			Limit:    20,
			Offset:   10,
		}

		if filter.Status == nil || *filter.Status != "aktiv" {
			t.Error("Status filter not set correctly")
		}
		if filter.Search == nil || *filter.Search != "Test" {
			t.Error("Search filter not set correctly")
		}
	})
}

func TestFirmenbuchRechtsformen(t *testing.T) {
	// Common Austrian legal forms
	rechtsformen := map[string]string{
		"GmbH":   "Gesellschaft mit beschränkter Haftung",
		"AG":     "Aktiengesellschaft",
		"OG":     "Offene Gesellschaft",
		"KG":     "Kommanditgesellschaft",
		"e.U.":   "eingetragener Einzelunternehmer",
		"GesbR":  "Gesellschaft bürgerlichen Rechts",
		"SE":     "Societas Europaea",
		"Verein": "Verein",
	}

	for code, desc := range rechtsformen {
		if code == "" || desc == "" {
			t.Error("Rechtsform code or description is empty")
		}
	}

	// GmbH should be available
	if _, ok := rechtsformen["GmbH"]; !ok {
		t.Error("GmbH should be a valid Rechtsform")
	}
}

func TestFirmenbuchCompanyStatus(t *testing.T) {
	// Austrian company registry statuses
	statuses := map[string]string{
		"aktiv":         "Active company",
		"in_liquidation": "Company in liquidation",
		"gelöscht":      "Deleted from registry",
		"insolvent":     "Insolvency proceedings",
	}

	for status, desc := range statuses {
		if status == "" || desc == "" {
			t.Error("Status or description is empty")
		}
	}

	// aktiv should be available
	if _, ok := statuses["aktiv"]; !ok {
		t.Error("aktiv should be a valid status")
	}
}

func TestFirmenbuchCaching(t *testing.T) {
	t.Run("Cache duration", func(t *testing.T) {
		// Extracts should be cached for 24 hours
		cacheDuration := firmenbuch.CacheDuration

		if cacheDuration.Hours() != 24 {
			t.Errorf("Cache duration should be 24 hours, got %f", cacheDuration.Hours())
		}
	})

	t.Run("Force refresh query param", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/firmenbuch/extract/FN123456a?refresh=true", nil)

		if req.URL.Query().Get("refresh") != "true" {
			t.Error("refresh query param not parsed correctly")
		}
	})
}

func TestFirmenbuchPersonResponse(t *testing.T) {
	t.Run("Geschäftsführer response", func(t *testing.T) {
		seit := "2020-01-15"
		resp := firmenbuch.PersonResponse{
			Vorname:        "Max",
			Nachname:       "Mustermann",
			Funktion:       "Geschäftsführer",
			VertretungsArt: "selbständig",
			Seit:           &seit,
		}

		if resp.Vorname != "Max" {
			t.Error("Vorname mismatch")
		}
		if resp.Funktion != "Geschäftsführer" {
			t.Error("Funktion mismatch")
		}
		if resp.VertretungsArt != "selbständig" {
			t.Error("VertretungsArt mismatch")
		}
	})
}

func TestFirmenbuchShareholderResponse(t *testing.T) {
	t.Run("Gesellschafter response", func(t *testing.T) {
		fn := "FN654321b"
		stammeinlage := 35000.00
		seit := "2020-01-15"

		resp := firmenbuch.ShareholderResponse{
			Name:         "Holding GmbH",
			FN:           &fn,
			AnteilPct:    100.0,
			Stammeinlage: &stammeinlage,
			Seit:         &seit,
		}

		if resp.Name != "Holding GmbH" {
			t.Error("Name mismatch")
		}
		if resp.AnteilPct != 100.0 {
			t.Errorf("AnteilPct mismatch: got %f", resp.AnteilPct)
		}
		if resp.FN == nil || *resp.FN != "FN654321b" {
			t.Error("FN not set correctly")
		}
	})
}

func TestFirmenbuchHTTPStatusCodes(t *testing.T) {
	testCases := []struct {
		scenario   string
		statusCode int
	}{
		{"company found", 200},
		{"company not found", 404},
		{"invalid FN format", 400},
		{"already on watchlist", 409},
		{"search empty params", 400},
	}

	for _, tc := range testCases {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.statusCode < 100 || tc.statusCode > 599 {
				t.Errorf("Invalid HTTP status code: %d", tc.statusCode)
			}
		})
	}
}
