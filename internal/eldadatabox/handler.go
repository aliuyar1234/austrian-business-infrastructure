package eldadatabox

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/api"
)

// Handler handles HTTP requests for ELDA databox operations
type Handler struct {
	service *Service
}

// NewHandler creates a new ELDA databox handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the router for ELDA databox endpoints
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Sync operations
	r.Post("/accounts/{id}/sync", h.Sync)
	r.Get("/accounts/{id}/sync-status", h.GetSyncStatus)

	// Document operations
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Get("/{id}/content", h.GetContent)
	r.Post("/{id}/mark-read", h.MarkAsRead)
	r.Delete("/{id}", h.Delete)

	// Stats
	r.Get("/accounts/{id}/unread-count", h.GetUnreadCount)
	r.Get("/categories", h.GetCategories)

	return r
}

// Sync handles POST /api/v1/elda-databox/accounts/{id}/sync
func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	accountID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Ungültige Account-ID")
		return
	}

	result, err := h.service.Sync(r.Context(), accountID)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, result)
}

// GetSyncStatus handles GET /api/v1/elda-databox/accounts/{id}/sync-status
func (h *Handler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	accountID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Ungültige Account-ID")
		return
	}

	status, err := h.service.GetSyncStatus(r.Context(), accountID)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, status)
}

// List handles GET /api/v1/elda-databox/
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := ListFilter{
		Limit:  100,
		Offset: 0,
	}

	// Parse query parameters
	if v := r.URL.Query().Get("elda_account_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.ELDAAccountID = &id
		}
	}

	if v := r.URL.Query().Get("category"); v != "" {
		filter.Category = v
	}

	if v := r.URL.Query().Get("unread"); v == "true" {
		filter.Unread = true
	}

	if v := r.URL.Query().Get("start_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.StartDate = &t
		}
	}

	if v := r.URL.Query().Get("end_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.EndDate = &t
		}
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil && limit > 0 && limit <= 500 {
			filter.Limit = limit
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if offset, err := strconv.Atoi(v); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	documents, err := h.service.List(r.Context(), filter)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	count, _ := h.service.Count(r.Context(), filter)

	api.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   documents,
		"total":  count,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// Get handles GET /api/v1/elda-databox/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	doc, err := h.service.Get(r.Context(), id)
	if err != nil {
		api.RespondError(w, http.StatusNotFound, "Dokument nicht gefunden")
		return
	}

	api.RespondJSON(w, http.StatusOK, doc)
}

// GetContent handles GET /api/v1/elda-databox/{id}/content
func (h *Handler) GetContent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	content, contentType, err := h.service.GetContent(r.Context(), id)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get document for filename
	doc, _ := h.service.Get(r.Context(), id)
	filename := "document"
	if doc != nil {
		filename = doc.Name
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// MarkAsRead handles POST /api/v1/elda-databox/{id}/mark-read
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	if err := h.service.MarkAsRead(r.Context(), id); err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]string{"message": "Als gelesen markiert"})
}

// Delete handles DELETE /api/v1/elda-databox/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]string{"message": "Dokument gelöscht"})
}

// GetUnreadCount handles GET /api/v1/elda-databox/accounts/{id}/unread-count
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	accountID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Ungültige Account-ID")
		return
	}

	count, err := h.service.GetUnreadCount(r.Context(), accountID)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]int{"unread_count": count})
}

// GetCategories handles GET /api/v1/elda-databox/categories
func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories := h.service.GetCategories()
	api.RespondJSON(w, http.StatusOK, map[string][]string{"categories": categories})
}


// RegisterRoutes registers ELDA databox routes with the router
func RegisterRoutes(r chi.Router, service *Service) {
	handler := NewHandler(service)
	r.Mount("/elda-databox", handler.Routes())
}
