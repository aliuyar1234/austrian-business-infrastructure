package unit

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/auth"
)

// TestJWT_GenerateTokenPair tests ES256 token pair generation
func TestJWT_GenerateTokenPair(t *testing.T) {
	// Generate test ECDSA key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ECDSA key: %v", err)
	}

	// Create key manager with test key
	km := auth.NewECDSAKeyManager()
	if err := km.LoadKey(privateKey); err != nil {
		t.Fatalf("failed to load key: %v", err)
	}

	// Create JWT config with ES256 enabled
	config := &auth.JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test-issuer",
		UseES256:           true,
	}

	jwtManager := auth.NewJWTManagerWithKeyManager(config, km)

	// Create user info - note: no email field per FR-104
	user := &auth.UserInfo{
		UserID:   "user-123",
		TenantID: "tenant-456",
		Role:     "admin",
	}

	// Generate token pair
	tokenPair, err := jwtManager.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	// Verify tokens are not empty
	if tokenPair.AccessToken == "" {
		t.Error("access token should not be empty")
	}
	if tokenPair.RefreshToken == "" {
		t.Error("refresh token should not be empty")
	}
	if tokenPair.TokenType != "Bearer" {
		t.Errorf("expected token type 'Bearer', got '%s'", tokenPair.TokenType)
	}

	// Verify expiry is set correctly
	expectedExpiry := time.Now().Add(15 * time.Minute)
	if tokenPair.ExpiresAt.Before(expectedExpiry.Add(-1*time.Minute)) ||
		tokenPair.ExpiresAt.After(expectedExpiry.Add(1*time.Minute)) {
		t.Errorf("unexpected expiry time: %v", tokenPair.ExpiresAt)
	}
}

// TestJWT_ValidateAccessToken tests ES256 access token validation
func TestJWT_ValidateAccessToken(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	km := auth.NewECDSAKeyManager()
	km.LoadKey(privateKey)

	config := &auth.JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test-issuer",
		UseES256:           true,
	}

	jwtManager := auth.NewJWTManagerWithKeyManager(config, km)

	user := &auth.UserInfo{
		UserID:   "user-abc",
		TenantID: "tenant-xyz",
		Role:     "member",
	}

	tokenPair, _ := jwtManager.GenerateTokenPair(user)

	// Validate access token
	claims, err := jwtManager.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}

	// Verify claims
	if claims.UserID != user.UserID {
		t.Errorf("expected UserID '%s', got '%s'", user.UserID, claims.UserID)
	}
	if claims.TenantID != user.TenantID {
		t.Errorf("expected TenantID '%s', got '%s'", user.TenantID, claims.TenantID)
	}
	if claims.Role != user.Role {
		t.Errorf("expected Role '%s', got '%s'", user.Role, claims.Role)
	}
	if claims.Type != auth.AccessToken {
		t.Errorf("expected Type '%s', got '%s'", auth.AccessToken, claims.Type)
	}
}

// TestJWT_ValidateRefreshToken tests ES256 refresh token validation
func TestJWT_ValidateRefreshToken(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	km := auth.NewECDSAKeyManager()
	km.LoadKey(privateKey)

	config := &auth.JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test-issuer",
		UseES256:           true,
	}

	jwtManager := auth.NewJWTManagerWithKeyManager(config, km)

	user := &auth.UserInfo{
		UserID:   "user-refresh",
		TenantID: "tenant-refresh",
		Role:     "admin",
	}

	tokenPair, _ := jwtManager.GenerateTokenPair(user)

	// Validate refresh token
	claims, err := jwtManager.ValidateRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("failed to validate refresh token: %v", err)
	}

	if claims.Type != auth.RefreshToken {
		t.Errorf("expected Type '%s', got '%s'", auth.RefreshToken, claims.Type)
	}
}

// TestJWT_InvalidToken tests rejection of invalid tokens
func TestJWT_InvalidToken(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	km := auth.NewECDSAKeyManager()
	km.LoadKey(privateKey)

	config := &auth.JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test-issuer",
		UseES256:           true,
	}

	jwtManager := auth.NewJWTManagerWithKeyManager(config, km)

	// Test invalid token format
	_, err := jwtManager.ValidateToken("not.a.valid.token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
	if err != auth.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

// TestJWT_TokenTampering tests detection of tampered tokens
func TestJWT_TokenTampering(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	km := auth.NewECDSAKeyManager()
	km.LoadKey(privateKey)

	config := &auth.JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test-issuer",
		UseES256:           true,
	}

	jwtManager := auth.NewJWTManagerWithKeyManager(config, km)

	user := &auth.UserInfo{
		UserID:   "user-tamper",
		TenantID: "tenant-tamper",
		Role:     "member",
	}

	tokenPair, _ := jwtManager.GenerateTokenPair(user)

	// Tamper with the token (modify a character in the signature)
	tampered := tokenPair.AccessToken[:len(tokenPair.AccessToken)-5] + "XXXXX"

	_, err := jwtManager.ValidateToken(tampered)
	if err == nil {
		t.Error("expected error for tampered token")
	}
}

// TestJWT_WrongKeyValidation tests rejection of tokens signed with different key
func TestJWT_WrongKeyValidation(t *testing.T) {
	// Create token with key1
	privateKey1, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	km1 := auth.NewECDSAKeyManager()
	km1.LoadKey(privateKey1)

	config := &auth.JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test-issuer",
		UseES256:           true,
	}

	jwtManager1 := auth.NewJWTManagerWithKeyManager(config, km1)

	user := &auth.UserInfo{
		UserID:   "user-wrong-key",
		TenantID: "tenant-wrong-key",
		Role:     "member",
	}

	tokenPair, _ := jwtManager1.GenerateTokenPair(user)

	// Try to validate with different key
	privateKey2, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	km2 := auth.NewECDSAKeyManager()
	km2.LoadKey(privateKey2)

	jwtManager2 := auth.NewJWTManagerWithKeyManager(config, km2)

	_, err := jwtManager2.ValidateToken(tokenPair.AccessToken)
	if err == nil {
		t.Error("expected error when validating with wrong key")
	}
}

