package config

import (
	"os"
	"strconv"
	"time"
)

// PortalConfig holds client portal configuration
type PortalConfig struct {
	// Domain
	Domain      string
	CORSOrigins []string

	// Invitations
	InvitationExpiryHours int

	// Sessions
	SessionTimeout time.Duration

	// Uploads
	MaxUploadSize      int64 // bytes
	AllowedUploadTypes []string
	UploadPath         string

	// WebSocket
	WebSocketPath string
	WebSocketPingInterval time.Duration
}

// LoadPortalConfig loads portal configuration from environment variables
func LoadPortalConfig() *PortalConfig {
	return &PortalConfig{
		// Domain
		Domain:      getEnv("PORTAL_DOMAIN", "localhost:3001"),
		CORSOrigins: getEnvList("PORTAL_CORS_ORIGINS", []string{"http://localhost:3001"}),

		// Invitations (24 hours default)
		InvitationExpiryHours: getEnvInt("PORTAL_INVITATION_EXPIRY_HOURS", 24),

		// Sessions (30 minutes default)
		SessionTimeout: getEnvDuration("PORTAL_SESSION_TIMEOUT", 30*time.Minute),

		// Uploads (25MB default)
		MaxUploadSize:      getEnvInt64("PORTAL_MAX_UPLOAD_SIZE", 25*1024*1024),
		AllowedUploadTypes: getEnvList("PORTAL_ALLOWED_UPLOAD_TYPES", []string{"pdf", "jpg", "jpeg", "png", "doc", "docx", "xls", "xlsx"}),
		UploadPath:         getEnv("PORTAL_UPLOAD_PATH", "./data/client-uploads"),

		// WebSocket
		WebSocketPath:        getEnv("PORTAL_WS_PATH", "/api/v1/ws/messages"),
		WebSocketPingInterval: getEnvDuration("PORTAL_WS_PING_INTERVAL", 30*time.Second),
	}
}

// AllowedMimeTypes returns a map of allowed mime types for uploads
func (c *PortalConfig) AllowedMimeTypes() map[string]bool {
	return map[string]bool{
		"application/pdf":                                                       true,
		"image/jpeg":                                                             true,
		"image/png":                                                              true,
		"application/msword":                                                     true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel":                                               true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":      true,
	}
}

// IsAllowedExtension checks if a file extension is allowed for upload
func (c *PortalConfig) IsAllowedExtension(ext string) bool {
	// Remove leading dot if present
	if len(ext) > 0 && ext[0] == '.' {
		ext = ext[1:]
	}

	for _, allowed := range c.AllowedUploadTypes {
		if toLowerCase(ext) == toLowerCase(allowed) {
			return true
		}
	}
	return false
}

func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}
