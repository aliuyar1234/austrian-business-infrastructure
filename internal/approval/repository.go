package approval

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrApprovalNotFound = errors.New("approval request not found")
	ErrInvalidStatus    = errors.New("invalid approval status")
	ErrAlreadyResponded = errors.New("approval already responded")
)

// Status represents the status of an approval request
type Status string

const (
	StatusPending           Status = "pending"
	StatusApproved          Status = "approved"
	StatusRejected          Status = "rejected"
	StatusRevisionRequested Status = "revision_requested"
)

// ValidStatuses contains all valid status values
var ValidStatuses = []Status{StatusPending, StatusApproved, StatusRejected, StatusRevisionRequested}

// IsValidStatus checks if a status is valid
func IsValidStatus(status string) bool {
	for _, s := range ValidStatuses {
		if string(s) == status {
			return true
		}
	}
	return false
}

// ApprovalRequest represents an approval request
type ApprovalRequest struct {
	ID              uuid.UUID  `json:"id"`
	DocumentID      uuid.UUID  `json:"document_id"`
	ClientID        uuid.UUID  `json:"client_id"`

	// Request
	RequestedBy     uuid.UUID  `json:"requested_by"`
	RequestedAt     time.Time  `json:"requested_at"`
	Message         *string    `json:"message,omitempty"`

	// Response
	Status          Status     `json:"status"`
	RespondedAt     *time.Time `json:"responded_at,omitempty"`
	ResponseComment *string    `json:"response_comment,omitempty"`

	// Tracking
	ReminderSentAt  *time.Time `json:"reminder_sent_at,omitempty"`
	ReminderCount   int        `json:"reminder_count"`

	CreatedAt       time.Time  `json:"created_at"`

	// Joined fields
	DocumentTitle string `json:"document_title,omitempty"`
	ClientName    string `json:"client_name,omitempty"`
	ClientEmail   string `json:"client_email,omitempty"`
}

// Repository provides approval data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new approval repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// approvalColumns is the standard column list
const approvalColumns = `id, document_id, client_id, requested_by, requested_at,
	message, status, responded_at, response_comment, reminder_sent_at, reminder_count, created_at`

// Create creates a new approval request
func (r *Repository) Create(ctx context.Context, approval *ApprovalRequest) error {
	if approval.ID == uuid.Nil {
		approval.ID = uuid.New()
	}

	query := `
		INSERT INTO approval_requests (
			id, document_id, client_id, requested_by, message
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING requested_at, created_at
	`

	err := r.pool.QueryRow(ctx, query,
		approval.ID,
		approval.DocumentID,
		approval.ClientID,
		approval.RequestedBy,
		approval.Message,
	).Scan(&approval.RequestedAt, &approval.CreatedAt)

	if err != nil {
		return err
	}

	approval.Status = StatusPending
	return nil
}

// GetByID retrieves an approval by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error) {
	query := `SELECT ` + approvalColumns + ` FROM approval_requests WHERE id = $1`
	return r.scanApproval(r.pool.QueryRow(ctx, query, id))
}

// GetByIDWithDetails retrieves an approval with document and client details
func (r *Repository) GetByIDWithDetails(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error) {
	query := `
		SELECT ar.id, ar.document_id, ar.client_id, ar.requested_by, ar.requested_at,
			ar.message, ar.status, ar.responded_at, ar.response_comment,
			ar.reminder_sent_at, ar.reminder_count, ar.created_at,
			d.title as document_title, c.name as client_name, c.email as client_email
		FROM approval_requests ar
		JOIN documents d ON ar.document_id = d.id
		JOIN clients c ON ar.client_id = c.id
		WHERE ar.id = $1
	`

	approval := &ApprovalRequest{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&approval.ID, &approval.DocumentID, &approval.ClientID, &approval.RequestedBy,
		&approval.RequestedAt, &approval.Message, &approval.Status, &approval.RespondedAt,
		&approval.ResponseComment, &approval.ReminderSentAt, &approval.ReminderCount,
		&approval.CreatedAt, &approval.DocumentTitle, &approval.ClientName, &approval.ClientEmail,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrApprovalNotFound
		}
		return nil, err
	}

	return approval, nil
}

