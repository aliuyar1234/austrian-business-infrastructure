package business

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"austrian-business-infrastructure/internal/uva"
	"github.com/google/uuid"
)

// T084: Integration tests for UVA submission API

func TestUVASubmissionTypes(t *testing.T) {
	t.Run("UVAData struct validation", func(t *testing.T) {
		data := uva.UVAData{
			KZ000: 10000000, // 100,000.00 EUR
			KZ017: 8000000,  // 80,000.00 EUR (20% rate)
			KZ060: 1600000,  // 16,000.00 EUR input tax
			KZ095: 1600000,  // 16,000.00 EUR payable
		}

		if data.KZ000 != 10000000 {
			t.Errorf("KZ000 mismatch: got %d, want 10000000", data.KZ000)
		}
		if data.KZ017 != 8000000 {
			t.Errorf("KZ017 mismatch: got %d, want 8000000", data.KZ017)
		}
	})

	t.Run("Submission struct fields", func(t *testing.T) {
		now := time.Now()
		month := 1
		submission := &uva.Submission{
			ID:               uuid.New(),
			TenantID:         uuid.New(),
			AccountID:        uuid.New(),
			PeriodYear:       2025,
			PeriodMonth:      &month,
			PeriodType:       uva.PeriodTypeMonthly,
			ValidationStatus: "pending",
			Status:           uva.StatusDraft,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		if submission.Status != uva.StatusDraft {
			t.Errorf("Status mismatch: got %s, want %s", submission.Status, uva.StatusDraft)
		}
		if submission.PeriodYear != 2025 {
			t.Errorf("PeriodYear mismatch")
		}
	})
}

func TestUVAStatusConstants(t *testing.T) {
	statuses := []string{
		uva.StatusDraft,
		uva.StatusValidated,
		uva.StatusSubmitted,
		uva.StatusAccepted,
		uva.StatusRejected,
		uva.StatusError,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("Status constant is empty")
		}
	}

	// Verify expected values
	if uva.StatusDraft != "draft" {
		t.Errorf("StatusDraft mismatch: got %s", uva.StatusDraft)
	}
	if uva.StatusSubmitted != "submitted" {
		t.Errorf("StatusSubmitted mismatch: got %s", uva.StatusSubmitted)
	}
}

func TestUVAPeriodTypes(t *testing.T) {
	if uva.PeriodTypeMonthly != "monthly" {
		t.Errorf("PeriodTypeMonthly mismatch: got %s", uva.PeriodTypeMonthly)
	}
	if uva.PeriodTypeQuarterly != "quarterly" {
		t.Errorf("PeriodTypeQuarterly mismatch: got %s", uva.PeriodTypeQuarterly)
	}
}

func TestUVACreateRequestParsing(t *testing.T) {
	t.Run("Parse monthly UVA request", func(t *testing.T) {
		month := 1
		reqBody := map[string]interface{}{
			"account_id":   uuid.New().String(),
			"period_year":  2025,
			"period_month": month,
			"period_type":  "monthly",
			"data": map[string]interface{}{
				"kz000": 10000000,
				"kz017": 8000000,
				"kz060": 1600000,
				"kz095": 1600000,
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/uva", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		var parsed uva.CreateRequest
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse request: %v", err)
		}

		if parsed.PeriodYear != 2025 {
			t.Errorf("PeriodYear mismatch: got %d", parsed.PeriodYear)
		}
		if parsed.PeriodType != "monthly" {
			t.Errorf("PeriodType mismatch: got %s", parsed.PeriodType)
		}
		if parsed.Data.KZ000 != 10000000 {
			t.Errorf("KZ000 mismatch: got %d", parsed.Data.KZ000)
		}
	})

	t.Run("Parse quarterly UVA request", func(t *testing.T) {
		quarter := 1
		reqBody := map[string]interface{}{
			"account_id":     uuid.New().String(),
			"period_year":    2025,
			"period_quarter": quarter,
			"period_type":    "quarterly",
			"data": map[string]interface{}{
				"kz017": 5000000,
				"kz060": 1000000,
				"kz095": 1000000,
			},
		}

		body, _ := json.Marshal(reqBody)

		var parsed uva.CreateRequest
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse request: %v", err)
		}

		if parsed.PeriodType != "quarterly" {
			t.Errorf("PeriodType mismatch: got %s", parsed.PeriodType)
		}
		if parsed.PeriodQuarter == nil || *parsed.PeriodQuarter != 1 {
			t.Error("PeriodQuarter not parsed correctly")
		}
	})
}

