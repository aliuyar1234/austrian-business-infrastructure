package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/tenant"
	"github.com/austrian-business-infrastructure/fo/internal/user"
	"github.com/austrian-business-infrastructure/fo/pkg/cache"
)

// OAuthHandler handles OAuth authentication HTTP requests
type OAuthHandler struct {
	oauthManager   *OAuthManager
	tenantService  *tenant.Service
	userService    *user.Service
	sessionManager *SessionManager
	jwtManager     *JWTManager
	redis          *cache.Client
	logger         *slog.Logger
	appURL         string
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(
	oauthManager *OAuthManager,
	tenantService *tenant.Service,
	userService *user.Service,
	sessionManager *SessionManager,
	jwtManager *JWTManager,
	redis *cache.Client,
	logger *slog.Logger,
	appURL string,
) *OAuthHandler {
	return &OAuthHandler{
		oauthManager:   oauthManager,
		tenantService:  tenantService,
		userService:    userService,
		sessionManager: sessionManager,
		jwtManager:     jwtManager,
		redis:          redis,
		logger:         logger,
		appURL:         appURL,
	}
}

// RegisterRoutes registers OAuth routes
func (h *OAuthHandler) RegisterRoutes(router *api.Router) {
	router.HandleFunc("GET /api/v1/auth/oauth/{provider}", h.StartOAuth)
	router.HandleFunc("GET /api/v1/auth/oauth/{provider}/callback", h.OAuthCallback)
}

// StartOAuthRequest is the request to start OAuth flow
type StartOAuthRequest struct {
	TenantSlug string `json:"tenant_slug,omitempty"` // Optional - for joining existing tenant
}

// StartOAuth handles GET /api/v1/auth/oauth/{provider}
func (h *OAuthHandler) StartOAuth(w http.ResponseWriter, r *http.Request) {
	providerStr := r.PathValue("provider")
	provider, err := ValidateProvider(providerStr)
	if err != nil {
		api.BadRequest(w, "Invalid OAuth provider")
		return
	}

	if !h.oauthManager.IsEnabled() {
		api.JSONError(w, http.StatusServiceUnavailable, "OAuth is not enabled", "OAUTH_DISABLED")
		return
	}

	if !h.oauthManager.IsProviderConfigured(provider) {
		api.JSONError(w, http.StatusServiceUnavailable, "OAuth provider not configured", "OAUTH_PROVIDER_NOT_CONFIGURED")
		return
	}

	// Generate state for CSRF protection
	state, err := generateOAuthState()
	if err != nil {
		h.logger.Error("failed to generate OAuth state", "error", err)
		api.InternalError(w)
		return
	}

	// Store state in Redis with tenant slug if provided
	stateData := map[string]string{
		"state": state,
	}
	if tenantSlug := r.URL.Query().Get("tenant_slug"); tenantSlug != "" {
		stateData["tenant_slug"] = tenantSlug
	}

	stateJSON, _ := json.Marshal(stateData)
	if err := h.redis.Set(r.Context(), "oauth_state:"+state, string(stateJSON), 10*time.Minute).Err(); err != nil {
		h.logger.Error("failed to store OAuth state", "error", err)
		api.InternalError(w)
		return
	}

	// Get authorization URL
	authURL, err := h.oauthManager.GetAuthURL(provider, state)
	if err != nil {
		h.logger.Error("failed to get OAuth URL", "error", err)
		api.InternalError(w)
		return
	}

	// Redirect to OAuth provider
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// OAuthCallback handles GET /api/v1/auth/oauth/{provider}/callback
func (h *OAuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	providerStr := r.PathValue("provider")
	provider, err := ValidateProvider(providerStr)
	if err != nil {
		h.redirectWithError(w, r, "Invalid OAuth provider")
		return
	}

	// Check for error from provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		h.logger.Warn("OAuth error from provider", "error", errParam, "description", errDesc)
		h.redirectWithError(w, r, "Authentication was cancelled or failed")
		return
	}

	// Validate state
	state := r.URL.Query().Get("state")
	if state == "" {
		h.redirectWithError(w, r, "Missing state parameter")
		return
	}

	stateJSON, err := h.redis.Get(r.Context(), "oauth_state:"+state).Result()
	if err != nil {
		h.redirectWithError(w, r, "Invalid or expired state")
		return
	}

	// Delete state to prevent replay
	h.redis.Del(r.Context(), "oauth_state:"+state)

	var stateData map[string]string
	if err := json.Unmarshal([]byte(stateJSON), &stateData); err != nil {
		h.redirectWithError(w, r, "Invalid state data")
		return
	}

	// Get code
	code := r.URL.Query().Get("code")
	if code == "" {
		h.redirectWithError(w, r, "Missing authorization code")
		return
	}

	// Exchange code for user info
	userInfo, err := h.oauthManager.Exchange(r.Context(), provider, code)
	if err != nil {
		h.logger.Error("OAuth exchange failed", "error", err)
		h.redirectWithError(w, r, "Failed to authenticate with provider")
		return
	}

	// Find or create user
	u, tenantID, err := h.findOrCreateOAuthUser(r, userInfo, stateData["tenant_slug"])
	if err != nil {
		h.logger.Error("failed to process OAuth user", "error", err)
		h.redirectWithError(w, r, err.Error())
		return
	}

	// Generate tokens (Email intentionally excluded from JWT per FR-104)
	tokens, err := h.jwtManager.GenerateTokenPair(&UserInfo{
		UserID:   u.ID.String(),
		TenantID: tenantID,
		Role:     string(u.Role),
	})

	if err != nil {
		h.logger.Error("failed to generate tokens", "error", err)
		h.redirectWithError(w, r, "Failed to create session")
		return
	}

	// Create session
	_, err = h.sessionManager.CreateSession(
		r.Context(),
		u.ID,
		tokens.RefreshToken,
		r.UserAgent(),
		getClientIP(r),
	)

	if err != nil {
		h.logger.Error("failed to create session", "error", err)
	}

	// Redirect to frontend with tokens
	// In production, you'd set cookies or use a one-time code
	redirectURL := h.appURL + "/auth/callback?access_token=" + tokens.AccessToken + "&token_type=Bearer"
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (h *OAuthHandler) findOrCreateOAuthUser(r *http.Request, info *OAuthUserInfo, tenantSlug string) (*user.User, string, error) {
	ctx := r.Context()

	// Try to find existing OAuth user
	existingUser, err := h.userService.GetByOAuth(ctx, string(info.Provider), info.ID)
	if err == nil {
		return existingUser, existingUser.TenantID.String(), nil
	}

	if !errors.Is(err, user.ErrUserNotFound) {
		return nil, "", err
	}

	// User doesn't exist - need to create
	// If tenant slug provided, add to existing tenant
	if tenantSlug != "" {
		t, err := h.tenantService.GetBySlug(ctx, tenantSlug)
		if err != nil {
			return nil, "", errors.New("tenant not found")
		}

		// Check if email already exists in tenant
		existingByEmail, err := h.userService.GetByEmail(ctx, t.ID, info.Email)
		if err == nil {
			// Link OAuth to existing account
			// In a full implementation, you'd update the user with OAuth credentials
			return existingByEmail, t.ID.String(), nil
		}

		// Create new user in existing tenant
		provider := string(info.Provider)
		avatarURL := info.AvatarURL
		newUser, err := h.userService.CreateOAuthUser(ctx, t.ID, info.Email, info.Name, provider, info.ID, &avatarURL)
		if err != nil {
			return nil, "", err
		}

		return newUser, t.ID.String(), nil
	}

	// No tenant slug - create new tenant
	// Generate a slug from email
	slug := generateSlugFromEmail(info.Email)

	result, err := h.tenantService.CreateWithOwner(ctx, &tenant.CreateTenantInput{
		TenantName: info.Name + "'s Organization",
		TenantSlug: slug,
		OwnerName:  info.Name,
		OwnerEmail: info.Email,
		Password:   generateRandomPassword(), // Won't be used for OAuth users
	})

	if err != nil {
		return nil, "", err
	}

	// Update user with OAuth info (the create above created with password)
	// In a real implementation, CreateWithOwner would have an OAuth variant

	return result.Owner, result.Tenant.ID.String(), nil
}

func (h *OAuthHandler) redirectWithError(w http.ResponseWriter, r *http.Request, message string) {
	redirectURL := h.appURL + "/auth/error?message=" + message
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func generateOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateSlugFromEmail(email string) string {
	// Extract username part and sanitize
	slug := ""
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			break
		}
		c := email[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			slug += string(c)
		} else if c >= 'A' && c <= 'Z' {
			slug += string(c + 32) // lowercase
		}
	}

	if len(slug) == 0 {
		slug = "org"
	}

	// Add random suffix to ensure uniqueness
	b := make([]byte, 4)
	rand.Read(b)
	slug += "-" + base64.RawURLEncoding.EncodeToString(b)[:6]

	return slug
}

func generateRandomPassword() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
