package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAPIKeyNotFound = errors.New("API key not found")
	ErrAPIKeyExpired  = errors.New("API key has expired")
	ErrAPIKeyInactive = errors.New("API key is inactive")
)

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"`
	KeyPrefix  string     `json:"key_prefix"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Repository provides API key data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new API key repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create creates a new API key
func (r *Repository) Create(ctx context.Context, key *APIKey) error {
	if key.ID == uuid.Nil {
		key.ID = uuid.New()
	}

	query := `
		INSERT INTO api_keys (id, user_id, tenant_id, name, key_hash, key_prefix, scopes, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at
	`

	return r.pool.QueryRow(ctx, query,
		key.ID,
		key.UserID,
		key.TenantID,
		key.Name,
		key.KeyHash,
		key.KeyPrefix,
		key.Scopes,
		key.ExpiresAt,
		key.IsActive,
	).Scan(&key.CreatedAt)
}

// GetByID retrieves an API key by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*APIKey, error) {
	query := `
		SELECT id, user_id, tenant_id, name, key_hash, key_prefix, scopes, expires_at, last_used_at, is_active, created_at
		FROM api_keys
		WHERE id = $1
	`

	key := &APIKey{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&key.ID,
		&key.UserID,
		&key.TenantID,
		&key.Name,
		&key.KeyHash,
		&key.KeyPrefix,
		&key.Scopes,
		&key.ExpiresAt,
		&key.LastUsedAt,
		&key.IsActive,
		&key.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}

	return key, nil
}

// GetByKeyHash retrieves an API key by its hash
func (r *Repository) GetByKeyHash(ctx context.Context, keyHash string) (*APIKey, error) {
	query := `
		SELECT id, user_id, tenant_id, name, key_hash, key_prefix, scopes, expires_at, last_used_at, is_active, created_at
		FROM api_keys
		WHERE key_hash = $1
	`

	key := &APIKey{}
	err := r.pool.QueryRow(ctx, query, keyHash).Scan(
		&key.ID,
		&key.UserID,
		&key.TenantID,
		&key.Name,
		&key.KeyHash,
		&key.KeyPrefix,
		&key.Scopes,
		&key.ExpiresAt,
		&key.LastUsedAt,
		&key.IsActive,
		&key.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}

	return key, nil
}

// GetByPrefix retrieves an API key by its prefix (for lookup before full validation)
func (r *Repository) GetByPrefix(ctx context.Context, prefix string) (*APIKey, error) {
	query := `
		SELECT id, user_id, tenant_id, name, key_hash, key_prefix, scopes, expires_at, last_used_at, is_active, created_at
		FROM api_keys
		WHERE key_prefix = $1 AND is_active = true
	`

	key := &APIKey{}
	err := r.pool.QueryRow(ctx, query, prefix).Scan(
		&key.ID,
		&key.UserID,
		&key.TenantID,
		&key.Name,
		&key.KeyHash,
		&key.KeyPrefix,
		&key.Scopes,
		&key.ExpiresAt,
		&key.LastUsedAt,
		&key.IsActive,
		&key.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}

	return key, nil
}

// ListByUser returns all API keys for a user
func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*APIKey, error) {
	query := `
		SELECT id, user_id, tenant_id, name, key_hash, key_prefix, scopes, expires_at, last_used_at, is_active, created_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		key := &APIKey{}
		if err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.TenantID,
			&key.Name,
			&key.KeyHash,
			&key.KeyPrefix,
			&key.Scopes,
			&key.ExpiresAt,
			&key.LastUsedAt,
			&key.IsActive,
			&key.CreatedAt,
		); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, rows.Err()
}

// ListByTenant returns all API keys for a tenant
func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*APIKey, error) {
	query := `
		SELECT id, user_id, tenant_id, name, key_hash, key_prefix, scopes, expires_at, last_used_at, is_active, created_at
		FROM api_keys
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		key := &APIKey{}
		if err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.TenantID,
			&key.Name,
			&key.KeyHash,
			&key.KeyPrefix,
			&key.Scopes,
			&key.ExpiresAt,
			&key.LastUsedAt,
			&key.IsActive,
			&key.CreatedAt,
		); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, rows.Err()
}

// Delete deletes (revokes) an API key
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

// Deactivate deactivates an API key (soft delete)
func (r *Repository) Deactivate(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET is_active = false WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

// UpdateLastUsed updates the last_used_at timestamp
func (r *Repository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// HashKey creates a SHA-256 hash of an API key
func HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
