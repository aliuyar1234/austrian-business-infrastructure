package zm

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/google/uuid"
)

// Handler handles ZM HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new ZM handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers ZM routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	// Admin-only: create, update, delete, submit, import (financial submissions)
	router.Handle("POST /api/v1/zm", requireAuth(requireAdmin(http.HandlerFunc(h.Create))))
	router.Handle("PUT /api/v1/zm/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.Update))))
	router.Handle("DELETE /api/v1/zm/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.Delete))))
	router.Handle("POST /api/v1/zm/{id}/submit", requireAuth(requireAdmin(http.HandlerFunc(h.Submit))))
	router.Handle("POST /api/v1/zm/import", requireAuth(requireAdmin(http.HandlerFunc(h.ImportCSV))))

	// Member access: read-only and validation
	router.Handle("GET /api/v1/zm", requireAuth(http.HandlerFunc(h.List)))
	router.Handle("GET /api/v1/zm/{id}", requireAuth(http.HandlerFunc(h.Get)))
	router.Handle("POST /api/v1/zm/{id}/validate", requireAuth(http.HandlerFunc(h.Validate)))
	router.Handle("GET /api/v1/zm/{id}/xml", requireAuth(http.HandlerFunc(h.GetXML)))
}

// CreateRequest represents the create ZM request
type CreateRequest struct {
	AccountID     string  `json:"account_id"`
	PeriodYear    int     `json:"period_year"`
	PeriodQuarter int     `json:"period_quarter"`
	Entries       []Entry `json:"entries"`
}

// Create handles POST /api/v1/zm
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		api.BadRequest(w, "invalid account_id")
		return
	}

	input := &CreateSubmissionInput{
		AccountID:     accountID,
		PeriodYear:    req.PeriodYear,
		PeriodQuarter: req.PeriodQuarter,
		Entries:       req.Entries,
	}

	submission, err := h.service.Create(r.Context(), tenantID, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusCreated, h.toResponse(submission))
}

// List handles GET /api/v1/zm
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	filter := ListFilter{
		TenantID: tenantID,
		Limit:    50,
		Offset:   0,
	}

	if accountIDStr := r.URL.Query().Get("account_id"); accountIDStr != "" {
		if accountID, err := uuid.Parse(accountIDStr); err == nil {
			filter.AccountID = &accountID
		}
	}

	if yearStr := r.URL.Query().Get("period_year"); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil {
			filter.PeriodYear = &year
		}
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	submissions, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		api.InternalError(w)
		return
	}

	items := make([]*SubmissionResponse, 0, len(submissions))
	for _, s := range submissions {
		items = append(items, h.toResponse(s))
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// Get handles GET /api/v1/zm/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid submission ID")
		return
	}

	submission, err := h.service.Get(r.Context(), id, tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toResponse(submission))
}

// UpdateRequest represents the update ZM request
type UpdateRequest struct {
	Entries []Entry `json:"entries"`
}

// Update handles PUT /api/v1/zm/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid submission ID")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	submission, err := h.service.Update(r.Context(), id, tenantID, &UpdateSubmissionInput{Entries: req.Entries})
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toResponse(submission))
}

// Delete handles DELETE /api/v1/zm/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid submission ID")
		return
	}

	if err := h.service.Delete(r.Context(), id, tenantID); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Validate handles POST /api/v1/zm/{id}/validate
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid submission ID")
		return
	}

	submission, err := h.service.Validate(r.Context(), id, tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toResponse(submission))
}

// SubmitRequest represents the submit ZM request
type SubmitRequest struct {
	DryRun bool `json:"dry_run"`
}

// Submit handles POST /api/v1/zm/{id}/submit
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	userID, err := h.getUserID(r)
	if err != nil {
		api.Unauthorized(w, "user not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid submission ID")
		return
	}

	var req SubmitRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.BadRequest(w, "invalid request body")
			return
		}
	}

	submission, err := h.service.Submit(r.Context(), id, tenantID, userID, req.DryRun)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toResponse(submission))
}

// GetXML handles GET /api/v1/zm/{id}/xml
func (h *Handler) GetXML(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid submission ID")
		return
	}

	xmlContent, err := h.service.GetXML(r.Context(), id, tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=zm.xml")
	w.WriteHeader(http.StatusOK)
	w.Write(xmlContent)
}

// ImportCSVRequest represents the import CSV request
type ImportCSVRequest struct {
	AccountID     string `json:"account_id"`
	PeriodYear    int    `json:"period_year"`
	PeriodQuarter int    `json:"period_quarter"`
}

