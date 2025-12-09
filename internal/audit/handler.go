package audit

import (
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/google/uuid"
)

// Handler handles audit log HTTP requests
type Handler struct {
	repo   *Repository
	logger *slog.Logger
}

// NewHandler creates a new audit handler
func NewHandler(repo *Repository, logger *slog.Logger) *Handler {
	return &Handler{
		repo:   repo,
		logger: logger,
	}
}

// RegisterRoutes registers audit routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	router.Handle("GET /api/v1/audit-logs", requireAuth(requireAdmin(http.HandlerFunc(h.List))))
	router.Handle("GET /api/v1/audit-logs/statistics", requireAuth(requireAdmin(http.HandlerFunc(h.Statistics))))
	router.Handle("GET /api/v1/audit-logs/export", requireAuth(requireAdmin(http.HandlerFunc(h.Export))))
	router.Handle("GET /api/v1/audit-logs/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.GetByID))))
}

// AuditLogDTO is a data transfer object for audit logs
type AuditLogDTO struct {
	ID           string                 `json:"id"`
	UserID       *string                `json:"user_id,omitempty"`
	Action       string                 `json:"action"`
	ResourceType *string                `json:"resource_type,omitempty"`
	ResourceID   *string                `json:"resource_id,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	IPAddress    *string                `json:"ip_address,omitempty"`
	UserAgent    *string                `json:"user_agent,omitempty"`
	CreatedAt    string                 `json:"created_at"`
}

// ListResponse represents a list audit logs response
type ListResponse struct {
	Logs       []*AuditLogDTO `json:"logs"`
	Total      int64          `json:"total"`
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
	HasMore    bool           `json:"has_more"`
}

// StatisticsResponse represents audit log statistics
type StatisticsResponse struct {
	Total         int64            `json:"total"`
	ByAction      map[string]int64 `json:"by_action"`
	ByResourceType map[string]int64 `json:"by_resource_type"`
	Last24h       int64            `json:"last_24h"`
	Last7d        int64            `json:"last_7d"`
	Last30d       int64            `json:"last_30d"`
}

// GetByID handles GET /api/v1/audit-logs/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	idStr := r.PathValue("id")
	logID, err := uuid.Parse(idStr)
	if err != nil {
		api.BadRequest(w, "Invalid log ID format")
		return
	}

	log, err := h.repo.GetByID(r.Context(), tenantID, logID)
	if err != nil {
		if err.Error() == "audit log not found" {
			api.NotFound(w, "Audit log not found")
			return
		}
		h.logger.Error("failed to get audit log", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, toAuditLogDTO(log))
}

// Statistics handles GET /api/v1/audit-logs/statistics
func (h *Handler) Statistics(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	stats, err := h.repo.GetStatistics(r.Context(), tenantID)
	if err != nil {
		h.logger.Error("failed to get audit statistics", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, stats)
}

// List handles GET /api/v1/audit-logs
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	filter := h.parseFilter(r, &tenantID)

	logs, err := h.repo.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("failed to list audit logs", "error", err)
		api.InternalError(w)
		return
	}

	total, err := h.repo.Count(r.Context(), filter)
	if err != nil {
		h.logger.Error("failed to count audit logs", "error", err)
		api.InternalError(w)
		return
	}

	dtos := make([]*AuditLogDTO, len(logs))
	for i, log := range logs {
		dtos[i] = toAuditLogDTO(log)
	}

	api.JSONResponse(w, http.StatusOK, ListResponse{
		Logs:    dtos,
		Total:   total,
		Limit:   filter.Limit,
		Offset:  filter.Offset,
		HasMore: int64(filter.Offset+len(logs)) < total,
	})
}

// Export handles GET /api/v1/audit-logs/export
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	filter := h.parseFilter(r, &tenantID)
	filter.Limit = 10000 // Max export limit
	filter.Offset = 0

	logs, err := h.repo.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("failed to export audit logs", "error", err)
		api.InternalError(w)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	switch format {
	case "json":
		h.exportJSON(w, logs)
	case "csv":
		h.exportCSV(w, logs)
	default:
		api.BadRequest(w, "Invalid format. Use 'json' or 'csv'")
	}
}

func (h *Handler) exportJSON(w http.ResponseWriter, logs []*AuditLog) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=audit-logs.json")

	dtos := make([]*AuditLogDTO, len(logs))
	for i, log := range logs {
		dtos[i] = toAuditLogDTO(log)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":       dtos,
		"exported_at": time.Now().Format(time.RFC3339),
	})
}

func (h *Handler) exportCSV(w http.ResponseWriter, logs []*AuditLog) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=audit-logs.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"ID", "User ID", "Action", "Resource Type", "Resource ID", "IP Address", "User Agent", "Created At"})

	// Write data
	for _, log := range logs {
		userID := ""
		if log.UserID != nil {
			userID = log.UserID.String()
		}

		resourceType := ""
		if log.ResourceType != nil {
			resourceType = *log.ResourceType
		}

		resourceID := ""
		if log.ResourceID != nil {
			resourceID = log.ResourceID.String()
		}

		ipAddress := ""
		if log.IPAddress != nil {
			ipAddress = *log.IPAddress
		}

		userAgent := ""
		if log.UserAgent != nil {
			userAgent = *log.UserAgent
		}

		writer.Write([]string{
			log.ID.String(),
			userID,
			log.Action,
			resourceType,
			resourceID,
			ipAddress,
			userAgent,
			log.CreatedAt.Format(time.RFC3339),
		})
	}
}

func (h *Handler) parseFilter(r *http.Request, tenantID *uuid.UUID) *ListFilter {
	filter := &ListFilter{
		TenantID: tenantID,
		Limit:    50,
		Offset:   0,
	}

	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if id, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &id
		}
	}

	if action := r.URL.Query().Get("action"); action != "" {
		filter.Action = &action
	}

	if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
		filter.ResourceType = &resourceType
	}

	if resourceIDStr := r.URL.Query().Get("resource_id"); resourceIDStr != "" {
		if id, err := uuid.Parse(resourceIDStr); err == nil {
			filter.ResourceID = &id
		}
	}

	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		if t, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			filter.StartDate = &t
		}
	}

	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		if t, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			filter.EndDate = &t
		}
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

	return filter
}

func toAuditLogDTO(log *AuditLog) *AuditLogDTO {
	dto := &AuditLogDTO{
		ID:           log.ID.String(),
		Action:       log.Action,
		ResourceType: log.ResourceType,
		Details:      log.Details,
		IPAddress:    log.IPAddress,
		UserAgent:    log.UserAgent,
		CreatedAt:    log.CreatedAt.Format(time.RFC3339),
	}

	if log.UserID != nil {
		s := log.UserID.String()
		dto.UserID = &s
	}

	if log.ResourceID != nil {
		s := log.ResourceID.String()
		dto.ResourceID = &s
	}

	return dto
}
