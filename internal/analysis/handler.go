package analysis

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler provides HTTP endpoints for document analysis
type Handler struct {
	service *Service
}

// NewHandler creates a new analysis handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the router for analysis endpoints
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Document analysis
	r.Post("/documents/{documentId}/analyze", h.AnalyzeDocument)
	r.Get("/documents/{documentId}/analysis", h.GetDocumentAnalysis)
	r.Get("/documents/{documentId}/deadlines", h.GetDocumentDeadlines)
	r.Get("/documents/{documentId}/amounts", h.GetDocumentAmounts)
	r.Get("/documents/{documentId}/action-items", h.GetDocumentActionItems)
	r.Get("/documents/{documentId}/suggestions", h.GetDocumentSuggestions)

	// Analysis management
	r.Get("/analyses", h.ListAnalyses)
	r.Get("/analyses/{analysisId}", h.GetAnalysis)
	r.Get("/analyses/stats", h.GetAnalysisStats)

	// Deadlines
	r.Get("/deadlines/upcoming", h.GetUpcomingDeadlines)
	r.Put("/deadlines/{deadlineId}", h.UpdateDeadline)
	r.Post("/deadlines/{deadlineId}/acknowledge", h.AcknowledgeDeadline)

	// Amounts
	r.Put("/amounts/{amountId}", h.UpdateAmount)
	r.Delete("/amounts/{amountId}", h.DeleteAmount)

	// Action items
	r.Get("/action-items/pending", h.GetPendingActionItems)
	r.Put("/action-items/{itemId}", h.UpdateActionItem)
	r.Delete("/action-items/{itemId}", h.DeleteActionItem)
	r.Post("/action-items/{itemId}/complete", h.CompleteActionItem)
	r.Post("/action-items/{itemId}/cancel", h.CancelActionItem)

	// Suggestions
	r.Post("/suggestions/{suggestionId}/use", h.UseSuggestion)
	r.Post("/documents/{documentId}/suggest-response", h.GenerateSuggestion)

	// Response templates
	r.Get("/response-templates", h.ListResponseTemplates)
	r.Post("/response-templates", h.CreateResponseTemplate)
	r.Get("/response-templates/{templateId}", h.GetResponseTemplate)
	r.Put("/response-templates/{templateId}", h.UpdateResponseTemplate)
	r.Delete("/response-templates/{templateId}", h.DeleteResponseTemplate)

	// Quick analysis (without storing)
	r.Post("/quick/classify", h.QuickClassify)
	r.Post("/quick/summarize", h.QuickSummarize)
	r.Post("/quick/deadlines", h.QuickExtractDeadlines)

	// Direct PDF analysis
	r.Post("/analyze-pdf", h.AnalyzePDF)

	return r
}

// AnalyzeDocumentRequest represents the analysis request body
type AnalyzeDocumentRequest struct {
	IncludeOCR         *bool `json:"include_ocr,omitempty"`
	IncludeClassify    *bool `json:"include_classify,omitempty"`
	IncludeSummary     *bool `json:"include_summary,omitempty"`
	IncludeDeadlines   *bool `json:"include_deadlines,omitempty"`
	IncludeAmounts     *bool `json:"include_amounts,omitempty"`
	IncludeActionItems *bool `json:"include_action_items,omitempty"`
	IncludeSuggestions *bool `json:"include_suggestions,omitempty"`
}

