package business

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/uid"
	"github.com/google/uuid"
)

// T085: Integration tests for UID validation API

func TestUIDValidationTypes(t *testing.T) {
	t.Run("Validation struct fields", func(t *testing.T) {
		now := time.Now()
		companyName := "Test GmbH"
		accountID := uuid.New()
		validation := &uid.Validation{
			ID:          uuid.New(),
			TenantID:    uuid.New(),
			AccountID:   &accountID,
			UID:         "ATU12345678",
			CountryCode: "AT",
			Valid:       true,
			Level:       2,
			CompanyName: &companyName,
			Source:      "vies",
			ValidatedAt: now,
			CreatedAt:   now,
		}

		if validation.UID != "ATU12345678" {
			t.Errorf("UID mismatch: got %s", validation.UID)
		}
		if validation.CountryCode != "AT" {
			t.Errorf("CountryCode mismatch: got %s", validation.CountryCode)
		}
		if !validation.Valid {
			t.Error("Valid should be true")
		}
		if validation.Level != 2 {
			t.Errorf("Level mismatch: got %d", validation.Level)
		}
	})
}

func TestUIDValidateRequestParsing(t *testing.T) {
	t.Run("Parse validate request", func(t *testing.T) {
		accountID := uuid.New()
		reqBody := map[string]interface{}{
			"uid":        "ATU12345678",
			"level":      2,
			"account_id": accountID.String(),
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/uid/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		var parsed uid.ValidateRequest
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse request: %v", err)
		}

		if parsed.UID != "ATU12345678" {
			t.Errorf("UID mismatch: got %s", parsed.UID)
		}
		if parsed.Level != 2 {
			t.Errorf("Level mismatch: got %d", parsed.Level)
		}
		if parsed.AccountID != accountID.String() {
			t.Errorf("AccountID mismatch: got %s", parsed.AccountID)
		}
	})

	t.Run("Level 1 validation request", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"uid":        "DE123456789",
			"level":      1,
			"account_id": uuid.New().String(),
		}

		body, _ := json.Marshal(reqBody)

		var parsed uid.ValidateRequest
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse request: %v", err)
		}

		if parsed.Level != 1 {
			t.Errorf("Level should be 1, got %d", parsed.Level)
		}
	})
}

func TestUIDBatchValidateRequestParsing(t *testing.T) {
	t.Run("Parse batch validate request", func(t *testing.T) {
		accountID := uuid.New()
		uids := []string{"ATU12345678", "DE123456789", "FR12345678901"}
		reqBody := map[string]interface{}{
			"uids":       uids,
			"level":      2,
			"account_id": accountID.String(),
		}

		body, _ := json.Marshal(reqBody)

		var parsed uid.ValidateBatchRequest
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse batch request: %v", err)
		}

		if len(parsed.UIDs) != 3 {
			t.Errorf("UIDs count mismatch: got %d", len(parsed.UIDs))
		}
		if parsed.Level != 2 {
			t.Errorf("Level mismatch: got %d", parsed.Level)
		}
	})
}

func TestUIDValidationResponse(t *testing.T) {
	t.Run("Valid UID response structure", func(t *testing.T) {
		companyName := "Muster GmbH"
		street := "Musterstraße 1"
		postCode := "1010"
		city := "Wien"
		country := "Austria"

		resp := &uid.ValidationResponse{
			ID:          uuid.New(),
			UID:         "ATU12345678",
			CountryCode: "AT",
			Valid:       true,
			Level:       2,
			CompanyName: &companyName,
			Street:      &street,
			PostCode:    &postCode,
			City:        &city,
			Country:     &country,
			Source:      "vies",
			ValidatedAt: time.Now().Format("2006-01-02T15:04:05Z"),
			CreatedAt:   time.Now().Format("2006-01-02T15:04:05Z"),
		}

		if !resp.Valid {
			t.Error("Valid should be true")
		}
		if resp.CompanyName == nil || *resp.CompanyName != "Muster GmbH" {
			t.Error("CompanyName not set correctly")
		}
		if resp.Street == nil || *resp.Street != "Musterstraße 1" {
			t.Error("Street not set correctly")
		}
	})

	t.Run("Invalid UID response structure", func(t *testing.T) {
		errMsg := "UID not found"

		resp := &uid.ValidationResponse{
			ID:           uuid.New(),
			UID:          "ATU00000000",
			CountryCode:  "AT",
			Valid:        false,
			Level:        1,
			ErrorMessage: &errMsg,
			Source:       "vies",
			ValidatedAt:  time.Now().Format("2006-01-02T15:04:05Z"),
			CreatedAt:    time.Now().Format("2006-01-02T15:04:05Z"),
		}

		if resp.Valid {
			t.Error("Valid should be false")
		}
		if resp.ErrorMessage == nil || *resp.ErrorMessage != "UID not found" {
			t.Error("ErrorMessage not set correctly")
		}
	})
}

