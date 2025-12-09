package auth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/audit"
	"github.com/austrian-business-infrastructure/fo/internal/crypto"
	"github.com/google/uuid"
)

// Setup2FAResponse is the response for 2FA setup
type Setup2FAResponse struct {
	Secret   string `json:"secret"` // Base64 encoded for display
	OTPURL   string `json:"otp_url"`
	QRCode   string `json:"qr_code"` // Base64 PNG
	Issuer   string `json:"issuer"`
	Account  string `json:"account"`
}

// Verify2FARequest is the request to verify and enable 2FA
type Verify2FARequest struct {
	TOTPCode string `json:"totp_code"`
}

// Verify2FAResponse is the response after enabling 2FA
type Verify2FAResponse struct {
	Enabled       bool     `json:"enabled"`
	RecoveryCodes []string `json:"recovery_codes"` // Show once, never again
}

// Disable2FARequest is the request to disable 2FA
type Disable2FARequest struct {
	TOTPCode string `json:"totp_code"`
	Password string `json:"password"`
}

// RecoveryLoginRequest is the request for login with recovery code
type RecoveryLoginRequest struct {
	ChallengeToken string `json:"challenge_token"`
	RecoveryCode   string `json:"recovery_code"`
}

// RegenerateRecoveryRequest is the request to regenerate recovery codes
type RegenerateRecoveryRequest struct {
	TOTPCode string `json:"totp_code"`
}

// RegenerateRecoveryResponse is the response with new recovery codes
type RegenerateRecoveryResponse struct {
	RecoveryCodes []string `json:"recovery_codes"`
	CodesUsed     int      `json:"previous_codes_used"`
}

// TwoFAStatusResponse shows 2FA status
type TwoFAStatusResponse struct {
	Enabled             bool `json:"enabled"`
	RecoveryCodesRemaining int  `json:"recovery_codes_remaining"`
}

// Register2FARoutes registers 2FA-specific routes
func (h *Handler) Register2FARoutes(router *api.Router, authMiddleware func(http.Handler) http.Handler) {
	// These routes require authentication
	router.Handle("POST /api/v1/auth/2fa/setup", authMiddleware(http.HandlerFunc(h.Setup2FA)))
	router.Handle("POST /api/v1/auth/2fa/verify", authMiddleware(http.HandlerFunc(h.Verify2FA)))
	router.Handle("DELETE /api/v1/auth/2fa", authMiddleware(http.HandlerFunc(h.Disable2FA)))
	router.Handle("POST /api/v1/auth/2fa/recovery-codes/regenerate", authMiddleware(http.HandlerFunc(h.RegenerateRecoveryCodes)))
	router.Handle("GET /api/v1/auth/2fa/status", authMiddleware(http.HandlerFunc(h.Get2FAStatus)))

	// This route does not require full auth - uses challenge token
	router.HandleFunc("POST /api/v1/auth/login/recovery", h.LoginWithRecovery)
}

// Setup2FA handles POST /api/v1/auth/2fa/setup
// Generates a new TOTP secret and QR code for the user
func (h *Handler) Setup2FA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := api.GetUserID(ctx)

	if userID == "" {
		api.JSONError(w, http.StatusUnauthorized, "Authentication required", api.ErrCodeUnauthorized)
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		api.InternalError(w)
		return
	}

	// Get user
	u, err := h.userService.GetByID(ctx, userUUID)
	if err != nil {
		h.logger.Error("failed to get user for 2FA setup", "error", err)
		api.InternalError(w)
		return
	}

	// Check if 2FA is already enabled
	if u.TOTPEnabled {
		api.JSONError(w, http.StatusConflict, "2FA is already enabled", "2FA_ALREADY_ENABLED")
		return
	}

	// Generate TOTP secret
	totpMgr := NewTOTPManager(nil) // keyManager not needed for generation
	setupInfo, err := totpMgr.GenerateSecret(u.Email)
	if err != nil {
		h.logger.Error("failed to generate TOTP secret", "error", err)
		api.InternalError(w)
		return
	}

	// Store the unencrypted secret temporarily in Redis for verification
	// We'll encrypt and store in DB only after verification
	if h.redis != nil {
		secretKey := "2fa_setup:" + userID
		h.redis.Set(ctx, secretKey, base64.StdEncoding.EncodeToString(setupInfo.Secret), challenge2FATTL)
	}

	api.JSONResponse(w, http.StatusOK, Setup2FAResponse{
		Secret:  base64.StdEncoding.EncodeToString(setupInfo.Secret),
		OTPURL:  setupInfo.OTPURL,
		QRCode:  base64.StdEncoding.EncodeToString(setupInfo.QRCode),
		Issuer:  setupInfo.Issuer,
		Account: setupInfo.Account,
	})
}

