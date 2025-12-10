package analysis

import (
	"context"
	"fmt"
	"strings"

	"austrian-business-infrastructure/internal/ai"
)

// DocumentType represents the type of document
type DocumentType string

const (
	DocTypeBescheid       DocumentType = "bescheid"       // Official decision
	DocTypeErsuchen       DocumentType = "ersuchen"       // Request for information
	DocTypeMitteilung     DocumentType = "mitteilung"     // Information/notification
	DocTypeMahnung        DocumentType = "mahnung"        // Payment reminder/warning
	DocTypeRechnung       DocumentType = "rechnung"       // Invoice
	DocTypeBestätigung    DocumentType = "bestätigung"    // Confirmation
	DocTypeAntrag         DocumentType = "antrag"         // Application
	DocTypeVorhalt        DocumentType = "vorhalt"        // Preliminary assessment
	DocTypeZahlungsbefehl DocumentType = "zahlungsbefehl" // Payment order
	DocTypeSonstige       DocumentType = "sonstige"       // Other
)

// DocumentSubtype represents more specific document types
type DocumentSubtype string

const (
	SubtypeEinkommensteuer    DocumentSubtype = "einkommensteuer"
	SubtypeUmsatzsteuer       DocumentSubtype = "umsatzsteuer"
	SubtypeKörperschaftsteuer DocumentSubtype = "körperschaftsteuer"
	SubtypeLohnsteuer         DocumentSubtype = "lohnsteuer"
	SubtypeSozialversicherung DocumentSubtype = "sozialversicherung"
	SubtypeGewerbe            DocumentSubtype = "gewerbe"
	SubtypeZoll               DocumentSubtype = "zoll"
	SubtypeFinanzamt          DocumentSubtype = "finanzamt"
	SubtypeGKK                DocumentSubtype = "gkk"
	SubtypeWKO                DocumentSubtype = "wko"
	SubtypeSonstige           DocumentSubtype = "sonstige"
)

// ClassificationResult contains the result of document classification
type ClassificationResult struct {
	DocumentType    DocumentType    `json:"document_type"`
	DocumentSubtype DocumentSubtype `json:"document_subtype"`
	Confidence      float64         `json:"confidence"`
	Reasoning       string          `json:"reasoning"`
	Keywords        []string        `json:"keywords"`
	Urgency         string          `json:"urgency"`
	RequiresAction  bool            `json:"requires_action"`
	SuggestedTags   []string        `json:"suggested_tags"`
}

// Classifier handles document classification using AI
type Classifier struct {
	aiClient     *ai.Client
	promptLoader *ai.PromptLoader
}

// NewClassifier creates a new document classifier
func NewClassifier(aiClient *ai.Client, promptLoader *ai.PromptLoader) *Classifier {
	return &Classifier{
		aiClient:     aiClient,
		promptLoader: promptLoader,
	}
}

// Classify analyzes document text and returns classification
func (c *Classifier) Classify(ctx context.Context, text string) (*ClassificationResult, error) {
	// Load classification prompt
	prompt, err := c.promptLoader.Get(ctx, ai.PromptClassification)
	var systemPrompt, userTemplate string
	if err != nil {
		// Fall back to default prompt
		systemPrompt = defaultClassificationPrompt
		userTemplate = "Analyze this Austrian official document:\n\n{document_text}"
	} else {
		systemPrompt = prompt.SystemPrompt
		userTemplate = prompt.UserPromptTemplate
	}

	// Truncate text if too long (keep first ~4000 chars for classification)
	truncatedText := text
	if len(text) > 4000 {
		truncatedText = text[:4000] + "\n\n[Text truncated for classification...]"
	}

	// Build user prompt
	userPrompt := strings.ReplaceAll(userTemplate, "{document_text}", truncatedText)

	// Call Claude API
	response, err := c.aiClient.CompleteWithRetry(ctx, systemPrompt, userPrompt, 0.1, 2)
	if err != nil {
		return nil, fmt.Errorf("AI classification failed: %w", err)
	}

	// Parse response
	parsed, err := ai.ParseClassification(response.GetText())
	if err != nil {
		return nil, fmt.Errorf("parse classification response: %w", err)
	}

	// Map to result - map Priority to Urgency
	urgency := "normal"
	requiresAction := false
	switch parsed.Priority {
	case "critical":
		urgency = "critical"
		requiresAction = true
	case "high":
		urgency = "high"
		requiresAction = true
	case "medium":
		urgency = "normal"
	case "low":
		urgency = "low"
	}

	result := &ClassificationResult{
		DocumentType:    DocumentType(parsed.DocumentType),
		DocumentSubtype: DocumentSubtype(parsed.DocumentSubtype),
		Confidence:      parsed.Confidence,
		Reasoning:       parsed.Reasoning,
		Keywords:        []string{},
		Urgency:         urgency,
		RequiresAction:  requiresAction,
		SuggestedTags:   []string{},
	}

	// Validate document type
	if !isValidDocumentType(result.DocumentType) {
		result.DocumentType = DocTypeSonstige
	}

	return result, nil
}

// ClassifyWithFallback attempts classification with fallback to heuristics
func (c *Classifier) ClassifyWithFallback(ctx context.Context, text string, title string) (*ClassificationResult, error) {
	// Try AI classification first
	result, err := c.Classify(ctx, text)
	if err == nil && result.Confidence > 0.5 {
		return result, nil
	}

	// Fall back to heuristic classification
	return c.classifyHeuristic(text, title), nil
}

