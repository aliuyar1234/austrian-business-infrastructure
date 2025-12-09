package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// ServerConfig holds all server configuration
type ServerConfig struct {
	// Server
	ServerHost string
	ServerPort int
	LogLevel   string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// JWT
	JWTSecret             string
	JWTAccessTokenExpiry  time.Duration
	JWTRefreshTokenExpiry time.Duration

	// Encryption
	EncryptionKey string

	// OAuth2
	GoogleClientID       string
	GoogleClientSecret   string
	MicrosoftClientID    string
	MicrosoftClientSecret string
	OAuthEnabled         bool

	// Rate Limiting
	RateLimitRequestsPerMinute int
	RateLimitLoginPerMinute    int

	// Email
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	// Application
	AppName        string
	AppURL         string
	AllowedOrigins []string

	// Features
	EnableRegistration bool

	// AI Configuration
	ClaudeAPIKey       string
	ClaudeModel        string
	ClaudeMaxTokens    int
	AIEnabled          bool
	AIMaxCostPerDoc    int // max cost in cents per document analysis
	AIRateLimitPerMin  int

	// OCR Configuration
	OCRProvider          string // hunyuan, tesseract, auto
	OCRHunyuanURL        string // URL for HunyuanOCR bridge service
	OCRTesseractPath     string // Path to tesseract binary
	OCRConfidenceMin     float64

	// Storage
	StorageType           string
	StorageLocalPath      string
	StorageS3Endpoint     string
	StorageS3Bucket       string
	StorageS3Region       string
	StorageS3AccessKeyID  string
	StorageS3SecretKey    string
	StorageS3UseSSL       bool

	// ELDA Configuration
	ELDAEndpoint          string
	ELDATestEndpoint      string
	ELDACertPath          string
	ELDATimeoutSeconds    int
	ELDARetryMax          int
	ELDACertExpiryWarnDays int
	ELDATestMode          bool
}

// LoadServerConfig loads configuration from environment variables
func LoadServerConfig() (*ServerConfig, error) {
	cfg := &ServerConfig{
		// Server defaults
		ServerHost: getEnv("SERVER_HOST", "0.0.0.0"),
		ServerPort: getEnvInt("SERVER_PORT", 8080),
		LogLevel:   getEnv("LOG_LEVEL", "info"),

		// Required
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		RedisURL:      os.Getenv("REDIS_URL"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		EncryptionKey: os.Getenv("ENCRYPTION_KEY"),

		// JWT timing
		JWTAccessTokenExpiry:  getEnvDuration("JWT_ACCESS_TOKEN_EXPIRY", 15*time.Minute),
		JWTRefreshTokenExpiry: getEnvDuration("JWT_REFRESH_TOKEN_EXPIRY", 7*24*time.Hour),

		// OAuth2
		GoogleClientID:        os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:    os.Getenv("GOOGLE_CLIENT_SECRET"),
		MicrosoftClientID:     os.Getenv("MICROSOFT_CLIENT_ID"),
		MicrosoftClientSecret: os.Getenv("MICROSOFT_CLIENT_SECRET"),
		OAuthEnabled:          getEnvBool("ENABLE_OAUTH", false),

		// Rate limiting
		RateLimitRequestsPerMinute: getEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 100),
		RateLimitLoginPerMinute:    getEnvInt("RATE_LIMIT_LOGIN_PER_MINUTE", 5),

		// Email
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     getEnvInt("SMTP_PORT", 587),
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     getEnv("SMTP_FROM", "noreply@example.com"),

		// Application
		AppName:        getEnv("APP_NAME", "Austrian Business Platform"),
		AppURL:         getEnv("APP_URL", "http://localhost:8080"),
		AllowedOrigins: getEnvList("ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:8080"}),

		// Features
		EnableRegistration: getEnvBool("ENABLE_REGISTRATION", true),

		// AI Configuration
		ClaudeAPIKey:      os.Getenv("CLAUDE_API_KEY"),
		ClaudeModel:       getEnv("CLAUDE_MODEL", "claude-sonnet-4-20250514"),
		ClaudeMaxTokens:   getEnvInt("CLAUDE_MAX_TOKENS", 4096),
		AIEnabled:         getEnvBool("AI_ENABLED", true),
		AIMaxCostPerDoc:   getEnvInt("AI_MAX_COST_PER_DOC_CENTS", 10), // 10 cents max
		AIRateLimitPerMin: getEnvInt("AI_RATE_LIMIT_PER_MIN", 60),

		// OCR Configuration
		OCRProvider:      getEnv("OCR_PROVIDER", "auto"), // auto, hunyuan, tesseract
		OCRHunyuanURL:    getEnv("OCR_HUNYUAN_URL", "http://localhost:8090"),
		OCRTesseractPath: getEnv("OCR_TESSERACT_PATH", "tesseract"),
		OCRConfidenceMin: getEnvFloat("OCR_CONFIDENCE_MIN", 0.7),

		// Storage
		StorageType:           getEnv("STORAGE_TYPE", "local"),
		StorageLocalPath:      getEnv("STORAGE_LOCAL_PATH", "./data/documents"),
		StorageS3Endpoint:     os.Getenv("STORAGE_S3_ENDPOINT"),
		StorageS3Bucket:       getEnv("STORAGE_S3_BUCKET", "documents"),
		StorageS3Region:       getEnv("STORAGE_S3_REGION", "us-east-1"),
		StorageS3AccessKeyID:  os.Getenv("STORAGE_S3_ACCESS_KEY_ID"),
		StorageS3SecretKey:    os.Getenv("STORAGE_S3_SECRET_KEY"),
		StorageS3UseSSL:       getEnvBool("STORAGE_S3_USE_SSL", true),

		// ELDA Configuration
		ELDAEndpoint:           getEnv("ELDA_ENDPOINT", "https://elda.sozvers.at/elda-webservice/"),
		ELDATestEndpoint:       getEnv("ELDA_TEST_ENDPOINT", "https://elda-test.sozvers.at/elda-webservice/"),
		ELDACertPath:           getEnv("ELDA_CERT_PATH", "/etc/elda/certs"),
		ELDATimeoutSeconds:     getEnvInt("ELDA_TIMEOUT_SECONDS", 60),
		ELDARetryMax:           getEnvInt("ELDA_RETRY_MAX", 3),
		ELDACertExpiryWarnDays: getEnvInt("ELDA_CERT_EXPIRY_WARN_DAYS", 30),
		ELDATestMode:           getEnvBool("ELDA_TEST_MODE", false),
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all required configuration is present
// In production, this will reject insecure defaults to prevent misconfiguration
func (c *ServerConfig) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if c.EncryptionKey == "" {
		return fmt.Errorf("ENCRYPTION_KEY is required")
	}
	if len(c.EncryptionKey) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes for AES-256")
	}

	// Reject insecure defaults in production (fail-fast for self-hosted users)
	env := getEnv("APP_ENV", "production")
	if env == "production" || env == "prod" {
		// Block known insecure defaults
		insecureSecrets := []string{
			"dev-jwt-secret-change-in-production",
			"your-256-bit-secret-key-change-in-production",
			"change-me",
			"secret",
		}
		for _, insecure := range insecureSecrets {
			if c.JWTSecret == insecure {
				return fmt.Errorf("JWT_SECRET contains an insecure default value - please generate a secure secret with: openssl rand -hex 32")
			}
		}

		// Block default encryption key
		if c.EncryptionKey == "12345678901234567890123456789012" ||
			c.EncryptionKey == "your-32-byte-encryption-key-here" {
			return fmt.Errorf("ENCRYPTION_KEY contains an insecure default value - please generate a secure key with: openssl rand -hex 16")
		}

		// Block dev database passwords in DATABASE_URL
		if containsAny(c.DatabaseURL, []string{"abp_dev_password", "password", "postgres:postgres"}) {
			return fmt.Errorf("DATABASE_URL contains an insecure default password - please use a strong password")
		}
	}

	return nil
}

