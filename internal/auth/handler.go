package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"austrian-business-infrastructure/internal/api"
	"austrian-business-infrastructure/internal/audit"
	"austrian-business-infrastructure/internal/crypto"
	"austrian-business-infrastructure/internal/tenant"
	"austrian-business-infrastructure/internal/user"
	"austrian-business-infrastructure/pkg/cache"
	"github.com/google/uuid"
)

// Handler handles authentication HTTP requests
type Handler struct {
	tenantService  *tenant.Service
	userService    *user.Service
	sessionManager *SessionManager
	jwtManager     *JWTManager
	rateLimiter    *RateLimiter
	auditLogger    *audit.Logger
	redis          *cache.Client
	logger         *slog.Logger
	cookieConfig   *CookieConfig
	trustedProxies map[string]bool // Trusted proxy IPs/CIDRs for X-Forwarded-For
}

// NewHandler creates a new auth handler
func NewHandler(
	tenantService *tenant.Service,
	userService *user.Service,
	sessionManager *SessionManager,
	jwtManager *JWTManager,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		tenantService:  tenantService,
		userService:    userService,
		sessionManager: sessionManager,
		jwtManager:     jwtManager,
		logger:         logger,
		cookieConfig:   DefaultCookieConfig(),
	}
}

// NewHandlerWithSecurity creates a new auth handler with security features
func NewHandlerWithSecurity(
	tenantService *tenant.Service,
	userService *user.Service,
	sessionManager *SessionManager,
	jwtManager *JWTManager,
	rateLimiter *RateLimiter,
	auditLogger *audit.Logger,
	redis *cache.Client,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		tenantService:  tenantService,
		userService:    userService,
		sessionManager: sessionManager,
		jwtManager:     jwtManager,
		rateLimiter:    rateLimiter,
		auditLogger:    auditLogger,
		redis:          redis,
		logger:         logger,
		cookieConfig:   DefaultCookieConfig(),
	}
}

// RegisterRoutes registers auth routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth func(http.Handler) http.Handler) {
	router.HandleFunc("POST /api/v1/auth/register", h.Register)
	router.HandleFunc("POST /api/v1/auth/login", h.Login)
	router.HandleFunc("POST /api/v1/auth/login/2fa", h.Login2FA)
	router.HandleFunc("POST /api/v1/auth/refresh", h.Refresh)
	router.HandleFunc("POST /api/v1/auth/logout", h.Logout)
	router.Handle("GET /api/v1/auth/me", requireAuth(http.HandlerFunc(h.Me)))

	// Password reset endpoints (public)
	router.HandleFunc("POST /api/v1/auth/forgot-password", h.ForgotPassword)
	router.HandleFunc("POST /api/v1/auth/reset-password", h.ResetPassword)

	// Profile and password change endpoints (authenticated)
	router.Handle("PATCH /api/v1/auth/profile", requireAuth(http.HandlerFunc(h.UpdateProfile)))
	router.Handle("POST /api/v1/auth/change-password", requireAuth(http.HandlerFunc(h.ChangePassword)))
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
}

// RegisterResponse represents a registration response
type RegisterResponse struct {
	Tenant      *TenantDTO `json:"tenant"`
	User        *UserDTO   `json:"user"`
	AccessToken string     `json:"access_token"`
	TokenType   string     `json:"token_type"`
	ExpiresIn   int        `json:"expires_in"`
}

