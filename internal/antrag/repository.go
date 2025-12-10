package antrag

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"austrian-business-infrastructure/internal/foerderung"
)

// Repository handles application (Antrag) database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new application repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new application
func (r *Repository) Create(ctx context.Context, a *foerderung.FoerderungsAntrag) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()

	attachmentsJSON, _ := json.Marshal(a.Attachments)
	timelineJSON, _ := json.Marshal(a.Timeline)

	_, err := r.db.Exec(ctx, `
		INSERT INTO foerderungs_antraege (
			id, tenant_id, profile_id, foerderung_id,
			status, internal_reference, submitted_at,
			requested_amount, approved_amount,
			decision_date, decision_notes,
			attachments, timeline, notes,
			created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`,
		a.ID, a.TenantID, a.ProfileID, a.FoerderungID,
		a.Status, a.InternalReference, a.SubmittedAt,
		a.RequestedAmount, a.ApprovedAmount,
		a.DecisionDate, a.DecisionNotes,
		attachmentsJSON, timelineJSON, a.Notes,
		a.CreatedBy, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create antrag: %w", err)
	}

	return nil
}

// GetByID retrieves an application by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*foerderung.FoerderungsAntrag, error) {
	var a foerderung.FoerderungsAntrag
	var attachmentsJSON, timelineJSON []byte

	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, profile_id, foerderung_id,
			status, internal_reference, submitted_at,
			requested_amount, approved_amount,
			decision_date, decision_notes,
			attachments, timeline, notes,
			created_by, created_at, updated_at
		FROM foerderungs_antraege
		WHERE id = $1
	`, id).Scan(
		&a.ID, &a.TenantID, &a.ProfileID, &a.FoerderungID,
		&a.Status, &a.InternalReference, &a.SubmittedAt,
		&a.RequestedAmount, &a.ApprovedAmount,
		&a.DecisionDate, &a.DecisionNotes,
		&attachmentsJSON, &timelineJSON, &a.Notes,
		&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("antrag not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get antrag: %w", err)
	}

	if attachmentsJSON != nil {
		json.Unmarshal(attachmentsJSON, &a.Attachments)
	}
	if timelineJSON != nil {
		json.Unmarshal(timelineJSON, &a.Timeline)
	}

	return &a, nil
}

// GetByIDAndTenant retrieves an application ensuring tenant access
func (r *Repository) GetByIDAndTenant(ctx context.Context, id, tenantID uuid.UUID) (*foerderung.FoerderungsAntrag, error) {
	var a foerderung.FoerderungsAntrag
	var attachmentsJSON, timelineJSON []byte

	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, profile_id, foerderung_id,
			status, internal_reference, submitted_at,
			requested_amount, approved_amount,
			decision_date, decision_notes,
			attachments, timeline, notes,
			created_by, created_at, updated_at
		FROM foerderungs_antraege
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(
		&a.ID, &a.TenantID, &a.ProfileID, &a.FoerderungID,
		&a.Status, &a.InternalReference, &a.SubmittedAt,
		&a.RequestedAmount, &a.ApprovedAmount,
		&a.DecisionDate, &a.DecisionNotes,
		&attachmentsJSON, &timelineJSON, &a.Notes,
		&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("antrag not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get antrag: %w", err)
	}

	if attachmentsJSON != nil {
		json.Unmarshal(attachmentsJSON, &a.Attachments)
	}
	if timelineJSON != nil {
		json.Unmarshal(timelineJSON, &a.Timeline)
	}

	return &a, nil
}

// ListFilter defines filters for listing applications
type ListFilter struct {
	TenantID     uuid.UUID
	ProfileID    *uuid.UUID
	FoerderungID *uuid.UUID
	Status       string
	Limit        int
	Offset       int
}

// List retrieves applications with filters
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]*foerderung.FoerderungsAntrag, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	// Build query
	query := `
		SELECT id, tenant_id, profile_id, foerderung_id,
			status, internal_reference, submitted_at,
			requested_amount, approved_amount,
			decision_date, decision_notes,
			attachments, timeline, notes,
			created_by, created_at, updated_at
		FROM foerderungs_antraege
		WHERE tenant_id = $1
	`
	countQuery := `SELECT COUNT(*) FROM foerderungs_antraege WHERE tenant_id = $1`

	args := []interface{}{filter.TenantID}
	argIdx := 2

	if filter.ProfileID != nil {
		query += fmt.Sprintf(" AND profile_id = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND profile_id = $%d", argIdx)
		args = append(args, *filter.ProfileID)
		argIdx++
	}
	if filter.FoerderungID != nil {
		query += fmt.Sprintf(" AND foerderung_id = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND foerderung_id = $%d", argIdx)
		args = append(args, *filter.FoerderungID)
		argIdx++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	// Get total count
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count antraege: %w", err)
	}

	// Add pagination
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list antraege: %w", err)
	}
	defer rows.Close()

	antraege := make([]*foerderung.FoerderungsAntrag, 0)
	for rows.Next() {
		var a foerderung.FoerderungsAntrag
		var attachmentsJSON, timelineJSON []byte

		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.ProfileID, &a.FoerderungID,
			&a.Status, &a.InternalReference, &a.SubmittedAt,
			&a.RequestedAmount, &a.ApprovedAmount,
			&a.DecisionDate, &a.DecisionNotes,
			&attachmentsJSON, &timelineJSON, &a.Notes,
			&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan antrag: %w", err)
		}

		if attachmentsJSON != nil {
			json.Unmarshal(attachmentsJSON, &a.Attachments)
		}
		if timelineJSON != nil {
			json.Unmarshal(timelineJSON, &a.Timeline)
		}

		antraege = append(antraege, &a)
	}

	return antraege, total, nil
}

// Update updates an application
func (r *Repository) Update(ctx context.Context, a *foerderung.FoerderungsAntrag) error {
	a.UpdatedAt = time.Now()

	attachmentsJSON, _ := json.Marshal(a.Attachments)
	timelineJSON, _ := json.Marshal(a.Timeline)

	result, err := r.db.Exec(ctx, `
		UPDATE foerderungs_antraege SET
			status = $2, internal_reference = $3, submitted_at = $4,
			requested_amount = $5, approved_amount = $6,
			decision_date = $7, decision_notes = $8,
			attachments = $9, timeline = $10, notes = $11,
			updated_at = $12
		WHERE id = $1
	`,
		a.ID, a.Status, a.InternalReference, a.SubmittedAt,
		a.RequestedAmount, a.ApprovedAmount,
		a.DecisionDate, a.DecisionNotes,
		attachmentsJSON, timelineJSON, a.Notes,
		a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update antrag: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("antrag not found")
	}

	return nil
}

// Delete deletes an application
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM foerderungs_antraege WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete antrag: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("antrag not found")
	}

	return nil
}

// GetStats retrieves application statistics for a tenant
func (r *Repository) GetStats(ctx context.Context, tenantID uuid.UUID) (*AntragStats, error) {
	var stats AntragStats

	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'planned') AS planned,
			COUNT(*) FILTER (WHERE status = 'drafting') AS drafting,
			COUNT(*) FILTER (WHERE status = 'submitted') AS submitted,
			COUNT(*) FILTER (WHERE status = 'in_review') AS in_review,
			COUNT(*) FILTER (WHERE status = 'approved') AS approved,
			COUNT(*) FILTER (WHERE status = 'rejected') AS rejected,
			COALESCE(SUM(requested_amount), 0) AS total_requested,
			COALESCE(SUM(approved_amount), 0) AS total_approved
		FROM foerderungs_antraege
		WHERE tenant_id = $1
	`, tenantID).Scan(
		&stats.Total, &stats.Planned, &stats.Drafting,
		&stats.Submitted, &stats.InReview, &stats.Approved, &stats.Rejected,
		&stats.TotalRequested, &stats.TotalApproved,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get antrag stats: %w", err)
	}

	// Calculate success rate
	decided := stats.Approved + stats.Rejected
	if decided > 0 {
		stats.SuccessRate = float64(stats.Approved) / float64(decided) * 100
	}

	return &stats, nil
}

// AntragStats holds aggregate application statistics
type AntragStats struct {
	Total          int     `json:"total"`
	Planned        int     `json:"planned"`
	Drafting       int     `json:"drafting"`
	Submitted      int     `json:"submitted"`
	InReview       int     `json:"in_review"`
	Approved       int     `json:"approved"`
	Rejected       int     `json:"rejected"`
	TotalRequested int     `json:"total_requested"`
	TotalApproved  int     `json:"total_approved"`
	SuccessRate    float64 `json:"success_rate"`
}
