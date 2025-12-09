package approval

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/client"
	"github.com/austrian-business-infrastructure/fo/internal/tenant"
)

// Handler handles approval-related HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new approval handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// StaffRoutes returns the routes for staff managing approvals
func (h *Handler) StaffRoutes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/", h.ListPending)
	r.Get("/{id}", h.GetByID)

	return r
}

// PortalRoutes returns the routes for portal clients
func (h *Handler) PortalRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListForClient)
	r.Get("/{id}", h.GetByIDForClient)
	r.Post("/{id}/approve", h.Approve)
	r.Post("/{id}/reject", h.Reject)

	return r
}

// Create creates a new approval request
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userIDStr := api.GetUserID(ctx)
	userID, _ := uuid.Parse(userIDStr)

	var req struct {
		DocumentID uuid.UUID `json:"document_id"`
		ClientID   uuid.UUID `json:"client_id"`
		Message    *string   `json:"message,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.DocumentID == uuid.Nil || req.ClientID == uuid.Nil {
		http.Error(w, "document_id and client_id are required", http.StatusBadRequest)
		return
	}

	approval, err := h.service.Create(ctx, &CreateRequest{
		DocumentID:  req.DocumentID,
		ClientID:    req.ClientID,
		RequestedBy: userID,
		Message:     req.Message,
	})
	if err != nil {
		http.Error(w, "failed to create approval request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(approval)
}

// ListPending returns pending approvals for the tenant
func (h *Handler) ListPending(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	approvals, total, err := h.service.ListPendingForTenant(ctx, tenantID, limit, offset)
	if err != nil {
		http.Error(w, "failed to list approvals", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"approvals": approvals,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetByID returns an approval by ID
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	approvalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid approval ID", http.StatusBadRequest)
		return
	}

	approval, err := h.service.GetByIDWithDetails(ctx, approvalID)
	if err != nil {
		if errors.Is(err, ErrApprovalNotFound) {
			http.Error(w, "approval not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get approval", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approval)
}

// ============== Portal Endpoints ==============

// ListForClient returns approvals for the current client
func (h *Handler) ListForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var status *Status
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		if !IsValidStatus(statusStr) {
			http.Error(w, "invalid status", http.StatusBadRequest)
			return
		}
		s := Status(statusStr)
		status = &s
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	approvals, total, err := h.service.ListForClient(ctx, claims.ClientID, status, limit, offset)
	if err != nil {
		http.Error(w, "failed to list approvals", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"approvals": approvals,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetByIDForClient returns an approval by ID for the current client
func (h *Handler) GetByIDForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	approvalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid approval ID", http.StatusBadRequest)
		return
	}

	approval, err := h.service.GetByIDWithDetails(ctx, approvalID)
	if err != nil {
		if errors.Is(err, ErrApprovalNotFound) {
			http.Error(w, "approval not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get approval", http.StatusInternalServerError)
		return
	}

	// Verify client access
	if approval.ClientID != claims.ClientID {
		http.Error(w, "approval not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approval)
}

// Approve approves a document
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	approvalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid approval ID", http.StatusBadRequest)
		return
	}

	// Verify client owns this approval
	approval, err := h.service.GetByID(ctx, approvalID)
	if err != nil {
		if errors.Is(err, ErrApprovalNotFound) {
			http.Error(w, "approval not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get approval", http.StatusInternalServerError)
		return
	}

	if approval.ClientID != claims.ClientID {
		http.Error(w, "approval not found", http.StatusNotFound)
		return
	}

	if approval.Status != StatusPending {
		http.Error(w, "approval already responded", http.StatusConflict)
		return
	}

	if err := h.service.Approve(ctx, approvalID); err != nil {
		http.Error(w, "failed to approve", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

// Reject rejects a document
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	approvalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid approval ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Comment == "" {
		http.Error(w, "comment is required", http.StatusBadRequest)
		return
	}

	// Verify client owns this approval
	approval, err := h.service.GetByID(ctx, approvalID)
	if err != nil {
		if errors.Is(err, ErrApprovalNotFound) {
			http.Error(w, "approval not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get approval", http.StatusInternalServerError)
		return
	}

	if approval.ClientID != claims.ClientID {
		http.Error(w, "approval not found", http.StatusNotFound)
		return
	}

	if approval.Status != StatusPending {
		http.Error(w, "approval already responded", http.StatusConflict)
		return
	}

	if err := h.service.Reject(ctx, approvalID, req.Comment); err != nil {
		http.Error(w, "failed to reject", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}
