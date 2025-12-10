package config

import (
	"os"
	"time"
)

// SignatureConfig holds digital signature configuration
type SignatureConfig struct {
	// A-Trust API
	ATrustAPIURL    string
	ATrustAPIKey    string
	ATrustSandbox   bool
	ATrustTimeout   time.Duration
	ATrustRetryMax  int

	// ID Austria OIDC
	IDAustriaIssuer       string
	IDAustriaClientID     string
	IDAustriaClientSecret string
	IDAustriaRedirectURL  string
	IDAustriaScopes       []string

	// Signature Settings
	SignatureLinkExpiryDays    int
	SignatureReminderDays      int
	SignatureMaxBatchSize      int
	SignatureVisualEnabled     bool
	SignatureTimestampProvider string

	// PDF Settings
	PDFMaxSizeBytes int64 // Maximum PDF size for signing (default 100MB)

	// Callback URLs
	SigningCallbackURL    string
	PortalSigningBasePath string

	// Cost tracking
	SignatureCostCents int // Per-signature cost in cents for tracking
}

// LoadSignatureConfig loads signature configuration from environment variables
func LoadSignatureConfig() *SignatureConfig {
	return &SignatureConfig{
		// A-Trust API
		ATrustAPIURL:   getEnv("ATRUST_API_URL", "https://api.a-trust.at/v1"),
		ATrustAPIKey:   os.Getenv("ATRUST_API_KEY"),
		ATrustSandbox:  getEnvBool("ATRUST_SANDBOX", true),
		ATrustTimeout:  getEnvDuration("ATRUST_TIMEOUT", 30*time.Second),
		ATrustRetryMax: getEnvInt("ATRUST_RETRY_MAX", 3),

		// ID Austria OIDC
		IDAustriaIssuer:       getEnv("IDAUSTRIA_ISSUER", "https://eid.gv.at"),
		IDAustriaClientID:     os.Getenv("IDAUSTRIA_CLIENT_ID"),
		IDAustriaClientSecret: os.Getenv("IDAUSTRIA_CLIENT_SECRET"),
		IDAustriaRedirectURL:  getEnv("IDAUSTRIA_REDIRECT_URL", "http://localhost:8080/api/v1/sign/callback"),
		IDAustriaScopes:       getEnvList("IDAUSTRIA_SCOPES", []string{"openid", "profile", "signature"}),

		// Signature Settings
		SignatureLinkExpiryDays:    getEnvInt("SIGNATURE_LINK_EXPIRY_DAYS", 14),
		SignatureReminderDays:      getEnvInt("SIGNATURE_REMINDER_DAYS", 7),
		SignatureMaxBatchSize:      getEnvInt("SIGNATURE_MAX_BATCH_SIZE", 100),
		SignatureVisualEnabled:     getEnvBool("SIGNATURE_VISUAL_ENABLED", true),
		SignatureTimestampProvider: getEnv("SIGNATURE_TIMESTAMP_PROVIDER", "https://timestamp.a-trust.at"),

		// PDF Settings (default 100MB)
		PDFMaxSizeBytes: getEnvInt64("SIGNATURE_PDF_MAX_SIZE", 100*1024*1024),

		// Callback URLs
		SigningCallbackURL:    getEnv("SIGNING_CALLBACK_URL", "http://localhost:8080/api/v1/sign"),
		PortalSigningBasePath: getEnv("PORTAL_SIGNING_BASE_PATH", "http://localhost:3001/sign"),

		// Cost tracking (example: 30 cents per signature)
		SignatureCostCents: getEnvInt("SIGNATURE_COST_CENTS", 30),
	}
}

// IsATrustConfigured returns true if A-Trust API is configured
func (c *SignatureConfig) IsATrustConfigured() bool {
	return c.ATrustAPIKey != ""
}

// IsIDAustriaConfigured returns true if ID Austria OIDC is configured
func (c *SignatureConfig) IsIDAustriaConfigured() bool {
	return c.IDAustriaClientID != "" && c.IDAustriaClientSecret != ""
}

// IsFullyConfigured returns true if both A-Trust and ID Austria are configured
func (c *SignatureConfig) IsFullyConfigured() bool {
	return c.IsATrustConfigured() && c.IsIDAustriaConfigured()
}

// SigningLinkExpiry returns the duration for signing link expiry
func (c *SignatureConfig) SigningLinkExpiry() time.Duration {
	return time.Duration(c.SignatureLinkExpiryDays) * 24 * time.Hour
}

// ReminderThreshold returns the duration before expiry when reminders should be sent
func (c *SignatureConfig) ReminderThreshold() time.Duration {
	return time.Duration(c.SignatureReminderDays) * 24 * time.Hour
}