// AnalyzeDocument initiates document analysis
func (h *Handler) AnalyzeDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	documentID, err := uuid.Parse(chi.URLParam(r, "documentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid document ID")
		return
	}

	tenantID := getTenantID(r)
	if tenantID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "Missing tenant context")
		return
	}

	// Parse options
	opts := DefaultOptions()
	if r.ContentLength > 0 {
		var req AnalyzeDocumentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.IncludeOCR != nil {
				opts.IncludeOCR = *req.IncludeOCR
			}
			if req.IncludeClassify != nil {
				opts.IncludeClassify = *req.IncludeClassify
			}
			if req.IncludeSummary != nil {
				opts.IncludeSummary = *req.IncludeSummary
			}
			if req.IncludeDeadlines != nil {
				opts.IncludeDeadlines = *req.IncludeDeadlines
			}
			if req.IncludeAmounts != nil {
				opts.IncludeAmounts = *req.IncludeAmounts
			}
			if req.IncludeActionItems != nil {
				opts.IncludeActionItems = *req.IncludeActionItems
			}
			if req.IncludeSuggestions != nil {
				opts.IncludeSuggestions = *req.IncludeSuggestions
			}
		}
	}

	result, err := h.service.AnalyzeDocument(ctx, documentID, tenantID, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetDocumentAnalysis returns the analysis for a document
func (h *Handler) GetDocumentAnalysis(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	documentID, err := uuid.Parse(chi.URLParam(r, "documentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid document ID")
		return
	}

	result, err := h.service.GetFullAnalysis(ctx, documentID)
	if err != nil {
		if err == ErrAnalysisNotFound {
			writeError(w, http.StatusNotFound, "Analysis not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetDocumentDeadlines returns deadlines for a document
func (h *Handler) GetDocumentDeadlines(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	documentID, err := uuid.Parse(chi.URLParam(r, "documentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid document ID")
		return
	}

	deadlines, err := h.service.repo.GetDeadlinesByDocument(ctx, documentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deadlines": deadlines,
		"count":     len(deadlines),
	})
}

// GetDocumentAmounts returns amounts for a document
func (h *Handler) GetDocumentAmounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	documentID, err := uuid.Parse(chi.URLParam(r, "documentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid document ID")
		return
	}

	amounts, err := h.service.repo.GetAmountsByDocument(ctx, documentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"amounts": amounts,
		"count":   len(amounts),
	})
}

// GetDocumentActionItems returns action items for a document
func (h *Handler) GetDocumentActionItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	documentID, err := uuid.Parse(chi.URLParam(r, "documentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid document ID")
		return
	}

	items, err := h.service.repo.GetActionItemsByDocument(ctx, documentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"action_items": items,
		"count":        len(items),
	})
}

// GetDocumentSuggestions returns suggestions for a document
func (h *Handler) GetDocumentSuggestions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	documentID, err := uuid.Parse(chi.URLParam(r, "documentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid document ID")
		return
	}

	suggestions, err := h.service.repo.GetSuggestionsByDocument(ctx, documentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"suggestions": suggestions,
		"count":       len(suggestions),
	})
}

// ListAnalyses returns all analyses for a tenant
func (h *Handler) ListAnalyses(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := getTenantID(r)
	if tenantID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "Missing tenant context")
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	analyses, total, err := h.service.ListAnalyses(ctx, tenantID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"analyses": analyses,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetAnalysis returns a single analysis by ID
func (h *Handler) GetAnalysis(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	analysisID, err := uuid.Parse(chi.URLParam(r, "analysisId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid analysis ID")
		return
	}

	analysis, err := h.service.GetAnalysis(ctx, analysisID)
	if err != nil {
		if err == ErrAnalysisNotFound {
			writeError(w, http.StatusNotFound, "Analysis not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, analysis)
}

// GetAnalysisStats returns analysis statistics
func (h *Handler) GetAnalysisStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := getTenantID(r)
	if tenantID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "Missing tenant context")
		return
	}

	stats, err := h.service.GetStats(ctx, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// GetUpcomingDeadlines returns upcoming deadlines
func (h *Handler) GetUpcomingDeadlines(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := getTenantID(r)
	if tenantID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "Missing tenant context")
		return
	}

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	deadlines, err := h.service.GetUpcomingDeadlines(ctx, tenantID, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deadlines": deadlines,
		"count":     len(deadlines),
		"days":      days,
	})
}

// AcknowledgeDeadline acknowledges a deadline
func (h *Handler) AcknowledgeDeadline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	deadlineID, err := uuid.Parse(chi.URLParam(r, "deadlineId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid deadline ID")
		return
	}

	if err := h.service.AcknowledgeDeadline(ctx, deadlineID); err != nil {
		if err == ErrDeadlineNotFound {
			writeError(w, http.StatusNotFound, "Deadline not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

// GetPendingActionItems returns pending action items
func (h *Handler) GetPendingActionItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := getTenantID(r)
	if tenantID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "Missing tenant context")
		return
	}

	items, err := h.service.GetPendingActionItems(ctx, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"action_items": items,
		"count":        len(items),
	})
}

