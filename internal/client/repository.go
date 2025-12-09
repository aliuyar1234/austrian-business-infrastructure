package client

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrClientNotFound    = errors.New("client not found")
	ErrClientEmailExists = errors.New("email already exists for this tenant")
	ErrInvalidStatus     = errors.New("invalid client status")
)

// Status represents client account status
type Status string

const (
	StatusInvited  Status = "invited"
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// ValidStatuses contains all valid status values
var ValidStatuses = []Status{StatusInvited, StatusActive, StatusInactive}

// IsValidStatus checks if a status is valid
func IsValidStatus(status string) bool {
	for _, s := range ValidStatuses {
		if string(s) == status {
			return true
		}
	}
	return false
}

// Client represents a client (Mandant) in the portal
type Client struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"` // Set after activation

	// Profile
	Email       string  `json:"email"`
	Name        string  `json:"name"`
	CompanyName *string `json:"company_name,omitempty"`
	Phone       *string `json:"phone,omitempty"`

	// Status
	Status      Status     `json:"status"`
	InvitedAt   *time.Time `json:"invited_at,omitempty"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`

	// Settings
	NotificationEmail  bool   `json:"notification_email"`
	NotificationPortal bool   `json:"notification_portal"`
	Language           string `json:"language"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ClientAccountAccess represents a client's access to an account
type ClientAccountAccess struct {
	ID               uuid.UUID `json:"id"`
	ClientID         uuid.UUID `json:"client_id"`
	AccountID        uuid.UUID `json:"account_id"`
	CanUpload        bool      `json:"can_upload"`
	CanViewDocuments bool      `json:"can_view_documents"`
	CanApprove       bool      `json:"can_approve"`
	CreatedAt        time.Time `json:"created_at"`
}

// ClientWithAccounts includes account access information
type ClientWithAccounts struct {
	Client
	Accounts         []AccountAccess `json:"accounts"`
	PendingUploads   int             `json:"pending_uploads"`
	PendingApprovals int             `json:"pending_approvals"`
}

// AccountAccess represents simplified account access info
type AccountAccess struct {
	AccountID        uuid.UUID `json:"account_id"`
	AccountName      string    `json:"account_name"`
	CanUpload        bool      `json:"can_upload"`
	CanViewDocuments bool      `json:"can_view_documents"`
	CanApprove       bool      `json:"can_approve"`
}

// Repository provides client data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new client repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// clientColumns is the standard column list for client queries
const clientColumns = `id, tenant_id, user_id, email, name, company_name, phone,
	status, invited_at, activated_at, last_login_at,
	notification_email, notification_portal, language,
	created_at, updated_at`

// Create creates a new client
func (r *Repository) Create(ctx context.Context, client *Client) error {
	if client.ID == uuid.Nil {
		client.ID = uuid.New()
	}

	if !IsValidStatus(string(client.Status)) {
		return ErrInvalidStatus
	}

	query := `
		INSERT INTO clients (
			id, tenant_id, email, name, company_name, phone,
			status, invited_at, notification_email, notification_portal, language
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		client.ID,
		client.TenantID,
		client.Email,
		client.Name,
		client.CompanyName,
		client.Phone,
		client.Status,
		client.InvitedAt,
		client.NotificationEmail,
		client.NotificationPortal,
		client.Language,
	).Scan(&client.CreatedAt, &client.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrClientEmailExists
		}
		return err
	}

	return nil
}

// GetByID retrieves a client by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Client, error) {
	query := `SELECT ` + clientColumns + ` FROM clients WHERE id = $1`
	return r.scanClient(r.pool.QueryRow(ctx, query, id))
}

// GetByEmail retrieves a client by email within a tenant
func (r *Repository) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*Client, error) {
	query := `SELECT ` + clientColumns + ` FROM clients WHERE tenant_id = $1 AND email = $2`
	return r.scanClient(r.pool.QueryRow(ctx, query, tenantID, email))
}

// GetByUserID retrieves a client by their associated user ID
func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Client, error) {
	query := `SELECT ` + clientColumns + ` FROM clients WHERE user_id = $1`
	return r.scanClient(r.pool.QueryRow(ctx, query, userID))
}

// ListByTenant returns all clients for a tenant with optional filters
func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID, status *Status, limit, offset int) ([]*Client, int, error) {
	// Build query with optional status filter
	countQuery := `SELECT COUNT(*) FROM clients WHERE tenant_id = $1`
	listQuery := `SELECT ` + clientColumns + ` FROM clients WHERE tenant_id = $1`

	args := []interface{}{tenantID}

	if status != nil {
		countQuery += ` AND status = $2`
		listQuery += ` AND status = $2`
		args = append(args, *status)
	}

	listQuery += ` ORDER BY created_at DESC`

	// Get total count
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if limit > 0 {
		listQuery += ` LIMIT $` + itoa(len(args)+1)
		args = append(args, limit)
	}
	if offset > 0 {
		listQuery += ` OFFSET $` + itoa(len(args)+1)
		args = append(args, offset)
	}

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var clients []*Client
	for rows.Next() {
		client, err := r.scanClientFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		clients = append(clients, client)
	}

	return clients, total, rows.Err()
}

