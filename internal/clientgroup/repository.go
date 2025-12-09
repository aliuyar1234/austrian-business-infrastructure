package clientgroup

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrGroupNotFound  = errors.New("client group not found")
	ErrMemberExists   = errors.New("client already in group")
	ErrMemberNotFound = errors.New("member not found in group")
)

// ClientGroup represents a group of clients
type ClientGroup struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Color       *string   `json:"color,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Joined fields
	MemberCount int `json:"member_count,omitempty"`
}

// GroupMember represents a client's membership in a group
type GroupMember struct {
	GroupID   uuid.UUID `json:"group_id"`
	ClientID  uuid.UUID `json:"client_id"`
	AddedAt   time.Time `json:"added_at"`

	// Joined fields
	ClientName  string `json:"client_name,omitempty"`
	ClientEmail string `json:"client_email,omitempty"`
}

// Repository provides client group data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new client group repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create creates a new client group
func (r *Repository) Create(ctx context.Context, group *ClientGroup) error {
	if group.ID == uuid.Nil {
		group.ID = uuid.New()
	}

	query := `
		INSERT INTO client_groups (id, tenant_id, name, description, color)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		group.ID,
		group.TenantID,
		group.Name,
		group.Description,
		group.Color,
	).Scan(&group.CreatedAt, &group.UpdatedAt)

	return err
}

// GetByID retrieves a group by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*ClientGroup, error) {
	query := `
		SELECT g.id, g.tenant_id, g.name, g.description, g.color, g.created_at, g.updated_at,
			(SELECT COUNT(*) FROM client_group_members WHERE group_id = g.id) as member_count
		FROM client_groups g
		WHERE g.id = $1
	`

	group := &ClientGroup{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&group.ID, &group.TenantID, &group.Name, &group.Description,
		&group.Color, &group.CreatedAt, &group.UpdatedAt, &group.MemberCount,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}

	return group, nil
}

// ListByTenant returns all groups for a tenant
func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*ClientGroup, error) {
	query := `
		SELECT g.id, g.tenant_id, g.name, g.description, g.color, g.created_at, g.updated_at,
			(SELECT COUNT(*) FROM client_group_members WHERE group_id = g.id) as member_count
		FROM client_groups g
		WHERE g.tenant_id = $1
		ORDER BY g.name ASC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*ClientGroup
	for rows.Next() {
		group := &ClientGroup{}
		err := rows.Scan(
			&group.ID, &group.TenantID, &group.Name, &group.Description,
			&group.Color, &group.CreatedAt, &group.UpdatedAt, &group.MemberCount,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

// Update updates a group
func (r *Repository) Update(ctx context.Context, group *ClientGroup) error {
	query := `
		UPDATE client_groups
		SET name = $2, description = $3, color = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		group.ID,
		group.Name,
		group.Description,
		group.Color,
	).Scan(&group.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGroupNotFound
		}
		return err
	}

	return nil
}

// Delete deletes a group
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	// Members are deleted via CASCADE
	query := `DELETE FROM client_groups WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrGroupNotFound
	}

	return nil
}

// AddMember adds a client to a group
func (r *Repository) AddMember(ctx context.Context, groupID, clientID uuid.UUID) error {
	query := `
		INSERT INTO client_group_members (group_id, client_id)
		VALUES ($1, $2)
		ON CONFLICT (group_id, client_id) DO NOTHING
	`

	result, err := r.pool.Exec(ctx, query, groupID, clientID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrMemberExists
	}

	return nil
}

// RemoveMember removes a client from a group
func (r *Repository) RemoveMember(ctx context.Context, groupID, clientID uuid.UUID) error {
	query := `DELETE FROM client_group_members WHERE group_id = $1 AND client_id = $2`

	result, err := r.pool.Exec(ctx, query, groupID, clientID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}

	return nil
}

// SetMembers replaces all members of a group
func (r *Repository) SetMembers(ctx context.Context, groupID uuid.UUID, clientIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Remove all existing members
	_, err = tx.Exec(ctx, `DELETE FROM client_group_members WHERE group_id = $1`, groupID)
	if err != nil {
		return err
	}

	// Add new members
	for _, clientID := range clientIDs {
		_, err = tx.Exec(ctx,
			`INSERT INTO client_group_members (group_id, client_id) VALUES ($1, $2)`,
			groupID, clientID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// ListMembers returns all members of a group
func (r *Repository) ListMembers(ctx context.Context, groupID uuid.UUID) ([]*GroupMember, error) {
	query := `
		SELECT m.group_id, m.client_id, m.added_at, c.name as client_name, c.email as client_email
		FROM client_group_members m
		JOIN clients c ON m.client_id = c.id
		WHERE m.group_id = $1
		ORDER BY c.name ASC
	`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*GroupMember
	for rows.Next() {
		member := &GroupMember{}
		err := rows.Scan(
			&member.GroupID, &member.ClientID, &member.AddedAt,
			&member.ClientName, &member.ClientEmail,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, rows.Err()
}

// ListGroupsForClient returns all groups a client belongs to
func (r *Repository) ListGroupsForClient(ctx context.Context, clientID uuid.UUID) ([]*ClientGroup, error) {
	query := `
		SELECT g.id, g.tenant_id, g.name, g.description, g.color, g.created_at, g.updated_at
		FROM client_groups g
		JOIN client_group_members m ON g.id = m.group_id
		WHERE m.client_id = $1
		ORDER BY g.name ASC
	`

	rows, err := r.pool.Query(ctx, query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*ClientGroup
	for rows.Next() {
		group := &ClientGroup{}
		err := rows.Scan(
			&group.ID, &group.TenantID, &group.Name, &group.Description,
			&group.Color, &group.CreatedAt, &group.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

// GetMemberClientIDs returns all client IDs in a group
func (r *Repository) GetMemberClientIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT client_id FROM client_group_members WHERE group_id = $1`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clientIDs []uuid.UUID
	for rows.Next() {
		var clientID uuid.UUID
		if err := rows.Scan(&clientID); err != nil {
			return nil, err
		}
		clientIDs = append(clientIDs, clientID)
	}

	return clientIDs, rows.Err()
}
