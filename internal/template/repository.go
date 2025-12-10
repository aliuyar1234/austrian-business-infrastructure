package template

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles signature template database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new template repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new signature template
func (r *Repository) Create(ctx context.Context, template *SignatureTemplate) error {
	template.ID = uuid.New()
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	template.IsActive = true
	template.UsageCount = 0

	_, err := r.db.Exec(ctx, `
		INSERT INTO signature_templates (
			id, tenant_id, name, description, signers, fields, settings,
			is_active, usage_count, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		template.ID, template.TenantID, template.Name, template.Description,
		template.Signers, template.Fields, template.Settings,
		template.IsActive, template.UsageCount, template.CreatedBy,
		template.CreatedAt, template.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	return nil
}

// GetByID retrieves a template by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*SignatureTemplate, error) {
	var template SignatureTemplate
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, signers, fields, settings,
			   is_active, usage_count, created_by, created_at, updated_at
		FROM signature_templates
		WHERE id = $1
	`, id).Scan(
		&template.ID, &template.TenantID, &template.Name, &template.Description,
		&template.Signers, &template.Fields, &template.Settings,
		&template.IsActive, &template.UsageCount, &template.CreatedBy,
		&template.CreatedAt, &template.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &template, nil
}

// GetByIDAndTenant retrieves a template by ID ensuring tenant access
func (r *Repository) GetByIDAndTenant(ctx context.Context, id, tenantID uuid.UUID) (*SignatureTemplate, error) {
	var template SignatureTemplate
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, signers, fields, settings,
			   is_active, usage_count, created_by, created_at, updated_at
		FROM signature_templates
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(
		&template.ID, &template.TenantID, &template.Name, &template.Description,
		&template.Signers, &template.Fields, &template.Settings,
		&template.IsActive, &template.UsageCount, &template.CreatedBy,
		&template.CreatedAt, &template.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &template, nil
}

// ListByTenant retrieves all templates for a tenant
func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID, activeOnly bool, limit, offset int) ([]*SignatureTemplate, int, error) {
	// Get total count
	var countQuery string
	var countArgs []interface{}
	if activeOnly {
		countQuery = `SELECT COUNT(*) FROM signature_templates WHERE tenant_id = $1 AND is_active = true`
		countArgs = []interface{}{tenantID}
	} else {
		countQuery = `SELECT COUNT(*) FROM signature_templates WHERE tenant_id = $1`
		countArgs = []interface{}{tenantID}
	}

	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count templates: %w", err)
	}

	// Get templates
	var query string
	var args []interface{}
	if activeOnly {
		query = `
			SELECT id, tenant_id, name, description, signers, fields, settings,
				   is_active, usage_count, created_by, created_at, updated_at
			FROM signature_templates
			WHERE tenant_id = $1 AND is_active = true
			ORDER BY name ASC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{tenantID, limit, offset}
	} else {
		query = `
			SELECT id, tenant_id, name, description, signers, fields, settings,
				   is_active, usage_count, created_by, created_at, updated_at
			FROM signature_templates
			WHERE tenant_id = $1
			ORDER BY name ASC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{tenantID, limit, offset}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []*SignatureTemplate
	for rows.Next() {
		var template SignatureTemplate
		if err := rows.Scan(
			&template.ID, &template.TenantID, &template.Name, &template.Description,
			&template.Signers, &template.Fields, &template.Settings,
			&template.IsActive, &template.UsageCount, &template.CreatedBy,
			&template.CreatedAt, &template.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, &template)
	}

	return templates, total, nil
}

// Update updates a signature template
func (r *Repository) Update(ctx context.Context, template *SignatureTemplate) error {
	template.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, `
		UPDATE signature_templates
		SET name = $2, description = $3, signers = $4, fields = $5, settings = $6,
			is_active = $7, updated_at = $8
		WHERE id = $1
	`,
		template.ID, template.Name, template.Description,
		template.Signers, template.Fields, template.Settings,
		template.IsActive, template.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("template not found")
	}

	return nil
}

// Delete soft-deletes a template by setting is_active to false
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `
		UPDATE signature_templates SET is_active = false, updated_at = $2 WHERE id = $1
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("template not found")
	}

	return nil
}

// HardDelete permanently deletes a template
func (r *Repository) HardDelete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM signature_templates WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("template not found")
	}

	return nil
}

// IncrementUsageCount increments the usage count of a template
func (r *Repository) IncrementUsageCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE signature_templates
		SET usage_count = usage_count + 1, updated_at = $2
		WHERE id = $1
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to increment usage count: %w", err)
	}

	return nil
}

// Search searches templates by name or description
func (r *Repository) Search(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*SignatureTemplate, error) {
	searchPattern := "%" + query + "%"

	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, description, signers, fields, settings,
			   is_active, usage_count, created_by, created_at, updated_at
		FROM signature_templates
		WHERE tenant_id = $1 AND is_active = true
		  AND (name ILIKE $2 OR description ILIKE $2)
		ORDER BY usage_count DESC, name ASC
		LIMIT $3
	`, tenantID, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search templates: %w", err)
	}
	defer rows.Close()

	var templates []*SignatureTemplate
	for rows.Next() {
		var template SignatureTemplate
		if err := rows.Scan(
			&template.ID, &template.TenantID, &template.Name, &template.Description,
			&template.Signers, &template.Fields, &template.Settings,
			&template.IsActive, &template.UsageCount, &template.CreatedBy,
			&template.CreatedAt, &template.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, &template)
	}

	return templates, nil
}

// GetMostUsed retrieves the most frequently used templates
func (r *Repository) GetMostUsed(ctx context.Context, tenantID uuid.UUID, limit int) ([]*SignatureTemplate, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, description, signers, fields, settings,
			   is_active, usage_count, created_by, created_at, updated_at
		FROM signature_templates
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY usage_count DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get most used templates: %w", err)
	}
	defer rows.Close()

	var templates []*SignatureTemplate
	for rows.Next() {
		var template SignatureTemplate
		if err := rows.Scan(
			&template.ID, &template.TenantID, &template.Name, &template.Description,
			&template.Signers, &template.Fields, &template.Settings,
			&template.IsActive, &template.UsageCount, &template.CreatedBy,
			&template.CreatedAt, &template.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, &template)
	}

	return templates, nil
}
