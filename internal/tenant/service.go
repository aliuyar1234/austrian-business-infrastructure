package tenant

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/austrian-business-infrastructure/fo/internal/user"
	"github.com/austrian-business-infrastructure/fo/pkg/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// CreateTenantInput contains input for creating a tenant
type CreateTenantInput struct {
	TenantName string
	TenantSlug string
	OwnerName  string
	OwnerEmail string
	Password   string
}

// CreateTenantResult contains the result of tenant creation
type CreateTenantResult struct {
	Tenant *Tenant
	Owner  *user.User
}

// Service provides tenant business logic
type Service struct {
	tenantRepo *Repository
	userRepo   *user.Repository
	pool       *pgxpool.Pool
}

// NewService creates a new tenant service
func NewService(pool *pgxpool.Pool, tenantRepo *Repository, userRepo *user.Repository) *Service {
	return &Service{
		tenantRepo: tenantRepo,
		userRepo:   userRepo,
		pool:       pool,
	}
}

// CreateWithOwner creates a new tenant with an owner user in a transaction
func (s *Service) CreateWithOwner(ctx context.Context, input *CreateTenantInput) (*CreateTenantResult, error) {
	// Validate tenant slug
	if !isValidSlug(input.TenantSlug) {
		return nil, ErrInvalidTenantSlug
	}

	// Validate password
	if err := crypto.ValidatePassword(input.Password, nil); err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	// Hash password
	passwordHash, err := crypto.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create tenant
	tenant := &Tenant{
		ID:       uuid.New(),
		Name:     input.TenantName,
		Slug:     normalizeSlug(input.TenantSlug),
		Settings: make(map[string]interface{}),
	}

	// Insert tenant using transaction
	tenantQuery := `
		INSERT INTO tenants (id, name, slug, settings)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`
	err = tx.QueryRow(ctx, tenantQuery,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		tenant.Settings,
	).Scan(&tenant.CreatedAt, &tenant.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err, "tenant_slug") {
			return nil, ErrTenantSlugExists
		}
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Create owner user
	owner := &user.User{
		ID:           uuid.New(),
		TenantID:     tenant.ID,
		Email:        strings.ToLower(input.OwnerEmail),
		PasswordHash: &passwordHash,
		Name:         input.OwnerName,
		Role:         user.RoleOwner,
		IsActive:     true,
	}

	// Insert user using transaction
	userQuery := `
		INSERT INTO users (
			id, tenant_id, email, password_hash, name, role,
			email_verified, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`
	err = tx.QueryRow(ctx, userQuery,
		owner.ID,
		owner.TenantID,
		owner.Email,
		owner.PasswordHash,
		owner.Name,
		owner.Role,
		owner.EmailVerified,
		owner.IsActive,
	).Scan(&owner.CreatedAt, &owner.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create owner: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &CreateTenantResult{
		Tenant: tenant,
		Owner:  owner,
	}, nil
}

// GetByID retrieves a tenant by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	return s.tenantRepo.GetByID(ctx, id)
}

// GetBySlug retrieves a tenant by slug
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Tenant, error) {
	return s.tenantRepo.GetBySlug(ctx, normalizeSlug(slug))
}

// Update updates tenant information
func (s *Service) Update(ctx context.Context, tenant *Tenant) error {
	if !isValidSlug(tenant.Slug) {
		return ErrInvalidTenantSlug
	}
	tenant.Slug = normalizeSlug(tenant.Slug)
	return s.tenantRepo.Update(ctx, tenant)
}

// isValidSlug validates tenant slug format
func isValidSlug(slug string) bool {
	if len(slug) == 0 || len(slug) > 100 {
		return false
	}
	return slugRegex.MatchString(strings.ToLower(slug))
}

// normalizeSlug converts slug to lowercase
func normalizeSlug(slug string) string {
	return strings.ToLower(slug)
}
