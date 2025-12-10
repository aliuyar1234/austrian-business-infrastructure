package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"austrian-business-infrastructure/internal/job"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SoftDeleteCleanup is the job type for purging soft-deleted records
const SoftDeleteCleanupJobType = "soft_delete_cleanup"

// SoftDeleteCleanupHandler handles purging of soft-deleted records
type SoftDeleteCleanupHandler struct {
	db              *pgxpool.Pool
	logger          *slog.Logger
	gracePeriodDays int
	batchSize       int
}

// SoftDeleteCleanupConfig holds configuration for the soft delete cleanup handler
type SoftDeleteCleanupConfig struct {
	Logger          *slog.Logger
	GracePeriodDays int // How long to keep soft-deleted records (default: 30)
	BatchSize       int // How many records to process per batch (default: 100)
}

// NewSoftDeleteCleanupHandler creates a new soft delete cleanup handler
func NewSoftDeleteCleanupHandler(
	db *pgxpool.Pool,
	cfg *SoftDeleteCleanupConfig,
) *SoftDeleteCleanupHandler {
	logger := slog.Default()
	gracePeriodDays := 30
	batchSize := 100

	if cfg != nil {
		if cfg.Logger != nil {
			logger = cfg.Logger
		}
		if cfg.GracePeriodDays > 0 {
			gracePeriodDays = cfg.GracePeriodDays
		}
		if cfg.BatchSize > 0 {
			batchSize = cfg.BatchSize
		}
	}

	return &SoftDeleteCleanupHandler{
		db:              db,
		logger:          logger,
		gracePeriodDays: gracePeriodDays,
		batchSize:       batchSize,
	}
}

// SoftDeleteCleanupPayload defines the job payload
type SoftDeleteCleanupPayload struct {
	GracePeriodDays *int `json:"grace_period_days,omitempty"` // Override default grace period
}

// SoftDeleteCleanupResult contains the results of a cleanup operation
type SoftDeleteCleanupResult struct {
	AccountsDeleted int64 `json:"accounts_deleted"`
	UsersDeleted    int64 `json:"users_deleted"`
	TenantsDeleted  int64 `json:"tenants_deleted"`
}

// Handle executes the soft delete cleanup job
func (h *SoftDeleteCleanupHandler) Handle(ctx context.Context, j *job.Job) (json.RawMessage, error) {
	h.logger.Info("starting soft delete cleanup job", "job_id", j.ID)

	// Parse payload
	var payload SoftDeleteCleanupPayload
	if len(j.Payload) > 0 {
		if err := json.Unmarshal(j.Payload, &payload); err != nil {
			return nil, fmt.Errorf("parse payload: %w", err)
		}
	}

	// Determine grace period
	gracePeriodDays := h.gracePeriodDays
	if payload.GracePeriodDays != nil {
		gracePeriodDays = *payload.GracePeriodDays
	}

	cutoff := time.Now().AddDate(0, 0, -gracePeriodDays)

	var result SoftDeleteCleanupResult

	// Delete old soft-deleted accounts
	accountsDeleted, err := h.cleanupAccounts(ctx, cutoff)
	if err != nil {
		h.logger.Error("failed to cleanup accounts", "error", err)
	} else {
		result.AccountsDeleted = accountsDeleted
	}

	// Delete old soft-deleted users
	usersDeleted, err := h.cleanupUsers(ctx, cutoff)
	if err != nil {
		h.logger.Error("failed to cleanup users", "error", err)
	} else {
		result.UsersDeleted = usersDeleted
	}

	// Delete old soft-deleted tenants (be very careful with this)
	tenantsDeleted, err := h.cleanupTenants(ctx, cutoff)
	if err != nil {
		h.logger.Error("failed to cleanup tenants", "error", err)
	} else {
		result.TenantsDeleted = tenantsDeleted
	}

	h.logger.Info("soft delete cleanup completed",
		"accounts_deleted", result.AccountsDeleted,
		"users_deleted", result.UsersDeleted,
		"tenants_deleted", result.TenantsDeleted)

	return json.Marshal(result)
}

