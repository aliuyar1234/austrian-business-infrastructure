package task

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

// Status represents task status
type Status string

const (
	StatusOpen      Status = "open"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
)

// Priority represents task priority
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// ClientTask represents a task assigned to a client
type ClientTask struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	ClientID    uuid.UUID  `json:"client_id"`
	CreatedBy   uuid.UUID  `json:"created_by"`

	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Status      Status     `json:"status"`
	Priority    Priority   `json:"priority"`

	DueDate     *time.Time `json:"due_date,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Linked resources
	DocumentID  *uuid.UUID `json:"document_id,omitempty"`
	UploadID    *uuid.UUID `json:"upload_id,omitempty"`
	ApprovalID  *uuid.UUID `json:"approval_id,omitempty"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Joined fields
	ClientName  string     `json:"client_name,omitempty"`
	CreatorName string     `json:"creator_name,omitempty"`
}

// Repository provides task data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new task repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create creates a new task
func (r *Repository) Create(ctx context.Context, task *ClientTask) error {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}

	query := `
		INSERT INTO client_tasks (
			id, tenant_id, client_id, created_by, title, description,
			status, priority, due_date, document_id, upload_id, approval_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		task.ID,
		task.TenantID,
		task.ClientID,
		task.CreatedBy,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		task.DueDate,
		task.DocumentID,
		task.UploadID,
		task.ApprovalID,
	).Scan(&task.CreatedAt, &task.UpdatedAt)

	return err
}

// GetByID retrieves a task by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*ClientTask, error) {
	query := `
		SELECT t.id, t.tenant_id, t.client_id, t.created_by, t.title, t.description,
			t.status, t.priority, t.due_date, t.completed_at,
			t.document_id, t.upload_id, t.approval_id, t.created_at, t.updated_at,
			c.name as client_name, u.name as creator_name
		FROM client_tasks t
		JOIN clients c ON t.client_id = c.id
		LEFT JOIN users u ON t.created_by = u.id
		WHERE t.id = $1
	`

	task := &ClientTask{}
	var creatorName *string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&task.ID, &task.TenantID, &task.ClientID, &task.CreatedBy,
		&task.Title, &task.Description, &task.Status, &task.Priority,
		&task.DueDate, &task.CompletedAt, &task.DocumentID, &task.UploadID,
		&task.ApprovalID, &task.CreatedAt, &task.UpdatedAt,
		&task.ClientName, &creatorName,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	if creatorName != nil {
		task.CreatorName = *creatorName
	}

	return task, nil
}

// ListByTenant returns tasks for a tenant
func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID, status *Status, limit, offset int) ([]*ClientTask, int, error) {
	countQuery := `SELECT COUNT(*) FROM client_tasks WHERE tenant_id = $1`
	listQuery := `
		SELECT t.id, t.tenant_id, t.client_id, t.created_by, t.title, t.description,
			t.status, t.priority, t.due_date, t.completed_at,
			t.document_id, t.upload_id, t.approval_id, t.created_at, t.updated_at,
			c.name as client_name
		FROM client_tasks t
		JOIN clients c ON t.client_id = c.id
		WHERE t.tenant_id = $1
	`

	args := []interface{}{tenantID}
	argNum := 2

	if status != nil {
		countQuery += ` AND status = $2`
		listQuery += ` AND t.status = $2`
		args = append(args, *status)
		argNum++
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery += ` ORDER BY COALESCE(t.due_date, t.created_at) ASC`

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

	var tasks []*ClientTask
	for rows.Next() {
		task := &ClientTask{}
		err := rows.Scan(
			&task.ID, &task.TenantID, &task.ClientID, &task.CreatedBy,
			&task.Title, &task.Description, &task.Status, &task.Priority,
			&task.DueDate, &task.CompletedAt, &task.DocumentID, &task.UploadID,
			&task.ApprovalID, &task.CreatedAt, &task.UpdatedAt, &task.ClientName,
		)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}

	return tasks, total, rows.Err()
}

// ListByClient returns tasks for a client
func (r *Repository) ListByClient(ctx context.Context, clientID uuid.UUID, status *Status, limit, offset int) ([]*ClientTask, int, error) {
	countQuery := `SELECT COUNT(*) FROM client_tasks WHERE client_id = $1`
	listQuery := `
		SELECT t.id, t.tenant_id, t.client_id, t.created_by, t.title, t.description,
			t.status, t.priority, t.due_date, t.completed_at,
			t.document_id, t.upload_id, t.approval_id, t.created_at, t.updated_at
		FROM client_tasks t
		WHERE t.client_id = $1
	`

	args := []interface{}{clientID}
	argNum := 2

	if status != nil {
		countQuery += ` AND status = $2`
		listQuery += ` AND t.status = $2`
		args = append(args, *status)
		argNum++
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery += ` ORDER BY COALESCE(t.due_date, t.created_at) ASC`

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

	var tasks []*ClientTask
	for rows.Next() {
		task := &ClientTask{}
		err := rows.Scan(
			&task.ID, &task.TenantID, &task.ClientID, &task.CreatedBy,
			&task.Title, &task.Description, &task.Status, &task.Priority,
			&task.DueDate, &task.CompletedAt, &task.DocumentID, &task.UploadID,
			&task.ApprovalID, &task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}

	return tasks, total, rows.Err()
}

// Update updates a task
func (r *Repository) Update(ctx context.Context, task *ClientTask) error {
	query := `
		UPDATE client_tasks
		SET title = $2, description = $3, status = $4, priority = $5,
			due_date = $6, document_id = $7, upload_id = $8, approval_id = $9,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		task.ID,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		task.DueDate,
		task.DocumentID,
		task.UploadID,
		task.ApprovalID,
	).Scan(&task.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTaskNotFound
		}
		return err
	}

	return nil
}

// Complete marks a task as completed
func (r *Repository) Complete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE client_tasks
		SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'open'
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	return nil
}

// Cancel cancels a task
func (r *Repository) Cancel(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE client_tasks
		SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND status = 'open'
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	return nil
}

// Delete deletes a task
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM client_tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	return nil
}

// CountOpenByClient counts open tasks for a client
func (r *Repository) CountOpenByClient(ctx context.Context, clientID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM client_tasks WHERE client_id = $1 AND status = 'open'`

	var count int
	err := r.pool.QueryRow(ctx, query, clientID).Scan(&count)
	return count, err
}

// ListOverdue returns overdue tasks
func (r *Repository) ListOverdue(ctx context.Context, tenantID uuid.UUID) ([]*ClientTask, error) {
	query := `
		SELECT t.id, t.tenant_id, t.client_id, t.created_by, t.title, t.description,
			t.status, t.priority, t.due_date, t.completed_at,
			t.document_id, t.upload_id, t.approval_id, t.created_at, t.updated_at,
			c.name as client_name
		FROM client_tasks t
		JOIN clients c ON t.client_id = c.id
		WHERE t.tenant_id = $1 AND t.status = 'open' AND t.due_date < NOW()
		ORDER BY t.due_date ASC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*ClientTask
	for rows.Next() {
		task := &ClientTask{}
		err := rows.Scan(
			&task.ID, &task.TenantID, &task.ClientID, &task.CreatedBy,
			&task.Title, &task.Description, &task.Status, &task.Priority,
			&task.DueDate, &task.CompletedAt, &task.DocumentID, &task.UploadID,
			&task.ApprovalID, &task.CreatedAt, &task.UpdatedAt, &task.ClientName,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
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
