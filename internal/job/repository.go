package job

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles job history and dead letter database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new job repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// JobHistoryFilter holds filter options for job history queries
type JobHistoryFilter struct {
	TenantID   uuid.UUID
	Type       string
	Status     string
	DateFrom   *time.Time
	DateTo     *time.Time
	ScheduleID *uuid.UUID
	Limit      int
	Offset     int
}

// ListHistory retrieves job history with filtering
func (r *Repository) ListHistory(ctx context.Context, filter JobHistoryFilter) ([]*JobHistory, int, error) {
	baseQuery := `FROM job_history WHERE tenant_id = $1`
	args := []interface{}{filter.TenantID}
	argNum := 2

	if filter.Type != "" {
		baseQuery += fmt.Sprintf(" AND type = $%d", argNum)
		args = append(args, filter.Type)
		argNum++
	}

	if filter.Status != "" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, filter.Status)
		argNum++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND started_at >= $%d", argNum)
		args = append(args, filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND started_at <= $%d", argNum)
		args = append(args, filter.DateTo)
		argNum++
	}

	if filter.ScheduleID != nil {
		baseQuery += fmt.Sprintf(" AND schedule_id = $%d", argNum)
		args = append(args, filter.ScheduleID)
		argNum++
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count job history: %w", err)
	}

	// Fetch rows
	selectQuery := `
		SELECT id, tenant_id, job_id, schedule_id, type, payload, status, result,
		       error_message, started_at, completed_at, duration_ms, worker_id, created_at
	` + baseQuery + " ORDER BY started_at DESC"

	if filter.Limit > 0 {
		selectQuery += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, filter.Limit)
		argNum++
	}

	if filter.Offset > 0 {
		selectQuery += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query job history: %w", err)
	}
	defer rows.Close()

	var history []*JobHistory
	for rows.Next() {
		h := &JobHistory{}
		err := rows.Scan(
			&h.ID, &h.TenantID, &h.JobID, &h.ScheduleID, &h.Type, &h.Payload,
			&h.Status, &h.Result, &h.ErrorMessage, &h.StartedAt, &h.CompletedAt,
			&h.DurationMs, &h.WorkerID, &h.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan job history: %w", err)
		}
		history = append(history, h)
	}

	return history, total, rows.Err()
}

// GetHistoryByID retrieves a single job history entry
func (r *Repository) GetHistoryByID(ctx context.Context, id uuid.UUID) (*JobHistory, error) {
	query := `
		SELECT id, tenant_id, job_id, schedule_id, type, payload, status, result,
		       error_message, started_at, completed_at, duration_ms, worker_id, created_at
		FROM job_history WHERE id = $1
	`

	h := &JobHistory{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&h.ID, &h.TenantID, &h.JobID, &h.ScheduleID, &h.Type, &h.Payload,
		&h.Status, &h.Result, &h.ErrorMessage, &h.StartedAt, &h.CompletedAt,
		&h.DurationMs, &h.WorkerID, &h.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get job history: %w", err)
	}

	return h, nil
}

// ListDeadLetters retrieves dead letter queue entries
func (r *Repository) ListDeadLetters(ctx context.Context, tenantID uuid.UUID, acknowledged bool, limit, offset int) ([]*DeadLetter, int, error) {
	baseQuery := `FROM dead_letters WHERE tenant_id = $1 AND acknowledged = $2`
	args := []interface{}{tenantID, acknowledged}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count dead letters: %w", err)
	}

	// Fetch rows
	selectQuery := `
		SELECT id, tenant_id, original_job_id, type, payload, errors, max_retries,
		       total_attempts, first_attempted_at, last_attempted_at, acknowledged,
		       acknowledged_by, acknowledged_at, created_at
	` + baseQuery + " ORDER BY created_at DESC LIMIT $3 OFFSET $4"
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query dead letters: %w", err)
	}
	defer rows.Close()

	var deadLetters []*DeadLetter
	for rows.Next() {
		dl := &DeadLetter{}
		var errorsJSON []byte
		err := rows.Scan(
			&dl.ID, &dl.TenantID, &dl.OriginalJobID, &dl.Type, &dl.Payload, &errorsJSON,
			&dl.MaxRetries, &dl.TotalAttempts, &dl.FirstAttemptedAt, &dl.LastAttemptedAt,
			&dl.Acknowledged, &dl.AcknowledgedBy, &dl.AcknowledgedAt, &dl.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan dead letter: %w", err)
		}
		// Parse errors JSON array
		if len(errorsJSON) > 0 {
			// Simple JSON array parsing
			dl.Errors = parseJSONStringArray(errorsJSON)
		}
		deadLetters = append(deadLetters, dl)
	}

	return deadLetters, total, rows.Err()
}

