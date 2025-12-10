package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"austrian-business-infrastructure/internal/account"
	"austrian-business-infrastructure/internal/job"
	syncpkg "austrian-business-infrastructure/internal/sync"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DataboxSyncPayload contains the job payload for databox sync
type DataboxSyncPayload struct {
	AccountID *uuid.UUID `json:"account_id,omitempty"` // nil = all accounts for tenant
	TenantID  uuid.UUID  `json:"tenant_id"`
	FromDate  string     `json:"from_date,omitempty"` // Optional date filter
	ToDate    string     `json:"to_date,omitempty"`   // Optional date filter
}

// DataboxSyncResult contains the result of a databox sync job
type DataboxSyncResult struct {
	AccountsSynced    int    `json:"accounts_synced"`
	DocumentsFound    int    `json:"documents_found"`
	DocumentsNew      int    `json:"documents_new"`
	DocumentsSkipped  int    `json:"documents_skipped"`
	Errors            []string `json:"errors,omitempty"`
	Duration          string   `json:"duration"`
}

// DataboxSyncHandler handles databox synchronization jobs
type DataboxSyncHandler struct {
	db          *pgxpool.Pool
	syncService *syncpkg.Service
	accountRepo *account.Repository
	logger      *slog.Logger

	// Concurrency control
	maxConcurrent int
	semaphore     chan struct{}

	// Callbacks for real-time notifications
	onProgress func(ctx context.Context, tenantID, jobID uuid.UUID, found, new, skipped int)
	onComplete func(ctx context.Context, tenantID, jobID uuid.UUID, result *DataboxSyncResult)
}

// DataboxSyncHandlerConfig holds handler configuration
type DataboxSyncHandlerConfig struct {
	MaxConcurrent int
	Logger        *slog.Logger
}

