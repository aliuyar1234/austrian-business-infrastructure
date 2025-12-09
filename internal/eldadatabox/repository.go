package eldadatabox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/austrian-business-infrastructure/fo/internal/elda"
)

var (
	ErrDocumentNotFound = errors.New("ELDA document not found")
)

// Repository handles ELDA document database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new ELDA databox repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new ELDA document record
func (r *Repository) Create(ctx context.Context, doc *elda.ELDADocument) error {
	query := `
		INSERT INTO elda_documents (
			id, elda_account_id, elda_document_id, name, category,
			content_type, size, received_at, is_read, storage_path,
			description, synced_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13
		)
		ON CONFLICT (elda_account_id, elda_document_id) DO UPDATE SET
			name = EXCLUDED.name,
			category = EXCLUDED.category,
			is_read = EXCLUDED.is_read,
			synced_at = EXCLUDED.synced_at
	`

	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}
	now := time.Now()
	doc.SyncedAt = now
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}

	_, err := r.db.Exec(ctx, query,
		doc.ID, doc.ELDAAccountID, doc.ELDADocumentID, doc.Name, doc.Category,
		doc.ContentType, doc.Size, doc.ReceivedAt, doc.IsRead, doc.StoragePath,
		doc.Description, doc.SyncedAt, doc.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create ELDA document: %w", err)
	}

	return nil
}

// GetByID retrieves an ELDA document by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*elda.ELDADocument, error) {
	query := `
		SELECT
			id, elda_account_id, elda_document_id, name, category,
			content_type, size, received_at, is_read, storage_path,
			description, synced_at, created_at
		FROM elda_documents
		WHERE id = $1
	`

	doc := &elda.ELDADocument{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&doc.ID, &doc.ELDAAccountID, &doc.ELDADocumentID, &doc.Name, &doc.Category,
		&doc.ContentType, &doc.Size, &doc.ReceivedAt, &doc.IsRead, &doc.StoragePath,
		&doc.Description, &doc.SyncedAt, &doc.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("get ELDA document: %w", err)
	}

	return doc, nil
}

// GetByELDADocumentID retrieves an ELDA document by its ELDA document ID
func (r *Repository) GetByELDADocumentID(ctx context.Context, accountID uuid.UUID, eldaDocID string) (*elda.ELDADocument, error) {
	query := `
		SELECT
			id, elda_account_id, elda_document_id, name, category,
			content_type, size, received_at, is_read, storage_path,
			description, synced_at, created_at
		FROM elda_documents
		WHERE elda_account_id = $1 AND elda_document_id = $2
	`

	doc := &elda.ELDADocument{}
	err := r.db.QueryRow(ctx, query, accountID, eldaDocID).Scan(
		&doc.ID, &doc.ELDAAccountID, &doc.ELDADocumentID, &doc.Name, &doc.Category,
		&doc.ContentType, &doc.Size, &doc.ReceivedAt, &doc.IsRead, &doc.StoragePath,
		&doc.Description, &doc.SyncedAt, &doc.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("get ELDA document by elda_id: %w", err)
	}

	return doc, nil
}

// Update updates an ELDA document record
func (r *Repository) Update(ctx context.Context, doc *elda.ELDADocument) error {
	query := `
		UPDATE elda_documents SET
			is_read = $2,
			storage_path = $3,
			synced_at = $4
		WHERE id = $1
	`

	doc.SyncedAt = time.Now()

	result, err := r.db.Exec(ctx, query, doc.ID, doc.IsRead, doc.StoragePath, doc.SyncedAt)
	if err != nil {
		return fmt.Errorf("update ELDA document: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// ListFilter contains filter options for listing documents
type ListFilter struct {
	ELDAAccountID *uuid.UUID
	Category      string
	Unread        bool
	StartDate     *time.Time
	EndDate       *time.Time
	Limit         int
	Offset        int
}

// List retrieves ELDA documents with filters
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]*elda.ELDADocument, error) {
	query := `
		SELECT
			id, elda_account_id, elda_document_id, name, category,
			content_type, size, received_at, is_read, storage_path,
			description, synced_at, created_at
		FROM elda_documents
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter.ELDAAccountID != nil {
		query += fmt.Sprintf(" AND elda_account_id = $%d", argIndex)
		args = append(args, *filter.ELDAAccountID)
		argIndex++
	}

	if filter.Category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, filter.Category)
		argIndex++
	}

	if filter.Unread {
		query += " AND is_read = false"
	}

	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND received_at >= $%d", argIndex)
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND received_at <= $%d", argIndex)
		args = append(args, *filter.EndDate)
		argIndex++
	}

	query += " ORDER BY received_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list ELDA documents: %w", err)
	}
	defer rows.Close()

	var results []*elda.ELDADocument
	for rows.Next() {
		doc := &elda.ELDADocument{}
		err := rows.Scan(
			&doc.ID, &doc.ELDAAccountID, &doc.ELDADocumentID, &doc.Name, &doc.Category,
			&doc.ContentType, &doc.Size, &doc.ReceivedAt, &doc.IsRead, &doc.StoragePath,
			&doc.Description, &doc.SyncedAt, &doc.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ELDA document: %w", err)
		}
		results = append(results, doc)
	}

	return results, nil
}

// Count returns the count of documents matching the filter
func (r *Repository) Count(ctx context.Context, filter ListFilter) (int, error) {
	query := `SELECT COUNT(*) FROM elda_documents WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filter.ELDAAccountID != nil {
		query += fmt.Sprintf(" AND elda_account_id = $%d", argIndex)
		args = append(args, *filter.ELDAAccountID)
		argIndex++
	}

	if filter.Category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, filter.Category)
		argIndex++
	}

	if filter.Unread {
		query += " AND is_read = false"
	}

	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND received_at >= $%d", argIndex)
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND received_at <= $%d", argIndex)
		args = append(args, *filter.EndDate)
		argIndex++
	}

	var count int
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count ELDA documents: %w", err)
	}

	return count, nil
}

// GetUnreadCount returns the count of unread documents for an account
func (r *Repository) GetUnreadCount(ctx context.Context, accountID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM elda_documents WHERE elda_account_id = $1 AND is_read = false`

	var count int
	err := r.db.QueryRow(ctx, query, accountID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get unread count: %w", err)
	}

	return count, nil
}

// GetLastSyncTime returns the last sync time for an account
func (r *Repository) GetLastSyncTime(ctx context.Context, accountID uuid.UUID) (*time.Time, error) {
	query := `SELECT MAX(synced_at) FROM elda_documents WHERE elda_account_id = $1`

	var lastSync *time.Time
	err := r.db.QueryRow(ctx, query, accountID).Scan(&lastSync)
	if err != nil {
		return nil, fmt.Errorf("get last sync time: %w", err)
	}

	return lastSync, nil
}

// MarkAsRead marks a document as read
func (r *Repository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE elda_documents SET is_read = true WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// SetStoragePath sets the local storage path for a document
func (r *Repository) SetStoragePath(ctx context.Context, id uuid.UUID, storagePath string) error {
	query := `UPDATE elda_documents SET storage_path = $2 WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id, storagePath)
	if err != nil {
		return fmt.Errorf("set storage path: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// Delete deletes an ELDA document
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM elda_documents WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete ELDA document: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}

	return nil
}
