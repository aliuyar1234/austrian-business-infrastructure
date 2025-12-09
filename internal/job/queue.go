package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Queue errors
var (
	ErrJobNotFound     = errors.New("job not found")
	ErrDuplicateJob    = errors.New("duplicate job (idempotency key)")
	ErrNoJobsAvailable = errors.New("no jobs available")
)

// Queue manages the PostgreSQL-based job queue
type Queue struct {
	db       *pgxpool.Pool
	workerID string
	logger   *slog.Logger
}

// QueueConfig holds queue configuration
type QueueConfig struct {
	WorkerID string
	Logger   *slog.Logger
}

// NewQueue creates a new job queue
func NewQueue(db *pgxpool.Pool, cfg *QueueConfig) *Queue {
	logger := slog.Default()
	if cfg != nil && cfg.Logger != nil {
		logger = cfg.Logger
	}

	workerID := "default"
	if cfg != nil && cfg.WorkerID != "" {
		workerID = cfg.WorkerID
	}

	return &Queue{
		db:       db,
		workerID: workerID,
		logger:   logger,
	}
}

// Enqueue adds a new job to the queue
func (q *Queue) Enqueue(ctx context.Context, tenantID uuid.UUID, jobType string, payload interface{}, opts *EnqueueOptions) (*Job, error) {
	if opts == nil {
		opts = DefaultEnqueueOptions()
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	job := &Job{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Type:           jobType,
		Payload:        payloadBytes,
		Priority:       opts.Priority,
		Status:         StatusPending,
		MaxRetries:     opts.MaxRetries,
		RetryCount:     0,
		RunAt:          opts.RunAt,
		TimeoutSeconds: opts.TimeoutSeconds,
		IdempotencyKey: opts.IdempotencyKey,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	query := `
		INSERT INTO jobs (
			id, tenant_id, type, payload, priority, status, max_retries, retry_count,
			run_at, timeout_seconds, idempotency_key, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL
		DO NOTHING
		RETURNING id
	`

	var returnedID uuid.UUID
	err = q.db.QueryRow(ctx, query,
		job.ID, job.TenantID, job.Type, job.Payload, job.Priority, job.Status,
		job.MaxRetries, job.RetryCount, job.RunAt, job.TimeoutSeconds,
		nullString(job.IdempotencyKey), job.CreatedAt, job.UpdatedAt,
	).Scan(&returnedID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDuplicateJob
		}
		return nil, fmt.Errorf("insert job: %w", err)
	}

	q.logger.Debug("job enqueued",
		"job_id", job.ID,
		"type", job.Type,
		"priority", job.Priority,
		"run_at", job.RunAt)

	return job, nil
}

// Dequeue fetches and claims the next available job
// Uses SELECT FOR UPDATE SKIP LOCKED for concurrent worker coordination
func (q *Queue) Dequeue(ctx context.Context) (*Job, error) {
	query := `
		UPDATE jobs
		SET status = $1, started_at = $2, worker_id = $3, updated_at = $2
		WHERE id = (
			SELECT id FROM jobs
			WHERE status = $4 AND run_at <= $2
			ORDER BY priority DESC, run_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, tenant_id, type, payload, priority, status, max_retries, retry_count,
		          last_error, run_at, started_at, timeout_seconds, worker_id,
		          idempotency_key, created_at, updated_at
	`

	now := time.Now()
	job := &Job{}

	err := q.db.QueryRow(ctx, query, StatusRunning, now, q.workerID, StatusPending).Scan(
		&job.ID, &job.TenantID, &job.Type, &job.Payload, &job.Priority, &job.Status,
		&job.MaxRetries, &job.RetryCount, &job.LastError, &job.RunAt, &job.StartedAt,
		&job.TimeoutSeconds, &job.WorkerID, &job.IdempotencyKey, &job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoJobsAvailable
		}
		return nil, fmt.Errorf("dequeue job: %w", err)
	}

	q.logger.Debug("job dequeued",
		"job_id", job.ID,
		"type", job.Type,
		"priority", job.Priority)

	return job, nil
}

