package analysis

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/ai"
)

// Extractor handles extraction of structured data from documents
type Extractor struct {
	aiClient     *ai.Client
	promptLoader *ai.PromptLoader
}

// NewExtractor creates a new data extractor
func NewExtractor(aiClient *ai.Client, promptLoader *ai.PromptLoader) *Extractor {
	return &Extractor{
		aiClient:     aiClient,
		promptLoader: promptLoader,
	}
}

// ExtractedDeadline represents an extracted deadline
type ExtractedDeadline struct {
	Type        string    `json:"type"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	SourceText  string    `json:"source_text"`
	Confidence  float64   `json:"confidence"`
	IsHard      bool      `json:"is_hard"`
}

// ExtractDeadlines extracts deadlines from document text
func (e *Extractor) ExtractDeadlines(ctx context.Context, text string) ([]ExtractedDeadline, error) {
	// Load deadline extraction prompt
	prompt, err := e.promptLoader.Get(ctx, ai.PromptDeadline)
	var systemPrompt, userTemplate string
	if err != nil {
		systemPrompt = defaultDeadlinePrompt
		userTemplate = "Extract all deadlines from this Austrian official document:\n\n{document_text}\n\nToday's date is: {current_date}"
	} else {
		systemPrompt = prompt.SystemPrompt
		userTemplate = prompt.UserPromptTemplate
	}

	// Truncate text if needed (keep ~6000 chars for extraction)
	truncatedText := text
	if len(text) > 6000 {
		truncatedText = text[:6000] + "\n\n[Text truncated...]"
	}

	userPrompt := strings.ReplaceAll(userTemplate, "{document_text}", truncatedText)
	userPrompt = strings.ReplaceAll(userPrompt, "{current_date}", time.Now().Format("02.01.2006"))
	userPrompt = strings.ReplaceAll(userPrompt, "{delivery_date}", "unbekannt")

	response, err := e.aiClient.CompleteWithRetry(ctx, systemPrompt, userPrompt, 0.1, 2)
	if err != nil {
		return nil, fmt.Errorf("AI deadline extraction failed: %w", err)
	}

	parsed, err := ai.ParseDeadlines(response.GetText())
	if err != nil {
		// Fall back to regex extraction
		return e.extractDeadlinesRegex(text), nil
	}

	var deadlines []ExtractedDeadline
	for _, d := range parsed.Deadlines {
		deadline := ExtractedDeadline{
			Type:        d.DeadlineType,
			Description: d.CalculationRule,
			SourceText:  d.SourceText,
			Confidence:  d.Confidence,
			IsHard:      true, // Default to hard deadline
		}

		// Parse date (from YYYY-MM-DD format)
		if date, err := time.Parse("2006-01-02", d.DeadlineDate); err == nil {
			deadline.Date = date
		} else if date, err := parseGermanDate(d.DeadlineDate); err == nil {
			deadline.Date = date
		} else {
			continue // Skip if date can't be parsed
		}

		deadlines = append(deadlines, deadline)
	}

	return deadlines, nil
}

// extractDeadlinesRegex uses regex patterns to find deadlines
func (e *Extractor) extractDeadlinesRegex(text string) []ExtractedDeadline {
	var deadlines []ExtractedDeadline

	// German date patterns: DD.MM.YYYY or DD. Month YYYY
	datePattern := regexp.MustCompile(`\b(\d{1,2})\.(\d{1,2})\.(\d{4})\b`)

	// Look for deadline keywords near dates
	deadlineKeywords := []string{
		"bis zum", "bis spätestens", "frist bis", "fristende",
		"innerhalb von", "binnen", "einlangen bis", "abzugeben bis",
		"zu entrichten bis", "zahlbar bis", "rechtskräftig bis",
	}

	matches := datePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		dateStr := text[match[0]:match[1]]
		date, err := parseGermanDate(dateStr)
		if err != nil {
			continue
		}

		// Check surrounding context for deadline keywords
		start := match[0] - 100
		if start < 0 {
			start = 0
		}
		end := match[1] + 50
		if end > len(text) {
			end = len(text)
		}
		context := strings.ToLower(text[start:end])

		isDeadline := false
		for _, kw := range deadlineKeywords {
			if strings.Contains(context, kw) {
				isDeadline = true
				break
			}
		}

		if isDeadline {
			deadline := ExtractedDeadline{
				Type:        "response",
				Date:        date,
				Description: "Extracted deadline",
				SourceText:  text[start:end],
				Confidence:  0.6,
				IsHard:      true,
			}

			// Classify deadline type
			if strings.Contains(context, "zahlung") || strings.Contains(context, "entrichten") {
				deadline.Type = "payment"
			} else if strings.Contains(context, "berufung") || strings.Contains(context, "beschwerde") {
				deadline.Type = "appeal"
			} else if strings.Contains(context, "einreich") || strings.Contains(context, "abgeben") {
				deadline.Type = "submission"
			}

			deadlines = append(deadlines, deadline)
		}
	}

	return deadlines
}

// ExtractedAmount represents an extracted monetary amount
type ExtractedAmount struct {
	Type        string     `json:"type"`
	Amount      float64    `json:"amount"`
	Currency    string     `json:"currency"`
	Description string     `json:"description"`
	SourceText  string     `json:"source_text"`
	Confidence  float64    `json:"confidence"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// ExtractAmounts extracts monetary amounts from document text
func (e *Extractor) ExtractAmounts(ctx context.Context, text string) ([]ExtractedAmount, error) {
	prompt, err := e.promptLoader.Get(ctx, ai.PromptAmount)
	var systemPrompt, userTemplate string
	if err != nil {
		systemPrompt = defaultAmountPrompt
		userTemplate = "Extract all monetary amounts from this Austrian official document:\n\n{document_text}"
	} else {
		systemPrompt = prompt.SystemPrompt
		userTemplate = prompt.UserPromptTemplate
	}

	truncatedText := text
	if len(text) > 6000 {
		truncatedText = text[:6000] + "\n\n[Text truncated...]"
	}

	userPrompt := strings.ReplaceAll(userTemplate, "{document_text}", truncatedText)

	response, err := e.aiClient.CompleteWithRetry(ctx, systemPrompt, userPrompt, 0.1, 2)
	if err != nil {
		return nil, fmt.Errorf("AI amount extraction failed: %w", err)
	}

	parsed, err := ai.ParseAmounts(response.GetText())
	if err != nil {
		return e.extractAmountsRegex(text), nil
	}

	var amounts []ExtractedAmount
	for _, a := range parsed.Amounts {
		amount := ExtractedAmount{
			Type:        a.AmountType,
			Amount:      a.Amount,
			Currency:    a.Currency,
			Description: a.Label,
			SourceText:  a.SourceText,
			Confidence:  a.Confidence,
		}

		amounts = append(amounts, amount)
	}

	return amounts, nil
}

// extractAmountsRegex uses regex patterns to find amounts
func (e *Extractor) extractAmountsRegex(text string) []ExtractedAmount {
	var amounts []ExtractedAmount

	// Euro amount patterns: €1.234,56 or 1.234,56 EUR or 1234,56€
	amountPattern := regexp.MustCompile(`(€\s*)?(\d{1,3}(?:\.\d{3})*(?:,\d{2})?)\s*(€|EUR)?`)

	matches := amountPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		amountStr := text[match[4]:match[5]]

		// Parse amount
		amountStr = strings.ReplaceAll(amountStr, ".", "")
		amountStr = strings.ReplaceAll(amountStr, ",", ".")
		amountVal, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amountVal == 0 {
			continue
		}

		// Skip very small or very large amounts
		if amountVal < 1 || amountVal > 10000000 {
			continue
		}

		// Get context
		start := match[0] - 50
		if start < 0 {
			start = 0
		}
		end := match[1] + 30
		if end > len(text) {
			end = len(text)
		}
		context := strings.ToLower(text[start:end])

		amount := ExtractedAmount{
			Type:        "other",
			Amount:      amountVal,
			Currency:    "EUR",
			Description: "Extracted amount",
			SourceText:  text[start:end],
			Confidence:  0.5,
		}

		// Classify amount type
		if strings.Contains(context, "nachzahlung") || strings.Contains(context, "zahlen") {
			amount.Type = "payment_due"
			amount.Confidence = 0.7
		} else if strings.Contains(context, "guthaben") || strings.Contains(context, "erstattung") {
			amount.Type = "refund"
			amount.Confidence = 0.7
		} else if strings.Contains(context, "steuer") || strings.Contains(context, "abgabe") {
			amount.Type = "tax"
			amount.Confidence = 0.6
		} else if strings.Contains(context, "strafe") || strings.Contains(context, "zuschlag") {
			amount.Type = "penalty"
			amount.Confidence = 0.7
		}

		amounts = append(amounts, amount)
	}

	return amounts
}

