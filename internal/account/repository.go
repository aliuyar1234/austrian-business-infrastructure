package account

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAccountNotFound = errors.New("account not found")
	ErrAccountDeleted  = errors.New("account has been deleted")
)

// Account represents an external service account
type Account struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	Credentials     []byte     `json:"-"` // Never expose encrypted credentials directly
	CredentialsIV   []byte     `json:"-"`
	Status          string     `json:"status"`
	LastVerifiedAt  *time.Time `json:"last_verified_at,omitempty"`
	LastSyncAt      *time.Time `json:"last_sync_at,omitempty"`
	NextSyncAt      *time.Time `json:"next_sync_at,omitempty"`
	SyncInterval    string     `json:"sync_interval"`    // hourly, 4hourly, daily, weekly, disabled
	AutoSyncEnabled bool       `json:"auto_sync_enabled"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

// ConnectionTest represents a connection test result
type ConnectionTest struct {
	ID           uuid.UUID  `json:"id"`
	AccountID    uuid.UUID  `json:"account_id"`
	Success      bool       `json:"success"`
	DurationMs   *int       `json:"duration_ms,omitempty"`
	ErrorCode    *string    `json:"error_code,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// AccountWithStats extends Account with computed stats
type AccountWithStats struct {
	Account
	UnreadCount    int `json:"unread_count"`
	TotalDocuments int `json:"total_documents"`
}

// ListFilter defines account list filtering options
type ListFilter struct {
	TenantID        uuid.UUID
	Type            string
	Status          string
	Search          string
	TagIDs          []uuid.UUID
	AutoSyncEnabled bool
	DueForSync      bool // Filter accounts that are due for sync
	Limit           int
	Offset          int
	IncludeDeleted  bool
}

