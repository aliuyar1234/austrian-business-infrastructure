package sigfield

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles signature field database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new signature field repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateField creates a new signature field
func (r *Repository) CreateField(ctx context.Context, field *SignatureField) error {
	field.ID = uuid.New()
	field.CreatedAt = time.Now()
	field.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		INSERT INTO signature_fields (
			id, document_id, tenant_id, signer_id, page, x, y, width, height,
			field_name, required, show_name, show_date, show_reason, custom_text,
			font_size, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`,
		field.ID, field.DocumentID, field.TenantID, field.SignerID,
		field.Page, field.X, field.Y, field.Width, field.Height,
		field.FieldName, field.Required, field.ShowName, field.ShowDate, field.ShowReason,
		field.CustomText, field.FontSize, field.CreatedAt, field.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create signature field: %w", err)
	}

	return nil
}

// GetFieldByID retrieves a signature field by ID
func (r *Repository) GetFieldByID(ctx context.Context, id uuid.UUID) (*SignatureField, error) {
	var field SignatureField
	err := r.db.QueryRow(ctx, `
		SELECT id, document_id, tenant_id, signer_id, page, x, y, width, height,
			   field_name, required, show_name, show_date, show_reason, custom_text,
			   font_size, created_at, updated_at
		FROM signature_fields
		WHERE id = $1
	`, id).Scan(
		&field.ID, &field.DocumentID, &field.TenantID, &field.SignerID,
		&field.Page, &field.X, &field.Y, &field.Width, &field.Height,
		&field.FieldName, &field.Required, &field.ShowName, &field.ShowDate, &field.ShowReason,
		&field.CustomText, &field.FontSize, &field.CreatedAt, &field.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("signature field not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get signature field: %w", err)
	}

	return &field, nil
}

// ListFieldsByDocument retrieves all signature fields for a document
func (r *Repository) ListFieldsByDocument(ctx context.Context, documentID uuid.UUID) ([]*SignatureField, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, document_id, tenant_id, signer_id, page, x, y, width, height,
			   field_name, required, show_name, show_date, show_reason, custom_text,
			   font_size, created_at, updated_at
		FROM signature_fields
		WHERE document_id = $1
		ORDER BY page, y DESC, x
	`, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list signature fields: %w", err)
	}
	defer rows.Close()

	var fields []*SignatureField
	for rows.Next() {
		var field SignatureField
		if err := rows.Scan(
			&field.ID, &field.DocumentID, &field.TenantID, &field.SignerID,
			&field.Page, &field.X, &field.Y, &field.Width, &field.Height,
			&field.FieldName, &field.Required, &field.ShowName, &field.ShowDate, &field.ShowReason,
			&field.CustomText, &field.FontSize, &field.CreatedAt, &field.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan signature field: %w", err)
		}
		fields = append(fields, &field)
	}

	return fields, nil
}

// ListFieldsBySigner retrieves all signature fields assigned to a signer
func (r *Repository) ListFieldsBySigner(ctx context.Context, signerID uuid.UUID) ([]*SignatureField, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, document_id, tenant_id, signer_id, page, x, y, width, height,
			   field_name, required, show_name, show_date, show_reason, custom_text,
			   font_size, created_at, updated_at
		FROM signature_fields
		WHERE signer_id = $1
		ORDER BY page, y DESC, x
	`, signerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list signature fields: %w", err)
	}
	defer rows.Close()

	var fields []*SignatureField
	for rows.Next() {
		var field SignatureField
		if err := rows.Scan(
			&field.ID, &field.DocumentID, &field.TenantID, &field.SignerID,
			&field.Page, &field.X, &field.Y, &field.Width, &field.Height,
			&field.FieldName, &field.Required, &field.ShowName, &field.ShowDate, &field.ShowReason,
			&field.CustomText, &field.FontSize, &field.CreatedAt, &field.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan signature field: %w", err)
		}
		fields = append(fields, &field)
	}

	return fields, nil
}

// UpdateField updates a signature field
func (r *Repository) UpdateField(ctx context.Context, field *SignatureField) error {
	field.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, `
		UPDATE signature_fields
		SET signer_id = $2, page = $3, x = $4, y = $5, width = $6, height = $7,
			field_name = $8, required = $9, show_name = $10, show_date = $11,
			show_reason = $12, custom_text = $13, font_size = $14, updated_at = $15
		WHERE id = $1
	`,
		field.ID, field.SignerID, field.Page, field.X, field.Y, field.Width, field.Height,
		field.FieldName, field.Required, field.ShowName, field.ShowDate,
		field.ShowReason, field.CustomText, field.FontSize, field.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update signature field: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("signature field not found")
	}

	return nil
}

// DeleteField deletes a signature field
func (r *Repository) DeleteField(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM signature_fields WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete signature field: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("signature field not found")
	}

	return nil
}

// DeleteFieldsByDocument deletes all signature fields for a document
func (r *Repository) DeleteFieldsByDocument(ctx context.Context, documentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM signature_fields WHERE document_id = $1`, documentID)
	if err != nil {
		return fmt.Errorf("failed to delete signature fields: %w", err)
	}

	return nil
}

// AssignFieldToSigner assigns a signature field to a specific signer
func (r *Repository) AssignFieldToSigner(ctx context.Context, fieldID, signerID uuid.UUID) error {
	result, err := r.db.Exec(ctx, `
		UPDATE signature_fields SET signer_id = $2, updated_at = $3 WHERE id = $1
	`, fieldID, signerID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to assign field to signer: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("signature field not found")
	}

	return nil
}

// CopyFieldsToDocument copies signature fields from a template document to a new document
func (r *Repository) CopyFieldsToDocument(ctx context.Context, sourceDocID, targetDocID, tenantID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO signature_fields (
			id, document_id, tenant_id, signer_id, page, x, y, width, height,
			field_name, required, show_name, show_date, show_reason, custom_text,
			font_size, created_at, updated_at
		)
		SELECT
			gen_random_uuid(), $2, $3, signer_id, page, x, y, width, height,
			field_name, required, show_name, show_date, show_reason, custom_text,
			font_size, NOW(), NOW()
		FROM signature_fields
		WHERE document_id = $1
	`, sourceDocID, targetDocID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to copy signature fields: %w", err)
	}

	return nil
}
