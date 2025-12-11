package security

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

var (
	// ErrNoTenantContext indicates tenant context is not set
	ErrNoTenantContext = errors.New("tenant context not set")
	// ErrInvalidTenantID indicates the tenant ID is invalid
	ErrInvalidTenantID = errors.New("invalid tenant ID")
	// ErrCrossTenantAccess indicates an attempted cross-tenant access violation
	ErrCrossTenantAccess = errors.New("cross-tenant access attempt detected")
)

// TenantContext holds the current tenant context for RLS
type TenantContext struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

// rlsContextKey type for RLS context values
type rlsContextKey string

const (
	tenantContextKey rlsContextKey = "tenant_context"
)

// RLSManager manages Row-Level Security context
type RLSManager struct {
	auditLogger  RLSAuditLogger
	alertHandler RLSAlertHandler
	mu           sync.RWMutex
}

// RLSAuditLogger interface for logging RLS events
type RLSAuditLogger interface {
	LogCrossTenantAttempt(ctx context.Context, event *CrossTenantEvent) error
}

// RLSAlertHandler interface for alerting on security violations
type RLSAlertHandler interface {
	AlertCrossTenantAccess(ctx context.Context, event *CrossTenantEvent) error
}

// CrossTenantEvent represents a cross-tenant access attempt
type CrossTenantEvent struct {
	RequestedTenantID uuid.UUID `json:"requested_tenant_id"`
	ActualTenantID    uuid.UUID `json:"actual_tenant_id"`
	UserID            uuid.UUID `json:"user_id"`
	Operation         string    `json:"operation"`
	ResourceType      string    `json:"resource_type"`
	ResourceID        string    `json:"resource_id,omitempty"`
	IPAddress         string    `json:"ip_address"` // Already anonymized
	UserAgent         string    `json:"user_agent"`
}

// NewRLSManager creates a new RLS manager
func NewRLSManager(auditLogger RLSAuditLogger, alertHandler RLSAlertHandler) *RLSManager {
	return &RLSManager{
		auditLogger:  auditLogger,
		alertHandler: alertHandler,
	}
}

// SetTenantContext sets the PostgreSQL session variable for RLS
// This MUST be called on each database connection before executing queries
//
// PostgreSQL RLS policies use: current_setting('app.tenant_id', true)::uuid
//
// Example usage:
//
//	conn, err := pool.Acquire(ctx)
//	if err != nil { return err }
//	defer conn.Release()
//
//	if err := rlsManager.SetTenantContext(ctx, conn, tenantID); err != nil {
//	    return err
//	}
//	// Now all queries on this connection are RLS-filtered
func (m *RLSManager) SetTenantContext(ctx context.Context, conn DBConn, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenantID
	}

	// Set the PostgreSQL session variable using parameterized query to prevent SQL injection
	// Note: SET doesn't support $1 placeholders directly, but uuid.UUID.String() is safe
	// because UUID validation is performed above and uuid.UUID only contains hex chars and dashes
	_, err := conn.ExecContext(ctx, "SELECT set_config('app.tenant_id', $1, false)", tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	return nil
}

// ClearTenantContext removes the tenant context from a connection
// Call this when returning a connection to a pool in multi-tenant scenarios
func (m *RLSManager) ClearTenantContext(ctx context.Context, conn DBConn) error {
	_, err := conn.ExecContext(ctx, "RESET app.tenant_id")
	if err != nil {
		return fmt.Errorf("failed to clear tenant context: %w", err)
	}
	return nil
}

// SetTenantContextSQL sets tenant context using raw SQL connection
func SetTenantContextSQL(ctx context.Context, db *sql.DB, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenantID
	}

	// Use parameterized query via set_config to prevent SQL injection
	_, err := db.ExecContext(ctx, "SELECT set_config('app.tenant_id', $1, false)", tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	return nil
}

// SetTenantContextTx sets tenant context within a transaction
func SetTenantContextTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenantID
	}

	// Use parameterized query via set_config with is_local=true for transaction scope
	_, err := tx.ExecContext(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to set tenant context in transaction: %w", err)
	}
	return nil
}

// DBConn interface for database connection operations
type DBConn interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// WithTenantContext adds tenant context to a context.Context
func WithTenantContext(ctx context.Context, tenantID, userID uuid.UUID) context.Context {
	tc := &TenantContext{
		TenantID: tenantID,
		UserID:   userID,
	}
	return context.WithValue(ctx, tenantContextKey, tc)
}

// GetTenantContext retrieves tenant context from context.Context
func GetTenantContext(ctx context.Context) (*TenantContext, error) {
	tc, ok := ctx.Value(tenantContextKey).(*TenantContext)
	if !ok || tc == nil {
		return nil, ErrNoTenantContext
	}
	return tc, nil
}

