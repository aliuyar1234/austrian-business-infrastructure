package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserEmailExists  = errors.New("email already exists for this tenant")
	ErrInvalidRole      = errors.New("invalid user role")
)

// Role represents user roles
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

// ValidRoles contains all valid role values
var ValidRoles = []Role{RoleOwner, RoleAdmin, RoleMember, RoleViewer}

// IsValidRole checks if a role is valid
func IsValidRole(role string) bool {
	for _, r := range ValidRoles {
		if string(r) == role {
			return true
		}
	}
	return false
}

// User represents a user in the system
type User struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	Email           string     `json:"email"`
	PasswordHash    *string    `json:"-"`
	Name            string     `json:"name"`
	Role            Role       `json:"role"`
	EmailVerified   bool       `json:"email_verified"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	OAuthProvider   *string    `json:"oauth_provider,omitempty"`
	OAuthID         *string    `json:"-"`
	AvatarURL       *string    `json:"avatar_url,omitempty"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	IsActive        bool       `json:"is_active"`
	// 2FA fields (Security Layer)
	TOTPSecret        []byte `json:"-"` // Encrypted TOTP secret
	TOTPEnabled       bool   `json:"totp_enabled"`
	RecoveryCodes     []byte `json:"-"` // Encrypted recovery codes
	RecoveryCodesUsed int    `json:"recovery_codes_used,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Repository provides user data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new user repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create creates a new user
func (r *Repository) Create(ctx context.Context, user *User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	if !IsValidRole(string(user.Role)) {
		return ErrInvalidRole
	}

	query := `
		INSERT INTO users (
			id, tenant_id, email, password_hash, name, role,
			email_verified, oauth_provider, oauth_id, avatar_url, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		user.ID,
		user.TenantID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Role,
		user.EmailVerified,
		user.OAuthProvider,
		user.OAuthID,
		user.AvatarURL,
		user.IsActive,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrUserEmailExists
		}
		return err
	}

	return nil
}

// userColumns is the standard column list for user queries
const userColumns = `id, tenant_id, email, password_hash, name, role,
	email_verified, email_verified_at, oauth_provider, oauth_id,
	avatar_url, last_login_at, is_active, totp_secret, totp_enabled,
	recovery_codes, recovery_codes_used, created_at, updated_at`

// GetByID retrieves a user by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	return r.scanUser(r.pool.QueryRow(ctx, query, id))
}

// GetByEmail retrieves a user by email within a tenant
func (r *Repository) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE tenant_id = $1 AND email = $2`
	return r.scanUser(r.pool.QueryRow(ctx, query, tenantID, email))
}

// GetByEmailGlobal retrieves a user by email across all tenants (for login)
func (r *Repository) GetByEmailGlobal(ctx context.Context, email string) (*User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE email = $1 AND is_active = true LIMIT 1`
	return r.scanUser(r.pool.QueryRow(ctx, query, email))
}

// GetByOAuth retrieves a user by OAuth provider and ID
func (r *Repository) GetByOAuth(ctx context.Context, provider, oauthID string) (*User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE oauth_provider = $1 AND oauth_id = $2`
	return r.scanUser(r.pool.QueryRow(ctx, query, provider, oauthID))
}

// ListByTenant returns all users for a tenant
func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE tenant_id = $1 ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user, err := r.scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// Update updates a user
func (r *Repository) Update(ctx context.Context, user *User) error {
	if !IsValidRole(string(user.Role)) {
		return ErrInvalidRole
	}

	query := `
		UPDATE users
		SET email = $2, name = $3, role = $4, email_verified = $5,
			avatar_url = $6, is_active = $7, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.Role,
		user.EmailVerified,
		user.AvatarURL,
		user.IsActive,
	).Scan(&user.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		if isDuplicateKeyError(err) {
			return ErrUserEmailExists
		}
		return err
	}

	return nil
}

