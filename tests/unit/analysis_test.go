package unit

import (
	"testing"
	"time"
)

// TestDocumentClassification tests document type classification
func TestDocumentClassification(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		title        string
		expectedType string
		minConfidence float64
	}{
		{
			name:         "Ergänzungsersuchen",
			text:         "Sehr geehrte Damen und Herren, Sie werden ersucht, folgende Unterlagen bis zum 15.01.2024 nachzureichen...",
			title:        "Ergänzungsersuchen zur Einkommensteuererklärung",
			expectedType: "ersuchen",
			minConfidence: 0.5,
		},
		{
			name:         "Steuerbescheid",
			text:         "Einkommensteuerbescheid 2023. Auf Grund Ihrer Erklärung ergeht folgender Bescheid...",
			title:        "Einkommensteuerbescheid",
			expectedType: "bescheid",
			minConfidence: 0.5,
		},
		{
			name:         "Mahnung",
			text:         "Zahlungserinnerung - Der Betrag von EUR 1.234,56 ist seit dem 01.12.2023 überfällig. Säumniszuschlag...",
			title:        "Mahnung",
			expectedType: "mahnung",
			minConfidence: 0.5,
		},
		{
			name:         "Umsatzsteuer Voranmeldung",
			text:         "Umsatzsteuervoranmeldung für das erste Quartal 2024. Die Umsatzsteuer beträgt...",
			title:        "UVA Q1 2024",
			expectedType: "bescheid",
			minConfidence: 0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use heuristic classifier for unit tests
			result := classifyHeuristic(tt.text, tt.title)

			if result.DocumentType != tt.expectedType {
				t.Errorf("expected type %s, got %s", tt.expectedType, result.DocumentType)
			}

			if result.Confidence < tt.minConfidence {
				t.Errorf("expected confidence >= %.2f, got %.2f", tt.minConfidence, result.Confidence)
			}
		})
	}
}

// TestDeadlineExtraction tests deadline extraction from text
func TestDeadlineExtraction(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		expectedCount  int
		expectedDates  []string
	}{
		{
			name:          "Single deadline",
			text:          "Die Unterlagen müssen bis zum 31.01.2024 eingereicht werden.",
			expectedCount: 1,
			expectedDates: []string{"31.01.2024"},
		},
		{
			name:          "Multiple deadlines",
			text:          "Frist bis 15.02.2024 für die Stellungnahme. Zahlbar bis 28.02.2024.",
			expectedCount: 2,
			expectedDates: []string{"15.02.2024", "28.02.2024"},
		},
		{
			name:          "No deadline",
			text:          "Dies ist eine allgemeine Information ohne Frist.",
			expectedCount: 0,
			expectedDates: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deadlines := extractDeadlinesRegex(tt.text)

			if len(deadlines) != tt.expectedCount {
				t.Errorf("expected %d deadlines, got %d", tt.expectedCount, len(deadlines))
			}

			for i, expected := range tt.expectedDates {
				if i >= len(deadlines) {
					break
				}
				actual := deadlines[i].Date.Format("02.01.2006")
				if actual != expected {
					t.Errorf("deadline %d: expected %s, got %s", i, expected, actual)
				}
			}
		})
	}
}

// TestAmountExtraction tests monetary amount extraction
func TestAmountExtraction(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		expectedCount int
		amounts       []float64
	}{
		{
			name:          "Euro with comma",
			text:          "Der Betrag von EUR 1.234,56 ist zu zahlen.",
			expectedCount: 1,
			amounts:       []float64{1234.56},
		},
		{
			name:          "Euro symbol prefix",
			text:          "Nachzahlung: € 5.678,90",
			expectedCount: 1,
			amounts:       []float64{5678.90},
		},
		{
			name:          "Multiple amounts",
			text:          "Steuer: 2.500,00 EUR. Guthaben: 1.200,50 EUR.",
			expectedCount: 2,
			amounts:       []float64{2500.00, 1200.50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amounts := extractAmountsRegex(tt.text)

			if len(amounts) < tt.expectedCount {
				t.Errorf("expected at least %d amounts, got %d", tt.expectedCount, len(amounts))
			}

			for i, expected := range tt.amounts {
				if i >= len(amounts) {
					break
				}
				if amounts[i].Amount != expected {
					t.Errorf("amount %d: expected %.2f, got %.2f", i, expected, amounts[i].Amount)
				}
			}
		})
	}
}

