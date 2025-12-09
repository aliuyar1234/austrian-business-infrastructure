package unit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	"github.com/austrian-business-infrastructure/fo/internal/auth"
)

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to create miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

func TestTokenRevocationList_RevokeToken(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	revList := auth.NewTokenRevocationList(client)
	ctx := context.Background()

	// Revoke a token
	tokenID := "test-token-123"
	expiry := time.Now().Add(time.Hour)

	err := revList.RevokeToken(ctx, tokenID, expiry)
	if err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	// Check if token is revoked
	revoked, err := revList.IsRevoked(ctx, tokenID)
	if err != nil {
		t.Fatalf("failed to check revocation: %v", err)
	}

	if !revoked {
		t.Error("expected token to be revoked")
	}
}

func TestTokenRevocationList_NotRevoked(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	revList := auth.NewTokenRevocationList(client)
	ctx := context.Background()

	// Check non-existent token
	revoked, err := revList.IsRevoked(ctx, "non-existent-token")
	if err != nil {
		t.Fatalf("failed to check revocation: %v", err)
	}

	if revoked {
		t.Error("expected token to not be revoked")
	}
}

func TestTokenRevocationList_ExpiredToken(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	revList := auth.NewTokenRevocationList(client)
	ctx := context.Background()

	// Try to revoke an already expired token
	tokenID := "expired-token"
	expiry := time.Now().Add(-time.Hour) // Already expired

	err := revList.RevokeToken(ctx, tokenID, expiry)
	if err != nil {
		t.Fatalf("revoking expired token should not error: %v", err)
	}

	// Token should not be in the list (no point storing expired tokens)
	revoked, err := revList.IsRevoked(ctx, tokenID)
	if err != nil {
		t.Fatalf("failed to check revocation: %v", err)
	}

	if revoked {
		t.Error("expired token should not be stored in revocation list")
	}
}

func TestTokenRevocationList_RevokeAllUserTokens(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	revList := auth.NewTokenRevocationList(client)
	ctx := context.Background()

	userID := "user-123"

	// Revoke all tokens for user
	err := revList.RevokeAllUserTokens(ctx, userID)
	if err != nil {
		t.Fatalf("failed to revoke user tokens: %v", err)
	}

	// Token issued before revocation should be revoked
	oldIssuedAt := time.Now().Add(-time.Hour)
	revoked, err := revList.IsUserTokenRevoked(ctx, userID, oldIssuedAt)
	if err != nil {
		t.Fatalf("failed to check user revocation: %v", err)
	}
	if !revoked {
		t.Error("token issued before revocation should be revoked")
	}

	// Token issued after revocation should not be revoked
	newIssuedAt := time.Now().Add(time.Second)
	revoked, err = revList.IsUserTokenRevoked(ctx, userID, newIssuedAt)
	if err != nil {
		t.Fatalf("failed to check user revocation: %v", err)
	}
	if revoked {
		t.Error("token issued after revocation should not be revoked")
	}
}

func TestTokenRevocationList_RevokeAllTenantTokens(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	revList := auth.NewTokenRevocationList(client)
	ctx := context.Background()

	tenantID := "tenant-456"

	// Revoke all tokens for tenant
	err := revList.RevokeAllTenantTokens(ctx, tenantID)
	if err != nil {
		t.Fatalf("failed to revoke tenant tokens: %v", err)
	}

	// Token issued before revocation should be revoked
	oldIssuedAt := time.Now().Add(-time.Hour)
	revoked, err := revList.IsTenantTokenRevoked(ctx, tenantID, oldIssuedAt)
	if err != nil {
		t.Fatalf("failed to check tenant revocation: %v", err)
	}
	if !revoked {
		t.Error("token issued before tenant revocation should be revoked")
	}
}

