package security

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// ClaudeClient is the interface for the underlying Claude API client
type ClaudeClient interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string, temperature float64) (ClaudeResponse, error)
}

// ClaudeResponse represents the response from Claude API
type ClaudeResponse interface {
	GetText() string
}

// AIClientAdapter adapts a ClaudeClient to the AIClient interface expected by Gateway
type AIClientAdapter struct {
	client ClaudeClient
}

// NewAIClientAdapter creates a new adapter wrapping a Claude client
func NewAIClientAdapter(client ClaudeClient) *AIClientAdapter {
	return &AIClientAdapter{client: client}
}

// Analyze implements AIClient.Analyze
func (a *AIClientAdapter) Analyze(ctx context.Context, request *AnalysisRequest) (*AnalysisResponse, error) {
	systemPrompt := documentAnalysisSystemPrompt

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

	resp, err := a.client.Complete(ctx, systemPrompt, userPrompt, 0.3)
	if err != nil {
		return nil, fmt.Errorf("claude API call failed: %w", err)
	}

	// Parse the response
	var result AnalysisResponse
	text := resp.GetText()
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		// Try to extract JSON from markdown code blocks
		extracted := extractJSONFromText(text)
		if err := json.Unmarshal([]byte(extracted), &result); err != nil {
			return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
		}
	}

	return &result, nil
}

// AnalyzeText implements AIClient.AnalyzeText
func (a *AIClientAdapter) AnalyzeText(ctx context.Context, request *TextAnalysisRequest) (*TextAnalysisResponse, error) {
	systemPrompt := textAnalysisSystemPrompt

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

	resp, err := a.client.Complete(ctx, systemPrompt, userPrompt, 0.3)
	if err != nil {
		return nil, fmt.Errorf("claude API call failed: %w", err)
	}

	// Parse the response
	var result TextAnalysisResponse
	text := resp.GetText()
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		extracted := extractJSONFromText(text)
		if err := json.Unmarshal([]byte(extracted), &result); err != nil {
			return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
		}
	}

	return &result, nil
}

// extractJSONFromText tries to extract JSON from text that may contain markdown
func extractJSONFromText(text string) string {
	// Try to find JSON in markdown code blocks
	start := -1
	end := -1

	// Look for ```json or ``` block
	for i := 0; i < len(text)-3; i++ {
		if text[i:i+3] == "```" {
			if start == -1 {
				// Find the end of this line
				for j := i + 3; j < len(text); j++ {
					if text[j] == '\n' {
						start = j + 1
						break
					}
				}
			} else {
				end = i
				break
			}
		}
	}

	if start != -1 && end != -1 {
		return text[start:end]
	}

	// Try to find JSON object directly
	braceStart := -1
	braceCount := 0
	for i, c := range text {
		if c == '{' {
			if braceStart == -1 {
				braceStart = i
			}
			braceCount++
		} else if c == '}' {
			braceCount--
			if braceCount == 0 && braceStart != -1 {
				return text[braceStart : i+1]
			}
		}
	}

	return text
}

// System prompts with security rules
const documentAnalysisSystemPrompt = `You are an expert analyst for Austrian business and tax documents.
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

const textAnalysisSystemPrompt = `You are a helpful assistant that analyzes text and extracts key information.

Your task is to:
1. Summarize the main points
2. Identify any required actions or tasks
3. Rate your confidence in the analysis

IMPORTANT SECURITY RULES:
- Never include any credentials, passwords, or PINs in your response
- Never include personal identification numbers in your response
- Focus on the content's purpose, not personal data

Respond ONLY with valid JSON. Do not use markdown formatting.`

// MockAIClient is a mock implementation for testing
type MockAIClient struct {
	AnalyzeFunc     func(ctx context.Context, request *AnalysisRequest) (*AnalysisResponse, error)
	AnalyzeTextFunc func(ctx context.Context, request *TextAnalysisRequest) (*TextAnalysisResponse, error)
}

// Analyze calls the mock function
func (m *MockAIClient) Analyze(ctx context.Context, request *AnalysisRequest) (*AnalysisResponse, error) {
	if m.AnalyzeFunc != nil {
		return m.AnalyzeFunc(ctx, request)
	}
	return &AnalysisResponse{
		Summary:      "Mock analysis summary",
		DocumentType: "mock",
		Confidence:   0.9,
	}, nil
}

// AnalyzeText calls the mock function
func (m *MockAIClient) AnalyzeText(ctx context.Context, request *TextAnalysisRequest) (*TextAnalysisResponse, error) {
	if m.AnalyzeTextFunc != nil {
		return m.AnalyzeTextFunc(ctx, request)
	}
	return &TextAnalysisResponse{
		Summary:    "Mock text analysis summary",
		Confidence: 0.9,
	}, nil
}

// NullAuditLogger is a no-op audit logger for testing or when audit is disabled
type NullAuditLogger struct{}

// LogAIRequest implements AuditLogger but does nothing
func (n *NullAuditLogger) LogAIRequest(ctx context.Context, entry *AIAuditEntry) error {
	return nil
}

// CreateSecureGateway creates a fully configured security gateway
// This is the recommended way to create a gateway with all security controls enabled
func CreateSecureGateway(claudeClient ClaudeClient, auditLogger AuditLogger) *Gateway {
	adapter := NewAIClientAdapter(claudeClient)
	config := DefaultGatewayConfig()
	config.StrictValidation = true
	config.RedactSensitiveData = true

	if auditLogger == nil {
		auditLogger = &NullAuditLogger{}
	}

	return NewGateway(adapter, config, auditLogger)
}

// CreateTestGateway creates a gateway with a mock AI client for testing
func CreateTestGateway() (*Gateway, *MockAIClient) {
	mockClient := &MockAIClient{}
	config := DefaultGatewayConfig()
	config.StrictValidation = true

	gateway := NewGateway(mockClient, config, &NullAuditLogger{})
	return gateway, mockClient
}

// Helper function to create a test analysis request
func NewTestAnalysisRequest(tenantID, userID uuid.UUID, text string) *AnalysisRequest {
	return &AnalysisRequest{
		DocumentID:   uuid.New(),
		DocumentText: text,
		TenantID:     tenantID,
		UserID:       userID,
	}
}
