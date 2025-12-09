package lohnzettel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/austrian-business-infrastructure/fo/internal/elda"
)

var (
	ErrLohnzettelNotFound = errors.New("Lohnzettel not found")
	ErrBatchNotFound      = errors.New("Lohnzettel batch not found")
	ErrDuplicate          = errors.New("Lohnzettel already exists")
)

// Repository handles Lohnzettel database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new Lohnzettel repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new Lohnzettel record
func (r *Repository) Create(ctx context.Context, l *elda.Lohnzettel) error {
	l16DataJSON, _ := json.Marshal(l.L16Data)

	query := `
		INSERT INTO lohnzettel (
			id, elda_account_id, year, sv_nummer, familienname, vorname, geburtsdatum,
			l16_data, status, batch_id, is_berichtigung, berichtigt_id,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15
		)
	`

	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	now := time.Now()
	l.CreatedAt = now
	l.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		l.ID, l.ELDAAccountID, l.Year, l.SVNummer, l.Familienname, l.Vorname, l.Geburtsdatum,
		l16DataJSON, l.Status, l.BatchID, l.IsBerichtigung, l.BerichtigtID,
		l.CreatedBy, l.CreatedAt, l.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create lohnzettel: %w", err)
	}

	return nil
}

// GetByID retrieves a Lohnzettel by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*elda.Lohnzettel, error) {
	query := `
		SELECT
			id, elda_account_id, year, sv_nummer, familienname, vorname, geburtsdatum,
			l16_data, status, protokollnummer, batch_id,
			submitted_at, request_xml, response_xml, error_message, error_code,
			is_berichtigung, berichtigt_id, created_by, created_at, updated_at
		FROM lohnzettel
		WHERE id = $1
	`

	l := &elda.Lohnzettel{}
	var l16DataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&l.ID, &l.ELDAAccountID, &l.Year, &l.SVNummer, &l.Familienname, &l.Vorname, &l.Geburtsdatum,
		&l16DataJSON, &l.Status, &l.Protokollnummer, &l.BatchID,
		&l.SubmittedAt, &l.RequestXML, &l.ResponseXML, &l.ErrorMessage, &l.ErrorCode,
		&l.IsBerichtigung, &l.BerichtigtID, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLohnzettelNotFound
		}
		return nil, fmt.Errorf("get lohnzettel: %w", err)
	}

	if len(l16DataJSON) > 0 {
		json.Unmarshal(l16DataJSON, &l.L16Data)
	}

	return l, nil
}

// Update updates a Lohnzettel record
func (r *Repository) Update(ctx context.Context, l *elda.Lohnzettel) error {
	l16DataJSON, _ := json.Marshal(l.L16Data)

	query := `
		UPDATE lohnzettel SET
			l16_data = $2,
			status = $3,
			protokollnummer = $4,
			batch_id = $5,
			submitted_at = $6,
			request_xml = $7,
			response_xml = $8,
			error_message = $9,
			error_code = $10,
			updated_at = $11
		WHERE id = $1
	`

	l.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, query,
		l.ID, l16DataJSON, l.Status, l.Protokollnummer, l.BatchID,
		l.SubmittedAt, l.RequestXML, l.ResponseXML, l.ErrorMessage, l.ErrorCode,
		l.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update lohnzettel: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrLohnzettelNotFound
	}

	return nil
}

// Delete deletes a Lohnzettel
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM lohnzettel WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete lohnzettel: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrLohnzettelNotFound
	}

	return nil
}

// ListFilter contains filter options for listing Lohnzettel
type ListFilter struct {
	ELDAAccountID *uuid.UUID
	Year          *int
	Status        *elda.L16Status
	BatchID       *uuid.UUID
	Limit         int
	Offset        int
}

// Get retrieves a Lohnzettel by ID (alias for GetByID)
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*elda.Lohnzettel, error) {
	return r.GetByID(ctx, id)
}

