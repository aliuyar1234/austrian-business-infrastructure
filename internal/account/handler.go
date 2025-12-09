package account

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/account/types"
	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/google/uuid"
)

// connectionTestPool limits concurrent background connection tests to prevent DoS (CWE-400)
var connectionTestPool = make(chan struct{}, 10) // Max 10 concurrent tests
var connectionTestOnce sync.Once

func initConnectionTestPool() {
	connectionTestOnce.Do(func() {
		// Pool is already initialized via make()
	})
}

// tryBackgroundConnectionTest attempts to run a connection test in the background
// Returns false if the pool is full (rate limited)
func (h *Handler) tryBackgroundConnectionTest(accountID, tenantID uuid.UUID) bool {
	select {
	case connectionTestPool <- struct{}{}:
		go func() {
			defer func() { <-connectionTestPool }()
			// Use a new context since the request context may be cancelled
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, _ = h.service.TestConnection(ctx, accountID, tenantID)
		}()
		return true
	default:
		// Pool is full, skip background test
		return false
	}
}

// Handler handles account HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new account handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers account routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	// Admin-only: create, update, delete accounts (sensitive credential management)
	router.Handle("POST /api/v1/accounts", requireAuth(requireAdmin(http.HandlerFunc(h.Create))))
	router.Handle("PUT /api/v1/accounts/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.Update))))
	router.Handle("DELETE /api/v1/accounts/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.Delete))))

	// Member access: read-only operations
	router.Handle("GET /api/v1/accounts", requireAuth(http.HandlerFunc(h.List)))
	router.Handle("GET /api/v1/accounts/{id}", requireAuth(http.HandlerFunc(h.Get)))
	router.Handle("POST /api/v1/accounts/{id}/test", requireAuth(http.HandlerFunc(h.TestConnection)))
	router.Handle("GET /api/v1/accounts/{id}/tests", requireAuth(http.HandlerFunc(h.GetConnectionTests)))
}

// CreateAccountRequest represents the create account request
type CreateAccountRequest struct {
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Credentials json.RawMessage `json:"credentials"`
}

// AccountResponse represents an account in API responses
type AccountResponse struct {
	ID             uuid.UUID   `json:"id"`
	Name           string      `json:"name"`
	Type           string      `json:"type"`
	Status         string      `json:"status"`
	Credentials    interface{} `json:"credentials,omitempty"`
	LastVerifiedAt *string     `json:"last_verified_at,omitempty"`
	LastSyncAt     *string     `json:"last_sync_at,omitempty"`
	ErrorMessage   *string     `json:"error_message,omitempty"`
	CreatedAt      string      `json:"created_at"`
	UpdatedAt      string      `json:"updated_at"`
}

// Create handles POST /api/v1/accounts
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.BadRequest(w, "invalid tenant ID")
		return
	}

	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" {
		api.BadRequest(w, "name is required")
		return
	}
	if req.Type == "" {
		api.BadRequest(w, "type is required")
		return
	}

	// Parse credentials based on type
	creds, err := h.parseCredentials(req.Type, req.Credentials)
	if err != nil {
		api.BadRequest(w, err.Error())
		return
	}

	// Create account
	input := &CreateAccountInput{
		TenantID:    tenantUUID,
		Name:        req.Name,
		Type:        req.Type,
		Credentials: creds,
	}

	account, err := h.service.CreateAccount(r.Context(), input)
	if err != nil {
		switch err {
		case ErrDuplicateTID:
			api.Conflict(w, "TID already exists for this tenant")
		case ErrInvalidTID:
			api.BadRequest(w, "invalid TID format")
		case ErrInvalidBenID:
			api.BadRequest(w, "invalid BenID format")
		case ErrInvalidDienstgeberNr:
			api.BadRequest(w, "invalid Dienstgebernummer format")
		case ErrInvalidPIN:
			api.BadRequest(w, "PIN is required")
		case ErrInvalidAccountType:
			api.BadRequest(w, "invalid account type")
		default:
			api.InternalError(w)
		}
		return
	}

	// Auto-test connection (rate-limited to prevent DoS)
	h.tryBackgroundConnectionTest(account.ID, tenantUUID)

	api.JSONResponse(w, http.StatusCreated, h.toResponse(account, nil))
}

// List handles GET /api/v1/accounts
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.BadRequest(w, "invalid tenant ID")
		return
	}

	// Parse query parameters
	filter := ListFilter{
		TenantID: tenantUUID,
		Type:     r.URL.Query().Get("type"),
		Status:   r.URL.Query().Get("status"),
		Search:   r.URL.Query().Get("search"),
		Limit:    50,
		Offset:   0,
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

	// Parse tag filter
	if tagStr := r.URL.Query().Get("tags"); tagStr != "" {
		// Parse comma-separated UUIDs
		// For simplicity, skip tag parsing for now
	}

	accounts, total, err := h.service.ListAccounts(r.Context(), filter)
	if err != nil {
		api.InternalError(w)
		return
	}

	// Convert to response format
	items := make([]*AccountResponse, 0, len(accounts))
	for _, account := range accounts {
		// Get masked credentials for list view
		_, maskedCreds, _ := h.service.GetAccountWithMaskedCredentials(r.Context(), account.ID, tenantUUID)
		items = append(items, h.toResponse(account, maskedCreds))
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": total,
		"limit": filter.Limit,
		"offset": filter.Offset,
	})
}

