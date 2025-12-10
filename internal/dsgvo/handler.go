package dsgvo

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"austrian-business-infrastructure/internal/api"
	"github.com/google/uuid"
)

// Handler handles DSGVO-related HTTP requests
type Handler struct {
	exporter        *Exporter
	deletionManager *DeletionManager
	repo            Repository
	auditLogger     AuditLogger
	logger          *slog.Logger
}

// Repository interface for DSGVO data persistence
type Repository interface {
	// Export operations
	CreateExportRequest(ctx interface{}, req *ExportRequest) error
	GetExportRequest(ctx interface{}, tenantID, exportID uuid.UUID) (*ExportRequest, error)
	ListExportRequests(ctx interface{}, tenantID uuid.UUID) ([]*ExportRequest, error)
	UpdateExportRequest(ctx interface{}, req *ExportRequest) error

	// Deletion operations
	CreateDeletionRequest(ctx interface{}, req *DeletionRequest) error
	GetDeletionRequest(ctx interface{}, tenantID uuid.UUID) (*DeletionRequest, error)
	GetDeletionRequestByID(ctx interface{}, tenantID, deletionID uuid.UUID) (*DeletionRequest, error)
	UpdateDeletionRequest(ctx interface{}, req *DeletionRequest) error
	GetPendingDeletionRequests(ctx interface{}) ([]*DeletionRequest, error)
}

// AuditLogger interface for audit logging
type AuditLogger interface {
	Log(ctx interface{}, logCtx interface{}, action string, details map[string]interface{}) error
}

// NewHandler creates a new DSGVO handler
func NewHandler(exporter *Exporter, deletionManager *DeletionManager, repo Repository, auditLogger AuditLogger, logger *slog.Logger) *Handler {
	return &Handler{
		exporter:        exporter,
		deletionManager: deletionManager,
		repo:            repo,
		auditLogger:     auditLogger,
		logger:          logger,
	}
}

// RegisterRoutes registers DSGVO routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	// Export endpoints
	router.Handle("POST /api/v1/dsgvo/export", requireAuth(requireAdmin(http.HandlerFunc(h.CreateExport))))
	router.Handle("GET /api/v1/dsgvo/export", requireAuth(requireAdmin(http.HandlerFunc(h.ListExports))))
	router.Handle("GET /api/v1/dsgvo/export/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.GetExport))))
	router.Handle("GET /api/v1/dsgvo/export/{id}/download", requireAuth(requireAdmin(http.HandlerFunc(h.DownloadExport))))

	// Deletion endpoints
	router.Handle("POST /api/v1/dsgvo/deletion", requireAuth(requireAdmin(http.HandlerFunc(h.CreateDeletion))))
	router.Handle("GET /api/v1/dsgvo/deletion", requireAuth(requireAdmin(http.HandlerFunc(h.GetDeletion))))
	router.Handle("DELETE /api/v1/dsgvo/deletion", requireAuth(requireAdmin(http.HandlerFunc(h.CancelDeletion))))

	// PII Registry endpoint
	router.Handle("GET /api/v1/dsgvo/pii-registry", requireAuth(requireAdmin(http.HandlerFunc(h.GetPIIRegistry))))
}

// ====================
// Export Handlers
// ====================

// CreateExportRequest is the request body for creating an export
type CreateExportRequest struct {
	// Optional: can include filters in future
}