// List retrieves Lohnzettel with filters
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]*elda.Lohnzettel, error) {
	query := `
		SELECT
			id, elda_account_id, year, sv_nummer, familienname, vorname, geburtsdatum,
			l16_data, status, protokollnummer, batch_id,
			submitted_at, error_message, error_code,
			is_berichtigung, berichtigt_id, created_by, created_at, updated_at
		FROM lohnzettel
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter.ELDAAccountID != nil {
		query += fmt.Sprintf(" AND elda_account_id = $%d", argIndex)
		args = append(args, *filter.ELDAAccountID)
		argIndex++
	}

	if filter.Year != nil {
		query += fmt.Sprintf(" AND year = $%d", argIndex)
		args = append(args, *filter.Year)
		argIndex++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.BatchID != nil {
		query += fmt.Sprintf(" AND batch_id = $%d", argIndex)
		args = append(args, *filter.BatchID)
		argIndex++
	}

	query += " ORDER BY year DESC, familienname, vorname"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list lohnzettel: %w", err)
	}
	defer rows.Close()

	var results []*elda.Lohnzettel
	for rows.Next() {
		l := &elda.Lohnzettel{}
		var l16DataJSON []byte

		err := rows.Scan(
			&l.ID, &l.ELDAAccountID, &l.Year, &l.SVNummer, &l.Familienname, &l.Vorname, &l.Geburtsdatum,
			&l16DataJSON, &l.Status, &l.Protokollnummer, &l.BatchID,
			&l.SubmittedAt, &l.ErrorMessage, &l.ErrorCode,
			&l.IsBerichtigung, &l.BerichtigtID, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan lohnzettel: %w", err)
		}

		if len(l16DataJSON) > 0 {
			json.Unmarshal(l16DataJSON, &l.L16Data)
		}

		results = append(results, l)
	}

	return results, nil
}

// Count returns the count of Lohnzettel matching the filter
func (r *Repository) Count(ctx context.Context, filter ListFilter) (int, error) {
	query := `SELECT COUNT(*) FROM lohnzettel WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filter.ELDAAccountID != nil {
		query += fmt.Sprintf(" AND elda_account_id = $%d", argIndex)
		args = append(args, *filter.ELDAAccountID)
		argIndex++
	}

	if filter.Year != nil {
		query += fmt.Sprintf(" AND year = $%d", argIndex)
		args = append(args, *filter.Year)
		argIndex++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.BatchID != nil {
		query += fmt.Sprintf(" AND batch_id = $%d", argIndex)
		args = append(args, *filter.BatchID)
		argIndex++
	}

	var count int
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count lohnzettel: %w", err)
	}

	return count, nil
}

// GetBatch retrieves a batch by ID with its Lohnzettel
func (r *Repository) GetBatch(ctx context.Context, id uuid.UUID) (*elda.LohnzettelBatch, error) {
	return r.GetBatchWithLohnzettel(ctx, id)
}

// GetSubmittedYears returns years that have submitted Lohnzettel
func (r *Repository) GetSubmittedYears(ctx context.Context, eldaAccountID uuid.UUID) ([]int, error) {
	query := `
		SELECT DISTINCT year
		FROM lohnzettel
		WHERE elda_account_id = $1
		  AND status IN ('submitted', 'accepted')
		ORDER BY year DESC
	`

	rows, err := r.db.Query(ctx, query, eldaAccountID)
	if err != nil {
		return nil, fmt.Errorf("get submitted years: %w", err)
	}
	defer rows.Close()

	var years []int
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			return nil, fmt.Errorf("scan year: %w", err)
		}
		years = append(years, year)
	}

	return years, nil
}

// ListByAccount retrieves all Lohnzettel for an ELDA account
func (r *Repository) ListByAccount(ctx context.Context, eldaAccountID uuid.UUID, filter *ListFilter) ([]*elda.Lohnzettel, error) {
	query := `
		SELECT
			id, elda_account_id, year, sv_nummer, familienname, vorname, geburtsdatum,
			l16_data, status, protokollnummer, batch_id,
			submitted_at, error_message, error_code,
			is_berichtigung, berichtigt_id, created_by, created_at, updated_at
		FROM lohnzettel
		WHERE elda_account_id = $1
	`
	args := []interface{}{eldaAccountID}
	argIndex := 2

	if filter != nil {
		if filter.Year != nil {
			query += fmt.Sprintf(" AND year = $%d", argIndex)
			args = append(args, *filter.Year)
			argIndex++
		}
		if filter.Status != nil {
			query += fmt.Sprintf(" AND status = $%d", argIndex)
			args = append(args, *filter.Status)
			argIndex++
		}
	}

	query += " ORDER BY year DESC, familienname, vorname"

	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list lohnzettel: %w", err)
	}
	defer rows.Close()

	var results []*elda.Lohnzettel
	for rows.Next() {
		l := &elda.Lohnzettel{}
		var l16DataJSON []byte

		err := rows.Scan(
			&l.ID, &l.ELDAAccountID, &l.Year, &l.SVNummer, &l.Familienname, &l.Vorname, &l.Geburtsdatum,
			&l16DataJSON, &l.Status, &l.Protokollnummer, &l.BatchID,
			&l.SubmittedAt, &l.ErrorMessage, &l.ErrorCode,
			&l.IsBerichtigung, &l.BerichtigtID, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan lohnzettel: %w", err)
		}

		if len(l16DataJSON) > 0 {
			json.Unmarshal(l16DataJSON, &l.L16Data)
		}

		results = append(results, l)
	}

	return results, nil
}

