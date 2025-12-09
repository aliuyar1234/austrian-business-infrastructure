package imports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrImportJobNotFound = errors.New("import job not found")

// ImportJob represents a bulk import job
type ImportJob struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	UserID        uuid.UUID       `json:"user_id"`
	Status        string          `json:"status"`
	TotalRows     *int            `json:"total_rows,omitempty"`
	ProcessedRows int             `json:"processed_rows"`
	SuccessCount  int             `json:"success_count"`
	ErrorCount    int             `json:"error_count"`
	Errors        json.RawMessage `json:"errors,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
}

// ImportError represents an error during import
type ImportError struct {
	RowNumber int    `json:"row_number"`
	Message   string `json:"message"`
}

// Repository handles import job database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new import repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new import job
func (r *Repository) Create(ctx context.Context, job *ImportJob) (*ImportJob, error) {
	query := `
		INSERT INTO import_jobs (tenant_id, user_id, status, total_rows)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		job.TenantID,
		job.UserID,
		"pending",
		job.TotalRows,
	).Scan(&job.ID, &job.CreatedAt)

	if err != nil {
		return nil, err
	}

	job.Status = "pending"
	return job, nil
}

// GetByID retrieves an import job by ID
func (r *Repository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*ImportJob, error) {
	query := `
		SELECT id, tenant_id, user_id, status, total_rows, processed_rows,
		       success_count, error_count, errors, created_at, completed_at
		FROM import_jobs
		WHERE id = $1 AND tenant_id = $2
	`

	var job ImportJob
	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&job.ID,
		&job.TenantID,
		&job.UserID,
		&job.Status,
		&job.TotalRows,
		&job.ProcessedRows,
		&job.SuccessCount,
		&job.ErrorCount,
		&job.Errors,
		&job.CreatedAt,
		&job.CompletedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrImportJobNotFound
	}
	if err != nil {
		return nil, err
	}

	return &job, nil
}

// List retrieves import jobs for a tenant
func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*ImportJob, int, error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM import_jobs WHERE tenant_id = $1`
	if err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch rows
	query := `
		SELECT id, tenant_id, user_id, status, total_rows, processed_rows,
		       success_count, error_count, errors, created_at, completed_at
		FROM import_jobs
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*ImportJob
	for rows.Next() {
		var job ImportJob
		err := rows.Scan(
			&job.ID,
			&job.TenantID,
			&job.UserID,
			&job.Status,
			&job.TotalRows,
			&job.ProcessedRows,
			&job.SuccessCount,
			&job.ErrorCount,
			&job.Errors,
			&job.CreatedAt,
			&job.CompletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, &job)
	}

	return jobs, total, rows.Err()
}

// UpdateProgress updates import job progress
func (r *Repository) UpdateProgress(ctx context.Context, id uuid.UUID, processed, success, errorCount int) error {
	query := `
		UPDATE import_jobs
		SET processed_rows = $1, success_count = $2, error_count = $3
		WHERE id = $4
	`

	_, err := r.db.Exec(ctx, query, processed, success, errorCount, id)
	return err
}

// UpdateStatus updates import job status
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE import_jobs SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, id)
	return err
}

// Complete marks an import job as complete
func (r *Repository) Complete(ctx context.Context, id uuid.UUID, importErrors []ImportError) error {
	errorsJSON, err := json.Marshal(importErrors)
	if err != nil {
		return err
	}

	status := "completed"
	if len(importErrors) > 0 {
		status = "completed" // Still completed, just with errors
	}

	query := `
		UPDATE import_jobs
		SET status = $1, errors = $2, completed_at = NOW()
		WHERE id = $3
	`

	_, err = r.db.Exec(ctx, query, status, errorsJSON, id)
	return err
}

// Fail marks an import job as failed
func (r *Repository) Fail(ctx context.Context, id uuid.UUID, errorMsg string) error {
	errorsJSON, _ := json.Marshal([]ImportError{{RowNumber: 0, Message: errorMsg}})

	query := `
		UPDATE import_jobs
		SET status = 'failed', errors = $1, completed_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, errorsJSON, id)
	return err
}