// GetDeadLetterByID retrieves a single dead letter entry
func (r *Repository) GetDeadLetterByID(ctx context.Context, id uuid.UUID) (*DeadLetter, error) {
	query := `
		SELECT id, tenant_id, original_job_id, type, payload, errors, max_retries,
		       total_attempts, first_attempted_at, last_attempted_at, acknowledged,
		       acknowledged_by, acknowledged_at, created_at
		FROM dead_letters WHERE id = $1
	`

	dl := &DeadLetter{}
	var errorsJSON []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&dl.ID, &dl.TenantID, &dl.OriginalJobID, &dl.Type, &dl.Payload, &errorsJSON,
		&dl.MaxRetries, &dl.TotalAttempts, &dl.FirstAttemptedAt, &dl.LastAttemptedAt,
		&dl.Acknowledged, &dl.AcknowledgedBy, &dl.AcknowledgedAt, &dl.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dead letter: %w", err)
	}

	if len(errorsJSON) > 0 {
		dl.Errors = parseJSONStringArray(errorsJSON)
	}

	return dl, nil
}

// AcknowledgeDeadLetter marks a dead letter as acknowledged
func (r *Repository) AcknowledgeDeadLetter(ctx context.Context, id, userID uuid.UUID) error {
	query := `
		UPDATE dead_letters
		SET acknowledged = TRUE, acknowledged_by = $1, acknowledged_at = NOW()
		WHERE id = $2 AND acknowledged = FALSE
	`

	result, err := r.db.Exec(ctx, query, userID, id)
	if err != nil {
		return fmt.Errorf("acknowledge dead letter: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrJobNotFound
	}

	return nil
}

// GetJobMetrics returns aggregated job metrics for a tenant
func (r *Repository) GetJobMetrics(ctx context.Context, tenantID uuid.UUID) (*JobMetrics, error) {
	metrics := &JobMetrics{}

	// Pending jobs count
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE tenant_id = $1 AND status = 'pending'`, tenantID).Scan(&metrics.QueueLength)
	if err != nil {
		return nil, fmt.Errorf("count pending jobs: %w", err)
	}

	// Running jobs count
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE tenant_id = $1 AND status = 'running'`, tenantID).Scan(&metrics.RunningJobs)
	if err != nil {
		return nil, fmt.Errorf("count running jobs: %w", err)
	}

	// Jobs completed in last 24 hours
	yesterday := time.Now().Add(-24 * time.Hour)
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0)
		FROM job_history WHERE tenant_id = $1 AND started_at >= $2
	`, tenantID, yesterday).Scan(&metrics.JobsLast24h, &metrics.SuccessLast24h)
	if err != nil {
		return nil, fmt.Errorf("count recent jobs: %w", err)
	}

	// Calculate success rate
	if metrics.JobsLast24h > 0 {
		metrics.SuccessRate = float64(metrics.SuccessLast24h) / float64(metrics.JobsLast24h) * 100
	}

	// Average duration
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(duration_ms), 0) FROM job_history WHERE tenant_id = $1 AND started_at >= $2
	`, tenantID, yesterday).Scan(&metrics.AvgDurationMs)
	if err != nil {
		return nil, fmt.Errorf("avg duration: %w", err)
	}

	// Dead letters count
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM dead_letters WHERE tenant_id = $1 AND acknowledged = FALSE`, tenantID).Scan(&metrics.DeadLetters)
	if err != nil {
		return nil, fmt.Errorf("count dead letters: %w", err)
	}

	return metrics, nil
}

// JobMetrics contains aggregated job statistics
type JobMetrics struct {
	QueueLength   int64   `json:"queue_length"`
	RunningJobs   int64   `json:"running_jobs"`
	JobsLast24h   int64   `json:"jobs_last_24h"`
	SuccessLast24h int64  `json:"success_last_24h"`
	SuccessRate   float64 `json:"success_rate"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
	DeadLetters   int64   `json:"dead_letters"`
}

// parseJSONStringArray parses a JSON array of strings
func parseJSONStringArray(data []byte) []string {
	// Simple parser for ["a", "b", "c"] format
	var result []string
	if len(data) < 2 {
		return result
	}

	// Remove brackets
	content := string(data[1 : len(data)-1])
	if content == "" {
		return result
	}

	// Split by comma and clean up
	inQuote := false
	current := ""
	for _, c := range content {
		if c == '"' {
			inQuote = !inQuote
		} else if c == ',' && !inQuote {
			if trimmed := trimQuotes(current); trimmed != "" {
				result = append(result, trimmed)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if trimmed := trimQuotes(current); trimmed != "" {
		result = append(result, trimmed)
	}

	return result
}

func trimQuotes(s string) string {
	// Trim spaces and quotes
	for len(s) > 0 && (s[0] == ' ' || s[0] == '"') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '"') {
		s = s[:len(s)-1]
	}
	return s
}
