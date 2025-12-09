package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/audit"
	"github.com/austrian-business-infrastructure/fo/internal/job"
	"github.com/austrian-business-infrastructure/fo/internal/storage"
	"github.com/google/uuid"
)

// AuditArchive is the job type for archiving old audit logs
const AuditArchiveJobType = "audit_archive"

// AuditArchiveHandler handles audit log archiving
type AuditArchiveHandler struct {
	auditRepo     *audit.Repository
	storageClient storage.Client
	logger        *slog.Logger
	retentionDays int
	batchSize     int
}

// AuditArchiveConfig holds configuration for the audit archive handler
type AuditArchiveConfig struct {
	Logger        *slog.Logger
	RetentionDays int // How long to keep logs before archiving (default: 90)
	BatchSize     int // How many logs to process per batch (default: 1000)
}

// NewAuditArchiveHandler creates a new audit archive handler
func NewAuditArchiveHandler(
	auditRepo *audit.Repository,
	storageClient storage.Client,
	cfg *AuditArchiveConfig,
) *AuditArchiveHandler {
	logger := slog.Default()
	retentionDays := 90
	batchSize := 1000

	if cfg != nil {
		if cfg.Logger != nil {
			logger = cfg.Logger
		}
		if cfg.RetentionDays > 0 {
			retentionDays = cfg.RetentionDays
		}
		if cfg.BatchSize > 0 {
			batchSize = cfg.BatchSize
		}
	}

	return &AuditArchiveHandler{
		auditRepo:     auditRepo,
		storageClient: storageClient,
		logger:        logger,
		retentionDays: retentionDays,
		batchSize:     batchSize,
	}
}

// AuditArchivePayload defines the job payload
type AuditArchivePayload struct {
	TenantID      *uuid.UUID `json:"tenant_id,omitempty"`      // Optional: specific tenant
	RetentionDays *int       `json:"retention_days,omitempty"` // Override default retention
}

// AuditArchiveResult contains the results of an archive operation
type AuditArchiveResult struct {
	TenantsProcessed int            `json:"tenants_processed"`
	TotalArchived    int64          `json:"total_archived"`
	TotalDeleted     int64          `json:"total_deleted"`
	ArchiveFiles     []string       `json:"archive_files"`
	TenantResults    []TenantResult `json:"tenant_results"`
}