// TenantDTO is a data transfer object for tenants
type TenantDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// UserDTO is a data transfer object for users
type UserDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// Register handles POST /api/v1/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	// Validate required fields
	if req.TenantName == "" || req.TenantSlug == "" || req.Name == "" || req.Email == "" || req.Password == "" {
		api.ValidationError(w, map[string]string{
			"error": "All fields are required: tenant_name, tenant_slug, name, email, password",
		})
		return
	}

	// Create tenant with owner
	result, err := h.tenantService.CreateWithOwner(r.Context(), &tenant.CreateTenantInput{
		TenantName: req.TenantName,
		TenantSlug: req.TenantSlug,
		OwnerName:  req.Name,
		OwnerEmail: req.Email,
		Password:   req.Password,
	})

	if err != nil {
		h.handleRegistrationError(w, err)
		return
	}

	// Generate tokens (Email intentionally excluded from JWT per FR-104)
	tokens, err := h.jwtManager.GenerateTokenPair(&UserInfo{
		UserID:   result.Owner.ID.String(),
		TenantID: result.Tenant.ID.String(),
		Role:     string(result.Owner.Role),
	})

	if err != nil {
		h.logger.Error("failed to generate tokens", "error", err)
		api.InternalError(w)
		return
	}

	// Create session
	_, err = h.sessionManager.CreateSession(
		r.Context(),
		result.Owner.ID,
		tokens.RefreshToken,
		r.UserAgent(),
		h.getClientIP(r),
	)

	if err != nil {
		h.logger.Error("failed to create session", "error", err)
		// Continue - tokens are still valid
	}

	// Set refresh token as httpOnly cookie (SECURITY: not accessible via JavaScript)
	refreshExpiry := time.Now().Add(h.jwtManager.config.RefreshTokenExpiry)
	SetRefreshTokenCookie(w, tokens.RefreshToken, refreshExpiry, h.cookieConfig)

	api.JSONResponse(w, http.StatusCreated, RegisterResponse{
		Tenant: &TenantDTO{
			ID:   result.Tenant.ID.String(),
			Name: result.Tenant.Name,
			Slug: result.Tenant.Slug,
		},
		User: &UserDTO{
			ID:    result.Owner.ID.String(),
			Email: result.Owner.Email,
			Name:  result.Owner.Name,
			Role:  string(result.Owner.Role),
		},
		AccessToken: tokens.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900, // 15 minutes
		// NOTE: refresh_token intentionally NOT in response body - it's in httpOnly cookie
	})
}

func (h *Handler) handleRegistrationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, tenant.ErrTenantSlugExists):
		api.Conflict(w, "Tenant slug already exists")
	case errors.Is(err, tenant.ErrInvalidTenantSlug):
		api.ValidationError(w, map[string]string{
			"tenant_slug": "Invalid slug format. Use lowercase letters, numbers, and hyphens only.",
		})
	case errors.Is(err, ErrPasswordTooShort):
		api.ValidationError(w, map[string]string{
			"password": "Password must be at least 12 characters",
		})
	case errors.Is(err, ErrPasswordNoUppercase):
		api.ValidationError(w, map[string]string{
			"password": "Password must contain at least one uppercase letter",
		})
	case errors.Is(err, ErrPasswordNoLowercase):
		api.ValidationError(w, map[string]string{
			"password": "Password must contain at least one lowercase letter",
		})
	case errors.Is(err, ErrPasswordNoDigit):
		api.ValidationError(w, map[string]string{
			"password": "Password must contain at least one digit",
		})
	case errors.Is(err, user.ErrInvalidEmail):
		api.ValidationError(w, map[string]string{
			"email": "Invalid email format",
		})
	default:
		h.logger.Error("registration failed", "error", err)
		api.InternalError(w)
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a login response (when 2FA is not required)
// NOTE: refresh_token is sent as httpOnly cookie, not in response body
type LoginResponse struct {
	User        *UserDTO `json:"user"`
	TenantID    string   `json:"tenant_id"`
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int      `json:"expires_in"`
}

// Login2FARequiredResponse is returned when 2FA verification is needed
type Login2FARequiredResponse struct {
	RequiresTwoFactor bool   `json:"requires_2fa"`
	ChallengeToken    string `json:"challenge_token"`
	ExpiresIn         int    `json:"expires_in"`
}

// Login2FARequest represents a 2FA verification request
type Login2FARequest struct {
	ChallengeToken string `json:"challenge_token"`
	TOTPCode       string `json:"totp_code"`
}