// Verify2FA handles POST /api/v1/auth/2fa/verify
// Verifies the TOTP code and enables 2FA, returning recovery codes
func (h *Handler) Verify2FA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := api.GetUserID(ctx)
	tenantID := api.GetTenantID(ctx)
	clientIP := h.getClientIP(r)

	if userID == "" {
		api.JSONError(w, http.StatusUnauthorized, "Authentication required", api.ErrCodeUnauthorized)
		return
	}

	var req Verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	if req.TOTPCode == "" {
		api.ValidationError(w, map[string]string{
			"totp_code": "TOTP code is required",
		})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		api.InternalError(w)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.InternalError(w)
		return
	}

	// Get the temporary secret from Redis
	if h.redis == nil {
		api.JSONError(w, http.StatusServiceUnavailable, "Service unavailable", "SERVICE_UNAVAILABLE")
		return
	}

	secretKey := "2fa_setup:" + userID
	secretB64, err := h.redis.Get(ctx, secretKey).Result()
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "No 2FA setup in progress. Call POST /2fa/setup first.", "NO_2FA_SETUP")
		return
	}

	secret, err := base64.StdEncoding.DecodeString(secretB64)
	if err != nil {
		api.InternalError(w)
		return
	}

	// Verify the TOTP code
	totpMgr := NewTOTPManager(nil)
	if !totpMgr.VerifyCodePlainSecret(secret, req.TOTPCode) {
		api.JSONError(w, http.StatusUnauthorized, "Invalid TOTP code", api.ErrCodeInvalidCredentials)
		return
	}

	// Get tenant key for encryption
	masterKey, err := crypto.GetKeyManager().GetMasterKey()
	if err != nil {
		h.logger.Error("failed to get master key", "error", err)
		api.InternalError(w)
		return
	}

	tenantKey, err := crypto.DeriveTenantKey(masterKey, tenantUUID)
	if err != nil {
		h.logger.Error("failed to derive tenant key", "error", err)
		api.InternalError(w)
		return
	}
	defer crypto.Zero(tenantKey)

	// Encrypt the TOTP secret
	encryptedSecret, err := crypto.Encrypt(secret, tenantKey)
	if err != nil {
		h.logger.Error("failed to encrypt TOTP secret", "error", err)
		api.InternalError(w)
		return
	}

	// Generate recovery codes
	recoveryMgr := NewRecoveryCodeManager()
	plainCodes, encryptedCodes, err := recoveryMgr.GenerateCodes(tenantKey)
	if err != nil {
		h.logger.Error("failed to generate recovery codes", "error", err)
		api.InternalError(w)
		return
	}

	// Store in database
	if err := h.userService.SetTOTPSecret(ctx, userUUID, encryptedSecret); err != nil {
		h.logger.Error("failed to store TOTP secret", "error", err)
		api.InternalError(w)
		return
	}

	if err := h.userService.SetRecoveryCodes(ctx, userUUID, encryptedCodes); err != nil {
		h.logger.Error("failed to store recovery codes", "error", err)
		api.InternalError(w)
		return
	}

	if err := h.userService.EnableTOTP(ctx, userUUID); err != nil {
		h.logger.Error("failed to enable TOTP", "error", err)
		api.InternalError(w)
		return
	}

	// Delete temporary secret
	h.redis.Del(ctx, secretKey)

	// Audit log
	h.logAuthEvent(ctx, audit.Event2FAEnabled, &userUUID, &tenantUUID, clientIP, r.UserAgent(), nil)

	api.JSONResponse(w, http.StatusOK, Verify2FAResponse{
		Enabled:       true,
		RecoveryCodes: plainCodes,
	})
}

