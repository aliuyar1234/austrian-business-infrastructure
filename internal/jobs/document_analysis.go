package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"austrian-business-infrastructure/internal/analysis"
	"austrian-business-infrastructure/internal/document"
	"austrian-business-infrastructure/internal/job"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DocumentAnalysisPayload contains the job payload for document analysis
type DocumentAnalysisPayload struct {
	DocumentID uuid.UUID `json:"document_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Options    *analysis.AnalysisOptions `json:"options,omitempty"`
	Priority   string `json:"priority,omitempty"` // high, normal, low
	RetryCount int    `json:"retry_count,omitempty"`
}

// DocumentAnalysisResult contains the result of a document analysis job
type DocumentAnalysisResult struct {
	AnalysisID      *uuid.UUID `json:"analysis_id,omitempty"`
	DocumentType    string     `json:"document_type,omitempty"`
	DeadlinesFound  int        `json:"deadlines_found"`
	AmountsFound    int        `json:"amounts_found"`
	ActionItems     int        `json:"action_items"`
	Suggestions     int        `json:"suggestions"`
	ProcessingTimeMs int       `json:"processing_time_ms"`
	Success         bool       `json:"success"`
	Error           string     `json:"error,omitempty"`
}

// DocumentAnalysisHandler handles document analysis jobs
type DocumentAnalysisHandler struct {
	db              *pgxpool.Pool
	analysisService *analysis.Service
	docRepo         *document.Repository
	logger          *slog.Logger

	// Retry configuration
	maxRetries    int
	retryDelay    time.Duration

	// Callbacks
	onComplete func(ctx context.Context, tenantID, documentID uuid.UUID, result *DocumentAnalysisResult)
	onError    func(ctx context.Context, tenantID, documentID uuid.UUID, err error)
}

// DocumentAnalysisHandlerConfig holds handler configuration
type DocumentAnalysisHandlerConfig struct {
	MaxRetries int
	RetryDelay time.Duration
	Logger     *slog.Logger
	DocRepo    *document.Repository
}

// NewDocumentAnalysisHandler creates a new document analysis handler
func NewDocumentAnalysisHandler(
	db *pgxpool.Pool,
	analysisService *analysis.Service,
	cfg *DocumentAnalysisHandlerConfig,
) *DocumentAnalysisHandler {
	maxRetries := 3
	retryDelay := 30 * time.Second
	logger := slog.Default()
	var docRepo *document.Repository

	if cfg != nil {
		if cfg.MaxRetries > 0 {
			maxRetries = cfg.MaxRetries
		}
		if cfg.RetryDelay > 0 {
			retryDelay = cfg.RetryDelay
		}
		if cfg.Logger != nil {
			logger = cfg.Logger
		}
		if cfg.DocRepo != nil {
			docRepo = cfg.DocRepo
		}
	}

	return &DocumentAnalysisHandler{
		db:              db,
		analysisService: analysisService,
		docRepo:         docRepo,
		logger:          logger,
		maxRetries:      maxRetries,
		retryDelay:      retryDelay,
	}
}

// SetCompleteCallback sets the callback for analysis completion
func (h *DocumentAnalysisHandler) SetCompleteCallback(fn func(ctx context.Context, tenantID, documentID uuid.UUID, result *DocumentAnalysisResult)) {
	h.onComplete = fn
}

// SetErrorCallback sets the callback for analysis errors
func (h *DocumentAnalysisHandler) SetErrorCallback(fn func(ctx context.Context, tenantID, documentID uuid.UUID, err error)) {
	h.onError = fn
}

// Handle processes a document analysis job
func (h *DocumentAnalysisHandler) Handle(ctx context.Context, j *job.Job) (json.RawMessage, error) {
	startTime := time.Now()

	// Parse payload
	var payload DocumentAnalysisPayload
	if err := j.PayloadTo(&payload); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	logger := h.logger.With(
		"job_id", j.ID,
		"document_id", payload.DocumentID,
		"tenant_id", payload.TenantID,
	)

	logger.Info("starting document analysis")

	// Get analysis options
	opts := analysis.DefaultOptions()
	if payload.Options != nil {
		opts = *payload.Options
	}

	// Perform analysis
	analysisResult, err := h.analysisService.AnalyzeDocument(ctx, payload.DocumentID, payload.TenantID, opts)

	result := &DocumentAnalysisResult{
		ProcessingTimeMs: int(time.Since(startTime).Milliseconds()),
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()

		logger.Error("document analysis failed",
			"error", err,
			"retry_count", payload.RetryCount,
			"duration_ms", result.ProcessingTimeMs)

		// Check if we should retry
		if payload.RetryCount < h.maxRetries {
			logger.Info("scheduling retry",
				"retry_count", payload.RetryCount+1,
				"max_retries", h.maxRetries)

			// Create retry job
			retryPayload := payload
			retryPayload.RetryCount++
			if err := h.scheduleRetry(ctx, j, &retryPayload); err != nil {
				logger.Error("failed to schedule retry", "error", err)
			}
		}

		// Emit error callback
		if h.onError != nil {
			h.onError(ctx, payload.TenantID, payload.DocumentID, err)
		}

		resultJSON, _ := json.Marshal(result)
		return resultJSON, err
	}

	// Success
	result.Success = true
	result.AnalysisID = &analysisResult.Analysis.ID
	result.DocumentType = analysisResult.Analysis.DocumentType
	result.DeadlinesFound = len(analysisResult.Deadlines)
	result.AmountsFound = len(analysisResult.Amounts)
	result.ActionItems = len(analysisResult.ActionItems)
	result.Suggestions = len(analysisResult.Suggestions)

	// Update document status to "read" after successful analysis (with tenant isolation)
	if h.docRepo != nil {
		if err := h.docRepo.UpdateStatus(ctx, payload.TenantID, payload.DocumentID, document.StatusRead); err != nil {
			logger.Warn("failed to update document status after analysis",
				"document_id", payload.DocumentID,
				"error", err)
		}
	}

	logger.Info("document analysis completed",
		"analysis_id", result.AnalysisID,
		"document_type", result.DocumentType,
		"deadlines", result.DeadlinesFound,
		"amounts", result.AmountsFound,
		"action_items", result.ActionItems,
		"duration_ms", result.ProcessingTimeMs)

	// Emit completion callback
	if h.onComplete != nil {
		h.onComplete(ctx, payload.TenantID, payload.DocumentID, result)
	}

	resultJSON, _ := json.Marshal(result)
	return resultJSON, nil
}

// scheduleRetry schedules a retry job
func (h *DocumentAnalysisHandler) scheduleRetry(ctx context.Context, originalJob *job.Job, payload *DocumentAnalysisPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal retry payload: %w", err)
	}

	retryJob := &job.Job{
		TenantID: payload.TenantID,
		Type:     job.TypeDocumentAnalysis,
		Payload:  payloadJSON,
		Priority: originalJob.Priority,
		Status:   job.StatusPending,
		RunAt:    time.Now().Add(h.retryDelay * time.Duration(payload.RetryCount)),
	}

	// Insert retry job
	query := `
		INSERT INTO jobs (tenant_id, type, payload, priority, status, run_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = h.db.Exec(ctx, query,
		retryJob.TenantID, retryJob.Type, retryJob.Payload,
		retryJob.Priority, retryJob.Status, retryJob.RunAt)

	return err
}

// TriggerAnalysisForNewDocument creates an analysis job for a newly synced document
func TriggerAnalysisForNewDocument(ctx context.Context, db *pgxpool.Pool, tenantID, documentID uuid.UUID, priority string) error {
	if priority == "" {
		priority = "normal"
	}

	payload := DocumentAnalysisPayload{
		DocumentID: documentID,
		TenantID:   tenantID,
		Priority:   priority,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// Map priority to job priority
	jobPriority := 5 // normal
	switch priority {
	case "high":
		jobPriority = 1
	case "low":
		jobPriority = 10
	}

	query := `
		INSERT INTO jobs (tenant_id, type, payload, priority, status, run_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
	_, err = db.Exec(ctx, query, tenantID, job.TypeDocumentAnalysis, payloadJSON, jobPriority, job.StatusPending)

	return err
}

// CreateAnalysisSchedule creates a scheduled job for periodic document analysis
func CreateAnalysisSchedule(ctx context.Context, scheduler *job.Scheduler, tenantID uuid.UUID) error {
	// Note: Usually analysis is triggered on-demand when documents arrive
	// This creates a fallback schedule to catch any missed documents
	schedule := &job.Schedule{
		TenantID: tenantID,
		Name:     "document-analysis-catchup",
		JobType:  job.TypeDocumentAnalysis,
		Interval: job.IntervalDaily,
		Enabled:  false, // Disabled by default, enable if needed
		Timezone: "UTC",
	}

	// Payload for batch processing unanalyzed documents
	schedule.JobPayload, _ = json.Marshal(map[string]interface{}{
		"tenant_id": tenantID,
		"batch":     true,
	})

	return scheduler.CreateSchedule(ctx, schedule)
}
