package account

import (
	"regexp"
	"strings"
)

// Sensitive field patterns to redact from logs
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)"pin"\s*:\s*"[^"]*"`),
	regexp.MustCompile(`(?i)"password"\s*:\s*"[^"]*"`),
	regexp.MustCompile(`(?i)"secret"\s*:\s*"[^"]*"`),
	regexp.MustCompile(`(?i)"certificate_password"\s*:\s*"[^"]*"`),
	regexp.MustCompile(`(?i)"cert_password"\s*:\s*"[^"]*"`),
	regexp.MustCompile(`(?i)"api_key"\s*:\s*"[^"]*"`),
	regexp.MustCompile(`(?i)"token"\s*:\s*"[^"]*"`),
	regexp.MustCompile(`(?i)"credentials"\s*:\s*"[^"]*"`),
}

// Sensitive field names to mask
var sensitiveFields = []string{
	"pin",
	"password",
	"secret",
	"certificate_password",
	"cert_password",
	"api_key",
	"token",
}

// RedactCredentials removes sensitive data from a string
func RedactCredentials(s string) string {
	result := s

	for _, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Find the colon position
			colonIdx := strings.Index(match, ":")
			if colonIdx == -1 {
				return match
			}
			// Keep the field name, redact the value
			fieldPart := match[:colonIdx+1]
			return fieldPart + `"[REDACTED]"`
		})
	}

	return result
}

// RedactMap removes sensitive values from a map
func RedactMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range m {
		if isSensitiveField(k) {
			result[k] = "[REDACTED]"
			continue
		}

		// Recursively redact nested maps
		if nested, ok := v.(map[string]interface{}); ok {
			result[k] = RedactMap(nested)
			continue
		}

		result[k] = v
	}

	return result
}

// RedactStruct creates a copy with sensitive fields masked
func RedactStruct(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return RedactMap(val)
	case map[string]string:
		result := make(map[string]string)
		for k, v := range val {
			if isSensitiveField(k) {
				result[k] = "[REDACTED]"
			} else {
				result[k] = v
			}
		}
		return result
	default:
		return v
	}
}

func isSensitiveField(name string) bool {
	lower := strings.ToLower(name)
	for _, field := range sensitiveFields {
		if lower == field || strings.Contains(lower, field) {
			return true
		}
	}
	return false
}

// MaskString masks all but the last n characters of a string
func MaskString(s string, showLast int) string {
	if len(s) <= showLast {
		return strings.Repeat("*", len(s))
	}
	masked := strings.Repeat("*", len(s)-showLast)
	return masked + s[len(s)-showLast:]
}

// MaskTID masks a TID showing only last 4 digits
func MaskTID(tid string) string {
	return MaskString(tid, 4)
}

// MaskBenID masks a BenID showing only last 3 characters
func MaskBenID(benID string) string {
	return MaskString(benID, 3)
}

// LogSafeCredentials returns a map of credentials safe for logging
func LogSafeCredentials(accountType string, creds interface{}) map[string]string {
	result := make(map[string]string)
	result["type"] = accountType

	switch accountType {
	case AccountTypeFinanzOnline:
		// Only show masked identifiers, never PIN
		result["has_tid"] = "true"
		result["has_ben_id"] = "true"
		result["has_pin"] = "true"

	case AccountTypeELDA:
		result["has_dienstgeber_nr"] = "true"
		result["has_pin"] = "true"
		result["has_certificate"] = "true"

	case AccountTypeFirmenbuch:
		result["has_username"] = "true"
		result["has_password"] = "true"
	}

	return result
}