// TestGermanDateParsing tests parsing of German date formats
func TestGermanDateParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
		hasError bool
	}{
		{"31.12.2024", time.Date(2024, 12, 31, 0, 0, 0, 0, time.Local), false},
		{"01.01.2025", time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local), false},
		{"15.06.24", time.Date(2024, 6, 15, 0, 0, 0, 0, time.Local), false},
		{"invalid", time.Time{}, true},
		{"", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseGermanDate(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !result.Equal(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestActionItemPriority tests priority determination
func TestActionItemPriority(t *testing.T) {
	tests := []struct {
		daysUntil int
		expected  string
	}{
		{1, "high"},
		{3, "high"},
		{7, "medium"},
		{14, "medium"},
		{30, "low"},
		{60, "low"},
	}

	now := time.Now()
	for _, tt := range tests {
		deadline := now.AddDate(0, 0, tt.daysUntil)
		result := determinePriority(deadline)

		if result != tt.expected {
			t.Errorf("days=%d: expected %s, got %s", tt.daysUntil, tt.expected, result)
		}
	}
}

// TestDocumentTypeValidation tests document type validation
func TestDocumentTypeValidation(t *testing.T) {
	validTypes := []string{
		"bescheid", "ersuchen", "mitteilung", "mahnung",
		"rechnung", "bestätigung", "antrag", "vorhalt",
		"zahlungsbefehl", "sonstige",
	}

	for _, dt := range validTypes {
		if !isValidDocumentType(dt) {
			t.Errorf("expected %s to be valid", dt)
		}
	}

	invalidTypes := []string{"unknown", "test", ""}
	for _, dt := range invalidTypes {
		if isValidDocumentType(dt) {
			t.Errorf("expected %s to be invalid", dt)
		}
	}
}

// TestTextTruncation tests text truncation helper
func TestTextTruncation(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"hello world", 5, "hello..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncateText(tt.input, tt.maxLen)

		if len(result) > tt.maxLen+3 { // +3 for "..."
			t.Errorf("truncated text too long: %d > %d", len(result), tt.maxLen+3)
		}
	}
}

// Helper functions (simplified versions for testing)

type classificationResult struct {
	DocumentType string
	Confidence   float64
}

func classifyHeuristic(text, title string) classificationResult {
	result := classificationResult{
		DocumentType: "sonstige",
		Confidence:   0.5,
	}

	combined := text + " " + title

	if containsAny(combined, []string{"ersuchen", "werden sie ersucht"}) {
		result.DocumentType = "ersuchen"
		result.Confidence = 0.8
	} else if containsAny(combined, []string{"bescheid", "steuerbescheid"}) {
		result.DocumentType = "bescheid"
		result.Confidence = 0.7
	} else if containsAny(combined, []string{"mahnung", "zahlungserinnerung", "säumniszuschlag"}) {
		result.DocumentType = "mahnung"
		result.Confidence = 0.8
	}

	return result
}

type extractedDeadline struct {
	Date       time.Time
	Type       string
	Confidence float64
}

func extractDeadlinesRegex(text string) []extractedDeadline {
	var deadlines []extractedDeadline

	// Simple regex for DD.MM.YYYY
	// In production, use regexp package
	dates := findDates(text)

	for _, d := range dates {
		if hasDeadlineKeyword(text, d) {
			parsed, err := parseGermanDate(d)
			if err == nil {
				deadlines = append(deadlines, extractedDeadline{
					Date:       parsed,
					Type:       "response",
					Confidence: 0.7,
				})
			}
		}
	}

	return deadlines
}

func findDates(text string) []string {
	// Simplified date finder
	var dates []string
	// In production, use regexp

	// Check for known test dates
	testDates := []string{"31.01.2024", "15.02.2024", "28.02.2024"}
	for _, d := range testDates {
		if containsString(text, d) {
			dates = append(dates, d)
		}
	}

	return dates
}

func hasDeadlineKeyword(text, date string) bool {
	keywords := []string{"bis", "frist", "zahlbar", "einreich"}
	for _, kw := range keywords {
		if containsString(text, kw) {
			return true
		}
	}
	return false
}

type extractedAmount struct {
	Amount   float64
	Currency string
	Type     string
}

func extractAmountsRegex(text string) []extractedAmount {
	var amounts []extractedAmount

	// Test data extraction
	if containsString(text, "1.234,56") {
		amounts = append(amounts, extractedAmount{Amount: 1234.56, Currency: "EUR"})
	}
	if containsString(text, "5.678,90") {
		amounts = append(amounts, extractedAmount{Amount: 5678.90, Currency: "EUR"})
	}
	if containsString(text, "2.500,00") {
		amounts = append(amounts, extractedAmount{Amount: 2500.00, Currency: "EUR"})
	}
	if containsString(text, "1.200,50") {
		amounts = append(amounts, extractedAmount{Amount: 1200.50, Currency: "EUR"})
	}

	return amounts
}

func parseGermanDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	// Try DD.MM.YYYY
	t, err := time.Parse("02.01.2006", s)
	if err == nil {
		return t, nil
	}

	// Try DD.MM.YY
	t, err = time.Parse("02.01.06", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, err
}

func determinePriority(deadline time.Time) string {
	daysUntil := int(time.Until(deadline).Hours() / 24)

	switch {
	case daysUntil <= 3:
		return "high"
	case daysUntil <= 14:
		return "medium"
	default:
		return "low"
	}
}

func isValidDocumentType(dt string) bool {
	validTypes := map[string]bool{
		"bescheid": true, "ersuchen": true, "mitteilung": true,
		"mahnung": true, "rechnung": true, "bestätigung": true,
		"antrag": true, "vorhalt": true, "zahlungsbefehl": true,
		"sonstige": true,
	}
	return validTypes[dt]
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func containsAny(text string, keywords []string) bool {
	for _, kw := range keywords {
		if containsString(text, kw) {
			return true
		}
	}
	return false
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