// SummaryResult contains a document summary
type SummaryResult struct {
	Summary      string   `json:"summary"`
	KeyPoints    []string `json:"key_points"`
	ActionNeeded bool     `json:"action_needed"`
	Language     string   `json:"language"`
}

// Summarize creates a plain-language summary of the document
func (e *Extractor) Summarize(ctx context.Context, text string) (*SummaryResult, error) {
	prompt, err := e.promptLoader.Get(ctx, ai.PromptSummary)
	var systemPrompt, userTemplate string
	if err != nil {
		systemPrompt = defaultSummaryPrompt
		userTemplate = "Summarize this Austrian official document in plain German:\n\n{document_text}"
	} else {
		systemPrompt = prompt.SystemPrompt
		userTemplate = prompt.UserPromptTemplate
	}

	// Keep more text for summarization
	truncatedText := text
	if len(text) > 10000 {
		truncatedText = text[:10000] + "\n\n[Text truncated...]"
	}

	userPrompt := strings.ReplaceAll(userTemplate, "{document_text}", truncatedText)

	response, err := e.aiClient.CompleteWithRetry(ctx, systemPrompt, userPrompt, 0.3, 2)
	if err != nil {
		return nil, fmt.Errorf("AI summarization failed: %w", err)
	}

	parsed, err := ai.ParseSummary(response.GetText())
	if err != nil {
		// Return raw text as summary
		return &SummaryResult{
			Summary:   response.GetText(),
			KeyPoints: []string{},
			Language:  "de",
		}, nil
	}

	return &SummaryResult{
		Summary:      parsed.Summary,
		KeyPoints:    parsed.KeyPoints,
		ActionNeeded: parsed.ActionRequired,
		Language:     "de",
	}, nil
}

