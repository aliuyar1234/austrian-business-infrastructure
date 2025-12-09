package session

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/auth"
	"github.com/google/uuid"
)

// Handler handles session HTTP requests
type Handler struct {
	sessionMgr *auth.SessionManager
	logger     *slog.Logger
}

// NewHandler creates a new session handler
func NewHandler(sessionMgr *auth.SessionManager, logger *slog.Logger) *Handler {
	return &Handler{
		sessionMgr: sessionMgr,
		logger:     logger,
	}
}

// RegisterRoutes registers session routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth func(http.Handler) http.Handler) {
	router.Handle("GET /api/v1/sessions", requireAuth(http.HandlerFunc(h.List)))
	router.Handle("DELETE /api/v1/sessions/{id}", requireAuth(http.HandlerFunc(h.Terminate)))
	router.Handle("DELETE /api/v1/sessions", requireAuth(http.HandlerFunc(h.TerminateAll)))
}

// SessionDTO is a data transfer object for sessions
type SessionDTO struct {
	ID         string  `json:"id"`
	UserAgent  *string `json:"user_agent,omitempty"`
	IPAddress  *string `json:"ip_address,omitempty"`
	ExpiresAt  string  `json:"expires_at"`
	CreatedAt  string  `json:"created_at"`
	LastUsedAt string  `json:"last_used_at"`
	IsCurrent  bool    `json:"is_current"`
}

// List handles GET /api/v1/sessions
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(api.GetUserID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	sessions, err := h.sessionMgr.ListUserSessions(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list sessions", "error", err)
		api.InternalError(w)
		return
	}

	// Get current session ID from token if possible
	// This is a simplified approach - in production you'd track session ID in JWT
	currentSessionID := "" // Would need to be passed or looked up

	dtos := make([]*SessionDTO, len(sessions))
	for i, s := range sessions {
		dtos[i] = &SessionDTO{
			ID:         s.ID.String(),
			UserAgent:  s.UserAgent,
			IPAddress:  s.IPAddress,
			ExpiresAt:  s.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
			CreatedAt:  s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			LastUsedAt: s.LastUsedAt.Format("2006-01-02T15:04:05Z07:00"),
			IsCurrent:  s.ID.String() == currentSessionID,
		}
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"sessions": dtos,
	})
}

// Terminate handles DELETE /api/v1/sessions/{id}
func (h *Handler) Terminate(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		api.BadRequest(w, "Invalid session ID")
		return
	}

	userID, err := uuid.Parse(api.GetUserID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	// Verify session belongs to user
	sessions, err := h.sessionMgr.ListUserSessions(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list sessions", "error", err)
		api.InternalError(w)
		return
	}

	found := false
	for _, s := range sessions {
		if s.ID == sessionID {
			found = true
			break
		}
	}

	if !found {
		api.NotFound(w, "Session not found")
		return
	}

	if err := h.sessionMgr.DeleteSession(r.Context(), sessionID); err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			api.NotFound(w, "Session not found")
			return
		}
		h.logger.Error("failed to terminate session", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "Session terminated",
	})
}

// TerminateAll handles DELETE /api/v1/sessions
func (h *Handler) TerminateAll(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(api.GetUserID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	// Optional: exclude current session
	excludeCurrent := r.URL.Query().Get("exclude_current") == "true"

	if excludeCurrent {
		// Would need current session ID - for now just terminate all
		h.logger.Warn("exclude_current not implemented, terminating all sessions")
	}

	if err := h.sessionMgr.DeleteAllUserSessions(r.Context(), userID); err != nil {
		h.logger.Error("failed to terminate all sessions", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "All sessions terminated",
	})
}
