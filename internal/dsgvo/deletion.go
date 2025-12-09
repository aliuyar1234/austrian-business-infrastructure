package dsgvo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrDeletionNotFound indicates the deletion request was not found
	ErrDeletionNotFound = errors.New("deletion request not found")
	// ErrDeletionAlreadyPending indicates a deletion request is already pending
	ErrDeletionAlreadyPending = errors.New("deletion request already pending for this tenant")
	// ErrDeletionAlreadyExecuted indicates the deletion was already executed
	ErrDeletionAlreadyExecuted = errors.New("deletion has already been executed")
	// ErrDeletionCancelled indicates the deletion was cancelled
	ErrDeletionCancelled = errors.New("deletion request has been cancelled")
	// ErrGracePeriodNotExpired indicates the grace period has not yet expired
	ErrGracePeriodNotExpired = errors.New("deletion grace period has not expired")
)

// DeletionStatus represents the status of a deletion request
type DeletionStatus string

const (
	DeletionStatusPending   DeletionStatus = "pending"
	DeletionStatusCancelled DeletionStatus = "cancelled"
	DeletionStatusExecuting DeletionStatus = "executing"
	DeletionStatusCompleted DeletionStatus = "completed"
	DeletionStatusFailed    DeletionStatus = "failed"
)

// GracePeriodDays is the number of days before deletion can be executed (DSGVO requirement)
const GracePeriodDays = 30