// ExportResponse is the response for export operations
type ExportResponse struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	FileSize    *int64     `json:"file_size,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       *string    `json:"error,omitempty"`
}

// CreateExport handles POST /api/v1/dsgvo/export
func (h *Handler) CreateExport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	userID, err := uuid.Parse(api.GetUserID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	// Create export request
	exportReq := &ExportRequest{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RequestedBy: userID,
		Status:      ExportStatusPending,
		CreatedAt:   time.Now(),
	}

	// Save to database
	if err := h.repo.CreateExportRequest(r.Context(), exportReq); err != nil {
		h.logger.Error("failed to create export request", "error", err)
		api.InternalError(w)
		return
	}

	// Start export in background
	go func() {
		if err := h.exporter.CreateExport(r.Context(), exportReq); err != nil {
			h.logger.Error("export failed", "export_id", exportReq.ID, "error", err)
		}
		// Update in database
		_ = h.repo.UpdateExportRequest(r.Context(), exportReq)
	}()

	// Log audit event
	if h.auditLogger != nil {
		h.auditLogger.Log(r.Context(), nil, "data.export_requested", map[string]interface{}{
			"export_id": exportReq.ID.String(),
		})
	}

	api.JSONResponse(w, http.StatusAccepted, toExportResponse(exportReq))
}

// ListExports handles GET /api/v1/dsgvo/export
func (h *Handler) ListExports(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	exports, err := h.repo.ListExportRequests(r.Context(), tenantID)
	if err != nil {
		h.logger.Error("failed to list exports", "error", err)
		api.InternalError(w)
		return
	}

	responses := make([]*ExportResponse, len(exports))
	for i, exp := range exports {
		responses[i] = toExportResponse(exp)
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"exports": responses,
	})
}

// GetExport handles GET /api/v1/dsgvo/export/{id}
func (h *Handler) GetExport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	exportID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "Invalid export ID")
		return
	}

	export, err := h.repo.GetExportRequest(r.Context(), tenantID, exportID)
	if err != nil {
		if err == ErrExportNotFound {
			api.NotFound(w, "Export not found")
			return
		}
		h.logger.Error("failed to get export", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, toExportResponse(export))
}

// DownloadExport handles GET /api/v1/dsgvo/export/{id}/download
func (h *Handler) DownloadExport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	exportID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "Invalid export ID")
		return
	}

	export, err := h.repo.GetExportRequest(r.Context(), tenantID, exportID)
	if err != nil {
		if err == ErrExportNotFound {
			api.NotFound(w, "Export not found")
			return
		}
		h.logger.Error("failed to get export", "error", err)
		api.InternalError(w)
		return
	}

	// Get file reader
	reader, err := h.exporter.GetExportFile(export)
	if err != nil {
		if err == ErrExportNotReady {
			api.JSONError(w, http.StatusConflict, "Export is still being processed", "EXPORT_NOT_READY")
			return
		}
		if err == ErrExportExpired {
			api.JSONError(w, http.StatusGone, "Export has expired", "EXPORT_EXPIRED")
			return
		}
		h.logger.Error("failed to get export file", "error", err)
		api.InternalError(w)
		return
	}
	defer reader.Close()

	// Set headers for download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=dsgvo-export-"+export.ID.String()+".zip")
	if export.FileSize != nil {
		w.Header().Set("Content-Length", strconv.FormatInt(*export.FileSize, 10))
	}

	// Stream file - os.File implements io.ReadSeeker
	if seeker, ok := reader.(io.ReadSeeker); ok {
		http.ServeContent(w, r, "export.zip", time.Now(), seeker)
	} else {
		// Fallback to simple copy if not seekable
		_, _ = io.Copy(w, reader)
	}
}

// ====================
// Deletion Handlers
// ====================

// CreateDeletionRequest is the request body for creating a deletion
type CreateDeletionRequestBody struct {
	Reason string `json:"reason,omitempty"`
}

// DeletionResponse is the response for deletion operations
type DeletionResponse struct {
	ID              string     `json:"id"`
	Status          string     `json:"status"`
	Reason          *string    `json:"reason,omitempty"`
	ScheduledFor    time.Time  `json:"scheduled_for"`
	GracePeriodDays int        `json:"grace_period_days"`
	RemainingDays   int        `json:"remaining_days"`
	CreatedAt       time.Time  `json:"created_at"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
	ExecutedAt      *time.Time `json:"executed_at,omitempty"`
	Error           *string    `json:"error,omitempty"`
}