// ActionItemResult represents a generated action item
type ActionItemResult struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"`
	Category    string     `json:"category"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	SourceText  string     `json:"source_text"`
	Confidence  float64    `json:"confidence"`
}

// ExtractActionItems generates action items from the document
func (e *Extractor) ExtractActionItems(ctx context.Context, text string, classification *ClassificationResult, deadlines []ExtractedDeadline) ([]ActionItemResult, error) {
	// For now, use heuristic-based action items
	// AI-based action item generation can be added later
	return e.generateBasicActionItems(classification, deadlines), nil
}

// extractActionItemsAI extracts action items using AI (for future use)
func (e *Extractor) extractActionItemsAI(ctx context.Context, text string, classification *ClassificationResult, deadlines []ExtractedDeadline) ([]ActionItemResult, error) {
	systemPrompt := defaultActionItemsPrompt

	truncatedText := text
	if len(text) > 6000 {
		truncatedText = text[:6000] + "\n\n[Text truncated...]"
	}

	// Build context
	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("Document type: %s (%s)\n", classification.DocumentType, classification.DocumentSubtype))
	contextBuilder.WriteString(fmt.Sprintf("Urgency: %s\n", classification.Urgency))
	if len(deadlines) > 0 {
		contextBuilder.WriteString("Deadlines:\n")
		for _, d := range deadlines {
			contextBuilder.WriteString(fmt.Sprintf("- %s: %s (%s)\n", d.Type, d.Date.Format("02.01.2006"), d.Description))
		}
	}

	userPrompt := "Generate action items for this document:\n\nContext:\n" + contextBuilder.String() + "\n\nDocument text:\n" + truncatedText

	response, err := e.aiClient.CompleteWithRetry(ctx, systemPrompt, userPrompt, 0.2, 2)
	if err != nil {
		// Generate basic action items from classification
		return e.generateBasicActionItems(classification, deadlines), nil
	}

	// Parse response - for now just return basic items
	_ = response

	// If parsing fails, generate basic items
	return e.generateBasicActionItems(classification, deadlines), nil
}

// generateBasicActionItems creates action items from classification and deadlines
func (e *Extractor) generateBasicActionItems(classification *ClassificationResult, deadlines []ExtractedDeadline) []ActionItemResult {
	var items []ActionItemResult

	// Create action items based on document type
	switch classification.DocumentType {
	case DocTypeErsuchen:
		item := ActionItemResult{
			Title:       "Ergänzungsersuchen beantworten",
			Description: "Auf das Ergänzungsersuchen der Behörde antworten",
			Priority:    PriorityHigh,
			Category:    "response",
			Confidence:  0.9,
		}
		if len(deadlines) > 0 {
			item.DueDate = &deadlines[0].Date
		}
		items = append(items, item)

	case DocTypeMahnung:
		items = append(items, ActionItemResult{
			Title:       "Mahnung prüfen und begleichen",
			Description: "Offenen Betrag prüfen und ggf. überweisen",
			Priority:    PriorityHigh,
			Category:    "payment",
			Confidence:  0.9,
		})

	case DocTypeBescheid:
		items = append(items, ActionItemResult{
			Title:       "Bescheid prüfen",
			Description: "Steuerbescheid auf Richtigkeit prüfen",
			Priority:    PriorityMedium,
			Category:    "review",
			Confidence:  0.8,
		})

	case DocTypeZahlungsbefehl:
		item := ActionItemResult{
			Title:       "Zahlungsbefehl sofort bearbeiten",
			Description: "Zahlungsbefehl prüfen, ggf. Einspruch einlegen oder zahlen",
			Priority:    PriorityHigh,
			Category:    "urgent",
			Confidence:  0.95,
		}
		if len(deadlines) > 0 {
			item.DueDate = &deadlines[0].Date
		}
		items = append(items, item)
	}

	// Add deadline-based action items
	for _, d := range deadlines {
		dueDate := d.Date
		items = append(items, ActionItemResult{
			Title:       fmt.Sprintf("Frist: %s", d.Description),
			Description: fmt.Sprintf("Frist endet am %s", d.Date.Format("02.01.2006")),
			Priority:    determinePriority(d.Date),
			Category:    d.Type,
			DueDate:     &dueDate,
			SourceText:  d.SourceText,
			Confidence:  d.Confidence,
		})
	}

	return items
}

// determinePriority determines priority based on deadline date
func determinePriority(deadline time.Time) string {
	daysUntil := time.Until(deadline).Hours() / 24
	switch {
	case daysUntil <= 3:
		return PriorityHigh
	case daysUntil <= 14:
		return PriorityMedium
	default:
		return PriorityLow
	}
}

// SuggestionResult represents a response suggestion
type SuggestionResult struct {
	Type       string  `json:"type"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Reasoning  string  `json:"reasoning"`
	Confidence float64 `json:"confidence"`
}