// TenantResult contains results for a single tenant
type TenantResult struct {
	TenantID     string `json:"tenant_id"`
	Archived     int64  `json:"archived"`
	Deleted      int64  `json:"deleted"`
	ArchiveFile  string `json:"archive_file,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// Handle executes the audit archive job
func (h *AuditArchiveHandler) Handle(ctx context.Context, j *job.Job) (json.RawMessage, error) {
	h.logger.Info("starting audit archive job", "job_id", j.ID)

	// Parse payload
	var payload AuditArchivePayload
	if len(j.Payload) > 0 {
		if err := json.Unmarshal(j.Payload, &payload); err != nil {
			return nil, fmt.Errorf("parse payload: %w", err)
		}
	}

	// Determine retention period
	retentionDays := h.retentionDays
	if payload.RetentionDays != nil {
		retentionDays = *payload.RetentionDays
	}

	olderThan := time.Now().AddDate(0, 0, -retentionDays)

	var result AuditArchiveResult

	if payload.TenantID != nil {
		// Archive for specific tenant
		tenantResult := h.archiveTenant(ctx, *payload.TenantID, olderThan)
		result.TenantsProcessed = 1
		result.TotalArchived = tenantResult.Archived
		result.TotalDeleted = tenantResult.Deleted
		if tenantResult.ArchiveFile != "" {
			result.ArchiveFiles = []string{tenantResult.ArchiveFile}
		}
		result.TenantResults = []TenantResult{tenantResult}
	} else {
		// Archive for all tenants
		tenantIDs, err := h.auditRepo.GetAllTenantIDs(ctx)
		if err != nil {
			return nil, fmt.Errorf("get tenant IDs: %w", err)
		}

		for _, tenantID := range tenantIDs {
			tenantResult := h.archiveTenant(ctx, tenantID, olderThan)
			result.TenantsProcessed++
			result.TotalArchived += tenantResult.Archived
			result.TotalDeleted += tenantResult.Deleted
			if tenantResult.ArchiveFile != "" {
				result.ArchiveFiles = append(result.ArchiveFiles, tenantResult.ArchiveFile)
			}
			result.TenantResults = append(result.TenantResults, tenantResult)
		}
	}

	h.logger.Info("audit archive completed",
		"tenants", result.TenantsProcessed,
		"archived", result.TotalArchived,
		"deleted", result.TotalDeleted)

	return json.Marshal(result)
}

// archiveTenant archives audit logs for a single tenant
func (h *AuditArchiveHandler) archiveTenant(ctx context.Context, tenantID uuid.UUID, olderThan time.Time) TenantResult {
	result := TenantResult{
		TenantID: tenantID.String(),
	}

	// Count logs to archive
	count, err := h.auditRepo.CountOlderThan(ctx, tenantID, olderThan)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("count logs: %v", err)
		h.logger.Error("failed to count audit logs",
			"tenant_id", tenantID,
			"error", err)
		return result
	}

	if count == 0 {
		h.logger.Debug("no audit logs to archive",
			"tenant_id", tenantID)
		return result
	}

	// Generate archive filename
	archivePath := h.generateArchivePath(tenantID, olderThan)

	// Export logs to storage
	archived, err := h.exportLogs(ctx, tenantID, olderThan, archivePath)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("export logs: %v", err)
		h.logger.Error("failed to export audit logs",
			"tenant_id", tenantID,
			"error", err)
		return result
	}

	result.Archived = archived
	result.ArchiveFile = archivePath

	// Delete archived logs
	deleted, err := h.auditRepo.DeleteOlderThan(ctx, tenantID, olderThan, h.batchSize)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("delete logs: %v", err)
		h.logger.Error("failed to delete archived audit logs",
			"tenant_id", tenantID,
			"error", err)
		return result
	}

	result.Deleted = deleted

	h.logger.Info("tenant audit archive completed",
		"tenant_id", tenantID,
		"archived", archived,
		"deleted", deleted,
		"archive_file", archivePath)

	return result
}

// exportLogs exports audit logs to storage
func (h *AuditArchiveHandler) exportLogs(ctx context.Context, tenantID uuid.UUID, olderThan time.Time, archivePath string) (int64, error) {
	// Use a pipe to stream JSON directly to storage
	pr, pw := io.Pipe()

	// Start JSON encoding in a goroutine
	errChan := make(chan error, 1)
	countChan := make(chan int64, 1)

	go func() {
		defer pw.Close()

		encoder := json.NewEncoder(pw)
		var totalCount int64

		// Write opening bracket
		if _, err := pw.Write([]byte(`{"logs":[`)); err != nil {
			errChan <- err
			return
		}

		first := true
		offset := 0

		for {
			logs, err := h.auditRepo.ListForArchive(ctx, tenantID, olderThan, h.batchSize)
			if err != nil {
				errChan <- err
				return
			}

			if len(logs) == 0 {
				break
			}

			for _, log := range logs {
				if !first {
					if _, err := pw.Write([]byte(",")); err != nil {
						errChan <- err
						return
					}
				}
				first = false

				if err := encoder.Encode(log); err != nil {
					errChan <- err
					return
				}
				totalCount++
			}

			offset += len(logs)
			if len(logs) < h.batchSize {
				break
			}
		}

		// Write closing metadata
		metadata := fmt.Sprintf(`],"metadata":{"tenant_id":"%s","archived_at":"%s","count":%d}}`,
			tenantID.String(),
			time.Now().Format(time.RFC3339),
			totalCount)

		if _, err := pw.Write([]byte(metadata)); err != nil {
			errChan <- err
			return
		}

		countChan <- totalCount
		errChan <- nil
	}()

	// Upload to storage
	if err := h.storageClient.Put(ctx, archivePath, pr, "application/json"); err != nil {
		pr.Close() // Close reader to stop the encoder goroutine
		return 0, fmt.Errorf("upload archive: %w", err)
	}

	// Wait for encoding to complete
	if err := <-errChan; err != nil {
		return 0, fmt.Errorf("encode logs: %w", err)
	}

	return <-countChan, nil
}

// generateArchivePath generates the storage path for an archive file
func (h *AuditArchiveHandler) generateArchivePath(tenantID uuid.UUID, olderThan time.Time) string {
	// Format: archives/audit/{tenant_id}/{year}/{month}/audit-{date}.json
	return filepath.Join(
		"archives",
		"audit",
		tenantID.String(),
		olderThan.Format("2006"),
		olderThan.Format("01"),
		fmt.Sprintf("audit-%s.json", time.Now().Format("20060102-150405")),
	)
}

// Register registers the audit archive handler with a job registry
func (h *AuditArchiveHandler) Register(registry *job.Registry) {
	registry.MustRegister(AuditArchiveJobType, h)
}
