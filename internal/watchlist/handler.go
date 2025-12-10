package watchlist

import (
	"encoding/json"
	"net/http"
	"time"

	"austrian-business-infrastructure/internal/api"
	"austrian-business-infrastructure/internal/fb"
	"github.com/google/uuid"
)

// Handler handles watchlist HTTP requests
type Handler struct {
	repo     *Repository
	fbClient *fb.Client
}

// NewHandler creates a new watchlist handler
func NewHandler(repo *Repository, fbClient *fb.Client) *Handler {
	return &Handler{
		repo:     repo,
		fbClient: fbClient,
	}
}

// RegisterRoutes registers watchlist routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/watchlist", h.List)
	mux.HandleFunc("POST /api/v1/watchlist", h.Create)
	mux.HandleFunc("GET /api/v1/watchlist/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/watchlist/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/watchlist/{id}", h.Delete)
	mux.HandleFunc("POST /api/v1/watchlist/{id}/check", h.CheckNow)
}

// ItemResponse represents a watchlist item in API responses
type ItemResponse struct {
	ID             string  `json:"id"`
	CompanyNumber  string  `json:"company_number"`
	CompanyName    string  `json:"company_name"`
	AccountID      *string `json:"account_id,omitempty"`
	CheckEnabled   bool    `json:"check_enabled"`
	NotifyOnChange bool    `json:"notify_on_change"`
	LastCheckedAt  *string `json:"last_checked_at,omitempty"`
	LastChangedAt  *string `json:"last_changed_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// List lists watchlist items for a tenant
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	items, err := h.repo.List(ctx, tenantUUID, false)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to list watchlist items", api.ErrCodeInternalError)
		return
	}

	response := make([]*ItemResponse, len(items))
	for i, item := range items {
		response[i] = itemToResponse(item)
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"items": response,
	})
}

// CreateRequest represents the request to create a watchlist item
type CreateRequest struct {
	CompanyNumber  string  `json:"company_number"`
	CompanyName    string  `json:"company_name,omitempty"`
	AccountID      *string `json:"account_id,omitempty"`
	CheckEnabled   *bool   `json:"check_enabled,omitempty"`
	NotifyOnChange *bool   `json:"notify_on_change,omitempty"`
}

// Create creates a new watchlist item
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid request body", api.ErrCodeBadRequest)
		return
	}

	// Validate company number (FN)
	if req.CompanyNumber == "" {
		api.JSONError(w, http.StatusBadRequest, "company_number is required", api.ErrCodeValidation)
		return
	}

	// Validate FN format
	if err := fb.ValidateFN(req.CompanyNumber); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid company_number format (expected FN format like FN123456a)", api.ErrCodeValidation)
		return
	}

	// Parse optional account ID
	var accountID *uuid.UUID
	if req.AccountID != nil && *req.AccountID != "" {
		id, err := uuid.Parse(*req.AccountID)
		if err != nil {
			api.JSONError(w, http.StatusBadRequest, "invalid account_id", api.ErrCodeBadRequest)
			return
		}
		accountID = &id
	}

	// Set defaults
	checkEnabled := true
	if req.CheckEnabled != nil {
		checkEnabled = *req.CheckEnabled
	}

	notifyOnChange := true
	if req.NotifyOnChange != nil {
		notifyOnChange = *req.NotifyOnChange
	}

	// If company name not provided, try to fetch from FB
	companyName := req.CompanyName
	if companyName == "" && h.fbClient != nil {
		extract, err := h.fbClient.Extract(req.CompanyNumber)
		if err == nil && extract != nil {
			companyName = extract.Firma
		}
	}

	item := &Item{
		TenantID:       tenantUUID,
		AccountID:      accountID,
		CompanyNumber:  req.CompanyNumber,
		CompanyName:    companyName,
		CheckEnabled:   checkEnabled,
		NotifyOnChange: notifyOnChange,
	}

	if err := h.repo.Create(ctx, item); err != nil {
		if err == ErrDuplicateCompany {
			api.JSONError(w, http.StatusConflict, "company already in watchlist", api.ErrCodeConflict)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to create watchlist item", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusCreated, itemToResponse(item))
}

// Get retrieves a watchlist item by ID
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	itemID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid item ID", api.ErrCodeBadRequest)
		return
	}

	item, err := h.repo.GetByID(ctx, itemID)
	if err != nil {
		if err == ErrWatchlistItemNotFound {
			api.JSONError(w, http.StatusNotFound, "watchlist item not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get watchlist item", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, itemToResponse(item))
}

// UpdateRequest represents the request to update a watchlist item
type UpdateRequest struct {
	CompanyName    *string `json:"company_name,omitempty"`
	AccountID      *string `json:"account_id,omitempty"`
	CheckEnabled   *bool   `json:"check_enabled,omitempty"`
	NotifyOnChange *bool   `json:"notify_on_change,omitempty"`
}

// Update updates a watchlist item
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	itemID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid item ID", api.ErrCodeBadRequest)
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid request body", api.ErrCodeBadRequest)
		return
	}

	// Get existing item
	item, err := h.repo.GetByID(ctx, itemID)
	if err != nil {
		if err == ErrWatchlistItemNotFound {
			api.JSONError(w, http.StatusNotFound, "watchlist item not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get watchlist item", api.ErrCodeInternalError)
		return
	}

	// Verify tenant
	if item.TenantID != tenantUUID {
		api.JSONError(w, http.StatusNotFound, "watchlist item not found", api.ErrCodeNotFound)
		return
	}

	// Update fields
	if req.CompanyName != nil {
		item.CompanyName = *req.CompanyName
	}
	if req.AccountID != nil {
		if *req.AccountID == "" {
			item.AccountID = nil
		} else {
			id, err := uuid.Parse(*req.AccountID)
			if err != nil {
				api.JSONError(w, http.StatusBadRequest, "invalid account_id", api.ErrCodeBadRequest)
				return
			}
			item.AccountID = &id
		}
	}
	if req.CheckEnabled != nil {
		item.CheckEnabled = *req.CheckEnabled
	}
	if req.NotifyOnChange != nil {
		item.NotifyOnChange = *req.NotifyOnChange
	}

	if err := h.repo.Update(ctx, item); err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to update watchlist item", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, itemToResponse(item))
}

// Delete deletes a watchlist item
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	itemID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid item ID", api.ErrCodeBadRequest)
		return
	}

	if err := h.repo.Delete(ctx, itemID, tenantUUID); err != nil {
		if err == ErrWatchlistItemNotFound {
			api.JSONError(w, http.StatusNotFound, "watchlist item not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to delete watchlist item", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// CheckNow triggers an immediate check for a specific watchlist item
func (h *Handler) CheckNow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	itemID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid item ID", api.ErrCodeBadRequest)
		return
	}

	// Get existing item
	item, err := h.repo.GetByID(ctx, itemID)
	if err != nil {
		if err == ErrWatchlistItemNotFound {
			api.JSONError(w, http.StatusNotFound, "watchlist item not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get watchlist item", api.ErrCodeInternalError)
		return
	}

	// Verify tenant
	if item.TenantID != tenantUUID {
		api.JSONError(w, http.StatusNotFound, "watchlist item not found", api.ErrCodeNotFound)
		return
	}

	// Check if fbClient is available
	if h.fbClient == nil {
		api.JSONError(w, http.StatusServiceUnavailable, "Firmenbuch service not configured", api.ErrCodeInternalError)
		return
	}

	// Fetch current data from FB
	extract, err := h.fbClient.Extract(item.CompanyNumber)
	if err != nil {
		api.JSONError(w, http.StatusBadGateway, "failed to fetch Firmenbuch data", api.ErrCodeInternalError)
		return
	}

	// Serialize for storage
	snapshotBytes, err := json.Marshal(extract)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to serialize snapshot", api.ErrCodeInternalError)
		return
	}

	// Determine if changed
	changed := item.LastSnapshot == nil || !jsonEqual(item.LastSnapshot, snapshotBytes)

	// Update snapshot
	if err := h.repo.UpdateSnapshot(ctx, item.ID, snapshotBytes, changed); err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to update snapshot", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "checked",
		"changed": changed,
		"extract": extract,
	})
}

// Helper functions

func itemToResponse(item *Item) *ItemResponse {
	resp := &ItemResponse{
		ID:             item.ID.String(),
		CompanyNumber:  item.CompanyNumber,
		CompanyName:    item.CompanyName,
		CheckEnabled:   item.CheckEnabled,
		NotifyOnChange: item.NotifyOnChange,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
	}

	if item.AccountID != nil {
		s := item.AccountID.String()
		resp.AccountID = &s
	}

	if item.LastCheckedAt != nil {
		s := item.LastCheckedAt.Format(time.RFC3339)
		resp.LastCheckedAt = &s
	}

	if item.LastChangedAt != nil {
		s := item.LastChangedAt.Format(time.RFC3339)
		resp.LastChangedAt = &s
	}

	return resp
}

// jsonEqual compares two JSON byte slices for equality
func jsonEqual(a, b []byte) bool {
	var va, vb interface{}
	if err := json.Unmarshal(a, &va); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		return false
	}

	// Re-marshal to normalize
	na, _ := json.Marshal(va)
	nb, _ := json.Marshal(vb)

	return string(na) == string(nb)
}
