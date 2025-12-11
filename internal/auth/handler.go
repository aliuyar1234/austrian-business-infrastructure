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