// Repository handles account database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new account repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new account
func (r *Repository) Create(ctx context.Context, account *Account) (*Account, error) {
	query := `
		INSERT INTO accounts (tenant_id, name, type, credentials, credentials_iv, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		account.TenantID,
		account.Name,
		account.Type,
		account.Credentials,
		account.CredentialsIV,
		"unverified",
	).Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)

	if err != nil {
		return nil, err
	}

	account.Status = "unverified"
	return account, nil
}

// GetByID retrieves an account by ID (with tenant verification)
func (r *Repository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*Account, error) {
	query := `
		SELECT id, tenant_id, name, type, credentials, credentials_iv, status,
		       last_verified_at, last_sync_at, next_sync_at, sync_interval, auto_sync_enabled,
		       error_message, created_at, updated_at, deleted_at
		FROM accounts
		WHERE id = $1 AND tenant_id = $2
	`

	account, err := r.scanAccount(r.db.QueryRow(ctx, query, id, tenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, err
	}

	if account.DeletedAt != nil {
		return nil, ErrAccountDeleted
	}

	return account, nil
}

// GetByIDOnly retrieves an account by ID without tenant verification (internal use)
func (r *Repository) GetByIDOnly(ctx context.Context, id uuid.UUID) (*Account, error) {
	query := `
		SELECT id, tenant_id, name, type, credentials, credentials_iv, status,
		       last_verified_at, last_sync_at, next_sync_at, sync_interval, auto_sync_enabled,
		       error_message, created_at, updated_at, deleted_at
		FROM accounts
		WHERE id = $1
	`

	account, err := r.scanAccount(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, err
	}

	if account.DeletedAt != nil {
		return nil, ErrAccountDeleted
	}

	return account, nil
}

// scanAccount scans a row into an Account struct
func (r *Repository) scanAccount(row pgx.Row) (*Account, error) {
	var account Account
	var syncInterval *string

	err := row.Scan(
		&account.ID,
		&account.TenantID,
		&account.Name,
		&account.Type,
		&account.Credentials,
		&account.CredentialsIV,
		&account.Status,
		&account.LastVerifiedAt,
		&account.LastSyncAt,
		&account.NextSyncAt,
		&syncInterval,
		&account.AutoSyncEnabled,
		&account.ErrorMessage,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.DeletedAt,
	)
	if err != nil {
		return nil, err
	}

	if syncInterval != nil {
		account.SyncInterval = *syncInterval
	} else {
		account.SyncInterval = "4hourly" // default
	}

	return &account, nil
}

// List retrieves accounts with filtering and pagination
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]*Account, int, error) {
	baseQuery := `
		FROM accounts a
		WHERE a.tenant_id = $1
	`
	args := []interface{}{filter.TenantID}
	argNum := 2

	if !filter.IncludeDeleted {
		baseQuery += " AND a.deleted_at IS NULL"
	}

	if filter.Type != "" {
		baseQuery += " AND a.type = $" + itoa(argNum)
		args = append(args, filter.Type)
		argNum++
	}

	if filter.Status != "" {
		baseQuery += " AND a.status = $" + itoa(argNum)
		args = append(args, filter.Status)
		argNum++
	}

	if filter.Search != "" {
		baseQuery += " AND a.name ILIKE $" + itoa(argNum)
		args = append(args, "%"+filter.Search+"%")
		argNum++
	}

	if len(filter.TagIDs) > 0 {
		baseQuery += " AND EXISTS (SELECT 1 FROM account_tags at WHERE at.account_id = a.id AND at.tag_id = ANY($" + itoa(argNum) + "))"
		args = append(args, filter.TagIDs)
		argNum++
	}

	if filter.AutoSyncEnabled {
		baseQuery += " AND a.auto_sync_enabled = TRUE"
	}

	if filter.DueForSync {
		baseQuery += " AND (a.next_sync_at IS NULL OR a.next_sync_at <= NOW())"
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch rows
	selectQuery := `
		SELECT a.id, a.tenant_id, a.name, a.type, a.credentials, a.credentials_iv, a.status,
		       a.last_verified_at, a.last_sync_at, a.next_sync_at, a.sync_interval, a.auto_sync_enabled,
		       a.error_message, a.created_at, a.updated_at, a.deleted_at
	` + baseQuery + " ORDER BY a.created_at DESC"

	if filter.Limit > 0 {
		selectQuery += " LIMIT $" + itoa(argNum)
		args = append(args, filter.Limit)
		argNum++
	}

	if filter.Offset > 0 {
		selectQuery += " OFFSET $" + itoa(argNum)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var accounts []*Account
	for rows.Next() {
		var account Account
		var syncInterval *string
		err := rows.Scan(
			&account.ID,
			&account.TenantID,
			&account.Name,
			&account.Type,
			&account.Credentials,
			&account.CredentialsIV,
			&account.Status,
			&account.LastVerifiedAt,
			&account.LastSyncAt,
			&account.NextSyncAt,
			&syncInterval,
			&account.AutoSyncEnabled,
			&account.ErrorMessage,
			&account.CreatedAt,
			&account.UpdatedAt,
			&account.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		if syncInterval != nil {
			account.SyncInterval = *syncInterval
		} else {
			account.SyncInterval = "4hourly"
		}
		accounts = append(accounts, &account)
	}

	return accounts, total, rows.Err()
}

// Update updates account fields (not credentials)
func (r *Repository) Update(ctx context.Context, account *Account) error {
	query := `
		UPDATE accounts
		SET name = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, account.Name, account.ID, account.TenantID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAccountNotFound
	}

	return nil
}

// UpdateCredentials updates account credentials
func (r *Repository) UpdateCredentials(ctx context.Context, id, tenantID uuid.UUID, credentials, iv []byte) error {
	query := `
		UPDATE accounts
		SET credentials = $1, credentials_iv = $2, status = 'unverified', updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, credentials, iv, id, tenantID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAccountNotFound
	}

	return nil
}

// UpdateStatus updates account status and optional error message
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	var query string
	var args []interface{}

	if status == "verified" {
		query = `
			UPDATE accounts
			SET status = $1, last_verified_at = NOW(), error_message = NULL, updated_at = NOW()
			WHERE id = $2 AND deleted_at IS NULL
		`
		args = []interface{}{status, id}
	} else {
		query = `
			UPDATE accounts
			SET status = $1, error_message = $2, updated_at = NOW()
			WHERE id = $3 AND deleted_at IS NULL
		`
		args = []interface{}{status, errorMsg, id}
	}

	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAccountNotFound
	}

	return nil
}

// SoftDelete marks an account as deleted
func (r *Repository) SoftDelete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `
		UPDATE accounts
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAccountNotFound
	}

	return nil
}

