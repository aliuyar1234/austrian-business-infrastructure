package watchlist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository errors
var (
	ErrWatchlistItemNotFound = errors.New("watchlist item not found")
	ErrDuplicateCompany      = errors.New("company already in watchlist")
)

// Item represents a watchlist entry
type Item struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	AccountID      *uuid.UUID      `json:"account_id,omitempty"`
	CompanyNumber  string          `json:"company_number"` // FN number e.g., "FN123456d"
	CompanyName    string          `json:"company_name"`
	LastSnapshot   json.RawMessage `json:"last_snapshot,omitempty"`
	LastCheckedAt  *time.Time      `json:"last_checked_at,omitempty"`
	LastChangedAt  *time.Time      `json:"last_changed_at,omitempty"`
	CheckEnabled   bool            `json:"check_enabled"`
	NotifyOnChange bool            `json:"notify_on_change"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Repository handles watchlist database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new watchlist repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new watchlist item
func (r *Repository) Create(ctx context.Context, item *Item) error {
	now := time.Now()
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	item.CreatedAt = now
	item.UpdatedAt = now

	query := `
		INSERT INTO watchlist (
			id, tenant_id, account_id, company_number, company_name,
			check_enabled, notify_on_change, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(ctx, query,
		item.ID, item.TenantID, item.AccountID, item.CompanyNumber, item.CompanyName,
		item.CheckEnabled, item.NotifyOnChange, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		// Check for unique constraint violation
		if isDuplicateError(err) {
			return ErrDuplicateCompany
		}
		return fmt.Errorf("create watchlist item: %w", err)
	}

	return nil
}

// GetByID retrieves a watchlist item by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Item, error) {
	query := `
		SELECT id, tenant_id, account_id, company_number, company_name,
		       last_snapshot, last_checked_at, last_changed_at, check_enabled,
		       notify_on_change, created_at, updated_at
		FROM watchlist WHERE id = $1
	`

	item := &Item{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID, &item.TenantID, &item.AccountID, &item.CompanyNumber, &item.CompanyName,
		&item.LastSnapshot, &item.LastCheckedAt, &item.LastChangedAt, &item.CheckEnabled,
		&item.NotifyOnChange, &item.CreatedAt, &item.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWatchlistItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get watchlist item: %w", err)
	}

	return item, nil
}

// List retrieves watchlist items for a tenant
func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, enabledOnly bool) ([]*Item, error) {
	query := `
		SELECT id, tenant_id, account_id, company_number, company_name,
		       last_snapshot, last_checked_at, last_changed_at, check_enabled,
		       notify_on_change, created_at, updated_at
		FROM watchlist WHERE tenant_id = $1
	`

	if enabledOnly {
		query += " AND check_enabled = TRUE"
	}

	query += " ORDER BY company_name"

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list watchlist items: %w", err)
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		item := &Item{}
		err := rows.Scan(
			&item.ID, &item.TenantID, &item.AccountID, &item.CompanyNumber, &item.CompanyName,
			&item.LastSnapshot, &item.LastCheckedAt, &item.LastChangedAt, &item.CheckEnabled,
			&item.NotifyOnChange, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan watchlist item: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// ListAllEnabled retrieves all enabled watchlist items across all tenants
func (r *Repository) ListAllEnabled(ctx context.Context) ([]*Item, error) {
	query := `
		SELECT id, tenant_id, account_id, company_number, company_name,
		       last_snapshot, last_checked_at, last_changed_at, check_enabled,
		       notify_on_change, created_at, updated_at
		FROM watchlist WHERE check_enabled = TRUE
		ORDER BY tenant_id, company_name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list all enabled watchlist items: %w", err)
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		item := &Item{}
		err := rows.Scan(
			&item.ID, &item.TenantID, &item.AccountID, &item.CompanyNumber, &item.CompanyName,
			&item.LastSnapshot, &item.LastCheckedAt, &item.LastChangedAt, &item.CheckEnabled,
			&item.NotifyOnChange, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan watchlist item: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// Update updates a watchlist item
func (r *Repository) Update(ctx context.Context, item *Item) error {
	item.UpdatedAt = time.Now()

	query := `
		UPDATE watchlist SET
			company_name = $1, check_enabled = $2, notify_on_change = $3,
			account_id = $4, updated_at = $5
		WHERE id = $6 AND tenant_id = $7
	`

	result, err := r.db.Exec(ctx, query,
		item.CompanyName, item.CheckEnabled, item.NotifyOnChange,
		item.AccountID, item.UpdatedAt,
		item.ID, item.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update watchlist item: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrWatchlistItemNotFound
	}

	return nil
}

// UpdateSnapshot updates the snapshot data for a watchlist item
func (r *Repository) UpdateSnapshot(ctx context.Context, id uuid.UUID, snapshot json.RawMessage, changed bool) error {
	now := time.Now()

	query := `
		UPDATE watchlist SET
			last_snapshot = $1, last_checked_at = $2, updated_at = $2
	`

	args := []interface{}{snapshot, now}

	if changed {
		query += `, last_changed_at = $3 WHERE id = $4`
		args = append(args, now, id)
	} else {
		query += ` WHERE id = $3`
		args = append(args, id)
	}

	_, err := r.db.Exec(ctx, query, args...)
	return err
}

// Delete deletes a watchlist item
func (r *Repository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `DELETE FROM watchlist WHERE id = $1 AND tenant_id = $2`
	result, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete watchlist item: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrWatchlistItemNotFound
	}

	return nil
}

// GetByCompanyNumber retrieves a watchlist item by company number and tenant
func (r *Repository) GetByCompanyNumber(ctx context.Context, tenantID uuid.UUID, companyNumber string) (*Item, error) {
	query := `
		SELECT id, tenant_id, account_id, company_number, company_name,
		       last_snapshot, last_checked_at, last_changed_at, check_enabled,
		       notify_on_change, created_at, updated_at
		FROM watchlist WHERE tenant_id = $1 AND company_number = $2
	`

	item := &Item{}
	err := r.db.QueryRow(ctx, query, tenantID, companyNumber).Scan(
		&item.ID, &item.TenantID, &item.AccountID, &item.CompanyNumber, &item.CompanyName,
		&item.LastSnapshot, &item.LastCheckedAt, &item.LastChangedAt, &item.CheckEnabled,
		&item.NotifyOnChange, &item.CreatedAt, &item.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWatchlistItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get watchlist item by company: %w", err)
	}

	return item, nil
}

// isDuplicateError checks if the error is a unique constraint violation
func isDuplicateError(err error) bool {
	return err != nil && (err.Error() == "ERROR: duplicate key value violates unique constraint" ||
		// pgx uses error codes
		err.Error() == "23505")
}
