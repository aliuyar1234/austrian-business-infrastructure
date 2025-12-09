package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ClassificationResponse represents a document classification result
type ClassificationResponse struct {
	DocumentType    string  `json:"document_type"`
	DocumentSubtype string  `json:"document_subtype"`
	Priority        string  `json:"priority"`
	Confidence      float64 `json:"confidence"`
	Reasoning       string  `json:"reasoning"`
}

// DeadlineResponse represents deadline extraction results
type DeadlineResponse struct {
	Deadlines []ExtractedDeadline `json:"deadlines"`
}

// ExtractedDeadline represents a single extracted deadline
type ExtractedDeadline struct {
	DeadlineType    string  `json:"deadline_type"`
	DeadlineDate    string  `json:"deadline_date"` // YYYY-MM-DD
	CalculationRule string  `json:"calculation_rule"`
	SourceText      string  `json:"source_text"`
	Confidence      float64 `json:"confidence"`
}

// SummaryResponse represents document summarization results
type SummaryResponse struct {
	Summary        string   `json:"summary"`
	KeyPoints      []string `json:"key_points"`
	ActionRequired bool     `json:"action_required"`
	Tone           string   `json:"tone"`
}

// AmountResponse represents amount extraction results
type AmountResponse struct {
	Amounts []ExtractedAmount `json:"amounts"`
}

// ExtractedAmount represents a single extracted amount
type ExtractedAmount struct {
	AmountType string  `json:"amount_type"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	IsNegative bool    `json:"is_negative"`
	Label      string  `json:"label"`
	SourceText string  `json:"source_text"`
	Confidence float64 `json:"confidence"`
}

// SuggestionResponse represents response suggestion results
type SuggestionResponse struct {
	SuggestionText    string   `json:"suggestion_text"`
	KeyPoints         []string `json:"key_points"`
	RequiredDocuments []string `json:"required_documents"`
	Confidence        float64  `json:"confidence"`
	Warnings          []string `json:"warnings"`
}

// ParseClassification parses a classification response from Claude
func ParseClassification(text string) (*ClassificationResponse, error) {
	jsonStr := extractJSON(text)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var resp ClassificationResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("parse classification JSON: %w", err)
	}

	// Validate
	if err := validateClassification(&resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ParseDeadlines parses deadline extraction response from Claude
func ParseDeadlines(text string) (*DeadlineResponse, error) {
	jsonStr := extractJSON(text)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var resp DeadlineResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("parse deadline JSON: %w", err)
	}

	// Validate each deadline
	for i, d := range resp.Deadlines {
		if err := validateDeadline(&d); err != nil {
			return nil, fmt.Errorf("deadline %d: %w", i, err)
		}
	}

	return &resp, nil
}

// ParseSummary parses a summary response from Claude
func ParseSummary(text string) (*SummaryResponse, error) {
	jsonStr := extractJSON(text)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var resp SummaryResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("parse summary JSON: %w", err)
	}

	return &resp, nil
}

// ParseAmounts parses amount extraction response from Claude
func ParseAmounts(text string) (*AmountResponse, error) {
	jsonStr := extractJSON(text)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var resp AmountResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("parse amount JSON: %w", err)
	}

	// Validate each amount
	for i, a := range resp.Amounts {
		if err := validateAmount(&a); err != nil {
			return nil, fmt.Errorf("amount %d: %w", i, err)
		}
	}

	return &resp, nil
}

// ParseSuggestion parses a suggestion response from Claude
func ParseSuggestion(text string) (*SuggestionResponse, error) {
	jsonStr := extractJSON(text)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var resp SuggestionResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("parse suggestion JSON: %w", err)
	}

	return &resp, nil
}

// extractJSON extracts JSON from text that might have markdown formatting
func extractJSON(text string) string {
	// Try to find JSON in markdown code blocks first
	codeBlockPattern := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)```")
	matches := codeBlockPattern.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Try to find raw JSON (object or array)
	text = strings.TrimSpace(text)

	// Find JSON object
	startObj := strings.Index(text, "{")
	endObj := strings.LastIndex(text, "}")
	if startObj != -1 && endObj > startObj {
		return text[startObj : endObj+1]
	}

	// Find JSON array
	startArr := strings.Index(text, "[")
	endArr := strings.LastIndex(text, "]")
	if startArr != -1 && endArr > startArr {
		return text[startArr : endArr+1]
	}

	return ""
}

// Validation functions

func validateClassification(c *ClassificationResponse) error {
	validTypes := map[string]bool{
		"bescheid": true, "ersuchen": true, "info": true,
		"rechnung": true, "mahnung": true, "sonstige": true,
	}
	if !validTypes[c.DocumentType] {
		return fmt.Errorf("invalid document_type: %s", c.DocumentType)
	}

	validPriorities := map[string]bool{
		"critical": true, "high": true, "medium": true, "low": true,
	}
	if !validPriorities[c.Priority] {
		return fmt.Errorf("invalid priority: %s", c.Priority)
	}

	if c.Confidence < 0 || c.Confidence > 1 {
		return fmt.Errorf("confidence out of range: %f", c.Confidence)
	}

	return nil
}

func validateDeadline(d *ExtractedDeadline) error {
	validTypes := map[string]bool{
		"response": true, "payment": true, "appeal": true,
		"submission": true, "other": true,
	}
	if !validTypes[d.DeadlineType] {
		return fmt.Errorf("invalid deadline_type: %s", d.DeadlineType)
	}

	// Validate date format YYYY-MM-DD
	datePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !datePattern.MatchString(d.DeadlineDate) {
		return fmt.Errorf("invalid date format: %s", d.DeadlineDate)
	}

	if d.Confidence < 0 || d.Confidence > 1 {
		return fmt.Errorf("confidence out of range: %f", d.Confidence)
	}

	return nil
}

func validateAmount(a *ExtractedAmount) error {
	validTypes := map[string]bool{
		"tax_due": true, "refund": true, "penalty": true,
		"fee": true, "total": true, "other": true,
	}
	if !validTypes[a.AmountType] {
		return fmt.Errorf("invalid amount_type: %s", a.AmountType)
	}

	if a.Currency == "" {
		a.Currency = "EUR"
	}

	if a.Confidence < 0 || a.Confidence > 1 {
		return fmt.Errorf("confidence out of range: %f", a.Confidence)
	}

	return nil
}

// ConfidenceLevel returns a human-readable confidence level
func ConfidenceLevel(confidence float64) string {
	switch {
	case confidence >= 0.8:
		return "high"
	case confidence >= 0.5:
		return "medium"
	default:
		return "low"
	}
}

// NeedsReview returns true if the confidence is below the review threshold
func NeedsReview(confidence float64) bool {
	return confidence < 0.8
}
