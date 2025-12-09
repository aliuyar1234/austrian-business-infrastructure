package account

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/google/uuid"
)

// AuditAction defines audit action types
type AuditAction string

const (
	AuditActionCreate           AuditAction = "account.create"
	AuditActionView             AuditAction = "account.view"
	AuditActionUpdate           AuditAction = "account.update"
	AuditActionDelete           AuditAction = "account.delete"
	AuditActionCredentialAccess AuditAction = "account.credential_access"
	AuditActionCredentialUpdate AuditAction = "account.credential_update"
	AuditActionConnectionTest   AuditAction = "account.connection_test"
)

// AuditEntry represents an audit log entry for account operations
type AuditEntry struct {
	Timestamp   time.Time         `json:"timestamp"`
	Action      AuditAction       `json:"action"`
	TenantID    uuid.UUID         `json:"tenant_id"`
	UserID      uuid.UUID         `json:"user_id"`
	AccountID   *uuid.UUID        `json:"account_id,omitempty"`
	AccountType string            `json:"account_type,omitempty"`
	IPAddress   string            `json:"ip_address,omitempty"`
	UserAgent   string            `json:"user_agent,omitempty"`
	Success     bool              `json:"success"`
	Details     map[string]string `json:"details,omitempty"`
}

// AuditLogger logs account-related audit events
type AuditLogger struct {
	logger *slog.Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger *slog.Logger) *AuditLogger {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuditLogger{logger: logger}
}

// Log logs an audit entry
func (al *AuditLogger) Log(entry *AuditEntry) {
	attrs := []slog.Attr{
		slog.String("action", string(entry.Action)),
		slog.String("tenant_id", entry.TenantID.String()),
		slog.String("user_id", entry.UserID.String()),
		slog.Bool("success", entry.Success),
	}

	if entry.AccountID != nil {
		attrs = append(attrs, slog.String("account_id", entry.AccountID.String()))
	}
	if entry.AccountType != "" {
		attrs = append(attrs, slog.String("account_type", entry.AccountType))
	}
	if entry.IPAddress != "" {
		attrs = append(attrs, slog.String("ip_address", entry.IPAddress))
	}
	if entry.UserAgent != "" {
		attrs = append(attrs, slog.String("user_agent", entry.UserAgent))
	}

	// Add any additional details
	for k, v := range entry.Details {
		attrs = append(attrs, slog.String(k, v))
	}

	al.logger.LogAttrs(context.Background(), slog.LevelInfo, "audit",
		attrs...,
	)
}

// LogCreate logs an account creation event
func (al *AuditLogger) LogCreate(ctx context.Context, r *http.Request, accountID uuid.UUID, accountType string, success bool) {
	tenantID, _ := uuid.Parse(api.GetTenantID(ctx))
	userID, _ := uuid.Parse(api.GetUserID(ctx))

	al.Log(&AuditEntry{
		Timestamp:   time.Now().UTC(),
		Action:      AuditActionCreate,
		TenantID:    tenantID,
		UserID:      userID,
		AccountID:   &accountID,
		AccountType: accountType,
		IPAddress:   getClientIP(r),
		UserAgent:   r.UserAgent(),
		Success:     success,
	})
}

// LogView logs an account view event
func (al *AuditLogger) LogView(ctx context.Context, r *http.Request, accountID uuid.UUID) {
	tenantID, _ := uuid.Parse(api.GetTenantID(ctx))
	userID, _ := uuid.Parse(api.GetUserID(ctx))

	al.Log(&AuditEntry{
		Timestamp: time.Now().UTC(),
		Action:    AuditActionView,
		TenantID:  tenantID,
		UserID:    userID,
		AccountID: &accountID,
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
		Success:   true,
	})
}

// LogCredentialAccess logs when credentials are decrypted/accessed
func (al *AuditLogger) LogCredentialAccess(ctx context.Context, r *http.Request, accountID uuid.UUID, reason string) {
	tenantID, _ := uuid.Parse(api.GetTenantID(ctx))
	userID, _ := uuid.Parse(api.GetUserID(ctx))

	al.Log(&AuditEntry{
		Timestamp: time.Now().UTC(),
		Action:    AuditActionCredentialAccess,
		TenantID:  tenantID,
		UserID:    userID,
		AccountID: &accountID,
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
		Success:   true,
		Details:   map[string]string{"reason": reason},
	})
}

// LogCredentialUpdate logs credential updates
func (al *AuditLogger) LogCredentialUpdate(ctx context.Context, r *http.Request, accountID uuid.UUID, success bool) {
	tenantID, _ := uuid.Parse(api.GetTenantID(ctx))
	userID, _ := uuid.Parse(api.GetUserID(ctx))

	al.Log(&AuditEntry{
		Timestamp: time.Now().UTC(),
		Action:    AuditActionCredentialUpdate,
		TenantID:  tenantID,
		UserID:    userID,
		AccountID: &accountID,
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
		Success:   success,
	})
}

// LogDelete logs account deletion
func (al *AuditLogger) LogDelete(ctx context.Context, r *http.Request, accountID uuid.UUID, hardDelete bool) {
	tenantID, _ := uuid.Parse(api.GetTenantID(ctx))
	userID, _ := uuid.Parse(api.GetUserID(ctx))

	deleteType := "soft"
	if hardDelete {
		deleteType = "hard"
	}

	al.Log(&AuditEntry{
		Timestamp: time.Now().UTC(),
		Action:    AuditActionDelete,
		TenantID:  tenantID,
		UserID:    userID,
		AccountID: &accountID,
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
		Success:   true,
		Details:   map[string]string{"delete_type": deleteType},
	})
}

// LogConnectionTest logs connection test events
func (al *AuditLogger) LogConnectionTest(ctx context.Context, r *http.Request, accountID uuid.UUID, success bool, errorMsg string) {
	tenantID, _ := uuid.Parse(api.GetTenantID(ctx))
	userID, _ := uuid.Parse(api.GetUserID(ctx))

	details := make(map[string]string)
	if errorMsg != "" {
		details["error"] = errorMsg
	}

	al.Log(&AuditEntry{
		Timestamp: time.Now().UTC(),
		Action:    AuditActionConnectionTest,
		TenantID:  tenantID,
		UserID:    userID,
		AccountID: &accountID,
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
		Success:   success,
		Details:   details,
	})
}

func getClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	return r.RemoteAddr
}

// CredentialAccessMiddleware logs credential access attempts
type CredentialAccessMiddleware struct {
	auditLogger *AuditLogger
}

// NewCredentialAccessMiddleware creates a new credential access middleware
func NewCredentialAccessMiddleware(auditLogger *AuditLogger) *CredentialAccessMiddleware {
	return &CredentialAccessMiddleware{auditLogger: auditLogger}
}

// Wrap wraps a handler to log credential access
func (m *CredentialAccessMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The actual logging happens in the service/handler when credentials are decrypted
		// This middleware just ensures the logging infrastructure is available
		next.ServeHTTP(w, r)
	})
}
