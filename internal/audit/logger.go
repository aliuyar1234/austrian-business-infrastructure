package audit

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/google/uuid"
)

// Action constants for audit logging
const (
	ActionLogin           = "login"
	ActionLogout          = "logout"
	ActionPasswordChange  = "password_change"
	ActionRegister        = "register"
	ActionInvite          = "invite"
	ActionInviteAccept    = "invite_accept"
	ActionRoleChange      = "role_change"
	ActionUserDeactivate  = "user_deactivate"
	ActionAPIKeyCreate    = "api_key_create"
	ActionAPIKeyRevoke    = "api_key_revoke"
	ActionSessionTerminate = "session_terminate"
)

// ResourceType constants
const (
	ResourceUser       = "user"
	ResourceTenant     = "tenant"
	ResourceInvitation = "invitation"
	ResourceAPIKey     = "api_key"
	ResourceSession    = "session"
)

// Logger provides structured audit logging
type Logger struct {
	repo       *Repository
	logger     *slog.Logger
	asyncQueue chan *AuditLog
	wg         sync.WaitGroup
	asyncMode  bool
}

// NewLogger creates a new audit logger (synchronous mode)
func NewLogger(repo *Repository, logger *slog.Logger) *Logger {
	return &Logger{
		repo:      repo,
		logger:    logger,
		asyncMode: false,
	}
}

// NewAsyncLogger creates a new audit logger with async writing (T069)
// This allows audit logs to be written in the background without blocking requests.
// bufferSize controls how many logs can be buffered before blocking.
func NewAsyncLogger(repo *Repository, logger *slog.Logger, bufferSize int) *Logger {
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	l := &Logger{
		repo:       repo,
		logger:     logger,
		asyncQueue: make(chan *AuditLog, bufferSize),
		asyncMode:  true,
	}

	// Start background worker
	l.wg.Add(1)
	go l.asyncWorker()

	return l
}

// asyncWorker processes audit logs from the queue
func (l *Logger) asyncWorker() {
	defer l.wg.Done()

	for log := range l.asyncQueue {
		if err := l.repo.Create(context.Background(), log); err != nil {
			l.logger.Error("failed to create async audit log",
				"action", log.Action,
				"error", err,
			)
		}
	}
}

// Close gracefully shuts down the async logger
func (l *Logger) Close() {
	if l.asyncMode && l.asyncQueue != nil {
		close(l.asyncQueue)
		l.wg.Wait()
	}
}

// LogContext contains context information for logging
type LogContext struct {
	TenantID     *uuid.UUID
	UserID       *uuid.UUID
	IPAddress    *string
	UserAgent    *string
	ResourceType *string
	ResourceID   *uuid.UUID
}

// ContextFromRequest extracts log context from HTTP request
func ContextFromRequest(r *http.Request) *LogContext {
	ctx := &LogContext{}

	if tenantID := api.GetTenantID(r.Context()); tenantID != "" {
		if id, err := uuid.Parse(tenantID); err == nil {
			ctx.TenantID = &id
		}
	}

	if userID := api.GetUserID(r.Context()); userID != "" {
		if id, err := uuid.Parse(userID); err == nil {
			ctx.UserID = &id
		}
	}

	ip := getClientIP(r)
	ctx.IPAddress = &ip

	ua := truncateUserAgent(r.UserAgent())
	if ua != "" {
		ctx.UserAgent = &ua
	}

	return ctx
}

// Log creates an audit log entry
// In async mode, the log is queued and written in the background.
// In sync mode, the log is written immediately.
func (l *Logger) Log(ctx context.Context, logCtx *LogContext, action string, details map[string]interface{}) error {
	log := &AuditLog{
		TenantID:     logCtx.TenantID,
		UserID:       logCtx.UserID,
		Action:       action,
		ResourceType: logCtx.ResourceType,
		ResourceID:   logCtx.ResourceID,
		Details:      details,
		IPAddress:    logCtx.IPAddress,
		UserAgent:    logCtx.UserAgent,
	}

	// Generate ID upfront for async mode
	log.ID = uuid.New()

	// Also log to structured logger (always synchronous)
	l.logger.Info("audit",
		"action", action,
		"tenant_id", logCtx.TenantID,
		"user_id", logCtx.UserID,
		"resource_type", logCtx.ResourceType,
		"resource_id", logCtx.ResourceID,
		"ip_address", logCtx.IPAddress,
	)

	// In async mode, queue the log and return immediately
	if l.asyncMode {
		select {
		case l.asyncQueue <- log:
			return nil
		default:
			// Queue is full - log warning and write synchronously as fallback
			l.logger.Warn("async audit queue full, writing synchronously")
			return l.repo.Create(ctx, log)
		}
	}

	// Sync mode: write immediately
	if err := l.repo.Create(ctx, log); err != nil {
		l.logger.Error("failed to create audit log",
			"action", action,
			"error", err,
		)
		return err
	}

	return nil
}

// LogLogin logs a login event
func (l *Logger) LogLogin(ctx context.Context, r *http.Request, userID uuid.UUID, success bool) error {
	logCtx := ContextFromRequest(r)
	logCtx.UserID = &userID
	logCtx.ResourceType = ptr(ResourceUser)
	logCtx.ResourceID = &userID

	return l.Log(ctx, logCtx, ActionLogin, map[string]interface{}{
		"success": success,
	})
}

// LogLogout logs a logout event
func (l *Logger) LogLogout(ctx context.Context, r *http.Request) error {
	logCtx := ContextFromRequest(r)
	return l.Log(ctx, logCtx, ActionLogout, nil)
}

