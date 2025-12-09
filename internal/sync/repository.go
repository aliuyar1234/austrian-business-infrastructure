package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository errors
var (
	ErrSyncJobNotFound = errors.New("sync job not found")
)

// SyncJob represents a databox synchronization job
type SyncJob struct {
	ID               uuid.UUID
	TenantID         uuid.UUID
	AccountID        *uuid.UUID
	Status           string
	JobType          string
	DocumentsFound   int
	DocumentsNew     int
	DocumentsSkipped int
	ErrorMessage     string
	StartedAt        *time.Time
	CompletedAt      *time.Time
	CreatedAt        time.Time

	// Joined field
	AccountName string
}

// SyncJobFilter holds filter criteria for listing sync jobs
type SyncJobFilter struct {
	TenantID  uuid.UUID
	AccountID *uuid.UUID
	Status    string
	Limit     int
	Offset    int
}

// SyncJobRepository handles sync job database operations
type SyncJobRepository struct {
	db *pgxpool.Pool
}

// NewSyncJobRepository creates a new sync job repository
func NewSyncJobRepository(db *pgxpool.Pool) *SyncJobRepository {
	return &SyncJobRepository{db: db}
}

// Create inserts a new sync job
func (r *SyncJobRepository) Create(ctx context.Context, job *SyncJob) error {
	query := `
		INSERT INTO sync_jobs (tenant_id, account_id, status, job_type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		job.TenantID, job.AccountID, job.Status, job.JobType,
	).Scan(&job.ID, &job.CreatedAt)

	if err != nil {
		return fmt.Errorf("create sync job: %w", err)
	}

	return nil
}

// GetByID retrieves a sync job by ID
func (r *SyncJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*SyncJob, error) {
	query := `
		SELECT s.id, s.tenant_id, s.account_id, s.status, s.job_type,
			s.documents_found, s.documents_new, s.documents_skipped,
			s.error_message, s.started_at, s.completed_at, s.created_at,
			COALESCE(a.name, '') as account_name
		FROM sync_jobs s
		LEFT JOIN accounts a ON s.account_id = a.id
		WHERE s.id = $1
	`

	job := &SyncJob{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.TenantID, &job.AccountID, &job.Status, &job.JobType,
		&job.DocumentsFound, &job.DocumentsNew, &job.DocumentsSkipped,
		&job.ErrorMessage, &job.StartedAt, &job.CompletedAt, &job.CreatedAt,
		&job.AccountName,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSyncJobNotFound
		}
		return nil, fmt.Errorf("get sync job: %w", err)
	}

	return job, nil
}

// List returns sync jobs matching the filter
func (r *SyncJobRepository) List(ctx context.Context, filter *SyncJobFilter) ([]*SyncJob, int, error) {
	baseQuery := `
		SELECT s.id, s.tenant_id, s.account_id, s.status, s.job_type,
			s.documents_found, s.documents_new, s.documents_skipped,
			s.error_message, s.started_at, s.completed_at, s.created_at,
			COALESCE(a.name, '') as account_name
		FROM sync_jobs s
		LEFT JOIN accounts a ON s.account_id = a.id
		WHERE s.tenant_id = $1
	`

	countQuery := `
		SELECT COUNT(*)
		FROM sync_jobs s
		WHERE s.tenant_id = $1
	`

	args := []interface{}{filter.TenantID}
	argNum := 2
	conditions := ""

	if filter.AccountID != nil {
		conditions += fmt.Sprintf(" AND s.account_id = $%d", argNum)
		args = append(args, *filter.AccountID)
		argNum++
	}

	if filter.Status != "" {
		conditions += fmt.Sprintf(" AND s.status = $%d", argNum)
		args = append(args, filter.Status)
		argNum++
	}

	// Get total count
	var totalCount int
	err := r.db.QueryRow(ctx, countQuery+conditions, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("count sync jobs: %w", err)
	}

	// Apply ordering and pagination
	baseQuery += conditions + " ORDER BY s.created_at DESC"

	if filter.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list sync jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*SyncJob
	for rows.Next() {
		job := &SyncJob{}
		err := rows.Scan(
			&job.ID, &job.TenantID, &job.AccountID, &job.Status, &job.JobType,
			&job.DocumentsFound, &job.DocumentsNew, &job.DocumentsSkipped,
			&job.ErrorMessage, &job.StartedAt, &job.CompletedAt, &job.CreatedAt,
			&job.AccountName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan sync job: %w", err)
		}
		jobs = append(jobs, job)
	}

	return jobs, totalCount, nil
}

// Start marks a sync job as running
func (r *SyncJobRepository) Start(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE sync_jobs SET status = 'running', started_at = NOW() WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("start sync job: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSyncJobNotFound
	}

	return nil
}

// Complete marks a sync job as completed
func (r *SyncJobRepository) Complete(ctx context.Context, id uuid.UUID, found, new, skipped int) error {
	query := `
		UPDATE sync_jobs
		SET status = 'completed', completed_at = NOW(),
			documents_found = $2, documents_new = $3, documents_skipped = $4
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, found, new, skipped)
	if err != nil {
		return fmt.Errorf("complete sync job: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSyncJobNotFound
	}

	return nil
}

