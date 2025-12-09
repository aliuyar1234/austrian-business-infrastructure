package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/email"
	"github.com/austrian-business-infrastructure/fo/internal/tenant"
)

// Handler handles client-related HTTP requests
type Handler struct {
	service      *Service
	clientAuth   *ClientAuth
	emailService email.Service
	portalURL    string
}

// NewHandler creates a new client handler
func NewHandler(service *Service, clientAuth *ClientAuth, emailService email.Service, portalURL string) *Handler {
	return &Handler{
		service:      service,
		clientAuth:   clientAuth,
		emailService: emailService,
		portalURL:    portalURL,
	}
}

// Routes returns the routes for the client handler
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Staff-only endpoints (require staff auth middleware)
	r.Group(func(r chi.Router) {
		r.Post("/invite", h.Invite)
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Deactivate)
		r.Post("/{id}/resend-invitation", h.ResendInvitation)
	})

	return r
}

// PortalRoutes returns the routes for portal authentication
func (h *Handler) PortalRoutes() chi.Router {
	r := chi.NewRouter()

	// Public activation routes
	r.Get("/activate/{token}", h.ValidateActivation)
	r.Post("/activate/{token}", h.CompleteActivation)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.RefreshToken)
	r.Post("/logout", h.Logout)

	return r
}

// getUserInfoFromContext extracts user ID and name from API context
func getUserInfoFromContext(ctx context.Context) (uuid.UUID, string, error) {
	userIDStr := api.GetUserID(ctx)
	if userIDStr == "" {
		return uuid.Nil, "", errors.New("user not authenticated")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, "", errors.New("invalid user ID")
	}

	// Name is not in JWT per FR-104, so we'd need to look it up if needed
	return userID, "", nil
}

