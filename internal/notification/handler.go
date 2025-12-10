package notification

import (
	"encoding/json"
	"net/http"

	"austrian-business-infrastructure/internal/api"
	"github.com/google/uuid"
)

// Handler handles notification HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new notification handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers notification routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/notifications/preferences", h.GetPreferences)
	mux.HandleFunc("PUT /api/v1/notifications/preferences", h.UpdatePreferences)
}

// PreferencesResponse represents notification preferences in API responses
type PreferencesResponse struct {
	EmailEnabled  bool     `json:"email_enabled"`
	EmailMode     string   `json:"email_mode"` // immediate, digest, off
	DigestTime    string   `json:"digest_time,omitempty"`
	DocumentTypes []string `json:"document_types,omitempty"`
	AccountIDs    []string `json:"account_ids,omitempty"`
}

// GetPreferences returns notification preferences
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)
	userID := api.GetUserID(ctx)

	if tenantID == "" || userID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid user ID", api.ErrCodeBadRequest)
		return
	}

	prefs, err := h.service.GetPreferences(ctx, userUUID, tenantUUID)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to get preferences", api.ErrCodeInternalError)
		return
	}

	// Convert to response format
	accountIDs := make([]string, len(prefs.AccountIDs))
	for i, id := range prefs.AccountIDs {
		accountIDs[i] = id.String()
	}

	response := &PreferencesResponse{
		EmailEnabled:  prefs.EmailEnabled,
		EmailMode:     prefs.EmailMode,
		DigestTime:    prefs.DigestTime,
		DocumentTypes: prefs.DocumentTypes,
		AccountIDs:    accountIDs,
	}

	api.JSONResponse(w, http.StatusOK, response)
}

// UpdatePreferencesRequest represents the request to update preferences
type UpdatePreferencesRequest struct {
	EmailEnabled  bool     `json:"email_enabled"`
	EmailMode     string   `json:"email_mode"`
	DigestTime    string   `json:"digest_time,omitempty"`
	DocumentTypes []string `json:"document_types,omitempty"`
	AccountIDs    []string `json:"account_ids,omitempty"`
}

// UpdatePreferences updates notification preferences
func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)
	userID := api.GetUserID(ctx)

	if tenantID == "" || userID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid user ID", api.ErrCodeBadRequest)
		return
	}

	var req UpdatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid request body", api.ErrCodeBadRequest)
		return
	}

	// Validate email mode
	if req.EmailMode != ModeImmediate && req.EmailMode != ModeDigest && req.EmailMode != ModeOff {
		api.JSONError(w, http.StatusBadRequest, "invalid email mode", api.ErrCodeValidation)
		return
	}

	// Parse account IDs
	accountIDs := make([]uuid.UUID, 0, len(req.AccountIDs))
	for _, idStr := range req.AccountIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			api.JSONError(w, http.StatusBadRequest, "invalid account ID", api.ErrCodeValidation)
			return
		}
		accountIDs = append(accountIDs, id)
	}

	prefs := &NotificationPreferences{
		UserID:        userUUID,
		TenantID:      tenantUUID,
		EmailEnabled:  req.EmailEnabled,
		EmailMode:     req.EmailMode,
		DigestTime:    req.DigestTime,
		DocumentTypes: req.DocumentTypes,
		AccountIDs:    accountIDs,
	}

	if err := h.service.UpdatePreferences(ctx, prefs); err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to update preferences", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}
