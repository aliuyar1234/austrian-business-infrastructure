package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrInputTooLarge indicates the input exceeds size limits
	ErrInputTooLarge = errors.New("input exceeds maximum size limit")
	// ErrInputContainsDangerousContent indicates prompt injection attempt
	ErrInputContainsDangerousContent = errors.New("input contains potentially dangerous content")
	// ErrOutputContainsSensitiveData indicates AI output leaked sensitive data
	ErrOutputContainsSensitiveData = errors.New("AI output contains sensitive data")
	// ErrOutputValidationFailed indicates the output doesn't match expected schema
	ErrOutputValidationFailed = errors.New("AI output validation failed")
	// ErrAIRequestFailed indicates the AI request failed
	ErrAIRequestFailed = errors.New("AI request failed")
)

// AIClient interface for the underlying AI client
type AIClient interface {
	// Analyze sends a document analysis request to the AI
	Analyze(ctx context.Context, request *AnalysisRequest) (*AnalysisResponse, error)
	// AnalyzeText sends a text analysis request
	AnalyzeText(ctx context.Context, request *TextAnalysisRequest) (*TextAnalysisResponse, error)
}

// AnalysisRequest represents a document analysis request
type AnalysisRequest struct {
	DocumentID   uuid.UUID `json:"document_id"`
	DocumentText string    `json:"document_text"`
	DocumentType string    `json:"document_type,omitempty"`
	Prompt       string    `json:"prompt,omitempty"`
	TenantID     uuid.UUID `json:"tenant_id"`
	UserID       uuid.UUID `json:"user_id"`
}

// AnalysisResponse represents the AI analysis response
type AnalysisResponse struct {
	Summary      string   `json:"summary"`
	DocumentType string   `json:"document_type"`
	Deadline     string   `json:"deadline,omitempty"`
	Amount       *float64 `json:"amount,omitempty"`
	ActionItems  []string `json:"action_items,omitempty"`
	Confidence   float64  `json:"confidence"`
	ProcessedAt  time.Time `json:"processed_at"`
}

// TextAnalysisRequest represents a text-only analysis request
type TextAnalysisRequest struct {
	Text     string    `json:"text"`
	Prompt   string    `json:"prompt,omitempty"`
	TenantID uuid.UUID `json:"tenant_id"`
	UserID   uuid.UUID `json:"user_id"`
}

// TextAnalysisResponse represents text analysis response
type TextAnalysisResponse struct {
	Summary     string    `json:"summary"`
	ActionItems []string  `json:"action_items,omitempty"`
	Confidence  float64   `json:"confidence"`
	ProcessedAt time.Time `json:"processed_at"`
}

// GatewayConfig holds configuration for the AI Gateway
type GatewayConfig struct {
	MaxInputSize        int  // Maximum input size in bytes
	AllowExtraFields    bool // Allow extra fields in AI output
	StrictValidation    bool // Fail on any validation error
	RedactSensitiveData bool // Redact sensitive data instead of failing
}

// DefaultGatewayConfig returns sensible defaults
func DefaultGatewayConfig() *GatewayConfig {
	return &GatewayConfig{
		MaxInputSize:        DefaultMaxInputLength,
		AllowExtraFields:    false,
		StrictValidation:    true,
		RedactSensitiveData: true,
	}
}

// Gateway wraps an AI client with security controls
type Gateway struct {
	client            AIClient
	sanitizer         *Sanitizer
	suspiciousDetector *SuspiciousOutputDetector
	validator         *OutputValidator
	config            *GatewayConfig
	auditLogger       AuditLogger
}

// AuditLogger interface for logging AI interactions
type AuditLogger interface {
	LogAIRequest(ctx context.Context, entry *AIAuditEntry) error
}

