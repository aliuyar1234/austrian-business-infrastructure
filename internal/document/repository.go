package document

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pagination limits
const (
	DefaultPageSize = 50
	MaxPageSize     = 500
)

// Repository errors
var (
	ErrDocumentNotFound      = errors.New("document not found")
	ErrDuplicateDocument     = errors.New("document already exists")
	ErrSignedURLNotSupported = errors.New("signed URLs not supported")
	ErrDocumentTooLarge      = errors.New("document exceeds maximum allowed size")
)

// Document represents a stored document
type Document struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	AccountID      uuid.UUID
	ExternalID     string
	Type           string
	Title          string
	Sender         string
	ReceivedAt     time.Time
	ContentHash    string
	StoragePath    string
	FileSize       int
	MimeType       string
	Status         string
	ArchivedAt     *time.Time
	RetentionUntil *time.Time
	Deadline       *time.Time
	Metadata       map[string]interface{}
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// Joined fields for list queries
	AccountName string
	AccountType string
}

// DocumentFilter holds filter criteria for listing documents
type DocumentFilter struct {
	TenantID    uuid.UUID
	AccountID   *uuid.UUID
	AccountIDs  []uuid.UUID
	Status      string
	Type        string
	Search      string
	DateFrom    *time.Time
	DateTo      *time.Time
	Archived    bool
	Limit       int
	Offset      int
	SortBy      string
	SortDesc    bool
}

// DocumentStats holds statistics about documents
type DocumentStats struct {
	TotalCount int
	NewCount   int
	ReadCount  int
	ByType     map[string]int
	ByStatus   map[string]int
	ByAccount  map[uuid.UUID]int
}

