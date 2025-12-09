package share

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrShareNotFound    = errors.New("share not found")
	ErrShareExists      = errors.New("document already shared with this client")
)

// DocumentShare represents a document shared with a client
type DocumentShare struct {
	ID            uuid.UUID  `json:"id"`
	DocumentID    uuid.UUID  `json:"document_id"`
	ClientID      uuid.UUID  `json:"client_id"`

	// Share Info
	SharedBy      uuid.UUID  `json:"shared_by"`
	SharedAt      time.Time  `json:"shared_at"`

	// Access
	CanDownload   bool       `json:"can_download"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`

	// Tracking
	FirstViewedAt *time.Time `json:"first_viewed_at,omitempty"`
	ViewCount     int        `json:"view_count"`

	// Joined fields
	DocumentTitle string `json:"document_title,omitempty"`
	DocumentType  string `json:"document_type,omitempty"`
	AccountName   string `json:"account_name,omitempty"`
}

// Repository provides document share data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new share repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// shareColumns is the standard column list
const shareColumns = `id, document_id, client_id, shared_by, shared_at,
	can_download, expires_at, first_viewed_at, view_count`

// Create creates a new share record
func (r *Repository) Create(ctx context.Context, share *DocumentShare) error {
	if share.ID == uuid.Nil {
		share.ID = uuid.New()
	}

	query := `
		INSERT INTO document_shares (
			id, document_id, client_id, shared_by, can_download, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING shared_at
	`

	err := r.pool.QueryRow(ctx, query,
		share.ID,
		share.DocumentID,
		share.ClientID,
		share.SharedBy,
		share.CanDownload,
		share.ExpiresAt,
	).Scan(&share.SharedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrShareExists
		}
		return err
	}

	return nil
}

// GetByID retrieves a share by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*DocumentShare, error) {
	query := `SELECT ` + shareColumns + ` FROM document_shares WHERE id = $1`
	return r.scanShare(r.pool.QueryRow(ctx, query, id))
}

// GetByDocumentAndClient retrieves a share by document and client
func (r *Repository) GetByDocumentAndClient(ctx context.Context, documentID, clientID uuid.UUID) (*DocumentShare, error) {
	query := `SELECT ` + shareColumns + ` FROM document_shares WHERE document_id = $1 AND client_id = $2`
	return r.scanShare(r.pool.QueryRow(ctx, query, documentID, clientID))
}

// ListByClient returns all shares for a client (documents shared with them)
func (r *Repository) ListByClient(ctx context.Context, clientID uuid.UUID, limit, offset int) ([]*DocumentShare, int, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM document_shares ds
		WHERE ds.client_id = $1
		AND (ds.expires_at IS NULL OR ds.expires_at > NOW())
	`

	listQuery := `
		SELECT ds.id, ds.document_id, ds.client_id, ds.shared_by, ds.shared_at,
			ds.can_download, ds.expires_at, ds.first_viewed_at, ds.view_count,
			d.title as document_title, d.type as document_type, a.name as account_name
		FROM document_shares ds
		JOIN documents d ON ds.document_id = d.id
		JOIN accounts a ON d.account_id = a.id
		WHERE ds.client_id = $1
		AND (ds.expires_at IS NULL OR ds.expires_at > NOW())
		ORDER BY ds.shared_at DESC
	`

	var total int
	err := r.pool.QueryRow(ctx, countQuery, clientID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args := []interface{}{clientID}
	if limit > 0 {
		listQuery += ` LIMIT $2`
		args = append(args, limit)
	}
	if offset > 0 {
		listQuery += ` OFFSET $3`
		args = append(args, offset)
	}

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shares []*DocumentShare
	for rows.Next() {
		share := &DocumentShare{}
		err := rows.Scan(
			&share.ID, &share.DocumentID, &share.ClientID, &share.SharedBy,
			&share.SharedAt, &share.CanDownload, &share.ExpiresAt,
			&share.FirstViewedAt, &share.ViewCount,
			&share.DocumentTitle, &share.DocumentType, &share.AccountName,
		)
		if err != nil {
			return nil, 0, err
		}
		shares = append(shares, share)
	}

	return shares, total, rows.Err()
}

// ListByDocument returns all shares for a document
func (r *Repository) ListByDocument(ctx context.Context, documentID uuid.UUID) ([]*DocumentShare, error) {
	query := `
		SELECT ds.id, ds.document_id, ds.client_id, ds.shared_by, ds.shared_at,
			ds.can_download, ds.expires_at, ds.first_viewed_at, ds.view_count,
			c.name as client_name
		FROM document_shares ds
		JOIN clients c ON ds.client_id = c.id
		WHERE ds.document_id = $1
		ORDER BY ds.shared_at DESC
	`

	rows, err := r.pool.Query(ctx, query, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*DocumentShare
	for rows.Next() {
		share := &DocumentShare{}
		var clientName string
		err := rows.Scan(
			&share.ID, &share.DocumentID, &share.ClientID, &share.SharedBy,
			&share.SharedAt, &share.CanDownload, &share.ExpiresAt,
			&share.FirstViewedAt, &share.ViewCount,
			&clientName,
		)
		if err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}

	return shares, rows.Err()
}

// Delete removes a share
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM document_shares WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrShareNotFound
	}

	return nil
}

// RecordView records a document view
func (r *Repository) RecordView(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE document_shares
		SET view_count = view_count + 1,
			first_viewed_at = COALESCE(first_viewed_at, NOW())
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrShareNotFound
	}

	return nil
}

func (r *Repository) scanShare(row pgx.Row) (*DocumentShare, error) {
	share := &DocumentShare{}
	err := row.Scan(
		&share.ID, &share.DocumentID, &share.ClientID, &share.SharedBy,
		&share.SharedAt, &share.CanDownload, &share.ExpiresAt,
		&share.FirstViewedAt, &share.ViewCount,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrShareNotFound
		}
		return nil, err
	}

	return share, nil
}

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