// CreateBatch creates a new Lohnzettel batch
func (r *Repository) CreateBatch(ctx context.Context, batch *elda.LohnzettelBatch) error {
	query := `
		INSERT INTO lohnzettel_batches (
			id, elda_account_id, year,
			total_lohnzettel, submitted_count, accepted_count, rejected_count,
			status, created_by, created_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9, $10
		)
	`

	if batch.ID == uuid.Nil {
		batch.ID = uuid.New()
	}
	batch.CreatedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		batch.ID, batch.ELDAAccountID, batch.Year,
		batch.TotalLohnzettel, batch.SubmittedCount, batch.AcceptedCount, batch.RejectedCount,
		batch.Status, batch.CreatedBy, batch.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}

	return nil
}

// GetBatchByID retrieves a batch by ID
func (r *Repository) GetBatchByID(ctx context.Context, id uuid.UUID) (*elda.LohnzettelBatch, error) {
	query := `
		SELECT
			id, elda_account_id, year,
			total_lohnzettel, submitted_count, accepted_count, rejected_count,
			status, started_at, completed_at, created_by, created_at
		FROM lohnzettel_batches
		WHERE id = $1
	`

	batch := &elda.LohnzettelBatch{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&batch.ID, &batch.ELDAAccountID, &batch.Year,
		&batch.TotalLohnzettel, &batch.SubmittedCount, &batch.AcceptedCount, &batch.RejectedCount,
		&batch.Status, &batch.StartedAt, &batch.CompletedAt, &batch.CreatedBy, &batch.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBatchNotFound
		}
		return nil, fmt.Errorf("get batch: %w", err)
	}

	return batch, nil
}

// GetBatchWithLohnzettel retrieves a batch with all its Lohnzettel
func (r *Repository) GetBatchWithLohnzettel(ctx context.Context, id uuid.UUID) (*elda.LohnzettelBatch, error) {
	batch, err := r.GetBatchByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get Lohnzettel for this batch
	query := `
		SELECT
			id, elda_account_id, year, sv_nummer, familienname, vorname, geburtsdatum,
			l16_data, status, protokollnummer, batch_id,
			submitted_at, error_message, error_code,
			is_berichtigung, berichtigt_id, created_by, created_at, updated_at
		FROM lohnzettel
		WHERE batch_id = $1
		ORDER BY familienname, vorname
	`

	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("get batch lohnzettel: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		l := &elda.Lohnzettel{}
		var l16DataJSON []byte

		err := rows.Scan(
			&l.ID, &l.ELDAAccountID, &l.Year, &l.SVNummer, &l.Familienname, &l.Vorname, &l.Geburtsdatum,
			&l16DataJSON, &l.Status, &l.Protokollnummer, &l.BatchID,
			&l.SubmittedAt, &l.ErrorMessage, &l.ErrorCode,
			&l.IsBerichtigung, &l.BerichtigtID, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan batch lohnzettel: %w", err)
		}

		if len(l16DataJSON) > 0 {
			json.Unmarshal(l16DataJSON, &l.L16Data)
		}

		batch.Lohnzettel = append(batch.Lohnzettel, l)
	}

	return batch, nil
}

