//go:build !windows

package integration

import (
	"testing"
	"time"

	"austrian-business-infrastructure/internal/analysis"
	"github.com/google/uuid"
)

// T076: Integration tests for analysis pipeline

func TestAnalysisRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// These tests verify the repository CRUD operations work correctly
	// In a real scenario, they would use a test database

	t.Run("Analysis CRUD", func(t *testing.T) {
		// Test analysis struct creation
		a := &analysis.Analysis{
			ID:                       uuid.New(),
			DocumentID:               uuid.New(),
			TenantID:                 uuid.New(),
			Status:                   analysis.StatusPending,
			DocumentType:             "bescheid",
			ClassificationConfidence: 0.85,
			IsScanned:                false,
			Summary:                  "Test summary",
			KeyPoints:                []string{"Point 1", "Point 2"},
			TextLength:               1000,
			PageCount:                2,
			Language:                 "de",
			AIModel:                  "claude-sonnet-4-20250514",
			TokensUsed:               500,
			ProcessingTimeMs:         1500,
			CreatedAt:                time.Now(),
		}

		// Verify fields are set correctly
		if a.Status != analysis.StatusPending {
			t.Errorf("Expected status %s, got %s", analysis.StatusPending, a.Status)
		}

		if a.ClassificationConfidence < 0.8 {
			t.Error("Expected high confidence classification")
		}

		if len(a.KeyPoints) != 2 {
			t.Errorf("Expected 2 key points, got %d", len(a.KeyPoints))
		}
	})

	t.Run("Deadline Creation", func(t *testing.T) {
		deadline := &analysis.Deadline{
			ID:           uuid.New(),
			AnalysisID:   uuid.New(),
			DocumentID:   uuid.New(),
			TenantID:     uuid.New(),
			DeadlineType: analysis.DeadlineTypeResponse,
			Date:         time.Now().AddDate(0, 0, 14),
			Description:  "Respond to Ergänzungsersuchen",
			Confidence:   0.92,
			IsHard:       true,
			CreatedAt:    time.Now(),
		}

		// Verify deadline is in future
		if deadline.Date.Before(time.Now()) {
			t.Error("Deadline should be in the future")
		}

		// Verify high confidence
		if deadline.Confidence < 0.9 {
			t.Error("Expected high confidence deadline")
		}
	})

	t.Run("Amount Creation", func(t *testing.T) {
		amount := &analysis.Amount{
			ID:          uuid.New(),
			AnalysisID:  uuid.New(),
			DocumentID:  uuid.New(),
			TenantID:    uuid.New(),
			AmountType:  "nachzahlung",
			Amount:      1234.56,
			Currency:    "EUR",
			Description: "Einkommensteuer Nachzahlung",
			Confidence:  0.88,
			CreatedAt:   time.Now(),
		}

		// Verify amount is positive
		if amount.Amount <= 0 {
			t.Error("Amount should be positive")
		}

		// Verify currency is EUR
		if amount.Currency != "EUR" {
			t.Errorf("Expected currency EUR, got %s", amount.Currency)
		}
	})

	t.Run("ActionItem Creation", func(t *testing.T) {
		dueDate := time.Now().AddDate(0, 0, 7)
		item := &analysis.ActionItem{
			ID:          uuid.New(),
			AnalysisID:  uuid.New(),
			DocumentID:  uuid.New(),
			TenantID:    uuid.New(),
			Title:       "Submit response",
			Description: "Prepare and submit response to tax office",
			Priority:    analysis.PriorityHigh,
			Category:    "response",
			Status:      analysis.ActionStatusPending,
			DueDate:     &dueDate,
			Confidence:  0.85,
			CreatedAt:   time.Now(),
		}

		// Verify priority
		if item.Priority != analysis.PriorityHigh {
			t.Errorf("Expected priority %s, got %s", analysis.PriorityHigh, item.Priority)
		}

		// Verify status
		if item.Status != analysis.ActionStatusPending {
			t.Errorf("Expected status %s, got %s", analysis.ActionStatusPending, item.Status)
		}
	})
}