// Disable2FA handles DELETE /api/v1/auth/2fa
// Disables 2FA for the user (requires TOTP code and password)
func (h *Handler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := api.GetUserID(ctx)
	tenantID := api.GetTenantID(ctx)
	clientIP := h.getClientIP(r)

	if userID == "" {
		api.JSONError(w, http.StatusUnauthorized, "Authentication required", api.ErrCodeUnauthorized)
		return
	}

	var req Disable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	if req.TOTPCode == "" || req.Password == "" {
		api.ValidationError(w, map[string]string{
			"error": "TOTP code and password are required",
		})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		api.InternalError(w)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.InternalError(w)
		return
	}

	// Get user and verify password
	u, err := h.userService.GetByID(ctx, userUUID)
	if err != nil {
		h.logger.Error("failed to get user", "error", err)
		api.InternalError(w)
		return
	}

	if !u.TOTPEnabled {
		api.JSONError(w, http.StatusBadRequest, "2FA is not enabled", "2FA_NOT_ENABLED")
		return
	}

	// Verify password
	if err := h.userService.VerifyPassword(ctx, u.ID, req.Password); err != nil {
		api.JSONError(w, http.StatusUnauthorized, "Invalid password", api.ErrCodeInvalidCredentials)
		return
	}

	// Get tenant key
	masterKey, err := crypto.GetKeyManager().GetMasterKey()
	if err != nil {
		h.logger.Error("failed to get master key", "error", err)
		api.InternalError(w)
		return
	}

	tenantKey, err := crypto.DeriveTenantKey(masterKey, tenantUUID)
	if err != nil {
		h.logger.Error("failed to derive tenant key", "error", err)
		api.InternalError(w)
		return
	}
	defer crypto.Zero(tenantKey)

	// Verify TOTP code
	totpMgr := NewTOTPManager(nil)
	valid, err := totpMgr.VerifyCode(u.TOTPSecret, req.TOTPCode, tenantKey)
	if err != nil || !valid {
		api.JSONError(w, http.StatusUnauthorized, "Invalid TOTP code", api.ErrCodeInvalidCredentials)
		return
	}

	// Disable 2FA
	if err := h.userService.DisableTOTP(ctx, userUUID); err != nil {
		h.logger.Error("failed to disable 2FA", "error", err)
		api.InternalError(w)
		return
	}

	// Audit log
	h.logAuthEvent(ctx, audit.Event2FADisabled, &userUUID, &tenantUUID, clientIP, r.UserAgent(), nil)

	api.JSONResponse(w, http.StatusOK, map[string]bool{
		"disabled": true,
	})
}

// LoginWithRecovery handles POST /api/v1/auth/login/recovery
// Login using a recovery code instead of TOTP
func (h *Handler) LoginWithRecovery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clientIP := h.getClientIP(r)

	var req RecoveryLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	if req.ChallengeToken == "" || req.RecoveryCode == "" {
		api.ValidationError(w, map[string]string{
			"error": "Challenge token and recovery code are required",
		})
		return
	}

	// Validate challenge token and get user
	u, err := h.validate2FAChallenge(ctx, req.ChallengeToken)
	if err != nil {
		h.logAuthEvent(ctx, audit.EventLoginFailed, nil, nil, clientIP, r.UserAgent(), map[string]any{
			"reason": "invalid_recovery_challenge",
		})
		api.JSONError(w, http.StatusUnauthorized, "Invalid or expired challenge", api.ErrCodeInvalidToken)
		return
	}

	// Get tenant key
	masterKey, err := crypto.GetKeyManager().GetMasterKey()
	if err != nil {
		h.logger.Error("failed to get master key", "error", err)
		api.InternalError(w)
		return
	}

	tenantKey, err := crypto.DeriveTenantKey(masterKey, u.TenantID)
	if err != nil {
		h.logger.Error("failed to derive tenant key", "error", err)
		api.InternalError(w)
		return
	}
	defer crypto.Zero(tenantKey)

	// Validate recovery code
	recoveryMgr := NewRecoveryCodeManager()
	updatedCodes, err := recoveryMgr.ValidateCode(u.RecoveryCodes, req.RecoveryCode, tenantKey)
	if err != nil {
		h.logAuthEvent(ctx, audit.EventLoginFailed, &u.ID, &u.TenantID, clientIP, r.UserAgent(), map[string]any{
			"reason": "invalid_recovery_code",
		})
		api.JSONError(w, http.StatusUnauthorized, "Invalid recovery code", api.ErrCodeInvalidCredentials)
		return
	}

	// Update recovery codes in database
	if err := h.userService.SetRecoveryCodes(ctx, u.ID, updatedCodes); err != nil {
		h.logger.Error("failed to update recovery codes", "error", err)
		// Continue - login should still succeed
	}

	// Increment used count
	if err := h.userService.IncrementRecoveryCodesUsed(ctx, u.ID); err != nil {
		h.logger.Error("failed to increment recovery codes used", "error", err)
	}

	// Delete challenge token
	h.delete2FAChallenge(ctx, req.ChallengeToken)

	// Complete login
	h.completeLogin(w, r, u, clientIP)
}

