package uva

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrBatchNotFound      = errors.New("batch not found")
	ErrDuplicatePeriod    = errors.New("submission for this period already exists")
)

// Repository handles UVA database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new UVA repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new UVA submission
func (r *Repository) Create(ctx context.Context, s *Submission) (*Submission, error) {
	s.ID = uuid.New()
	s.CreatedAt = time.Now()
	s.UpdatedAt = s.CreatedAt
	s.Status = StatusDraft
	s.ValidationStatus = "pending"

	query := `
		INSERT INTO uva_submissions (
			id, tenant_id, account_id, period_year, period_month, period_quarter,
			period_type, data, validation_status, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		s.ID, s.TenantID, s.AccountID, s.PeriodYear, s.PeriodMonth, s.PeriodQuarter,
		s.PeriodType, s.Data, s.ValidationStatus, s.Status, s.CreatedAt, s.UpdatedAt,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	return s, nil
}

// GetByID retrieves a submission by ID
func (r *Repository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*Submission, error) {
	query := `
		SELECT id, tenant_id, account_id, period_year, period_month, period_quarter,
			period_type, data, validation_status, validation_errors, status,
			fo_reference, xml_content, submitted_at, submitted_by,
			response_code, response_message, created_at, updated_at
		FROM uva_submissions
		WHERE id = $1 AND tenant_id = $2`

	var s Submission
	var periodMonth, periodQuarter sql.NullInt32
	var foRef, respMsg sql.NullString
	var respCode sql.NullInt32
	var submittedAt sql.NullTime
	var submittedBy uuid.NullUUID
	var validationErrors, xmlContent []byte

	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&s.ID, &s.TenantID, &s.AccountID, &s.PeriodYear, &periodMonth, &periodQuarter,
		&s.PeriodType, &s.Data, &s.ValidationStatus, &validationErrors, &s.Status,
		&foRef, &xmlContent, &submittedAt, &submittedBy,
		&respCode, &respMsg, &s.CreatedAt, &s.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubmissionNotFound
		}
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	if periodMonth.Valid {
		m := int(periodMonth.Int32)
		s.PeriodMonth = &m
	}
	if periodQuarter.Valid {
		q := int(periodQuarter.Int32)
		s.PeriodQuarter = &q
	}
	if foRef.Valid {
		s.FOReference = &foRef.String
	}
	if respMsg.Valid {
		s.ResponseMessage = &respMsg.String
	}
	if respCode.Valid {
		c := int(respCode.Int32)
		s.ResponseCode = &c
	}
	if submittedAt.Valid {
		s.SubmittedAt = &submittedAt.Time
	}
	if submittedBy.Valid {
		s.SubmittedBy = &submittedBy.UUID
	}
	if len(validationErrors) > 0 {
		s.ValidationErrors = validationErrors
	}
	if len(xmlContent) > 0 {
		s.XMLContent = xmlContent
	}

	return &s, nil
}

// List retrieves submissions with filtering
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]*Submission, int, error) {
	baseQuery := `
		FROM uva_submissions
		WHERE tenant_id = $1`

	args := []interface{}{filter.TenantID}
	argIdx := 2

	if filter.AccountID != nil {
		baseQuery += fmt.Sprintf(" AND account_id = $%d", argIdx)
		args = append(args, *filter.AccountID)
		argIdx++
	}

	if filter.PeriodYear != nil {
		baseQuery += fmt.Sprintf(" AND period_year = $%d", argIdx)
		args = append(args, *filter.PeriodYear)
		argIdx++
	}

	if filter.PeriodType != nil {
		baseQuery += fmt.Sprintf(" AND period_type = $%d", argIdx)
		args = append(args, *filter.PeriodType)
		argIdx++
	}

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count submissions: %w", err)
	}

	// Get paginated results
	selectQuery := `
		SELECT id, tenant_id, account_id, period_year, period_month, period_quarter,
			period_type, data, validation_status, validation_errors, status,
			fo_reference, submitted_at, created_at, updated_at
		` + baseQuery + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list submissions: %w", err)
	}
	defer rows.Close()

	var submissions []*Submission
	for rows.Next() {
		var s Submission
		var periodMonth, periodQuarter sql.NullInt32
		var foRef sql.NullString
		var submittedAt sql.NullTime
		var validationErrors []byte

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.AccountID, &s.PeriodYear, &periodMonth, &periodQuarter,
			&s.PeriodType, &s.Data, &s.ValidationStatus, &validationErrors, &s.Status,
			&foRef, &submittedAt, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan submission: %w", err)
		}

		if periodMonth.Valid {
			m := int(periodMonth.Int32)
			s.PeriodMonth = &m
		}
		if periodQuarter.Valid {
			q := int(periodQuarter.Int32)
			s.PeriodQuarter = &q
		}
		if foRef.Valid {
			s.FOReference = &foRef.String
		}
		if submittedAt.Valid {
			s.SubmittedAt = &submittedAt.Time
		}
		if len(validationErrors) > 0 {
			s.ValidationErrors = validationErrors
		}

		submissions = append(submissions, &s)
	}

	return submissions, total, nil
}

// Update updates a submission
func (r *Repository) Update(ctx context.Context, s *Submission) error {
	s.UpdatedAt = time.Now()

	query := `
		UPDATE uva_submissions SET
			data = $1,
			validation_status = $2,
			validation_errors = $3,
			status = $4,
			updated_at = $5
		WHERE id = $6 AND tenant_id = $7`

	result, err := r.db.Exec(ctx, query,
		s.Data, s.ValidationStatus, s.ValidationErrors, s.Status,
		s.UpdatedAt, s.ID, s.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSubmissionNotFound
	}

	return nil
}

// UpdateSubmissionResult updates the submission with FO result
func (r *Repository) UpdateSubmissionResult(ctx context.Context, id, tenantID uuid.UUID, foRef string, respCode int, respMsg string, status string) error {
	now := time.Now()

	query := `
		UPDATE uva_submissions SET
			fo_reference = $1,
			response_code = $2,
			response_message = $3,
			status = $4,
			submitted_at = $5,
			updated_at = $5
		WHERE id = $6 AND tenant_id = $7`

	result, err := r.db.Exec(ctx, query,
		foRef, respCode, respMsg, status, now, id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update submission result: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSubmissionNotFound
	}

	return nil
}

// SaveXMLContent saves the generated XML content
func (r *Repository) SaveXMLContent(ctx context.Context, id, tenantID uuid.UUID, xmlContent []byte) error {
	query := `
		UPDATE uva_submissions SET
			xml_content = $1,
			updated_at = $2
		WHERE id = $3 AND tenant_id = $4`

	result, err := r.db.Exec(ctx, query, xmlContent, time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to save XML content: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSubmissionNotFound
	}

	return nil
}

// Delete deletes a submission (only drafts)
func (r *Repository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `
		DELETE FROM uva_submissions
		WHERE id = $1 AND tenant_id = $2 AND status = 'draft'`

	result, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete submission: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSubmissionNotFound
	}

	return nil
}

// CheckDuplicatePeriod checks if a submission for the same period exists
func (r *Repository) CheckDuplicatePeriod(ctx context.Context, tenantID, accountID uuid.UUID, year int, periodType string, periodValue int, excludeID *uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM uva_submissions
			WHERE tenant_id = $1 AND account_id = $2 AND period_year = $3 AND period_type = $4
			AND CASE
				WHEN period_type = 'monthly' THEN period_month = $5
				ELSE period_quarter = $5
			END
			AND status NOT IN ('rejected', 'error')`

	args := []interface{}{tenantID, accountID, year, periodType, periodValue}

	if excludeID != nil {
		query += ` AND id != $6`
		args = append(args, *excludeID)
	}

	query += `)`

	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate period: %w", err)
	}

	return exists, nil
}

