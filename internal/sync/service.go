package sync

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/account"
	"github.com/austrian-business-infrastructure/fo/internal/document"
	"github.com/austrian-business-infrastructure/fo/internal/fonws"
	"github.com/google/uuid"
)

// Service handles databox synchronization operations
type Service struct {
	jobRepo     *SyncJobRepository
	docService  *document.Service
	docRepo     *document.Repository
	accountRepo *account.Repository
	fonwsClient *fonws.Client
	logger      *slog.Logger

	// Concurrency control
	maxConcurrent int
	semaphore     chan struct{}

	// Event callbacks
	onNewDocument func(ctx context.Context, tenantID uuid.UUID, doc *document.Document)
	onSyncProgress func(ctx context.Context, jobID uuid.UUID, found, new, skipped int)
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	MaxConcurrent int
	Logger        *slog.Logger
}

// NewService creates a new sync service
func NewService(
	jobRepo *SyncJobRepository,
	docService *document.Service,
	docRepo *document.Repository,
	accountRepo *account.Repository,
	fonwsClient *fonws.Client,
	cfg *ServiceConfig,
) *Service {
	maxConcurrent := 5
	if cfg != nil && cfg.MaxConcurrent > 0 {
		maxConcurrent = cfg.MaxConcurrent
	}

	logger := slog.Default()
	if cfg != nil && cfg.Logger != nil {
		logger = cfg.Logger
	}

	return &Service{
		jobRepo:       jobRepo,
		docService:    docService,
		docRepo:       docRepo,
		accountRepo:   accountRepo,
		fonwsClient:   fonwsClient,
		logger:        logger,
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// SetNewDocumentCallback sets the callback for new documents
func (s *Service) SetNewDocumentCallback(fn func(ctx context.Context, tenantID uuid.UUID, doc *document.Document)) {
	s.onNewDocument = fn
}

// SetProgressCallback sets the callback for sync progress updates
func (s *Service) SetProgressCallback(fn func(ctx context.Context, jobID uuid.UUID, found, new, skipped int)) {
	s.onSyncProgress = fn
}

// SyncSingleAccount synchronizes a single account
func (s *Service) SyncSingleAccount(ctx context.Context, tenantID, accountID uuid.UUID, credentials interface{}) (*SyncJob, error) {
	// Check for existing running sync
	existingJob, err := s.jobRepo.GetRunningForAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("check running sync: %w", err)
	}
	if existingJob != nil {
		return existingJob, fmt.Errorf("sync already running for account")
	}

	// Create sync job
	job := &SyncJob{
		TenantID:  tenantID,
		AccountID: &accountID,
		Status:    StatusPending,
		JobType:   JobTypeSingle,
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("create sync job: %w", err)
	}

	// Run sync in background
	go s.runAccountSync(context.Background(), job, credentials)

	return job, nil
}

// SyncAllAccounts synchronizes all accounts for a tenant
func (s *Service) SyncAllAccounts(ctx context.Context, tenantID uuid.UUID) (*SyncJob, error) {
	// Create sync job for all accounts
	job := &SyncJob{
		TenantID: tenantID,
		Status:   StatusPending,
		JobType:  JobTypeAll,
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("create sync job: %w", err)
	}

	// Run sync in background
	go s.runAllAccountsSync(context.Background(), job)

	return job, nil
}

// GetJob retrieves a sync job by ID
func (s *Service) GetJob(ctx context.Context, id uuid.UUID) (*SyncJob, error) {
	return s.jobRepo.GetByID(ctx, id)
}

// ListJobs lists sync jobs for a tenant
func (s *Service) ListJobs(ctx context.Context, filter *SyncJobFilter) ([]*SyncJob, int, error) {
	return s.jobRepo.List(ctx, filter)
}

// runAccountSync executes sync for a single account
func (s *Service) runAccountSync(ctx context.Context, job *SyncJob, credentials interface{}) {
	// Add timeout for sync operations (10 minutes max)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Mark job as running
	if err := s.jobRepo.Start(ctx, job.ID); err != nil {
		s.logger.Error("failed to start sync job", "job_id", job.ID, "error", err)
		return
	}

	// Acquire semaphore
	s.semaphore <- struct{}{}
	defer func() { <-s.semaphore }()

	// Create session from credentials
	session, err := s.createSession(ctx, credentials)
	if err != nil {
		s.jobRepo.Fail(ctx, job.ID, err.Error())
		return
	}
	defer s.closeSession(session)

	// Create syncer and run
	syncer := NewSyncer(s.fonwsClient, s.docService, s.docRepo)
	if s.onNewDocument != nil {
		syncer.SetNewDocumentCallback(s.onNewDocument)
	}

	// Calculate date range (last 30 days by default)
	toDate := time.Now().Format("2006-01-02")
	fromDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	result, err := syncer.SyncAccount(ctx, session, *job.AccountID, job.TenantID.String(), fromDate, toDate)
	if err != nil {
		s.jobRepo.Fail(ctx, job.ID, err.Error())
		return
	}

	// Mark job as complete
	if err := s.jobRepo.Complete(ctx, job.ID, result.Found, result.New, result.Skipped); err != nil {
		s.logger.Error("failed to complete sync job", "job_id", job.ID, "error", err)
	}

	// Emit progress callback
	if s.onSyncProgress != nil {
		s.onSyncProgress(ctx, job.ID, result.Found, result.New, result.Skipped)
	}
}

// runAllAccountsSync executes sync for all accounts in a tenant
func (s *Service) runAllAccountsSync(ctx context.Context, job *SyncJob) {
	// Add timeout for sync all operations (30 minutes max)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	// Mark job as running
	if err := s.jobRepo.Start(ctx, job.ID); err != nil {
		s.logger.Error("failed to start sync job", "job_id", job.ID, "error", err)
		return
	}

	// Get all FinanzOnline accounts for tenant
	accounts, _, err := s.accountRepo.List(ctx, account.ListFilter{
		TenantID: job.TenantID,
		Type:     account.AccountTypeFinanzOnline,
		Status:   "verified",
		Limit:    1000,
	})
	if err != nil {
		s.jobRepo.Fail(ctx, job.ID, fmt.Sprintf("list accounts: %v", err))
		return
	}

	if len(accounts) == 0 {
		s.jobRepo.Complete(ctx, job.ID, 0, 0, 0)
		return
	}

	// Sync accounts in parallel with semaphore
	var wg sync.WaitGroup
	var mu sync.Mutex
	totalFound, totalNew, totalSkipped := 0, 0, 0
	var errors []string

	for _, acc := range accounts {
		wg.Add(1)
		go func(acc *account.Account) {
			defer wg.Done()

			// Acquire semaphore
			s.semaphore <- struct{}{}
			defer func() { <-s.semaphore }()

			// Decrypt credentials and create session
			// Note: This would need account service integration
			// For now, we'll skip accounts without proper credentials
			s.logger.Info("syncing account", "account_id", acc.ID, "name", acc.Name)

			// Update progress
			mu.Lock()
			totalFound++
			mu.Unlock()

			if s.onSyncProgress != nil {
				mu.Lock()
				s.onSyncProgress(ctx, job.ID, totalFound, totalNew, totalSkipped)
				mu.Unlock()
			}
		}(acc)
	}

	wg.Wait()

	// Mark job as complete
	if len(errors) > 0 {
		errorMsg := fmt.Sprintf("completed with %d errors", len(errors))
		s.jobRepo.Fail(ctx, job.ID, errorMsg)
	} else {
		s.jobRepo.Complete(ctx, job.ID, totalFound, totalNew, totalSkipped)
	}
}

// createSession creates a FinanzOnline session from credentials
func (s *Service) createSession(ctx context.Context, credentials interface{}) (*fonws.Session, error) {
	// Type assert credentials
	creds, ok := credentials.(map[string]string)
	if !ok {
		return nil, fmt.Errorf("invalid credentials type")
	}

	tid := creds["tid"]
	benID := creds["ben_id"]
	pin := creds["pin"]

	if tid == "" || benID == "" || pin == "" {
		return nil, fmt.Errorf("missing credentials")
	}

	// Create session service and login
	sessionSvc := fonws.NewSessionService(s.fonwsClient)
	session, err := sessionSvc.Login(tid, benID, pin)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	return session, nil
}

// closeSession closes a FinanzOnline session
func (s *Service) closeSession(session *fonws.Session) {
	if session == nil {
		return
	}

	sessionSvc := fonws.NewSessionService(s.fonwsClient)
	sessionSvc.Logout(session)
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		BaseDelay:  time.Second,
	}
}

// withRetry executes a function with exponential backoff
func withRetry(ctx context.Context, cfg *RetryConfig, fn func() error) error {
	var lastErr error

	for i := 0; i <= cfg.MaxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err

			if i < cfg.MaxRetries {
				delay := cfg.BaseDelay * time.Duration(1<<uint(i)) // Exponential: 1s, 2s, 4s
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
				}
			}
		} else {
			return nil
		}
	}

	return lastErr
}