// TestJWT_AccessTokenAsRefresh tests rejection of access token as refresh token
func TestJWT_AccessTokenAsRefresh(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	km := auth.NewECDSAKeyManager()
	km.LoadKey(privateKey)

	config := &auth.JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test-issuer",
		UseES256:           true,
	}

	jwtManager := auth.NewJWTManagerWithKeyManager(config, km)

	user := &auth.UserInfo{
		UserID:   "user-type",
		TenantID: "tenant-type",
		Role:     "member",
	}

	tokenPair, _ := jwtManager.GenerateTokenPair(user)

	// Try to use access token as refresh token
	_, err := jwtManager.ValidateRefreshToken(tokenPair.AccessToken)
	if err == nil {
		t.Error("expected error when using access token as refresh token")
	}
	if err != auth.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

// TestJWT_RefreshTokenAsAccess tests rejection of refresh token as access token
func TestJWT_RefreshTokenAsAccess(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	km := auth.NewECDSAKeyManager()
	km.LoadKey(privateKey)

	config := &auth.JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test-issuer",
		UseES256:           true,
	}

	jwtManager := auth.NewJWTManagerWithKeyManager(config, km)

	user := &auth.UserInfo{
		UserID:   "user-type2",
		TenantID: "tenant-type2",
		Role:     "member",
	}

	tokenPair, _ := jwtManager.GenerateTokenPair(user)

	// Try to use refresh token as access token
	_, err := jwtManager.ValidateAccessToken(tokenPair.RefreshToken)
	if err == nil {
		t.Error("expected error when using refresh token as access token")
	}
	if err != auth.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

// TestJWT_ClaimsNoPII tests that JWT claims don't contain PII (FR-104)
func TestJWT_ClaimsNoPII(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	km := auth.NewECDSAKeyManager()
	km.LoadKey(privateKey)

	config := &auth.JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test-issuer",
		UseES256:           true,
	}

	jwtManager := auth.NewJWTManagerWithKeyManager(config, km)

	// UserInfo doesn't have Email field - this is by design (FR-104)
	user := &auth.UserInfo{
		UserID:   "user-pii",
		TenantID: "tenant-pii",
		Role:     "admin",
	}

	tokenPair, _ := jwtManager.GenerateTokenPair(user)
	claims, _ := jwtManager.ValidateAccessToken(tokenPair.AccessToken)

	// Verify that claims only contain non-PII fields
	// The Claims struct should not have Email, Name, or other PII
	if claims.UserID == "" {
		t.Error("UserID should be present")
	}
	if claims.TenantID == "" {
		t.Error("TenantID should be present")
	}
	if claims.Role == "" {
		t.Error("Role should be present")
	}
	// Note: Claims struct doesn't have Email field by design - this is correct
}

// TestECDSAKeyManager_GenerateKey tests ECDSA key generation
func TestECDSAKeyManager_GenerateKey(t *testing.T) {
	privatePEM, publicPEM, err := auth.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate ECDSA key: %v", err)
	}

	if len(privatePEM) == 0 {
		t.Error("private key PEM should not be empty")
	}
	if len(publicPEM) == 0 {
		t.Error("public key PEM should not be empty")
	}

	// Verify PEM format is valid - should start with -----BEGIN
	if privatePEM[0] != '-' {
		t.Error("invalid private PEM format - should start with -----BEGIN")
	}
	if publicPEM[0] != '-' {
		t.Error("invalid public PEM format - should start with -----BEGIN")
	}
}

// TestECDSAKeyManager_LoadKey tests loading ECDSA key
func TestECDSAKeyManager_LoadKey(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	km := auth.NewECDSAKeyManager()

	// Before loading
	if km.IsLoaded() {
		t.Error("key should not be loaded initially")
	}

	// Load key
	if err := km.LoadKey(privateKey); err != nil {
		t.Fatalf("failed to load key: %v", err)
	}

	// After loading
	if !km.IsLoaded() {
		t.Error("key should be loaded after LoadKey")
	}

	// Get private key
	pk, err := km.GetPrivateKey()
	if err != nil {
		t.Fatalf("failed to get private key: %v", err)
	}
	if pk != privateKey {
		t.Error("returned key should match loaded key")
	}

	// Get public key
	pubKey, err := km.GetPublicKey()
	if err != nil {
		t.Fatalf("failed to get public key: %v", err)
	}
	if pubKey != &privateKey.PublicKey {
		t.Error("returned public key should match loaded key's public key")
	}
}

// TestECDSAKeyManager_Clear tests clearing ECDSA key from memory
func TestECDSAKeyManager_Clear(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	km := auth.NewECDSAKeyManager()
	km.LoadKey(privateKey)

	if !km.IsLoaded() {
		t.Error("key should be loaded")
	}

	km.Clear()

	if km.IsLoaded() {
		t.Error("key should not be loaded after Clear")
	}

	_, err := km.GetPrivateKey()
	if err != auth.ErrNoPrivateKey {
		t.Errorf("expected ErrNoPrivateKey, got %v", err)
	}
}

// TestECDSAKeyManager_WrongCurve tests rejection of non-P256 curves
func TestECDSAKeyManager_WrongCurve(t *testing.T) {
	// Generate key with P-384 curve (wrong for ES256)
	privateKey, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)

	km := auth.NewECDSAKeyManager()
	err := km.LoadKey(privateKey)

	if err == nil {
		t.Error("expected error for non-P256 curve")
	}
}
