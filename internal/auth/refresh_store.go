package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	// ErrRefreshTokenNotFound indicates the refresh token doesn't exist
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	// ErrRefreshTokenUsed indicates the refresh token was already used
	ErrRefreshTokenUsed = errors.New("refresh token already used")
	// ErrRefreshTokenExpired indicates the refresh token has expired
	ErrRefreshTokenExpired = errors.New("refresh token expired")
)

const (
	// RefreshTokenLength is the length of opaque refresh tokens in bytes
	RefreshTokenLength = 32
	// RefreshTokenPrefix is the Redis key prefix for refresh tokens
	RefreshTokenPrefix = "refresh:"
	// RefreshTokenTTL is the default TTL for refresh tokens (7 days)
	RefreshTokenTTL = 7 * 24 * time.Hour
)

// RefreshTokenData contains refresh token metadata stored in Redis
type RefreshTokenData struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RefreshStore manages opaque refresh tokens in Redis.
// Refresh tokens are one-time use (FR-107) and stored encrypted.
type RefreshStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRefreshStore creates a new refresh token store
func NewRefreshStore(client *redis.Client) *RefreshStore {
	return &RefreshStore{
		client: client,
		ttl:    RefreshTokenTTL,
	}
}

// NewRefreshStoreWithTTL creates a new refresh token store with custom TTL
func NewRefreshStoreWithTTL(client *redis.Client, ttl time.Duration) *RefreshStore {
	return &RefreshStore{
		client: client,
		ttl:    ttl,
	}
}

// Create creates a new refresh token and stores it in Redis.
// Returns the opaque token string that should be returned to the client.
func (s *RefreshStore) Create(ctx context.Context, userID, tenantID, ipAddress, userAgent string) (string, error) {
	// Generate random opaque token
	tokenBytes := make([]byte, RefreshTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	// Generate unique ID for the token
	tokenID := uuid.New().String()

	// Create token data
	now := time.Now()
	data := &RefreshTokenData{
		ID:        tokenID,
		UserID:    userID,
		TenantID:  tenantID,
		IPAddress: ipAddress,
		UserAgent: truncateUA(userAgent),
		Used:      false,
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to serialize token data: %w", err)
	}

	// Store in Redis with TTL
	key := RefreshTokenPrefix + token
	if err := s.client.Set(ctx, key, jsonData, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return token, nil
}

// Validate validates a refresh token and returns its data.
// Returns an error if the token is not found, expired, or already used.
// NOTE: This does NOT mark the token as used - call Use() separately.
func (s *RefreshStore) Validate(ctx context.Context, token string) (*RefreshTokenData, error) {
	key := RefreshTokenPrefix + token

	jsonData, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	var data RefreshTokenData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse token data: %w", err)
	}

	// Check if expired
	if time.Now().After(data.ExpiresAt) {
		// Clean up expired token
		s.client.Del(ctx, key)
		return nil, ErrRefreshTokenExpired
	}

	// Check if already used
	if data.Used {
		return nil, ErrRefreshTokenUsed
	}

	return &data, nil
}

// Use marks a refresh token as used (one-time use per FR-107).
// This is called during token rotation - the old token becomes invalid.
// Returns the token data after marking as used.
func (s *RefreshStore) Use(ctx context.Context, token string) (*RefreshTokenData, error) {
	key := RefreshTokenPrefix + token

	// Use Redis transaction to atomically check and mark as used
	var data *RefreshTokenData

	err := s.client.Watch(ctx, func(tx *redis.Tx) error {
		// Get current data
		jsonData, err := tx.Get(ctx, key).Bytes()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return ErrRefreshTokenNotFound
			}
			return err
		}

		var tokenData RefreshTokenData
		if err := json.Unmarshal(jsonData, &tokenData); err != nil {
			return err
		}

		// Check if expired
		if time.Now().After(tokenData.ExpiresAt) {
			return ErrRefreshTokenExpired
		}

		// Check if already used
		if tokenData.Used {
			return ErrRefreshTokenUsed
		}

		// Mark as used
		tokenData.Used = true
		data = &tokenData

		// Serialize and update
		newJsonData, err := json.Marshal(tokenData)
		if err != nil {
			return err
		}

		// Calculate remaining TTL
		remainingTTL := time.Until(tokenData.ExpiresAt)
		if remainingTTL <= 0 {
			return ErrRefreshTokenExpired
		}

		// Update in transaction
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newJsonData, remainingTTL)
			return nil
		})
		return err
	}, key)

	if err != nil {
		return nil, err
	}

	return data, nil
}

// Rotate uses the current token and creates a new one (token rotation per FR-107).
// This is the primary method for refresh token handling.
// Returns the new token and the user info from the old token.
func (s *RefreshStore) Rotate(ctx context.Context, oldToken, ipAddress, userAgent string) (newToken string, data *RefreshTokenData, err error) {
	// Use the old token (marks it as used)
	data, err = s.Use(ctx, oldToken)
	if err != nil {
		return "", nil, err
	}

	// Create a new token
	newToken, err = s.Create(ctx, data.UserID, data.TenantID, ipAddress, userAgent)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create new refresh token: %w", err)
	}

	return newToken, data, nil
}

// Delete deletes a refresh token (for logout)
func (s *RefreshStore) Delete(ctx context.Context, token string) error {
	key := RefreshTokenPrefix + token
	return s.client.Del(ctx, key).Err()
}

// DeleteAllForUser deletes all refresh tokens for a user.
// This is used when user changes password or for forced logout.
func (s *RefreshStore) DeleteAllForUser(ctx context.Context, userID string) error {
	// Scan for all tokens belonging to this user
	// Note: This is a slow operation, use sparingly
	var cursor uint64
	var keys []string

	for {
		var scanKeys []string
		var err error
		scanKeys, cursor, err = s.client.Scan(ctx, cursor, RefreshTokenPrefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan refresh tokens: %w", err)
		}

		for _, key := range scanKeys {
			// Check if this token belongs to the user
			jsonData, err := s.client.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}

			var data RefreshTokenData
			if err := json.Unmarshal(jsonData, &data); err != nil {
				continue
			}

			if data.UserID == userID {
				keys = append(keys, key)
			}
		}

		if cursor == 0 {
			break
		}
	}

	// Delete all found keys
	if len(keys) > 0 {
		if err := s.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete refresh tokens: %w", err)
		}
	}

	return nil
}

// truncateUA truncates user agent to a reasonable length
func truncateUA(ua string) string {
	const maxLen = 255
	if len(ua) <= maxLen {
		return ua
	}
	return ua[:maxLen]
}