// Login handles POST /api/v1/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clientIP := h.getClientIP(r)

	// Rate limiting (FR-106: 10/min per IP)
	// Login uses fail-closed: reject requests when rate limiter backend unavailable
	if h.rateLimiter != nil {
		if err := h.rateLimiter.CheckLogin(ctx, clientIP); err != nil {
			if errors.Is(err, ErrRateLimited) {
				w.Header().Set("Retry-After", "60")
				api.JSONError(w, http.StatusTooManyRequests, "Too many login attempts", "RATE_LIMITED")
				return
			}
			// Fail-closed for login: reject on backend errors to prevent brute-force during outages
			h.logger.Error("rate limit check failed, rejecting login", "error", err, "ip", clientIP)
			api.JSONError(w, http.StatusServiceUnavailable, "Service temporarily unavailable", "SERVICE_UNAVAILABLE")
			return
		}
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		api.ValidationError(w, map[string]string{
			"error": "Email and password are required",
		})
		return
	}

	// Authenticate user
	u, err := h.userService.Authenticate(ctx, req.Email, req.Password)
	if err != nil {
		// Audit log failed login
		h.logAuthEvent(ctx, audit.EventLoginFailed, nil, nil, clientIP, r.UserAgent(), map[string]any{
			"reason": "invalid_credentials",
		})
		switch {
		case errors.Is(err, ErrPasswordInvalid), errors.Is(err, user.ErrUserNotFound):
			api.JSONError(w, http.StatusUnauthorized, "Invalid email or password", api.ErrCodeInvalidCredentials)
		case errors.Is(err, user.ErrUserInactive):
			api.JSONError(w, http.StatusUnauthorized, "Account is inactive", api.ErrCodeUnauthorized)
		default:
			h.logger.Error("login failed", "error", err)
			api.InternalError(w)
		}
		return
	}

	// Check if 2FA is enabled - return challenge token instead of tokens
	if u.TOTPEnabled {
		challengeToken, err := h.create2FAChallenge(ctx, u)
		if err != nil {
			h.logger.Error("failed to create 2FA challenge", "error", err)
			api.InternalError(w)
			return
		}

		api.JSONResponse(w, http.StatusOK, Login2FARequiredResponse{
			RequiresTwoFactor: true,
			ChallengeToken:    challengeToken,
			ExpiresIn:         300, // 5 minutes
		})
		return
	}

	// No 2FA - generate tokens directly
	h.completeLogin(w, r, u, clientIP)
}

// Login2FA handles POST /api/v1/auth/login/2fa
func (h *Handler) Login2FA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clientIP := h.getClientIP(r)

	var req Login2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	if req.ChallengeToken == "" || req.TOTPCode == "" {
		api.ValidationError(w, map[string]string{
			"error": "Challenge token and TOTP code are required",
		})
		return
	}

	// Validate challenge token and get user
	u, err := h.validate2FAChallenge(ctx, req.ChallengeToken)
	if err != nil {
		h.logAuthEvent(ctx, audit.EventLoginFailed, nil, nil, clientIP, r.UserAgent(), map[string]any{
			"reason": "invalid_2fa_challenge",
		})
		api.JSONError(w, http.StatusUnauthorized, "Invalid or expired challenge", api.ErrCodeInvalidToken)
		return
	}

	// Verify TOTP code (will be implemented in Phase 4)
	// For now, we'll check using a placeholder that Phase 4 will implement
	if !h.verifyTOTP(ctx, u, req.TOTPCode) {
		h.logAuthEvent(ctx, audit.EventLoginFailed, &u.ID, &u.TenantID, clientIP, r.UserAgent(), map[string]any{
			"reason": "invalid_totp_code",
		})
		api.JSONError(w, http.StatusUnauthorized, "Invalid TOTP code", api.ErrCodeInvalidCredentials)
		return
	}

	// Delete challenge token (one-time use)
	h.delete2FAChallenge(ctx, req.ChallengeToken)

	// Complete login
	h.completeLogin(w, r, u, clientIP)
}

