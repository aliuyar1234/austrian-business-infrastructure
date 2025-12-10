package foerderung

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles Förderung database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new Förderung repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ============================================
// FÖRDERUNGEN CRUD
// ============================================

// Create creates a new Förderung
func (r *Repository) Create(ctx context.Context, f *Foerderung) error {
	f.ID = uuid.New()
	f.CreatedAt = time.Now()
	f.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		INSERT INTO foerderungen (
			id, name, short_name, description, provider, type,
			funding_rate_min, funding_rate_max, max_amount, min_amount,
			target_size, target_age, target_legal_forms, target_industries, target_states,
			topics, categories, requirements, eligibility_criteria,
			application_deadline, deadline_type, call_start, call_end,
			url, application_url, guideline_url,
			combinable_with, not_combinable_with,
			status, is_highlighted, source, source_id, last_updated_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35)
	`,
		f.ID, f.Name, f.ShortName, f.Description, f.Provider, f.Type,
		f.FundingRateMin, f.FundingRateMax, f.MaxAmount, f.MinAmount,
		f.TargetSize, f.TargetAge, f.TargetLegalForms, f.TargetIndustries, f.TargetStates,
		f.Topics, f.Categories, f.Requirements, f.EligibilityCriteria,
		f.ApplicationDeadline, f.DeadlineType, f.CallStart, f.CallEnd,
		f.URL, f.ApplicationURL, f.GuidelineURL,
		f.CombinableWith, f.NotCombinableWith,
		f.Status, f.IsHighlighted, f.Source, f.SourceID, f.LastUpdatedAt,
		f.CreatedAt, f.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create foerderung: %w", err)
	}

	return nil
}

// GetByID retrieves a Förderung by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Foerderung, error) {
	var f Foerderung
	err := r.db.QueryRow(ctx, `
		SELECT id, name, short_name, description, provider, type,
			funding_rate_min, funding_rate_max, max_amount, min_amount,
			target_size, target_age, target_legal_forms, target_industries, target_states,
			topics, categories, requirements, eligibility_criteria,
			application_deadline, deadline_type, call_start, call_end,
			url, application_url, guideline_url,
			combinable_with, not_combinable_with,
			status, is_highlighted, source, source_id, last_updated_at,
			created_at, updated_at
		FROM foerderungen
		WHERE id = $1
	`, id).Scan(
		&f.ID, &f.Name, &f.ShortName, &f.Description, &f.Provider, &f.Type,
		&f.FundingRateMin, &f.FundingRateMax, &f.MaxAmount, &f.MinAmount,
		&f.TargetSize, &f.TargetAge, &f.TargetLegalForms, &f.TargetIndustries, &f.TargetStates,
		&f.Topics, &f.Categories, &f.Requirements, &f.EligibilityCriteria,
		&f.ApplicationDeadline, &f.DeadlineType, &f.CallStart, &f.CallEnd,
		&f.URL, &f.ApplicationURL, &f.GuidelineURL,
		&f.CombinableWith, &f.NotCombinableWith,
		&f.Status, &f.IsHighlighted, &f.Source, &f.SourceID, &f.LastUpdatedAt,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("foerderung not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get foerderung: %w", err)
	}

	return &f, nil
}

// ListFilter defines filters for listing Förderungen
type ListFilter struct {
	Provider string
	Type     FoerderungType
	Status   FoerderungStatus
	State    string
	Topic    string
	Search   string
	Limit    int
	Offset   int
}

// List retrieves Förderungen with filters
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]*Foerderung, int, error) {
	// Set defaults
	if filter.Limit <= 0 {
		filter.Limit = 50
	}

	// Build query
	query := `
		SELECT id, name, short_name, description, provider, type,
			funding_rate_min, funding_rate_max, max_amount, min_amount,
			target_size, target_age, target_legal_forms, target_industries, target_states,
			topics, categories, requirements, eligibility_criteria,
			application_deadline, deadline_type, call_start, call_end,
			url, application_url, guideline_url,
			combinable_with, not_combinable_with,
			status, is_highlighted, source, source_id, last_updated_at,
			created_at, updated_at
		FROM foerderungen
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM foerderungen WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if filter.Provider != "" {
		query += fmt.Sprintf(" AND provider = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND provider = $%d", argIdx)
		args = append(args, filter.Provider)
		argIdx++
	}
	if filter.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, filter.Type)
		argIdx++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.State != "" {
		query += fmt.Sprintf(" AND $%d = ANY(target_states)", argIdx)
		countQuery += fmt.Sprintf(" AND $%d = ANY(target_states)", argIdx)
		args = append(args, filter.State)
		argIdx++
	}
	if filter.Topic != "" {
		query += fmt.Sprintf(" AND $%d = ANY(topics)", argIdx)
		countQuery += fmt.Sprintf(" AND $%d = ANY(topics)", argIdx)
		args = append(args, filter.Topic)
		argIdx++
	}
	if filter.Search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx)
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	// Get total count
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count foerderungen: %w", err)
	}

	// Add pagination
	query += fmt.Sprintf(" ORDER BY is_highlighted DESC, name ASC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list foerderungen: %w", err)
	}
	defer rows.Close()

	var foerderungen []*Foerderung
	for rows.Next() {
		var f Foerderung
		if err := rows.Scan(
			&f.ID, &f.Name, &f.ShortName, &f.Description, &f.Provider, &f.Type,
			&f.FundingRateMin, &f.FundingRateMax, &f.MaxAmount, &f.MinAmount,
			&f.TargetSize, &f.TargetAge, &f.TargetLegalForms, &f.TargetIndustries, &f.TargetStates,
			&f.Topics, &f.Categories, &f.Requirements, &f.EligibilityCriteria,
			&f.ApplicationDeadline, &f.DeadlineType, &f.CallStart, &f.CallEnd,
			&f.URL, &f.ApplicationURL, &f.GuidelineURL,
			&f.CombinableWith, &f.NotCombinableWith,
			&f.Status, &f.IsHighlighted, &f.Source, &f.SourceID, &f.LastUpdatedAt,
			&f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan foerderung: %w", err)
		}
		foerderungen = append(foerderungen, &f)
	}

	return foerderungen, total, nil
}

