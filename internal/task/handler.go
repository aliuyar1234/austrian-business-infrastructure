package task

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/client"
	"github.com/austrian-business-infrastructure/fo/internal/tenant"
)

// Handler handles task-related HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new task handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// StaffRoutes returns the routes for staff managing tasks
func (h *Handler) StaffRoutes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/overdue", h.ListOverdue)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Post("/{id}/cancel", h.Cancel)
	r.Delete("/{id}", h.Delete)

	return r
}

// PortalRoutes returns the routes for portal clients
func (h *Handler) PortalRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListForClient)
	r.Get("/{id}", h.GetByIDForClient)
	r.Post("/{id}/complete", h.CompleteForClient)

	return r
}

// ============== Staff Endpoints ==============

// Create creates a new task
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
		ClientID    uuid.UUID  `json:"client_id"`
		Title       string     `json:"title"`
		Description *string    `json:"description,omitempty"`
		Priority    Priority   `json:"priority"`
		DueDate     *time.Time `json:"due_date,omitempty"`
		DocumentID  *uuid.UUID `json:"document_id,omitempty"`
		UploadID    *uuid.UUID `json:"upload_id,omitempty"`
		ApprovalID  *uuid.UUID `json:"approval_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ClientID == uuid.Nil || req.Title == "" {
		http.Error(w, "client_id and title are required", http.StatusBadRequest)
		return
	}

	task, err := h.service.Create(ctx, &CreateRequest{
		TenantID:    tenantID,
		ClientID:    req.ClientID,
		CreatedBy:   userID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
		DocumentID:  req.DocumentID,
		UploadID:    req.UploadID,
		ApprovalID:  req.ApprovalID,
	})
	if err != nil {
		http.Error(w, "failed to create task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

// List returns tasks for the tenant
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var status *Status
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
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

	tasks, total, err := h.service.ListForTenant(ctx, tenantID, status, limit, offset)
	if err != nil {
		http.Error(w, "failed to list tasks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks":  tasks,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ListOverdue returns overdue tasks
func (h *Handler) ListOverdue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	tasks, err := h.service.ListOverdue(ctx, tenantID)
	if err != nil {
		http.Error(w, "failed to list overdue tasks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": tasks,
	})
}

// GetByID returns a task by ID
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	task, err := h.service.GetByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get task", http.StatusInternalServerError)
		return
	}

	// Verify tenant access
	if task.TenantID != tenantID {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// Update updates a task
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	// Verify tenant access
	existing, err := h.service.GetByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get task", http.StatusInternalServerError)
		return
	}

	if existing.TenantID != tenantID {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	task, err := h.service.Update(ctx, taskID, &req)
	if err != nil {
		http.Error(w, "failed to update task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// Cancel cancels a task
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	// Verify tenant access
	existing, err := h.service.GetByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get task", http.StatusInternalServerError)
		return
	}

	if existing.TenantID != tenantID {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	if err := h.service.Cancel(ctx, taskID); err != nil {
		http.Error(w, "failed to cancel task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete deletes a task
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	// Verify tenant access
	existing, err := h.service.GetByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get task", http.StatusInternalServerError)
		return
	}

	if existing.TenantID != tenantID {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	if err := h.service.Delete(ctx, taskID); err != nil {
		http.Error(w, "failed to delete task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============== Portal Endpoints ==============

// ListForClient returns tasks for the current client
func (h *Handler) ListForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var status *Status
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
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

	tasks, total, err := h.service.ListForClient(ctx, claims.ClientID, status, limit, offset)
	if err != nil {
		http.Error(w, "failed to list tasks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks":  tasks,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetByIDForClient returns a task by ID for the current client
func (h *Handler) GetByIDForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	task, err := h.service.GetByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get task", http.StatusInternalServerError)
		return
	}

	// Verify client access
	if task.ClientID != claims.ClientID {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// CompleteForClient marks a task as completed by the client
func (h *Handler) CompleteForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	// Verify client access
	task, err := h.service.GetByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get task", http.StatusInternalServerError)
		return
	}

	if task.ClientID != claims.ClientID {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	if task.Status != StatusOpen {
		http.Error(w, "task is not open", http.StatusConflict)
		return
	}

	if err := h.service.Complete(ctx, taskID); err != nil {
		http.Error(w, "failed to complete task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "completed"})
}