// GenerateSuggestions creates response suggestions for the document
func (e *Extractor) GenerateSuggestions(ctx context.Context, text string, classification *ClassificationResult) ([]SuggestionResult, error) {
	// Only generate suggestions for documents requiring response
	if classification.DocumentType != DocTypeErsuchen && classification.DocumentType != DocTypeVorhalt {
		return nil, nil
	}

	prompt, err := e.promptLoader.Get(ctx, ai.PromptSuggestion)
	var systemPrompt, userTemplate string
	if err != nil {
		systemPrompt = defaultSuggestionPrompt
		userTemplate = "Generate response suggestions for this document:\n\n{document_text}"
	} else {
		systemPrompt = prompt.SystemPrompt
		userTemplate = prompt.UserPromptTemplate
	}

	truncatedText := text
	if len(text) > 8000 {
		truncatedText = text[:8000] + "\n\n[Text truncated...]"
	}

	userPrompt := strings.ReplaceAll(userTemplate, "{document_text}", truncatedText)
	userPrompt = strings.ReplaceAll(userPrompt, "{client_context}", "Keine zusätzlichen Informationen")

	response, err := e.aiClient.CompleteWithRetry(ctx, systemPrompt, userPrompt, 0.5, 2)
	if err != nil {
		return nil, fmt.Errorf("AI suggestion generation failed: %w", err)
	}

	parsed, err := ai.ParseSuggestion(response.GetText())
	if err != nil {
		return nil, fmt.Errorf("parse suggestion response: %w", err)
	}

	return []SuggestionResult{{
		Type:       "formal_response",
		Title:      "Antwortvorschlag",
		Content:    parsed.SuggestionText,
		Reasoning:  strings.Join(parsed.Warnings, "; "),
		Confidence: parsed.Confidence,
	}}, nil
}