// completeLogin generates tokens and creates session
func (h *Handler) completeLogin(w http.ResponseWriter, r *http.Request, u *user.User, clientIP string) {
	ctx := r.Context()

	// Generate tokens (Email intentionally excluded from JWT per FR-104)
	tokens, err := h.jwtManager.GenerateTokenPair(&UserInfo{
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
	_, err = h.sessionManager.CreateSession(
		ctx,
		u.ID,
		tokens.RefreshToken,
		r.UserAgent(),
		clientIP,
	)

	if err != nil {
		h.logger.Error("failed to create session", "error", err)
		// Continue - tokens are still valid
	}

	// Audit log successful login
	h.logAuthEvent(ctx, audit.EventLogin, &u.ID, &u.TenantID, clientIP, r.UserAgent(), nil)

	// Set refresh token as httpOnly cookie (SECURITY: not accessible via JavaScript)
	refreshExpiry := time.Now().Add(h.jwtManager.config.RefreshTokenExpiry)
	SetRefreshTokenCookie(w, tokens.RefreshToken, refreshExpiry, h.cookieConfig)

	api.JSONResponse(w, http.StatusOK, LoginResponse{
		User: &UserDTO{
			ID:    u.ID.String(),
			Email: u.Email,
			Name:  u.Name,
			Role:  string(u.Role),
		},
		TenantID:    u.TenantID.String(),
		AccessToken: tokens.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900,
		// NOTE: refresh_token intentionally NOT in response body - it's in httpOnly cookie
	})
}

// RefreshResponse represents a token refresh response
// NOTE: refresh_token is sent as httpOnly cookie, not in response body
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// Refresh handles POST /api/v1/auth/refresh
// Reads refresh token from httpOnly cookie (not request body)
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from httpOnly cookie
	refreshToken, err := GetRefreshTokenFromCookie(r)
	if err != nil {
		api.JSONError(w, http.StatusUnauthorized, "No refresh token", api.ErrCodeInvalidToken)
		return
	}

	// Validate the refresh token (JWT)
	claims, err := h.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		// Clear invalid cookie
		ClearRefreshTokenCookie(w, h.cookieConfig)
		switch {
		case errors.Is(err, ErrExpiredToken):
			api.JSONError(w, http.StatusUnauthorized, "Refresh token has expired", api.ErrCodeTokenExpired)
		default:
			api.JSONError(w, http.StatusUnauthorized, "Invalid refresh token", api.ErrCodeInvalidToken)
		}
		return
	}

	// Validate session exists in database
	session, err := h.sessionManager.ValidateRefreshToken(r.Context(), refreshToken)
	if err != nil {
		// Clear invalid cookie
		ClearRefreshTokenCookie(w, h.cookieConfig)
		switch {
		case errors.Is(err, ErrSessionNotFound):
			api.JSONError(w, http.StatusUnauthorized, "Session not found", api.ErrCodeInvalidToken)
		case errors.Is(err, ErrSessionExpired):
			api.JSONError(w, http.StatusUnauthorized, "Session has expired", api.ErrCodeTokenExpired)
		default:
			h.logger.Error("failed to validate session", "error", err)
			api.InternalError(w)
		}
		return
	}

	// Generate new token pair (Email intentionally excluded from JWT per FR-104)
	tokens, err := h.jwtManager.GenerateTokenPair(&UserInfo{
		UserID:   claims.UserID,
		TenantID: claims.TenantID,
		Role:     claims.Role,
	})

	if err != nil {
		h.logger.Error("failed to generate tokens", "error", err)
		api.InternalError(w)
		return
	}

	// Rotate refresh token in session (token rotation for security)
	if err := h.sessionManager.RotateRefreshToken(r.Context(), session.ID, tokens.RefreshToken); err != nil {
		h.logger.Error("failed to rotate refresh token", "error", err)
		// Continue - old token is still valid
	}

	// Set new refresh token as httpOnly cookie
	refreshExpiry := time.Now().Add(h.jwtManager.config.RefreshTokenExpiry)
	SetRefreshTokenCookie(w, tokens.RefreshToken, refreshExpiry, h.cookieConfig)

	api.JSONResponse(w, http.StatusOK, RefreshResponse{
		AccessToken: tokens.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900,
	})
}

// Logout handles POST /api/v1/auth/logout
// Reads refresh token from httpOnly cookie and clears it
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	refreshToken, err := GetRefreshTokenFromCookie(r)
	if err == nil && refreshToken != "" {
		// Delete session from database
		if err := h.sessionManager.DeleteByRefreshToken(r.Context(), refreshToken); err != nil {
			h.logger.Error("failed to delete session", "error", err)
			// Continue - best effort logout
		}
	}

	// Always clear the cookie
	ClearRefreshTokenCookie(w, h.cookieConfig)

	api.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// MeResponse represents the current user response
type MeResponse struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	TenantID   string `json:"tenantId"`
	TenantName string `json:"tenantName"`
}