// cleanupAccounts permanently deletes soft-deleted accounts past the grace period
func (h *SoftDeleteCleanupHandler) cleanupAccounts(ctx context.Context, cutoff time.Time) (int64, error) {
	// First, get the IDs to delete
	query := `
		SELECT id FROM accounts
		WHERE deleted_at IS NOT NULL AND deleted_at < $1
		LIMIT $2
	`

	var totalDeleted int64

	for {
		rows, err := h.db.Query(ctx, query, cutoff, h.batchSize)
		if err != nil {
			return totalDeleted, fmt.Errorf("query accounts: %w", err)
		}

		var ids []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return totalDeleted, fmt.Errorf("scan account id: %w", err)
			}
			ids = append(ids, id)
		}
		rows.Close()

		if len(ids) == 0 {
			break
		}

		// Delete in batch
		deleteQuery := `DELETE FROM accounts WHERE id = ANY($1)`
		result, err := h.db.Exec(ctx, deleteQuery, ids)
		if err != nil {
			return totalDeleted, fmt.Errorf("delete accounts: %w", err)
		}

		deleted := result.RowsAffected()
		totalDeleted += deleted

		h.logger.Debug("deleted accounts batch",
			"count", deleted,
			"total", totalDeleted)

		if len(ids) < h.batchSize {
			break
		}
	}

	return totalDeleted, nil
}

// cleanupUsers permanently deletes soft-deleted users past the grace period
func (h *SoftDeleteCleanupHandler) cleanupUsers(ctx context.Context, cutoff time.Time) (int64, error) {
	// Check if users table has deleted_at column
	var exists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'users' AND column_name = 'deleted_at'
		)
	`
	if err := h.db.QueryRow(ctx, checkQuery).Scan(&exists); err != nil {
		return 0, fmt.Errorf("check column exists: %w", err)
	}

	if !exists {
		// Users table doesn't support soft delete
		return 0, nil
	}

	query := `
		SELECT id FROM users
		WHERE deleted_at IS NOT NULL AND deleted_at < $1
		LIMIT $2
	`

	var totalDeleted int64

	for {
		rows, err := h.db.Query(ctx, query, cutoff, h.batchSize)
		if err != nil {
			return totalDeleted, fmt.Errorf("query users: %w", err)
		}

		var ids []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return totalDeleted, fmt.Errorf("scan user id: %w", err)
			}
			ids = append(ids, id)
		}
		rows.Close()

		if len(ids) == 0 {
			break
		}

		deleteQuery := `DELETE FROM users WHERE id = ANY($1)`
		result, err := h.db.Exec(ctx, deleteQuery, ids)
		if err != nil {
			return totalDeleted, fmt.Errorf("delete users: %w", err)
		}

		deleted := result.RowsAffected()
		totalDeleted += deleted

		if len(ids) < h.batchSize {
			break
		}
	}

	return totalDeleted, nil
}

// cleanupTenants permanently deletes soft-deleted tenants past the grace period
func (h *SoftDeleteCleanupHandler) cleanupTenants(ctx context.Context, cutoff time.Time) (int64, error) {
	// Check if tenants table has deleted_at column
	var exists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'tenants' AND column_name = 'deleted_at'
		)
	`
	if err := h.db.QueryRow(ctx, checkQuery).Scan(&exists); err != nil {
		return 0, fmt.Errorf("check column exists: %w", err)
	}

	if !exists {
		// Tenants table doesn't support soft delete
		return 0, nil
	}

	// For tenants, we need to be extra careful
	// Only delete tenants that have no associated users/accounts
	query := `
		SELECT t.id FROM tenants t
		WHERE t.deleted_at IS NOT NULL AND t.deleted_at < $1
		AND NOT EXISTS (SELECT 1 FROM users u WHERE u.tenant_id = t.id)
		AND NOT EXISTS (SELECT 1 FROM accounts a WHERE a.tenant_id = t.id)
		LIMIT $2
	`

	var totalDeleted int64

	for {
		rows, err := h.db.Query(ctx, query, cutoff, h.batchSize)
		if err != nil {
			return totalDeleted, fmt.Errorf("query tenants: %w", err)
		}

		var ids []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return totalDeleted, fmt.Errorf("scan tenant id: %w", err)
			}
			ids = append(ids, id)
		}
		rows.Close()

		if len(ids) == 0 {
			break
		}

		deleteQuery := `DELETE FROM tenants WHERE id = ANY($1)`
		result, err := h.db.Exec(ctx, deleteQuery, ids)
		if err != nil {
			return totalDeleted, fmt.Errorf("delete tenants: %w", err)
		}

		deleted := result.RowsAffected()
		totalDeleted += deleted

		if len(ids) < h.batchSize {
			break
		}
	}

	return totalDeleted, nil
}

// Register registers the soft delete cleanup handler with a job registry
func (h *SoftDeleteCleanupHandler) Register(registry *job.Registry) {
	registry.MustRegister(SoftDeleteCleanupJobType, h)
}