// CompleteActionItem marks an action item as completed
func (h *Handler) CompleteActionItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid action item ID")
		return
	}

	if err := h.service.CompleteActionItem(ctx, itemID); err != nil {
		if err == ErrActionItemNotFound {
			writeError(w, http.StatusNotFound, "Action item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

// CancelActionItem marks an action item as cancelled
func (h *Handler) CancelActionItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid action item ID")
		return
	}

	if err := h.service.CancelActionItem(ctx, itemID); err != nil {
		if err == ErrActionItemNotFound {
			writeError(w, http.StatusNotFound, "Action item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// UseSuggestion marks a suggestion as used
func (h *Handler) UseSuggestion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	suggestionID, err := uuid.Parse(chi.URLParam(r, "suggestionId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid suggestion ID")
		return
	}

	if err := h.service.UseSuggestion(ctx, suggestionID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "marked_as_used"})
}

// QuickClassifyRequest represents a quick classify request
type QuickClassifyRequest struct {
	Text  string `json:"text"`
	Title string `json:"title,omitempty"`
}

// QuickClassify performs classification without storing
func (h *Handler) QuickClassify(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req QuickClassifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "Text is required")
		return
	}

	result, err := h.service.QuickClassify(ctx, req.Text, req.Title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// QuickSummarizeRequest represents a quick summarize request
type QuickSummarizeRequest struct {
	Text string `json:"text"`
}

// QuickSummarize performs summarization without storing
func (h *Handler) QuickSummarize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req QuickSummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "Text is required")
		return
	}

	result, err := h.service.QuickSummary(ctx, req.Text)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// QuickExtractDeadlines extracts deadlines without storing
func (h *Handler) QuickExtractDeadlines(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req QuickSummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "Text is required")
		return
	}

	deadlines, err := h.service.QuickExtractDeadlines(ctx, req.Text)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deadlines": deadlines,
		"count":     len(deadlines),
	})
}