// Update updates a client
func (r *Repository) Update(ctx context.Context, client *Client) error {
	if !IsValidStatus(string(client.Status)) {
		return ErrInvalidStatus
	}

	query := `
		UPDATE clients
		SET email = $2, name = $3, company_name = $4, phone = $5,
			status = $6, notification_email = $7, notification_portal = $8,
			language = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		client.ID,
		client.Email,
		client.Name,
		client.CompanyName,
		client.Phone,
		client.Status,
		client.NotificationEmail,
		client.NotificationPortal,
		client.Language,
	).Scan(&client.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrClientNotFound
		}
		if isDuplicateKeyError(err) {
			return ErrClientEmailExists
		}
		return err
	}

	return nil
}

// Activate activates a client and links them to a user
func (r *Repository) Activate(ctx context.Context, clientID, userID uuid.UUID) error {
	query := `
		UPDATE clients
		SET user_id = $2, status = 'active', activated_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'invited'
		RETURNING id
	`

	var id uuid.UUID
	err := r.pool.QueryRow(ctx, query, clientID, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrClientNotFound
		}
		return err
	}

	return nil
}

// Deactivate deactivates a client
func (r *Repository) Deactivate(ctx context.Context, clientID uuid.UUID) error {
	query := `
		UPDATE clients
		SET status = 'inactive', updated_at = NOW()
		WHERE id = $1 AND status = 'active'
	`

	result, err := r.pool.Exec(ctx, query, clientID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrClientNotFound
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *Repository) UpdateLastLogin(ctx context.Context, clientID uuid.UUID) error {
	query := `
		UPDATE clients
		SET last_login_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, clientID)
	return err
}

// ============== Account Access Methods ==============

// AddAccountAccess grants a client access to an account
func (r *Repository) AddAccountAccess(ctx context.Context, access *ClientAccountAccess) error {
	if access.ID == uuid.Nil {
		access.ID = uuid.New()
	}

	query := `
		INSERT INTO client_account_access (id, client_id, account_id, can_upload, can_view_documents, can_approve)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (client_id, account_id) DO UPDATE SET
			can_upload = EXCLUDED.can_upload,
			can_view_documents = EXCLUDED.can_view_documents,
			can_approve = EXCLUDED.can_approve
		RETURNING created_at
	`

	err := r.pool.QueryRow(ctx, query,
		access.ID,
		access.ClientID,
		access.AccountID,
		access.CanUpload,
		access.CanViewDocuments,
		access.CanApprove,
	).Scan(&access.CreatedAt)

	return err
}

// RemoveAccountAccess removes a client's access to an account
func (r *Repository) RemoveAccountAccess(ctx context.Context, clientID, accountID uuid.UUID) error {
	query := `DELETE FROM client_account_access WHERE client_id = $1 AND account_id = $2`

	result, err := r.pool.Exec(ctx, query, clientID, accountID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("access not found")
	}

	return nil
}

// GetAccountAccess returns all accounts a client has access to
func (r *Repository) GetAccountAccess(ctx context.Context, clientID uuid.UUID) ([]AccountAccess, error) {
	query := `
		SELECT caa.account_id, a.name, caa.can_upload, caa.can_view_documents, caa.can_approve
		FROM client_account_access caa
		JOIN accounts a ON caa.account_id = a.id
		WHERE caa.client_id = $1
		ORDER BY a.name
	`

	rows, err := r.pool.Query(ctx, query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accesses []AccountAccess
	for rows.Next() {
		var a AccountAccess
		if err := rows.Scan(&a.AccountID, &a.AccountName, &a.CanUpload, &a.CanViewDocuments, &a.CanApprove); err != nil {
			return nil, err
		}
		accesses = append(accesses, a)
	}

	return accesses, rows.Err()
}

// HasAccountAccess checks if a client has access to an account
func (r *Repository) HasAccountAccess(ctx context.Context, clientID, accountID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM client_account_access WHERE client_id = $1 AND account_id = $2)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, clientID, accountID).Scan(&exists)
	return exists, err
}

// ============== Helper Methods ==============

func (r *Repository) scanClient(row pgx.Row) (*Client, error) {
	client := &Client{}
	err := row.Scan(
		&client.ID,
		&client.TenantID,
		&client.UserID,
		&client.Email,
		&client.Name,
		&client.CompanyName,
		&client.Phone,
		&client.Status,
		&client.InvitedAt,
		&client.ActivatedAt,
		&client.LastLoginAt,
		&client.NotificationEmail,
		&client.NotificationPortal,
		&client.Language,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClientNotFound
		}
		return nil, err
	}

	return client, nil
}

func (r *Repository) scanClientFromRows(rows pgx.Rows) (*Client, error) {
	client := &Client{}
	err := rows.Scan(
		&client.ID,
		&client.TenantID,
		&client.UserID,
		&client.Email,
		&client.Name,
		&client.CompanyName,
		&client.Phone,
		&client.Status,
		&client.InvitedAt,
		&client.ActivatedAt,
		&client.LastLoginAt,
		&client.NotificationEmail,
		&client.NotificationPortal,
		&client.Language,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return client, nil
}

// isDuplicateKeyError checks if error is a unique constraint violation
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsSubstring(errStr, "23505") || containsSubstring(errStr, "unique constraint")
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func itoa(i int) string {
	if i < 0 {
		return "-" + uitoa(uint(-i))
	}
	return uitoa(uint(i))
}

func uitoa(val uint) string {
	if val == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf) - 1
	for val != 0 {
		buf[i] = byte('0' + val%10)
		val /= 10
		i--
	}
	return string(buf[i+1:])
}
