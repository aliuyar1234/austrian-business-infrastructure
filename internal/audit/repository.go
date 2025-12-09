package audit

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNoTenantContext is returned when a tenant-scoped operation is attempted
	// without a tenant context. Audit log reads require tenant context.
	ErrNoTenantContext = errors.New("no tenant context: audit log query requires tenant scope")
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           uuid.UUID              `json:"id"`
	TenantID     *uuid.UUID             `json:"tenant_id,omitempty"`
	UserID       *uuid.UUID             `json:"user_id,omitempty"`
	Action       string                 `json:"action"`
	ResourceType *string                `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID             `json:"resource_id,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	IPAddress    *string                `json:"ip_address,omitempty"`
	UserAgent    *string                `json:"user_agent,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// ListFilter contains filters for listing audit logs
type ListFilter struct {
	TenantID     *uuid.UUID
	UserID       *uuid.UUID
	Action       *string
	ResourceType *string
	ResourceID   *uuid.UUID
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int
	Offset       int
}

// Repository provides audit log data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new audit log repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create creates a new audit log entry
func (r *Repository) Create(ctx context.Context, log *AuditLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	query := `
		INSERT INTO audit_logs (id, tenant_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at
	`

	return r.pool.QueryRow(ctx, query,
		log.ID,
		log.TenantID,
		log.UserID,
		log.Action,
		log.ResourceType,
		log.ResourceID,
		log.Details,
		log.IPAddress,
		log.UserAgent,
	).Scan(&log.CreatedAt)
}

// List returns audit logs matching the filter.
// IMPORTANT: TenantID is REQUIRED for security - queries without tenant context will fail.
// This enforces tenant isolation at the repository level.
func (r *Repository) List(ctx context.Context, filter *ListFilter) ([]*AuditLog, error) {
	// Enforce tenant context - no tenant = no results (fail closed)
	if filter.TenantID == nil {
		return nil, ErrNoTenantContext
	}

	query := `
		SELECT id, tenant_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE tenant_id = $1
	`
	args := []interface{}{*filter.TenantID}
	argNum := 2

	// TenantID is already handled above, skip if provided again

	if filter.UserID != nil {
		query += " AND user_id = $" + itoa(argNum)
		args = append(args, *filter.UserID)
		argNum++
	}

	if filter.Action != nil {
		query += " AND action = $" + itoa(argNum)
		args = append(args, *filter.Action)
		argNum++
	}

	if filter.ResourceType != nil {
		query += " AND resource_type = $" + itoa(argNum)
		args = append(args, *filter.ResourceType)
		argNum++
	}

	if filter.ResourceID != nil {
		query += " AND resource_id = $" + itoa(argNum)
		args = append(args, *filter.ResourceID)
		argNum++
	}

	if filter.StartDate != nil {
		query += " AND created_at >= $" + itoa(argNum)
		args = append(args, *filter.StartDate)
		argNum++
	}

	if filter.EndDate != nil {
		query += " AND created_at <= $" + itoa(argNum)
		args = append(args, *filter.EndDate)
		argNum++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT $" + itoa(argNum)
		args = append(args, filter.Limit)
		argNum++
	}

	if filter.Offset > 0 {
		query += " OFFSET $" + itoa(argNum)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		if err := rows.Scan(
			&log.ID,
			&log.TenantID,
			&log.UserID,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&log.Details,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// Count returns the count of audit logs matching the filter.
// IMPORTANT: TenantID is REQUIRED for security - queries without tenant context will fail.
func (r *Repository) Count(ctx context.Context, filter *ListFilter) (int64, error) {
	// Enforce tenant context - no tenant = fail closed
	if filter.TenantID == nil {
		return 0, ErrNoTenantContext
	}

	query := `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1`
	args := []interface{}{*filter.TenantID}
	argNum := 2

	// TenantID is already handled above

	if filter.UserID != nil {
		query += " AND user_id = $" + itoa(argNum)
		args = append(args, *filter.UserID)
		argNum++
	}

	if filter.Action != nil {
		query += " AND action = $" + itoa(argNum)
		args = append(args, *filter.Action)
		argNum++
	}

	if filter.ResourceType != nil {
		query += " AND resource_type = $" + itoa(argNum)
		args = append(args, *filter.ResourceType)
		argNum++
	}

	if filter.StartDate != nil {
		query += " AND created_at >= $" + itoa(argNum)
		args = append(args, *filter.StartDate)
		argNum++
	}

	if filter.EndDate != nil {
		query += " AND created_at <= $" + itoa(argNum)
		args = append(args, *filter.EndDate)
	}

	var count int64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

// itoa converts int to string (simple implementation)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var s string
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// GetByID returns a single audit log entry by ID
// IMPORTANT: TenantID is REQUIRED for security - queries without tenant context will fail.
func (r *Repository) GetByID(ctx context.Context, tenantID uuid.UUID, logID uuid.UUID) (*AuditLog, error) {
	query := `
		SELECT id, tenant_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE tenant_id = $1 AND id = $2
	`

	log := &AuditLog{}
	err := r.pool.QueryRow(ctx, query, tenantID, logID).Scan(
		&log.ID,
		&log.TenantID,
		&log.UserID,
		&log.Action,
		&log.ResourceType,
		&log.ResourceID,
		&log.Details,
		&log.IPAddress,
		&log.UserAgent,
		&log.CreatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, errors.New("audit log not found")
		}
		return nil, err
	}

	return log, nil
}

// StatisticsResult represents audit log statistics
type StatisticsResult struct {
	Total         int64            `json:"total"`
	ByAction      map[string]int64 `json:"by_action"`
	ByResourceType map[string]int64 `json:"by_resource_type"`
	Last24h       int64            `json:"last_24h"`
	Last7d        int64            `json:"last_7d"`
	Last30d       int64            `json:"last_30d"`
}

// GetStatistics returns audit log statistics for a tenant
func (r *Repository) GetStatistics(ctx context.Context, tenantID uuid.UUID) (*StatisticsResult, error) {
	stats := &StatisticsResult{
		ByAction:       make(map[string]int64),
		ByResourceType: make(map[string]int64),
	}

	// Get total count
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1`, tenantID).Scan(&stats.Total)
	if err != nil {
		return nil, err
	}

	// Get counts by action
	actionRows, err := r.pool.Query(ctx, `
		SELECT action, COUNT(*)
		FROM audit_logs
		WHERE tenant_id = $1
		GROUP BY action
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer actionRows.Close()

	for actionRows.Next() {
		var action string
		var count int64
		if err := actionRows.Scan(&action, &count); err != nil {
			return nil, err
		}
		stats.ByAction[action] = count
	}

	// Get counts by resource type
	resourceRows, err := r.pool.Query(ctx, `
		SELECT resource_type, COUNT(*)
		FROM audit_logs
		WHERE tenant_id = $1 AND resource_type IS NOT NULL
		GROUP BY resource_type
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer resourceRows.Close()

	for resourceRows.Next() {
		var resourceType string
		var count int64
		if err := resourceRows.Scan(&resourceType, &count); err != nil {
			return nil, err
		}
		stats.ByResourceType[resourceType] = count
	}

	// Get time-based counts
	now := time.Now()
	err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND created_at >= $2`,
		tenantID, now.Add(-24*time.Hour)).Scan(&stats.Last24h)
	if err != nil {
		return nil, err
	}

	err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND created_at >= $2`,
		tenantID, now.Add(-7*24*time.Hour)).Scan(&stats.Last7d)
	if err != nil {
		return nil, err
	}

	err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND created_at >= $2`,
		tenantID, now.Add(-30*24*time.Hour)).Scan(&stats.Last30d)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// ListForArchive returns audit logs older than the given date
func (r *Repository) ListForArchive(ctx context.Context, tenantID uuid.UUID, olderThan time.Time, limit int) ([]*AuditLog, error) {
	query := `
		SELECT id, tenant_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE tenant_id = $1 AND created_at < $2
		ORDER BY created_at ASC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, tenantID, olderThan, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		if err := rows.Scan(
			&log.ID,
			&log.TenantID,
			&log.UserID,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&log.Details,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// DeleteOlderThan deletes audit logs older than the given date
func (r *Repository) DeleteOlderThan(ctx context.Context, tenantID uuid.UUID, olderThan time.Time, batchSize int) (int64, error) {
	// Delete in batches to avoid locking too many rows
	query := `
		DELETE FROM audit_logs
		WHERE id IN (
			SELECT id FROM audit_logs
			WHERE tenant_id = $1 AND created_at < $2
			ORDER BY created_at ASC
			LIMIT $3
		)
	`

	var totalDeleted int64
	for {
		result, err := r.pool.Exec(ctx, query, tenantID, olderThan, batchSize)
		if err != nil {
			return totalDeleted, err
		}

		deleted := result.RowsAffected()
		totalDeleted += deleted

		if deleted < int64(batchSize) {
			break
		}
	}

	return totalDeleted, nil
}

// CountOlderThan counts audit logs older than the given date
func (r *Repository) CountOlderThan(ctx context.Context, tenantID uuid.UUID, olderThan time.Time) (int64, error) {
	query := `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND created_at < $2`
	var count int64
	err := r.pool.QueryRow(ctx, query, tenantID, olderThan).Scan(&count)
	return count, err
}

// GetAllTenantIDs returns all unique tenant IDs in audit logs
func (r *Repository) GetAllTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	query := `SELECT DISTINCT tenant_id FROM audit_logs WHERE tenant_id IS NOT NULL`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenantIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		tenantIDs = append(tenantIDs, id)
	}

	return tenantIDs, rows.Err()
}