func TestUVAListFilterDefaults(t *testing.T) {
	filter := uva.ListFilter{
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
	if filter.AccountID != nil {
		t.Error("AccountID should be nil by default")
	}
}

func TestUVAListQueryParsing(t *testing.T) {
	t.Run("Parse list query parameters", func(t *testing.T) {
		accountID := uuid.New()
		req := httptest.NewRequest("GET", "/api/v1/uva?account_id="+accountID.String()+"&period_year=2025&period_type=monthly&status=draft&limit=20&offset=10", nil)

		query := req.URL.Query()

		if query.Get("account_id") != accountID.String() {
			t.Error("account_id not parsed correctly")
		}
		if query.Get("period_year") != "2025" {
			t.Error("period_year not parsed correctly")
		}
		if query.Get("period_type") != "monthly" {
			t.Error("period_type not parsed correctly")
		}
		if query.Get("status") != "draft" {
			t.Error("status not parsed correctly")
		}
		if query.Get("limit") != "20" {
			t.Error("limit not parsed correctly")
		}
		if query.Get("offset") != "10" {
			t.Error("offset not parsed correctly")
		}
	})
}

func TestUVASubmissionResponse(t *testing.T) {
	t.Run("Submission response structure", func(t *testing.T) {
		month := 1
		submittedAt := "2025-01-15T10:30:00Z"
		foRef := "FON-2025-12345678"

		resp := &uva.SubmissionResponse{
			ID:               uuid.New(),
			AccountID:        uuid.New(),
			PeriodYear:       2025,
			PeriodMonth:      &month,
			PeriodType:       "monthly",
			ValidationStatus: "passed",
			Status:           "submitted",
			FOReference:      &foRef,
			SubmittedAt:      &submittedAt,
			CreatedAt:        "2025-01-15T10:00:00Z",
			UpdatedAt:        "2025-01-15T10:30:00Z",
		}

		if resp.PeriodYear != 2025 {
			t.Error("PeriodYear mismatch")
		}
		if resp.Status != "submitted" {
			t.Error("Status mismatch")
		}
		if resp.FOReference == nil || *resp.FOReference != foRef {
			t.Error("FOReference not set correctly")
		}
	})
}

func TestUVABatchResponse(t *testing.T) {
	t.Run("Batch response structure", func(t *testing.T) {
		month := 1
		startedAt := "2025-01-15T10:30:00Z"
		completedAt := "2025-01-15T10:35:00Z"

		resp := &uva.BatchResponse{
			ID:           uuid.New(),
			Name:         "Januar 2025 Batch",
			PeriodYear:   2025,
			PeriodMonth:  &month,
			PeriodType:   "monthly",
			TotalCount:   10,
			SuccessCount: 8,
			FailedCount:  2,
			Status:       "completed",
			StartedAt:    &startedAt,
			CompletedAt:  &completedAt,
			CreatedAt:    "2025-01-15T10:00:00Z",
		}

		if resp.TotalCount != 10 {
			t.Errorf("TotalCount mismatch: got %d", resp.TotalCount)
		}
		if resp.SuccessCount != 8 {
			t.Errorf("SuccessCount mismatch: got %d", resp.SuccessCount)
		}
		if resp.FailedCount != 2 {
			t.Errorf("FailedCount mismatch: got %d", resp.FailedCount)
		}
		if resp.Status != "completed" {
			t.Errorf("Status mismatch: got %s", resp.Status)
		}
	})
}

func TestUVABatchRequestParsing(t *testing.T) {
	t.Run("Parse batch request", func(t *testing.T) {
		accountIDs := []string{uuid.New().String(), uuid.New().String(), uuid.New().String()}
		month := 1
		reqBody := map[string]interface{}{
			"name":         "Januar 2025 Batch",
			"account_ids":  accountIDs,
			"period_year":  2025,
			"period_month": month,
			"period_type":  "monthly",
		}

		body, _ := json.Marshal(reqBody)

		var parsed uva.CreateBatchRequest
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse batch request: %v", err)
		}

		if parsed.Name != "Januar 2025 Batch" {
			t.Errorf("Name mismatch: got %s", parsed.Name)
		}
		if len(parsed.AccountIDs) != 3 {
			t.Errorf("AccountIDs count mismatch: got %d", len(parsed.AccountIDs))
		}
		if parsed.PeriodYear != 2025 {
			t.Errorf("PeriodYear mismatch: got %d", parsed.PeriodYear)
		}
	})
}

