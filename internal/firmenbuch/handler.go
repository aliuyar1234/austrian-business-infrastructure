package firmenbuch

import (
	"encoding/json"
	"net/http"
	"strconv"

	"austrian-business-infrastructure/internal/api"
	"github.com/google/uuid"
)

// Handler handles firmenbuch HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new firmenbuch handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers firmenbuch routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	// Admin-only: watchlist management
	router.Handle("POST /api/v1/firmenbuch/watchlist", requireAuth(requireAdmin(http.HandlerFunc(h.AddToWatchlist))))
	router.Handle("DELETE /api/v1/firmenbuch/watchlist/{fn}", requireAuth(requireAdmin(http.HandlerFunc(h.RemoveFromWatchlist))))

	// Member access: search, read, validate operations
	router.Handle("GET /api/v1/firmenbuch/search", requireAuth(http.HandlerFunc(h.Search)))
	router.Handle("GET /api/v1/firmenbuch/extract/{fn}", requireAuth(http.HandlerFunc(h.GetExtract)))
	router.Handle("POST /api/v1/firmenbuch/validate", requireAuth(http.HandlerFunc(h.ValidateFN)))
	router.Handle("GET /api/v1/firmenbuch/companies", requireAuth(http.HandlerFunc(h.ListCompanies)))
	router.Handle("GET /api/v1/firmenbuch/companies/{fn}", requireAuth(http.HandlerFunc(h.GetCompany)))
	router.Handle("GET /api/v1/firmenbuch/companies/{fn}/history", requireAuth(http.HandlerFunc(h.GetHistory)))
	router.Handle("GET /api/v1/firmenbuch/watchlist", requireAuth(http.HandlerFunc(h.ListWatchlist)))
}

// Search handles GET /api/v1/firmenbuch/search
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	input := &SearchInput{
		Name: r.URL.Query().Get("name"),
		FN:   r.URL.Query().Get("fn"),
		Ort:  r.URL.Query().Get("ort"),
	}

	if maxHitsStr := r.URL.Query().Get("max_hits"); maxHitsStr != "" {
		if maxHits, err := strconv.Atoi(maxHitsStr); err == nil && maxHits > 0 {
			input.MaxHits = maxHits
		}
	}

	resp, err := h.service.Search(r.Context(), tenantID, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, resp)
}

// GetExtract handles GET /api/v1/firmenbuch/extract/{fn}
func (h *Handler) GetExtract(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	fn := r.PathValue("fn")
	if fn == "" {
		api.BadRequest(w, "fn is required")
		return
	}

	forceRefresh := r.URL.Query().Get("refresh") == "true"

	resp, err := h.service.GetExtract(r.Context(), tenantID, fn, forceRefresh)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, resp)
}

// ValidateFN handles POST /api/v1/firmenbuch/validate
func (h *Handler) ValidateFN(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FN string `json:"fn"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	if input.FN == "" {
		api.BadRequest(w, "fn is required")
		return
	}

	err := h.service.ValidateFN(input.FN)

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"fn":    input.FN,
		"valid": err == nil,
		"error": func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
	})
}

// ListCompanies handles GET /api/v1/firmenbuch/companies
func (h *Handler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	filter := ListFilter{
		TenantID: tenantID,
		Limit:    50,
		Offset:   0,
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = &search
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

	companies, total, err := h.service.ListCachedCompanies(r.Context(), filter)
	if err != nil {
		api.InternalError(w)
		return
	}

	items := make([]map[string]interface{}, 0, len(companies))
	for _, company := range companies {
		item := map[string]interface{}{
			"id":         company.ID,
			"fn":         company.FN,
			"name":       company.Name,
			"rechtsform": company.Rechtsform,
			"sitz":       company.Sitz,
			"status":     company.Status,
			"created_at": company.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if company.LastFetchedAt != nil {
			item["last_fetched_at"] = company.LastFetchedAt.Format("2006-01-02T15:04:05Z")
		}
		items = append(items, item)
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetCompany handles GET /api/v1/firmenbuch/companies/{fn}
func (h *Handler) GetCompany(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	fn := r.PathValue("fn")
	if fn == "" {
		api.BadRequest(w, "fn is required")
		return
	}

	// Use cached version
	resp, err := h.service.GetExtract(r.Context(), tenantID, fn, false)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, resp)
}

// GetHistory handles GET /api/v1/firmenbuch/companies/{fn}/history
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	fn := r.PathValue("fn")
	if fn == "" {
		api.BadRequest(w, "fn is required")
		return
	}

	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	entries, total, err := h.service.GetCompanyHistory(r.Context(), tenantID, fn, limit, offset)
	if err != nil {
		h.handleError(w, err)
		return
	}

	items := make([]HistoryResponse, 0, len(entries))
	for _, entry := range entries {
		items = append(items, HistoryResponse{
			ID:         entry.ID,
			ChangeType: entry.ChangeType,
			OldValue:   entry.OldValue,
			NewValue:   entry.NewValue,
			DetectedAt: entry.DetectedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// AddToWatchlist handles POST /api/v1/firmenbuch/watchlist
func (h *Handler) AddToWatchlist(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	var input WatchlistInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	if input.FN == "" {
		api.BadRequest(w, "fn is required")
		return
	}

	entry, err := h.service.AddToWatchlist(r.Context(), tenantID, &input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusCreated, h.toWatchlistResponse(entry))
}

// ListWatchlist handles GET /api/v1/firmenbuch/watchlist
func (h *Handler) ListWatchlist(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	entries, total, err := h.service.ListWatchlist(r.Context(), tenantID, limit, offset)
	if err != nil {
		api.InternalError(w)
		return
	}

	items := make([]WatchlistResponse, 0, len(entries))
	for _, entry := range entries {
		items = append(items, h.toWatchlistResponse(entry))
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// RemoveFromWatchlist handles DELETE /api/v1/firmenbuch/watchlist/{fn}
func (h *Handler) RemoveFromWatchlist(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	fn := r.PathValue("fn")
	if fn == "" {
		api.BadRequest(w, "fn is required")
		return
	}

	if err := h.service.RemoveFromWatchlist(r.Context(), tenantID, fn); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *Handler) getTenantID(r *http.Request) (uuid.UUID, error) {
	tenantIDStr := api.GetTenantID(r.Context())
	if tenantIDStr == "" {
		return uuid.Nil, ErrCompanyNotFound
	}
	return uuid.Parse(tenantIDStr)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch err {
	case ErrCompanyNotFound:
		api.NotFound(w, "company not found")
	case ErrWatchlistNotFound:
		api.NotFound(w, "watchlist entry not found")
	case ErrInvalidFN:
		api.BadRequest(w, "invalid Firmenbuch number format (expected: FN followed by 1-9 digits and a lowercase letter, e.g., FN123456a)")
	case ErrSearchEmpty:
		api.BadRequest(w, "at least one search parameter (name, fn, ort) is required")
	case ErrAlreadyOnWatch:
		api.Conflict(w, "company already on watchlist")
	default:
		api.InternalError(w)
	}
}

func (h *Handler) toWatchlistResponse(entry *WatchlistEntry) WatchlistResponse {
	resp := WatchlistResponse{
		ID:         entry.ID,
		FN:         entry.FN,
		Name:       entry.Name,
		LastStatus: entry.LastStatus,
		Notes:      entry.Notes,
		CreatedAt:  entry.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if entry.LastChecked != nil {
		d := entry.LastChecked.Format("2006-01-02T15:04:05Z")
		resp.LastChecked = &d
	}

	return resp
}
