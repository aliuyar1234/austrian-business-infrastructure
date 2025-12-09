package security

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

// Handler handles AI-related HTTP requests with security controls
type Handler struct {
	gateway *Gateway
}

// NewHandler creates a new AI security handler
func NewHandler(gateway *Gateway) *Handler {
	return &Handler{gateway: gateway}
}

// AnalyzeDocumentRequest is the HTTP request for document analysis
type AnalyzeDocumentRequest struct {
	DocumentID   string `json:"document_id"`
	DocumentText string `json:"document_text"`
	DocumentType string `json:"document_type,omitempty"`
	Prompt       string `json:"prompt,omitempty"`
}

// AnalyzeTextRequest is the HTTP request for text analysis
type AnalyzeTextRequest struct {
	Text   string `json:"text"`
	Prompt string `json:"prompt,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// AnalyzeDocument handles POST /api/v1/ai/analyze
// Requires authentication and extracts tenant_id/user_id from context
func (h *Handler) AnalyzeDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant and user from context (set by auth middleware)
	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing tenant context")
		return
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing user context")
		return
	}

	// Parse request
	var req AnalyzeDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "failed to parse request body")
		return
	}

	// Validate required fields
	if req.DocumentText == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "document_text is required")
		return
	}

	// Parse document ID if provided
	var docID uuid.UUID
	if req.DocumentID != "" {
		docID, err = uuid.Parse(req.DocumentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_field", "invalid document_id format")
			return
		}
	} else {
		docID = uuid.New() // Generate new ID for ad-hoc analysis
	}

	// Build analysis request
	analysisReq := &AnalysisRequest{
		DocumentID:   docID,
		DocumentText: req.DocumentText,
		DocumentType: req.DocumentType,
		Prompt:       req.Prompt,
		TenantID:     tenantID,
		UserID:       userID,
	}

	// Call the gateway
	response, err := h.gateway.Analyze(ctx, analysisReq)
	if err != nil {
		handleGatewayError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// AnalyzeText handles POST /api/v1/ai/analyze/text
func (h *Handler) AnalyzeText(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant and user from context
	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing tenant context")
		return
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing user context")
		return
	}

	// Parse request
	var req AnalyzeTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "failed to parse request body")
		return
	}

	// Validate required fields
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "text is required")
		return
	}

	// Build analysis request
	textReq := &TextAnalysisRequest{
		Text:     req.Text,
		Prompt:   req.Prompt,
		TenantID: tenantID,
		UserID:   userID,
	}

	// Call the gateway
	response, err := h.gateway.AnalyzeText(ctx, textReq)
	if err != nil {
		handleGatewayError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// CheckSafety handles POST /api/v1/ai/check-safety
// Pre-flight check to validate input before analysis
func (h *Handler) CheckSafety(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "failed to parse request body")
		return
	}

	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "text is required")
		return
	}

	// Check safety
	result := h.gateway.CheckInputSafety(req.Text)

	response := struct {
		Safe          bool   `json:"safe"`
		WasTruncated  bool   `json:"was_truncated"`
		WasFiltered   bool   `json:"was_filtered"`
		FilteredCount int    `json:"filtered_count"`
		OriginalLen   int    `json:"original_length"`
		SanitizedLen  int    `json:"sanitized_length"`
		SanitizedText string `json:"sanitized_text,omitempty"`
	}{
		Safe:          !result.WasFiltered,
		WasTruncated:  result.WasTruncated,
		WasFiltered:   result.WasFiltered,
		FilteredCount: result.FilteredCount,
		OriginalLen:   result.OriginalLen,
		SanitizedLen:  len(result.Text),
	}

	// Only include sanitized text if something changed
	if result.WasFiltered || result.WasTruncated {
		response.SanitizedText = result.Text
	}

	writeJSON(w, http.StatusOK, response)
}

// handleGatewayError converts gateway errors to HTTP responses
func handleGatewayError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInputTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "input_too_large", "input exceeds maximum size limit")
	case errors.Is(err, ErrInputContainsDangerousContent):
		writeError(w, http.StatusBadRequest, "dangerous_content", "input contains potentially dangerous content")
	case errors.Is(err, ErrOutputContainsSensitiveData):
		writeError(w, http.StatusInternalServerError, "sensitive_data_leak", "AI output contained sensitive data")
	case errors.Is(err, ErrOutputValidationFailed):
		writeError(w, http.StatusInternalServerError, "validation_failed", "AI output validation failed")
	case errors.Is(err, ErrAIRequestFailed):
		writeError(w, http.StatusServiceUnavailable, "ai_unavailable", "AI service temporarily unavailable")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

// Context key types
type contextKey string

const (
	contextKeyTenantID contextKey = "tenant_id"
	contextKeyUserID   contextKey = "user_id"
)

// getTenantIDFromContext extracts tenant ID from request context
func getTenantIDFromContext(ctx interface{ Value(any) any }) (uuid.UUID, error) {
	value := ctx.Value(contextKeyTenantID)
	if value == nil {
		// Try string key for compatibility
		value = ctx.Value("tenant_id")
	}
	if value == nil {
		return uuid.Nil, errors.New("tenant_id not in context")
	}

	switch v := value.(type) {
	case uuid.UUID:
		return v, nil
	case string:
		return uuid.Parse(v)
	default:
		return uuid.Nil, errors.New("invalid tenant_id type in context")
	}
}

// getUserIDFromContext extracts user ID from request context
func getUserIDFromContext(ctx interface{ Value(any) any }) (uuid.UUID, error) {
	value := ctx.Value(contextKeyUserID)
	if value == nil {
		// Try string key for compatibility
		value = ctx.Value("user_id")
	}
	if value == nil {
		return uuid.Nil, errors.New("user_id not in context")
	}

	switch v := value.(type) {
	case uuid.UUID:
		return v, nil
	case string:
		return uuid.Parse(v)
	default:
		return uuid.Nil, errors.New("invalid user_id type in context")
	}
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// RegisterRoutes registers AI endpoints with a router
// This is a helper for integration - actual registration depends on router implementation
type RouteRegistrar interface {
	Post(pattern string, handler http.HandlerFunc)
}

// RegisterRoutes registers the AI security handler routes
func (h *Handler) RegisterRoutes(r RouteRegistrar, authMiddleware func(http.Handler) http.Handler) {
	// Wrap handlers with auth middleware
	analyzeDoc := http.HandlerFunc(h.AnalyzeDocument)
	analyzeText := http.HandlerFunc(h.AnalyzeText)
	checkSafety := http.HandlerFunc(h.CheckSafety)

	if authMiddleware != nil {
		r.Post("/api/v1/ai/analyze", func(w http.ResponseWriter, r *http.Request) {
			authMiddleware(analyzeDoc).ServeHTTP(w, r)
		})
		r.Post("/api/v1/ai/analyze/text", func(w http.ResponseWriter, r *http.Request) {
			authMiddleware(analyzeText).ServeHTTP(w, r)
		})
		r.Post("/api/v1/ai/check-safety", func(w http.ResponseWriter, r *http.Request) {
			authMiddleware(checkSafety).ServeHTTP(w, r)
		})
	} else {
		r.Post("/api/v1/ai/analyze", h.AnalyzeDocument)
		r.Post("/api/v1/ai/analyze/text", h.AnalyzeText)
		r.Post("/api/v1/ai/check-safety", h.CheckSafety)
	}
}