// NewDataboxSyncHandler creates a new databox sync handler
func NewDataboxSyncHandler(
	db *pgxpool.Pool,
	syncService *syncpkg.Service,
	accountRepo *account.Repository,
	cfg *DataboxSyncHandlerConfig,
) *DataboxSyncHandler {
	maxConcurrent := 5
	logger := slog.Default()

	if cfg != nil {
		if cfg.MaxConcurrent > 0 {
			maxConcurrent = cfg.MaxConcurrent
		}
		if cfg.Logger != nil {
			logger = cfg.Logger
		}
	}

	return &DataboxSyncHandler{
		db:            db,
		syncService:   syncService,
		accountRepo:   accountRepo,
		logger:        logger,
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// SetProgressCallback sets the callback for sync progress updates
func (h *DataboxSyncHandler) SetProgressCallback(fn func(ctx context.Context, tenantID, jobID uuid.UUID, found, new, skipped int)) {
	h.onProgress = fn
}

// SetCompleteCallback sets the callback for sync completion
func (h *DataboxSyncHandler) SetCompleteCallback(fn func(ctx context.Context, tenantID, jobID uuid.UUID, result *DataboxSyncResult)) {
	h.onComplete = fn
}

// Handle processes a databox sync job
func (h *DataboxSyncHandler) Handle(ctx context.Context, j *job.Job) (json.RawMessage, error) {
	startTime := time.Now()

	// Parse payload
	var payload DataboxSyncPayload
	if err := j.PayloadTo(&payload); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	logger := h.logger.With(
		"job_id", j.ID,
		"tenant_id", payload.TenantID,
	)

	logger.Info("starting databox sync")

	var result *DataboxSyncResult
	var err error

	if payload.AccountID != nil {
		// Sync single account
		logger = logger.With("account_id", *payload.AccountID)
		result, err = h.syncSingleAccount(ctx, j.ID, payload)
	} else {
		// Sync all accounts for tenant
		result, err = h.syncAllAccounts(ctx, j.ID, payload)
	}

	result.Duration = time.Since(startTime).String()

	if err != nil {
		logger.Error("databox sync failed", "error", err, "duration", result.Duration)
		result.Errors = append(result.Errors, err.Error())
	} else {
		logger.Info("databox sync completed",
			"accounts_synced", result.AccountsSynced,
			"documents_found", result.DocumentsFound,
			"documents_new", result.DocumentsNew,
			"duration", result.Duration)
	}

	// Emit completion callback
	if h.onComplete != nil {
		h.onComplete(ctx, payload.TenantID, j.ID, result)
	}

	// Marshal result
	resultJSON, _ := json.Marshal(result)
	return resultJSON, err
}

// syncSingleAccount syncs a single account
func (h *DataboxSyncHandler) syncSingleAccount(ctx context.Context, jobID uuid.UUID, payload DataboxSyncPayload) (*DataboxSyncResult, error) {
	result := &DataboxSyncResult{}

	// Get account
	acc, err := h.accountRepo.GetByIDOnly(ctx, *payload.AccountID)
	if err != nil {
		return result, fmt.Errorf("get account: %w", err)
	}

	// Check if auto-sync is enabled for this account
	if !acc.AutoSyncEnabled {
		h.logger.Debug("auto-sync disabled for account", "account_id", acc.ID)
		return result, nil
	}

	// Perform sync using sync service
	syncJob, err := h.syncService.SyncSingleAccount(ctx, payload.TenantID, acc.ID, nil)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	// Wait for sync to complete (it runs in background)
	// In production, we'd poll the sync job status or use callbacks
	_ = syncJob

	result.AccountsSynced = 1
	// Note: Actual document counts would come from sync service callbacks

	return result, nil
}

// syncAllAccounts syncs all accounts for a tenant
func (h *DataboxSyncHandler) syncAllAccounts(ctx context.Context, jobID uuid.UUID, payload DataboxSyncPayload) (*DataboxSyncResult, error) {
	result := &DataboxSyncResult{}

	// Get all FinanzOnline accounts that are due for sync
	accounts, _, err := h.accountRepo.List(ctx, account.ListFilter{
		TenantID:        payload.TenantID,
		Type:            account.AccountTypeFinanzOnline,
		Status:          "verified",
		AutoSyncEnabled: true,
		Limit:           1000,
	})
	if err != nil {
		return result, fmt.Errorf("list accounts: %w", err)
	}

	if len(accounts) == 0 {
		h.logger.Debug("no accounts to sync", "tenant_id", payload.TenantID)
		return result, nil
	}

	// Filter accounts that are due for sync
	accountsToSync := h.filterAccountsDueForSync(accounts)
	if len(accountsToSync) == 0 {
		h.logger.Debug("no accounts due for sync", "tenant_id", payload.TenantID)
		return result, nil
	}

	h.logger.Info("syncing accounts", "count", len(accountsToSync))

	// Sync accounts in parallel with semaphore
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, acc := range accountsToSync {
		wg.Add(1)
		go func(acc *account.Account) {
			defer wg.Done()

			// Acquire semaphore
			h.semaphore <- struct{}{}
			defer func() { <-h.semaphore }()

			h.logger.Debug("syncing account", "account_id", acc.ID, "name", acc.Name)

			// Trigger sync for this account
			syncJob, err := h.syncService.SyncSingleAccount(ctx, payload.TenantID, acc.ID, nil)
			if err != nil {
				mu.Lock()
				result.Errors = append(result.Errors, fmt.Sprintf("account %s: %v", acc.Name, err))
				mu.Unlock()
				return
			}

			_ = syncJob // Sync runs in background

			mu.Lock()
			result.AccountsSynced++
			mu.Unlock()

			// Emit progress
			if h.onProgress != nil {
				mu.Lock()
				h.onProgress(ctx, payload.TenantID, jobID, result.DocumentsFound, result.DocumentsNew, result.DocumentsSkipped)
				mu.Unlock()
			}
		}(acc)
	}

	wg.Wait()

	return result, nil
}

// filterAccountsDueForSync filters accounts that are due for sync based on their interval
func (h *DataboxSyncHandler) filterAccountsDueForSync(accounts []*account.Account) []*account.Account {
	now := time.Now()
	var due []*account.Account

	for _, acc := range accounts {
		if !acc.AutoSyncEnabled {
			continue
		}

		// If never synced, it's due
		if acc.LastSyncAt == nil {
			due = append(due, acc)
			continue
		}

		// Calculate interval
		interval := job.IntervalToDuration(acc.SyncInterval)
		nextSyncAt := acc.LastSyncAt.Add(interval)

		if now.After(nextSyncAt) || now.Equal(nextSyncAt) {
			due = append(due, acc)
		}
	}

	return due
}

// CreateDefaultSchedules creates default sync schedules for a tenant
func CreateDefaultSchedules(ctx context.Context, scheduler *job.Scheduler, tenantID uuid.UUID) error {
	// Create a schedule for automatic databox sync
	schedule := &job.Schedule{
		TenantID: tenantID,
		Name:     "databox-sync",
		JobType:  job.TypeDataboxSync,
		Interval: job.Interval4Hourly,
		Enabled:  true,
		Timezone: "UTC",
	}
	schedule.JobPayload, _ = json.Marshal(DataboxSyncPayload{
		TenantID: tenantID,
	})

	return scheduler.CreateSchedule(ctx, schedule)
}