// AnalyzePDF analyzes an uploaded PDF
func (h *Handler) AnalyzePDF(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := getTenantID(r)
	if tenantID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "Missing tenant context")
		return
	}

	// Limit upload size to 10MB
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

	// Parse multipart form
	if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
		writeError(w, http.StatusBadRequest, "File too large or invalid form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	// Read PDF data
	pdfData, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	// Parse options from query params
	opts := DefaultOptions()
	if r.URL.Query().Get("classify_only") == "true" {
		opts = AnalysisOptions{IncludeClassify: true}
	}
	if r.URL.Query().Get("summary_only") == "true" {
		opts = AnalysisOptions{IncludeSummary: true}
	}

	result, err := h.service.ProcessPDFBytes(ctx, tenantID, pdfData, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Helper functions

func getTenantID(r *http.Request) uuid.UUID {
	// Get from context (set by auth middleware)
	if id, ok := r.Context().Value("tenant_id").(uuid.UUID); ok {
		return id
	}
	// Try header for API access
	if id := r.Header.Get("X-Tenant-ID"); id != "" {
		if parsed, err := uuid.Parse(id); err == nil {
			return parsed
		}
	}
	return uuid.Nil
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// UpdateDeadlineRequest represents a deadline update request
type UpdateDeadlineRequest struct {
	Date            *string  `json:"date,omitempty"`
	Description     *string  `json:"description,omitempty"`
	IsAcknowledged  *bool    `json:"is_acknowledged,omitempty"`
	ManuallySet     *bool    `json:"manually_set,omitempty"`
	CorrectedByUser *bool    `json:"corrected_by_user,omitempty"`
	Notes           *string  `json:"notes,omitempty"`
}

// UpdateDeadline updates a deadline (T029 - manual deadline correction)
func (h *Handler) UpdateDeadline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	deadlineID, err := uuid.Parse(chi.URLParam(r, "deadlineId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid deadline ID")
		return
	}

	var req UpdateDeadlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	deadline, err := h.service.UpdateDeadline(ctx, deadlineID, &req)
	if err != nil {
		if err == ErrDeadlineNotFound {
			writeError(w, http.StatusNotFound, "Deadline not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, deadline)
}

// UpdateActionItemRequest represents an action item update request
type UpdateActionItemRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Priority    *string `json:"priority,omitempty"`
	Status      *string `json:"status,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	AssignedTo  *string `json:"assigned_to,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

// UpdateActionItem updates an action item (T045)
func (h *Handler) UpdateActionItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid action item ID")
		return
	}

	var req UpdateActionItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	item, err := h.service.UpdateActionItem(ctx, itemID, &req)
	if err != nil {
		if err == ErrActionItemNotFound {
			writeError(w, http.StatusNotFound, "Action item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, item)
}

// DeleteActionItem deletes an action item (T046)
func (h *Handler) DeleteActionItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid action item ID")
		return
	}

	if err := h.service.DeleteActionItem(ctx, itemID); err != nil {
		if err == ErrActionItemNotFound {
			writeError(w, http.StatusNotFound, "Action item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GenerateSuggestionRequest represents a suggestion generation request
type GenerateSuggestionRequest struct {
	Context string `json:"context,omitempty"`
	Style   string `json:"style,omitempty"` // formal, informal, technical
}

// GenerateSuggestion generates a response suggestion for a document (T059)
func (h *Handler) GenerateSuggestion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	documentID, err := uuid.Parse(chi.URLParam(r, "documentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid document ID")
		return
	}

	tenantID := getTenantID(r)
	if tenantID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "Missing tenant context")
		return
	}

	var req GenerateSuggestionRequest
	if r.ContentLength > 0 {
		json.NewDecoder(r.Body).Decode(&req)
	}

	suggestion, err := h.service.GenerateSuggestion(ctx, documentID, tenantID, req.Context, req.Style)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Add legal disclaimer (T061)
	response := map[string]interface{}{
		"suggestion": suggestion,
		"disclaimer": "HINWEIS: Diese Antwortvorschläge wurden automatisch generiert und stellen keine Rechtsberatung dar. " +
			"Bitte überprüfen Sie alle Vorschläge sorgfältig vor der Verwendung. " +
			"Für verbindliche Auskünfte wenden Sie sich bitte an Ihren Steuerberater oder Rechtsanwalt.",
	}

	writeJSON(w, http.StatusOK, response)
}

// ResponseTemplateRequest represents a response template request
type ResponseTemplateRequest struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Content     string   `json:"content"`
	Description string   `json:"description,omitempty"`
	Variables   []string `json:"variables,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// ListResponseTemplates lists all response templates (T063)
func (h *Handler) ListResponseTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := getTenantID(r)
	if tenantID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "Missing tenant context")
		return
	}

	category := r.URL.Query().Get("category")

	templates, err := h.service.ListResponseTemplates(ctx, tenantID, category)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	})
}

// CreateResponseTemplate creates a new response template (T062)
func (h *Handler) CreateResponseTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := getTenantID(r)
	if tenantID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "Missing tenant context")
		return
	}

	var req ResponseTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.Content == "" {
		writeError(w, http.StatusBadRequest, "Name and content are required")
		return
	}

	template, err := h.service.CreateResponseTemplate(ctx, tenantID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, template)
}

// GetResponseTemplate gets a single response template
func (h *Handler) GetResponseTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	templateID, err := uuid.Parse(chi.URLParam(r, "templateId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid template ID")
		return
	}

	template, err := h.service.GetResponseTemplate(ctx, templateID)
	if err != nil {
		if err == ErrTemplateNotFound {
			writeError(w, http.StatusNotFound, "Template not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, template)
}

// UpdateResponseTemplate updates a response template
func (h *Handler) UpdateResponseTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	templateID, err := uuid.Parse(chi.URLParam(r, "templateId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid template ID")
		return
	}

	var req ResponseTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	template, err := h.service.UpdateResponseTemplate(ctx, templateID, &req)
	if err != nil {
		if err == ErrTemplateNotFound {
			writeError(w, http.StatusNotFound, "Template not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, template)
}

// DeleteResponseTemplate deletes a response template
func (h *Handler) DeleteResponseTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	templateID, err := uuid.Parse(chi.URLParam(r, "templateId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid template ID")
		return
	}

	if err := h.service.DeleteResponseTemplate(ctx, templateID); err != nil {
		if err == ErrTemplateNotFound {
			writeError(w, http.StatusNotFound, "Template not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateAmountRequest represents an amount update request (T074)
type UpdateAmountRequest struct {
	Amount      *float64 `json:"amount,omitempty"`
	Currency    *string  `json:"currency,omitempty"`
	AmountType  *string  `json:"amount_type,omitempty"`
	Description *string  `json:"description,omitempty"`
	DueDate     *string  `json:"due_date,omitempty"`
	Notes       *string  `json:"notes,omitempty"`
}

// UpdateAmount updates an extracted amount (T074 - manual correction)
func (h *Handler) UpdateAmount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	amountID, err := uuid.Parse(chi.URLParam(r, "amountId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid amount ID")
		return
	}

	var req UpdateAmountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	amount, err := h.service.UpdateAmount(ctx, amountID, &req)
	if err != nil {
		if err == ErrAmountNotFound {
			writeError(w, http.StatusNotFound, "Amount not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, amount)
}

// DeleteAmount deletes an extracted amount
func (h *Handler) DeleteAmount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	amountID, err := uuid.Parse(chi.URLParam(r, "amountId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid amount ID")
		return
	}

	if err := h.service.DeleteAmount(ctx, amountID); err != nil {
		if err == ErrAmountNotFound {
			writeError(w, http.StatusNotFound, "Amount not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