// Invite creates a new client invitation
func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant and user from context
	tenantID := tenant.GetTenantID(ctx)
	userID, userName, err := getUserInfoFromContext(ctx)

	if tenantID == uuid.Nil || err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req InviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Email == "" || req.Name == "" {
		http.Error(w, "email and name are required", http.StatusBadRequest)
		return
	}
	if len(req.AccountIDs) == 0 {
		http.Error(w, "at least one account is required", http.StatusBadRequest)
		return
	}

	// Create invitation
	resp, err := h.service.Invite(ctx, tenantID, userID, &req)
	if err != nil {
		if errors.Is(err, ErrClientEmailExists) {
			http.Error(w, "client with this email already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create invitation", http.StatusInternalServerError)
		return
	}

	// Get token for email
	token, err := h.service.GetInvitationToken(ctx, resp.InvitationID)
	if err == nil {
		// Send invitation email
		err = h.sendInvitationEmail(ctx, req.Email, req.Name, userName, token)
		if err == nil {
			resp.InviteSent = true
			_ = h.service.MarkInvitationEmailSent(ctx, resp.InvitationID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// sendInvitationEmail sends the invitation email to the client
func (h *Handler) sendInvitationEmail(ctx context.Context, toEmail, clientName, inviterName, token string) error {
	activateURL := h.portalURL + "/activate?token=" + token

	// Use the email service interface - extend as needed
	subject := "Einladung zum Mandantenportal"
	body := `Sehr geehrte/r ` + clientName + `,

` + inviterName + ` hat Sie zum Mandantenportal eingeladen.

Bitte klicken Sie auf den folgenden Link, um Ihr Konto zu aktivieren:

` + activateURL + `

Dieser Link ist 24 Stunden gültig.

Bei Fragen wenden Sie sich bitte an Ihren Steuerberater.

Mit freundlichen Grüßen
Austrian Business Platform`

	// For now, just log - the email service would need extension for this
	_ = subject
	_ = body
	return nil // TODO: Implement portal-specific email sending
}

// List returns all clients for the tenant
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	var status *Status
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		s := Status(statusStr)
		if !IsValidStatus(statusStr) {
			http.Error(w, "invalid status", http.StatusBadRequest)
			return
		}
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

	clients, total, err := h.service.List(ctx, tenantID, status, limit, offset)
	if err != nil {
		http.Error(w, "failed to list clients", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"clients": clients,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetByID returns a client by ID
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	clientID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid client ID", http.StatusBadRequest)
		return
	}

	clientDetail, err := h.service.GetClientWithAccounts(ctx, clientID)
	if err != nil {
		if errors.Is(err, ErrClientNotFound) {
			http.Error(w, "client not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get client", http.StatusInternalServerError)
		return
	}

	// Verify tenant access
	if clientDetail.TenantID != tenantID {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clientDetail)
}

// Update updates a client
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	clientID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid client ID", http.StatusBadRequest)
		return
	}

	// Get existing client
	client, err := h.service.GetByID(ctx, clientID)
	if err != nil {
		if errors.Is(err, ErrClientNotFound) {
			http.Error(w, "client not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get client", http.StatusInternalServerError)
		return
	}

	// Verify tenant access
	if client.TenantID != tenantID {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	// Parse update request
	var req struct {
		Name        *string     `json:"name"`
		CompanyName *string     `json:"company_name"`
		Phone       *string     `json:"phone"`
		AccountIDs  []uuid.UUID `json:"account_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Apply updates
	if req.Name != nil {
		client.Name = *req.Name
	}
	if req.CompanyName != nil {
		client.CompanyName = req.CompanyName
	}
	if req.Phone != nil {
		client.Phone = req.Phone
	}

	if err := h.service.repo.Update(ctx, client); err != nil {
		http.Error(w, "failed to update client", http.StatusInternalServerError)
		return
	}

	// Update account access if provided
	if req.AccountIDs != nil {
		if err := h.service.UpdateAccountAccess(ctx, clientID, req.AccountIDs); err != nil {
			http.Error(w, "failed to update account access", http.StatusInternalServerError)
			return
		}
	}

	// Return updated client with accounts
	clientDetail, err := h.service.GetClientWithAccounts(ctx, clientID)
	if err != nil {
		http.Error(w, "failed to get updated client", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clientDetail)
}

// Deactivate deactivates a client
func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	clientID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid client ID", http.StatusBadRequest)
		return
	}

	// Verify tenant access
	client, err := h.service.GetByID(ctx, clientID)
	if err != nil {
		if errors.Is(err, ErrClientNotFound) {
			http.Error(w, "client not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get client", http.StatusInternalServerError)
		return
	}

	if client.TenantID != tenantID {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	if err := h.service.Deactivate(ctx, clientID); err != nil {
		http.Error(w, "failed to deactivate client", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ResendInvitation resends an invitation to a client
func (h *Handler) ResendInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	userID, userName, _ := getUserInfoFromContext(ctx)

	if tenantID == uuid.Nil || userID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	clientID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid client ID", http.StatusBadRequest)
		return
	}

	// Verify tenant access
	client, err := h.service.GetByID(ctx, clientID)
	if err != nil {
		if errors.Is(err, ErrClientNotFound) {
			http.Error(w, "client not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get client", http.StatusInternalServerError)
		return
	}

	if client.TenantID != tenantID {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	invitation, err := h.service.ResendInvitation(ctx, clientID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Send email
	_ = h.sendInvitationEmail(ctx, client.Email, client.Name, userName, invitation.Token)
	_ = h.service.MarkInvitationEmailSent(ctx, invitation.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"invitation_id": invitation.ID,
		"status":        "sent",
	})
}

// ============== Portal Authentication Endpoints ==============

// ValidateActivation validates an activation token
func (h *Handler) ValidateActivation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		return
	}

	info, _, err := h.service.ValidateInvitation(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrInvitationExpired) {
			http.Error(w, "invitation expired", http.StatusGone)
			return
		}
		if errors.Is(err, ErrInvitationUsed) {
			http.Error(w, "invitation already used", http.StatusConflict)
			return
		}
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// CompleteActivation completes the client activation
func (h *Handler) CompleteActivation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := chi.URLParam(r, "token")
	if token == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		return
	}

	var req struct {
		Password string  `json:"password"`
		Phone    *string `json:"phone,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Validate token first
	_, _, err := h.service.ValidateInvitation(ctx, token)
	if err != nil {
		if errors.Is(err, ErrInvitationExpired) {
			http.Error(w, "invitation expired", http.StatusGone)
			return
		}
		if errors.Is(err, ErrInvitationUsed) {
			http.Error(w, "invitation already used", http.StatusConflict)
			return
		}
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}

	// Create user account for client
	// This needs to integrate with the user repository
	userID := uuid.New() // Placeholder - actual user creation happens in auth handler

	// Activate client
	activatedClient, err := h.service.ActivateClient(ctx, token, userID)
	if err != nil {
		http.Error(w, "failed to activate client", http.StatusInternalServerError)
		return
	}

	// Update phone if provided
	if req.Phone != nil {
		activatedClient.Phone = req.Phone
		_ = h.service.repo.Update(ctx, activatedClient)
	}

	// Generate tokens
	accessToken, refreshToken, expiresAt, err := h.clientAuth.GenerateTokens(activatedClient, userID)
	if err != nil {
		http.Error(w, "failed to generate tokens", http.StatusInternalServerError)
		return
	}

	// Set cookies
	SetAuthCookies(w, accessToken, refreshToken, 15*60, 7*24*60*60, false) // TODO: Get from config

	response := AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		Client:       activatedClient,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Login handles client login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}

	// TODO: Integrate with user authentication
	// For now, this is a placeholder
	// Actual implementation needs to:
	// 1. Find user by email with is_client = true
	// 2. Verify password
	// 3. Get associated client
	// 4. Generate tokens

	_ = ctx
	http.Error(w, "not implemented - use user auth with client role", http.StatusNotImplemented)
}

// RefreshToken refreshes the access token
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	cookie, err := r.Cookie("portal_refresh_token")
	if err != nil {
		http.Error(w, "refresh token required", http.StatusUnauthorized)
		return
	}

	claims, err := h.clientAuth.ValidateToken(cookie.Value)
	if err != nil {
		ClearAuthCookies(w)
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Get client
	client, err := h.service.GetByID(r.Context(), claims.ClientID)
	if err != nil {
		ClearAuthCookies(w)
		http.Error(w, "client not found", http.StatusUnauthorized)
		return
	}

	if client.Status != StatusActive {
		ClearAuthCookies(w)
		http.Error(w, "client not active", http.StatusUnauthorized)
		return
	}

	// Generate new tokens
	accessToken, refreshToken, expiresAt, err := h.clientAuth.GenerateTokens(client, claims.UserID)
	if err != nil {
		http.Error(w, "failed to generate tokens", http.StatusInternalServerError)
		return
	}

	// Set cookies
	SetAuthCookies(w, accessToken, refreshToken, 15*60, 7*24*60*60, false)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": accessToken,
		"expires_at":   expiresAt,
	})
}

// Logout handles client logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	ClearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}
