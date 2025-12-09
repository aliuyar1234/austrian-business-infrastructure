package upload

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUploadNotFound = errors.New("upload not found")
	ErrInvalidStatus  = errors.New("invalid upload status")
)

// Status represents the processing status of an upload
type Status string

const (
	StatusNew       Status = "new"
	StatusProcessed Status = "processed"
	StatusArchived  Status = "archived"
)

// ValidStatuses contains all valid status values
var ValidStatuses = []Status{StatusNew, StatusProcessed, StatusArchived}

// IsValidStatus checks if a status is valid
func IsValidStatus(status string) bool {
	for _, s := range ValidStatuses {
		if string(s) == status {
			return true
		}
	}
	return false
}

// Category represents the type of uploaded document
type Category string

const (
	CategoryRechnung    Category = "rechnung"
	CategoryBeleg       Category = "beleg"
	CategoryVertrag     Category = "vertrag"
	CategoryKontoauszug Category = "kontoauszug"
	CategorySonstiges   Category = "sonstiges"
)

// ValidCategories contains all valid category values
var ValidCategories = []Category{CategoryRechnung, CategoryBeleg, CategoryVertrag, CategoryKontoauszug, CategorySonstiges}

// IsValidCategory checks if a category is valid
func IsValidCategory(category string) bool {
	if category == "" {
		return true // Optional field
	}
	for _, c := range ValidCategories {
		if string(c) == category {
			return true
		}
	}
	return false
}

// Upload represents a client upload
type Upload struct {
	ID          uuid.UUID  `json:"id"`
	ClientID    uuid.UUID  `json:"client_id"`
	AccountID   uuid.UUID  `json:"account_id"`

	// File Info
	Filename    string  `json:"filename"`
	StoragePath string  `json:"storage_path"`
	FileSize    int64   `json:"file_size"`
	MimeType    *string `json:"mime_type,omitempty"`
	ContentHash *string `json:"content_hash,omitempty"`

	// Metadata
	Category   *Category  `json:"category,omitempty"`
	Note       *string    `json:"note,omitempty"`
	UploadDate time.Time  `json:"upload_date"`

	// Processing
	Status      Status     `json:"status"`
	ProcessedBy *uuid.UUID `json:"processed_by,omitempty"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	// Joined fields (optional)
	ClientName  string `json:"client_name,omitempty"`
	AccountName string `json:"account_name,omitempty"`
}

// Repository provides upload data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new upload repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// uploadColumns is the standard column list for upload queries
const uploadColumns = `id, client_id, account_id, filename, storage_path, file_size,
	mime_type, content_hash, category, note, upload_date, status,
	processed_by, processed_at, created_at`

// Create creates a new upload record
func (r *Repository) Create(ctx context.Context, upload *Upload) error {
	if upload.ID == uuid.Nil {
		upload.ID = uuid.New()
	}

	if !IsValidStatus(string(upload.Status)) {
		return ErrInvalidStatus
	}

	query := `
		INSERT INTO client_uploads (
			id, client_id, account_id, filename, storage_path, file_size,
			mime_type, content_hash, category, note, upload_date, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at
	`

	err := r.pool.QueryRow(ctx, query,
		upload.ID,
		upload.ClientID,
		upload.AccountID,
		upload.Filename,
		upload.StoragePath,
		upload.FileSize,
		upload.MimeType,
		upload.ContentHash,
		upload.Category,
		upload.Note,
		upload.UploadDate,
		upload.Status,
	).Scan(&upload.CreatedAt)

	return err
}

// GetByID retrieves an upload by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Upload, error) {
	query := `SELECT ` + uploadColumns + ` FROM client_uploads WHERE id = $1`
	return r.scanUpload(r.pool.QueryRow(ctx, query, id))
}

// ListByClient returns all uploads for a client
func (r *Repository) ListByClient(ctx context.Context, clientID uuid.UUID, status *Status, limit, offset int) ([]*Upload, int, error) {
	return r.list(ctx, &clientID, nil, status, limit, offset)
}

// ListByAccount returns all uploads for an account
func (r *Repository) ListByAccount(ctx context.Context, accountID uuid.UUID, status *Status, limit, offset int) ([]*Upload, int, error) {
	return r.list(ctx, nil, &accountID, status, limit, offset)
}

// ListByTenant returns all uploads for a tenant (joins with clients)
func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID, status *Status, limit, offset int) ([]*Upload, int, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM client_uploads cu
		JOIN clients c ON cu.client_id = c.id
		WHERE c.tenant_id = $1
	`
	listQuery := `
		SELECT cu.id, cu.client_id, cu.account_id, cu.filename, cu.storage_path,
			cu.file_size, cu.mime_type, cu.content_hash, cu.category, cu.note,
			cu.upload_date, cu.status, cu.processed_by, cu.processed_at, cu.created_at,
			c.name as client_name, a.name as account_name
		FROM client_uploads cu
		JOIN clients c ON cu.client_id = c.id
		JOIN accounts a ON cu.account_id = a.id
		WHERE c.tenant_id = $1
	`

	args := []interface{}{tenantID}

	if status != nil {
		countQuery += ` AND cu.status = $2`
		listQuery += ` AND cu.status = $2`
		args = append(args, *status)
	}

	listQuery += ` ORDER BY cu.upload_date DESC`

	// Get total count
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if limit > 0 {
		listQuery += ` LIMIT $` + itoa(len(args)+1)
		args = append(args, limit)
	}
	if offset > 0 {
		listQuery += ` OFFSET $` + itoa(len(args)+1)
		args = append(args, offset)
	}

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var uploads []*Upload
	for rows.Next() {
		upload := &Upload{}
		err := rows.Scan(
			&upload.ID, &upload.ClientID, &upload.AccountID, &upload.Filename,
			&upload.StoragePath, &upload.FileSize, &upload.MimeType, &upload.ContentHash,
			&upload.Category, &upload.Note, &upload.UploadDate, &upload.Status,
			&upload.ProcessedBy, &upload.ProcessedAt, &upload.CreatedAt,
			&upload.ClientName, &upload.AccountName,
		)
		if err != nil {
			return nil, 0, err
		}
		uploads = append(uploads, upload)
	}

	return uploads, total, rows.Err()
}

func (r *Repository) list(ctx context.Context, clientID, accountID *uuid.UUID, status *Status, limit, offset int) ([]*Upload, int, error) {
	countQuery := `SELECT COUNT(*) FROM client_uploads WHERE 1=1`
	listQuery := `SELECT ` + uploadColumns + ` FROM client_uploads WHERE 1=1`

	args := []interface{}{}
	argNum := 1

	if clientID != nil {
		countQuery += ` AND client_id = $` + itoa(argNum)
		listQuery += ` AND client_id = $` + itoa(argNum)
		args = append(args, *clientID)
		argNum++
	}

	if accountID != nil {
		countQuery += ` AND account_id = $` + itoa(argNum)
		listQuery += ` AND account_id = $` + itoa(argNum)
		args = append(args, *accountID)
		argNum++
	}

	if status != nil {
		countQuery += ` AND status = $` + itoa(argNum)
		listQuery += ` AND status = $` + itoa(argNum)
		args = append(args, *status)
		argNum++
	}

	listQuery += ` ORDER BY upload_date DESC`

	// Get total count
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if limit > 0 {
		listQuery += ` LIMIT $` + itoa(argNum)
		args = append(args, limit)
		argNum++
	}
	if offset > 0 {
		listQuery += ` OFFSET $` + itoa(argNum)
		args = append(args, offset)
	}

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var uploads []*Upload
	for rows.Next() {
		upload, err := r.scanUploadFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		uploads = append(uploads, upload)
	}

	return uploads, total, rows.Err()
}

// MarkProcessed marks an upload as processed
func (r *Repository) MarkProcessed(ctx context.Context, uploadID, processedBy uuid.UUID) error {
	query := `
		UPDATE client_uploads
		SET status = 'processed', processed_by = $2, processed_at = NOW()
		WHERE id = $1 AND status = 'new'
	`

	result, err := r.pool.Exec(ctx, query, uploadID, processedBy)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUploadNotFound
	}

	return nil
}