// ListByClient returns all approvals for a client
func (r *Repository) ListByClient(ctx context.Context, clientID uuid.UUID, status *Status, limit, offset int) ([]*ApprovalRequest, int, error) {
	countQuery := `SELECT COUNT(*) FROM approval_requests WHERE client_id = $1`
	listQuery := `
		SELECT ar.id, ar.document_id, ar.client_id, ar.requested_by, ar.requested_at,
			ar.message, ar.status, ar.responded_at, ar.response_comment,
			ar.reminder_sent_at, ar.reminder_count, ar.created_at,
			d.title as document_title
		FROM approval_requests ar
		JOIN documents d ON ar.document_id = d.id
		WHERE ar.client_id = $1
	`

	args := []interface{}{clientID}
	argNum := 2

	if status != nil {
		countQuery += ` AND status = $2`
		listQuery += ` AND ar.status = $2`
		args = append(args, *status)
		argNum++
	}

	listQuery += ` ORDER BY ar.requested_at DESC`

	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		listQuery += ` LIMIT $` + itoa(argNum)
		args = append(args, limit)
		argNum++
	}
	if offset > 0 {
		listQuery += ` OFFSET $` + itoa(argNum)
		args = append(args, offset)
	}

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var approvals []*ApprovalRequest
	for rows.Next() {
		approval := &ApprovalRequest{}
		err := rows.Scan(
			&approval.ID, &approval.DocumentID, &approval.ClientID, &approval.RequestedBy,
			&approval.RequestedAt, &approval.Message, &approval.Status, &approval.RespondedAt,
			&approval.ResponseComment, &approval.ReminderSentAt, &approval.ReminderCount,
			&approval.CreatedAt, &approval.DocumentTitle,
		)
		if err != nil {
			return nil, 0, err
		}
		approvals = append(approvals, approval)
	}

	return approvals, total, rows.Err()
}

// ListPendingByTenant returns all pending approvals for a tenant
func (r *Repository) ListPendingByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*ApprovalRequest, int, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM approval_requests ar
		JOIN clients c ON ar.client_id = c.id
		WHERE c.tenant_id = $1 AND ar.status = 'pending'
	`

	listQuery := `
		SELECT ar.id, ar.document_id, ar.client_id, ar.requested_by, ar.requested_at,
			ar.message, ar.status, ar.responded_at, ar.response_comment,
			ar.reminder_sent_at, ar.reminder_count, ar.created_at,
			d.title as document_title, c.name as client_name, c.email as client_email
		FROM approval_requests ar
		JOIN documents d ON ar.document_id = d.id
		JOIN clients c ON ar.client_id = c.id
		WHERE c.tenant_id = $1 AND ar.status = 'pending'
		ORDER BY ar.requested_at ASC
	`

	var total int
	err := r.pool.QueryRow(ctx, countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args := []interface{}{tenantID}
	if limit > 0 {
		listQuery += ` LIMIT $2`
		args = append(args, limit)
	}
	if offset > 0 {
		listQuery += ` OFFSET $3`
		args = append(args, offset)
	}

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var approvals []*ApprovalRequest
	for rows.Next() {
		approval := &ApprovalRequest{}
		err := rows.Scan(
			&approval.ID, &approval.DocumentID, &approval.ClientID, &approval.RequestedBy,
			&approval.RequestedAt, &approval.Message, &approval.Status, &approval.RespondedAt,
			&approval.ResponseComment, &approval.ReminderSentAt, &approval.ReminderCount,
			&approval.CreatedAt, &approval.DocumentTitle, &approval.ClientName, &approval.ClientEmail,
		)
		if err != nil {
			return nil, 0, err
		}
		approvals = append(approvals, approval)
	}

	return approvals, total, rows.Err()
}

// Approve approves an approval request
func (r *Repository) Approve(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE approval_requests
		SET status = 'approved', responded_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrApprovalNotFound
	}

	return nil
}

// Reject rejects an approval request
func (r *Repository) Reject(ctx context.Context, id uuid.UUID, comment string) error {
	query := `
		UPDATE approval_requests
		SET status = 'rejected', responded_at = NOW(), response_comment = $2
		WHERE id = $1 AND status = 'pending'
	`

	result, err := r.pool.Exec(ctx, query, id, comment)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrApprovalNotFound
	}

	return nil
}

// RequestRevision requests a revision on an approval request
func (r *Repository) RequestRevision(ctx context.Context, id uuid.UUID, comment string) error {
	query := `
		UPDATE approval_requests
		SET status = 'revision_requested', responded_at = NOW(), response_comment = $2
		WHERE id = $1 AND status = 'pending'
	`

	result, err := r.pool.Exec(ctx, query, id, comment)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrApprovalNotFound
	}

	return nil
}

// UpdateReminderSent updates the reminder tracking
func (r *Repository) UpdateReminderSent(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE approval_requests
		SET reminder_sent_at = NOW(), reminder_count = reminder_count + 1
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// CountPendingByClient counts pending approvals for a client
func (r *Repository) CountPendingByClient(ctx context.Context, clientID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM approval_requests WHERE client_id = $1 AND status = 'pending'`

	var count int
	err := r.pool.QueryRow(ctx, query, clientID).Scan(&count)
	return count, err
}

func (r *Repository) scanApproval(row pgx.Row) (*ApprovalRequest, error) {
	approval := &ApprovalRequest{}
	err := row.Scan(
		&approval.ID, &approval.DocumentID, &approval.ClientID, &approval.RequestedBy,
		&approval.RequestedAt, &approval.Message, &approval.Status, &approval.RespondedAt,
		&approval.ResponseComment, &approval.ReminderSentAt, &approval.ReminderCount,
		&approval.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrApprovalNotFound
		}
		return nil, err
	}

	return approval, nil
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