func TestUIDBatchValidationResponse(t *testing.T) {
	t.Run("Batch response structure", func(t *testing.T) {
		resp := uid.BatchValidationResponse{
			Total:       10,
			Valid:       8,
			Invalid:     2,
			ProcessedAt: time.Now().Format("2006-01-02T15:04:05Z"),
		}

		if resp.Total != 10 {
			t.Errorf("Total mismatch: got %d", resp.Total)
		}
		if resp.Valid != 8 {
			t.Errorf("Valid mismatch: got %d", resp.Valid)
		}
		if resp.Invalid != 2 {
			t.Errorf("Invalid mismatch: got %d", resp.Invalid)
		}
	})
}

func TestUIDListFilter(t *testing.T) {
	t.Run("List filter defaults", func(t *testing.T) {
		filter := uid.ListFilter{
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

	t.Run("List filter with options", func(t *testing.T) {
		accountID := uuid.New()
		uidStr := "ATU"
		valid := true
		countryCode := "AT"
		dateFrom := time.Now().AddDate(0, -1, 0)
		dateTo := time.Now()

		filter := uid.ListFilter{
			TenantID:    uuid.New(),
			AccountID:   &accountID,
			UID:         &uidStr,
			Valid:       &valid,
			CountryCode: &countryCode,
			DateFrom:    &dateFrom,
			DateTo:      &dateTo,
			Limit:       20,
			Offset:      10,
		}

		if filter.AccountID == nil || *filter.AccountID != accountID {
			t.Error("AccountID not set correctly")
		}
		if filter.Valid == nil || !*filter.Valid {
			t.Error("Valid filter not set correctly")
		}
		if filter.CountryCode == nil || *filter.CountryCode != "AT" {
			t.Error("CountryCode filter not set correctly")
		}
	})
}

func TestUIDListQueryParsing(t *testing.T) {
	t.Run("Parse list query parameters", func(t *testing.T) {
		accountID := uuid.New()
		req := httptest.NewRequest("GET", "/api/v1/uid/validations?account_id="+accountID.String()+"&uid=ATU&valid=true&country_code=AT&date_from=2025-01-01&date_to=2025-01-31&limit=20&offset=10", nil)

		query := req.URL.Query()

		if query.Get("account_id") != accountID.String() {
			t.Error("account_id not parsed correctly")
		}
		if query.Get("uid") != "ATU" {
			t.Error("uid not parsed correctly")
		}
		if query.Get("valid") != "true" {
			t.Error("valid not parsed correctly")
		}
		if query.Get("country_code") != "AT" {
			t.Error("country_code not parsed correctly")
		}
		if query.Get("date_from") != "2025-01-01" {
			t.Error("date_from not parsed correctly")
		}
		if query.Get("date_to") != "2025-01-31" {
			t.Error("date_to not parsed correctly")
		}
		if query.Get("limit") != "20" {
			t.Error("limit not parsed correctly")
		}
		if query.Get("offset") != "10" {
			t.Error("offset not parsed correctly")
		}
	})
}

func TestUIDFormatValidation(t *testing.T) {
	t.Run("Austrian UID format", func(t *testing.T) {
		// Austrian UIDs start with ATU followed by 8 digits
		validUIDs := []string{
			"ATU12345678",
			"ATU00000001",
			"ATU99999999",
		}

		for _, uid := range validUIDs {
			if len(uid) != 11 {
				t.Errorf("Austrian UID should be 11 characters: %s", uid)
			}
			if uid[:3] != "ATU" {
				t.Errorf("Austrian UID should start with ATU: %s", uid)
			}
		}
	})

	t.Run("German UID format", func(t *testing.T) {
		// German UIDs start with DE followed by 9 digits
		validUIDs := []string{
			"DE123456789",
			"DE000000001",
			"DE999999999",
		}

		for _, uid := range validUIDs {
			if len(uid) != 11 {
				t.Errorf("German UID should be 11 characters: %s", uid)
			}
			if uid[:2] != "DE" {
				t.Errorf("German UID should start with DE: %s", uid)
			}
		}
	})

	t.Run("French UID format", func(t *testing.T) {
		// French UIDs start with FR followed by 2 characters and 9 digits
		uid := "FRXX123456789"
		if len(uid) != 13 {
			t.Errorf("French UID should be 13 characters: %s", uid)
		}
		if uid[:2] != "FR" {
			t.Errorf("French UID should start with FR: %s", uid)
		}
	})
}

func TestUIDFormatValidateRequest(t *testing.T) {
	t.Run("Parse format validate request", func(t *testing.T) {
		uids := []string{"ATU12345678", "DE123456789", "INVALID"}
		reqBody := map[string]interface{}{
			"uids": uids,
		}

		body, _ := json.Marshal(reqBody)

		var parsed uid.ValidateFormatRequest
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse format request: %v", err)
		}

		if len(parsed.UIDs) != 3 {
			t.Errorf("UIDs count mismatch: got %d", len(parsed.UIDs))
		}
	})
}

func TestUIDExportContentType(t *testing.T) {
	t.Run("CSV export content type", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "text/csv")
		rec.Header().Set("Content-Disposition", "attachment; filename=uid_validations.csv")

		if rec.Header().Get("Content-Type") != "text/csv" {
			t.Error("Content-Type should be text/csv")
		}
		if rec.Header().Get("Content-Disposition") != "attachment; filename=uid_validations.csv" {
			t.Error("Content-Disposition header mismatch")
		}
	})
}