// ImportCSV handles POST /api/v1/zm/import
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	// Check content type
	contentType := r.Header.Get("Content-Type")
	if contentType == "text/csv" || contentType == "application/csv" {
		// Direct CSV upload
		accountIDStr := r.URL.Query().Get("account_id")
		yearStr := r.URL.Query().Get("period_year")
		quarterStr := r.URL.Query().Get("period_quarter")

		accountID, err := uuid.Parse(accountIDStr)
		if err != nil {
			api.BadRequest(w, "invalid account_id")
			return
		}

		year, err := strconv.Atoi(yearStr)
		if err != nil {
			api.BadRequest(w, "invalid period_year")
			return
		}

		quarter, err := strconv.Atoi(quarterStr)
		if err != nil {
			api.BadRequest(w, "invalid period_quarter")
			return
		}

		csvData, err := io.ReadAll(r.Body)
		if err != nil {
			api.BadRequest(w, "failed to read CSV data")
			return
		}

		submission, err := h.service.ImportCSV(r.Context(), tenantID, accountID, year, quarter, csvData)
		if err != nil {
			h.handleError(w, err)
			return
		}

		api.JSONResponse(w, http.StatusCreated, h.toResponse(submission))
		return
	}

	// Multipart form upload
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		api.BadRequest(w, "failed to parse multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		api.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	accountIDStr := r.FormValue("account_id")
	yearStr := r.FormValue("period_year")
	quarterStr := r.FormValue("period_quarter")

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		api.BadRequest(w, "invalid account_id")
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		api.BadRequest(w, "invalid period_year")
		return
	}

	quarter, err := strconv.Atoi(quarterStr)
	if err != nil {
		api.BadRequest(w, "invalid period_quarter")
		return
	}

	csvData, err := io.ReadAll(file)
	if err != nil {
		api.BadRequest(w, "failed to read CSV data")
		return
	}

	submission, err := h.service.ImportCSV(r.Context(), tenantID, accountID, year, quarter, csvData)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusCreated, h.toResponse(submission))
}

// Helper methods

func (h *Handler) getTenantID(r *http.Request) (uuid.UUID, error) {
	tenantIDStr := api.GetTenantID(r.Context())
	if tenantIDStr == "" {
		return uuid.Nil, ErrAccountNotFound
	}
	return uuid.Parse(tenantIDStr)
}

func (h *Handler) getUserID(r *http.Request) (uuid.UUID, error) {
	userIDStr := api.GetUserID(r.Context())
	if userIDStr == "" {
		return uuid.Nil, ErrAccountNotFound
	}
	return uuid.Parse(userIDStr)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch err {
	case ErrSubmissionNotFound:
		api.NotFound(w, "submission not found")
	case ErrDuplicatePeriod:
		api.Conflict(w, "submission for this period already exists")
	case ErrInvalidQuarter:
		api.BadRequest(w, "quarter must be between 1 and 4")
	case ErrInvalidYear:
		api.BadRequest(w, "year must be between 2000 and 2100")
	case ErrSubmissionNotDraft:
		api.BadRequest(w, "submission is not in draft status")
	case ErrAccountNotFound:
		api.NotFound(w, "account not found")
	case ErrValidationFailed:
		api.BadRequest(w, "validation failed")
	case ErrNoEntries:
		api.BadRequest(w, "ZM must have at least one entry")
	case ErrSubmissionFailed:
		api.JSONError(w, http.StatusBadGateway, "submission to FinanzOnline failed", "FO_ERROR")
	default:
		api.InternalError(w)
	}
}

func (h *Handler) toResponse(s *Submission) *SubmissionResponse {
	resp := &SubmissionResponse{
		ID:               s.ID,
		AccountID:        s.AccountID,
		PeriodYear:       s.PeriodYear,
		PeriodQuarter:    s.PeriodQuarter,
		EntryCount:       s.EntryCount,
		TotalAmount:      s.TotalAmount,
		TotalAmountEUR:   float64(s.TotalAmount) / 100.0,
		ValidationStatus: s.ValidationStatus,
		ValidationErrors: s.ValidationErrors,
		Status:           s.Status,
		FOReference:      s.FOReference,
		CreatedAt:        s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Parse entries
	var entries []Entry
	if err := json.Unmarshal(s.Entries, &entries); err == nil {
		resp.Entries = entries
	}

	if s.SubmittedAt != nil {
		t := s.SubmittedAt.Format("2006-01-02T15:04:05Z")
		resp.SubmittedAt = &t
	}

	return resp
}
