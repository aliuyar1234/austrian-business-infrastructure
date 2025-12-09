package uid

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrValidationNotFound = errors.New("validation not found")
)

// Repository handles UID validation database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new UID validation repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new UID validation record
func (r *Repository) Create(ctx context.Context, v *Validation) (*Validation, error) {
	v.ID = uuid.New()
	v.CreatedAt = time.Now()
	v.ValidatedAt = time.Now()

	query := `
		INSERT INTO uid_validations (
			id, tenant_id, uid, country_code, valid, level,
			company_name, street, post_code, city, country,
			error_code, error_message, source, validated_at, validated_by,
			account_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		v.ID, v.TenantID, v.UID, v.CountryCode, v.Valid, v.Level,
		v.CompanyName, v.Street, v.PostCode, v.City, v.Country,
		v.ErrorCode, v.ErrorMessage, v.Source, v.ValidatedAt, v.ValidatedBy,
		v.AccountID, v.CreatedAt,
	).Scan(&v.ID, &v.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create validation: %w", err)
	}

	return v, nil
}

// GetByID retrieves a validation by ID
func (r *Repository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*Validation, error) {
	query := `
		SELECT id, tenant_id, uid, country_code, valid, level,
			company_name, street, post_code, city, country,
			error_code, error_message, source, validated_at, validated_by,
			account_id, created_at
		FROM uid_validations
		WHERE id = $1 AND tenant_id = $2`

	var v Validation
	var companyName, street, postCode, city, country, errorMessage sql.NullString
	var errorCode sql.NullInt32
	var validatedBy, accountID uuid.NullUUID

	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&v.ID, &v.TenantID, &v.UID, &v.CountryCode, &v.Valid, &v.Level,
		&companyName, &street, &postCode, &city, &country,
		&errorCode, &errorMessage, &v.Source, &v.ValidatedAt, &validatedBy,
		&accountID, &v.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrValidationNotFound
		}
		return nil, fmt.Errorf("failed to get validation: %w", err)
	}

	if companyName.Valid {
		v.CompanyName = &companyName.String
	}
	if street.Valid {
		v.Street = &street.String
	}
	if postCode.Valid {
		v.PostCode = &postCode.String
	}
	if city.Valid {
		v.City = &city.String
	}
	if country.Valid {
		v.Country = &country.String
	}
	if errorCode.Valid {
		c := int(errorCode.Int32)
		v.ErrorCode = &c
	}
	if errorMessage.Valid {
		v.ErrorMessage = &errorMessage.String
	}
	if validatedBy.Valid {
		v.ValidatedBy = &validatedBy.UUID
	}
	if accountID.Valid {
		v.AccountID = &accountID.UUID
	}

	return &v, nil
}

// List retrieves validations with filtering
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]*Validation, int, error) {
	baseQuery := `
		FROM uid_validations
		WHERE tenant_id = $1`

	args := []interface{}{filter.TenantID}
	argIdx := 2

	if filter.AccountID != nil {
		baseQuery += fmt.Sprintf(" AND account_id = $%d", argIdx)
		args = append(args, *filter.AccountID)
		argIdx++
	}

	if filter.UID != nil {
		baseQuery += fmt.Sprintf(" AND uid ILIKE $%d", argIdx)
		args = append(args, "%"+*filter.UID+"%")
		argIdx++
	}

	if filter.Valid != nil {
		baseQuery += fmt.Sprintf(" AND valid = $%d", argIdx)
		args = append(args, *filter.Valid)
		argIdx++
	}

	if filter.CountryCode != nil {
		baseQuery += fmt.Sprintf(" AND country_code = $%d", argIdx)
		args = append(args, *filter.CountryCode)
		argIdx++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND validated_at >= $%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND validated_at <= $%d", argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count validations: %w", err)
	}

	// Get paginated results
	selectQuery := `
		SELECT id, tenant_id, uid, country_code, valid, level,
			company_name, street, post_code, city, country,
			error_code, error_message, source, validated_at, validated_by,
			account_id, created_at
		` + baseQuery + `
		ORDER BY validated_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list validations: %w", err)
	}
	defer rows.Close()

	var validations []*Validation
	for rows.Next() {
		var v Validation
		var companyName, street, postCode, city, country, errorMessage sql.NullString
		var errorCode sql.NullInt32
		var validatedBy, accountID uuid.NullUUID

		err := rows.Scan(
			&v.ID, &v.TenantID, &v.UID, &v.CountryCode, &v.Valid, &v.Level,
			&companyName, &street, &postCode, &city, &country,
			&errorCode, &errorMessage, &v.Source, &v.ValidatedAt, &validatedBy,
			&accountID, &v.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan validation: %w", err)
		}

		if companyName.Valid {
			v.CompanyName = &companyName.String
		}
		if street.Valid {
			v.Street = &street.String
		}
		if postCode.Valid {
			v.PostCode = &postCode.String
		}
		if city.Valid {
			v.City = &city.String
		}
		if country.Valid {
			v.Country = &country.String
		}
		if errorCode.Valid {
			c := int(errorCode.Int32)
			v.ErrorCode = &c
		}
		if errorMessage.Valid {
			v.ErrorMessage = &errorMessage.String
		}
		if validatedBy.Valid {
			v.ValidatedBy = &validatedBy.UUID
		}
		if accountID.Valid {
			v.AccountID = &accountID.UUID
		}

		validations = append(validations, &v)
	}

	return validations, total, nil
}

