package profil

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"austrian-business-infrastructure/internal/foerderung"
)

// Repository handles profile database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new profile repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new profile
func (r *Repository) Create(ctx context.Context, p *foerderung.Unternehmensprofil) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	if p.Status == "" {
		p.Status = foerderung.ProfileStatusDraft
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO unternehmensprofile (
			id, tenant_id, account_id, name, legal_form, founded_year, state, district,
			employees_count, annual_revenue, balance_total, industry, onace_codes,
			is_startup, project_description, investment_amount, project_topics,
			is_kmu, company_age_category, status, derived_from_account,
			last_search_at, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
		)
	`,
		p.ID, p.TenantID, p.AccountID, p.Name, p.LegalForm, p.FoundedYear, p.State, p.District,
		p.EmployeesCount, p.AnnualRevenue, p.BalanceTotal, p.Industry, p.OnaceCodes,
		p.IsStartup, p.ProjectDescription, p.InvestmentAmount, p.ProjectTopics,
		p.IsKMU, p.CompanyAgeCategory, p.Status, p.DerivedFromAccount,
		p.LastSearchAt, p.CreatedBy, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	return nil
}

// GetByID retrieves a profile by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*foerderung.Unternehmensprofil, error) {
	return r.scanProfile(r.db.QueryRow(ctx, `
		SELECT id, tenant_id, account_id, name, legal_form, founded_year, state, district,
			employees_count, annual_revenue, balance_total, industry, onace_codes,
			is_startup, project_description, investment_amount, project_topics,
			is_kmu, company_age_category, status, derived_from_account,
			last_search_at, created_by, created_at, updated_at
		FROM unternehmensprofile
		WHERE id = $1
	`, id))
}

// GetByIDAndTenant retrieves a profile ensuring tenant access
func (r *Repository) GetByIDAndTenant(ctx context.Context, id, tenantID uuid.UUID) (*foerderung.Unternehmensprofil, error) {
	return r.scanProfile(r.db.QueryRow(ctx, `
		SELECT id, tenant_id, account_id, name, legal_form, founded_year, state, district,
			employees_count, annual_revenue, balance_total, industry, onace_codes,
			is_startup, project_description, investment_amount, project_topics,
			is_kmu, company_age_category, status, derived_from_account,
			last_search_at, created_by, created_at, updated_at
		FROM unternehmensprofile
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID))
}

// ListByTenant retrieves all profiles for a tenant
func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*foerderung.Unternehmensprofil, int, error) {
	if limit <= 0 {
		limit = 20
	}

	// Count
	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM unternehmensprofile WHERE tenant_id = $1
	`, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count profiles: %w", err)
	}

	// List
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, account_id, name, legal_form, founded_year, state, district,
			employees_count, annual_revenue, balance_total, industry, onace_codes,
			is_startup, project_description, investment_amount, project_topics,
			is_kmu, company_age_category, status, derived_from_account,
			last_search_at, created_by, created_at, updated_at
		FROM unternehmensprofile
		WHERE tenant_id = $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list profiles: %w", err)
	}
	defer rows.Close()

	profiles := make([]*foerderung.Unternehmensprofil, 0)
	for rows.Next() {
		p, err := r.scanProfileFromRows(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan profile: %w", err)
		}
		profiles = append(profiles, p)
	}

	return profiles, total, nil
}

// ListByAccount retrieves all profiles for an account
func (r *Repository) ListByAccount(ctx context.Context, accountID uuid.UUID) ([]*foerderung.Unternehmensprofil, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, account_id, name, legal_form, founded_year, state, district,
			employees_count, annual_revenue, balance_total, industry, onace_codes,
			is_startup, project_description, investment_amount, project_topics,
			is_kmu, company_age_category, status, derived_from_account,
			last_search_at, created_by, created_at, updated_at
		FROM unternehmensprofile
		WHERE account_id = $1
		ORDER BY updated_at DESC
	`, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}
	defer rows.Close()

	profiles := make([]*foerderung.Unternehmensprofil, 0)
	for rows.Next() {
		p, err := r.scanProfileFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}
		profiles = append(profiles, p)
	}

	return profiles, nil
}

// Update updates a profile
func (r *Repository) Update(ctx context.Context, p *foerderung.Unternehmensprofil) error {
	p.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, `
		UPDATE unternehmensprofile SET
			name = $2, legal_form = $3, founded_year = $4, state = $5, district = $6,
			employees_count = $7, annual_revenue = $8, balance_total = $9,
			industry = $10, onace_codes = $11, is_startup = $12,
			project_description = $13, investment_amount = $14, project_topics = $15,
			is_kmu = $16, company_age_category = $17, status = $18,
			derived_from_account = $19, last_search_at = $20, updated_at = $21
		WHERE id = $1
	`,
		p.ID, p.Name, p.LegalForm, p.FoundedYear, p.State, p.District,
		p.EmployeesCount, p.AnnualRevenue, p.BalanceTotal,
		p.Industry, p.OnaceCodes, p.IsStartup,
		p.ProjectDescription, p.InvestmentAmount, p.ProjectTopics,
		p.IsKMU, p.CompanyAgeCategory, p.Status,
		p.DerivedFromAccount, p.LastSearchAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("profile not found")
	}

	return nil
}

// Delete deletes a profile
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM unternehmensprofile WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("profile not found")
	}

	return nil
}

// UpdateLastSearchAt updates the last search timestamp
func (r *Repository) UpdateLastSearchAt(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE unternehmensprofile SET last_search_at = $2, updated_at = $2 WHERE id = $1
	`, id, now)
	if err != nil {
		return fmt.Errorf("failed to update last search at: %w", err)
	}
	return nil
}

// scanProfile scans a single profile from a row
func (r *Repository) scanProfile(row pgx.Row) (*foerderung.Unternehmensprofil, error) {
	var p foerderung.Unternehmensprofil
	err := row.Scan(
		&p.ID, &p.TenantID, &p.AccountID, &p.Name, &p.LegalForm, &p.FoundedYear, &p.State, &p.District,
		&p.EmployeesCount, &p.AnnualRevenue, &p.BalanceTotal, &p.Industry, &p.OnaceCodes,
		&p.IsStartup, &p.ProjectDescription, &p.InvestmentAmount, &p.ProjectTopics,
		&p.IsKMU, &p.CompanyAgeCategory, &p.Status, &p.DerivedFromAccount,
		&p.LastSearchAt, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("profile not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan profile: %w", err)
	}
	return &p, nil
}

// scanProfileFromRows scans a profile from rows iterator
func (r *Repository) scanProfileFromRows(rows pgx.Rows) (*foerderung.Unternehmensprofil, error) {
	var p foerderung.Unternehmensprofil
	err := rows.Scan(
		&p.ID, &p.TenantID, &p.AccountID, &p.Name, &p.LegalForm, &p.FoundedYear, &p.State, &p.District,
		&p.EmployeesCount, &p.AnnualRevenue, &p.BalanceTotal, &p.Industry, &p.OnaceCodes,
		&p.IsStartup, &p.ProjectDescription, &p.InvestmentAmount, &p.ProjectTopics,
		&p.IsKMU, &p.CompanyAgeCategory, &p.Status, &p.DerivedFromAccount,
		&p.LastSearchAt, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