// Me handles GET /api/v1/auth/me - returns current authenticated user
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by auth middleware)
	userID := api.GetUserID(r.Context())
	if userID == "" {
		api.JSONError(w, http.StatusUnauthorized, "Not authenticated", api.ErrCodeUnauthorized)
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		api.JSONError(w, http.StatusUnauthorized, "Invalid user ID", api.ErrCodeUnauthorized)
		return
	}

	// Get user from database
	u, err := h.userService.GetByID(r.Context(), userUUID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			api.JSONError(w, http.StatusUnauthorized, "User not found", api.ErrCodeUnauthorized)
			return
		}
		h.logger.Error("failed to get user", "error", err)
		api.InternalError(w)
		return
	}

	// Get tenant name
	tenantName := ""
	if h.tenantService != nil {
		t, err := h.tenantService.GetByID(r.Context(), u.TenantID)
		if err == nil && t != nil {
			tenantName = t.Name
		}
	}

	api.JSONResponse(w, http.StatusOK, MeResponse{
		ID:         u.ID.String(),
		Email:      u.Email,
		Name:       u.Name,
		Role:       string(u.Role),
		TenantID:   u.TenantID.String(),
		TenantName: tenantName,
	})
}

// getClientIP extracts client IP from request with trusted proxy validation
// This prevents IP spoofing attacks (CWE-290) by only trusting X-Forwarded-For
// from known proxy IPs (e.g., Caddy, Traefik, load balancers)
func (h *Handler) getClientIP(r *http.Request) string {
	// Extract remote IP (strip port if present)
	remoteAddr := r.RemoteAddr
	if idx := lastIndexByte(remoteAddr, ':'); idx != -1 {
		// Check if this is IPv6 [::1]:port format
		if remoteAddr[0] == '[' {
			if bracketIdx := lastIndexByte(remoteAddr, ']'); bracketIdx != -1 {
				remoteAddr = remoteAddr[1:bracketIdx]
			}
		} else {
			remoteAddr = remoteAddr[:idx]
		}
	}

	// Only trust forwarded headers if request comes from trusted proxy
	// If no trusted proxies configured, always use RemoteAddr (secure default)
	if len(h.trustedProxies) == 0 || !h.trustedProxies[remoteAddr] {
		return remoteAddr
	}

	// Request is from trusted proxy - check X-Forwarded-For
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Parse X-Forwarded-For: client, proxy1, proxy2
		// Get the rightmost non-trusted IP (real client)
		ips := splitXFF(xff)
		for i := len(ips) - 1; i >= 0; i-- {
			ip := ips[i]
			if !h.trustedProxies[ip] {
				return ip
			}
		}
	}

	// Check X-Real-IP (set by some proxies like nginx)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return trimSpace(xri)
	}

	return remoteAddr
}

// lastIndexByte returns the index of the last instance of c in s, or -1
func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// splitXFF splits X-Forwarded-For header and trims whitespace
func splitXFF(xff string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(xff); i++ {
		if i == len(xff) || xff[i] == ',' {
			ip := trimSpace(xff[start:i])
			if ip != "" {
				result = append(result, ip)
			}
			start = i + 1
		}
	}
	return result
}

// trimSpace removes leading/trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// SetTrustedProxies configures which proxy IPs are trusted for X-Forwarded-For
// Common values: "127.0.0.1", "::1", "10.0.0.0/8", Docker network IPs
func (h *Handler) SetTrustedProxies(proxies []string) {
	h.trustedProxies = make(map[string]bool)
	for _, p := range proxies {
		h.trustedProxies[p] = true
	}
}

// ============== 2FA Challenge Token Helpers ==============

const (
	challenge2FAPrefix = "2fa_challenge:"
	challenge2FATTL    = 5 * time.Minute
)

// challengeData stores the user info during 2FA verification
type challengeData struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
}

// create2FAChallenge creates a temporary challenge token for 2FA verification
func (h *Handler) create2FAChallenge(ctx context.Context, u *user.User) (string, error) {
	if h.redis == nil {
		return "", errors.New("redis not configured for 2FA challenges")
	}

	// Generate random challenge token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	// Store user info in Redis
	data := challengeData{
		UserID:   u.ID.String(),
		TenantID: u.TenantID.String(),
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	key := challenge2FAPrefix + token
	if err := h.redis.Set(ctx, key, string(dataJSON), challenge2FATTL).Err(); err != nil {
		return "", err
	}

	return token, nil
}

// validate2FAChallenge validates a challenge token and returns the associated user
func (h *Handler) validate2FAChallenge(ctx context.Context, token string) (*user.User, error) {
	if h.redis == nil {
		return nil, errors.New("redis not configured for 2FA challenges")
	}

	key := challenge2FAPrefix + token
	dataJSON, err := h.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var data challengeData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return nil, err
	}

	// Get user from database
	userID, err := uuid.Parse(data.UserID)
	if err != nil {
		return nil, err
	}

	return h.userService.GetByID(ctx, userID)
}