// ListActive retrieves all active Förderungen (for matching)
func (r *Repository) ListActive(ctx context.Context) ([]*Foerderung, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, short_name, description, provider, type,
			funding_rate_min, funding_rate_max, max_amount, min_amount,
			target_size, target_age, target_legal_forms, target_industries, target_states,
			topics, categories, requirements, eligibility_criteria,
			application_deadline, deadline_type, call_start, call_end,
			url, application_url, guideline_url,
			combinable_with, not_combinable_with,
			status, is_highlighted, source, source_id, last_updated_at,
			created_at, updated_at
		FROM foerderungen
		WHERE status = 'active'
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list active foerderungen: %w", err)
	}
	defer rows.Close()

	var foerderungen []*Foerderung
	for rows.Next() {
		var f Foerderung
		if err := rows.Scan(
			&f.ID, &f.Name, &f.ShortName, &f.Description, &f.Provider, &f.Type,
			&f.FundingRateMin, &f.FundingRateMax, &f.MaxAmount, &f.MinAmount,
			&f.TargetSize, &f.TargetAge, &f.TargetLegalForms, &f.TargetIndustries, &f.TargetStates,
			&f.Topics, &f.Categories, &f.Requirements, &f.EligibilityCriteria,
			&f.ApplicationDeadline, &f.DeadlineType, &f.CallStart, &f.CallEnd,
			&f.URL, &f.ApplicationURL, &f.GuidelineURL,
			&f.CombinableWith, &f.NotCombinableWith,
			&f.Status, &f.IsHighlighted, &f.Source, &f.SourceID, &f.LastUpdatedAt,
			&f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan foerderung: %w", err)
		}
		foerderungen = append(foerderungen, &f)
	}

	return foerderungen, nil
}

