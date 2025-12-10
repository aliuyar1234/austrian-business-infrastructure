package unit

import (
	"context"
	"database/sql"
	"testing"

	"austrian-business-infrastructure/internal/security"
	"github.com/google/uuid"
)

func TestRLS_WithTenantContext(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	ctx := security.WithTenantContext(context.Background(), tenantID, userID)

	// Get tenant context back
	tc, err := security.GetTenantContext(ctx)
	if err != nil {
		t.Fatalf("Failed to get tenant context: %v", err)
	}

	if tc.TenantID != tenantID {
		t.Errorf("TenantID = %v, want %v", tc.TenantID, tenantID)
	}

	if tc.UserID != userID {
		t.Errorf("UserID = %v, want %v", tc.UserID, userID)
	}
}

func TestRLS_GetTenantID(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	ctx := security.WithTenantContext(context.Background(), tenantID, userID)

	got, err := security.GetTenantID(ctx)
	if err != nil {
		t.Fatalf("Failed to get tenant ID: %v", err)
	}

	if got != tenantID {
		t.Errorf("GetTenantID() = %v, want %v", got, tenantID)
	}
}

func TestRLS_GetTenantID_NoContext(t *testing.T) {
	ctx := context.Background()

	_, err := security.GetTenantID(ctx)
	if err == nil {
		t.Error("Expected error when getting tenant ID without context")
	}

	if err != security.ErrNoTenantContext {
		t.Errorf("Expected ErrNoTenantContext, got %v", err)
	}
}

func TestRLS_GetTenantContext_NoContext(t *testing.T) {
	ctx := context.Background()

	_, err := security.GetTenantContext(ctx)
	if err != security.ErrNoTenantContext {
		t.Errorf("Expected ErrNoTenantContext, got %v", err)
	}
}

func TestRLS_MustGetTenantID(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	ctx := security.WithTenantContext(context.Background(), tenantID, userID)

	got := security.MustGetTenantID(ctx)
	if got != tenantID {
		t.Errorf("MustGetTenantID() = %v, want %v", got, tenantID)
	}
}

func TestRLS_MustGetTenantID_Panics(t *testing.T) {
	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected MustGetTenantID to panic without context")
		}
	}()

	_ = security.MustGetTenantID(ctx)
}

func TestRLSManager_ValidateTenantAccess_Success(t *testing.T) {
	manager := security.NewRLSManager(&security.NullRLSAuditLogger{}, &security.NullRLSAlertHandler{})

	tenantID := uuid.New()
	userID := uuid.New()

	ctx := security.WithTenantContext(context.Background(), tenantID, userID)

	// Same tenant - should succeed
	err := manager.ValidateTenantAccess(ctx, tenantID, "read", "document", "doc-123")
	if err != nil {
		t.Errorf("Expected no error for same tenant, got %v", err)
	}
}

func TestRLSManager_ValidateTenantAccess_Blocked(t *testing.T) {
	auditLogger := &mockRLSAuditLogger{}
	alertHandler := &mockRLSAlertHandler{}
	manager := security.NewRLSManager(auditLogger, alertHandler)

	tenantA := uuid.New()
	tenantB := uuid.New()
	userID := uuid.New()

	ctx := security.WithTenantContext(context.Background(), tenantA, userID)

	// Different tenant - should fail
	err := manager.ValidateTenantAccess(ctx, tenantB, "read", "document", "doc-123")
	if err != security.ErrCrossTenantAccess {
		t.Errorf("Expected ErrCrossTenantAccess, got %v", err)
	}

	// Verify logging occurred
	if !auditLogger.called {
		t.Error("Expected audit logger to be called")
	}

	// Verify alert occurred
	if !alertHandler.called {
		t.Error("Expected alert handler to be called")
	}
}

func TestRLSManager_DetectCrossTenantAccess(t *testing.T) {
	auditLogger := &mockRLSAuditLogger{}
	alertHandler := &mockRLSAlertHandler{}
	manager := security.NewRLSManager(auditLogger, alertHandler)

	tenantA := uuid.New()
	tenantB := uuid.New()
	userID := uuid.New()

	ctx := security.WithTenantContext(context.Background(), tenantA, userID)

	// Same tenant - no detection
	err := manager.DetectCrossTenantAccess(ctx, tenantA, tenantA, "read", "document")
	if err != nil {
		t.Errorf("Expected no error for same tenant, got %v", err)
	}

	// Different tenant - detection
	err = manager.DetectCrossTenantAccess(ctx, tenantA, tenantB, "read", "document")
	if err != security.ErrCrossTenantAccess {
		t.Errorf("Expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestRLSManager_SetTenantContext_InvalidID(t *testing.T) {
	manager := security.NewRLSManager(nil, nil)

	err := manager.SetTenantContext(context.Background(), &mockDBConn{}, uuid.Nil)
	if err != security.ErrInvalidTenantID {
		t.Errorf("Expected ErrInvalidTenantID for nil UUID, got %v", err)
	}
}

// Mock implementations

type mockRLSAuditLogger struct {
	called bool
	event  *security.CrossTenantEvent
}

func (m *mockRLSAuditLogger) LogCrossTenantAttempt(ctx context.Context, event *security.CrossTenantEvent) error {
	m.called = true
	m.event = event
	return nil
}

type mockRLSAlertHandler struct {
	called bool
	event  *security.CrossTenantEvent
}

func (m *mockRLSAlertHandler) AlertCrossTenantAccess(ctx context.Context, event *security.CrossTenantEvent) error {
	m.called = true
	m.event = event
	return nil
}

type mockDBConn struct {
	query string
}

type mockSQLResult struct{}

func (r mockSQLResult) LastInsertId() (int64, error) { return 0, nil }
func (r mockSQLResult) RowsAffected() (int64, error) { return 0, nil }

func (m *mockDBConn) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	m.query = query
	return mockSQLResult{}, nil
}