// delete2FAChallenge removes a used challenge token
func (h *Handler) delete2FAChallenge(ctx context.Context, token string) {
	if h.redis == nil {
		return
	}
	h.redis.Del(ctx, challenge2FAPrefix+token)
}

// verifyTOTP verifies a TOTP code for a user
func (h *Handler) verifyTOTP(ctx context.Context, u *user.User, code string) bool {
	if u.TOTPSecret == nil || len(u.TOTPSecret) == 0 {
		return false
	}

	// Get tenant key for decryption
	masterKey, err := crypto.GetKeyManager().GetMasterKey()
	if err != nil {
		h.logger.Error("failed to get master key for TOTP verification", "error", err)
		return false
	}

	tenantKey, err := crypto.DeriveTenantKey(masterKey, u.TenantID)
	if err != nil {
		h.logger.Error("failed to derive tenant key for TOTP verification", "error", err)
		return false
	}
	defer crypto.Zero(tenantKey)

	// Verify TOTP code
	totpMgr := NewTOTPManager(nil)
	valid, err := totpMgr.VerifyCode(u.TOTPSecret, code, tenantKey)
	if err != nil {
		h.logger.Error("TOTP verification error", "error", err)
		return false
	}

	return valid
}

// ============== Password Reset Endpoints ==============

const (
	passwordResetPrefix = "password_reset:"
	passwordResetTTL    = 1 * time.Hour
)

// ForgotPasswordRequest represents a forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword handles POST /api/v1/auth/forgot-password
// Generates a password reset token and stores it in Redis
// SECURITY: Always returns 200 OK to prevent user enumeration
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clientIP := h.getClientIP(r)

	var req ForgotPasswordRequest
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

	// Always return success to prevent user enumeration (CWE-203)
	defer func() {
		api.JSONResponse(w, http.StatusOK, map[string]string{
			"message": "If an account exists with that email, a password reset link has been sent",
		})
	}()

	// Try to find the user
	u, err := h.userService.GetByEmailGlobal(ctx, req.Email)
	if err != nil {
		// User not found - log but return success
		h.logger.Debug("password reset requested for non-existent email", "ip", clientIP)
		return
	}

	if !u.IsActive {
		// Inactive user - log but return success
		h.logger.Debug("password reset requested for inactive user", "user_id", u.ID, "ip", clientIP)
		return
	}

	// Generate reset token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		h.logger.Error("failed to generate reset token", "error", err)
		return
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	// Store token in Redis with user ID
	if h.redis != nil {
		key := passwordResetPrefix + token
		if err := h.redis.Set(ctx, key, u.ID.String(), passwordResetTTL).Err(); err != nil {
			h.logger.Error("failed to store reset token", "error", err)
			return
		}
	} else {
		h.logger.Error("redis not configured for password reset")
		return
	}

	// Log the password reset request for audit
	h.logAuthEvent(ctx, audit.EventPasswordReset, &u.ID, &u.TenantID, clientIP, r.UserAgent(), map[string]any{
		"action": "requested",
	})

	// TODO: Send email with reset link containing token
	// For now, log the token for development purposes
	h.logger.Info("password reset token generated",
		"user_id", u.ID,
		"token", token, // Remove this in production!
	)
}