// parseGermanDate parses dates in German format (DD.MM.YYYY)
func parseGermanDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)

	// Try DD.MM.YYYY format
	if t, err := time.Parse("02.01.2006", dateStr); err == nil {
		return t, nil
	}

	// Try DD.MM.YY format
	if t, err := time.Parse("02.01.06", dateStr); err == nil {
		return t, nil
	}

	// Try written format with German month names
	monthMap := map[string]string{
		"januar": "01", "jänner": "01",
		"februar": "02",
		"märz": "03",
		"april": "04",
		"mai": "05",
		"juni": "06",
		"juli": "07",
		"august": "08",
		"september": "09",
		"oktober": "10",
		"november": "11",
		"dezember": "12",
	}

	lower := strings.ToLower(dateStr)
	for monthName, monthNum := range monthMap {
		if strings.Contains(lower, monthName) {
			// Extract day and year
			dayPattern := regexp.MustCompile(`(\d{1,2})\.?\s*` + monthName)
			yearPattern := regexp.MustCompile(`(\d{4})`)

			dayMatch := dayPattern.FindStringSubmatch(lower)
			yearMatch := yearPattern.FindStringSubmatch(lower)

			if len(dayMatch) > 1 && len(yearMatch) > 1 {
				day, _ := strconv.Atoi(dayMatch[1])
				year, _ := strconv.Atoi(yearMatch[1])
				return time.Date(year, time.Month(mustAtoi(monthNum)), day, 0, 0, 0, 0, time.Local), nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// Default prompts
const defaultDeadlinePrompt = `Du bist ein Experte für österreichische Behördendokumente.
Extrahiere alle Fristen aus dem Dokument.

Gib die Antwort als JSON in diesem Format:
{
  "deadlines": [
    {
      "type": "response|payment|submission|appeal|other",
      "date": "DD.MM.YYYY",
      "description": "Kurze Beschreibung der Frist",
      "source_text": "Originaltext aus dem Dokument",
      "confidence": 0.0-1.0,
      "is_hard": true|false
    }
  ]
}

Wichtige Fristtypen:
- response: Antwortfrist auf Ersuchen
- payment: Zahlungsfrist
- submission: Einreichungsfrist
- appeal: Berufungs-/Beschwerdefrist`

const defaultAmountPrompt = `Du bist ein Experte für österreichische Behördendokumente.
Extrahiere alle Geldbeträge aus dem Dokument.

Gib die Antwort als JSON in diesem Format:
{
  "amounts": [
    {
      "type": "tax|payment_due|refund|penalty|fee|other",
      "amount": 1234.56,
      "currency": "EUR",
      "description": "Beschreibung des Betrags",
      "source_text": "Originaltext aus dem Dokument",
      "confidence": 0.0-1.0,
      "due_date": "DD.MM.YYYY (optional)"
    }
  ]
}`

const defaultSummaryPrompt = `Du bist ein Experte für österreichische Behördendokumente.
Erstelle eine verständliche Zusammenfassung des Dokuments in einfachem Deutsch.

Gib die Antwort als JSON in diesem Format:
{
  "summary": "Klare, verständliche Zusammenfassung (2-3 Sätze)",
  "key_points": ["Wichtiger Punkt 1", "Wichtiger Punkt 2"],
  "action_needed": true|false
}

Ziel: Das Dokument so erklären, dass es jeder verstehen kann.`

const defaultActionItemsPrompt = `Du bist ein Experte für österreichische Behördendokumente.
Erstelle Aktionspunkte basierend auf dem Dokument.

Gib die Antwort als JSON Array:
[
  {
    "title": "Kurzer Aktionspunkt",
    "description": "Detaillierte Beschreibung",
    "priority": "high|medium|low",
    "category": "response|payment|review|submission|other",
    "confidence": 0.0-1.0
  }
]`

const defaultSuggestionPrompt = `Du bist ein Experte für österreichische Behördendokumente.
Erstelle einen Antwortvorschlag für das Ergänzungsersuchen.

Gib die Antwort als JSON in diesem Format:
{
  "type": "formal_response",
  "title": "Titel des Antwortschreibens",
  "content": "Vollständiger Antworttext im formellen Stil",
  "reasoning": "Erklärung warum diese Antwort passend ist",
  "confidence": 0.0-1.0
}

Der Antworttext sollte:
- Formell und höflich sein
- Auf alle Punkte des Ersuchens eingehen
- Platzhalter für fehlende Informationen enthalten (z.B. [BETRAG EINFÜGEN])`
