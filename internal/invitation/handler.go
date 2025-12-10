package invitation

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"austrian-business-infrastructure/internal/api"
	"austrian-business-infrastructure/internal/auth"
	"austrian-business-infrastructure/internal/tenant"
	"austrian-business-infrastructure/internal/user"
	"austrian-business-infrastructure/pkg/crypto"
	"github.com/google/uuid"
)

// Handler handles invitation HTTP requests
type Handler struct {
	service       *Service
	tenantService *tenant.Service
	jwtManager    *auth.JWTManager
	sessionMgr    *auth.SessionManager
	logger        *slog.Logger
}

// NewHandler creates a new invitation handler
func NewHandler(
	service *Service,
	tenantService *tenant.Service,
	jwtManager *auth.JWTManager,
	sessionMgr *auth.SessionManager,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		service:       service,
		tenantService: tenantService,
		jwtManager:    jwtManager,
		sessionMgr:    sessionMgr,
		logger:        logger,
	}
}

// RegisterRoutes registers invitation routes
func (h *Handler) RegisterRoutes(router *api.Router, authMw *auth.AuthMiddleware) {
	// Protected routes - require authentication
	router.Handle("POST /api/v1/invitations",
		authMw.RequireAuth(
			authMw.RequireRole("admin")(http.HandlerFunc(h.Create)),
		),
	)
	router.Handle("GET /api/v1/invitations",
		authMw.RequireAuth(http.HandlerFunc(h.List)),
	)
	router.Handle("DELETE /api/v1/invitations/{id}",
		authMw.RequireAuth(
			authMw.RequireRole("admin")(http.HandlerFunc(h.Delete)),
		),
	)

	// Public routes - token-based access
	router.HandleFunc("GET /api/v1/invitations/validate/{token}", h.Validate)
	router.HandleFunc("POST /api/v1/invitations/{token}/accept", h.Accept)
}

// CreateRequest represents a create invitation request
type CreateRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// InvitationDTO is a data transfer object for invitations
type InvitationDTO struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	ExpiresAt string  `json:"expires_at"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}

// CreateResponse represents a create invitation response
type CreateResponse struct {
	Invitation *InvitationDTO `json:"invitation"`
	Token      string         `json:"token,omitempty"` // Only shown once
}

// Create handles POST /api/v1/invitations
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	if req.Email == "" {
		api.ValidationError(w, map[string]string{
			"email": "Email is required",
		})
		return
	}

	if req.Role == "" {
		req.Role = "member" // Default role
	}

	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	userID, err := uuid.Parse(api.GetUserID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	result, err := h.service.Create(r.Context(), &CreateInvitationInput{
		TenantID:  tenantID,
		Email:     req.Email,
		Role:      req.Role,
		InvitedBy: userID,
	})

	if err != nil {
		h.handleCreateError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusCreated, CreateResponse{
		Invitation: toInvitationDTO(result.Invitation),
		Token:      result.Token,
	})
}

func (h *Handler) handleCreateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrCannotInviteOwner):
		api.ValidationError(w, map[string]string{
			"role": "Cannot invite users as owner",
		})
	case errors.Is(err, ErrEmailAlreadyInTenant):
		api.Conflict(w, "Email is already a member of this organization")
	case errors.Is(err, ErrPendingInvitationExists):
		api.Conflict(w, "A pending invitation already exists for this email")
	case errors.Is(err, user.ErrInvalidRole):
		api.ValidationError(w, map[string]string{
			"role": "Invalid role. Must be admin, member, or viewer",
		})
	default:
		h.logger.Error("failed to create invitation", "error", err)
		api.InternalError(w)
	}
}

// List handles GET /api/v1/invitations
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(api.GetTenantID(r.Context()))
	if err != nil {
		api.InternalError(w)
		return
	}

	invitations, err := h.service.ListByTenant(r.Context(), tenantID)
	if err != nil {
		h.logger.Error("failed to list invitations", "error", err)
		api.InternalError(w)
		return
	}

	dtos := make([]*InvitationDTO, len(invitations))
	for i, inv := range invitations {
		dtos[i] = toInvitationDTO(inv)
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"invitations": dtos,
	})
}

// Delete handles DELETE /api/v1/invitations/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.BadRequest(w, "Invalid invitation ID")
		return
	}

	// Verify invitation belongs to tenant
	invitation, err := h.service.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrInvitationNotFound) {
			api.NotFound(w, "Invitation not found")
			return
		}
		h.logger.Error("failed to get invitation", "error", err)
		api.InternalError(w)
		return
	}

	tenantID, _ := uuid.Parse(api.GetTenantID(r.Context()))
	if invitation.TenantID != tenantID {
		api.NotFound(w, "Invitation not found")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrInvitationNotFound) {
			api.NotFound(w, "Invitation not found")
			return
		}
		h.logger.Error("failed to delete invitation", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "Invitation cancelled",
	})
}

// ValidateResponse represents a validate invitation response
type ValidateResponse struct {
	Valid      bool   `json:"valid"`
	Email      string `json:"email,omitempty"`
	Role       string `json:"role,omitempty"`
	TenantName string `json:"tenant_name,omitempty"`
	ExpiresAt  string `json:"expires_at,omitempty"`
	Error      string `json:"error,omitempty"`
}

// Validate handles GET /api/v1/invitations/validate/{token}
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		api.JSONResponse(w, http.StatusOK, ValidateResponse{
			Valid: false,
			Error: "Missing token",
		})
		return
	}

	invitation, err := h.service.GetByToken(r.Context(), token)
	if err != nil {
		response := ValidateResponse{Valid: false}
		switch {
		case errors.Is(err, ErrInvitationNotFound):
			response.Error = "Invitation not found"
		case errors.Is(err, ErrInvitationExpired):
			response.Error = "Invitation has expired"
		case errors.Is(err, ErrInvitationUsed):
			response.Error = "Invitation has already been used"
		default:
			response.Error = "Invalid invitation"
		}
		api.JSONResponse(w, http.StatusOK, response)
		return
	}

	// Get tenant name
	t, err := h.tenantService.GetByID(r.Context(), invitation.TenantID)
	tenantName := ""
	if err == nil {
		tenantName = t.Name
	}

	api.JSONResponse(w, http.StatusOK, ValidateResponse{
		Valid:      true,
		Email:      invitation.Email,
		Role:       invitation.Role,
		TenantName: tenantName,
		ExpiresAt:  invitation.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// AcceptRequest represents an accept invitation request
type AcceptRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// AcceptResponse represents an accept invitation response
type AcceptResponse struct {
	User         *UserDTO `json:"user"`
	TenantID     string   `json:"tenant_id"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	TokenType    string   `json:"token_type"`
	ExpiresIn    int      `json:"expires_in"`
}

