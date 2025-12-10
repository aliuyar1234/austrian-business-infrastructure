package config

import (
	"os"
	"strconv"
	"time"
)

// FoerderungConfig holds Förderungsradar configuration
type FoerderungConfig struct {
	// Search Settings
	SearchTimeoutSeconds int           // Max total search time (default: 30)
	SearchTimeout        time.Duration // Parsed duration
	RuleThresholdPercent int           // Min rule score to pass to LLM (default: 50)
	MaxResultsPerSearch  int           // Max results to return (default: 20)

	// LLM Settings
	LLMMaxConcurrent   int     // Max concurrent LLM requests (default: 10)
	LLMTimeoutSeconds  int     // Timeout per LLM request (default: 20)
	LLMTimeout         time.Duration
	LLMCostLimitCents  int     // Max cost per search in cents (default: 10)
	LLMModel           string  // Claude model to use (default: claude-sonnet-4-20250514)
	LLMTemperature     float64 // Temperature for LLM (default: 0.3)
	LLMMaxTokens       int     // Max tokens for LLM response (default: 2000)

	// Scoring Weights
	RuleScoreWeight float64 // Weight of rule score in total (default: 0.4)
	LLMScoreWeight  float64 // Weight of LLM score in total (default: 0.6)

	// Monitoring Settings
	MonitorDefaultThreshold int // Default notification threshold (default: 70)
	MonitorJobCron          string // Cron expression for monitor job (default: "0 6 * * *")
	MonitorDigestCron       string // Cron for digest emails (default: "0 8 * * 1")

	// Expiry Settings
	ExpiryJobCron string // Cron for expiry check (default: "0 1 * * *")
	ExpiryWarningDays int // Days before deadline to warn (default: 7)

	// Cache Settings
	SearchCacheTTLHours int // Cache search results (default: 24)
	FoerderungCacheTTLMinutes int // Cache Förderung data (default: 60)

	// Feature Flags
	LLMFallbackEnabled bool // Fall back to rule-only when LLM fails (default: true)
	CombinationHints   bool // Show combination hints (default: true)
	AutoDerive         bool // Auto-derive profiles from account data (default: true)
}

// LoadFoerderungConfig loads Förderungsradar configuration from environment
func LoadFoerderungConfig() *FoerderungConfig {
	cfg := &FoerderungConfig{
		// Search Settings
		SearchTimeoutSeconds: getEnvIntDefault("FOERDERUNG_SEARCH_TIMEOUT", 30),
		RuleThresholdPercent: getEnvIntDefault("FOERDERUNG_RULE_THRESHOLD", 50),
		MaxResultsPerSearch:  getEnvIntDefault("FOERDERUNG_MAX_RESULTS", 20),

		// LLM Settings
		LLMMaxConcurrent:   getEnvIntDefault("FOERDERUNG_LLM_MAX_CONCURRENT", 10),
		LLMTimeoutSeconds:  getEnvIntDefault("FOERDERUNG_LLM_TIMEOUT", 20),
		LLMCostLimitCents:  getEnvIntDefault("FOERDERUNG_LLM_COST_LIMIT", 10),
		LLMModel:           getEnvDefault("FOERDERUNG_LLM_MODEL", "claude-sonnet-4-20250514"),
		LLMTemperature:     getEnvFloatDefault("FOERDERUNG_LLM_TEMPERATURE", 0.3),
		LLMMaxTokens:       getEnvIntDefault("FOERDERUNG_LLM_MAX_TOKENS", 2000),

		// Scoring Weights
		RuleScoreWeight: getEnvFloatDefault("FOERDERUNG_RULE_WEIGHT", 0.4),
		LLMScoreWeight:  getEnvFloatDefault("FOERDERUNG_LLM_WEIGHT", 0.6),

		// Monitoring Settings
		MonitorDefaultThreshold: getEnvIntDefault("FOERDERUNG_MONITOR_THRESHOLD", 70),
		MonitorJobCron:          getEnvDefault("FOERDERUNG_MONITOR_CRON", "0 6 * * *"),
		MonitorDigestCron:       getEnvDefault("FOERDERUNG_DIGEST_CRON", "0 8 * * 1"),

		// Expiry Settings
		ExpiryJobCron:     getEnvDefault("FOERDERUNG_EXPIRY_CRON", "0 1 * * *"),
		ExpiryWarningDays: getEnvIntDefault("FOERDERUNG_EXPIRY_WARNING_DAYS", 7),

		// Cache Settings
		SearchCacheTTLHours:       getEnvIntDefault("FOERDERUNG_SEARCH_CACHE_TTL", 24),
		FoerderungCacheTTLMinutes: getEnvIntDefault("FOERDERUNG_CACHE_TTL", 60),

		// Feature Flags
		LLMFallbackEnabled: getEnvBoolDefault("FOERDERUNG_LLM_FALLBACK", true),
		CombinationHints:   getEnvBoolDefault("FOERDERUNG_COMBINATION_HINTS", true),
		AutoDerive:         getEnvBoolDefault("FOERDERUNG_AUTO_DERIVE", true),
	}

	// Parse durations
	cfg.SearchTimeout = time.Duration(cfg.SearchTimeoutSeconds) * time.Second
	cfg.LLMTimeout = time.Duration(cfg.LLMTimeoutSeconds) * time.Second

	return cfg
}

// Helper functions

func getEnvDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvIntDefault(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvFloatDefault(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvBoolDefault(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}
