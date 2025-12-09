package crypto

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// SecretProvider is an interface for retrieving secrets from various backends
type SecretProvider interface {
	// GetSecret retrieves a secret by name
	GetSecret(ctx context.Context, name string) (string, error)
	// GetSecretWithVersion retrieves a specific version of a secret
	GetSecretWithVersion(ctx context.Context, name, version string) (string, error)
	// Name returns the provider name for logging
	Name() string
}

// SecretManager manages secrets from multiple providers with caching
type SecretManager struct {
	providers []SecretProvider
	cache     *secretCache
	mu        sync.RWMutex
}

// SecretManagerConfig configures the secret manager
type SecretManagerConfig struct {
	// CacheTTL is how long to cache secrets (0 disables caching)
	CacheTTL time.Duration
	// Providers are checked in order until one succeeds
	Providers []SecretProvider
}

// NewSecretManager creates a new secret manager
func NewSecretManager(config *SecretManagerConfig) *SecretManager {
	sm := &SecretManager{
		providers: config.Providers,
	}

	if config.CacheTTL > 0 {
		sm.cache = newSecretCache(config.CacheTTL)
	}

	return sm
}

// GetSecret retrieves a secret, checking providers in order
func (sm *SecretManager) GetSecret(ctx context.Context, name string) (string, error) {
	// Check cache first
	if sm.cache != nil {
		if val, ok := sm.cache.Get(name); ok {
			return val, nil
		}
	}

	sm.mu.RLock()
	providers := sm.providers
	sm.mu.RUnlock()

	var lastErr error
	for _, provider := range providers {
		val, err := provider.GetSecret(ctx, name)
		if err == nil {
			// Cache successful result
			if sm.cache != nil {
				sm.cache.Set(name, val)
			}
			return val, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return "", fmt.Errorf("all secret providers failed for %s: %w", name, lastErr)
	}
	return "", fmt.Errorf("no secret providers configured")
}

// GetSecretBytes retrieves a secret and decodes it from base64
func (sm *SecretManager) GetSecretBytes(ctx context.Context, name string) ([]byte, error) {
	val, err := sm.GetSecret(ctx, name)
	if err != nil {
		return nil, err
	}

	// Try base64 decoding first
	decoded, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		// Not base64, return as raw bytes
		return []byte(val), nil
	}
	return decoded, nil
}

// MustGetSecret retrieves a secret or panics
func (sm *SecretManager) MustGetSecret(ctx context.Context, name string) string {
	val, err := sm.GetSecret(ctx, name)
	if err != nil {
		panic(fmt.Sprintf("failed to get required secret %s: %v", name, err))
	}
	return val
}

// InvalidateCache clears a specific secret from cache
func (sm *SecretManager) InvalidateCache(name string) {
	if sm.cache != nil {
		sm.cache.Delete(name)
	}
}

// InvalidateAllCache clears the entire cache
func (sm *SecretManager) InvalidateAllCache() {
	if sm.cache != nil {
		sm.cache.Clear()
	}
}

// secretCache provides TTL-based caching for secrets
type secretCache struct {
	data map[string]cacheEntry
	ttl  time.Duration
	mu   sync.RWMutex
}

type cacheEntry struct {
	value   string
	expires time.Time
}

func newSecretCache(ttl time.Duration) *secretCache {
	return &secretCache{
		data: make(map[string]cacheEntry),
		ttl:  ttl,
	}
}

func (c *secretCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok || time.Now().After(entry.expires) {
		return "", false
	}
	return entry.value, true
}

func (c *secretCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheEntry{
		value:   value,
		expires: time.Now().Add(c.ttl),
	}
}

func (c *secretCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *secretCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]cacheEntry)
}

// ============================================================================
// Built-in Providers
// ============================================================================

// EnvSecretProvider retrieves secrets from environment variables
type EnvSecretProvider struct {
	// Prefix is prepended to secret names when looking up env vars
	// e.g., Prefix="APP_" means GetSecret("DB_PASSWORD") looks for APP_DB_PASSWORD
	Prefix string
}

// NewEnvSecretProvider creates an environment variable secret provider
func NewEnvSecretProvider(prefix string) *EnvSecretProvider {
	return &EnvSecretProvider{Prefix: prefix}
}

func (p *EnvSecretProvider) Name() string {
	return "env"
}

func (p *EnvSecretProvider) GetSecret(ctx context.Context, name string) (string, error) {
	envName := p.Prefix + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	val := os.Getenv(envName)
	if val == "" {
		return "", fmt.Errorf("environment variable %s not set", envName)
	}
	return val, nil
}

