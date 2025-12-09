package sync

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/google/uuid"
)

// Handler handles sync HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new sync handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers sync routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/sync", h.SyncAll)
	mux.HandleFunc("POST /api/v1/accounts/{id}/sync", h.SyncAccount)
	mux.HandleFunc("GET /api/v1/sync/{id}", h.GetJob)
	mux.HandleFunc("GET /api/v1/sync", h.ListJobs)
}

// SyncJobResponse represents a sync job in API responses
type SyncJobResponse struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	AccountID        *uuid.UUID `json:"account_id,omitempty"`
	AccountName      string     `json:"account_name,omitempty"`
	Status           string     `json:"status"`
	JobType          string     `json:"job_type"`
	DocumentsFound   int        `json:"documents_found"`
	DocumentsNew     int        `json:"documents_new"`
	DocumentsSkipped int        `json:"documents_skipped"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	StartedAt        *string    `json:"started_at,omitempty"`
	CompletedAt      *string    `json:"completed_at,omitempty"`
	CreatedAt        string     `json:"created_at"`
}

// toResponse converts a SyncJob to SyncJobResponse
func toResponse(job *SyncJob) *SyncJobResponse {
	resp := &SyncJobResponse{
		ID:               job.ID,
		TenantID:         job.TenantID,
		AccountID:        job.AccountID,
		AccountName:      job.AccountName,
		Status:           job.Status,
		JobType:          job.JobType,
		DocumentsFound:   job.DocumentsFound,
		DocumentsNew:     job.DocumentsNew,
		DocumentsSkipped: job.DocumentsSkipped,
		ErrorMessage:     job.ErrorMessage,
		CreatedAt:        job.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if job.StartedAt != nil {
		s := job.StartedAt.Format("2006-01-02T15:04:05Z")
		resp.StartedAt = &s
	}

	if job.CompletedAt != nil {
		s := job.CompletedAt.Format("2006-01-02T15:04:05Z")
		resp.CompletedAt = &s
	}

	return resp
}

// SyncAllRequest represents the request to sync all accounts
type SyncAllRequest struct {
	AccountIDs []uuid.UUID `json:"account_ids,omitempty"`
}

// SyncAll triggers sync for all accounts (or specified accounts)
func (h *Handler) SyncAll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	job, err := h.service.SyncAllAccounts(ctx, tenantUUID)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to start sync", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusAccepted, toResponse(job))
}

// SyncAccountRequest represents the request to sync a single account
type SyncAccountRequest struct {
	Credentials map[string]string `json:"credentials,omitempty"`
}

// SyncAccount triggers sync for a single account
func (h *Handler) SyncAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	accountIDStr := r.PathValue("id")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid account ID", api.ErrCodeBadRequest)
		return
	}

	// Parse credentials from request body (optional)
	var req SyncAccountRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.JSONError(w, http.StatusBadRequest, "invalid request body", api.ErrCodeBadRequest)
			return
		}
	}

	// Start sync
	job, err := h.service.SyncSingleAccount(ctx, tenantUUID, accountID, req.Credentials)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, err.Error(), api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusAccepted, toResponse(job))
}

// GetJob returns a sync job by ID
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid job ID", api.ErrCodeBadRequest)
		return
	}

	job, err := h.service.GetJob(ctx, id)
	if err != nil {
		if err == ErrSyncJobNotFound {
			api.JSONError(w, http.StatusNotFound, "sync job not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get sync job", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, toResponse(job))
}

// ListJobsResponse represents the response for listing sync jobs
type ListJobsResponse struct {
	Jobs    []*SyncJobResponse `json:"jobs"`
	Total   int                `json:"total"`
	Limit   int                `json:"limit"`
	Offset  int                `json:"offset"`
	HasMore bool               `json:"has_more"`
}

// ListJobs returns a paginated list of sync jobs
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	filter := &SyncJobFilter{
		TenantID: tenantUUID,
		Limit:    50,
		Offset:   0,
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	if accountID := r.URL.Query().Get("account_id"); accountID != "" {
		if id, err := uuid.Parse(accountID); err == nil {
			filter.AccountID = &id
		}
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = status
	}

	jobs, total, err := h.service.ListJobs(ctx, filter)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to list sync jobs", api.ErrCodeInternalError)
		return
	}

	responses := make([]*SyncJobResponse, len(jobs))
	for i, job := range jobs {
		responses[i] = toResponse(job)
	}

	response := &ListJobsResponse{
		Jobs:    responses,
		Total:   total,
		Limit:   filter.Limit,
		Offset:  filter.Offset,
		HasMore: filter.Offset+len(jobs) < total,
	}

	api.JSONResponse(w, http.StatusOK, response)
}
