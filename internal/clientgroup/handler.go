package clientgroup

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/austrian-business-infrastructure/fo/internal/tenant"
)

// Handler handles client group HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new client group handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Routes returns the routes for client groups
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/{id}/members", h.ListMembers)
	r.Put("/{id}/members", h.SetMembers)
	r.Post("/{id}/members/{clientId}", h.AddMember)
	r.Delete("/{id}/members/{clientId}", h.RemoveMember)

	return r
}

// Create creates a new client group
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name        string  `json:"name"`
		Description *string `json:"description,omitempty"`
		Color       *string `json:"color,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	group, err := h.service.Create(ctx, &CreateRequest{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
	})
	if err != nil {
		http.Error(w, "failed to create group", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}

// List returns all client groups for the tenant
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	groups, err := h.service.ListForTenant(ctx, tenantID)
	if err != nil {
		http.Error(w, "failed to list groups", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"groups": groups,
	})
}

// GetByID returns a client group by ID
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid group ID", http.StatusBadRequest)
		return
	}

	group, err := h.service.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			http.Error(w, "group not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get group", http.StatusInternalServerError)
		return
	}

	// Verify tenant access
	if group.TenantID != tenantID {
		http.Error(w, "group not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(group)
}

// Update updates a client group
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid group ID", http.StatusBadRequest)
		return
	}

	// Verify tenant access
	existing, err := h.service.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			http.Error(w, "group not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get group", http.StatusInternalServerError)
		return
	}

	if existing.TenantID != tenantID {
		http.Error(w, "group not found", http.StatusNotFound)
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	group, err := h.service.Update(ctx, groupID, &req)
	if err != nil {
		http.Error(w, "failed to update group", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(group)
}

// Delete deletes a client group
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid group ID", http.StatusBadRequest)
		return
	}

	// Verify tenant access
	existing, err := h.service.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			http.Error(w, "group not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get group", http.StatusInternalServerError)
		return
	}

	if existing.TenantID != tenantID {
		http.Error(w, "group not found", http.StatusNotFound)
		return
	}

	if err := h.service.Delete(ctx, groupID); err != nil {
		http.Error(w, "failed to delete group", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListMembers returns all members of a group
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid group ID", http.StatusBadRequest)
		return
	}

	// Verify tenant access
	existing, err := h.service.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			http.Error(w, "group not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get group", http.StatusInternalServerError)
		return
	}

	if existing.TenantID != tenantID {
		http.Error(w, "group not found", http.StatusNotFound)
		return
	}

	members, err := h.service.ListMembers(ctx, groupID)
	if err != nil {
		http.Error(w, "failed to list members", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"members": members,
	})
}

// SetMembers replaces all members of a group
func (h *Handler) SetMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid group ID", http.StatusBadRequest)
		return
	}

	// Verify tenant access
	existing, err := h.service.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			http.Error(w, "group not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get group", http.StatusInternalServerError)
		return
	}

	if existing.TenantID != tenantID {
		http.Error(w, "group not found", http.StatusNotFound)
		return
	}

	var req struct {
		ClientIDs []uuid.UUID `json:"client_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.SetMembers(ctx, groupID, req.ClientIDs); err != nil {
		http.Error(w, "failed to set members", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddMember adds a client to a group
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid group ID", http.StatusBadRequest)
		return
	}

	clientID, err := uuid.Parse(chi.URLParam(r, "clientId"))
	if err != nil {
		http.Error(w, "invalid client ID", http.StatusBadRequest)
		return
	}

	// Verify tenant access
	existing, err := h.service.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			http.Error(w, "group not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get group", http.StatusInternalServerError)
		return
	}

	if existing.TenantID != tenantID {
		http.Error(w, "group not found", http.StatusNotFound)
		return
	}

	if err := h.service.AddMember(ctx, groupID, clientID); err != nil {
		if errors.Is(err, ErrMemberExists) {
			http.Error(w, "client already in group", http.StatusConflict)
			return
		}
		http.Error(w, "failed to add member", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveMember removes a client from a group
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid group ID", http.StatusBadRequest)
		return
	}

	clientID, err := uuid.Parse(chi.URLParam(r, "clientId"))
	if err != nil {
		http.Error(w, "invalid client ID", http.StatusBadRequest)
		return
	}

	// Verify tenant access
	existing, err := h.service.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			http.Error(w, "group not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get group", http.StatusInternalServerError)
		return
	}

	if existing.TenantID != tenantID {
		http.Error(w, "group not found", http.StatusNotFound)
		return
	}

	if err := h.service.RemoveMember(ctx, groupID, clientID); err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			http.Error(w, "member not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to remove member", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