// GetTenantID retrieves just the tenant ID from context
func GetTenantID(ctx context.Context) (uuid.UUID, error) {
	tc, err := GetTenantContext(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return tc.TenantID, nil
}

// MustGetTenantID retrieves tenant ID or panics - use only when tenant context is guaranteed
func MustGetTenantID(ctx context.Context) uuid.UUID {
	id, err := GetTenantID(ctx)
	if err != nil {
		panic("tenant context not set: " + err.Error())
	}
	return id
}

// ValidateTenantAccess checks if the requested tenant matches the context tenant
// Returns an error and logs the attempt if there's a mismatch
func (m *RLSManager) ValidateTenantAccess(ctx context.Context, requestedTenantID uuid.UUID, operation, resourceType, resourceID string) error {
	tc, err := GetTenantContext(ctx)
	if err != nil {
		return err
	}

	if tc.TenantID != requestedTenantID {
		event := &CrossTenantEvent{
			RequestedTenantID: requestedTenantID,
			ActualTenantID:    tc.TenantID,
			UserID:            tc.UserID,
			Operation:         operation,
			ResourceType:      resourceType,
			ResourceID:        resourceID,
		}

		// Log the attempt
		if m.auditLogger != nil {
			_ = m.auditLogger.LogCrossTenantAttempt(ctx, event)
		}

		// Alert security team
		if m.alertHandler != nil {
			_ = m.alertHandler.AlertCrossTenantAccess(ctx, event)
		}

		return ErrCrossTenantAccess
	}

	return nil
}

// DetectCrossTenantAccess detects and reports cross-tenant access attempts
// This is called when a query unexpectedly returns data for a different tenant
func (m *RLSManager) DetectCrossTenantAccess(ctx context.Context, expectedTenantID, actualTenantID uuid.UUID, operation, resourceType string) error {
	if expectedTenantID == actualTenantID {
		return nil // No violation
	}

	tc, _ := GetTenantContext(ctx)
	userID := uuid.Nil
	if tc != nil {
		userID = tc.UserID
	}

	event := &CrossTenantEvent{
		RequestedTenantID: expectedTenantID,
		ActualTenantID:    actualTenantID,
		UserID:            userID,
		Operation:         operation,
		ResourceType:      resourceType,
	}

	// Log the attempt
	if m.auditLogger != nil {
		_ = m.auditLogger.LogCrossTenantAttempt(ctx, event)
	}

	// Alert security team
	if m.alertHandler != nil {
		_ = m.alertHandler.AlertCrossTenantAccess(ctx, event)
	}

	return ErrCrossTenantAccess
}

// TenantAwarePool wraps a database pool with automatic tenant context setting
type TenantAwarePool struct {
	db         *sql.DB
	rlsManager *RLSManager
}

// NewTenantAwarePool creates a new tenant-aware database pool wrapper
func NewTenantAwarePool(db *sql.DB, rlsManager *RLSManager) *TenantAwarePool {
	return &TenantAwarePool{
		db:         db,
		rlsManager: rlsManager,
	}
}

// QueryContext executes a query with automatic tenant context
func (p *TenantAwarePool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant context required for query: %w", err)
	}

	// Set tenant context
	if err := SetTenantContextSQL(ctx, p.db, tenantID); err != nil {
		return nil, err
	}

	return p.db.QueryContext(ctx, query, args...)
}

// ExecContext executes a statement with automatic tenant context
func (p *TenantAwarePool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant context required for exec: %w", err)
	}

	// Set tenant context
	if err := SetTenantContextSQL(ctx, p.db, tenantID); err != nil {
		return nil, err
	}

	return p.db.ExecContext(ctx, query, args...)
}

// BeginTx starts a transaction with automatic tenant context
func (p *TenantAwarePool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant context required for transaction: %w", err)
	}

	tx, err := p.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Set tenant context within transaction
	if err := SetTenantContextTx(ctx, tx, tenantID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	return tx, nil
}

// DB returns the underlying database connection
func (p *TenantAwarePool) DB() *sql.DB {
	return p.db
}

// NullRLSAuditLogger is a no-op audit logger
type NullRLSAuditLogger struct{}

func (n *NullRLSAuditLogger) LogCrossTenantAttempt(ctx context.Context, event *CrossTenantEvent) error {
	return nil
}

// NullRLSAlertHandler is a no-op alert handler
type NullRLSAlertHandler struct{}

func (n *NullRLSAlertHandler) AlertCrossTenantAccess(ctx context.Context, event *CrossTenantEvent) error {
	return nil
}
