// Package dsgvo provides GDPR/DSGVO compliance features for data export and deletion
package dsgvo

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrExportNotFound indicates the export request was not found
	ErrExportNotFound = errors.New("export request not found")
	// ErrExportNotReady indicates the export is still being generated
	ErrExportNotReady = errors.New("export not ready for download")
	// ErrExportExpired indicates the export has expired
	ErrExportExpired = errors.New("export has expired")
)

// ExportStatus represents the status of an export request
type ExportStatus string

const (
	ExportStatusPending    ExportStatus = "pending"
	ExportStatusProcessing ExportStatus = "processing"
	ExportStatusCompleted  ExportStatus = "completed"
	ExportStatusFailed     ExportStatus = "failed"
	ExportStatusExpired    ExportStatus = "expired"
)

// ExportRequest represents a DSGVO data export request
type ExportRequest struct {
	ID          uuid.UUID     `json:"id"`
	TenantID    uuid.UUID     `json:"tenant_id"`
	RequestedBy uuid.UUID     `json:"requested_by"`
	Status      ExportStatus  `json:"status"`
	FilePath    *string       `json:"file_path,omitempty"`
	FileSize    *int64        `json:"file_size,omitempty"`
	Error       *string       `json:"error,omitempty"`
	ExpiresAt   *time.Time    `json:"expires_at,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
}

// ExportManifest describes the contents of an export archive
type ExportManifest struct {
	Version      string            `json:"version"`
	ExportDate   time.Time         `json:"export_date"`
	TenantID     string            `json:"tenant_id"`
	RequestedBy  string            `json:"requested_by"`
	DataTypes    []string          `json:"data_types"`
	RecordCounts map[string]int    `json:"record_counts"`
	Files        []ManifestFile    `json:"files"`
}

// ManifestFile describes a file in the export
type ManifestFile struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RecordCount int    `json:"record_count"`
}

// DataCollector interface for collecting tenant data
type DataCollector interface {
	// CollectUserData returns all user data for a tenant
	CollectUserData(ctx context.Context, tenantID uuid.UUID) ([]map[string]interface{}, error)
	// CollectAccountData returns all FinanzOnline account data for a tenant
	CollectAccountData(ctx context.Context, tenantID uuid.UUID) ([]map[string]interface{}, error)
	// CollectDocumentData returns all document metadata for a tenant
	CollectDocumentData(ctx context.Context, tenantID uuid.UUID) ([]map[string]interface{}, error)
	// CollectAuditLogData returns all audit logs for a tenant
	CollectAuditLogData(ctx context.Context, tenantID uuid.UUID) ([]map[string]interface{}, error)
}

// Exporter handles DSGVO data exports
type Exporter struct {
	collector  DataCollector
	exportDir  string
	expireDays int
}

// NewExporter creates a new DSGVO data exporter
func NewExporter(collector DataCollector, exportDir string) *Exporter {
	return &Exporter{
		collector:  collector,
		exportDir:  exportDir,
		expireDays: 7, // Exports expire after 7 days
	}
}

// CreateExport generates a complete DSGVO data export for a tenant
func (e *Exporter) CreateExport(ctx context.Context, request *ExportRequest) error {
	// Update status to processing
	request.Status = ExportStatusProcessing

	// Collect all data
	userData, err := e.collector.CollectUserData(ctx, request.TenantID)
	if err != nil {
		return e.failExport(request, fmt.Errorf("failed to collect user data: %w", err))
	}

	accountData, err := e.collector.CollectAccountData(ctx, request.TenantID)
	if err != nil {
		return e.failExport(request, fmt.Errorf("failed to collect account data: %w", err))
	}

	documentData, err := e.collector.CollectDocumentData(ctx, request.TenantID)
	if err != nil {
		return e.failExport(request, fmt.Errorf("failed to collect document data: %w", err))
	}

	auditLogData, err := e.collector.CollectAuditLogData(ctx, request.TenantID)
	if err != nil {
		return e.failExport(request, fmt.Errorf("failed to collect audit log data: %w", err))
	}

	// Create export directory
	exportPath := filepath.Join(e.exportDir, request.TenantID.String())
	if err := os.MkdirAll(exportPath, 0750); err != nil {
		return e.failExport(request, fmt.Errorf("failed to create export directory: %w", err))
	}

	// Create ZIP file
	zipPath := filepath.Join(exportPath, fmt.Sprintf("export-%s.zip", request.ID.String()))
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return e.failExport(request, fmt.Errorf("failed to create zip file: %w", err))
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Write data files
	manifest := &ExportManifest{
		Version:      "1.0",
		ExportDate:   time.Now(),
		TenantID:     request.TenantID.String(),
		RequestedBy:  request.RequestedBy.String(),
		DataTypes:    []string{"users", "accounts", "documents", "audit_logs"},
		RecordCounts: make(map[string]int),
		Files:        make([]ManifestFile, 0),
	}

	// Add users.json
	if err := e.writeJSONToZip(zipWriter, "users.json", userData); err != nil {
		return e.failExport(request, err)
	}
	manifest.RecordCounts["users"] = len(userData)
	manifest.Files = append(manifest.Files, ManifestFile{
		Name:        "users.json",
		Description: "User accounts and profile data",
		RecordCount: len(userData),
	})

	// Add accounts.json
	if err := e.writeJSONToZip(zipWriter, "accounts.json", accountData); err != nil {
		return e.failExport(request, err)
	}
	manifest.RecordCounts["accounts"] = len(accountData)
	manifest.Files = append(manifest.Files, ManifestFile{
		Name:        "accounts.json",
		Description: "FinanzOnline account configurations",
		RecordCount: len(accountData),
	})

	// Add documents.json
	if err := e.writeJSONToZip(zipWriter, "documents.json", documentData); err != nil {
		return e.failExport(request, err)
	}
	manifest.RecordCounts["documents"] = len(documentData)
	manifest.Files = append(manifest.Files, ManifestFile{
		Name:        "documents.json",
		Description: "Document metadata (content stored separately)",
		RecordCount: len(documentData),
	})

	// Add audit_logs.json
	if err := e.writeJSONToZip(zipWriter, "audit_logs.json", auditLogData); err != nil {
		return e.failExport(request, err)
	}
	manifest.RecordCounts["audit_logs"] = len(auditLogData)
	manifest.Files = append(manifest.Files, ManifestFile{
		Name:        "audit_logs.json",
		Description: "Security and activity audit logs",
		RecordCount: len(auditLogData),
	})

	// Add manifest.json
	if err := e.writeJSONToZip(zipWriter, "manifest.json", manifest); err != nil {
		return e.failExport(request, err)
	}

	// Close zip writer to flush
	if err := zipWriter.Close(); err != nil {
		return e.failExport(request, fmt.Errorf("failed to close zip: %w", err))
	}

	// Get file info
	fileInfo, err := zipFile.Stat()
	if err != nil {
		return e.failExport(request, fmt.Errorf("failed to stat zip file: %w", err))
	}

	// Update request with completion info
	now := time.Now()
	expiresAt := now.AddDate(0, 0, e.expireDays)
	fileSize := fileInfo.Size()

	request.Status = ExportStatusCompleted
	request.FilePath = &zipPath
	request.FileSize = &fileSize
	request.CompletedAt = &now
	request.ExpiresAt = &expiresAt

	return nil
}

// writeJSONToZip writes a JSON file to the zip archive
func (e *Exporter) writeJSONToZip(zipWriter *zip.Writer, filename string, data interface{}) error {
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create %s in zip: %w", filename, err)
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	return nil
}

// failExport marks an export as failed
func (e *Exporter) failExport(request *ExportRequest, err error) error {
	request.Status = ExportStatusFailed
	errMsg := err.Error()
	request.Error = &errMsg
	return err
}

// GetExportFile returns a reader for the export file
func (e *Exporter) GetExportFile(request *ExportRequest) (io.ReadCloser, error) {
	if request.Status != ExportStatusCompleted {
		return nil, ErrExportNotReady
	}

	if request.FilePath == nil {
		return nil, ErrExportNotReady
	}

	// Check if expired
	if request.ExpiresAt != nil && time.Now().After(*request.ExpiresAt) {
		return nil, ErrExportExpired
	}

	return os.Open(*request.FilePath)
}

// CleanupExpiredExports removes expired export files
func (e *Exporter) CleanupExpiredExports(requests []*ExportRequest) error {
	for _, req := range requests {
		if req.ExpiresAt != nil && time.Now().After(*req.ExpiresAt) && req.FilePath != nil {
			if err := os.Remove(*req.FilePath); err != nil && !os.IsNotExist(err) {
				// Log error but continue cleanup
				continue
			}
		}
	}
	return nil
}