// GetRecentByUID gets the most recent validation for a UID (for caching)
func (r *Repository) GetRecentByUID(ctx context.Context, tenantID uuid.UUID, uid string, maxAge time.Duration) (*Validation, error) {
	cutoff := time.Now().Add(-maxAge)

	query := `
		SELECT id, tenant_id, uid, country_code, valid, level,
			company_name, street, post_code, city, country,
			error_code, error_message, source, validated_at, validated_by,
			account_id, created_at
		FROM uid_validations
		WHERE tenant_id = $1 AND uid = $2 AND validated_at >= $3
		ORDER BY validated_at DESC
		LIMIT 1`

	var v Validation
	var companyName, street, postCode, city, country, errorMessage sql.NullString
	var errorCode sql.NullInt32
	var validatedBy, accountID uuid.NullUUID

	err := r.db.QueryRow(ctx, query, tenantID, uid, cutoff).Scan(
		&v.ID, &v.TenantID, &v.UID, &v.CountryCode, &v.Valid, &v.Level,
		&companyName, &street, &postCode, &city, &country,
		&errorCode, &errorMessage, &v.Source, &v.ValidatedAt, &validatedBy,
		&accountID, &v.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No recent validation found
		}
		return nil, fmt.Errorf("failed to get recent validation: %w", err)
	}

	if companyName.Valid {
		v.CompanyName = &companyName.String
	}
	if street.Valid {
		v.Street = &street.String
	}
	if postCode.Valid {
		v.PostCode = &postCode.String
	}
	if city.Valid {
		v.City = &city.String
	}
	if country.Valid {
		v.Country = &country.String
	}
	if errorCode.Valid {
		c := int(errorCode.Int32)
		v.ErrorCode = &c
	}
	if errorMessage.Valid {
		v.ErrorMessage = &errorMessage.String
	}
	if validatedBy.Valid {
		v.ValidatedBy = &validatedBy.UUID
	}
	if accountID.Valid {
		v.AccountID = &accountID.UUID
	}

	return &v, nil
}

// CountToday counts validations made today (for rate limiting)
func (r *Repository) CountToday(ctx context.Context, tenantID uuid.UUID) (int, error) {
	today := time.Now().Truncate(24 * time.Hour)

	query := `
		SELECT COUNT(*) FROM uid_validations
		WHERE tenant_id = $1 AND validated_at >= $2`

	var count int
	err := r.db.QueryRow(ctx, query, tenantID, today).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count today's validations: %w", err)
	}

	return count, nil
}