// classifyHeuristic uses keyword matching for classification
func (c *Classifier) classifyHeuristic(text, title string) *ClassificationResult {
	lowerText := strings.ToLower(text)
	lowerTitle := strings.ToLower(title)
	combined := lowerText + " " + lowerTitle

	result := &ClassificationResult{
		DocumentType:    DocTypeSonstige,
		DocumentSubtype: SubtypeSonstige,
		Confidence:      0.6,
		Reasoning:       "Classified using keyword matching",
		Keywords:        []string{},
		Urgency:         "normal",
		RequiresAction:  false,
	}

	// Check for document types by keywords
	if containsAny(combined, []string{"ergänzungsersuchen", "ersuchen um ergänzung", "werden sie ersucht"}) {
		result.DocumentType = DocTypeErsuchen
		result.RequiresAction = true
		result.Urgency = "high"
		result.Keywords = append(result.Keywords, "ersuchen")
	} else if containsAny(combined, []string{"bescheid", "abgabenbescheid", "einkommensteuerbescheid", "umsatzsteuerbescheid"}) {
		result.DocumentType = DocTypeBescheid
		result.Keywords = append(result.Keywords, "bescheid")
	} else if containsAny(combined, []string{"mahnung", "zahlungserinnerung", "säumniszuschlag"}) {
		result.DocumentType = DocTypeMahnung
		result.RequiresAction = true
		result.Urgency = "high"
		result.Keywords = append(result.Keywords, "mahnung")
	} else if containsAny(combined, []string{"vorhalt", "vorhaltsbeantwortung"}) {
		result.DocumentType = DocTypeVorhalt
		result.RequiresAction = true
		result.Keywords = append(result.Keywords, "vorhalt")
	} else if containsAny(combined, []string{"rechnung", "faktura", "invoice"}) {
		result.DocumentType = DocTypeRechnung
		result.Keywords = append(result.Keywords, "rechnung")
	} else if containsAny(combined, []string{"mitteilung", "information", "benachrichtigung"}) {
		result.DocumentType = DocTypeMitteilung
		result.Keywords = append(result.Keywords, "mitteilung")
	} else if containsAny(combined, []string{"zahlungsbefehl", "exekution"}) {
		result.DocumentType = DocTypeZahlungsbefehl
		result.RequiresAction = true
		result.Urgency = "critical"
		result.Keywords = append(result.Keywords, "zahlungsbefehl")
	}

	// Check for subtypes
	if containsAny(combined, []string{"einkommensteuer", "est", "einkommensteuererklärung"}) {
		result.DocumentSubtype = SubtypeEinkommensteuer
	} else if containsAny(combined, []string{"umsatzsteuer", "ust", "umsatzsteuervoranmeldung", "uva"}) {
		result.DocumentSubtype = SubtypeUmsatzsteuer
	} else if containsAny(combined, []string{"körperschaftsteuer", "köst"}) {
		result.DocumentSubtype = SubtypeKörperschaftsteuer
	} else if containsAny(combined, []string{"lohnsteuer", "l16", "lohnzettel"}) {
		result.DocumentSubtype = SubtypeLohnsteuer
	} else if containsAny(combined, []string{"sozialversicherung", "svnr", "gkk", "bva", "svs"}) {
		result.DocumentSubtype = SubtypeSozialversicherung
	} else if containsAny(combined, []string{"finanzamt", "fa "}) {
		result.DocumentSubtype = SubtypeFinanzamt
	}

	return result
}

// isValidDocumentType checks if a document type is valid
func isValidDocumentType(dt DocumentType) bool {
	switch dt {
	case DocTypeBescheid, DocTypeErsuchen, DocTypeMitteilung, DocTypeMahnung,
		DocTypeRechnung, DocTypeBestätigung, DocTypeAntrag, DocTypeVorhalt,
		DocTypeZahlungsbefehl, DocTypeSonstige:
		return true
	default:
		return false
	}
}

// containsAny checks if text contains any of the given keywords
func containsAny(text string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// Default classification prompt
const defaultClassificationPrompt = `Du bist ein Experte für österreichische Behördendokumente.
Analysiere das folgende Dokument und klassifiziere es.

Gib die Antwort als JSON in diesem Format:
{
  "document_type": "bescheid|ersuchen|mitteilung|mahnung|rechnung|bestätigung|antrag|vorhalt|zahlungsbefehl|sonstige",
  "document_subtype": "einkommensteuer|umsatzsteuer|körperschaftsteuer|lohnsteuer|sozialversicherung|gewerbe|zoll|finanzamt|gkk|wko|sonstige",
  "confidence": 0.0-1.0,
  "reasoning": "Kurze Erklärung der Klassifizierung",
  "keywords": ["gefundene", "schlüsselwörter"],
  "urgency": "low|normal|high|critical",
  "requires_action": true|false,
  "suggested_tags": ["vorgeschlagene", "tags"]
}

Wichtige Dokumenttypen:
- bescheid: Offizieller Bescheid einer Behörde (z.B. Steuerbescheid)
- ersuchen: Ergänzungsersuchen, Aufforderung zur Stellungnahme
- mitteilung: Information ohne Handlungsbedarf
- mahnung: Zahlungserinnerung, Säumniszuschlag
- vorhalt: Vorhaltsbeantwortung, Prüfungsfeststellung
- zahlungsbefehl: Gerichtlicher Zahlungsbefehl, Exekution

Urgency Kriterien:
- critical: Zahlungsbefehl, Exekutionsandrohung, sehr kurze Frist (<3 Tage)
- high: Ersuchen mit Frist, Mahnung, Frist <14 Tage
- normal: Bescheide ohne dringende Frist
- low: Informationen, Bestätigungen`