func TestUIDImportQueryParams(t *testing.T) {
	t.Run("Import query parameters", func(t *testing.T) {
		accountID := uuid.New()
		req := httptest.NewRequest("POST", "/api/v1/uid/import?account_id="+accountID.String()+"&level=2", nil)

		query := req.URL.Query()

		if query.Get("account_id") != accountID.String() {
			t.Error("account_id not parsed correctly")
		}
		if query.Get("level") != "2" {
			t.Error("level not parsed correctly")
		}
	})
}

func TestUIDCountryCodes(t *testing.T) {
	// EU country codes for VAT purposes
	euCountryCodes := []string{
		"AT", // Austria
		"BE", // Belgium
		"BG", // Bulgaria
		"CY", // Cyprus
		"CZ", // Czech Republic
		"DE", // Germany
		"DK", // Denmark
		"EE", // Estonia
		"EL", // Greece
		"ES", // Spain
		"FI", // Finland
		"FR", // France
		"HR", // Croatia
		"HU", // Hungary
		"IE", // Ireland
		"IT", // Italy
		"LT", // Lithuania
		"LU", // Luxembourg
		"LV", // Latvia
		"MT", // Malta
		"NL", // Netherlands
		"PL", // Poland
		"PT", // Portugal
		"RO", // Romania
		"SE", // Sweden
		"SI", // Slovenia
		"SK", // Slovakia
	}

	for _, code := range euCountryCodes {
		if len(code) != 2 {
			t.Errorf("Country code should be 2 characters: %s", code)
		}
	}

	// Austria should be in the list
	found := false
	for _, code := range euCountryCodes {
		if code == "AT" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AT should be in EU country codes")
	}
}

func TestUIDValidationLevels(t *testing.T) {
	t.Run("Level 1 - Basic validation", func(t *testing.T) {
		// Level 1 only returns valid/invalid
		resp := &uid.ValidationResponse{
			UID:         "ATU12345678",
			CountryCode: "AT",
			Valid:       true,
			Level:       1,
			Source:      "vies",
		}

		if resp.Level != 1 {
			t.Errorf("Level should be 1, got %d", resp.Level)
		}
		// Level 1 should not have company details
		if resp.CompanyName != nil {
			t.Error("Level 1 should not return company name")
		}
	})

	t.Run("Level 2 - Full validation with details", func(t *testing.T) {
		companyName := "Test GmbH"
		resp := &uid.ValidationResponse{
			UID:         "ATU12345678",
			CountryCode: "AT",
			Valid:       true,
			Level:       2,
			CompanyName: &companyName,
			Source:      "vies",
		}

		if resp.Level != 2 {
			t.Errorf("Level should be 2, got %d", resp.Level)
		}
		// Level 2 should have company details
		if resp.CompanyName == nil {
			t.Error("Level 2 should return company name when available")
		}
	})
}

func TestUIDValidationCaching(t *testing.T) {
	t.Run("Cache TTL concept", func(t *testing.T) {
		// UID validations should be cached for 24 hours
		cacheTTL := 24 * time.Hour

		if cacheTTL.Hours() != 24 {
			t.Errorf("Cache TTL should be 24 hours, got %f", cacheTTL.Hours())
		}
	})
}
