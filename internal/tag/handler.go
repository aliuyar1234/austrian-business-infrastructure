package tag

import (
	"encoding/json"
	"net/http"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/google/uuid"
)

// Handler handles tag HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new tag handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers tag routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth func(http.Handler) http.Handler) {
	router.Handle("POST /api/v1/tags", requireAuth(http.HandlerFunc(h.Create)))
	router.Handle("GET /api/v1/tags", requireAuth(http.HandlerFunc(h.List)))
	router.Handle("GET /api/v1/tags/{id}", requireAuth(http.HandlerFunc(h.Get)))
	router.Handle("PUT /api/v1/tags/{id}", requireAuth(http.HandlerFunc(h.Update)))
	router.Handle("DELETE /api/v1/tags/{id}", requireAuth(http.HandlerFunc(h.Delete)))
}

// RegisterAccountTagRoutes registers tag routes on accounts
func (h *Handler) RegisterAccountTagRoutes(router *api.Router, requireAuth func(http.Handler) http.Handler) {
	router.Handle("GET /api/v1/accounts/{id}/tags", requireAuth(http.HandlerFunc(h.GetAccountTags)))
	router.Handle("POST /api/v1/accounts/{id}/tags", requireAuth(http.HandlerFunc(h.AddAccountTag)))
	router.Handle("PUT /api/v1/accounts/{id}/tags", requireAuth(http.HandlerFunc(h.SetAccountTags)))
	router.Handle("DELETE /api/v1/accounts/{id}/tags/{tagId}", requireAuth(http.HandlerFunc(h.RemoveAccountTag)))
}

// CreateTagRequest represents the create tag request
type CreateTagRequest struct {
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
}

// Create handles POST /api/v1/tags
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	if req.Name == "" {
		api.BadRequest(w, "name is required")
		return
	}

	tag, err := h.service.CreateTag(r.Context(), tenantUUID, req.Name, req.Color)
	if err != nil {
		if err == ErrTagAlreadyExists {
			api.Conflict(w, "tag with this name already exists")
		} else {
			api.InternalError(w)
		}
		return
	}

	api.JSONResponse(w, http.StatusCreated, tag)
}

// List handles GET /api/v1/tags
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	tags, err := h.service.ListTags(r.Context(), tenantUUID)
	if err != nil {
		api.InternalError(w)
		return
	}

	if tags == nil {
		tags = []*Tag{}
	}

	api.JSONResponse(w, http.StatusOK, tags)
}

// Get handles GET /api/v1/tags/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	tagID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid tag ID")
		return
	}

	tag, err := h.service.GetTag(r.Context(), tagID, tenantUUID)
	if err != nil {
		if err == ErrTagNotFound {
			api.NotFound(w, "tag not found")
		} else {
			api.InternalError(w)
		}
		return
	}

	api.JSONResponse(w, http.StatusOK, tag)
}

// Update handles PUT /api/v1/tags/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	tagID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid tag ID")
		return
	}

	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	if req.Name == "" {
		api.BadRequest(w, "name is required")
		return
	}

	tag, err := h.service.UpdateTag(r.Context(), tagID, tenantUUID, req.Name, req.Color)
	if err != nil {
		if err == ErrTagNotFound {
			api.NotFound(w, "tag not found")
		} else if err == ErrTagAlreadyExists {
			api.Conflict(w, "tag with this name already exists")
		} else {
			api.InternalError(w)
		}
		return
	}

	api.JSONResponse(w, http.StatusOK, tag)
}

// Delete handles DELETE /api/v1/tags/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	tagID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid tag ID")
		return
	}

	err = h.service.DeleteTag(r.Context(), tagID, tenantUUID)
	if err != nil {
		if err == ErrTagNotFound {
			api.NotFound(w, "tag not found")
		} else {
			api.InternalError(w)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetAccountTags handles GET /api/v1/accounts/{id}/tags
func (h *Handler) GetAccountTags(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid account ID")
		return
	}

	tags, err := h.service.GetAccountTags(r.Context(), accountID)
	if err != nil {
		api.InternalError(w)
		return
	}

	if tags == nil {
		tags = []*Tag{}
	}

	api.JSONResponse(w, http.StatusOK, tags)
}

// AddTagRequest represents the add tag request
type AddTagRequest struct {
	TagID uuid.UUID `json:"tag_id"`
}

// AddAccountTag handles POST /api/v1/accounts/{id}/tags
func (h *Handler) AddAccountTag(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid account ID")
		return
	}

	var req AddTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	err = h.service.AddTagToAccount(r.Context(), accountID, req.TagID, tenantUUID)
	if err != nil {
		if err == ErrTagNotFound {
			api.NotFound(w, "tag not found")
		} else {
			api.InternalError(w)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetTagsRequest represents the set tags request
type SetTagsRequest struct {
	TagIDs []uuid.UUID `json:"tag_ids"`
}

// SetAccountTags handles PUT /api/v1/accounts/{id}/tags
func (h *Handler) SetAccountTags(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid account ID")
		return
	}

	var req SetTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	err = h.service.SetAccountTags(r.Context(), accountID, tenantUUID, req.TagIDs)
	if err != nil {
		if err == ErrTagNotFound {
			api.BadRequest(w, "one or more tags not found")
		} else {
			api.InternalError(w)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveAccountTag handles DELETE /api/v1/accounts/{id}/tags/{tagId}
func (h *Handler) RemoveAccountTag(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid account ID")
		return
	}

	tagID, err := uuid.Parse(r.PathValue("tagId"))
	if err != nil {
		api.BadRequest(w, "invalid tag ID")
		return
	}

	err = h.service.RemoveTagFromAccount(r.Context(), accountID, tagID)
	if err != nil {
		api.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
