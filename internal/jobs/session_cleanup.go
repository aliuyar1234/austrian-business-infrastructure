package jobs

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/austrian-business-infrastructure/fo/internal/auth"
	"github.com/austrian-business-infrastructure/fo/internal/job"
)

// SessionCleanup is the job type for cleaning up expired sessions
const SessionCleanupJobType = "session_cleanup"

// SessionCleanupHandler handles session cleanup
type SessionCleanupHandler struct {
	sessionManager *auth.SessionManager
	logger         *slog.Logger
}

// SessionCleanupConfig holds configuration for the session cleanup handler
type SessionCleanupConfig struct {
	Logger *slog.Logger
}

// NewSessionCleanupHandler creates a new session cleanup handler
func NewSessionCleanupHandler(
	sessionManager *auth.SessionManager,
	cfg *SessionCleanupConfig,
) *SessionCleanupHandler {
	logger := slog.Default()
	if cfg != nil && cfg.Logger != nil {
		logger = cfg.Logger
	}

	return &SessionCleanupHandler{
		sessionManager: sessionManager,
		logger:         logger,
	}
}

// SessionCleanupResult contains the results of a session cleanup
type SessionCleanupResult struct {
	SessionsDeleted int64 `json:"sessions_deleted"`
}

// Handle executes the session cleanup job
func (h *SessionCleanupHandler) Handle(ctx context.Context, j *job.Job) (json.RawMessage, error) {
	h.logger.Info("starting session cleanup job", "job_id", j.ID)

	deleted, err := h.sessionManager.CleanupExpired(ctx)
	if err != nil {
		h.logger.Error("session cleanup failed", "error", err)
		return nil, err
	}

	h.logger.Info("session cleanup completed", "sessions_deleted", deleted)

	return json.Marshal(SessionCleanupResult{
		SessionsDeleted: deleted,
	})
}

// Register registers the session cleanup handler with a job registry
func (h *SessionCleanupHandler) Register(registry *job.Registry) {
	registry.MustRegister(SessionCleanupJobType, h)
}