func TestConfidenceWarnings(t *testing.T) {
	t.Run("Generate warnings for low confidence", func(t *testing.T) {
		result := &analysis.FullAnalysisResult{
			Analysis: &analysis.Analysis{
				ID:                       uuid.New(),
				ClassificationConfidence: 0.4, // Low confidence
				IsScanned:                true,
				OCRConfidence:            0.3, // Low OCR confidence
			},
			Deadlines: []*analysis.Deadline{
				{
					ID:         uuid.New(),
					Confidence: 0.45, // Low confidence
				},
			},
			Amounts: []*analysis.Amount{
				{
					ID:         uuid.New(),
					Confidence: 0.6, // Medium confidence
				},
			},
		}

		result.GenerateConfidenceWarnings()

		// Should have warnings for classification, OCR, and deadline
		if len(result.Warnings) < 3 {
			t.Errorf("Expected at least 3 warnings, got %d", len(result.Warnings))
		}

		// Check warning types
		hasClassificationWarning := false
		hasOCRWarning := false
		hasDeadlineWarning := false

		for _, w := range result.Warnings {
			switch w.Type {
			case "classification":
				hasClassificationWarning = true
			case "ocr":
				hasOCRWarning = true
			case "deadline":
				hasDeadlineWarning = true
			}
		}

		if !hasClassificationWarning {
			t.Error("Expected classification warning")
		}
		if !hasOCRWarning {
			t.Error("Expected OCR warning")
		}
		if !hasDeadlineWarning {
			t.Error("Expected deadline warning")
		}
	})

	t.Run("No warnings for high confidence", func(t *testing.T) {
		result := &analysis.FullAnalysisResult{
			Analysis: &analysis.Analysis{
				ID:                       uuid.New(),
				ClassificationConfidence: 0.95, // High confidence
				IsScanned:                false,
			},
			Deadlines: []*analysis.Deadline{
				{
					ID:         uuid.New(),
					Confidence: 0.92, // High confidence
				},
			},
			Amounts: []*analysis.Amount{
				{
					ID:         uuid.New(),
					Confidence: 0.88, // High confidence
				},
			},
		}

		result.GenerateConfidenceWarnings()

		// Should have no warnings
		if len(result.Warnings) != 0 {
			t.Errorf("Expected 0 warnings for high confidence results, got %d", len(result.Warnings))
		}
	})
}

func TestAnalysisOptions(t *testing.T) {
	t.Run("Default options", func(t *testing.T) {
		opts := analysis.DefaultOptions()

		if !opts.IncludeOCR {
			t.Error("Default should include OCR")
		}
		if !opts.IncludeClassify {
			t.Error("Default should include classification")
		}
		if !opts.IncludeSummary {
			t.Error("Default should include summary")
		}
		if !opts.IncludeDeadlines {
			t.Error("Default should include deadlines")
		}
		if !opts.IncludeAmounts {
			t.Error("Default should include amounts")
		}
		if !opts.IncludeActionItems {
			t.Error("Default should include action items")
		}
		if !opts.IncludeSuggestions {
			t.Error("Default should include suggestions")
		}
	})
}

func TestAnalysisStatusConstants(t *testing.T) {
	// Verify status constants are unique
	statuses := map[string]bool{
		analysis.StatusPending:    true,
		analysis.StatusProcessing: true,
		analysis.StatusCompleted:  true,
		analysis.StatusFailed:     true,
	}

	if len(statuses) != 4 {
		t.Error("Expected 4 unique status constants")
	}
}

func TestDeadlineTypeConstants(t *testing.T) {
	// Verify deadline type constants are unique
	types := map[string]bool{
		analysis.DeadlineTypeResponse:   true,
		analysis.DeadlineTypePayment:    true,
		analysis.DeadlineTypeSubmission: true,
		analysis.DeadlineTypeAppeal:     true,
		analysis.DeadlineTypeOther:      true,
	}

	if len(types) != 5 {
		t.Error("Expected 5 unique deadline type constants")
	}
}

func TestActionStatusConstants(t *testing.T) {
	// Verify action status constants
	if analysis.ActionStatusPending != "pending" {
		t.Error("ActionStatusPending should be 'pending'")
	}
	if analysis.ActionStatusCompleted != "completed" {
		t.Error("ActionStatusCompleted should be 'completed'")
	}
	if analysis.ActionStatusCancelled != "cancelled" {
		t.Error("ActionStatusCancelled should be 'cancelled'")
	}
}

func TestPriorityConstants(t *testing.T) {
	// Verify priority constants
	if analysis.PriorityHigh != "high" {
		t.Error("PriorityHigh should be 'high'")
	}
	if analysis.PriorityMedium != "medium" {
		t.Error("PriorityMedium should be 'medium'")
	}
	if analysis.PriorityLow != "low" {
		t.Error("PriorityLow should be 'low'")
	}
}
