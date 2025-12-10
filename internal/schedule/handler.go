package schedule

import (
	"encoding/json"
	"net/http"

	"austrian-business-infrastructure/internal/account"
	"austrian-business-infrastructure/internal/api"
	"github.com/google/uuid"
)

// Handler handles schedule HTTP requests
type Handler struct {
	accountRepo *account.Repository
}

// NewHandler creates a new schedule handler
func NewHandler(accountRepo *account.Repository) *Handler {
	return &Handler{accountRepo: accountRepo}
}

// RegisterRoutes registers schedule routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/accounts/{id}/schedule", h.GetSchedule)
	mux.HandleFunc("PUT /api/v1/accounts/{id}/schedule", h.UpdateSchedule)
}

// ScheduleResponse represents the schedule in API responses
type ScheduleResponse struct {
	AccountID       string `json:"account_id"`
	AutoSyncEnabled bool   `json:"auto_sync_enabled"`
	SyncInterval    string `json:"sync_interval"`
	LastSyncAt      string `json:"last_sync_at,omitempty"`
	NextSyncAt      string `json:"next_sync_at,omitempty"`
}

// GetSchedule returns the sync schedule for an account
func (h *Handler) GetSchedule(w http.ResponseWriter, r *http.Request) {
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

	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid account ID", api.ErrCodeBadRequest)
		return
	}

	acc, err := h.accountRepo.GetByID(ctx, accountID, tenantUUID)
	if err != nil {
		if err == account.ErrAccountNotFound {
			api.JSONError(w, http.StatusNotFound, "account not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get account", api.ErrCodeInternalError)
		return
	}

	response := &ScheduleResponse{
		AccountID:       acc.ID.String(),
		AutoSyncEnabled: acc.AutoSyncEnabled,
		SyncInterval:    acc.SyncInterval,
	}

	if acc.LastSyncAt != nil {
		response.LastSyncAt = acc.LastSyncAt.Format("2006-01-02T15:04:05Z")
	}
	if acc.NextSyncAt != nil {
		response.NextSyncAt = acc.NextSyncAt.Format("2006-01-02T15:04:05Z")
	}

	api.JSONResponse(w, http.StatusOK, response)
}

// UpdateScheduleRequest represents the request to update schedule
type UpdateScheduleRequest struct {
	AutoSyncEnabled *bool   `json:"auto_sync_enabled,omitempty"`
	SyncInterval    *string `json:"sync_interval,omitempty"`
}

// UpdateSchedule updates the sync schedule for an account
func (h *Handler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
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

	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid account ID", api.ErrCodeBadRequest)
		return
	}

	var req UpdateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid request body", api.ErrCodeBadRequest)
		return
	}

	// Validate sync interval
	validIntervals := map[string]bool{
		"hourly":   true,
		"4hourly":  true,
		"daily":    true,
		"weekly":   true,
		"disabled": true,
	}
	if req.SyncInterval != nil && !validIntervals[*req.SyncInterval] {
		api.JSONError(w, http.StatusBadRequest, "invalid sync interval (must be: hourly, 4hourly, daily, weekly, disabled)", api.ErrCodeValidation)
		return
	}

	// Get existing account
	acc, err := h.accountRepo.GetByID(ctx, accountID, tenantUUID)
	if err != nil {
		if err == account.ErrAccountNotFound {
			api.JSONError(w, http.StatusNotFound, "account not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get account", api.ErrCodeInternalError)
		return
	}

	// Update schedule settings
	if err := h.accountRepo.UpdateSyncSettings(ctx, accountID, req.AutoSyncEnabled, req.SyncInterval); err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to update schedule", api.ErrCodeInternalError)
		return
	}

	// Return updated schedule
	if req.AutoSyncEnabled != nil {
		acc.AutoSyncEnabled = *req.AutoSyncEnabled
	}
	if req.SyncInterval != nil {
		acc.SyncInterval = *req.SyncInterval
	}

	response := &ScheduleResponse{
		AccountID:       acc.ID.String(),
		AutoSyncEnabled: acc.AutoSyncEnabled,
		SyncInterval:    acc.SyncInterval,
	}

	if acc.LastSyncAt != nil {
		response.LastSyncAt = acc.LastSyncAt.Format("2006-01-02T15:04:05Z")
	}
	if acc.NextSyncAt != nil {
		response.NextSyncAt = acc.NextSyncAt.Format("2006-01-02T15:04:05Z")
	}

	api.JSONResponse(w, http.StatusOK, response)
}
