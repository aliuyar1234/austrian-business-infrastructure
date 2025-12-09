package job

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/google/uuid"
)

// APIHandler handles job HTTP requests
type APIHandler struct {
	queue      *Queue
	repository *Repository
}

// NewAPIHandler creates a new job API handler
func NewAPIHandler(queue *Queue, repository *Repository) *APIHandler {
	return &APIHandler{
		queue:      queue,
		repository: repository,
	}
}

// RegisterRoutes registers job routes
func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/jobs", h.ListJobs)
	mux.HandleFunc("GET /api/v1/jobs/{id}", h.GetJob)
	mux.HandleFunc("POST /api/v1/jobs/{id}/retry", h.RetryJob)
	mux.HandleFunc("GET /api/v1/jobs/dead-letters", h.ListDeadLetters)
	mux.HandleFunc("POST /api/v1/jobs/dead-letters/{id}/acknowledge", h.AcknowledgeDeadLetter)
	mux.HandleFunc("GET /api/v1/jobs/metrics", h.GetMetrics)
}

// JobResponse represents a job in API responses
type JobResponse struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Status       string  `json:"status"`
	Payload      interface{} `json:"payload,omitempty"`
	Result       interface{} `json:"result,omitempty"`
	ErrorMessage string  `json:"error_message,omitempty"`
	StartedAt    string  `json:"started_at,omitempty"`
	CompletedAt  string  `json:"completed_at,omitempty"`
	DurationMs   int     `json:"duration_ms,omitempty"`
	WorkerID     string  `json:"worker_id,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

// ListJobs lists job history with filters
func (h *APIHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
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

	// Parse query parameters
	filter := JobHistoryFilter{
		TenantID: tenantUUID,
		Limit:    50,
		Offset:   0,
	}

	if typeFilter := r.URL.Query().Get("type"); typeFilter != "" {
		filter.Type = typeFilter
	}
	if statusFilter := r.URL.Query().Get("status"); statusFilter != "" {
		filter.Status = statusFilter
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
	if dateFromStr := r.URL.Query().Get("date_from"); dateFromStr != "" {
		if t, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			filter.DateFrom = &t
		}
	}
	if dateToStr := r.URL.Query().Get("date_to"); dateToStr != "" {
		if t, err := time.Parse("2006-01-02", dateToStr); err == nil {
			endOfDay := t.Add(24 * time.Hour)
			filter.DateTo = &endOfDay
		}
	}

	history, total, err := h.repository.ListHistory(ctx, filter)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to list jobs", api.ErrCodeInternalError)
		return
	}

	// Convert to response format
	jobs := make([]*JobResponse, len(history))
	for i, jh := range history {
		jobs[i] = jobHistoryToResponse(jh)
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"jobs":   jobs,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetJob retrieves a single job history entry
func (h *APIHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	jobID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid job ID", api.ErrCodeBadRequest)
		return
	}

	jh, err := h.repository.GetHistoryByID(ctx, jobID)
	if err != nil {
		if err == ErrJobNotFound {
			api.JSONError(w, http.StatusNotFound, "job not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get job", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, jobHistoryToResponse(jh))
}

// RetryJob re-enqueues a failed job
func (h *APIHandler) RetryJob(w http.ResponseWriter, r *http.Request) {
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

	jobID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid job ID", api.ErrCodeBadRequest)
		return
	}

	// Get the original job from history
	jh, err := h.repository.GetHistoryByID(ctx, jobID)
	if err != nil {
		if err == ErrJobNotFound {
			api.JSONError(w, http.StatusNotFound, "job not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get job", api.ErrCodeInternalError)
		return
	}

	// Re-enqueue the job
	newJob, err := h.queue.Enqueue(ctx, tenantUUID, jh.Type, jh.Payload, &EnqueueOptions{
		Priority:   PriorityNormal,
		RunAt:      time.Now(),
		MaxRetries: 3,
	})
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to retry job", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusAccepted, map[string]string{
		"message": "job re-enqueued",
		"job_id":  newJob.ID.String(),
	})
}

// DeadLetterResponse represents a dead letter in API responses
type DeadLetterResponse struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`
	Payload         interface{} `json:"payload"`
	Errors          []string `json:"errors"`
	TotalAttempts   int      `json:"total_attempts"`
	FirstAttemptedAt string  `json:"first_attempted_at"`
	LastAttemptedAt  string  `json:"last_attempted_at"`
	Acknowledged    bool     `json:"acknowledged"`
	CreatedAt       string   `json:"created_at"`
}