// ResetPasswordRequest represents a password reset request
type ResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// ResetPassword handles POST /api/v1/auth/reset-password
// Validates the reset token and updates the user's password
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clientIP := h.getClientIP(r)

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	if req.Token == "" || req.Password == "" {
		api.ValidationError(w, map[string]string{
			"error": "Token and password are required",
		})
		return
	}

	// Validate password meets policy
	if err := ValidatePassword(req.Password, nil); err != nil {
		h.handlePasswordValidationError(w, err)
		return
	}

	// Get user ID from Redis
	if h.redis == nil {
		h.logger.Error("redis not configured for password reset")
		api.InternalError(w)
		return
	}

	key := passwordResetPrefix + req.Token
	userIDStr, err := h.redis.Get(ctx, key).Result()
	if err != nil {
		h.logger.Debug("invalid or expired reset token", "ip", clientIP)
		api.JSONError(w, http.StatusBadRequest, "Invalid or expired reset token", api.ErrCodeInvalidToken)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("invalid user ID in reset token", "error", err)
		api.JSONError(w, http.StatusBadRequest, "Invalid reset token", api.ErrCodeInvalidToken)
		return
	}

	// Get user
	u, err := h.userService.GetByID(ctx, userID)
	if err != nil {
		h.logger.Error("user not found for reset token", "user_id", userID)
		api.JSONError(w, http.StatusBadRequest, "Invalid reset token", api.ErrCodeInvalidToken)
		return
	}

	// Hash new password
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		api.InternalError(w)
		return
	}

	// Update password in database
	if err := h.userService.UpdatePasswordDirect(ctx, userID, passwordHash); err != nil {
		h.logger.Error("failed to update password", "error", err)
		api.InternalError(w)
		return
	}

	// Delete the reset token (one-time use)
	h.redis.Del(ctx, key)

	// Invalidate all user sessions for security
	if err := h.sessionManager.DeleteAllUserSessions(ctx, userID); err != nil {
		h.logger.Error("failed to invalidate sessions after password reset", "error", err)
		// Continue - password was updated successfully
	}

	// Log the password reset completion
	h.logAuthEvent(ctx, audit.EventPasswordReset, &u.ID, &u.TenantID, clientIP, r.UserAgent(), map[string]any{
		"action": "completed",
	})

	api.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "Password has been reset successfully",
	})
}

// handlePasswordValidationError handles password validation errors
func (h *Handler) handlePasswordValidationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrPasswordTooShort):
		api.ValidationError(w, map[string]string{
			"password": "Password must be at least 12 characters",
		})
	case errors.Is(err, ErrPasswordNoUppercase):
		api.ValidationError(w, map[string]string{
			"password": "Password must contain at least one uppercase letter",
		})
	case errors.Is(err, ErrPasswordNoLowercase):
		api.ValidationError(w, map[string]string{
			"password": "Password must contain at least one lowercase letter",
		})
	case errors.Is(err, ErrPasswordNoDigit):
		api.ValidationError(w, map[string]string{
			"password": "Password must contain at least one digit",
		})
	default:
		api.ValidationError(w, map[string]string{
			"password": "Invalid password",
		})
	}
}

// ============== Profile Endpoints ==============

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// UpdateProfile handles PATCH /api/v1/auth/profile
// Requires authentication
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clientIP := h.getClientIP(r)

	// Get user ID from context (set by auth middleware)
	userIDStr := api.GetUserID(ctx)
	if userIDStr == "" {
		api.JSONError(w, http.StatusUnauthorized, "Not authenticated", api.ErrCodeUnauthorized)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.JSONError(w, http.StatusUnauthorized, "Invalid user ID", api.ErrCodeUnauthorized)
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	// Get current user
	u, err := h.userService.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			api.JSONError(w, http.StatusUnauthorized, "User not found", api.ErrCodeUnauthorized)
			return
		}
		h.logger.Error("failed to get user", "error", err)
		api.InternalError(w)
		return
	}

	// Track what was updated
	updated := false
	changes := map[string]any{}

	// Update name if provided
	if req.Name != "" && req.Name != u.Name {
		u.Name = req.Name
		updated = true
		changes["name"] = "updated"
	}

	// Update email if provided
	if req.Email != "" && req.Email != u.Email {
		// Validate email format
		if !isValidEmail(req.Email) {
			api.ValidationError(w, map[string]string{
				"email": "Invalid email format",
			})
			return
		}

		// Check email uniqueness within tenant
		existing, err := h.userService.GetByEmail(ctx, u.TenantID, req.Email)
		if err == nil && existing.ID != u.ID {
			api.ValidationError(w, map[string]string{
				"email": "Email is already in use",
			})
			return
		}

		u.Email = req.Email
		u.EmailVerified = false // Reset verification on email change
		updated = true
		changes["email"] = "updated"
	}

	if !updated {
		api.JSONResponse(w, http.StatusOK, MeResponse{
			ID:       u.ID.String(),
			Email:    u.Email,
			Name:     u.Name,
			Role:     string(u.Role),
			TenantID: u.TenantID.String(),
		})
		return
	}

	// Save changes
	if err := h.userService.UpdateProfile(ctx, u); err != nil {
		if errors.Is(err, user.ErrUserEmailExists) {
			api.ValidationError(w, map[string]string{
				"email": "Email is already in use",
			})
			return
		}
		h.logger.Error("failed to update profile", "error", err)
		api.InternalError(w)
		return
	}

	// Log profile update
	h.logAuthEvent(ctx, "auth.profile_updated", &u.ID, &u.TenantID, clientIP, r.UserAgent(), changes)

	// Get tenant name for response
	tenantName := ""
	if h.tenantService != nil {
		t, err := h.tenantService.GetByID(ctx, u.TenantID)
		if err == nil && t != nil {
			tenantName = t.Name
		}
	}

	api.JSONResponse(w, http.StatusOK, MeResponse{
		ID:         u.ID.String(),
		Email:      u.Email,
		Name:       u.Name,
		Role:       string(u.Role),
		TenantID:   u.TenantID.String(),
		TenantName: tenantName,
	})
}