// HardDelete permanently deletes an account (for GDPR)
func (r *Repository) HardDelete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `DELETE FROM accounts WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAccountNotFound
	}

	return nil
}

// SaveConnectionTest saves a connection test result
func (r *Repository) SaveConnectionTest(ctx context.Context, test *ConnectionTest) (*ConnectionTest, error) {
	query := `
		INSERT INTO connection_tests (account_id, success, duration_ms, error_code, error_message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		test.AccountID,
		test.Success,
		test.DurationMs,
		test.ErrorCode,
		test.ErrorMessage,
	).Scan(&test.ID, &test.CreatedAt)

	if err != nil {
		return nil, err
	}

	return test, nil
}

// GetConnectionTests retrieves recent connection tests for an account
func (r *Repository) GetConnectionTests(ctx context.Context, accountID uuid.UUID, limit int) ([]*ConnectionTest, error) {
	query := `
		SELECT id, account_id, success, duration_ms, error_code, error_message, created_at
		FROM connection_tests
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tests []*ConnectionTest
	for rows.Next() {
		var test ConnectionTest
		err := rows.Scan(
			&test.ID,
			&test.AccountID,
			&test.Success,
			&test.DurationMs,
			&test.ErrorCode,
			&test.ErrorMessage,
			&test.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tests = append(tests, &test)
	}

	return tests, rows.Err()
}

// GetLastConnectionTest retrieves the most recent connection test
func (r *Repository) GetLastConnectionTest(ctx context.Context, accountID uuid.UUID) (*ConnectionTest, error) {
	tests, err := r.GetConnectionTests(ctx, accountID, 1)
	if err != nil {
		return nil, err
	}
	if len(tests) == 0 {
		return nil, nil
	}
	return tests[0], nil
}

// UpdateSyncSettings updates sync-related settings for an account
func (r *Repository) UpdateSyncSettings(ctx context.Context, id uuid.UUID, autoSyncEnabled *bool, syncInterval *string) error {
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argNum := 1

	if autoSyncEnabled != nil {
		setClauses = append(setClauses, "auto_sync_enabled = $"+itoa(argNum))
		args = append(args, *autoSyncEnabled)
		argNum++
	}

	if syncInterval != nil {
		setClauses = append(setClauses, "sync_interval = $"+itoa(argNum))
		args = append(args, *syncInterval)
		argNum++
	}

	if len(args) == 0 {
		return nil // Nothing to update
	}

	query := "UPDATE accounts SET "
	for i, clause := range setClauses {
		if i > 0 {
			query += ", "
		}
		query += clause
	}
	query += " WHERE id = $" + itoa(argNum) + " AND deleted_at IS NULL"
	args = append(args, id)

	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAccountNotFound
	}

	return nil
}

// UpdateLastSyncAt updates the last sync timestamp for an account
func (r *Repository) UpdateLastSyncAt(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE accounts
		SET last_sync_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAccountNotFound
	}

	return nil
}

// CheckDuplicateTID checks if TID already exists for tenant (FO accounts)
func (r *Repository) CheckDuplicateTID(ctx context.Context, tenantID uuid.UUID, tid string, excludeID *uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM accounts
			WHERE tenant_id = $1
			  AND type = 'finanzonline'
			  AND credentials->>'tid' = $2
			  AND deleted_at IS NULL
	`
	args := []interface{}{tenantID, tid}

	if excludeID != nil {
		query += " AND id != $3"
		args = append(args, *excludeID)
	}

	query += ")"

	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}

// Helper function to convert int to string
func itoa(i int) string {
	return strconv.Itoa(i)
}
