package monitor

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/foerderung"
)

// Handler handles monitor HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new monitor handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers monitor routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/monitor", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Get("/{id}/notifications", h.GetNotifications)
		r.Post("/{id}/notifications/{notifId}/view", h.MarkViewed)
		r.Post("/{id}/notifications/{notifId}/dismiss", h.Dismiss)
	})
}

// CreateRequest represents the create monitor request
type CreateRequest struct {
	ProfileID          string `json:"profile_id"`
	MinScoreThreshold  int    `json:"min_score_threshold,omitempty"`
	NotificationEmail  bool   `json:"notification_email"`
	NotificationPortal bool   `json:"notification_portal"`
	DigestMode         string `json:"digest_mode,omitempty"`
}

// UpdateRequest represents the update monitor request
type UpdateRequest struct {
	IsActive          *bool   `json:"is_active,omitempty"`
	MinScoreThreshold *int    `json:"min_score_threshold,omitempty"`
	NotificationEmail *bool   `json:"notification_email,omitempty"`
	NotificationPortal *bool  `json:"notification_portal,omitempty"`
	DigestMode        *string `json:"digest_mode,omitempty"`
}

// MonitorResponse represents a monitor in API responses
type MonitorResponse struct {
	ID                 string  `json:"id"`
	TenantID           string  `json:"tenant_id"`
	ProfileID          string  `json:"profile_id"`
	IsActive           bool    `json:"is_active"`
	MinScoreThreshold  int     `json:"min_score_threshold"`
	NotificationEmail  bool    `json:"notification_email"`
	NotificationPortal bool    `json:"notification_portal"`
	DigestMode         string  `json:"digest_mode"`
	LastCheckAt        *string `json:"last_check_at,omitempty"`
	LastNotificationAt *string `json:"last_notification_at,omitempty"`
	MatchesFound       int     `json:"matches_found"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// ListResponse represents the list monitors response
type ListResponse struct {
	Monitors []*MonitorResponse `json:"monitors"`
	Total    int                `json:"total"`
	Limit    int                `json:"limit"`
	Offset   int                `json:"offset"`
}

// NotificationResponse represents a notification in API responses
type NotificationResponse struct {
	ID           string  `json:"id"`
	MonitorID    string  `json:"monitor_id"`
	FoerderungID string  `json:"foerderung_id"`
	Score        int     `json:"score"`
	MatchSummary *string `json:"match_summary,omitempty"`
	EmailSent    bool    `json:"email_sent"`
	EmailSentAt  *string `json:"email_sent_at,omitempty"`
	PortalNotified bool  `json:"portal_notified"`
	ViewedAt     *string `json:"viewed_at,omitempty"`
	Dismissed    bool    `json:"dismissed"`
	CreatedAt    string  `json:"created_at"`
}

// Create handles POST /api/v1/monitor
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	profileID, err := uuid.Parse(req.ProfileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid profile ID")
		return
	}

	if req.DigestMode != "" {
		if err := ValidateDigestMode(req.DigestMode); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	input := &CreateInput{
		TenantID:          tenantID,
		ProfileID:         profileID,
		MinScoreThreshold: req.MinScoreThreshold,
		NotificationEmail: req.NotificationEmail,
		NotificationPortal: req.NotificationPortal,
		DigestMode:        req.DigestMode,
	}

	monitor, err := h.service.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toMonitorResponse(monitor))
}

// List handles GET /api/v1/monitor
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	monitors, total, err := h.service.ListByTenant(r.Context(), tenantID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list monitors")
		return
	}

	resp := ListResponse{
		Monitors: make([]*MonitorResponse, 0, len(monitors)),
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}
	for _, m := range monitors {
		resp.Monitors = append(resp.Monitors, toMonitorResponse(m))
	}

	writeJSON(w, http.StatusOK, resp)
}

// Get handles GET /api/v1/monitor/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid monitor ID")
		return
	}

	monitor, err := h.service.GetByIDAndTenant(r.Context(), id, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Monitor not found")
		return
	}

	writeJSON(w, http.StatusOK, toMonitorResponse(monitor))
}

// Update handles PUT /api/v1/monitor/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid monitor ID")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.DigestMode != nil {
		if err := ValidateDigestMode(*req.DigestMode); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	input := &UpdateInput{
		IsActive:          req.IsActive,
		MinScoreThreshold: req.MinScoreThreshold,
		NotificationEmail: req.NotificationEmail,
		NotificationPortal: req.NotificationPortal,
		DigestMode:        req.DigestMode,
	}

	monitor, err := h.service.Update(r.Context(), id, tenantID, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toMonitorResponse(monitor))
}

// Delete handles DELETE /api/v1/monitor/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid monitor ID")
		return
	}

	if err := h.service.Delete(r.Context(), id, tenantID); err != nil {
		writeError(w, http.StatusNotFound, "Monitor not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetNotifications handles GET /api/v1/monitor/{id}/notifications
func (h *Handler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid monitor ID")
		return
	}

	// Verify access
	_, err = h.service.GetByIDAndTenant(r.Context(), id, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Monitor not found")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 50
	}

	notifications, err := h.service.GetNotifications(r.Context(), id, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get notifications")
		return
	}

	resp := make([]*NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		resp = append(resp, toNotificationResponse(n))
	}

	writeJSON(w, http.StatusOK, resp)
}

// MarkViewed handles POST /api/v1/monitor/{id}/notifications/{notifId}/view
func (h *Handler) MarkViewed(w http.ResponseWriter, r *http.Request) {
	notifID, err := uuid.Parse(chi.URLParam(r, "notifId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	if err := h.service.MarkNotificationViewed(r.Context(), notifID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to mark as viewed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Dismiss handles POST /api/v1/monitor/{id}/notifications/{notifId}/dismiss
func (h *Handler) Dismiss(w http.ResponseWriter, r *http.Request) {
	notifID, err := uuid.Parse(chi.URLParam(r, "notifId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	if err := h.service.DismissNotification(r.Context(), notifID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to dismiss")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func toMonitorResponse(m *foerderung.ProfilMonitor) *MonitorResponse {
	resp := &MonitorResponse{
		ID:                 m.ID.String(),
		TenantID:           m.TenantID.String(),
		ProfileID:          m.ProfileID.String(),
		IsActive:           m.IsActive,
		MinScoreThreshold:  m.MinScoreThreshold,
		NotificationEmail:  m.NotificationEmail,
		NotificationPortal: m.NotificationPortal,
		DigestMode:         m.DigestMode,
		MatchesFound:       m.MatchesFound,
		CreatedAt:          m.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          m.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if m.LastCheckAt != nil {
		s := m.LastCheckAt.Format("2006-01-02T15:04:05Z")
		resp.LastCheckAt = &s
	}
	if m.LastNotificationAt != nil {
		s := m.LastNotificationAt.Format("2006-01-02T15:04:05Z")
		resp.LastNotificationAt = &s
	}

	return resp
}

func toNotificationResponse(n *foerderung.MonitorNotification) *NotificationResponse {
	resp := &NotificationResponse{
		ID:           n.ID.String(),
		MonitorID:    n.MonitorID.String(),
		FoerderungID: n.FoerderungID.String(),
		Score:        n.Score,
		MatchSummary: n.MatchSummary,
		EmailSent:    n.EmailSent,
		PortalNotified: n.PortalNotified,
		Dismissed:    n.Dismissed,
		CreatedAt:    n.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if n.EmailSentAt != nil {
		s := n.EmailSentAt.Format("2006-01-02T15:04:05Z")
		resp.EmailSentAt = &s
	}
	if n.ViewedAt != nil {
		s := n.ViewedAt.Format("2006-01-02T15:04:05Z")
		resp.ViewedAt = &s
	}

	return resp
}

// Context helper functions

type contextKey string

const (
	tenantIDKey contextKey = "tenant_id"
)

func getTenantIDFromContext(r *http.Request) (uuid.UUID, error) {
	v := r.Context().Value(tenantIDKey)
	if v == nil {
		if h := r.Header.Get("X-Tenant-ID"); h != "" {
			return uuid.Parse(h)
		}
		return uuid.Nil, nil
	}
	if id, ok := v.(uuid.UUID); ok {
		return id, nil
	}
	return uuid.Nil, nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