// Get handles GET /api/v1/accounts/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.BadRequest(w, "invalid tenant ID")
		return
	}

	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid account ID")
		return
	}

	account, maskedCreds, err := h.service.GetAccountWithMaskedCredentials(r.Context(), accountID, tenantUUID)
	if err != nil {
		switch err {
		case ErrAccountNotFound:
			api.NotFound(w, "account not found")
		case ErrAccountDeleted:
			api.NotFound(w, "account has been deleted")
		default:
			api.InternalError(w)
		}
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toResponse(account, maskedCreds))
}

// UpdateAccountRequest represents the update account request
type UpdateAccountRequest struct {
	Name        *string         `json:"name,omitempty"`
	Credentials json.RawMessage `json:"credentials,omitempty"`
}

// Update handles PUT /api/v1/accounts/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.BadRequest(w, "invalid tenant ID")
		return
	}

	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid account ID")
		return
	}

	var req UpdateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	// Get existing account
	account, err := h.service.GetAccount(r.Context(), accountID, tenantUUID)
	if err != nil {
		switch err {
		case ErrAccountNotFound:
			api.NotFound(w, "account not found")
		default:
			api.InternalError(w)
		}
		return
	}

	// Update name if provided
	if req.Name != nil {
		account.Name = *req.Name
		if err := h.service.UpdateAccount(r.Context(), account); err != nil {
			api.InternalError(w)
			return
		}
	}

	// Update credentials if provided
	if len(req.Credentials) > 0 {
		creds, err := h.parseCredentials(account.Type, req.Credentials)
		if err != nil {
			api.BadRequest(w, err.Error())
			return
		}

		if err := h.service.UpdateCredentials(r.Context(), accountID, tenantUUID, account.Type, creds); err != nil {
			api.BadRequest(w, err.Error())
			return
		}

		// Auto-test after credential update (rate-limited to prevent DoS)
		h.tryBackgroundConnectionTest(accountID, tenantUUID)
	}

	// Fetch updated account
	account, maskedCreds, err := h.service.GetAccountWithMaskedCredentials(r.Context(), accountID, tenantUUID)
	if err != nil {
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toResponse(account, maskedCreds))
}

// Delete handles DELETE /api/v1/accounts/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.BadRequest(w, "invalid tenant ID")
		return
	}

	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid account ID")
		return
	}

	// Check for force delete (GDPR)
	force := r.URL.Query().Get("force") == "true"

	if force {
		err = h.service.ForceDeleteAccount(r.Context(), accountID, tenantUUID)
	} else {
		err = h.service.DeleteAccount(r.Context(), accountID, tenantUUID)
	}

	if err != nil {
		switch err {
		case ErrAccountNotFound:
			api.NotFound(w, "account not found")
		default:
			api.InternalError(w)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TestConnection handles POST /api/v1/accounts/{id}/test
func (h *Handler) TestConnection(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.BadRequest(w, "invalid tenant ID")
		return
	}

	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid account ID")
		return
	}

	test, err := h.service.TestConnection(r.Context(), accountID, tenantUUID)
	if err != nil {
		switch err {
		case ErrAccountNotFound:
			api.NotFound(w, "account not found")
		case ErrTestRateLimited:
			api.JSONError(w, http.StatusTooManyRequests, "connection test rate limited", "RATE_LIMITED")
		default:
			api.InternalError(w)
		}
		return
	}

	api.JSONResponse(w, http.StatusOK, test)
}

// GetConnectionTests handles GET /api/v1/accounts/{id}/tests
func (h *Handler) GetConnectionTests(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.BadRequest(w, "invalid tenant ID")
		return
	}

	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid account ID")
		return
	}

	// Verify account belongs to tenant
	_, err = h.service.GetAccount(r.Context(), accountID, tenantUUID)
	if err != nil {
		switch err {
		case ErrAccountNotFound:
			api.NotFound(w, "account not found")
		default:
			api.InternalError(w)
		}
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	tests, err := h.service.GetConnectionTests(r.Context(), accountID, limit)
	if err != nil {
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, tests)
}

func (h *Handler) parseCredentials(accountType string, raw json.RawMessage) (interface{}, error) {
	switch accountType {
	case AccountTypeFinanzOnline:
		var creds types.FinanzOnlineCredentials
		if err := json.Unmarshal(raw, &creds); err != nil {
			return nil, err
		}
		return &creds, nil

	case AccountTypeELDA:
		var creds types.ELDACredentials
		if err := json.Unmarshal(raw, &creds); err != nil {
			return nil, err
		}
		return &creds, nil

	case AccountTypeFirmenbuch:
		var creds types.FirmenbuchCredentials
		if err := json.Unmarshal(raw, &creds); err != nil {
			return nil, err
		}
		return &creds, nil

	default:
		return nil, ErrInvalidAccountType
	}
}

func (h *Handler) toResponse(account *Account, creds interface{}) *AccountResponse {
	resp := &AccountResponse{
		ID:           account.ID,
		Name:         account.Name,
		Type:         account.Type,
		Status:       account.Status,
		Credentials:  creds,
		ErrorMessage: account.ErrorMessage,
		CreatedAt:    account.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    account.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if account.LastVerifiedAt != nil {
		s := account.LastVerifiedAt.Format("2006-01-02T15:04:05Z")
		resp.LastVerifiedAt = &s
	}

	if account.LastSyncAt != nil {
		s := account.LastSyncAt.Format("2006-01-02T15:04:05Z")
		resp.LastSyncAt = &s
	}

	return resp
}
