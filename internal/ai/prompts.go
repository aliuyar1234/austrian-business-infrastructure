package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PromptType identifies the type of analysis prompt
type PromptType string

const (
	PromptClassification PromptType = "classification"
	PromptDeadline       PromptType = "deadline"
	PromptSummary        PromptType = "summary"
	PromptAmount         PromptType = "amount"
	PromptSuggestion     PromptType = "suggestion"
)

// Prompt represents an analysis prompt from the database
type Prompt struct {
	ID                 string          `json:"id"`
	PromptType         PromptType      `json:"prompt_type"`
	Version            int             `json:"version"`
	IsActive           bool            `json:"is_active"`
	SystemPrompt       string          `json:"system_prompt"`
	UserPromptTemplate string          `json:"user_prompt_template"`
	Model              string          `json:"model"`
	MaxTokens          int             `json:"max_tokens"`
	Temperature        float64         `json:"temperature"`
	ResponseSchema     json.RawMessage `json:"response_schema"`
	Description        string          `json:"description"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

// PromptLoader loads prompts from the database
type PromptLoader struct {
	db    *sql.DB
	cache map[PromptType]*Prompt
}

// NewPromptLoader creates a new prompt loader
func NewPromptLoader(db *sql.DB) *PromptLoader {
	return &PromptLoader{
		db:    db,
		cache: make(map[PromptType]*Prompt),
	}
}

// Get retrieves a prompt by type
func (l *PromptLoader) Get(ctx context.Context, promptType PromptType) (*Prompt, error) {
	// Check cache first
	if prompt, ok := l.cache[promptType]; ok {
		return prompt, nil
	}

	// Load from database
	prompt, err := l.loadFromDB(ctx, promptType)
	if err != nil {
		return nil, err
	}

	// Cache it
	l.cache[promptType] = prompt
	return prompt, nil
}

// loadFromDB loads a prompt from the database
func (l *PromptLoader) loadFromDB(ctx context.Context, promptType PromptType) (*Prompt, error) {
	query := `
		SELECT id, prompt_type, version, is_active, system_prompt, user_prompt_template,
		       model, max_tokens, temperature, response_schema, description, created_at, updated_at
		FROM analysis_prompts
		WHERE prompt_type = $1 AND is_active = true
		ORDER BY version DESC
		LIMIT 1
	`

	var prompt Prompt
	err := l.db.QueryRowContext(ctx, query, string(promptType)).Scan(
		&prompt.ID,
		&prompt.PromptType,
		&prompt.Version,
		&prompt.IsActive,
		&prompt.SystemPrompt,
		&prompt.UserPromptTemplate,
		&prompt.Model,
		&prompt.MaxTokens,
		&prompt.Temperature,
		&prompt.ResponseSchema,
		&prompt.Description,
		&prompt.CreatedAt,
		&prompt.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("prompt not found: %s", promptType)
	}
	if err != nil {
		return nil, fmt.Errorf("query prompt: %w", err)
	}

	return &prompt, nil
}

// Refresh clears the cache to reload prompts from database
func (l *PromptLoader) Refresh() {
	l.cache = make(map[PromptType]*Prompt)
}

// BuildUserPrompt builds the user prompt by replacing placeholders
func (p *Prompt) BuildUserPrompt(vars map[string]string) string {
	result := p.UserPromptTemplate
	for key, value := range vars {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// DefaultPrompts returns hardcoded fallback prompts
func DefaultPrompts() map[PromptType]*Prompt {
	return map[PromptType]*Prompt{
		PromptClassification: {
			PromptType:   PromptClassification,
			SystemPrompt: "Du bist ein Experte für österreichische Steuerdokumente und Behördenbriefe. Klassifiziere das folgende Dokument präzise. Antworte ausschließlich im JSON-Format.",
			UserPromptTemplate: `Analysiere dieses Dokument und klassifiziere es:

{document_text}

Antworte im folgenden JSON-Format:
{
  "document_type": "bescheid|ersuchen|info|rechnung|mahnung|sonstige",
  "document_subtype": "ergaenzungsersuchen|steuerbescheid|mahnbescheid|...",
  "priority": "critical|high|medium|low",
  "confidence": 0.0-1.0,
  "reasoning": "Kurze Begründung"
}`,
			Model:       "claude-sonnet-4-20250514",
			MaxTokens:   1024,
			Temperature: 0.3,
		},
		PromptDeadline: {
			PromptType:   PromptDeadline,
			SystemPrompt: "Du bist ein Experte für österreichische Steuerdokumente. Extrahiere alle Fristen aus dem Dokument. Berechne das konkrete Datum wenn nötig. Antworte ausschließlich im JSON-Format.",
			UserPromptTemplate: `Extrahiere alle Fristen aus diesem Dokument:

{document_text}

Heute ist der {current_date}.
Zustelldatum (falls bekannt): {delivery_date}

Antworte im folgenden JSON-Format:
{
  "deadlines": [
    {
      "deadline_type": "response|payment|appeal|submission|other",
      "deadline_date": "YYYY-MM-DD",
      "calculation_rule": "z.B. 4 Wochen ab Zustellung",
      "source_text": "Originaltext aus Dokument",
      "confidence": 0.0-1.0
    }
  ]
}`,
			Model:       "claude-sonnet-4-20250514",
			MaxTokens:   2048,
			Temperature: 0.2,
		},
		PromptSummary: {
			PromptType:   PromptSummary,
			SystemPrompt: "Du bist ein Experte für österreichische Steuerdokumente. Fasse das Dokument in einfacher Sprache zusammen. Vermeide Amtsdeutsch. Antworte ausschließlich im JSON-Format.",
			UserPromptTemplate: `Fasse dieses Dokument zusammen:

{document_text}

Antworte im folgenden JSON-Format:
{
  "summary": "Zusammenfassung in 2-4 Sätzen, einfache Sprache",
  "key_points": [
    "Wichtiger Punkt 1",
    "Wichtiger Punkt 2"
  ],
  "action_required": true|false,
  "tone": "neutral|dringend|positiv|negativ"
}`,
			Model:       "claude-sonnet-4-20250514",
			MaxTokens:   1024,
			Temperature: 0.4,
		},
		PromptAmount: {
			PromptType:   PromptAmount,
			SystemPrompt: "Du bist ein Experte für österreichische Steuerdokumente. Extrahiere alle Geldbeträge aus dem Dokument. Antworte ausschließlich im JSON-Format.",
			UserPromptTemplate: `Extrahiere alle Geldbeträge aus diesem Dokument:

{document_text}

Antworte im folgenden JSON-Format:
{
  "amounts": [
    {
      "amount_type": "tax_due|refund|penalty|fee|total|other",
      "amount": 1234.56,
      "currency": "EUR",
      "is_negative": false,
      "label": "z.B. Umsatzsteuer",
      "source_text": "Originaltext",
      "confidence": 0.0-1.0
    }
  ]
}`,
			Model:       "claude-sonnet-4-20250514",
			MaxTokens:   2048,
			Temperature: 0.2,
		},
		PromptSuggestion: {
			PromptType:   PromptSuggestion,
			SystemPrompt: "Du bist ein Steuerberater-Assistent. Erstelle einen Antwortvorschlag für das folgende Ergänzungsersuchen. Der Vorschlag sollte professionell und vollständig sein. Antworte ausschließlich im JSON-Format.",
			UserPromptTemplate: `Erstelle einen Antwortvorschlag für dieses Ergänzungsersuchen:

{document_text}

Kontext zum Mandanten: {client_context}

Antworte im folgenden JSON-Format:
{
  "suggestion_text": "Vollständiger Antworttext",
  "key_points": ["Punkt 1 der adressiert wird", "Punkt 2"],
  "required_documents": ["Dokument 1", "Dokument 2"],
  "confidence": 0.0-1.0,
  "warnings": ["Eventuelle Hinweise"]
}`,
			Model:       "claude-sonnet-4-20250514",
			MaxTokens:   4096,
			Temperature: 0.5,
		},
	}
}
