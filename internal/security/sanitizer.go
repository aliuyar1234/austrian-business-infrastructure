package security

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Default limits for AI input sanitization
const (
	DefaultMaxInputLength = 50 * 1024 // 50KB default
	MaxInputLength        = 100 * 1024 // 100KB hard limit
	FilteredMarker        = "[FILTERED]"
)

// Sanitizer sanitizes AI input to prevent prompt injection and other attacks
type Sanitizer struct {
	maxLength   int
	patterns    []*regexp.Regexp
	keywords    []string
}

// NewSanitizer creates a new input sanitizer
func NewSanitizer(maxLength int) *Sanitizer {
	if maxLength <= 0 || maxLength > MaxInputLength {
		maxLength = DefaultMaxInputLength
	}

	return &Sanitizer{
		maxLength:   maxLength,
		patterns:    defaultDangerousPatterns(),
		keywords:    defaultDangerousKeywords(),
	}
}

// defaultDangerousPatterns returns compiled regexes for dangerous input patterns
func defaultDangerousPatterns() []*regexp.Regexp {
	patterns := []string{
		// Prompt injection attempts
		`(?i)ignore\s+(previous|all|above|prior)\s+(instructions?|commands?)`,
		`(?i)forget\s+(everything|all|previous)`,
		`(?i)you\s+are\s+(now|a)\s+(different|new|the)`,
		`(?i)act\s+as\s+(if|a|an|the)`,
		`(?i)pretend\s+(to\s+be|you\s+are)`,
		`(?i)roleplay\s+as`,
		`(?i)system\s*:\s*`,
		`(?i)assistant\s*:\s*`,
		`(?i)user\s*:\s*`,
		`(?i)human\s*:\s*`,
		`(?i)\[system\]`,
		`(?i)\[assistant\]`,
		`(?i)<\|endoftext\|>`,
		`(?i)<\|im_start\|>`,
		`(?i)<\|im_end\|>`,
		// Common injection delimiters
		`\n{3,}`,                           // Multiple newlines (message boundary)
		`-{5,}`,                            // Long dashes (separator)
		`={5,}`,                            // Long equals (separator)
		`\*{5,}`,                           // Long asterisks
		// Script and code injection (for output that might be rendered)
		`(?i)<script[^>]*>`,
		`(?i)</script>`,
		`(?i)javascript\s*:`,
		`(?i)data\s*:\s*text/html`,
		// SQL/code injection (shouldn't appear in normal doc analysis)
		`(?i);\s*drop\s+`,
		`(?i);\s*delete\s+from`,
		`(?i);\s*truncate\s+`,
		`(?i)union\s+select`,
		`(?i)exec\s*\(`,
		`(?i)eval\s*\(`,
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if r, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, r)
		}
	}
	return compiled
}

// defaultDangerousKeywords returns keywords that might indicate credential exposure
func defaultDangerousKeywords() []string {
	return []string{
		// These keywords in AI prompts might be attempts to extract secrets
		"reveal your instructions",
		"show your system prompt",
		"what are your rules",
		"ignore your safety",
		"bypass your restrictions",
		"admin mode",
		"developer mode",
		"debug mode",
		"maintenance mode",
		"sudo",
		"root access",
		"override security",
	}
}

// SanitizeResult contains the result of sanitization
type SanitizeResult struct {
	Text          string
	WasTruncated  bool
	WasFiltered   bool
	FilteredCount int
	OriginalLen   int
}

// Sanitize sanitizes input text for safe AI processing
func (s *Sanitizer) Sanitize(input string) *SanitizeResult {
	result := &SanitizeResult{
		OriginalLen: len(input),
	}

	// Ensure valid UTF-8
	if !utf8.ValidString(input) {
		input = strings.ToValidUTF8(input, "")
	}

	// Truncate if too long
	if len(input) > s.maxLength {
		// Try to truncate at a word boundary
		input = truncateAtBoundary(input, s.maxLength)
		result.WasTruncated = true
	}

	// Replace dangerous patterns
	text := input
	for _, pattern := range s.patterns {
		matches := pattern.FindAllStringIndex(text, -1)
		if len(matches) > 0 {
			text = pattern.ReplaceAllString(text, FilteredMarker)
			result.WasFiltered = true
			result.FilteredCount += len(matches)
		}
	}

	// Check for dangerous keywords (case-insensitive)
	lowerText := strings.ToLower(text)
	for _, keyword := range s.keywords {
		if strings.Contains(lowerText, keyword) {
			// Replace the keyword
			idx := strings.Index(lowerText, keyword)
			if idx >= 0 {
				text = text[:idx] + FilteredMarker + text[idx+len(keyword):]
				lowerText = strings.ToLower(text)
				result.WasFiltered = true
				result.FilteredCount++
			}
		}
	}

	// Normalize whitespace (but preserve single newlines)
	text = normalizeWhitespace(text)

	result.Text = text
	return result
}

// truncateAtBoundary truncates text at a word or sentence boundary
func truncateAtBoundary(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	// Try to find a good break point
	text = text[:maxLen]

	// Look for last sentence boundary
	lastPeriod := strings.LastIndex(text, ". ")
	if lastPeriod > maxLen/2 {
		return text[:lastPeriod+1]
	}

	// Look for last word boundary
	lastSpace := strings.LastIndex(text, " ")
	if lastSpace > maxLen/2 {
		return text[:lastSpace]
	}

	// Just truncate
	return text
}

// normalizeWhitespace normalizes whitespace while preserving structure
func normalizeWhitespace(text string) string {
	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`[^\S\n]+`)
	text = spaceRegex.ReplaceAllString(text, " ")

	// Replace 3+ newlines with 2
	newlineRegex := regexp.MustCompile(`\n{3,}`)
	text = newlineRegex.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// IsSafeForAI does a quick check if input is safe for AI processing
func (s *Sanitizer) IsSafeForAI(input string) bool {
	if len(input) > s.maxLength {
		return false
	}

	// Check patterns
	for _, pattern := range s.patterns {
		if pattern.MatchString(input) {
			return false
		}
	}

	// Check keywords
	lowerInput := strings.ToLower(input)
	for _, keyword := range s.keywords {
		if strings.Contains(lowerInput, keyword) {
			return false
		}
	}

	return true
}