// ListDeadLetters lists dead letter queue entries
func (h *APIHandler) ListDeadLetters(w http.ResponseWriter, r *http.Request) {
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

	// Parse query parameters
	acknowledged := false
	if ackStr := r.URL.Query().Get("acknowledged"); ackStr == "true" {
		acknowledged = true
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

	deadLetters, total, err := h.repository.ListDeadLetters(ctx, tenantUUID, acknowledged, limit, offset)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to list dead letters", api.ErrCodeInternalError)
		return
	}

	// Convert to response format
	items := make([]*DeadLetterResponse, len(deadLetters))
	for i, dl := range deadLetters {
		items[i] = deadLetterToResponse(dl)
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"dead_letters": items,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

// AcknowledgeDeadLetter marks a dead letter as acknowledged
func (h *APIHandler) AcknowledgeDeadLetter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)
	userID := api.GetUserID(ctx)

	if tenantID == "" || userID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid user ID", api.ErrCodeBadRequest)
		return
	}

	dlID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid dead letter ID", api.ErrCodeBadRequest)
		return
	}

	if err := h.repository.AcknowledgeDeadLetter(ctx, dlID, userUUID); err != nil {
		if err == ErrJobNotFound {
			api.JSONError(w, http.StatusNotFound, "dead letter not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to acknowledge", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

// GetMetrics returns job metrics
func (h *APIHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
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

	metrics, err := h.repository.GetJobMetrics(ctx, tenantUUID)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to get metrics", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, metrics)
}

// Helper functions

func jobHistoryToResponse(jh *JobHistory) *JobResponse {
	resp := &JobResponse{
		ID:         jh.ID.String(),
		Type:       jh.Type,
		Status:     jh.Status,
		DurationMs: jh.DurationMs,
		WorkerID:   jh.WorkerID,
		CreatedAt:  jh.CreatedAt.Format(time.RFC3339),
	}

	if len(jh.Payload) > 0 {
		var payload interface{}
		if err := json.Unmarshal(jh.Payload, &payload); err == nil {
			resp.Payload = payload
		}
	}

	if len(jh.Result) > 0 {
		var result interface{}
		if err := json.Unmarshal(jh.Result, &result); err == nil {
			resp.Result = result
		}
	}

	if jh.ErrorMessage != "" {
		resp.ErrorMessage = jh.ErrorMessage
	}

	resp.StartedAt = jh.StartedAt.Format(time.RFC3339)
	resp.CompletedAt = jh.CompletedAt.Format(time.RFC3339)

	return resp
}

func deadLetterToResponse(dl *DeadLetter) *DeadLetterResponse {
	resp := &DeadLetterResponse{
		ID:              dl.ID.String(),
		Type:            dl.Type,
		Errors:          dl.Errors,
		TotalAttempts:   dl.TotalAttempts,
		FirstAttemptedAt: dl.FirstAttemptedAt.Format(time.RFC3339),
		LastAttemptedAt:  dl.LastAttemptedAt.Format(time.RFC3339),
		Acknowledged:    dl.Acknowledged,
		CreatedAt:       dl.CreatedAt.Format(time.RFC3339),
	}

	if len(dl.Payload) > 0 {
		var payload interface{}
		if err := json.Unmarshal(dl.Payload, &payload); err == nil {
			resp.Payload = payload
		}
	}

	return resp
}