// Update updates a Förderung
func (r *Repository) Update(ctx context.Context, f *Foerderung) error {
	f.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, `
		UPDATE foerderungen SET
			name = $2, short_name = $3, description = $4, provider = $5, type = $6,
			funding_rate_min = $7, funding_rate_max = $8, max_amount = $9, min_amount = $10,
			target_size = $11, target_age = $12, target_legal_forms = $13, target_industries = $14, target_states = $15,
			topics = $16, categories = $17, requirements = $18, eligibility_criteria = $19,
			application_deadline = $20, deadline_type = $21, call_start = $22, call_end = $23,
			url = $24, application_url = $25, guideline_url = $26,
			combinable_with = $27, not_combinable_with = $28,
			status = $29, is_highlighted = $30, source = $31, source_id = $32, last_updated_at = $33,
			updated_at = $34
		WHERE id = $1
	`,
		f.ID, f.Name, f.ShortName, f.Description, f.Provider, f.Type,
		f.FundingRateMin, f.FundingRateMax, f.MaxAmount, f.MinAmount,
		f.TargetSize, f.TargetAge, f.TargetLegalForms, f.TargetIndustries, f.TargetStates,
		f.Topics, f.Categories, f.Requirements, f.EligibilityCriteria,
		f.ApplicationDeadline, f.DeadlineType, f.CallStart, f.CallEnd,
		f.URL, f.ApplicationURL, f.GuidelineURL,
		f.CombinableWith, f.NotCombinableWith,
		f.Status, f.IsHighlighted, f.Source, f.SourceID, f.LastUpdatedAt,
		f.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update foerderung: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("foerderung not found")
	}

	return nil
}

// Delete deletes a Förderung (soft delete by setting status to closed)
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `
		UPDATE foerderungen SET status = 'closed', updated_at = $2 WHERE id = $1
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete foerderung: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("foerderung not found")
	}

	return nil
}

// ExpireOverdue marks Förderungen past their deadline as closed
func (r *Repository) ExpireOverdue(ctx context.Context) (int, error) {
	result, err := r.db.Exec(ctx, `
		UPDATE foerderungen
		SET status = 'closed', updated_at = NOW()
		WHERE status = 'active'
		  AND application_deadline IS NOT NULL
		  AND application_deadline < CURRENT_DATE
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to expire foerderungen: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// GetBySourceID retrieves a Förderung by source and source_id (for imports)
func (r *Repository) GetBySourceID(ctx context.Context, source, sourceID string) (*Foerderung, error) {
	var f Foerderung
	err := r.db.QueryRow(ctx, `
		SELECT id, name, short_name, description, provider, type,
			funding_rate_min, funding_rate_max, max_amount, min_amount,
			target_size, target_age, target_legal_forms, target_industries, target_states,
			topics, categories, requirements, eligibility_criteria,
			application_deadline, deadline_type, call_start, call_end,
			url, application_url, guideline_url,
			combinable_with, not_combinable_with,
			status, is_highlighted, source, source_id, last_updated_at,
			created_at, updated_at
		FROM foerderungen
		WHERE source = $1 AND source_id = $2
	`, source, sourceID).Scan(
		&f.ID, &f.Name, &f.ShortName, &f.Description, &f.Provider, &f.Type,
		&f.FundingRateMin, &f.FundingRateMax, &f.MaxAmount, &f.MinAmount,
		&f.TargetSize, &f.TargetAge, &f.TargetLegalForms, &f.TargetIndustries, &f.TargetStates,
		&f.Topics, &f.Categories, &f.Requirements, &f.EligibilityCriteria,
		&f.ApplicationDeadline, &f.DeadlineType, &f.CallStart, &f.CallEnd,
		&f.URL, &f.ApplicationURL, &f.GuidelineURL,
		&f.CombinableWith, &f.NotCombinableWith,
		&f.Status, &f.IsHighlighted, &f.Source, &f.SourceID, &f.LastUpdatedAt,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil // Not found is not an error for import checking
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get foerderung by source id: %w", err)
	}

	return &f, nil
}

// GetStats retrieves Förderung statistics
func (r *Repository) GetStats(ctx context.Context) (*FoerderungStats, error) {
	var stats FoerderungStats

	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'active') AS active,
			COUNT(*) FILTER (WHERE status = 'upcoming') AS upcoming,
			COUNT(*) FILTER (WHERE status = 'closed') AS closed,
			COUNT(DISTINCT provider) AS providers
		FROM foerderungen
	`).Scan(&stats.Total, &stats.Active, &stats.Upcoming, &stats.Closed, &stats.Providers)
	if err != nil {
		return nil, fmt.Errorf("failed to get foerderung stats: %w", err)
	}

	return &stats, nil
}

// FoerderungStats holds aggregate statistics
type FoerderungStats struct {
	Total     int `json:"total"`
	Active    int `json:"active"`
	Upcoming  int `json:"upcoming"`
	Closed    int `json:"closed"`
	Providers int `json:"providers"`
}