// UpdatePassword updates a user's password hash
func (r *Repository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, userID, passwordHash)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *Repository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET last_login_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// VerifyEmail marks a user's email as verified
func (r *Repository) VerifyEmail(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET email_verified = true, email_verified_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Deactivate deactivates a user
func (r *Repository) Deactivate(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// CountOwners counts the number of owners in a tenant
func (r *Repository) CountOwners(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM users
		WHERE tenant_id = $1 AND role = 'owner' AND is_active = true
	`

	var count int
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&count)
	return count, err
}

// scanUser scans a single user from a row
func (r *Repository) scanUser(row pgx.Row) (*User, error) {
	user := &User{}
	err := row.Scan(
		&user.ID,
		&user.TenantID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&user.OAuthProvider,
		&user.OAuthID,
		&user.AvatarURL,
		&user.LastLoginAt,
		&user.IsActive,
		&user.TOTPSecret,
		&user.TOTPEnabled,
		&user.RecoveryCodes,
		&user.RecoveryCodesUsed,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// scanUserFromRows scans a user from rows iterator
func (r *Repository) scanUserFromRows(rows pgx.Rows) (*User, error) {
	user := &User{}
	err := rows.Scan(
		&user.ID,
		&user.TenantID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&user.OAuthProvider,
		&user.OAuthID,
		&user.AvatarURL,
		&user.LastLoginAt,
		&user.IsActive,
		&user.TOTPSecret,
		&user.TOTPEnabled,
		&user.RecoveryCodes,
		&user.RecoveryCodesUsed,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// CreateWithPassword creates a user with a hashed password
func (r *Repository) CreateWithPassword(ctx context.Context, user *User, passwordHash string) (*User, error) {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	if !IsValidRole(string(user.Role)) {
		return nil, ErrInvalidRole
	}

	query := `
		INSERT INTO users (
			id, tenant_id, email, password_hash, name, role,
			email_verified, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		user.ID,
		user.TenantID,
		user.Email,
		passwordHash,
		user.Name,
		user.Role,
		user.EmailVerified,
		user.IsActive,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrUserEmailExists
		}
		return nil, err
	}

	user.PasswordHash = &passwordHash
	return user, nil
}

// isDuplicateKeyError checks if error is a unique constraint violation
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsSubstring(errStr, "23505") || containsSubstring(errStr, "unique constraint")
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============== 2FA Methods (Security Layer) ==============

// SetTOTPSecret sets the encrypted TOTP secret for a user
func (r *Repository) SetTOTPSecret(ctx context.Context, userID uuid.UUID, encryptedSecret []byte) error {
	query := `
		UPDATE users
		SET totp_secret = $2, updated_at = NOW()
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, userID, encryptedSecret)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// EnableTOTP enables TOTP for a user (after they've verified it works)
func (r *Repository) EnableTOTP(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET totp_enabled = true, updated_at = NOW()
		WHERE id = $1 AND totp_secret IS NOT NULL
	`
	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// DisableTOTP disables TOTP for a user (clears secret and recovery codes)
func (r *Repository) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET totp_enabled = false, totp_secret = NULL, recovery_codes = NULL,
			recovery_codes_used = 0, updated_at = NOW()
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// SetRecoveryCodes sets the encrypted recovery codes for a user
func (r *Repository) SetRecoveryCodes(ctx context.Context, userID uuid.UUID, encryptedCodes []byte) error {
	query := `
		UPDATE users
		SET recovery_codes = $2, recovery_codes_used = 0, updated_at = NOW()
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, userID, encryptedCodes)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// IncrementRecoveryCodesUsed increments the count of used recovery codes
func (r *Repository) IncrementRecoveryCodesUsed(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET recovery_codes_used = recovery_codes_used + 1, updated_at = NOW()
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// GetTOTPInfo retrieves just the 2FA-related fields for a user
func (r *Repository) GetTOTPInfo(ctx context.Context, userID uuid.UUID) (totpSecret []byte, totpEnabled bool, recoveryCodes []byte, recoveryCodesUsed int, err error) {
	query := `
		SELECT totp_secret, totp_enabled, recovery_codes, recovery_codes_used
		FROM users
		WHERE id = $1
	`
	err = r.pool.QueryRow(ctx, query, userID).Scan(&totpSecret, &totpEnabled, &recoveryCodes, &recoveryCodesUsed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil, 0, ErrUserNotFound
		}
		return nil, false, nil, 0, err
	}
	return totpSecret, totpEnabled, recoveryCodes, recoveryCodesUsed, nil
}
