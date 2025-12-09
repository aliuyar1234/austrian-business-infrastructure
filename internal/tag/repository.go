package tag

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTagNotFound      = errors.New("tag not found")
	ErrTagAlreadyExists = errors.New("tag with this name already exists")
)

// Tag represents a user-defined label for organizing accounts
type Tag struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Color     *string   `json:"color,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Repository handles tag database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new tag repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new tag
func (r *Repository) Create(ctx context.Context, tag *Tag) (*Tag, error) {
	query := `
		INSERT INTO tags (tenant_id, name, color)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		tag.TenantID,
		tag.Name,
		tag.Color,
	).Scan(&tag.ID, &tag.CreatedAt)

	if err != nil {
		// Check for unique constraint violation
		if isDuplicateError(err) {
			return nil, ErrTagAlreadyExists
		}
		return nil, err
	}

	return tag, nil
}

// GetByID retrieves a tag by ID
func (r *Repository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*Tag, error) {
	query := `
		SELECT id, tenant_id, name, color, created_at
		FROM tags
		WHERE id = $1 AND tenant_id = $2
	`

	var tag Tag
	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&tag.ID,
		&tag.TenantID,
		&tag.Name,
		&tag.Color,
		&tag.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTagNotFound
	}
	if err != nil {
		return nil, err
	}

	return &tag, nil
}

// List retrieves all tags for a tenant
func (r *Repository) List(ctx context.Context, tenantID uuid.UUID) ([]*Tag, error) {
	query := `
		SELECT id, tenant_id, name, color, created_at
		FROM tags
		WHERE tenant_id = $1
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*Tag
	for rows.Next() {
		var tag Tag
		err := rows.Scan(
			&tag.ID,
			&tag.TenantID,
			&tag.Name,
			&tag.Color,
			&tag.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}

	return tags, rows.Err()
}

// Update updates a tag
func (r *Repository) Update(ctx context.Context, tag *Tag) error {
	query := `
		UPDATE tags
		SET name = $1, color = $2
		WHERE id = $3 AND tenant_id = $4
	`

	result, err := r.db.Exec(ctx, query, tag.Name, tag.Color, tag.ID, tag.TenantID)
	if err != nil {
		if isDuplicateError(err) {
			return ErrTagAlreadyExists
		}
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrTagNotFound
	}

	return nil
}

// Delete deletes a tag
func (r *Repository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `DELETE FROM tags WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrTagNotFound
	}

	return nil
}

// AddToAccount assigns a tag to an account
func (r *Repository) AddToAccount(ctx context.Context, accountID, tagID uuid.UUID) error {
	query := `
		INSERT INTO account_tags (account_id, tag_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, accountID, tagID)
	return err
}

// RemoveFromAccount removes a tag from an account
func (r *Repository) RemoveFromAccount(ctx context.Context, accountID, tagID uuid.UUID) error {
	query := `DELETE FROM account_tags WHERE account_id = $1 AND tag_id = $2`
	_, err := r.db.Exec(ctx, query, accountID, tagID)
	return err
}

// GetAccountTags retrieves tags for an account
func (r *Repository) GetAccountTags(ctx context.Context, accountID uuid.UUID) ([]*Tag, error) {
	query := `
		SELECT t.id, t.tenant_id, t.name, t.color, t.created_at
		FROM tags t
		JOIN account_tags at ON at.tag_id = t.id
		WHERE at.account_id = $1
		ORDER BY t.name ASC
	`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*Tag
	for rows.Next() {
		var tag Tag
		err := rows.Scan(
			&tag.ID,
			&tag.TenantID,
			&tag.Name,
			&tag.Color,
			&tag.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}

	return tags, rows.Err()
}

// SetAccountTags replaces all tags for an account
func (r *Repository) SetAccountTags(ctx context.Context, accountID uuid.UUID, tagIDs []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Remove all existing tags
	_, err = tx.Exec(ctx, `DELETE FROM account_tags WHERE account_id = $1`, accountID)
	if err != nil {
		return err
	}

	// Add new tags
	for _, tagID := range tagIDs {
		_, err = tx.Exec(ctx, `INSERT INTO account_tags (account_id, tag_id) VALUES ($1, $2)`, accountID, tagID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func isDuplicateError(err error) bool {
	// Check for PostgreSQL unique violation error
	return err != nil && (contains(err.Error(), "duplicate key") || contains(err.Error(), "unique constraint"))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
