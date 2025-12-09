package tenant

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrTenantSlugExists    = errors.New("tenant slug already exists")
	ErrInvalidTenantSlug   = errors.New("invalid tenant slug format")
)

// Tenant represents a tenant/organization
type Tenant struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Slug      string                 `json:"slug"`
	Settings  map[string]interface{} `json:"settings"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// Repository provides tenant data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new tenant repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create creates a new tenant
func (r *Repository) Create(ctx context.Context, tenant *Tenant) error {
	if tenant.ID == uuid.Nil {
		tenant.ID = uuid.New()
	}

	query := `
		INSERT INTO tenants (id, name, slug, settings)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		tenant.Settings,
	).Scan(&tenant.CreatedAt, &tenant.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err, "tenant_slug") {
			return ErrTenantSlugExists
		}
		return err
	}

	return nil
}

// GetByID retrieves a tenant by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	query := `
		SELECT id, name, slug, settings, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`

	tenant := &Tenant{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.Settings,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}

	return tenant, nil
}

// GetBySlug retrieves a tenant by slug
func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Tenant, error) {
	query := `
		SELECT id, name, slug, settings, created_at, updated_at
		FROM tenants
		WHERE slug = $1
	`

	tenant := &Tenant{}
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.Settings,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}

	return tenant, nil
}

// Update updates a tenant
func (r *Repository) Update(ctx context.Context, tenant *Tenant) error {
	query := `
		UPDATE tenants
		SET name = $2, slug = $3, settings = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		tenant.Settings,
	).Scan(&tenant.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTenantNotFound
		}
		if isDuplicateKeyError(err, "tenant_slug") {
			return ErrTenantSlugExists
		}
		return err
	}

	return nil
}

// Delete deletes a tenant
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tenants WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrTenantNotFound
	}

	return nil
}

// Exists checks if a tenant with the given slug exists
func (r *Repository) Exists(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
	return exists, err
}

// isDuplicateKeyError checks if error is a unique constraint violation
func isDuplicateKeyError(err error, constraint string) bool {
	if err == nil {
		return false
	}
	// pgx wraps postgres errors, check for unique_violation (23505)
	errStr := err.Error()
	return contains(errStr, "23505") || contains(errStr, "unique constraint")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