// DeletionRequest represents a DSGVO data deletion request
type DeletionRequest struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	RequestedBy     uuid.UUID       `json:"requested_by"`
	Status          DeletionStatus  `json:"status"`
	Reason          *string         `json:"reason,omitempty"`
	ScheduledFor    time.Time       `json:"scheduled_for"`
	GracePeriodDays int             `json:"grace_period_days"`
	Error           *string         `json:"error,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	CancelledAt     *time.Time      `json:"cancelled_at,omitempty"`
	CancelledBy     *uuid.UUID      `json:"cancelled_by,omitempty"`
	ExecutedAt      *time.Time      `json:"executed_at,omitempty"`
}

// DeletionResult holds the result of a deletion execution
type DeletionResult struct {
	UsersDeleted     int `json:"users_deleted"`
	AccountsDeleted  int `json:"accounts_deleted"`
	DocumentsDeleted int `json:"documents_deleted"`
	AuditLogsDeleted int `json:"audit_logs_deleted"`
	FilesDeleted     int `json:"files_deleted"`
}

// DataDeleter interface for deleting tenant data
type DataDeleter interface {
	// DeleteTenantUsers deletes all users for a tenant except the requesting admin
	DeleteTenantUsers(ctx context.Context, tenantID uuid.UUID) (int, error)
	// DeleteTenantAccounts deletes all FinanzOnline accounts for a tenant
	DeleteTenantAccounts(ctx context.Context, tenantID uuid.UUID) (int, error)
	// DeleteTenantDocuments deletes all documents for a tenant
	DeleteTenantDocuments(ctx context.Context, tenantID uuid.UUID) (int, error)
	// DeleteTenantAuditLogs deletes all audit logs for a tenant
	DeleteTenantAuditLogs(ctx context.Context, tenantID uuid.UUID) (int, error)
	// DeleteTenantFiles deletes all stored files for a tenant
	DeleteTenantFiles(ctx context.Context, tenantID uuid.UUID) (int, error)
	// DeleteTenant deletes the tenant itself
	DeleteTenant(ctx context.Context, tenantID uuid.UUID) error
}

// DeletionManager handles DSGVO data deletion requests
type DeletionManager struct {
	deleter DataDeleter
}

// NewDeletionManager creates a new DSGVO deletion manager
func NewDeletionManager(deleter DataDeleter) *DeletionManager {
	return &DeletionManager{deleter: deleter}
}

// CreateDeletionRequest creates a new deletion request with grace period
func (m *DeletionManager) CreateDeletionRequest(tenantID, requestedBy uuid.UUID, reason *string) *DeletionRequest {
	now := time.Now()
	scheduledFor := now.AddDate(0, 0, GracePeriodDays)

	return &DeletionRequest{
		ID:              uuid.New(),
		TenantID:        tenantID,
		RequestedBy:     requestedBy,
		Status:          DeletionStatusPending,
		Reason:          reason,
		ScheduledFor:    scheduledFor,
		GracePeriodDays: GracePeriodDays,
		CreatedAt:       now,
	}
}

// CancelDeletionRequest cancels a pending deletion request
func (m *DeletionManager) CancelDeletionRequest(request *DeletionRequest, cancelledBy uuid.UUID) error {
	if request.Status == DeletionStatusCompleted {
		return ErrDeletionAlreadyExecuted
	}

	if request.Status == DeletionStatusCancelled {
		return ErrDeletionCancelled
	}

	if request.Status == DeletionStatusExecuting {
		return errors.New("cannot cancel deletion while executing")
	}

	now := time.Now()
	request.Status = DeletionStatusCancelled
	request.CancelledAt = &now
	request.CancelledBy = &cancelledBy

	return nil
}

// CanExecuteDeletion checks if a deletion request can be executed
func (m *DeletionManager) CanExecuteDeletion(request *DeletionRequest) error {
	if request.Status == DeletionStatusCompleted {
		return ErrDeletionAlreadyExecuted
	}

	if request.Status == DeletionStatusCancelled {
		return ErrDeletionCancelled
	}

	if time.Now().Before(request.ScheduledFor) {
		return ErrGracePeriodNotExpired
	}

	return nil
}

// ExecuteDeletion performs the actual data deletion
// This should only be called after the grace period has expired
func (m *DeletionManager) ExecuteDeletion(ctx context.Context, request *DeletionRequest) (*DeletionResult, error) {
	// Check if can execute
	if err := m.CanExecuteDeletion(request); err != nil {
		return nil, err
	}

	// Update status
	request.Status = DeletionStatusExecuting

	result := &DeletionResult{}
	var executionError error

	// Delete in order: users, accounts, documents, audit logs, files, tenant
	// Order matters for referential integrity

	// 1. Delete accounts first (may have foreign keys to users)
	if count, err := m.deleter.DeleteTenantAccounts(ctx, request.TenantID); err != nil {
		executionError = fmt.Errorf("failed to delete accounts: %w", err)
	} else {
		result.AccountsDeleted = count
	}

	// 2. Delete documents
	if executionError == nil {
		if count, err := m.deleter.DeleteTenantDocuments(ctx, request.TenantID); err != nil {
			executionError = fmt.Errorf("failed to delete documents: %w", err)
		} else {
			result.DocumentsDeleted = count
		}
	}

	// 3. Delete audit logs (keep for compliance, but anonymize tenant reference)
	if executionError == nil {
		if count, err := m.deleter.DeleteTenantAuditLogs(ctx, request.TenantID); err != nil {
			executionError = fmt.Errorf("failed to delete audit logs: %w", err)
		} else {
			result.AuditLogsDeleted = count
		}
	}

	// 4. Delete files
	if executionError == nil {
		if count, err := m.deleter.DeleteTenantFiles(ctx, request.TenantID); err != nil {
			executionError = fmt.Errorf("failed to delete files: %w", err)
		} else {
			result.FilesDeleted = count
		}
	}

	// 5. Delete users
	if executionError == nil {
		if count, err := m.deleter.DeleteTenantUsers(ctx, request.TenantID); err != nil {
			executionError = fmt.Errorf("failed to delete users: %w", err)
		} else {
			result.UsersDeleted = count
		}
	}

	// 6. Finally delete the tenant
	if executionError == nil {
		if err := m.deleter.DeleteTenant(ctx, request.TenantID); err != nil {
			executionError = fmt.Errorf("failed to delete tenant: %w", err)
		}
	}

	// Update request status
	now := time.Now()
	if executionError != nil {
		request.Status = DeletionStatusFailed
		errMsg := executionError.Error()
		request.Error = &errMsg
		return result, executionError
	}

	request.Status = DeletionStatusCompleted
	request.ExecutedAt = &now

	return result, nil
}

// GetRemainingGracePeriod returns the remaining time until deletion
func (m *DeletionManager) GetRemainingGracePeriod(request *DeletionRequest) time.Duration {
	if request.Status != DeletionStatusPending {
		return 0
	}

	remaining := time.Until(request.ScheduledFor)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// PIIRegistry describes where PII is stored in the system
type PIIRegistry struct {
	Locations []PIILocation `json:"locations"`
}

// PIILocation describes a specific location where PII is stored
type PIILocation struct {
	Entity       string   `json:"entity"`
	Table        string   `json:"table"`
	Fields       []string `json:"fields"`
	Description  string   `json:"description"`
	RetentionDays int     `json:"retention_days,omitempty"`
}

// GetPIIRegistry returns a registry of all PII locations in the system
// This is useful for DSGVO compliance audits
func GetPIIRegistry() *PIIRegistry {
	return &PIIRegistry{
		Locations: []PIILocation{
			{
				Entity:      "User",
				Table:       "users",
				Fields:      []string{"email", "name", "avatar_url"},
				Description: "User account information",
			},
			{
				Entity:      "Account",
				Table:       "fo_accounts",
				Fields:      []string{"teilnehmer_id", "benutzer_id", "encrypted_pin"},
				Description: "FinanzOnline account credentials (PIN encrypted at rest)",
			},
			{
				Entity:      "Invitation",
				Table:       "invitations",
				Fields:      []string{"email"},
				Description: "Pending user invitations",
			},
			{
				Entity:      "AuditLog",
				Table:       "audit_logs",
				Fields:      []string{"ip_address", "user_agent"},
				Description: "Security audit logs (IP anonymized, no content logged)",
				RetentionDays: 365,
			},
			{
				Entity:      "Document",
				Table:       "documents",
				Fields:      []string{"extracted_text", "ai_summary"},
				Description: "Document content and AI analysis results",
			},
			{
				Entity:      "ActionItem",
				Table:       "action_items",
				Fields:      []string{"description", "notes"},
				Description: "Action items extracted from documents",
			},
		},
	}
}