// Batch operations

// CreateBatch creates a new batch
func (r *Repository) CreateBatch(ctx context.Context, b *Batch) (*Batch, error) {
	b.ID = uuid.New()
	b.CreatedAt = time.Now()
	b.UpdatedAt = b.CreatedAt
	b.Status = "pending"

	query := `
		INSERT INTO uva_batches (
			id, tenant_id, name, period_year, period_month, period_quarter,
			period_type, total_count, success_count, failed_count, status,
			created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		b.ID, b.TenantID, b.Name, b.PeriodYear, b.PeriodMonth, b.PeriodQuarter,
		b.PeriodType, b.TotalCount, b.SuccessCount, b.FailedCount, b.Status,
		b.CreatedBy, b.CreatedAt, b.UpdatedAt,
	).Scan(&b.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	return b, nil
}

// GetBatchByID retrieves a batch by ID
func (r *Repository) GetBatchByID(ctx context.Context, id, tenantID uuid.UUID) (*Batch, error) {
	query := `
		SELECT id, tenant_id, name, period_year, period_month, period_quarter,
			period_type, total_count, success_count, failed_count, status,
			started_at, completed_at, created_by, created_at, updated_at
		FROM uva_batches
		WHERE id = $1 AND tenant_id = $2`

	var b Batch
	var periodMonth, periodQuarter sql.NullInt32
	var startedAt, completedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&b.ID, &b.TenantID, &b.Name, &b.PeriodYear, &periodMonth, &periodQuarter,
		&b.PeriodType, &b.TotalCount, &b.SuccessCount, &b.FailedCount, &b.Status,
		&startedAt, &completedAt, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBatchNotFound
		}
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}

	if periodMonth.Valid {
		m := int(periodMonth.Int32)
		b.PeriodMonth = &m
	}
	if periodQuarter.Valid {
		q := int(periodQuarter.Int32)
		b.PeriodQuarter = &q
	}
	if startedAt.Valid {
		b.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		b.CompletedAt = &completedAt.Time
	}

	return &b, nil
}

// UpdateBatchProgress updates batch progress
func (r *Repository) UpdateBatchProgress(ctx context.Context, id, tenantID uuid.UUID, successCount, failedCount int, status string) error {
	now := time.Now()

	query := `
		UPDATE uva_batches SET
			success_count = $1,
			failed_count = $2,
			status = $3,
			updated_at = $4`

	args := []interface{}{successCount, failedCount, status, now}

	if status == "running" {
		query += `, started_at = COALESCE(started_at, $5)`
		args = append(args, now)
	} else if status == "completed" || status == "failed" {
		query += `, completed_at = $5`
		args = append(args, now)
	}

	query += fmt.Sprintf(` WHERE id = $%d AND tenant_id = $%d`, len(args)+1, len(args)+2)
	args = append(args, id, tenantID)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update batch progress: %w", err)
	}

	return nil
}

// ListBatches lists batches for a tenant
func (r *Repository) ListBatches(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Batch, int, error) {
	countQuery := `SELECT COUNT(*) FROM uva_batches WHERE tenant_id = $1`
	var total int
	err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count batches: %w", err)
	}

	query := `
		SELECT id, tenant_id, name, period_year, period_month, period_quarter,
			period_type, total_count, success_count, failed_count, status,
			started_at, completed_at, created_by, created_at, updated_at
		FROM uva_batches
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list batches: %w", err)
	}
	defer rows.Close()

	var batches []*Batch
	for rows.Next() {
		var b Batch
		var periodMonth, periodQuarter sql.NullInt32
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(
			&b.ID, &b.TenantID, &b.Name, &b.PeriodYear, &periodMonth, &periodQuarter,
			&b.PeriodType, &b.TotalCount, &b.SuccessCount, &b.FailedCount, &b.Status,
			&startedAt, &completedAt, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan batch: %w", err)
		}

		if periodMonth.Valid {
			m := int(periodMonth.Int32)
			b.PeriodMonth = &m
		}
		if periodQuarter.Valid {
			q := int(periodQuarter.Int32)
			b.PeriodQuarter = &q
		}
		if startedAt.Valid {
			b.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			b.CompletedAt = &completedAt.Time
		}

		batches = append(batches, &b)
	}

	return batches, total, nil
}

