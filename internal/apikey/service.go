package apikey

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	// KeyLength is the length of the random part of the API key
	KeyLength = 32
	// PrefixLength is the visible prefix length
	PrefixLength = 8
	// KeyPrefix is the prefix for all API keys
	KeyPrefix = "abp_"
)

var (
	ErrInvalidScope = errors.New("invalid scope")
)

// ValidScopes contains all valid API key scopes
var ValidScopes = []string{
	"read:all",
	"write:all",
	"read:databox",
	"write:databox",
	"read:users",
	"write:users",
	"read:audit",
}

// CreateKeyInput contains input for creating an API key
type CreateKeyInput struct {
	UserID    uuid.UUID
	TenantID  uuid.UUID
	Name      string
	Scopes    []string
	ExpiresIn *time.Duration // Optional expiry
}

// CreateKeyResult contains the result of creating an API key
type CreateKeyResult struct {
	Key      *APIKey
	RawKey   string // Only returned once at creation
}

// Service provides API key business logic
type Service struct {
	repo *Repository
}

// NewService creates a new API key service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new API key
func (s *Service) Create(ctx context.Context, input *CreateKeyInput) (*CreateKeyResult, error) {
	// Validate scopes
	if err := validateScopes(input.Scopes); err != nil {
		return nil, err
	}

	// Generate key
	rawKey, err := generateKey()
	if err != nil {
		return nil, err
	}

	keyHash := HashKey(rawKey)
	keyPrefix := rawKey[:PrefixLength]

	var expiresAt *time.Time
	if input.ExpiresIn != nil {
		t := time.Now().Add(*input.ExpiresIn)
		expiresAt = &t
	}

	key := &APIKey{
		ID:        uuid.New(),
		UserID:    input.UserID,
		TenantID:  input.TenantID,
		Name:      input.Name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Scopes:    input.Scopes,
		ExpiresAt: expiresAt,
		IsActive:  true,
	}

	if err := s.repo.Create(ctx, key); err != nil {
		return nil, err
	}

	return &CreateKeyResult{
		Key:    key,
		RawKey: rawKey,
	}, nil
}

// GetByID retrieves an API key by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*APIKey, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByUser returns all API keys for a user
func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID) ([]*APIKey, error) {
	return s.repo.ListByUser(ctx, userID)
}

// ListByTenant returns all API keys for a tenant
func (s *Service) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*APIKey, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

// Revoke revokes (deletes) an API key
func (s *Service) Revoke(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// Validate validates an API key and returns it if valid
func (s *Service) Validate(ctx context.Context, rawKey string) (*APIKey, error) {
	if len(rawKey) < PrefixLength {
		return nil, ErrAPIKeyNotFound
	}

	keyHash := HashKey(rawKey)

	key, err := s.repo.GetByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	if !key.IsActive {
		return nil, ErrAPIKeyInactive
	}

	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, ErrAPIKeyExpired
	}

	// Update last used (async, don't fail on error)
	go func() {
		_ = s.repo.UpdateLastUsed(context.Background(), key.ID)
	}()

	return key, nil
}

// HasScope checks if an API key has a specific scope
func HasScope(key *APIKey, requiredScope string) bool {
	for _, scope := range key.Scopes {
		if scope == requiredScope || scope == "read:all" && isReadScope(requiredScope) || scope == "write:all" {
			return true
		}
	}
	return false
}

func isReadScope(scope string) bool {
	return len(scope) > 5 && scope[:5] == "read:"
}

// generateKey generates a new API key
func generateKey() (string, error) {
	b := make([]byte, KeyLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return KeyPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// validateScopes validates that all scopes are valid
func validateScopes(scopes []string) error {
	validSet := make(map[string]bool)
	for _, s := range ValidScopes {
		validSet[s] = true
	}

	for _, scope := range scopes {
		if !validSet[scope] {
			return ErrInvalidScope
		}
	}

	return nil
}
