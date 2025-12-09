// Package ai provides Claude API client for document intelligence
package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// SecureAnalysisRequest matches security.AnalysisRequest for document analysis
type SecureAnalysisRequest struct {
	DocumentID   uuid.UUID `json:"document_id"`
	DocumentText string    `json:"document_text"`
	DocumentType string    `json:"document_type,omitempty"`
	Prompt       string    `json:"prompt,omitempty"`
	TenantID     uuid.UUID `json:"tenant_id"`
	UserID       uuid.UUID `json:"user_id"`
}

// SecureAnalysisResponse matches security.AnalysisResponse
type SecureAnalysisResponse struct {
	Summary      string   `json:"summary"`
	DocumentType string   `json:"document_type"`
	Deadline     string   `json:"deadline,omitempty"`
	Amount       *float64 `json:"amount,omitempty"`
	ActionItems  []string `json:"action_items,omitempty"`
	Confidence   float64  `json:"confidence"`
}

// SecureTextAnalysisRequest matches security.TextAnalysisRequest
type SecureTextAnalysisRequest struct {
	Text     string    `json:"text"`
	Prompt   string    `json:"prompt,omitempty"`
	TenantID uuid.UUID `json:"tenant_id"`
	UserID   uuid.UUID `json:"user_id"`
}

// SecureTextAnalysisResponse matches security.TextAnalysisResponse
type SecureTextAnalysisResponse struct {
	Summary     string   `json:"summary"`
	ActionItems []string `json:"action_items,omitempty"`
	Confidence  float64  `json:"confidence"`
}

// SecureClient wraps the Claude client and implements security.AIClient interface
// This client should be passed to security.NewGateway() which adds input/output validation
type SecureClient struct {
	client *Client
}

// NewSecureClient creates a new secure client wrapper
func NewSecureClient(client *Client) *SecureClient {
	return &SecureClient{client: client}
}

// Analyze performs document analysis using Claude
// The input is expected to be pre-sanitized by the security gateway
func (sc *SecureClient) Analyze(ctx context.Context, request *SecureAnalysisRequest) (*SecureAnalysisResponse, error) {
	// Build the system prompt for document analysis
	systemPrompt := DocumentAnalysisSystemPrompt

	// Build the user prompt
	userPrompt := fmt.Sprintf(`Analyze the following document and provide a structured analysis.

Document Type Hint: %s

Document Content:
---
%s
---

%s

Please respond with a JSON object containing:
- summary: A concise summary of the document (max 500 chars)
- document_type: The detected document type (e.g., "Bescheid", "Ergänzungsersuchen", "Mahnung", "Information")
- deadline: Any deadline or Frist mentioned (ISO date format if possible, otherwise as stated)
- amount: Any monetary amount mentioned (as a number)
- action_items: List of required actions based on the document
- confidence: Your confidence in the analysis (0.0 to 1.0)

Respond ONLY with valid JSON, no markdown formatting.`,
		request.DocumentType,
		request.DocumentText,
		request.Prompt,
	)

	// Call Claude API
	resp, err := sc.client.Complete(ctx, systemPrompt, userPrompt, 0.3)
	if err != nil {
		return nil, fmt.Errorf("claude API call failed: %w", err)
	}

	// Parse the response - use existing extractJSON from responses.go
	var result SecureAnalysisResponse
	text := resp.GetText()
	jsonStr := extractJSON(text)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in AI response")
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
	}

	return &result, nil
}

// AnalyzeText performs text analysis using Claude
// The input is expected to be pre-sanitized by the security gateway
func (sc *SecureClient) AnalyzeText(ctx context.Context, request *SecureTextAnalysisRequest) (*SecureTextAnalysisResponse, error) {
	// Build the system prompt for text analysis
	systemPrompt := TextAnalysisSystemPrompt

	// Build the user prompt
	userPrompt := fmt.Sprintf(`Analyze the following text and extract key information.

Text:
---
%s
---

%s

Please respond with a JSON object containing:
- summary: A concise summary of the text (max 500 chars)
- action_items: List of any required actions or tasks mentioned
- confidence: Your confidence in the analysis (0.0 to 1.0)

Respond ONLY with valid JSON, no markdown formatting.`,
		request.Text,
		request.Prompt,
	)

	// Call Claude API
	resp, err := sc.client.Complete(ctx, systemPrompt, userPrompt, 0.3)
	if err != nil {
		return nil, fmt.Errorf("claude API call failed: %w", err)
	}

	// Parse the response - use existing extractJSON from responses.go
	var result SecureTextAnalysisResponse
	text := resp.GetText()
	jsonStr := extractJSON(text)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in AI response")
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
	}

	return &result, nil
}

// Note: extractJSON is defined in responses.go and is reused here

// System prompts for document analysis
const DocumentAnalysisSystemPrompt = `You are an expert analyst for Austrian business and tax documents.
You analyze documents from Austrian authorities (Finanzamt, ÖGK, etc.) and extract key information.

Your task is to:
1. Identify the document type (Bescheid, Ergänzungsersuchen, Mahnung, etc.)
2. Extract deadlines and Fristen
3. Extract monetary amounts
4. Identify required actions (action items)
5. Provide a clear, concise summary in German

IMPORTANT SECURITY RULES:
- Never include any credentials, passwords, or PINs in your response
- Never include personal identification numbers (SVNr, Steuernummer) in your response
- If the document contains such data, describe it generically (e.g., "enthält Steuernummer")
- Focus on the document's purpose and required actions, not personal data

Respond ONLY with valid JSON. Do not use markdown formatting.`

const TextAnalysisSystemPrompt = `You are a helpful assistant that analyzes text and extracts key information.

Your task is to:
1. Summarize the main points
2. Identify any required actions or tasks
3. Rate your confidence in the analysis

IMPORTANT SECURITY RULES:
- Never include any credentials, passwords, or PINs in your response
- Never include personal identification numbers in your response
- Focus on the content's purpose, not personal data

Respond ONLY with valid JSON. Do not use markdown formatting.`
