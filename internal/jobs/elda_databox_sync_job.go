package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ELDADataboxSyncJob synchronizes ELDA databox documents for all accounts
type ELDADataboxSyncJob struct {
	db       *pgxpool.Pool
	logger   *slog.Logger
	syncer   ELDADataboxSyncer
	notifier DataboxNotifier
}

// ELDADataboxSyncer interface for syncing ELDA databox
type ELDADataboxSyncer interface {
	Sync(ctx context.Context, accountID uuid.UUID) (*ELDASyncResult, error)
}

// ELDASyncResult contains ELDA-specific sync result
type ELDASyncResult struct {
	NewCount     int
	UpdatedCount int
	Errors       []string
}

// DataboxNotifier interface for notifications
type DataboxNotifier interface {
	NotifyNewDocuments(ctx context.Context, accountID uuid.UUID, count int) error
}

// NewELDADataboxSyncJob creates a new ELDA databox sync job
func NewELDADataboxSyncJob(db *pgxpool.Pool, logger *slog.Logger, syncer ELDADataboxSyncer, notifier DataboxNotifier) *ELDADataboxSyncJob {
	return &ELDADataboxSyncJob{
		db:       db,
		logger:   logger,
		syncer:   syncer,
		notifier: notifier,
	}
}

// Run executes the databox sync job
func (j *ELDADataboxSyncJob) Run(ctx context.Context) error {
	j.logger.Info("starting ELDA databox sync job")

	// Get all active ELDA accounts
	accounts, err := j.getActiveELDAAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get ELDA accounts: %w", err)
	}

	j.logger.Info("syncing ELDA databox for accounts", "count", len(accounts))

	var totalNew, totalUpdated int
	var errors []string

	for _, account := range accounts {
		j.logger.Debug("syncing account", "account_id", account.ID, "name", account.Name)

		syncResult, err := j.syncer.Sync(ctx, account.ID)
		if err != nil {
			j.logger.Error("sync failed", "account_id", account.ID, "error", err)
			errors = append(errors, fmt.Sprintf("%s: %s", account.Name, err.Error()))
			continue
		}

		totalNew += syncResult.NewCount
		totalUpdated += syncResult.UpdatedCount

		// Send notification for new documents
		if syncResult.NewCount > 0 && j.notifier != nil {
			if err := j.notifier.NotifyNewDocuments(ctx, account.ID, syncResult.NewCount); err != nil {
				j.logger.Warn("failed to send notification",
					"account_id", account.ID,
					"error", err)
			}
		}

		// Update last sync timestamp
		if err := j.updateLastSync(ctx, account.ID); err != nil {
			j.logger.Warn("failed to update last sync",
				"account_id", account.ID,
				"error", err)
		}
	}

	j.logger.Info("ELDA databox sync completed",
		"accounts", len(accounts),
		"new_documents", totalNew,
		"updated_documents", totalUpdated,
		"errors", len(errors))

	return nil
}

// getActiveELDAAccounts retrieves all active ELDA accounts
func (j *ELDADataboxSyncJob) getActiveELDAAccounts(ctx context.Context) ([]ELDAAccountInfo, error) {
	query := `
		SELECT id, name
		FROM elda_accounts
		WHERE active = true
		  AND databox_sync_enabled = true
	`

	rows, err := j.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query ELDA accounts: %w", err)
	}
	defer rows.Close()

	var accounts []ELDAAccountInfo
	for rows.Next() {
		var acc ELDAAccountInfo
		if err := rows.Scan(&acc.ID, &acc.Name); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, acc)
	}

	return accounts, nil
}

// updateLastSync updates the last sync timestamp for an account
func (j *ELDADataboxSyncJob) updateLastSync(ctx context.Context, accountID uuid.UUID) error {
	query := `
		UPDATE elda_accounts
		SET last_databox_sync = $2
		WHERE id = $1
	`

	_, err := j.db.Exec(ctx, query, accountID, time.Now())
	return err
}

// GetSyncStatus returns the sync status for all accounts
func (j *ELDADataboxSyncJob) GetSyncStatus(ctx context.Context) ([]ELDADataboxSyncStatus, error) {
	query := `
		SELECT
			ea.id,
			ea.name,
			ea.last_databox_sync,
			ea.databox_sync_enabled,
			COALESCE((SELECT COUNT(*) FROM elda_documents WHERE elda_account_id = ea.id AND is_read = false), 0) as unread_count,
			COALESCE((SELECT COUNT(*) FROM elda_documents WHERE elda_account_id = ea.id), 0) as total_count
		FROM elda_accounts ea
		WHERE ea.active = true
		ORDER BY ea.name
	`

	rows, err := j.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query sync status: %w", err)
	}
	defer rows.Close()

	var statuses []ELDADataboxSyncStatus
	for rows.Next() {
		var status ELDADataboxSyncStatus
		err := rows.Scan(
			&status.AccountID,
			&status.AccountName,
			&status.LastSyncAt,
			&status.SyncEnabled,
			&status.UnreadCount,
			&status.TotalCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan sync status: %w", err)
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// ELDADataboxSyncStatus contains sync status for an account
type ELDADataboxSyncStatus struct {
	AccountID   uuid.UUID  `json:"account_id"`
	AccountName string     `json:"account_name"`
	LastSyncAt  *time.Time `json:"last_sync_at,omitempty"`
	SyncEnabled bool       `json:"sync_enabled"`
	UnreadCount int        `json:"unread_count"`
	TotalCount  int        `json:"total_count"`
}

// EnableSync enables databox sync for an account
func (j *ELDADataboxSyncJob) EnableSync(ctx context.Context, accountID uuid.UUID) error {
	query := `UPDATE elda_accounts SET databox_sync_enabled = true WHERE id = $1`
	_, err := j.db.Exec(ctx, query, accountID)
	return err
}

// DisableSync disables databox sync for an account
func (j *ELDADataboxSyncJob) DisableSync(ctx context.Context, accountID uuid.UUID) error {
	query := `UPDATE elda_accounts SET databox_sync_enabled = false WHERE id = $1`
	_, err := j.db.Exec(ctx, query, accountID)
	return err
}
