package invitation

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
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationExpired  = errors.New("invitation has expired")
	ErrInvitationUsed     = errors.New("invitation has already been used")
)

// Invitation represents a team invitation
type Invitation struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	TokenHash  string     `json:"-"`
	InvitedBy  uuid.UUID  `json:"invited_by"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Repository provides invitation data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new invitation repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create creates a new invitation
func (r *Repository) Create(ctx context.Context, invitation *Invitation) error {
	if invitation.ID == uuid.Nil {
		invitation.ID = uuid.New()
	}

	query := `
		INSERT INTO invitations (id, tenant_id, email, role, token_hash, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`

	err := r.pool.QueryRow(ctx, query,
		invitation.ID,
		invitation.TenantID,
		invitation.Email,
		invitation.Role,
		invitation.TokenHash,
		invitation.InvitedBy,
		invitation.ExpiresAt,
	).Scan(&invitation.CreatedAt)

	return err
}

// GetByID retrieves an invitation by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Invitation, error) {
	query := `
		SELECT id, tenant_id, email, role, token_hash, invited_by, expires_at, accepted_at, created_at
		FROM invitations
		WHERE id = $1
	`

	invitation := &Invitation{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&invitation.ID,
		&invitation.TenantID,
		&invitation.Email,
		&invitation.Role,
		&invitation.TokenHash,
		&invitation.InvitedBy,
		&invitation.ExpiresAt,
		&invitation.AcceptedAt,
		&invitation.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}

	return invitation, nil
}

// GetByToken retrieves an invitation by token (hashed)
func (r *Repository) GetByToken(ctx context.Context, token string) (*Invitation, error) {
	tokenHash := hashToken(token)

	query := `
		SELECT id, tenant_id, email, role, token_hash, invited_by, expires_at, accepted_at, created_at
		FROM invitations
		WHERE token_hash = $1
	`

	invitation := &Invitation{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&invitation.ID,
		&invitation.TenantID,
		&invitation.Email,
		&invitation.Role,
		&invitation.TokenHash,
		&invitation.InvitedBy,
		&invitation.ExpiresAt,
		&invitation.AcceptedAt,
		&invitation.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}

	return invitation, nil
}

// MarkAccepted marks an invitation as accepted
func (r *Repository) MarkAccepted(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE invitations
		SET accepted_at = NOW()
		WHERE id = $1 AND accepted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrInvitationUsed
	}

	return nil
}

// ListByTenant returns all invitations for a tenant
func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*Invitation, error) {
	query := `
		SELECT id, tenant_id, email, role, token_hash, invited_by, expires_at, accepted_at, created_at
		FROM invitations
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []*Invitation
	for rows.Next() {
		inv := &Invitation{}
		if err := rows.Scan(
			&inv.ID,
			&inv.TenantID,
			&inv.Email,
			&inv.Role,
			&inv.TokenHash,
			&inv.InvitedBy,
			&inv.ExpiresAt,
			&inv.AcceptedAt,
			&inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		invitations = append(invitations, inv)
	}

	return invitations, rows.Err()
}

// ListPendingByEmail returns pending invitations for an email
func (r *Repository) ListPendingByEmail(ctx context.Context, email string) ([]*Invitation, error) {
	query := `
		SELECT id, tenant_id, email, role, token_hash, invited_by, expires_at, accepted_at, created_at
		FROM invitations
		WHERE email = $1 AND accepted_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []*Invitation
	for rows.Next() {
		inv := &Invitation{}
		if err := rows.Scan(
			&inv.ID,
			&inv.TenantID,
			&inv.Email,
			&inv.Role,
			&inv.TokenHash,
			&inv.InvitedBy,
			&inv.ExpiresAt,
			&inv.AcceptedAt,
			&inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		invitations = append(invitations, inv)
	}

	return invitations, rows.Err()
}

// Delete deletes an invitation
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM invitations WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrInvitationNotFound
	}

	return nil
}

// DeleteExpired removes expired invitations
func (r *Repository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM invitations WHERE expires_at < NOW() AND accepted_at IS NULL`
	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// hashToken creates a SHA-256 hash of a token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