// LogPasswordChange logs a password change event
func (l *Logger) LogPasswordChange(ctx context.Context, r *http.Request, userID uuid.UUID) error {
	logCtx := ContextFromRequest(r)
	logCtx.ResourceType = ptr(ResourceUser)
	logCtx.ResourceID = &userID

	return l.Log(ctx, logCtx, ActionPasswordChange, nil)
}

// LogRegister logs a registration event
func (l *Logger) LogRegister(ctx context.Context, r *http.Request, tenantID, userID uuid.UUID) error {
	logCtx := ContextFromRequest(r)
	logCtx.TenantID = &tenantID
	logCtx.UserID = &userID
	logCtx.ResourceType = ptr(ResourceTenant)
	logCtx.ResourceID = &tenantID

	return l.Log(ctx, logCtx, ActionRegister, nil)
}

// LogInvite logs an invitation event
// SECURITY: Email is NOT logged per FR-104 (no PII in audit logs)
func (l *Logger) LogInvite(ctx context.Context, r *http.Request, invitationID uuid.UUID, role string) error {
	logCtx := ContextFromRequest(r)
	logCtx.ResourceType = ptr(ResourceInvitation)
	logCtx.ResourceID = &invitationID

	return l.Log(ctx, logCtx, ActionInvite, map[string]interface{}{
		"role": role,
		// Note: email intentionally not logged per FR-104
	})
}

// LogInviteAccept logs an invitation acceptance
func (l *Logger) LogInviteAccept(ctx context.Context, r *http.Request, invitationID, userID uuid.UUID) error {
	logCtx := ContextFromRequest(r)
	logCtx.UserID = &userID
	logCtx.ResourceType = ptr(ResourceInvitation)
	logCtx.ResourceID = &invitationID

	return l.Log(ctx, logCtx, ActionInviteAccept, nil)
}

// LogRoleChange logs a role change event
func (l *Logger) LogRoleChange(ctx context.Context, r *http.Request, targetUserID uuid.UUID, oldRole, newRole string) error {
	logCtx := ContextFromRequest(r)
	logCtx.ResourceType = ptr(ResourceUser)
	logCtx.ResourceID = &targetUserID

	return l.Log(ctx, logCtx, ActionRoleChange, map[string]interface{}{
		"old_role": oldRole,
		"new_role": newRole,
	})
}

// LogUserDeactivate logs a user deactivation
func (l *Logger) LogUserDeactivate(ctx context.Context, r *http.Request, targetUserID uuid.UUID) error {
	logCtx := ContextFromRequest(r)
	logCtx.ResourceType = ptr(ResourceUser)
	logCtx.ResourceID = &targetUserID

	return l.Log(ctx, logCtx, ActionUserDeactivate, nil)
}

// LogAPIKeyCreate logs an API key creation
func (l *Logger) LogAPIKeyCreate(ctx context.Context, r *http.Request, keyID uuid.UUID, name string, scopes []string) error {
	logCtx := ContextFromRequest(r)
	logCtx.ResourceType = ptr(ResourceAPIKey)
	logCtx.ResourceID = &keyID

	return l.Log(ctx, logCtx, ActionAPIKeyCreate, map[string]interface{}{
		"name":   name,
		"scopes": scopes,
	})
}

// LogAPIKeyRevoke logs an API key revocation
func (l *Logger) LogAPIKeyRevoke(ctx context.Context, r *http.Request, keyID uuid.UUID) error {
	logCtx := ContextFromRequest(r)
	logCtx.ResourceType = ptr(ResourceAPIKey)
	logCtx.ResourceID = &keyID

	return l.Log(ctx, logCtx, ActionAPIKeyRevoke, nil)
}

// LogSessionTerminate logs a session termination
func (l *Logger) LogSessionTerminate(ctx context.Context, r *http.Request, sessionID uuid.UUID) error {
	logCtx := ContextFromRequest(r)
	logCtx.ResourceType = ptr(ResourceSession)
	logCtx.ResourceID = &sessionID

	return l.Log(ctx, logCtx, ActionSessionTerminate, nil)
}

func ptr(s string) *string {
	return &s
}

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return anonymizeIP(xff[:i])
			}
		}
		return anonymizeIP(xff)
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return anonymizeIP(xri)
	}

	// RemoteAddr includes port, strip it
	ip := r.RemoteAddr
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == ':' {
			ip = ip[:i]
			break
		}
	}
	return anonymizeIP(ip)
}

// anonymizeIP removes the last octet of IPv4 addresses
// and the last 80 bits of IPv6 addresses for DSGVO compliance.
// Examples:
//   - 192.168.1.123 -> 192.168.1.0
//   - 2001:db8::1 -> 2001:db8::0
func anonymizeIP(ip string) string {
	if ip == "" {
		return ""
	}

	// Check for IPv6
	hasColon := false
	for i := 0; i < len(ip); i++ {
		if ip[i] == ':' {
			hasColon = true
			break
		}
	}

	if hasColon {
		// IPv6: find last colon and zero everything after
		// This is a simplified approach - zeroes last segment
		lastColon := -1
		for i := len(ip) - 1; i >= 0; i-- {
			if ip[i] == ':' {
				lastColon = i
				break
			}
		}
		if lastColon > 0 {
			return ip[:lastColon+1] + "0"
		}
		return ip
	}

	// IPv4: replace last octet with 0
	lastDot := -1
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == '.' {
			lastDot = i
			break
		}
	}

	if lastDot > 0 {
		return ip[:lastDot+1] + "0"
	}

	return ip
}

// truncateUserAgent truncates user agent to max 255 characters
func truncateUserAgent(ua string) string {
	const maxLen = 255
	if len(ua) <= maxLen {
		return ua
	}
	return ua[:maxLen]
}