// AIAuditEntry represents an audit log entry for AI interactions
type AIAuditEntry struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	UserID          uuid.UUID `json:"user_id"`
	RequestType     string    `json:"request_type"` // "document_analysis", "text_analysis"
	InputSanitized  bool      `json:"input_sanitized"`
	InputFiltered   int       `json:"input_filtered_count"`
	OutputValidated bool      `json:"output_validated"`
	SuspiciousFound bool      `json:"suspicious_found"`
	SuspiciousTypes []string  `json:"suspicious_types,omitempty"`
	Success         bool      `json:"success"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	DurationMs      int64     `json:"duration_ms"`
	Timestamp       time.Time `json:"timestamp"`
}

// NewGateway creates a new AI Gateway with security controls
func NewGateway(client AIClient, config *GatewayConfig, auditLogger AuditLogger) *Gateway {
	if config == nil {
		config = DefaultGatewayConfig()
	}

	validator := NewOutputValidator()
	if config.AllowExtraFields {
		validator.WithAllowExtraFields(true)
	}

	return &Gateway{
		client:            client,
		sanitizer:         NewSanitizer(config.MaxInputSize),
		suspiciousDetector: NewSuspiciousOutputDetector(),
		validator:         validator,
		config:            config,
		auditLogger:       auditLogger,
	}
}

// Analyze performs a secure document analysis
func (g *Gateway) Analyze(ctx context.Context, request *AnalysisRequest) (*AnalysisResponse, error) {
	startTime := time.Now()
	auditEntry := &AIAuditEntry{
		ID:          uuid.New(),
		TenantID:    request.TenantID,
		UserID:      request.UserID,
		RequestType: "document_analysis",
		Timestamp:   startTime,
	}

	defer func() {
		auditEntry.DurationMs = time.Since(startTime).Milliseconds()
		if g.auditLogger != nil {
			// Best effort logging - don't fail the request if audit fails
			_ = g.auditLogger.LogAIRequest(ctx, auditEntry)
		}
	}()

	// Sanitize document text
	sanitizedDoc := g.sanitizer.Sanitize(request.DocumentText)
	auditEntry.InputSanitized = sanitizedDoc.WasFiltered || sanitizedDoc.WasTruncated
	auditEntry.InputFiltered = sanitizedDoc.FilteredCount

	if sanitizedDoc.WasFiltered && g.config.StrictValidation {
		auditEntry.Success = false
		auditEntry.ErrorMessage = "input contained dangerous content"
		return nil, ErrInputContainsDangerousContent
	}

	// Also sanitize optional prompt
	if request.Prompt != "" {
		sanitizedPrompt := g.sanitizer.Sanitize(request.Prompt)
		if sanitizedPrompt.WasFiltered && g.config.StrictValidation {
			auditEntry.Success = false
			auditEntry.ErrorMessage = "prompt contained dangerous content"
			return nil, ErrInputContainsDangerousContent
		}
		request.Prompt = sanitizedPrompt.Text
	}

	// Update request with sanitized text
	request.DocumentText = sanitizedDoc.Text

	// Call the underlying AI client
	response, err := g.client.Analyze(ctx, request)
	if err != nil {
		auditEntry.Success = false
		auditEntry.ErrorMessage = err.Error()
		return nil, fmt.Errorf("%w: %v", ErrAIRequestFailed, err)
	}

	// Validate the response
	validatedResponse, err := g.validateAnalysisResponse(response, auditEntry)
	if err != nil {
		return nil, err
	}

	auditEntry.Success = true
	return validatedResponse, nil
}

// AnalyzeText performs a secure text analysis
func (g *Gateway) AnalyzeText(ctx context.Context, request *TextAnalysisRequest) (*TextAnalysisResponse, error) {
	startTime := time.Now()
	auditEntry := &AIAuditEntry{
		ID:          uuid.New(),
		TenantID:    request.TenantID,
		UserID:      request.UserID,
		RequestType: "text_analysis",
		Timestamp:   startTime,
	}

	defer func() {
		auditEntry.DurationMs = time.Since(startTime).Milliseconds()
		if g.auditLogger != nil {
			_ = g.auditLogger.LogAIRequest(ctx, auditEntry)
		}
	}()

	// Sanitize input text
	sanitizedText := g.sanitizer.Sanitize(request.Text)
	auditEntry.InputSanitized = sanitizedText.WasFiltered || sanitizedText.WasTruncated
	auditEntry.InputFiltered = sanitizedText.FilteredCount

	if sanitizedText.WasFiltered && g.config.StrictValidation {
		auditEntry.Success = false
		auditEntry.ErrorMessage = "input contained dangerous content"
		return nil, ErrInputContainsDangerousContent
	}

	// Sanitize optional prompt
	if request.Prompt != "" {
		sanitizedPrompt := g.sanitizer.Sanitize(request.Prompt)
		if sanitizedPrompt.WasFiltered && g.config.StrictValidation {
			auditEntry.Success = false
			auditEntry.ErrorMessage = "prompt contained dangerous content"
			return nil, ErrInputContainsDangerousContent
		}
		request.Prompt = sanitizedPrompt.Text
	}

	// Update request with sanitized text
	request.Text = sanitizedText.Text

	// Call the underlying AI client
	response, err := g.client.AnalyzeText(ctx, request)
	if err != nil {
		auditEntry.Success = false
		auditEntry.ErrorMessage = err.Error()
		return nil, fmt.Errorf("%w: %v", ErrAIRequestFailed, err)
	}

	// Validate the response
	validatedResponse, err := g.validateTextAnalysisResponse(response, auditEntry)
	if err != nil {
		return nil, err
	}

	auditEntry.Success = true
	return validatedResponse, nil
}

// validateAnalysisResponse validates and sanitizes the AI response
func (g *Gateway) validateAnalysisResponse(response *AnalysisResponse, auditEntry *AIAuditEntry) (*AnalysisResponse, error) {
	// Convert to JSON for validation
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		auditEntry.ErrorMessage = "failed to marshal response"
		return nil, fmt.Errorf("%w: %v", ErrOutputValidationFailed, err)
	}

	// Validate against schema
	validationResult := g.validator.ValidateJSON(jsonBytes, DocumentAnalysisSchema)
	auditEntry.OutputValidated = validationResult.Valid

	if !validationResult.Valid && g.config.StrictValidation {
		auditEntry.ErrorMessage = fmt.Sprintf("validation errors: %v", validationResult.Errors)
		return nil, fmt.Errorf("%w: %v", ErrOutputValidationFailed, validationResult.Errors)
	}

	// Check for suspicious content in summary
	suspiciousResult := g.suspiciousDetector.Check(response.Summary)
	auditEntry.SuspiciousFound = suspiciousResult.IsSuspicious
	auditEntry.SuspiciousTypes = suspiciousResult.SuspiciousTypes

	if suspiciousResult.IsSuspicious {
		if g.config.RedactSensitiveData {
			// Redact and continue
			response.Summary = suspiciousResult.RedactedContent
		} else {
			auditEntry.ErrorMessage = "output contains sensitive data"
			return nil, ErrOutputContainsSensitiveData
		}
	}

	// Check action items for suspicious content
	for i, item := range response.ActionItems {
		itemResult := g.suspiciousDetector.Check(item)
		if itemResult.IsSuspicious {
			if g.config.RedactSensitiveData {
				response.ActionItems[i] = itemResult.RedactedContent
			} else {
				auditEntry.ErrorMessage = "action item contains sensitive data"
				return nil, ErrOutputContainsSensitiveData
			}
		}
	}

	response.ProcessedAt = time.Now()
	return response, nil
}

// validateTextAnalysisResponse validates and sanitizes text analysis response
func (g *Gateway) validateTextAnalysisResponse(response *TextAnalysisResponse, auditEntry *AIAuditEntry) (*TextAnalysisResponse, error) {
	auditEntry.OutputValidated = true // Text analysis has looser validation

	// Check for suspicious content in summary
	suspiciousResult := g.suspiciousDetector.Check(response.Summary)
	auditEntry.SuspiciousFound = suspiciousResult.IsSuspicious
	auditEntry.SuspiciousTypes = suspiciousResult.SuspiciousTypes

	if suspiciousResult.IsSuspicious {
		if g.config.RedactSensitiveData {
			response.Summary = suspiciousResult.RedactedContent
		} else {
			auditEntry.ErrorMessage = "output contains sensitive data"
			return nil, ErrOutputContainsSensitiveData
		}
	}

	// Check action items
	for i, item := range response.ActionItems {
		itemResult := g.suspiciousDetector.Check(item)
		if itemResult.IsSuspicious {
			if g.config.RedactSensitiveData {
				response.ActionItems[i] = itemResult.RedactedContent
			} else {
				auditEntry.ErrorMessage = "action item contains sensitive data"
				return nil, ErrOutputContainsSensitiveData
			}
		}
	}

	response.ProcessedAt = time.Now()
	return response, nil
}

// CheckInputSafety performs a pre-flight safety check without calling the AI
func (g *Gateway) CheckInputSafety(input string) *SanitizeResult {
	return g.sanitizer.Sanitize(input)
}

// GetConfig returns the current gateway configuration
func (g *Gateway) GetConfig() *GatewayConfig {
	return g.config
}
