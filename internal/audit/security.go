package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Additional Security Event Types (extending events.go)
const (
	// Token revocation events
	EventTokenRevoked         = "auth.token_revoked"
	EventUserTokensRevoked    = "auth.user_tokens_revoked"
	EventTenantTokensRevoked  = "auth.tenant_tokens_revoked"

	// Brute force detection
	EventBruteForceDetected   = "security.brute_force_detected"
	EventBruteForceBlocked    = "security.brute_force_blocked"

	// Session security
	EventSessionHijackSuspect = "security.session_hijack_suspect"
	EventSessionIPChanged     = "security.session_ip_changed"

	// Data access patterns
	EventSensitiveDataAccess  = "security.sensitive_data_access"
	EventBulkDataAccess       = "security.bulk_data_access"
	EventUnusualAccessPattern = "security.unusual_access_pattern"

	// Admin security events
	EventAdminAction          = "admin.action"
	EventConfigChanged        = "admin.config_changed"
	EventUserSuspended        = "admin.user_suspended"
	EventUserReactivated      = "admin.user_reactivated"
)

// Severity levels for security events
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Outcome of the security event
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomeBlocked Outcome = "blocked"
)

// SecurityEvent represents an enhanced security audit event
type SecurityEvent struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	EventType string            `json:"event_type"`
	Severity  Severity          `json:"severity"`
	Outcome   Outcome           `json:"outcome"`

	// Actor information
	TenantID  string `json:"tenant_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`

	// Request context
	RequestID string `json:"request_id,omitempty"`
	Method    string `json:"method,omitempty"`
	Path      string `json:"path,omitempty"`

	// Event details
	Resource   string            `json:"resource,omitempty"`
	Action     string            `json:"action,omitempty"`
	Message    string            `json:"message,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// SecurityAuditor provides enhanced security event logging
type SecurityAuditor struct {
	logger *slog.Logger
	store  SecurityEventStore
}

// SecurityEventStore persists security events (optional)
type SecurityEventStore interface {
	Store(ctx context.Context, event *SecurityEvent) error
}

// NewSecurityAuditor creates a new security auditor
func NewSecurityAuditor(logger *slog.Logger, store SecurityEventStore) *SecurityAuditor {
	if logger == nil {
		logger = slog.Default()
	}
	return &SecurityAuditor{
		logger: logger.With("component", "security_audit"),
		store:  store,
	}
}

// Log logs a security event
func (a *SecurityAuditor) Log(ctx context.Context, event *SecurityEvent) {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Determine log level based on severity
	level := slog.LevelInfo
	switch event.Severity {
	case SeverityLow:
		level = slog.LevelInfo
	case SeverityMedium:
		level = slog.LevelWarn
	case SeverityHigh, SeverityCritical:
		level = slog.LevelError
	}

	// Build log attributes
	attrs := []any{
		"event_id", event.ID,
		"event_type", event.EventType,
		"severity", event.Severity,
		"outcome", event.Outcome,
	}

	if event.TenantID != "" {
		attrs = append(attrs, "tenant_id", event.TenantID)
	}
	if event.UserID != "" {
		attrs = append(attrs, "user_id", event.UserID)
	}
	if event.UserEmail != "" {
		attrs = append(attrs, "user_email", event.UserEmail)
	}
	if event.IPAddress != "" {
		attrs = append(attrs, "ip_address", event.IPAddress)
	}
	if event.RequestID != "" {
		attrs = append(attrs, "request_id", event.RequestID)
	}
	if event.Resource != "" {
		attrs = append(attrs, "resource", event.Resource)
	}
	if event.Action != "" {
		attrs = append(attrs, "action", event.Action)
	}
	if event.Message != "" {
		attrs = append(attrs, "message", event.Message)
	}
	if len(event.Metadata) > 0 {
		metaJSON, _ := json.Marshal(event.Metadata)
		attrs = append(attrs, "metadata", string(metaJSON))
	}

	a.logger.Log(ctx, level, "security_event", attrs...)

	// Persist to store if configured
	if a.store != nil {
		if err := a.store.Store(ctx, event); err != nil {
			a.logger.Error("failed to persist security event", "error", err, "event_id", event.ID)
		}
	}
}

// Helper methods for common security events

// LogLoginSuccess logs a successful login
func (a *SecurityAuditor) LogLoginSuccess(ctx context.Context, tenantID, userID, email, ip, userAgent, requestID string) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventLogin,
		Severity:  SeverityInfo,
		Outcome:   OutcomeSuccess,
		TenantID:  tenantID,
		UserID:    userID,
		UserEmail: email,
		IPAddress: ip,
		UserAgent: userAgent,
		RequestID: requestID,
		Message:   "User logged in successfully",
	})
}

// LogLoginFailed logs a failed login attempt
func (a *SecurityAuditor) LogLoginFailed(ctx context.Context, email, ip, userAgent, requestID, reason string) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventLoginFailed,
		Severity:  SeverityMedium,
		Outcome:   OutcomeFailure,
		UserEmail: email,
		IPAddress: ip,
		UserAgent: userAgent,
		RequestID: requestID,
		Message:   "Login attempt failed: " + reason,
		Metadata:  map[string]string{"reason": reason},
	})
}

// LogLogout logs a logout event
func (a *SecurityAuditor) LogLogout(ctx context.Context, tenantID, userID, ip, requestID string) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventLogout,
		Severity:  SeverityInfo,
		Outcome:   OutcomeSuccess,
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: ip,
		RequestID: requestID,
		Message:   "User logged out",
	})
}

// LogTokenRevoked logs a token revocation
func (a *SecurityAuditor) LogTokenRevoked(ctx context.Context, tenantID, userID, tokenID, reason, requestID string) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventTokenRevoked,
		Severity:  SeverityMedium,
		Outcome:   OutcomeSuccess,
		TenantID:  tenantID,
		UserID:    userID,
		RequestID: requestID,
		Message:   "Token revoked: " + reason,
		Metadata:  map[string]string{"token_id": tokenID, "reason": reason},
	})
}

// LogAccessDenied logs an access denied event
func (a *SecurityAuditor) LogAccessDenied(ctx context.Context, tenantID, userID, ip, resource, action, requestID string) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventCrossTenantAttempt,
		Severity:  SeverityMedium,
		Outcome:   OutcomeBlocked,
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: ip,
		Resource:  resource,
		Action:    action,
		RequestID: requestID,
		Message:   "Access denied to resource",
	})
}

// LogRateLimited logs a rate limit event
func (a *SecurityAuditor) LogRateLimited(ctx context.Context, ip, endpoint, requestID string) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventRateLimited,
		Severity:  SeverityMedium,
		Outcome:   OutcomeBlocked,
		IPAddress: ip,
		Path:      endpoint,
		RequestID: requestID,
		Message:   "Rate limit exceeded",
	})
}

// LogBruteForce logs suspected brute force attempt
func (a *SecurityAuditor) LogBruteForce(ctx context.Context, ip, target, requestID string, attempts int) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventBruteForceDetected,
		Severity:  SeverityHigh,
		Outcome:   OutcomeBlocked,
		IPAddress: ip,
		RequestID: requestID,
		Message:   "Suspected brute force attack",
		Metadata: map[string]string{
			"target":   target,
			"attempts": string(rune(attempts)),
		},
	})
}

// LogPasswordChanged logs a password change
func (a *SecurityAuditor) LogPasswordChanged(ctx context.Context, tenantID, userID, ip, requestID string) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventPasswordChange,
		Severity:  SeverityMedium,
		Outcome:   OutcomeSuccess,
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: ip,
		RequestID: requestID,
		Message:   "Password changed",
	})
}

// Log2FAEnabled logs 2FA enablement
func (a *SecurityAuditor) Log2FAEnabled(ctx context.Context, tenantID, userID, ip, requestID string) {
	a.Log(ctx, &SecurityEvent{
		EventType: Event2FAEnabled,
		Severity:  SeverityInfo,
		Outcome:   OutcomeSuccess,
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: ip,
		RequestID: requestID,
		Message:   "Two-factor authentication enabled",
	})
}

// LogSensitiveDataAccess logs sensitive data access
func (a *SecurityAuditor) LogSensitiveDataAccess(ctx context.Context, tenantID, userID, resource, action, requestID string) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventSensitiveDataAccess,
		Severity:  SeverityInfo,
		Outcome:   OutcomeSuccess,
		TenantID:  tenantID,
		UserID:    userID,
		Resource:  resource,
		Action:    action,
		RequestID: requestID,
		Message:   "Sensitive data accessed",
	})
}

// LogAdminAction logs administrative actions
func (a *SecurityAuditor) LogAdminAction(ctx context.Context, tenantID, userID, action, target, requestID string) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventAdminAction,
		Severity:  SeverityMedium,
		Outcome:   OutcomeSuccess,
		TenantID:  tenantID,
		UserID:    userID,
		Action:    action,
		Resource:  target,
		RequestID: requestID,
		Message:   "Administrative action performed",
	})
}

// LogSessionHijackSuspect logs suspected session hijacking
func (a *SecurityAuditor) LogSessionHijackSuspect(ctx context.Context, tenantID, userID, originalIP, newIP, requestID string) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventSessionHijackSuspect,
		Severity:  SeverityHigh,
		Outcome:   OutcomeBlocked,
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: newIP,
		RequestID: requestID,
		Message:   "Suspected session hijacking",
		Metadata: map[string]string{
			"original_ip": originalIP,
			"new_ip":      newIP,
		},
	})
}

// LogUserTokensRevoked logs when all tokens for a user are revoked
func (a *SecurityAuditor) LogUserTokensRevoked(ctx context.Context, tenantID, userID, adminUserID, reason, requestID string) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventUserTokensRevoked,
		Severity:  SeverityMedium,
		Outcome:   OutcomeSuccess,
		TenantID:  tenantID,
		UserID:    userID,
		RequestID: requestID,
		Message:   "All tokens for user revoked",
		Metadata: map[string]string{
			"admin_user_id": adminUserID,
			"reason":        reason,
		},
	})
}

// LogTenantTokensRevoked logs when all tokens for a tenant are revoked
func (a *SecurityAuditor) LogTenantTokensRevoked(ctx context.Context, tenantID, adminUserID, reason, requestID string) {
	a.Log(ctx, &SecurityEvent{
		EventType: EventTenantTokensRevoked,
		Severity:  SeverityHigh,
		Outcome:   OutcomeSuccess,
		TenantID:  tenantID,
		RequestID: requestID,
		Message:   "All tokens for tenant revoked",
		Metadata: map[string]string{
			"admin_user_id": adminUserID,
			"reason":        reason,
		},
	})
}