// CreateDeletion handles POST /api/v1/dsgvo/deletion
func (h *Handler) CreateDeletion(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	userID, err := uuid.Parse(api.GetUserID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	// Check if deletion already exists
	existing, err := h.repo.GetDeletionRequest(r.Context(), tenantID)
	if err == nil && existing != nil && existing.Status == DeletionStatusPending {
		api.JSONError(w, http.StatusConflict, "A deletion request is already pending", "DELETION_PENDING")
		return
	}

	// Parse request body
	var req CreateDeletionRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Body is optional
	}

	var reason *string
	if req.Reason != "" {
		reason = &req.Reason
	}

	// Create deletion request
	deletionReq := h.deletionManager.CreateDeletionRequest(tenantID, userID, reason)

	// Save to database
	if err := h.repo.CreateDeletionRequest(r.Context(), deletionReq); err != nil {
		h.logger.Error("failed to create deletion request", "error", err)
		api.InternalError(w)
		return
	}

	// Log audit event
	if h.auditLogger != nil {
		h.auditLogger.Log(r.Context(), nil, "data.deletion_requested", map[string]interface{}{
			"deletion_id":  deletionReq.ID.String(),
			"scheduled_for": deletionReq.ScheduledFor.Format(time.RFC3339),
		})
	}

	api.JSONResponse(w, http.StatusAccepted, toDeletionResponse(deletionReq, h.deletionManager))
}

// GetDeletion handles GET /api/v1/dsgvo/deletion
func (h *Handler) GetDeletion(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	deletion, err := h.repo.GetDeletionRequest(r.Context(), tenantID)
	if err != nil {
		if err == ErrDeletionNotFound {
			api.NotFound(w, "No deletion request found")
			return
		}
		h.logger.Error("failed to get deletion request", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, toDeletionResponse(deletion, h.deletionManager))
}

// CancelDeletion handles DELETE /api/v1/dsgvo/deletion
func (h *Handler) CancelDeletion(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	userID, err := uuid.Parse(api.GetUserID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	deletion, err := h.repo.GetDeletionRequest(r.Context(), tenantID)
	if err != nil {
		if err == ErrDeletionNotFound {
			api.NotFound(w, "No deletion request found")
			return
		}
		h.logger.Error("failed to get deletion request", "error", err)
		api.InternalError(w)
		return
	}

	// Cancel the deletion
	if err := h.deletionManager.CancelDeletionRequest(deletion, userID); err != nil {
		if err == ErrDeletionAlreadyExecuted {
			api.JSONError(w, http.StatusConflict, "Deletion has already been executed", "DELETION_EXECUTED")
			return
		}
		if err == ErrDeletionCancelled {
			api.JSONError(w, http.StatusConflict, "Deletion was already cancelled", "DELETION_CANCELLED")
			return
		}
		api.InternalError(w)
		return
	}

	// Update in database
	if err := h.repo.UpdateDeletionRequest(r.Context(), deletion); err != nil {
		h.logger.Error("failed to update deletion request", "error", err)
		api.InternalError(w)
		return
	}

	// Log audit event
	if h.auditLogger != nil {
		h.auditLogger.Log(r.Context(), nil, "data.deletion_cancelled", map[string]interface{}{
			"deletion_id": deletion.ID.String(),
		})
	}

	api.JSONResponse(w, http.StatusOK, toDeletionResponse(deletion, h.deletionManager))
}

// ====================
// PII Registry Handler
// ====================

// GetPIIRegistry handles GET /api/v1/dsgvo/pii-registry
func (h *Handler) GetPIIRegistry(w http.ResponseWriter, r *http.Request) {
	registry := GetPIIRegistry()
	api.JSONResponse(w, http.StatusOK, registry)
}

// ====================
// Helper Functions
// ====================

func toExportResponse(exp *ExportRequest) *ExportResponse {
	return &ExportResponse{
		ID:          exp.ID.String(),
		Status:      string(exp.Status),
		FileSize:    exp.FileSize,
		ExpiresAt:   exp.ExpiresAt,
		CreatedAt:   exp.CreatedAt,
		CompletedAt: exp.CompletedAt,
		Error:       exp.Error,
	}
}

func toDeletionResponse(del *DeletionRequest, manager *DeletionManager) *DeletionResponse {
	remaining := manager.GetRemainingGracePeriod(del)
	remainingDays := int(remaining.Hours() / 24)
	if remainingDays < 0 {
		remainingDays = 0
	}

	return &DeletionResponse{
		ID:              del.ID.String(),
		Status:          string(del.Status),
		Reason:          del.Reason,
		ScheduledFor:    del.ScheduledFor,
		GracePeriodDays: del.GracePeriodDays,
		RemainingDays:   remainingDays,
		CreatedAt:       del.CreatedAt,
		CancelledAt:     del.CancelledAt,
		ExecutedAt:      del.ExecutedAt,
		Error:           del.Error,
	}
}