// UserDTO is a data transfer object for users
type UserDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// Accept handles POST /api/v1/invitations/{token}/accept
func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		api.BadRequest(w, "Missing token")
		return
	}

	var req AcceptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	if req.Name == "" || req.Password == "" {
		api.ValidationError(w, map[string]string{
			"error": "Name and password are required",
		})
		return
	}

	u, err := h.service.Accept(r.Context(), token, req.Name, req.Password)
	if err != nil {
		h.handleAcceptError(w, err)
		return
	}

	// Generate tokens (Email intentionally excluded from JWT per FR-104)
	tokens, err := h.jwtManager.GenerateTokenPair(&auth.UserInfo{
		UserID:   u.ID.String(),
		TenantID: u.TenantID.String(),
		Role:     string(u.Role),
	})

	if err != nil {
		h.logger.Error("failed to generate tokens", "error", err)
		api.InternalError(w)
		return
	}

	// Create session
	_, err = h.sessionMgr.CreateSession(
		r.Context(),
		u.ID,
		tokens.RefreshToken,
		r.UserAgent(),
		getClientIP(r),
	)

	if err != nil {
		h.logger.Error("failed to create session", "error", err)
	}

	api.JSONResponse(w, http.StatusOK, AcceptResponse{
		User: &UserDTO{
			ID:    u.ID.String(),
			Email: u.Email,
			Name:  u.Name,
			Role:  string(u.Role),
		},
		TenantID:     u.TenantID.String(),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
	})
}

func (h *Handler) handleAcceptError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvitationNotFound):
		api.NotFound(w, "Invitation not found")
	case errors.Is(err, ErrInvitationExpired):
		api.JSONError(w, http.StatusGone, "Invitation has expired", "INVITATION_EXPIRED")
	case errors.Is(err, ErrInvitationUsed):
		api.Conflict(w, "Invitation has already been used")
	case errors.Is(err, crypto.ErrPasswordTooShort):
		api.ValidationError(w, map[string]string{
			"password": "Password must be at least 12 characters",
		})
	case errors.Is(err, user.ErrUserEmailExists):
		api.Conflict(w, "An account with this email already exists")
	default:
		h.logger.Error("failed to accept invitation", "error", err)
		api.InternalError(w)
	}
}

func toInvitationDTO(inv *Invitation) *InvitationDTO {
	status := "pending"
	if inv.AcceptedAt != nil {
		status = "accepted"
	} else if time.Now().After(inv.ExpiresAt) {
		status = "expired"
	}

	return &InvitationDTO{
		ID:        inv.ID.String(),
		Email:     inv.Email,
		Role:      inv.Role,
		ExpiresAt: inv.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		Status:    status,
		CreatedAt: inv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	return r.RemoteAddr
}
