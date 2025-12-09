package user

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/google/uuid"
)

// Handler handles user HTTP requests
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler creates a new user handler
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers user routes with the given middleware
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	router.Handle("GET /api/v1/users", requireAuth(http.HandlerFunc(h.List)))
	router.Handle("GET /api/v1/users/{id}", requireAuth(http.HandlerFunc(h.Get)))
	router.Handle("PATCH /api/v1/users/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.Update))))
	router.Handle("DELETE /api/v1/users/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.Deactivate))))
	router.Handle("GET /api/v1/users/me", requireAuth(http.HandlerFunc(h.GetMe)))
	router.Handle("PATCH /api/v1/users/me", requireAuth(http.HandlerFunc(h.UpdateMe)))
}

// UserDTO is a data transfer object for users
type UserDTO struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	Name          string  `json:"name"`
	Role          string  `json:"role"`
	EmailVerified bool    `json:"email_verified"`
	AvatarURL     *string `json:"avatar_url,omitempty"`
	IsActive      bool    `json:"is_active"`
	LastLoginAt   *string `json:"last_login_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

// List handles GET /api/v1/users
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	users, err := h.service.ListByTenant(r.Context(), tenantID)
	if err != nil {
		h.logger.Error("failed to list users", "error", err)
		api.InternalError(w)
		return
	}

	dtos := make([]*UserDTO, len(users))
	for i, u := range users {
		dtos[i] = toUserDTO(u)
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"users": dtos,
	})
}

// Get handles GET /api/v1/users/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.BadRequest(w, "Invalid user ID")
		return
	}

	// Verify user belongs to same tenant
	tenantID, _ := uuid.Parse(api.GetTenantID(r.Context()))

	u, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			api.NotFound(w, "User not found")
			return
		}
		h.logger.Error("failed to get user", "error", err)
		api.InternalError(w)
		return
	}

	if u.TenantID != tenantID {
		api.NotFound(w, "User not found")
		return
	}

	api.JSONResponse(w, http.StatusOK, toUserDTO(u))
}

// UpdateRequest represents an update user request
type UpdateRequest struct {
	Name *string `json:"name,omitempty"`
	Role *string `json:"role,omitempty"`
}

// Update handles PATCH /api/v1/users/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.BadRequest(w, "Invalid user ID")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	// Verify user belongs to same tenant
	tenantID, _ := uuid.Parse(api.GetTenantID(r.Context()))
	actorID, _ := uuid.Parse(api.GetUserID(r.Context()))

	u, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			api.NotFound(w, "User not found")
			return
		}
		h.logger.Error("failed to get user", "error", err)
		api.InternalError(w)
		return
	}

	if u.TenantID != tenantID {
		api.NotFound(w, "User not found")
		return
	}

	// Update name if provided
	if req.Name != nil {
		u.Name = *req.Name
	}

	// Update role if provided
	if req.Role != nil {
		newRole := Role(*req.Role)
		if !IsValidRole(string(newRole)) {
			api.ValidationError(w, map[string]string{
				"role": "Invalid role. Must be owner, admin, member, or viewer",
			})
			return
		}

		if err := h.service.UpdateRole(r.Context(), id, newRole, actorID); err != nil {
			h.handleRoleError(w, err)
			return
		}
		u.Role = newRole
	} else if req.Name != nil {
		// Just update name
		if err := h.service.repo.Update(r.Context(), u); err != nil {
			h.logger.Error("failed to update user", "error", err)
			api.InternalError(w)
			return
		}
	}

	api.JSONResponse(w, http.StatusOK, toUserDTO(u))
}

func (h *Handler) handleRoleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrCannotRemoveLastOwner):
		api.Conflict(w, "Cannot remove the last owner of the organization")
	case errors.Is(err, ErrInvalidRole):
		api.ValidationError(w, map[string]string{
			"role": "Invalid role",
		})
	default:
		h.logger.Error("failed to update role", "error", err)
		api.InternalError(w)
	}
}

// Deactivate handles DELETE /api/v1/users/{id}
func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.BadRequest(w, "Invalid user ID")
		return
	}

	// Verify user belongs to same tenant
	tenantID, _ := uuid.Parse(api.GetTenantID(r.Context()))
	actorID, _ := uuid.Parse(api.GetUserID(r.Context()))

	// Cannot deactivate yourself
	if id == actorID {
		api.Conflict(w, "Cannot deactivate your own account")
		return
	}

	u, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			api.NotFound(w, "User not found")
			return
		}
		h.logger.Error("failed to get user", "error", err)
		api.InternalError(w)
		return
	}

	if u.TenantID != tenantID {
		api.NotFound(w, "User not found")
		return
	}

	if err := h.service.Deactivate(r.Context(), id); err != nil {
		if errors.Is(err, ErrCannotRemoveLastOwner) {
			api.Conflict(w, "Cannot deactivate the last owner of the organization")
			return
		}
		h.logger.Error("failed to deactivate user", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "User deactivated",
	})
}

// GetMe handles GET /api/v1/users/me
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(api.GetUserID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	u, err := h.service.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			api.NotFound(w, "User not found")
			return
		}
		h.logger.Error("failed to get user", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, toUserDTO(u))
}

// UpdateMeRequest represents an update current user request
type UpdateMeRequest struct {
	Name      *string `json:"name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// UpdateMe handles PATCH /api/v1/users/me
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(api.GetUserID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	var req UpdateMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	u, err := h.service.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			api.NotFound(w, "User not found")
			return
		}
		h.logger.Error("failed to get user", "error", err)
		api.InternalError(w)
		return
	}

	if req.Name != nil {
		u.Name = *req.Name
	}
	if req.AvatarURL != nil {
		u.AvatarURL = req.AvatarURL
	}

	if err := h.service.repo.Update(r.Context(), u); err != nil {
		h.logger.Error("failed to update user", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, toUserDTO(u))
}

func toUserDTO(u *User) *UserDTO {
	dto := &UserDTO{
		ID:            u.ID.String(),
		Email:         u.Email,
		Name:          u.Name,
		Role:          string(u.Role),
		EmailVerified: u.EmailVerified,
		AvatarURL:     u.AvatarURL,
		IsActive:      u.IsActive,
		CreatedAt:     u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if u.LastLoginAt != nil {
		formatted := u.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
		dto.LastLoginAt = &formatted
	}

	return dto
}
