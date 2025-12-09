package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken   = errors.New("invalid token")
	ErrExpiredToken   = errors.New("token has expired")
	ErrInvalidClaims  = errors.New("invalid token claims")
	ErrTokenNotActive = errors.New("token not yet active")
	ErrTokenRevoked   = errors.New("token has been revoked")
)

// TokenType distinguishes between access and refresh tokens
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims represents the JWT claims for authentication.
// SECURITY: No PII (email, name) is stored in JWT claims.
// Only IDs and role are included per FR-104.
type Claims struct {
	jwt.RegisteredClaims
	UserID   string    `json:"uid"`
	TenantID string    `json:"tid"`
	Role     string    `json:"role"`
	Type     TokenType `json:"type"`
	// Email field REMOVED per FR-104 - no PII in JWT
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	// Secret is deprecated - use ECDSAKeyManager for ES256 signing
	// Kept for backward compatibility during migration
	Secret             string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
	// UseES256 enables ES256 signing (ECDSA P-256) instead of HS256
	// This should be true for production per FR-105
	UseES256 bool
}

// DefaultJWTConfig returns default JWT configuration with ES256 enabled
func DefaultJWTConfig(secret string) *JWTConfig {
	return &JWTConfig{
		Secret:             secret,
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "austrian-business-platform",
		UseES256:           true, // Default to ES256 per FR-105
	}
}

// JWTManager handles JWT operations
type JWTManager struct {
	config     *JWTConfig
	keyManager *ECDSAKeyManager
	revoker    *TokenRevocationList
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(config *JWTConfig) *JWTManager {
	return &JWTManager{
		config:     config,
		keyManager: GetECDSAKeyManager(),
	}
}

// NewJWTManagerWithKeyManager creates a JWT manager with a specific key manager
func NewJWTManagerWithKeyManager(config *JWTConfig, km *ECDSAKeyManager) *JWTManager {
	return &JWTManager{
		config:     config,
		keyManager: km,
	}
}

// NewJWTManagerWithRevocation creates a JWT manager with revocation support
func NewJWTManagerWithRevocation(config *JWTConfig, revoker *TokenRevocationList) *JWTManager {
	return &JWTManager{
		config:     config,
		keyManager: GetECDSAKeyManager(),
		revoker:    revoker,
	}
}

// SetRevocationList sets the token revocation list for the JWT manager
func (m *JWTManager) SetRevocationList(revoker *TokenRevocationList) {
	m.revoker = revoker
}

// TokenPair contains both access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// UserInfo contains user information for token generation.
// SECURITY: Email is no longer included in JWT claims per FR-104.
type UserInfo struct {
	UserID   string
	TenantID string
	Role     string
	// Email is intentionally not included in JWT claims per FR-104
}

// GenerateTokenPair creates a new access and refresh token pair
func (m *JWTManager) GenerateTokenPair(user *UserInfo) (*TokenPair, error) {
	now := time.Now()

	// Generate access token
	accessExpiry := now.Add(m.config.AccessTokenExpiry)
	accessToken, err := m.generateToken(user, AccessToken, accessExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshExpiry := now.Add(m.config.RefreshTokenExpiry)
	refreshToken, err := m.generateToken(user, RefreshToken, refreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiry,
		TokenType:    "Bearer",
	}, nil
}

// GenerateAccessToken creates a new access token
func (m *JWTManager) GenerateAccessToken(user *UserInfo) (string, time.Time, error) {
	expiry := time.Now().Add(m.config.AccessTokenExpiry)
	token, err := m.generateToken(user, AccessToken, expiry)
	return token, expiry, err
}

// GenerateRefreshToken creates a new refresh token
func (m *JWTManager) GenerateRefreshToken(user *UserInfo) (string, time.Time, error) {
	expiry := time.Now().Add(m.config.RefreshTokenExpiry)
	token, err := m.generateToken(user, RefreshToken, expiry)
	return token, expiry, err
}

func (m *JWTManager) generateToken(user *UserInfo, tokenType TokenType, expiry time.Time) (string, error) {
	// Generate unique token ID
	jti, err := generateTokenID()
	if err != nil {
		return "", err
	}

	// SECURITY: No PII in JWT claims per FR-104
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    m.config.Issuer,
			Subject:   user.UserID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiry),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		UserID:   user.UserID,
		TenantID: user.TenantID,
		Role:     user.Role,
		Type:     tokenType,
		// Email intentionally NOT included per FR-104
	}

	// Use ES256 (ECDSA P-256) signing per FR-105
	if m.config.UseES256 {
		return m.signES256(claims)
	}

	// Fallback to HS256 (deprecated, for migration only)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Secret))
}

// signES256 signs the token using ECDSA P-256 (ES256)
func (m *JWTManager) signES256(claims *Claims) (string, error) {
	privateKey, err := m.keyManager.GetPrivateKey()
	if err != nil {
		return "", fmt.Errorf("ES256 signing failed: %w", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(privateKey)
}

// ValidateToken validates a token and returns claims.
// Supports both ES256 (ECDSA) and HS256 (HMAC) for backward compatibility.
// If a revocation list is configured, checks if the token has been revoked.
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	return m.ValidateTokenWithContext(context.Background(), tokenString)
}

// ValidateTokenWithContext validates a token with context for revocation checks.
func (m *JWTManager) ValidateTokenWithContext(ctx context.Context, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Check signing method
		switch token.Method.(type) {
		case *jwt.SigningMethodECDSA:
			// ES256 - use public key
			return m.getVerificationKey()
		case *jwt.SigningMethodHMAC:
			// HS256 - use secret (deprecated, for migration)
			if !m.config.UseES256 {
				return []byte(m.config.Secret), nil
			}
			return nil, fmt.Errorf("HS256 tokens not accepted when ES256 is enabled")
		default:
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	// Check revocation list if configured
	if m.revoker != nil {
		revoked, _, err := m.revoker.CheckRevocation(ctx, claims)
		if err != nil {
			// Fail closed on revocation check errors for security
			return nil, ErrTokenRevoked
		}
		if revoked {
			return nil, ErrTokenRevoked
		}
	}

	return claims, nil
}

// getVerificationKey returns the appropriate key for token verification
func (m *JWTManager) getVerificationKey() (interface{}, error) {
	if m.config.UseES256 {
		return m.keyManager.GetPublicKey()
	}
	return []byte(m.config.Secret), nil
}

// ValidateAccessToken validates an access token
func (m *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	return m.ValidateAccessTokenWithContext(context.Background(), tokenString)
}

// ValidateAccessTokenWithContext validates an access token with context for revocation checks
func (m *JWTManager) ValidateAccessTokenWithContext(ctx context.Context, tokenString string) (*Claims, error) {
	claims, err := m.ValidateTokenWithContext(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != AccessToken {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return m.ValidateRefreshTokenWithContext(context.Background(), tokenString)
}

// ValidateRefreshTokenWithContext validates a refresh token with context for revocation checks
func (m *JWTManager) ValidateRefreshTokenWithContext(ctx context.Context, tokenString string) (*Claims, error) {
	claims, err := m.ValidateTokenWithContext(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != RefreshToken {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// generateTokenID creates a unique token ID
func generateTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateSecureToken creates a random secure token (for refresh tokens stored in DB)
func GenerateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