// GetBatchSubmissions retrieves all submissions for a batch
func (r *Repository) GetBatchSubmissions(ctx context.Context, batchID, tenantID uuid.UUID) ([]*Submission, error) {
	// For now, batch submissions are tracked via a separate mechanism
	// This would query a batch_items join table in a full implementation
	return nil, nil
}

// SetSubmittedBy marks who submitted the UVA
func (r *Repository) SetSubmittedBy(ctx context.Context, id, tenantID, userID uuid.UUID) error {
	query := `
		UPDATE uva_submissions SET submitted_by = $1, updated_at = $2
		WHERE id = $3 AND tenant_id = $4`

	_, err := r.db.Exec(ctx, query, userID, time.Now(), id, tenantID)
	return err
}

// GetByPeriod retrieves a submission by period
func (r *Repository) GetByPeriod(ctx context.Context, tenantID, accountID uuid.UUID, year int, periodType string, periodValue int) (*Submission, error) {
	query := `
		SELECT id, tenant_id, account_id, period_year, period_month, period_quarter,
			period_type, data, validation_status, validation_errors, status,
			fo_reference, submitted_at, created_at, updated_at
		FROM uva_submissions
		WHERE tenant_id = $1 AND account_id = $2 AND period_year = $3 AND period_type = $4
		AND CASE
			WHEN period_type = 'monthly' THEN period_month = $5
			ELSE period_quarter = $5
		END
		AND status NOT IN ('rejected', 'error')
		ORDER BY created_at DESC
		LIMIT 1`

	var s Submission
	var periodMonth, periodQuarter sql.NullInt32
	var foRef sql.NullString
	var submittedAt sql.NullTime
	var validationErrors []byte

	err := r.db.QueryRow(ctx, query, tenantID, accountID, year, periodType, periodValue).Scan(
		&s.ID, &s.TenantID, &s.AccountID, &s.PeriodYear, &periodMonth, &periodQuarter,
		&s.PeriodType, &s.Data, &s.ValidationStatus, &validationErrors, &s.Status,
		&foRef, &submittedAt, &s.CreatedAt, &s.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubmissionNotFound
		}
		return nil, fmt.Errorf("failed to get submission by period: %w", err)
	}

	if periodMonth.Valid {
		m := int(periodMonth.Int32)
		s.PeriodMonth = &m
	}
	if periodQuarter.Valid {
		q := int(periodQuarter.Int32)
		s.PeriodQuarter = &q
	}
	if foRef.Valid {
		s.FOReference = &foRef.String
	}
	if submittedAt.Valid {
		s.SubmittedAt = &submittedAt.Time
	}
	if len(validationErrors) > 0 {
		s.ValidationErrors = json.RawMessage(validationErrors)
	}

	return &s, nil
}