// Complete marks a job as successfully completed
func (q *Queue) Complete(ctx context.Context, jobID uuid.UUID, result json.RawMessage) error {
	now := time.Now()

	// First, get the job details for history
	job, err := q.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}

	// Update job status
	query := `
		UPDATE jobs
		SET status = $1, completed_at = $2, updated_at = $2
		WHERE id = $3 AND status = $4
	`

	tag, err := q.db.Exec(ctx, query, StatusCompleted, now, jobID, StatusRunning)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}

	// Record in job history
	historyQuery := `
		INSERT INTO job_history (
			tenant_id, job_id, type, payload, status, result,
			started_at, completed_at, worker_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = q.db.Exec(ctx, historyQuery,
		job.TenantID, jobID, job.Type, job.Payload, StatusCompleted, result,
		job.StartedAt, now, q.workerID, now,
	)
	if err != nil {
		q.logger.Error("failed to record job history", "job_id", jobID, "error", err)
		// Don't return error - job completion is more important
	}

	q.logger.Debug("job completed", "job_id", jobID)

	return nil
}

// Fail marks a job as failed and handles retry logic
func (q *Queue) Fail(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	now := time.Now()

	// Get job to check retry count
	job, err := q.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}

	newRetryCount := job.RetryCount + 1

	// Check if we should move to dead letter queue
	if newRetryCount >= job.MaxRetries {
		return q.moveToDead(ctx, job, errMsg)
	}

	// Calculate exponential backoff delay: 1s, 2s, 4s, 8s, ...
	delay := time.Duration(1<<uint(newRetryCount)) * time.Second
	nextRunAt := now.Add(delay)

	query := `
		UPDATE jobs
		SET status = $1, retry_count = $2, last_error = $3, run_at = $4,
		    started_at = NULL, worker_id = NULL, updated_at = $5
		WHERE id = $6
	`

	_, err = q.db.Exec(ctx, query, StatusPending, newRetryCount, errMsg, nextRunAt, now, jobID)
	if err != nil {
		return fmt.Errorf("fail job: %w", err)
	}

	q.logger.Info("job failed, will retry",
		"job_id", jobID,
		"retry_count", newRetryCount,
		"max_retries", job.MaxRetries,
		"next_run_at", nextRunAt,
		"error", errMsg)

	return nil
}

// moveToDead moves a job to the dead letter queue
func (q *Queue) moveToDead(ctx context.Context, job *Job, lastError string) error {
	now := time.Now()

	// Collect all errors
	errors := []string{}
	if job.LastError != "" {
		errors = append(errors, job.LastError)
	}
	errors = append(errors, lastError)

	errorsJSON, _ := json.Marshal(errors)

	// Insert into dead letters
	insertQuery := `
		INSERT INTO dead_letters (
			tenant_id, original_job_id, type, payload, errors, max_retries,
			total_attempts, first_attempted_at, last_attempted_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := q.db.Exec(ctx, insertQuery,
		job.TenantID, job.ID, job.Type, job.Payload, errorsJSON, job.MaxRetries,
		job.RetryCount+1, job.StartedAt, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert dead letter: %w", err)
	}

	// Update job status to dead
	updateQuery := `
		UPDATE jobs
		SET status = $1, completed_at = $2, last_error = $3, updated_at = $2
		WHERE id = $4
	`

	_, err = q.db.Exec(ctx, updateQuery, StatusDead, now, lastError, job.ID)
	if err != nil {
		return fmt.Errorf("update job to dead: %w", err)
	}

	// Record in job history
	historyQuery := `
		INSERT INTO job_history (
			tenant_id, job_id, type, payload, status, error_message,
			started_at, completed_at, worker_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = q.db.Exec(ctx, historyQuery,
		job.TenantID, job.ID, job.Type, job.Payload, StatusFailed, lastError,
		job.StartedAt, now, q.workerID, now,
	)
	if err != nil {
		q.logger.Error("failed to record job history", "job_id", job.ID, "error", err)
	}

	q.logger.Warn("job moved to dead letter queue",
		"job_id", job.ID,
		"type", job.Type,
		"total_attempts", job.RetryCount+1,
		"error", lastError)

	return nil
}

// GetByID retrieves a job by its ID
func (q *Queue) GetByID(ctx context.Context, id uuid.UUID) (*Job, error) {
	query := `
		SELECT id, tenant_id, type, payload, priority, status, max_retries, retry_count,
		       last_error, run_at, started_at, completed_at, timeout_seconds, worker_id,
		       idempotency_key, created_at, updated_at
		FROM jobs WHERE id = $1
	`

	job := &Job{}
	var lastError, idempotencyKey, workerID *string

	err := q.db.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.TenantID, &job.Type, &job.Payload, &job.Priority, &job.Status,
		&job.MaxRetries, &job.RetryCount, &lastError, &job.RunAt, &job.StartedAt,
		&job.CompletedAt, &job.TimeoutSeconds, &workerID, &idempotencyKey,
		&job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("get job: %w", err)
	}

	if lastError != nil {
		job.LastError = *lastError
	}
	if idempotencyKey != nil {
		job.IdempotencyKey = *idempotencyKey
	}
	if workerID != nil {
		job.WorkerID = *workerID
	}

	return job, nil
}

// QueueLength returns the number of pending jobs
func (q *Queue) QueueLength(ctx context.Context) (int64, error) {
	var count int64
	err := q.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE status = $1`, StatusPending).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending jobs: %w", err)
	}
	return count, nil
}

// QueueLengthByType returns the number of pending jobs by type
func (q *Queue) QueueLengthByType(ctx context.Context, jobType string) (int64, error) {
	var count int64
	err := q.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE status = $1 AND type = $2`, StatusPending, jobType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending jobs by type: %w", err)
	}
	return count, nil
}

// CleanupStaleJobs marks jobs as failed if they've been running too long
func (q *Queue) CleanupStaleJobs(ctx context.Context) (int64, error) {
	now := time.Now()

	query := `
		UPDATE jobs
		SET status = $1, last_error = 'job timed out', updated_at = $2
		WHERE status = $3 AND started_at IS NOT NULL
		AND started_at + (timeout_seconds || ' seconds')::interval < $2
	`

	tag, err := q.db.Exec(ctx, query, StatusFailed, now, StatusRunning)
	if err != nil {
		return 0, fmt.Errorf("cleanup stale jobs: %w", err)
	}

	affected := tag.RowsAffected()
	if affected > 0 {
		q.logger.Warn("cleaned up stale jobs", "count", affected)
	}

	return affected, nil
}

// DeleteCompletedJobs removes old completed jobs (for cleanup)
func (q *Queue) DeleteCompletedJobs(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)

	tag, err := q.db.Exec(ctx, `
		DELETE FROM jobs
		WHERE status IN ($1, $2) AND completed_at < $3
	`, StatusCompleted, StatusDead, cutoff)

	if err != nil {
		return 0, fmt.Errorf("delete completed jobs: %w", err)
	}

	return tag.RowsAffected(), nil
}

// nullString returns nil for empty strings
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