// containsAny checks if s contains any of the substrings
func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// Address returns the server address in host:port format
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.ServerHost, c.ServerPort)
}

// StorageConfig returns a StorageConfig struct for document storage initialization
type StorageConfigResult struct {
	Type              string
	LocalPath         string
	S3Endpoint        string
	S3Bucket          string
	S3Region          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3UseSSL          bool
}

func (c *ServerConfig) StorageConfig() *StorageConfigResult {
	return &StorageConfigResult{
		Type:              c.StorageType,
		LocalPath:         c.StorageLocalPath,
		S3Endpoint:        c.StorageS3Endpoint,
		S3Bucket:          c.StorageS3Bucket,
		S3Region:          c.StorageS3Region,
		S3AccessKeyID:     c.StorageS3AccessKeyID,
		S3SecretAccessKey: c.StorageS3SecretKey,
		S3UseSSL:          c.StorageS3UseSSL,
	}
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvList(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		var result []string
		for _, s := range splitAndTrim(value, ",") {
			if s != "" {
				result = append(result, s)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range split(s, sep) {
		trimmed := trim(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func split(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// WorkerConfig holds worker process configuration
type WorkerConfig struct {
	// Database
	DatabaseURL string

	// Redis (optional for distributed locks)
	RedisURL string

	// Worker settings
	WorkerConcurrency int
	PollInterval      time.Duration
	ShutdownTimeout   time.Duration
	JobTimeout        time.Duration

	// Health server
	HealthPort int

	// Logging
	LogLevel string
}

// LoadWorkerConfig loads worker configuration from environment variables
func LoadWorkerConfig() (*WorkerConfig, error) {
	cfg := &WorkerConfig{
		// Required
		DatabaseURL: os.Getenv("DATABASE_URL"),

		// Optional
		RedisURL: os.Getenv("REDIS_URL"),

		// Worker settings with defaults
		WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 5),
		PollInterval:      getEnvDuration("WORKER_POLL_INTERVAL", 1*time.Second),
		ShutdownTimeout:   getEnvDuration("WORKER_SHUTDOWN_TIMEOUT", 30*time.Second),
		JobTimeout:        getEnvDuration("JOB_TIMEOUT", 30*time.Minute),

		// Health server
		HealthPort: getEnvInt("WORKER_HEALTH_PORT", 8081),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all required configuration is present
func (c *WorkerConfig) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.WorkerConcurrency < 1 {
		return fmt.Errorf("WORKER_CONCURRENCY must be at least 1")
	}
	if c.WorkerConcurrency > 100 {
		return fmt.Errorf("WORKER_CONCURRENCY must be at most 100")
	}
	return nil
}
