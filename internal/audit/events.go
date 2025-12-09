package audit

// Security Event Types for audit logging.
// These constants follow the pattern: category.action
//
// Format: {category}.{action}
// Categories: auth, credential, document, data, permission, ai
//
// All events are logged to the audit_log table without PII.
// IP addresses are anonymized (last octet zeroed).
// User agents are truncated to 255 characters.

// Authentication Events
const (
	// EventLogin is logged on successful login
	EventLogin = "auth.login"
	// EventLoginFailed is logged on failed login attempt
	EventLoginFailed = "auth.login_failed"
	// EventLogout is logged when user logs out
	EventLogout = "auth.logout"
	// EventTokenRefresh is logged when access token is refreshed
	EventTokenRefresh = "auth.token_refresh"
	// Event2FAEnabled is logged when user enables 2FA
	Event2FAEnabled = "auth.2fa_enabled"
	// Event2FADisabled is logged when user disables 2FA
	Event2FADisabled = "auth.2fa_disabled"
	// EventPasswordChange is logged when user changes password
	EventPasswordChange = "auth.password_change"
	// EventPasswordReset is logged when password is reset
	EventPasswordReset = "auth.password_reset"
	// EventRecoveryCodeUsed is logged when a recovery code is used for 2FA
	EventRecoveryCodeUsed = "auth.recovery_code_used"
	// EventSessionCreated is logged when a new session is created
	EventSessionCreated = "auth.session_created"
	// EventSessionTerminated is logged when a session is terminated
	EventSessionTerminated = "auth.session_terminated"
)

// Credential Events
const (
	// EventCredentialCreated is logged when FO/ELDA credentials are stored
	EventCredentialCreated = "credential.created"
	// EventCredentialUpdated is logged when credentials are modified
	EventCredentialUpdated = "credential.updated"
	// EventCredentialDeleted is logged when credentials are removed
	EventCredentialDeleted = "credential.deleted"
	// EventCredentialUsed is logged when credentials are used for API call
	EventCredentialUsed = "credential.used"
	// EventCredentialValidated is logged when credentials are verified
	EventCredentialValidated = "credential.validated"
	// EventCredentialFailed is logged when credential use fails
	EventCredentialFailed = "credential.failed"
)

// Document Events
const (
	// EventDocumentCreated is logged when a document is uploaded
	EventDocumentCreated = "document.created"
	// EventDocumentAccessed is logged when a document is viewed
	EventDocumentAccessed = "document.accessed"
	// EventDocumentDownloaded is logged when a document is downloaded
	EventDocumentDownloaded = "document.downloaded"
	// EventDocumentDeleted is logged when a document is removed
	EventDocumentDeleted = "document.deleted"
	// EventDocumentAnalyzed is logged when AI analysis is performed
	EventDocumentAnalyzed = "document.analyzed"
	// EventDocumentClassified is logged when document is classified
	EventDocumentClassified = "document.classified"
)

// Data/DSGVO Events
const (
	// EventExportRequested is logged when data export is requested (Art. 20)
	EventExportRequested = "data.export_requested"
	// EventExportCompleted is logged when export file is ready
	EventExportCompleted = "data.export_completed"
	// EventExportDownloaded is logged when export is downloaded
	EventExportDownloaded = "data.export_downloaded"
	// EventDeletionRequested is logged when tenant deletion is requested (Art. 17)
	EventDeletionRequested = "data.deletion_requested"
	// EventDeletionCancelled is logged when deletion is cancelled
	EventDeletionCancelled = "data.deletion_cancelled"
	// EventDeletionExecuted is logged when tenant data is deleted
	EventDeletionExecuted = "data.deletion_executed"
)

// Permission Events
const (
	// EventUserInvited is logged when user is invited to tenant
	EventUserInvited = "permission.user_invited"
	// EventUserRemoved is logged when user is removed from tenant
	EventUserRemoved = "permission.user_removed"
	// EventRoleChanged is logged when user role is changed
	EventRoleChanged = "permission.role_changed"
	// EventAPIKeyCreated is logged when API key is created
	EventAPIKeyCreated = "permission.api_key_created"
	// EventAPIKeyRevoked is logged when API key is revoked
	EventAPIKeyRevoked = "permission.api_key_revoked"
)

// AI Gateway Events
const (
	// EventAIRequestSent is logged when request is sent to AI
	EventAIRequestSent = "ai.request_sent"
	// EventAIResponseReceived is logged when AI response is received
	EventAIResponseReceived = "ai.response_received"
	// EventAIValidationFailed is logged when AI output fails validation
	EventAIValidationFailed = "ai.validation_failed"
	// EventAIInputSanitized is logged when dangerous patterns are filtered
	EventAIInputSanitized = "ai.input_sanitized"
	// EventAISuspiciousContent is logged when suspicious content detected
	EventAISuspiciousContent = "ai.suspicious_content"
)

// Security Events
const (
	// EventCrossTenantAttempt is logged when cross-tenant access is attempted
	EventCrossTenantAttempt = "security.cross_tenant_attempt"
	// EventRateLimited is logged when request is rate limited
	EventRateLimited = "security.rate_limited"
	// EventKeyRotationStarted is logged when key rotation begins
	EventKeyRotationStarted = "security.key_rotation_started"
	// EventKeyRotationCompleted is logged when key rotation completes
	EventKeyRotationCompleted = "security.key_rotation_completed"
	// EventKeyRotationFailed is logged when key rotation fails
	EventKeyRotationFailed = "security.key_rotation_failed"
)

// Resource Types for categorizing audit log entries
const (
	ResourceTypeUser       = "user"
	ResourceTypeTenant     = "tenant"
	ResourceTypeAccount    = "account"
	ResourceTypeDocument   = "document"
	ResourceTypeSession    = "session"
	ResourceTypeAPIKey     = "api_key"
	ResourceTypeInvitation = "invitation"
	ResourceTypeExport     = "export"
	ResourceTypeDeletion   = "deletion"
)

// EventCategory returns the category of an event (e.g., "auth" from "auth.login")
func EventCategory(event string) string {
	for i := 0; i < len(event); i++ {
		if event[i] == '.' {
			return event[:i]
		}
	}
	return event
}

// IsCriticalEvent returns true if the event is a critical security event
// that requires immediate alerting.
func IsCriticalEvent(event string) bool {
	switch event {
	case EventCrossTenantAttempt,
		EventAISuspiciousContent,
		EventCredentialFailed,
		EventKeyRotationFailed:
		return true
	default:
		return false
	}
}

// IsAuthEvent returns true if the event is an authentication event
func IsAuthEvent(event string) bool {
	return EventCategory(event) == "auth"
}

// RequiresUserID returns true if the event typically requires a user_id
func RequiresUserID(event string) bool {
	// System events may not have user_id
	switch event {
	case EventKeyRotationStarted,
		EventKeyRotationCompleted,
		EventKeyRotationFailed,
		EventDeletionExecuted:
		return false
	default:
		return true
	}
}
