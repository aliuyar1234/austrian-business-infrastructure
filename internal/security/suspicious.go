package security

import (
	"regexp"
	"strings"
)

// SuspiciousOutputDetector detects potentially leaked credentials or sensitive data in AI output
type SuspiciousOutputDetector struct {
	credentialPatterns []*regexp.Regexp
	sensitiveKeywords  []string
}

// NewSuspiciousOutputDetector creates a new suspicious output detector
func NewSuspiciousOutputDetector() *SuspiciousOutputDetector {
	return &SuspiciousOutputDetector{
		credentialPatterns: defaultCredentialPatterns(),
		sensitiveKeywords:  defaultSensitiveKeywords(),
	}
}

// defaultCredentialPatterns returns patterns that might indicate credential exposure
func defaultCredentialPatterns() []*regexp.Regexp {
	patterns := []string{
		// API keys and tokens
		`(?i)api[_-]?key\s*[:=]\s*\S{20,}`,
		`(?i)api[_-]?secret\s*[:=]\s*\S{20,}`,
		`(?i)access[_-]?token\s*[:=]\s*\S{20,}`,
		`(?i)secret[_-]?key\s*[:=]\s*\S{20,}`,
		`(?i)bearer\s+[a-zA-Z0-9\-_.]+`,

		// AWS credentials
		`(?i)aws[_-]?access[_-]?key[_-]?id\s*[:=]\s*AKIA[A-Z0-9]{16}`,
		`(?i)aws[_-]?secret[_-]?access[_-]?key\s*[:=]\s*\S{40}`,

		// Database connection strings
		`(?i)postgres://[^:]+:[^@]+@`,
		`(?i)mysql://[^:]+:[^@]+@`,
		`(?i)mongodb://[^:]+:[^@]+@`,
		`(?i)redis://:[^@]+@`,

		// Private keys
		`-----BEGIN\s+(RSA\s+)?PRIVATE KEY-----`,
		`-----BEGIN\s+ENCRYPTED\s+PRIVATE KEY-----`,
		`-----BEGIN\s+EC\s+PRIVATE KEY-----`,

		// JWT tokens (might contain sensitive claims)
		`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`,

		// FinanzOnline specific
		`(?i)teilnehmer[_-]?id\s*[:=]\s*\d{6,}`,
		`(?i)benutzer[_-]?id\s*[:=]\s*\S{4,}`,
		`(?i)fo[_-]?pin\s*[:=]\s*\S{4,}`,
		`(?i)elda[_-]?pin\s*[:=]\s*\S{4,}`,

		// Austrian tax IDs
		`\b\d{2}-\d{3}/\d{4}\b`, // Steuernummer format

		// Credit card numbers (basic pattern)
		`\b(?:\d{4}[- ]?){3}\d{4}\b`,

		// SSN-like patterns (Austrian Sozialversicherungsnummer)
		`\b\d{4}\s?\d{2}\s?\d{2}\s?\d{2}\b`,
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if r, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, r)
		}
	}
	return compiled
}

// defaultSensitiveKeywords returns keywords that might indicate sensitive data
func defaultSensitiveKeywords() []string {
	return []string{
		// Credential-related
		"password:",
		"passwort:",
		"pin:",
		"geheimzahl:",
		"secret:",
		"credential:",
		"api key:",
		"api-key:",
		"apikey:",
		"private key",
		"encryption key",

		// Authentication
		"session token:",
		"access token:",
		"refresh token:",
		"bearer token:",
		"auth token:",

		// Personal data (DSGVO relevant)
		"sozialversicherungsnummer:",
		"svnr:",
		"geburtsdatum:",
		"iban:",
		"bic:",
		"kontonummer:",
		"kreditkarte:",
		"kartennummer:",
	}
}

// SuspiciousResult contains the result of suspicious content detection
type SuspiciousResult struct {
	IsSuspicious     bool
	SuspiciousTypes  []string
	RedactedContent  string
	SuspiciousCount  int
}

// Check checks AI output for suspicious content
func (d *SuspiciousOutputDetector) Check(output string) *SuspiciousResult {
	result := &SuspiciousResult{
		SuspiciousTypes: make([]string, 0),
	}

	redacted := output

	// Check credential patterns
	for _, pattern := range d.credentialPatterns {
		if matches := pattern.FindAllString(output, -1); len(matches) > 0 {
			result.IsSuspicious = true
			result.SuspiciousCount += len(matches)
			result.SuspiciousTypes = append(result.SuspiciousTypes, "credential_pattern")
			// Redact matches
			redacted = pattern.ReplaceAllString(redacted, "[REDACTED]")
		}
	}

	// Check sensitive keywords
	lowerOutput := strings.ToLower(output)
	for _, keyword := range d.sensitiveKeywords {
		if strings.Contains(lowerOutput, keyword) {
			result.IsSuspicious = true
			result.SuspiciousCount++
			result.SuspiciousTypes = append(result.SuspiciousTypes, "sensitive_keyword")
			// Redact keyword and following content
			idx := strings.Index(strings.ToLower(redacted), keyword)
			if idx >= 0 {
				// Find end of line or next space after value
				end := idx + len(keyword)
				for end < len(redacted) && redacted[end] != '\n' && redacted[end] != ' ' {
					end++
				}
				redacted = redacted[:idx] + "[REDACTED]" + redacted[end:]
			}
		}
	}

	result.RedactedContent = redacted

	// Deduplicate suspicious types
	seen := make(map[string]bool)
	unique := make([]string, 0)
	for _, t := range result.SuspiciousTypes {
		if !seen[t] {
			seen[t] = true
			unique = append(unique, t)
		}
	}
	result.SuspiciousTypes = unique

	return result
}

// IsSafe returns true if output appears safe (no suspicious content)
func (d *SuspiciousOutputDetector) IsSafe(output string) bool {
	result := d.Check(output)
	return !result.IsSuspicious
}

// Redact removes suspicious content from output
func (d *SuspiciousOutputDetector) Redact(output string) string {
	result := d.Check(output)
	return result.RedactedContent
}
