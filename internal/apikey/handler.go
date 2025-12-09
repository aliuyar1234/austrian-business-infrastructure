package apikey

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/google/uuid"
)

// Handler handles API key HTTP requests
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler creates a new API key handler
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers API key routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth func(http.Handler) http.Handler) {
	router.Handle("POST /api/v1/api-keys", requireAuth(http.HandlerFunc(h.Create)))
	router.Handle("GET /api/v1/api-keys", requireAuth(http.HandlerFunc(h.List)))
	router.Handle("GET /api/v1/api-keys/{id}", requireAuth(http.HandlerFunc(h.Get)))
	router.Handle("DELETE /api/v1/api-keys/{id}", requireAuth(http.HandlerFunc(h.Revoke)))
}

// APIKeyDTO is a data transfer object for API keys
type APIKeyDTO struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	KeyPrefix  string   `json:"key_prefix"`
	Scopes     []string `json:"scopes"`
	ExpiresAt  *string  `json:"expires_at,omitempty"`
	LastUsedAt *string  `json:"last_used_at,omitempty"`
	IsActive   bool     `json:"is_active"`
	CreatedAt  string   `json:"created_at"`
}

// CreateRequest represents a create API key request
type CreateRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresIn *string  `json:"expires_in,omitempty"` // Duration string like "30d", "1y"
}

// CreateResponse represents a create API key response
type CreateResponse struct {
	APIKey *APIKeyDTO `json:"api_key"`
	Key    string     `json:"key"` // Only shown once
}

// Create handles POST /api/v1/api-keys
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	if req.Name == "" {
		api.ValidationError(w, map[string]string{
			"name": "Name is required",
		})
		return
	}

	if len(req.Scopes) == 0 {
		api.ValidationError(w, map[string]string{
			"scopes": "At least one scope is required",
		})
		return
	}

	userID, err := uuid.Parse(api.GetUserID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	var expiresIn *time.Duration
	if req.ExpiresIn != nil {
		d, err := parseDuration(*req.ExpiresIn)
		if err != nil {
			api.ValidationError(w, map[string]string{
				"expires_in": "Invalid duration format. Use formats like '30d', '1y', '6m'",
			})
			return
		}
		expiresIn = &d
	}

	result, err := h.service.Create(r.Context(), &CreateKeyInput{
		UserID:    userID,
		TenantID:  tenantID,
		Name:      req.Name,
		Scopes:    req.Scopes,
		ExpiresIn: expiresIn,
	})

	if err != nil {
		if errors.Is(err, ErrInvalidScope) {
			api.ValidationError(w, map[string]string{
				"scopes": "Invalid scope. Valid scopes: read:all, write:all, read:databox, write:databox, read:users, write:users, read:audit",
			})
			return
		}
		h.logger.Error("failed to create API key", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusCreated, CreateResponse{
		APIKey: toAPIKeyDTO(result.Key),
		Key:    result.RawKey,
	})
}

// List handles GET /api/v1/api-keys
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(api.GetUserID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	keys, err := h.service.ListByUser(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list API keys", "error", err)
		api.InternalError(w)
		return
	}

	dtos := make([]*APIKeyDTO, len(keys))
	for i, k := range keys {
		dtos[i] = toAPIKeyDTO(k)
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"api_keys": dtos,
	})
}

// Get handles GET /api/v1/api-keys/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.BadRequest(w, "Invalid API key ID")
		return
	}

	userID, _ := uuid.Parse(api.GetUserID(r.Context()))

	key, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			api.NotFound(w, "API key not found")
			return
		}
		h.logger.Error("failed to get API key", "error", err)
		api.InternalError(w)
		return
	}

	// Verify ownership
	if key.UserID != userID {
		api.NotFound(w, "API key not found")
		return
	}

	api.JSONResponse(w, http.StatusOK, toAPIKeyDTO(key))
}

// Revoke handles DELETE /api/v1/api-keys/{id}
func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.BadRequest(w, "Invalid API key ID")
		return
	}

	userID, _ := uuid.Parse(api.GetUserID(r.Context()))

	// Verify ownership
	key, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			api.NotFound(w, "API key not found")
			return
		}
		h.logger.Error("failed to get API key", "error", err)
		api.InternalError(w)
		return
	}

	if key.UserID != userID {
		api.NotFound(w, "API key not found")
		return
	}

	if err := h.service.Revoke(r.Context(), id); err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			api.NotFound(w, "API key not found")
			return
		}
		h.logger.Error("failed to revoke API key", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "API key revoked",
	})
}

func toAPIKeyDTO(k *APIKey) *APIKeyDTO {
	dto := &APIKeyDTO{
		ID:        k.ID.String(),
		Name:      k.Name,
		KeyPrefix: k.KeyPrefix,
		Scopes:    k.Scopes,
		IsActive:  k.IsActive,
		CreatedAt: k.CreatedAt.Format(time.RFC3339),
	}

	if k.ExpiresAt != nil {
		formatted := k.ExpiresAt.Format(time.RFC3339)
		dto.ExpiresAt = &formatted
	}

	if k.LastUsedAt != nil {
		formatted := k.LastUsedAt.Format(time.RFC3339)
		dto.LastUsedAt = &formatted
	}

	return dto
}

// parseDuration parses duration strings like "30d", "1y", "6m"
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, errors.New("invalid duration")
	}

	// Get the numeric part
	numStr := s[:len(s)-1]
	unit := s[len(s)-1]

	var num int
	for _, c := range numStr {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid duration")
		}
		num = num*10 + int(c-'0')
	}

	switch unit {
	case 'h':
		return time.Duration(num) * time.Hour, nil
	case 'd':
		return time.Duration(num) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(num) * 7 * 24 * time.Hour, nil
	case 'm':
		return time.Duration(num) * 30 * 24 * time.Hour, nil // Approximate month
	case 'y':
		return time.Duration(num) * 365 * 24 * time.Hour, nil // Approximate year
	default:
		return 0, errors.New("invalid duration unit")
	}
}