// RegenerateRecoveryCodes handles POST /api/v1/auth/2fa/recovery-codes/regenerate
func (h *Handler) RegenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := api.GetUserID(ctx)
	tenantID := api.GetTenantID(ctx)

	if userID == "" {
		api.JSONError(w, http.StatusUnauthorized, "Authentication required", api.ErrCodeUnauthorized)
		return
	}

	var req RegenerateRecoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	if req.TOTPCode == "" {
		api.ValidationError(w, map[string]string{
			"totp_code": "TOTP code is required",
		})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		api.InternalError(w)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.InternalError(w)
		return
	}

	// Get user
	u, err := h.userService.GetByID(ctx, userUUID)
	if err != nil {
		h.logger.Error("failed to get user", "error", err)
		api.InternalError(w)
		return
	}

	if !u.TOTPEnabled {
		api.JSONError(w, http.StatusBadRequest, "2FA is not enabled", "2FA_NOT_ENABLED")
		return
	}

	// Get tenant key
	masterKey, err := crypto.GetKeyManager().GetMasterKey()
	if err != nil {
		h.logger.Error("failed to get master key", "error", err)
		api.InternalError(w)
		return
	}

	tenantKey, err := crypto.DeriveTenantKey(masterKey, tenantUUID)
	if err != nil {
		h.logger.Error("failed to derive tenant key", "error", err)
		api.InternalError(w)
		return
	}
	defer crypto.Zero(tenantKey)

	// Verify TOTP code
	totpMgr := NewTOTPManager(nil)
	valid, err := totpMgr.VerifyCode(u.TOTPSecret, req.TOTPCode, tenantKey)
	if err != nil || !valid {
		api.JSONError(w, http.StatusUnauthorized, "Invalid TOTP code", api.ErrCodeInvalidCredentials)
		return
	}

	// Get count of previously used codes
	recoveryMgr := NewRecoveryCodeManager()
	usedCount, _ := recoveryMgr.CountUsedCodes(u.RecoveryCodes, tenantKey)

	// Generate new recovery codes
	plainCodes, encryptedCodes, err := recoveryMgr.GenerateCodes(tenantKey)
	if err != nil {
		h.logger.Error("failed to generate recovery codes", "error", err)
		api.InternalError(w)
		return
	}

	// Store new codes
	if err := h.userService.SetRecoveryCodes(ctx, userUUID, encryptedCodes); err != nil {
		h.logger.Error("failed to store recovery codes", "error", err)
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, RegenerateRecoveryResponse{
		RecoveryCodes: plainCodes,
		CodesUsed:     usedCount,
	})
}

// Get2FAStatus handles GET /api/v1/auth/2fa/status
func (h *Handler) Get2FAStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := api.GetUserID(ctx)
	tenantID := api.GetTenantID(ctx)

	if userID == "" {
		api.JSONError(w, http.StatusUnauthorized, "Authentication required", api.ErrCodeUnauthorized)
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		api.InternalError(w)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.InternalError(w)
		return
	}

	// Get user
	u, err := h.userService.GetByID(ctx, userUUID)
	if err != nil {
		h.logger.Error("failed to get user", "error", err)
		api.InternalError(w)
		return
	}

	remaining := 0
	if u.TOTPEnabled && len(u.RecoveryCodes) > 0 {
		// Get tenant key
		masterKey, err := crypto.GetKeyManager().GetMasterKey()
		if err == nil {
			tenantKey, err := crypto.DeriveTenantKey(masterKey, tenantUUID)
			if err == nil {
				recoveryMgr := NewRecoveryCodeManager()
				remaining, _ = recoveryMgr.GetRemainingCount(u.RecoveryCodes, tenantKey)
				crypto.Zero(tenantKey)
			}
		}
	}

	api.JSONResponse(w, http.StatusOK, TwoFAStatusResponse{
		Enabled:             u.TOTPEnabled,
		RecoveryCodesRemaining: remaining,
	})
}
