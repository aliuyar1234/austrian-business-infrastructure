package matcher

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"austrian-business-infrastructure/internal/foerderung"
)

// SearchRepository handles search database operations
type SearchRepository struct {
	db *pgxpool.Pool
}

// NewSearchRepository creates a new search repository
func NewSearchRepository(db *pgxpool.Pool) *SearchRepository {
	return &SearchRepository{db: db}
}

// Create creates a new search record
func (r *SearchRepository) Create(ctx context.Context, s *foerderung.FoerderungsSuche) error {
	s.ID = uuid.New()
	s.CreatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		INSERT INTO foerderungs_suchen (
			id, tenant_id, profile_id, total_foerderungen, total_matches, matches,
			status, phase, progress, started_at, completed_at, error_message,
			llm_tokens_used, llm_cost_cents, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`,
		s.ID, s.TenantID, s.ProfileID, s.TotalFoerderungen, s.TotalMatches, s.Matches,
		s.Status, s.Phase, s.Progress, s.StartedAt, s.CompletedAt, s.ErrorMessage,
		s.LLMTokensUsed, s.LLMCostCents, s.CreatedBy, s.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create search: %w", err)
	}

	return nil
}

// GetByID retrieves a search by ID
func (r *SearchRepository) GetByID(ctx context.Context, id uuid.UUID) (*foerderung.FoerderungsSuche, error) {
	var s foerderung.FoerderungsSuche
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, profile_id, total_foerderungen, total_matches, matches,
			status, phase, progress, started_at, completed_at, error_message,
			llm_tokens_used, llm_cost_cents, created_by, created_at
		FROM foerderungs_suchen
		WHERE id = $1
	`, id).Scan(
		&s.ID, &s.TenantID, &s.ProfileID, &s.TotalFoerderungen, &s.TotalMatches, &s.Matches,
		&s.Status, &s.Phase, &s.Progress, &s.StartedAt, &s.CompletedAt, &s.ErrorMessage,
		&s.LLMTokensUsed, &s.LLMCostCents, &s.CreatedBy, &s.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("search not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get search: %w", err)
	}

	return &s, nil
}

// GetByIDAndTenant retrieves a search ensuring tenant access
func (r *SearchRepository) GetByIDAndTenant(ctx context.Context, id, tenantID uuid.UUID) (*foerderung.FoerderungsSuche, error) {
	var s foerderung.FoerderungsSuche
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, profile_id, total_foerderungen, total_matches, matches,
			status, phase, progress, started_at, completed_at, error_message,
			llm_tokens_used, llm_cost_cents, created_by, created_at
		FROM foerderungs_suchen
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(
		&s.ID, &s.TenantID, &s.ProfileID, &s.TotalFoerderungen, &s.TotalMatches, &s.Matches,
		&s.Status, &s.Phase, &s.Progress, &s.StartedAt, &s.CompletedAt, &s.ErrorMessage,
		&s.LLMTokensUsed, &s.LLMCostCents, &s.CreatedBy, &s.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("search not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get search: %w", err)
	}

	return &s, nil
}

// ListByProfile retrieves all searches for a profile
func (r *SearchRepository) ListByProfile(ctx context.Context, profileID uuid.UUID, limit, offset int) ([]*foerderung.FoerderungsSuche, int, error) {
	if limit <= 0 {
		limit = 20
	}

	// Count
	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM foerderungs_suchen WHERE profile_id = $1
	`, profileID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count searches: %w", err)
	}

	// List
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, profile_id, total_foerderungen, total_matches, matches,
			status, phase, progress, started_at, completed_at, error_message,
			llm_tokens_used, llm_cost_cents, created_by, created_at
		FROM foerderungs_suchen
		WHERE profile_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, profileID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list searches: %w", err)
	}
	defer rows.Close()

	var searches []*foerderung.FoerderungsSuche
	for rows.Next() {
		var s foerderung.FoerderungsSuche
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.ProfileID, &s.TotalFoerderungen, &s.TotalMatches, &s.Matches,
			&s.Status, &s.Phase, &s.Progress, &s.StartedAt, &s.CompletedAt, &s.ErrorMessage,
			&s.LLMTokensUsed, &s.LLMCostCents, &s.CreatedBy, &s.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan search: %w", err)
		}
		searches = append(searches, &s)
	}

	return searches, total, nil
}

// ListByTenant retrieves all searches for a tenant
func (r *SearchRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*foerderung.FoerderungsSuche, int, error) {
	if limit <= 0 {
		limit = 20
	}

	// Count
	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM foerderungs_suchen WHERE tenant_id = $1
	`, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count searches: %w", err)
	}

	// List
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, profile_id, total_foerderungen, total_matches, matches,
			status, phase, progress, started_at, completed_at, error_message,
			llm_tokens_used, llm_cost_cents, created_by, created_at
		FROM foerderungs_suchen
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list searches: %w", err)
	}
	defer rows.Close()

	var searches []*foerderung.FoerderungsSuche
	for rows.Next() {
		var s foerderung.FoerderungsSuche
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.ProfileID, &s.TotalFoerderungen, &s.TotalMatches, &s.Matches,
			&s.Status, &s.Phase, &s.Progress, &s.StartedAt, &s.CompletedAt, &s.ErrorMessage,
			&s.LLMTokensUsed, &s.LLMCostCents, &s.CreatedBy, &s.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan search: %w", err)
		}
		searches = append(searches, &s)
	}

	return searches, total, nil
}

// Update updates a search record
func (r *SearchRepository) Update(ctx context.Context, s *foerderung.FoerderungsSuche) error {
	result, err := r.db.Exec(ctx, `
		UPDATE foerderungs_suchen SET
			total_foerderungen = $2, total_matches = $3, matches = $4,
			status = $5, phase = $6, progress = $7,
			started_at = $8, completed_at = $9, error_message = $10,
			llm_tokens_used = $11, llm_cost_cents = $12
		WHERE id = $1
	`,
		s.ID, s.TotalFoerderungen, s.TotalMatches, s.Matches,
		s.Status, s.Phase, s.Progress,
		s.StartedAt, s.CompletedAt, s.ErrorMessage,
		s.LLMTokensUsed, s.LLMCostCents,
	)
	if err != nil {
		return fmt.Errorf("failed to update search: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("search not found")
	}

	return nil
}

// UpdateStatus updates just the status and progress
func (r *SearchRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, progress int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE foerderungs_suchen SET status = $2, phase = $2, progress = $3 WHERE id = $1
	`, id, status, progress)
	if err != nil {
		return fmt.Errorf("failed to update search status: %w", err)
	}

	return nil
}

// Delete deletes a search record
func (r *SearchRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM foerderungs_suchen WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete search: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("search not found")
	}

	return nil
}