// MarkArchived marks an upload as archived
func (r *Repository) MarkArchived(ctx context.Context, uploadID uuid.UUID) error {
	query := `
		UPDATE client_uploads
		SET status = 'archived'
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, uploadID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUploadNotFound
	}

	return nil
}

// Delete deletes an upload record
func (r *Repository) Delete(ctx context.Context, uploadID uuid.UUID) error {
	query := `DELETE FROM client_uploads WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, uploadID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUploadNotFound
	}

	return nil
}

// CountNewByClient counts new uploads for a client
func (r *Repository) CountNewByClient(ctx context.Context, clientID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM client_uploads WHERE client_id = $1 AND status = 'new'`

	var count int
	err := r.pool.QueryRow(ctx, query, clientID).Scan(&count)
	return count, err
}

func (r *Repository) scanUpload(row pgx.Row) (*Upload, error) {
	upload := &Upload{}
	err := row.Scan(
		&upload.ID, &upload.ClientID, &upload.AccountID, &upload.Filename,
		&upload.StoragePath, &upload.FileSize, &upload.MimeType, &upload.ContentHash,
		&upload.Category, &upload.Note, &upload.UploadDate, &upload.Status,
		&upload.ProcessedBy, &upload.ProcessedAt, &upload.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUploadNotFound
		}
		return nil, err
	}

	return upload, nil
}

func (r *Repository) scanUploadFromRows(rows pgx.Rows) (*Upload, error) {
	upload := &Upload{}
	err := rows.Scan(
		&upload.ID, &upload.ClientID, &upload.AccountID, &upload.Filename,
		&upload.StoragePath, &upload.FileSize, &upload.MimeType, &upload.ContentHash,
		&upload.Category, &upload.Note, &upload.UploadDate, &upload.Status,
		&upload.ProcessedBy, &upload.ProcessedAt, &upload.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return upload, nil
}

func itoa(i int) string {
	if i < 0 {
		return "-" + uitoa(uint(-i))
	}
	return uitoa(uint(i))
}

func uitoa(val uint) string {
	if val == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf) - 1
	for val != 0 {
		buf[i] = byte('0' + val%10)
		val /= 10
		i--
	}
	return string(buf[i+1:])
}