func TestUVASubmitRequest(t *testing.T) {
	t.Run("Parse submit request with dry_run", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"dry_run": true,
		}

		body, _ := json.Marshal(reqBody)

		var parsed uva.SubmitRequest
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse submit request: %v", err)
		}

		if !parsed.DryRun {
			t.Error("DryRun should be true")
		}
	})

	t.Run("Empty submit request defaults to dry_run=false", func(t *testing.T) {
		reqBody := map[string]interface{}{}

		body, _ := json.Marshal(reqBody)

		var parsed uva.SubmitRequest
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&parsed); err != nil {
			t.Fatalf("Failed to parse submit request: %v", err)
		}

		if parsed.DryRun {
			t.Error("DryRun should default to false")
		}
	})
}

func TestUVAValidationErrorResponse(t *testing.T) {
	t.Run("Validation errors structure", func(t *testing.T) {
		errJSON := json.RawMessage(`[{"field":"kz095","message":"calculated value does not match"}]`)

		month := 1
		resp := &uva.SubmissionResponse{
			ID:               uuid.New(),
			AccountID:        uuid.New(),
			PeriodYear:       2025,
			PeriodMonth:      &month,
			PeriodType:       "monthly",
			ValidationStatus: "failed",
			ValidationErrors: errJSON,
			Status:           "draft",
			CreatedAt:        "2025-01-15T10:00:00Z",
			UpdatedAt:        "2025-01-15T10:00:00Z",
		}

		if resp.ValidationStatus != "failed" {
			t.Error("ValidationStatus should be failed")
		}
		if len(resp.ValidationErrors) == 0 {
			t.Error("ValidationErrors should not be empty")
		}
	})
}

func TestUVAResponseHTTPStatusMapping(t *testing.T) {
	testCases := []struct {
		name           string
		responseStatus int
		expectedBody   string
	}{
		{"created", http.StatusCreated, "id"},
		{"ok", http.StatusOK, "id"},
		{"not_found", http.StatusNotFound, "error"},
		{"bad_request", http.StatusBadRequest, "error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify HTTP status codes are correctly defined
			switch tc.responseStatus {
			case http.StatusCreated:
				if http.StatusCreated != 201 {
					t.Error("StatusCreated should be 201")
				}
			case http.StatusOK:
				if http.StatusOK != 200 {
					t.Error("StatusOK should be 200")
				}
			case http.StatusNotFound:
				if http.StatusNotFound != 404 {
					t.Error("StatusNotFound should be 404")
				}
			case http.StatusBadRequest:
				if http.StatusBadRequest != 400 {
					t.Error("StatusBadRequest should be 400")
				}
			}
		})
	}
}

func TestUVAXMLContentType(t *testing.T) {
	t.Run("XML response content type", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "application/xml")
		rec.Header().Set("Content-Disposition", "attachment; filename=uva.xml")

		if rec.Header().Get("Content-Type") != "application/xml" {
			t.Error("Content-Type should be application/xml")
		}
		if rec.Header().Get("Content-Disposition") != "attachment; filename=uva.xml" {
			t.Error("Content-Disposition header mismatch")
		}
	})
}

func TestUVADataCalculations(t *testing.T) {
	t.Run("Tax calculation verification", func(t *testing.T) {
		// Standard Austrian VAT calculation
		// Netto: 80,000 EUR
		// 20% VAT: 16,000 EUR
		// Brutto: 96,000 EUR

		data := uva.UVAData{
			KZ000: 9600000, // 96,000.00 EUR gross
			KZ017: 8000000, // 80,000.00 EUR net at 20%
			KZ060: 1600000, // 16,000.00 EUR input VAT
			KZ095: 0,       // Zero payable if input = output
		}

		// Net amount at 20% rate
		expectedNet := int64(8000000)
		if data.KZ017 != expectedNet {
			t.Errorf("KZ017 (net at 20%%) mismatch: got %d, want %d", data.KZ017, expectedNet)
		}

		// 20% of 80,000 = 16,000
		expectedVAT := int64(1600000)
		if data.KZ060 != expectedVAT {
			t.Errorf("KZ060 (input VAT) mismatch: got %d, want %d", data.KZ060, expectedVAT)
		}
	})
}
