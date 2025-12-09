package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenRevocationList manages revoked tokens using Redis
// Tokens are stored until their natural expiry, then automatically cleaned up
type TokenRevocationList struct {
	redis  redis.Cmdable
	prefix string
}

// NewTokenRevocationList creates a new token revocation list
func NewTokenRevocationList(redisClient redis.Cmdable) *TokenRevocationList {
	return &TokenRevocationList{
		redis:  redisClient,
		prefix: "token:revoked:",
	}
}

// RevokeToken adds a token to the revocation list
// The token ID (jti) is stored until the token's expiry time
func (r *TokenRevocationList) RevokeToken(ctx context.Context, tokenID string, expiry time.Time) error {
	if r.redis == nil {
		return fmt.Errorf("redis client not configured")
	}

	key := r.prefix + tokenID
	ttl := time.Until(expiry)
	if ttl <= 0 {
		// Token already expired, no need to revoke
		return nil
	}

	// Store with TTL matching token expiry (auto-cleanup)
	err := r.redis.Set(ctx, key, "revoked", ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

// RevokeTokenWithReason adds a token to the revocation list with a reason
func (r *TokenRevocationList) RevokeTokenWithReason(ctx context.Context, tokenID string, expiry time.Time, reason string) error {
	if r.redis == nil {
		return fmt.Errorf("redis client not configured")
	}

	key := r.prefix + tokenID
	ttl := time.Until(expiry)
	if ttl <= 0 {
		return nil
	}

	// Store reason as value for audit purposes
	err := r.redis.Set(ctx, key, reason, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

// IsRevoked checks if a token has been revoked
func (r *TokenRevocationList) IsRevoked(ctx context.Context, tokenID string) (bool, error) {
	if r.redis == nil {
		// Fail closed - if Redis unavailable, treat as revoked for security
		return true, fmt.Errorf("redis client not configured")
	}

	key := r.prefix + tokenID
	exists, err := r.redis.Exists(ctx, key).Result()
	if err != nil {
		// Fail closed on Redis errors for security
		return true, fmt.Errorf("failed to check revocation: %w", err)
	}

	return exists > 0, nil
}

// RevokeAllUserTokens revokes all tokens for a user by storing a "revoke all before" timestamp
// Any token issued before this timestamp is considered revoked
func (r *TokenRevocationList) RevokeAllUserTokens(ctx context.Context, userID string) error {
	if r.redis == nil {
		return fmt.Errorf("redis client not configured")
	}

	key := r.prefix + "user:" + userID + ":revoked_before"
	// Store current timestamp, TTL of 7 days (max refresh token lifetime)
	err := r.redis.Set(ctx, key, time.Now().Unix(), 7*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	return nil
}

// IsUserTokenRevoked checks if a user's token was issued before a mass revocation
func (r *TokenRevocationList) IsUserTokenRevoked(ctx context.Context, userID string, issuedAt time.Time) (bool, error) {
	if r.redis == nil {
		return true, fmt.Errorf("redis client not configured")
	}

	key := r.prefix + "user:" + userID + ":revoked_before"
	val, err := r.redis.Get(ctx, key).Int64()
	if err == redis.Nil {
		// No mass revocation for this user
		return false, nil
	}
	if err != nil {
		// Fail closed on Redis errors
		return true, fmt.Errorf("failed to check user revocation: %w", err)
	}

	// Token is revoked if issued before the revocation timestamp
	return issuedAt.Unix() < val, nil
}

// RevokeAllTenantTokens revokes all tokens for a tenant
func (r *TokenRevocationList) RevokeAllTenantTokens(ctx context.Context, tenantID string) error {
	if r.redis == nil {
		return fmt.Errorf("redis client not configured")
	}

	key := r.prefix + "tenant:" + tenantID + ":revoked_before"
	err := r.redis.Set(ctx, key, time.Now().Unix(), 7*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to revoke tenant tokens: %w", err)
	}

	return nil
}

// IsTenantTokenRevoked checks if a tenant's token was issued before a mass revocation
func (r *TokenRevocationList) IsTenantTokenRevoked(ctx context.Context, tenantID string, issuedAt time.Time) (bool, error) {
	if r.redis == nil {
		return true, fmt.Errorf("redis client not configured")
	}

	key := r.prefix + "tenant:" + tenantID + ":revoked_before"
	val, err := r.redis.Get(ctx, key).Int64()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return true, fmt.Errorf("failed to check tenant revocation: %w", err)
	}

	return issuedAt.Unix() < val, nil
}

// CheckRevocation performs a complete revocation check for a token
// Checks individual token, user-level, and tenant-level revocations
func (r *TokenRevocationList) CheckRevocation(ctx context.Context, claims *Claims) (bool, string, error) {
	// Check individual token revocation
	if claims.ID != "" {
		revoked, err := r.IsRevoked(ctx, claims.ID)
		if err != nil {
			return true, "revocation_check_failed", err
		}
		if revoked {
			return true, "token_revoked", nil
		}
	}

	// Check user-level revocation
	if claims.UserID != "" {
		revoked, err := r.IsUserTokenRevoked(ctx, claims.UserID, claims.IssuedAt.Time)
		if err != nil {
			return true, "revocation_check_failed", err
		}
		if revoked {
			return true, "user_tokens_revoked", nil
		}
	}

	// Check tenant-level revocation
	if claims.TenantID != "" {
		revoked, err := r.IsTenantTokenRevoked(ctx, claims.TenantID, claims.IssuedAt.Time)
		if err != nil {
			return true, "revocation_check_failed", err
		}
		if revoked {
			return true, "tenant_tokens_revoked", nil
		}
	}

	return false, "", nil
}