// ============== Change Password Endpoint ==============

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword handles POST /api/v1/auth/change-password
// Requires authentication
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clientIP := h.getClientIP(r)

	// Get user ID from context (set by auth middleware)
	userIDStr := api.GetUserID(ctx)
	if userIDStr == "" {
		api.JSONError(w, http.StatusUnauthorized, "Not authenticated", api.ErrCodeUnauthorized)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.JSONError(w, http.StatusUnauthorized, "Invalid user ID", api.ErrCodeUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		api.ValidationError(w, map[string]string{
			"error": "Current password and new password are required",
		})
		return
	}

	// Validate new password meets policy
	if err := ValidatePassword(req.NewPassword, nil); err != nil {
		h.handlePasswordValidationError(w, err)
		return
	}

	// Get user
	u, err := h.userService.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			api.JSONError(w, http.StatusUnauthorized, "User not found", api.ErrCodeUnauthorized)
			return
		}
		h.logger.Error("failed to get user", "error", err)
		api.InternalError(w)
		return
	}

	// Update password (this verifies current password internally)
	if err := h.userService.UpdatePassword(ctx, userID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, ErrPasswordInvalid) {
			api.JSONError(w, http.StatusUnauthorized, "Current password is incorrect", api.ErrCodeInvalidCredentials)
			return
		}
		h.logger.Error("failed to change password", "error", err)
		api.InternalError(w)
		return
	}

	// Log password change
	h.logAuthEvent(ctx, audit.EventPasswordChange, &u.ID, &u.TenantID, clientIP, r.UserAgent(), nil)

	// Optionally invalidate other sessions (keep current session active)
	// This provides security by logging out potentially compromised sessions
	// while keeping the user logged in on the current device
	refreshToken, err := GetRefreshTokenFromCookie(r)
	if err == nil && refreshToken != "" {
		// Get current session
		currentSession, err := h.sessionManager.ValidateRefreshToken(ctx, refreshToken)
		if err == nil {
			// Delete all sessions except current
			if err := h.sessionManager.DeleteAllUserSessionsExcept(ctx, userID, currentSession.ID); err != nil {
				h.logger.Error("failed to invalidate other sessions", "error", err)
				// Continue - password was changed successfully
			}
		}
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "Password changed successfully",
	})
}

// isValidEmail validates email format
func isValidEmail(email string) bool {
	// Simple email validation - matches the pattern in user service
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			if atIndex != -1 {
				return false // Multiple @
			}
			atIndex = i
		}
	}
	if atIndex < 1 || atIndex > len(email)-3 {
		return false
	}
	// Check for dot after @
	dotFound := false
	for i := atIndex + 1; i < len(email); i++ {
		if email[i] == '.' {
			dotFound = true
			break
		}
	}
	return dotFound
}

// ============== Audit Logging Helpers ==============

// logAuthEvent logs an authentication event (with IP anonymization per FR-103)
func (h *Handler) logAuthEvent(ctx context.Context, event string, userID, tenantID *uuid.UUID, ip, userAgent string, metadata map[string]any) {
	if h.auditLogger == nil {
		return
	}

	logCtx := &audit.LogContext{
		UserID:    userID,
		TenantID:  tenantID,
		IPAddress: &ip,
		UserAgent: &userAgent,
	}

	// The audit logger will handle IP anonymization internally
	h.auditLogger.Log(ctx, logCtx, event, metadata)
}