// Repository handles document database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new document repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create inserts a new document
func (r *Repository) Create(ctx context.Context, doc *Document) error {
	query := `
		INSERT INTO documents (
			account_id, external_id, type, title, sender, received_at,
			content_hash, storage_path, file_size, mime_type, status, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		doc.AccountID, doc.ExternalID, doc.Type, doc.Title, doc.Sender,
		doc.ReceivedAt, doc.ContentHash, doc.StoragePath, doc.FileSize,
		doc.MimeType, doc.Status, doc.Metadata,
	).Scan(&doc.ID, &doc.CreatedAt, &doc.UpdatedAt)

	if err != nil {
		if isDuplicateError(err) {
			return ErrDuplicateDocument
		}
		return fmt.Errorf("create document: %w", err)
	}

	return nil
}

// GetByID retrieves a document by ID with tenant isolation
func (r *Repository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Document, error) {
	query := `
		SELECT d.id, d.account_id, d.tenant_id, d.external_id, d.type, d.title, d.sender,
			d.received_at, d.content_hash, d.storage_path, d.file_size, d.mime_type,
			d.status, d.archived_at, d.retention_until, d.metadata, d.created_at, d.updated_at,
			a.name as account_name, a.type as account_type
		FROM documents d
		JOIN accounts a ON d.account_id = a.id
		WHERE d.id = $1 AND d.tenant_id = $2
	`

	doc := &Document{}
	var metadata []byte

	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&doc.ID, &doc.AccountID, &doc.TenantID, &doc.ExternalID, &doc.Type, &doc.Title, &doc.Sender,
		&doc.ReceivedAt, &doc.ContentHash, &doc.StoragePath, &doc.FileSize, &doc.MimeType,
		&doc.Status, &doc.ArchivedAt, &doc.RetentionUntil, &metadata, &doc.CreatedAt, &doc.UpdatedAt,
		&doc.AccountName, &doc.AccountType,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("get document: %w", err)
	}

	doc.Metadata = parseMetadata(metadata)
	return doc, nil
}

// GetByExternalID retrieves a document by external ID and account
func (r *Repository) GetByExternalID(ctx context.Context, accountID uuid.UUID, externalID string) (*Document, error) {
	query := `
		SELECT d.id, d.account_id, d.external_id, d.type, d.title, d.sender,
			d.received_at, d.content_hash, d.storage_path, d.file_size, d.mime_type,
			d.status, d.archived_at, d.retention_until, d.metadata, d.created_at, d.updated_at
		FROM documents d
		WHERE d.account_id = $1 AND d.external_id = $2
	`

	doc := &Document{}
	var metadata []byte

	err := r.db.QueryRow(ctx, query, accountID, externalID).Scan(
		&doc.ID, &doc.AccountID, &doc.ExternalID, &doc.Type, &doc.Title, &doc.Sender,
		&doc.ReceivedAt, &doc.ContentHash, &doc.StoragePath, &doc.FileSize, &doc.MimeType,
		&doc.Status, &doc.ArchivedAt, &doc.RetentionUntil, &metadata, &doc.CreatedAt, &doc.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("get document by external ID: %w", err)
	}

	doc.Metadata = parseMetadata(metadata)
	return doc, nil
}

// GetByContentHash checks if a document with the same content hash exists
func (r *Repository) GetByContentHash(ctx context.Context, accountID uuid.UUID, contentHash string) (*Document, error) {
	query := `
		SELECT id, account_id, external_id, type, title, sender,
			received_at, content_hash, storage_path, file_size, mime_type,
			status, archived_at, retention_until, metadata, created_at, updated_at
		FROM documents
		WHERE account_id = $1 AND content_hash = $2
		LIMIT 1
	`

	doc := &Document{}
	var metadata []byte

	err := r.db.QueryRow(ctx, query, accountID, contentHash).Scan(
		&doc.ID, &doc.AccountID, &doc.ExternalID, &doc.Type, &doc.Title, &doc.Sender,
		&doc.ReceivedAt, &doc.ContentHash, &doc.StoragePath, &doc.FileSize, &doc.MimeType,
		&doc.Status, &doc.ArchivedAt, &doc.RetentionUntil, &metadata, &doc.CreatedAt, &doc.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("get document by hash: %w", err)
	}

	doc.Metadata = parseMetadata(metadata)
	return doc, nil
}

// List returns documents matching the filter
func (r *Repository) List(ctx context.Context, filter *DocumentFilter) ([]*Document, int, error) {
	// Build query with filters
	baseQuery := `
		SELECT d.id, d.account_id, d.external_id, d.type, d.title, d.sender,
			d.received_at, d.content_hash, d.storage_path, d.file_size, d.mime_type,
			d.status, d.archived_at, d.retention_until, d.metadata, d.created_at, d.updated_at,
			a.name as account_name, a.type as account_type
		FROM documents d
		JOIN accounts a ON d.account_id = a.id
		WHERE a.tenant_id = $1
	`

	countQuery := `
		SELECT COUNT(*)
		FROM documents d
		JOIN accounts a ON d.account_id = a.id
		WHERE a.tenant_id = $1
	`

	args := []interface{}{filter.TenantID}
	argNum := 2

	// Apply filters
	conditions := ""

	if filter.AccountID != nil {
		conditions += fmt.Sprintf(" AND d.account_id = $%d", argNum)
		args = append(args, *filter.AccountID)
		argNum++
	}

	if len(filter.AccountIDs) > 0 {
		conditions += fmt.Sprintf(" AND d.account_id = ANY($%d)", argNum)
		args = append(args, filter.AccountIDs)
		argNum++
	}

	if filter.Status != "" {
		conditions += fmt.Sprintf(" AND d.status = $%d", argNum)
		args = append(args, filter.Status)
		argNum++
	}

	if filter.Type != "" {
		conditions += fmt.Sprintf(" AND d.type = $%d", argNum)
		args = append(args, filter.Type)
		argNum++
	}

	if filter.DateFrom != nil {
		conditions += fmt.Sprintf(" AND d.received_at >= $%d", argNum)
		args = append(args, *filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		conditions += fmt.Sprintf(" AND d.received_at <= $%d", argNum)
		args = append(args, *filter.DateTo)
		argNum++
	}

	if filter.Archived {
		conditions += " AND d.archived_at IS NOT NULL"
	} else {
		conditions += " AND d.archived_at IS NULL"
	}

	if filter.Search != "" {
		// Use full-text search with GIN index for performance
		// Falls back to ILIKE for single-character searches (FTS minimum is usually 2 chars)
		if len(filter.Search) >= 2 {
			conditions += fmt.Sprintf(" AND to_tsvector('german', COALESCE(d.title, '') || ' ' || COALESCE(d.sender, '')) @@ plainto_tsquery('german', $%d)", argNum)
			args = append(args, filter.Search)
		} else {
			// Fallback for very short searches
			conditions += fmt.Sprintf(" AND (d.title ILIKE $%d OR d.sender ILIKE $%d)", argNum, argNum)
			args = append(args, "%"+filter.Search+"%")
		}
		argNum++
	}

	// Get total count
	var totalCount int
	err := r.db.QueryRow(ctx, countQuery+conditions, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("count documents: %w", err)
	}

	// Apply sorting
	sortColumn := "d.received_at"
	if filter.SortBy != "" {
		switch filter.SortBy {
		case "title":
			sortColumn = "d.title"
		case "type":
			sortColumn = "d.type"
		case "status":
			sortColumn = "d.status"
		case "received_at":
			sortColumn = "d.received_at"
		}
	}

	sortDir := "DESC"
	if !filter.SortDesc {
		sortDir = "ASC"
	}

	baseQuery += conditions + fmt.Sprintf(" ORDER BY %s %s", sortColumn, sortDir)

	// Apply pagination with enforced limits
	limit := filter.Limit
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	baseQuery += fmt.Sprintf(" LIMIT %d", limit)

	if filter.Offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	// Execute query
	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var documents []*Document
	for rows.Next() {
		doc := &Document{}
		var metadata []byte

		err := rows.Scan(
			&doc.ID, &doc.AccountID, &doc.ExternalID, &doc.Type, &doc.Title, &doc.Sender,
			&doc.ReceivedAt, &doc.ContentHash, &doc.StoragePath, &doc.FileSize, &doc.MimeType,
			&doc.Status, &doc.ArchivedAt, &doc.RetentionUntil, &metadata, &doc.CreatedAt, &doc.UpdatedAt,
			&doc.AccountName, &doc.AccountType,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan document: %w", err)
		}

		doc.Metadata = parseMetadata(metadata)
		documents = append(documents, doc)
	}

	return documents, totalCount, nil
}

// UpdateStatus updates the status of a document with tenant isolation
func (r *Repository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	query := `UPDATE documents SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`

	result, err := r.db.Exec(ctx, query, status, id, tenantID)
	if err != nil {
		return fmt.Errorf("update document status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// Archive marks a document as archived with tenant isolation
func (r *Repository) Archive(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `UPDATE documents SET status = 'archived', archived_at = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("archive document: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// BulkArchive archives multiple documents with tenant isolation
func (r *Repository) BulkArchive(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (int, error) {
	query := `UPDATE documents SET status = 'archived', archived_at = NOW(), updated_at = NOW() WHERE id = ANY($1) AND tenant_id = $2`

	result, err := r.db.Exec(ctx, query, ids, tenantID)
	if err != nil {
		return 0, fmt.Errorf("bulk archive documents: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// Delete permanently deletes a document with tenant isolation
func (r *Repository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM documents WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// GetExpired returns documents past their retention date
// MaxExpiredLimit is the maximum number of expired documents to return
const MaxExpiredLimit = 100

func (r *Repository) GetExpired(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Document, int, error) {
	// Enforce limits
	if limit <= 0 || limit > MaxExpiredLimit {
		limit = MaxExpiredLimit
	}
	if offset < 0 {
		offset = 0
	}

	// Get total count first
	countQuery := `
		SELECT COUNT(*) FROM documents d
		JOIN accounts a ON d.account_id = a.id
		WHERE a.tenant_id = $1 AND d.retention_until < NOW()
	`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count expired documents: %w", err)
	}

	query := `
		SELECT d.id, d.account_id, d.external_id, d.type, d.title, d.sender,
			d.received_at, d.content_hash, d.storage_path, d.file_size, d.mime_type,
			d.status, d.archived_at, d.retention_until, d.metadata, d.created_at, d.updated_at
		FROM documents d
		JOIN accounts a ON d.account_id = a.id
		WHERE a.tenant_id = $1 AND d.retention_until < NOW()
		ORDER BY d.retention_until ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get expired documents: %w", err)
	}
	defer rows.Close()

	var documents []*Document
	for rows.Next() {
		doc := &Document{}
		var metadata []byte

		err := rows.Scan(
			&doc.ID, &doc.AccountID, &doc.ExternalID, &doc.Type, &doc.Title, &doc.Sender,
			&doc.ReceivedAt, &doc.ContentHash, &doc.StoragePath, &doc.FileSize, &doc.MimeType,
			&doc.Status, &doc.ArchivedAt, &doc.RetentionUntil, &metadata, &doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan expired document: %w", err)
		}

		doc.Metadata = parseMetadata(metadata)
		documents = append(documents, doc)
	}

	return documents, total, nil
}

// GetStats returns document statistics for a tenant
func (r *Repository) GetStats(ctx context.Context, tenantID uuid.UUID) (*DocumentStats, error) {
	stats := &DocumentStats{
		ByType:    make(map[string]int),
		ByStatus:  make(map[string]int),
		ByAccount: make(map[uuid.UUID]int),
	}

	// Total and status counts
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE d.status = 'new') as new_count,
			COUNT(*) FILTER (WHERE d.status = 'read') as read_count
		FROM documents d
		JOIN accounts a ON d.account_id = a.id
		WHERE a.tenant_id = $1 AND d.archived_at IS NULL
	`

	err := r.db.QueryRow(ctx, query, tenantID).Scan(
		&stats.TotalCount, &stats.NewCount, &stats.ReadCount,
	)
	if err != nil {
		return nil, fmt.Errorf("get document stats: %w", err)
	}

	// Populate ByStatus from counts
	stats.ByStatus["new"] = stats.NewCount
	stats.ByStatus["read"] = stats.ReadCount
	archivedCount := stats.TotalCount - stats.NewCount - stats.ReadCount
	if archivedCount > 0 {
		stats.ByStatus["archived"] = archivedCount
	}

	// By type
	typeQuery := `
		SELECT d.type, COUNT(*)
		FROM documents d
		JOIN accounts a ON d.account_id = a.id
		WHERE a.tenant_id = $1 AND d.archived_at IS NULL
		GROUP BY d.type
	`

	rows, err := r.db.Query(ctx, typeQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get type stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var docType string
		var count int
		if err := rows.Scan(&docType, &count); err != nil {
			return nil, fmt.Errorf("scan type stats: %w", err)
		}
		stats.ByType[docType] = count
	}

	// By account
	accountQuery := `
		SELECT d.account_id, COUNT(*)
		FROM documents d
		JOIN accounts a ON d.account_id = a.id
		WHERE a.tenant_id = $1 AND d.archived_at IS NULL
		GROUP BY d.account_id
	`

	rows2, err := r.db.Query(ctx, accountQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get account stats: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var accountID uuid.UUID
		var count int
		if err := rows2.Scan(&accountID, &count); err != nil {
			return nil, fmt.Errorf("scan account stats: %w", err)
		}
		stats.ByAccount[accountID] = count
	}

	return stats, nil
}

// GetUnreadCount returns count of unread documents per account
func (r *Repository) GetUnreadCount(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]int, error) {
	query := `
		SELECT d.account_id, COUNT(*)
		FROM documents d
		JOIN accounts a ON d.account_id = a.id
		WHERE a.tenant_id = $1 AND d.status = 'new' AND d.archived_at IS NULL
		GROUP BY d.account_id
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get unread counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[uuid.UUID]int)
	for rows.Next() {
		var accountID uuid.UUID
		var count int
		if err := rows.Scan(&accountID, &count); err != nil {
			return nil, fmt.Errorf("scan unread count: %w", err)
		}
		counts[accountID] = count
	}

	return counts, nil
}

// Helper functions

func isDuplicateError(err error) bool {
	return err != nil && (err.Error() == "duplicate key value" ||
		containsString(err.Error(), "duplicate key") ||
		containsString(err.Error(), "unique constraint"))
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr) >= 0
}

func searchString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func parseMetadata(data []byte) map[string]interface{} {
	if len(data) == 0 {
		return make(map[string]interface{})
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		// Return empty map on parse error rather than failing
		// This prevents corrupt metadata from breaking document reads
		return make(map[string]interface{})
	}

	if result == nil {
		return make(map[string]interface{})
	}

	return result
}

// Document status constants
const (
	StatusNew      = "new"
	StatusRead     = "read"
	StatusArchived = "archived"
)

// Document type constants
const (
	TypeBescheid   = "bescheid"
	TypeErsuchen   = "ersuchen"
	TypeMitteilung = "mitteilung"
	TypeMahnung    = "mahnung"
	TypeSonstige   = "sonstige"
)

// TypePriority returns the priority for a document type (lower = higher priority)
func TypePriority(docType string) int {
	switch docType {
	case TypeErsuchen:
		return 1
	case TypeMahnung:
		return 2
	case TypeBescheid:
		return 3
	case TypeMitteilung:
		return 4
	case TypeSonstige:
		return 5
	default:
		return 10
	}
}
