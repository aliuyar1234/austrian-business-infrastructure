package uva

import (
	"encoding/json"
	"net/http"
	"strconv"

	"austrian-business-infrastructure/internal/api"
	"github.com/google/uuid"
)

// Handler handles UVA HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new UVA handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers UVA routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	// Admin-only: create, update, delete, submit (financial submissions)
	router.Handle("POST /api/v1/uva", requireAuth(requireAdmin(http.HandlerFunc(h.Create))))
	router.Handle("PUT /api/v1/uva/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.Update))))
	router.Handle("DELETE /api/v1/uva/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.Delete))))
	router.Handle("POST /api/v1/uva/{id}/submit", requireAuth(requireAdmin(http.HandlerFunc(h.Submit))))
	router.Handle("POST /api/v1/uva/batches", requireAuth(requireAdmin(http.HandlerFunc(h.CreateBatch))))

	// Member access: read-only and validation
	// Batches use separate path to avoid conflict with {id} wildcard
	router.Handle("GET /api/v1/uva", requireAuth(http.HandlerFunc(h.List)))
	router.Handle("GET /api/v1/uva/batches", requireAuth(http.HandlerFunc(h.ListBatches)))
	router.Handle("GET /api/v1/uva-batches/{batchID}", requireAuth(http.HandlerFunc(h.GetBatch)))
	router.Handle("GET /api/v1/uva/{id}", requireAuth(http.HandlerFunc(h.Get)))
	router.Handle("POST /api/v1/uva/{id}/validate", requireAuth(http.HandlerFunc(h.Validate)))
	router.Handle("GET /api/v1/uva/{id}/xml", requireAuth(http.HandlerFunc(h.GetXML)))
}

// CreateRequest represents the create UVA request
type CreateRequest struct {
	AccountID     string  `json:"account_id"`
	PeriodYear    int     `json:"period_year"`
	PeriodMonth   *int    `json:"period_month,omitempty"`
	PeriodQuarter *int    `json:"period_quarter,omitempty"`
	PeriodType    string  `json:"period_type"`
	Data          UVAData `json:"data"`
}

// Create handles POST /api/v1/uva
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
		PeriodMonth:   req.PeriodMonth,
		PeriodQuarter: req.PeriodQuarter,
		PeriodType:    req.PeriodType,
		Data:          req.Data,
	}

	submission, err := h.service.Create(r.Context(), tenantID, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusCreated, h.toResponse(submission))
}

// List handles GET /api/v1/uva
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

	if periodType := r.URL.Query().Get("period_type"); periodType != "" {
		filter.PeriodType = &periodType
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

// Get handles GET /api/v1/uva/{id}
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

// UpdateRequest represents the update UVA request
type UpdateRequest struct {
	Data UVAData `json:"data"`
}

// Update handles PUT /api/v1/uva/{id}
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

	submission, err := h.service.Update(r.Context(), id, tenantID, &UpdateSubmissionInput{Data: req.Data})
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toResponse(submission))
}

// Delete handles DELETE /api/v1/uva/{id}
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

// Validate handles POST /api/v1/uva/{id}/validate
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

// SubmitRequest represents the submit UVA request
type SubmitRequest struct {
	DryRun bool `json:"dry_run"`
}

// Submit handles POST /api/v1/uva/{id}/submit
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

// GetXML handles GET /api/v1/uva/{id}/xml
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
	w.Header().Set("Content-Disposition", "attachment; filename=uva.xml")
	w.WriteHeader(http.StatusOK)
	w.Write(xmlContent)
}

// Batch handlers

// CreateBatchRequest represents the create batch request
type CreateBatchRequest struct {
	Name          string   `json:"name"`
	AccountIDs    []string `json:"account_ids"`
	PeriodYear    int      `json:"period_year"`
	PeriodMonth   *int     `json:"period_month,omitempty"`
	PeriodQuarter *int     `json:"period_quarter,omitempty"`
	PeriodType    string   `json:"period_type"`
}

// CreateBatch handles POST /api/v1/uva/batches
func (h *Handler) CreateBatch(w http.ResponseWriter, r *http.Request) {
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

	var req CreateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	if len(req.AccountIDs) == 0 {
		api.BadRequest(w, "at least one account_id is required")
		return
	}

	accountIDs := make([]uuid.UUID, 0, len(req.AccountIDs))
	for _, idStr := range req.AccountIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			api.BadRequest(w, "invalid account_id: "+idStr)
			return
		}
		accountIDs = append(accountIDs, id)
	}

	batch, err := h.service.CreateBatch(r.Context(), tenantID, userID, req.Name, accountIDs, req.PeriodYear, req.PeriodType, req.PeriodMonth, req.PeriodQuarter)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusCreated, h.toBatchResponse(batch))
}

// ListBatches handles GET /api/v1/uva/batches
func (h *Handler) ListBatches(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	batches, total, err := h.service.ListBatches(r.Context(), tenantID, limit, offset)
	if err != nil {
		api.InternalError(w)
		return
	}

	items := make([]*BatchResponse, 0, len(batches))
	for _, b := range batches {
		items = append(items, h.toBatchResponse(b))
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetBatch handles GET /api/v1/uva/batches/{id}
func (h *Handler) GetBatch(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		api.BadRequest(w, "invalid batch ID")
		return
	}

	batch, err := h.service.GetBatch(r.Context(), id, tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toBatchResponse(batch))
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
	case ErrBatchNotFound:
		api.NotFound(w, "batch not found")
	case ErrDuplicatePeriod:
		api.Conflict(w, "submission for this period already exists")
	case ErrInvalidPeriodType:
		api.BadRequest(w, "invalid period type, must be 'monthly' or 'quarterly'")
	case ErrInvalidMonth:
		api.BadRequest(w, "month must be between 1 and 12")
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
		PeriodMonth:      s.PeriodMonth,
		PeriodQuarter:    s.PeriodQuarter,
		PeriodType:       s.PeriodType,
		ValidationStatus: s.ValidationStatus,
		ValidationErrors: s.ValidationErrors,
		Status:           s.Status,
		FOReference:      s.FOReference,
		CreatedAt:        s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Parse data
	var data UVAData
	if err := json.Unmarshal(s.Data, &data); err == nil {
		resp.Data = data
	}

	if s.SubmittedAt != nil {
		t := s.SubmittedAt.Format("2006-01-02T15:04:05Z")
		resp.SubmittedAt = &t
	}

	return resp
}

func (h *Handler) toBatchResponse(b *Batch) *BatchResponse {
	resp := &BatchResponse{
		ID:            b.ID,
		Name:          b.Name,
		PeriodYear:    b.PeriodYear,
		PeriodMonth:   b.PeriodMonth,
		PeriodQuarter: b.PeriodQuarter,
		PeriodType:    b.PeriodType,
		TotalCount:    b.TotalCount,
		SuccessCount:  b.SuccessCount,
		FailedCount:   b.FailedCount,
		Status:        b.Status,
		CreatedAt:     b.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if b.StartedAt != nil {
		t := b.StartedAt.Format("2006-01-02T15:04:05Z")
		resp.StartedAt = &t
	}

	if b.CompletedAt != nil {
		t := b.CompletedAt.Format("2006-01-02T15:04:05Z")
		resp.CompletedAt = &t
	}

	return resp
}