// Fail marks a sync job as failed
func (r *SyncJobRepository) Fail(ctx context.Context, id uuid.UUID, errorMsg string) error {
	query := `UPDATE sync_jobs SET status = 'failed', completed_at = NOW(), error_message = $2 WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id, errorMsg)
	if err != nil {
		return fmt.Errorf("fail sync job: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSyncJobNotFound
	}

	return nil
}

// UpdateProgress updates the document counts during sync
func (r *SyncJobRepository) UpdateProgress(ctx context.Context, id uuid.UUID, found, new, skipped int) error {
	query := `
		UPDATE sync_jobs
		SET documents_found = $2, documents_new = $3, documents_skipped = $4
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, found, new, skipped)
	if err != nil {
		return fmt.Errorf("update sync progress: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSyncJobNotFound
	}

	return nil
}

// GetRunningForAccount checks if there's already a running sync for an account
func (r *SyncJobRepository) GetRunningForAccount(ctx context.Context, accountID uuid.UUID) (*SyncJob, error) {
	query := `
		SELECT id, tenant_id, account_id, status, job_type,
			documents_found, documents_new, documents_skipped,
			error_message, started_at, completed_at, created_at
		FROM sync_jobs
		WHERE account_id = $1 AND status = 'running'
		LIMIT 1
	`

	job := &SyncJob{}
	err := r.db.QueryRow(ctx, query, accountID).Scan(
		&job.ID, &job.TenantID, &job.AccountID, &job.Status, &job.JobType,
		&job.DocumentsFound, &job.DocumentsNew, &job.DocumentsSkipped,
		&job.ErrorMessage, &job.StartedAt, &job.CompletedAt, &job.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No running job
		}
		return nil, fmt.Errorf("get running sync job: %w", err)
	}

	return job, nil
}

// GetLastForAccount returns the most recent sync job for an account
func (r *SyncJobRepository) GetLastForAccount(ctx context.Context, accountID uuid.UUID) (*SyncJob, error) {
	query := `
		SELECT id, tenant_id, account_id, status, job_type,
			documents_found, documents_new, documents_skipped,
			error_message, started_at, completed_at, created_at
		FROM sync_jobs
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	job := &SyncJob{}
	err := r.db.QueryRow(ctx, query, accountID).Scan(
		&job.ID, &job.TenantID, &job.AccountID, &job.Status, &job.JobType,
		&job.DocumentsFound, &job.DocumentsNew, &job.DocumentsSkipped,
		&job.ErrorMessage, &job.StartedAt, &job.CompletedAt, &job.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get last sync job: %w", err)
	}

	return job, nil
}

// CleanupOld removes old completed sync jobs
func (r *SyncJobRepository) CleanupOld(ctx context.Context, tenantID uuid.UUID, olderThan time.Duration) (int, error) {
	query := `
		DELETE FROM sync_jobs
		WHERE tenant_id = $1
		AND status IN ('completed', 'failed')
		AND created_at < $2
	`

	cutoff := time.Now().Add(-olderThan)
	result, err := r.db.Exec(ctx, query, tenantID, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup old sync jobs: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// Sync job status constants
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// Sync job type constants
const (
	JobTypeSingle = "single"
	JobTypeAll    = "all"
)