func (p *EnvSecretProvider) GetSecretWithVersion(ctx context.Context, name, version string) (string, error) {
	// Env vars don't support versioning
	return p.GetSecret(ctx, name)
}

// FileSecretProvider retrieves secrets from files (e.g., Kubernetes secrets mounted as files)
type FileSecretProvider struct {
	// BasePath is the directory containing secret files
	BasePath string
}

// NewFileSecretProvider creates a file-based secret provider
func NewFileSecretProvider(basePath string) *FileSecretProvider {
	return &FileSecretProvider{BasePath: basePath}
}

func (p *FileSecretProvider) Name() string {
	return "file"
}

func (p *FileSecretProvider) GetSecret(ctx context.Context, name string) (string, error) {
	path := fmt.Sprintf("%s/%s", p.BasePath, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read secret file %s: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func (p *FileSecretProvider) GetSecretWithVersion(ctx context.Context, name, version string) (string, error) {
	// File secrets don't support versioning
	return p.GetSecret(ctx, name)
}

// JSONFileSecretProvider retrieves secrets from a JSON file
type JSONFileSecretProvider struct {
	FilePath string
	data     map[string]string
	loaded   bool
	mu       sync.RWMutex
}

// NewJSONFileSecretProvider creates a JSON file secret provider
func NewJSONFileSecretProvider(filePath string) *JSONFileSecretProvider {
	return &JSONFileSecretProvider{FilePath: filePath}
}

func (p *JSONFileSecretProvider) Name() string {
	return "json_file"
}

func (p *JSONFileSecretProvider) load() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.loaded {
		return nil
	}

	data, err := os.ReadFile(p.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read secrets file %s: %w", p.FilePath, err)
	}

	if err := json.Unmarshal(data, &p.data); err != nil {
		return fmt.Errorf("failed to parse secrets file: %w", err)
	}

	p.loaded = true
	return nil
}

func (p *JSONFileSecretProvider) GetSecret(ctx context.Context, name string) (string, error) {
	if err := p.load(); err != nil {
		return "", err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	val, ok := p.data[name]
	if !ok {
		return "", fmt.Errorf("secret %s not found in JSON file", name)
	}
	return val, nil
}

func (p *JSONFileSecretProvider) GetSecretWithVersion(ctx context.Context, name, version string) (string, error) {
	return p.GetSecret(ctx, name)
}

// ============================================================================
// Default Configuration
// ============================================================================

// DefaultSecretManager creates a secret manager with standard providers
// Priority: Environment variables > File secrets > JSON config
func DefaultSecretManager() *SecretManager {
	providers := []SecretProvider{
		NewEnvSecretProvider(""),
	}

	// Add file provider if secrets directory exists
	if _, err := os.Stat("/run/secrets"); err == nil {
		providers = append(providers, NewFileSecretProvider("/run/secrets"))
	}

	// Add Kubernetes-style secrets if they exist
	if _, err := os.Stat("/var/run/secrets"); err == nil {
		providers = append(providers, NewFileSecretProvider("/var/run/secrets"))
	}

	return NewSecretManager(&SecretManagerConfig{
		CacheTTL:  5 * time.Minute,
		Providers: providers,
	})
}

// SecretNames defines standard secret names used by the application
var SecretNames = struct {
	JWTPrivateKey      string
	JWTPublicKey       string
	DatabaseURL        string
	RedisURL           string
	MasterEncryptKey   string
	FinanzOnlineAPIKey string
	ELDAAPIKey         string
	FirmenbuchAPIKey   string
	SMTPPassword       string
	S3AccessKey        string
	S3SecretKey        string
}{
	JWTPrivateKey:      "JWT_ECDSA_PRIVATE_KEY",
	JWTPublicKey:       "JWT_ECDSA_PUBLIC_KEY",
	DatabaseURL:        "DATABASE_URL",
	RedisURL:           "REDIS_URL",
	MasterEncryptKey:   "MASTER_ENCRYPT_KEY",
	FinanzOnlineAPIKey: "FINANZONLINE_API_KEY",
	ELDAAPIKey:         "ELDA_API_KEY",
	FirmenbuchAPIKey:   "FIRMENBUCH_API_KEY",
	SMTPPassword:       "SMTP_PASSWORD",
	S3AccessKey:        "S3_ACCESS_KEY",
	S3SecretKey:        "S3_SECRET_KEY",
}