// UpdateBatch updates a batch record
func (r *Repository) UpdateBatch(ctx context.Context, batch *elda.LohnzettelBatch) error {
	query := `
		UPDATE lohnzettel_batches SET
			total_lohnzettel = $2,
			submitted_count = $3,
			accepted_count = $4,
			rejected_count = $5,
			status = $6,
			started_at = $7,
			completed_at = $8
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		batch.ID, batch.TotalLohnzettel, batch.SubmittedCount,
		batch.AcceptedCount, batch.RejectedCount, batch.Status,
		batch.StartedAt, batch.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("update batch: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBatchNotFound
	}

	return nil
}

// AssignToBatch assigns Lohnzettel to a batch
func (r *Repository) AssignToBatch(ctx context.Context, batchID uuid.UUID, lohnzettelIDs []uuid.UUID) error {
	query := `UPDATE lohnzettel SET batch_id = $1 WHERE id = ANY($2)`
	_, err := r.db.Exec(ctx, query, batchID, lohnzettelIDs)
	if err != nil {
		return fmt.Errorf("assign to batch: %w", err)
	}
	return nil
}

// BatchListFilter defines filter options for listing batches
type BatchListFilter struct {
	ELDAAccountID *uuid.UUID
	Year          *int
	Status        *elda.LohnzettelBatchStatus
	Limit         int
	Offset        int
}

// ListBatches retrieves batches with filters
func (r *Repository) ListBatches(ctx context.Context, filter BatchListFilter) ([]*elda.LohnzettelBatch, error) {
	query := `
		SELECT
			id, elda_account_id, year,
			total_lohnzettel, submitted_count, accepted_count, rejected_count,
			status, started_at, completed_at, created_by, created_at
		FROM lohnzettel_batches
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter.ELDAAccountID != nil {
		query += fmt.Sprintf(" AND elda_account_id = $%d", argIndex)
		args = append(args, *filter.ELDAAccountID)
		argIndex++
	}

	if filter.Year != nil {
		query += fmt.Sprintf(" AND year = $%d", argIndex)
		args = append(args, *filter.Year)
		argIndex++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list batches: %w", err)
	}
	defer rows.Close()

	var results []*elda.LohnzettelBatch
	for rows.Next() {
		batch := &elda.LohnzettelBatch{}
		err := rows.Scan(
			&batch.ID, &batch.ELDAAccountID, &batch.Year,
			&batch.TotalLohnzettel, &batch.SubmittedCount, &batch.AcceptedCount, &batch.RejectedCount,
			&batch.Status, &batch.StartedAt, &batch.CompletedAt, &batch.CreatedBy, &batch.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan batch: %w", err)
		}
		results = append(results, batch)
	}

	return results, nil
}

// GetDeadlineStatus returns Lohnzettel statistics for deadline monitoring
func (r *Repository) GetDeadlineStatus(ctx context.Context, eldaAccountID uuid.UUID, year int) (*DeadlineStatus, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'draft') as draft_count,
			COUNT(*) FILTER (WHERE status = 'validated') as validated_count,
			COUNT(*) FILTER (WHERE status = 'submitted') as submitted_count,
			COUNT(*) FILTER (WHERE status = 'accepted') as accepted_count,
			COUNT(*) FILTER (WHERE status = 'rejected') as rejected_count,
			COUNT(*) as total_count
		FROM lohnzettel
		WHERE elda_account_id = $1 AND year = $2
	`

	status := &DeadlineStatus{Year: year}
	err := r.db.QueryRow(ctx, query, eldaAccountID, year).Scan(
		&status.DraftCount,
		&status.ValidatedCount,
		&status.SubmittedCount,
		&status.AcceptedCount,
		&status.RejectedCount,
		&status.TotalCount,
	)
	if err != nil {
		return nil, fmt.Errorf("get deadline status: %w", err)
	}

	status.Deadline = elda.GetL16Deadline(year)
	status.DaysRemaining = elda.DaysUntilL16Deadline(year)
	status.IsOverdue = status.DaysRemaining == 0 && time.Now().After(status.Deadline)

	return status, nil
}

// DeadlineStatus contains L16 deadline status information
type DeadlineStatus struct {
	Year           int       `json:"year"`
	Deadline       time.Time `json:"deadline"`
	DaysRemaining  int       `json:"days_remaining"`
	IsOverdue      bool      `json:"is_overdue"`
	DraftCount     int       `json:"draft_count"`
	ValidatedCount int       `json:"validated_count"`
	SubmittedCount int       `json:"submitted_count"`
	AcceptedCount  int       `json:"accepted_count"`
	RejectedCount  int       `json:"rejected_count"`
	TotalCount     int       `json:"total_count"`
}