func TestTokenRevocationList_CheckRevocation_Individual(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	revList := auth.NewTokenRevocationList(client)
	ctx := context.Background()

	claims := &auth.Claims{
		UserID:   "user-123",
		TenantID: "tenant-456",
	}
	claims.ID = "token-789"                                        // ID is from jwt.RegisteredClaims
	claims.IssuedAt = jwt.NewNumericDate(time.Now().Add(-time.Hour)) // Set issued time

	// Revoke the specific token
	err := revList.RevokeToken(ctx, claims.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	// Check revocation
	revoked, reason, err := revList.CheckRevocation(ctx, claims)
	if err != nil {
		t.Fatalf("failed to check revocation: %v", err)
	}
	if !revoked {
		t.Error("expected token to be revoked")
	}
	if reason != "token_revoked" {
		t.Errorf("expected reason 'token_revoked', got '%s'", reason)
	}
}

func TestTokenRevocationList_RevokeWithReason(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	revList := auth.NewTokenRevocationList(client)
	ctx := context.Background()

	tokenID := "suspicious-token"
	reason := "suspicious_activity"
	expiry := time.Now().Add(time.Hour)

	err := revList.RevokeTokenWithReason(ctx, tokenID, expiry, reason)
	if err != nil {
		t.Fatalf("failed to revoke token with reason: %v", err)
	}

	// Token should be revoked
	revoked, err := revList.IsRevoked(ctx, tokenID)
	if err != nil {
		t.Fatalf("failed to check revocation: %v", err)
	}
	if !revoked {
		t.Error("expected token to be revoked")
	}
}

func TestJWTManager_WithRevocationList(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	// Create JWT manager with revocation support
	revList := auth.NewTokenRevocationList(client)
	config := auth.DefaultJWTConfig("test-secret")
	config.UseES256 = false // Use HS256 for simpler testing
	jwtManager := auth.NewJWTManager(config)
	jwtManager.SetRevocationList(revList)

	ctx := context.Background()

	// Generate a token pair
	userInfo := &auth.UserInfo{
		UserID:   "user-123",
		TenantID: "tenant-456",
		Role:     "admin",
	}

	tokenPair, err := jwtManager.GenerateTokenPair(userInfo)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	// Token should be valid initially
	claims, err := jwtManager.ValidateAccessTokenWithContext(ctx, tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("valid token should pass validation: %v", err)
	}

	// Revoke the token
	err = revList.RevokeToken(ctx, claims.ID, tokenPair.ExpiresAt)
	if err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	// Token should now fail validation
	_, err = jwtManager.ValidateAccessTokenWithContext(ctx, tokenPair.AccessToken)
	if err != auth.ErrTokenRevoked {
		t.Errorf("expected ErrTokenRevoked, got: %v", err)
	}
}

func TestJWTManager_UserLevelRevocation(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	revList := auth.NewTokenRevocationList(client)
	config := auth.DefaultJWTConfig("test-secret")
	config.UseES256 = false
	jwtManager := auth.NewJWTManager(config)
	jwtManager.SetRevocationList(revList)

	ctx := context.Background()

	userInfo := &auth.UserInfo{
		UserID:   "user-to-revoke",
		TenantID: "tenant-456",
		Role:     "member",
	}

	tokenPair, err := jwtManager.GenerateTokenPair(userInfo)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	// Wait for at least 1 second to ensure revocation timestamp is after token issuance
	// (revocation uses Unix timestamps with second granularity)
	time.Sleep(1100 * time.Millisecond)

	// Revoke all tokens for this user
	err = revList.RevokeAllUserTokens(ctx, userInfo.UserID)
	if err != nil {
		t.Fatalf("failed to revoke user tokens: %v", err)
	}

	// Token should now fail validation
	_, err = jwtManager.ValidateAccessTokenWithContext(ctx, tokenPair.AccessToken)
	if err != auth.ErrTokenRevoked {
		t.Errorf("expected ErrTokenRevoked after user-level revocation, got: %v", err)
	}
}

func TestJWTManager_TenantLevelRevocation(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	revList := auth.NewTokenRevocationList(client)
	config := auth.DefaultJWTConfig("test-secret")
	config.UseES256 = false
	jwtManager := auth.NewJWTManager(config)
	jwtManager.SetRevocationList(revList)

	ctx := context.Background()

	userInfo := &auth.UserInfo{
		UserID:   "user-123",
		TenantID: "tenant-to-revoke",
		Role:     "admin",
	}

	tokenPair, err := jwtManager.GenerateTokenPair(userInfo)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	// Wait for at least 1 second to ensure revocation timestamp is after token issuance
	// (revocation uses Unix timestamps with second granularity)
	time.Sleep(1100 * time.Millisecond)

	// Revoke all tokens for this tenant
	err = revList.RevokeAllTenantTokens(ctx, userInfo.TenantID)
	if err != nil {
		t.Fatalf("failed to revoke tenant tokens: %v", err)
	}

	// Token should now fail validation
	_, err = jwtManager.ValidateAccessTokenWithContext(ctx, tokenPair.AccessToken)
	if err != auth.ErrTokenRevoked {
		t.Errorf("expected ErrTokenRevoked after tenant-level revocation, got: %v", err)
	}
}

func TestJWTManager_NoRevocationList(t *testing.T) {
	// JWT manager without revocation list should still work
	config := auth.DefaultJWTConfig("test-secret")
	config.UseES256 = false
	jwtManager := auth.NewJWTManager(config)

	ctx := context.Background()

	userInfo := &auth.UserInfo{
		UserID:   "user-123",
		TenantID: "tenant-456",
		Role:     "admin",
	}

	tokenPair, err := jwtManager.GenerateTokenPair(userInfo)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	// Token should be valid (no revocation check performed)
	_, err = jwtManager.ValidateAccessTokenWithContext(ctx, tokenPair.AccessToken)
	if err != nil {
		t.Errorf("valid token should pass validation without revocation list: %v", err)
	}
}
