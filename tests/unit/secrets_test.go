package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/crypto"
)

func TestEnvSecretProvider(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_SECRET", "my-secret-value")
	defer os.Unsetenv("TEST_SECRET")

	provider := crypto.NewEnvSecretProvider("")
	ctx := context.Background()

	value, err := provider.GetSecret(ctx, "TEST_SECRET")
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}

	if value != "my-secret-value" {
		t.Errorf("expected 'my-secret-value', got '%s'", value)
	}
}

func TestEnvSecretProvider_WithPrefix(t *testing.T) {
	os.Setenv("APP_DATABASE_URL", "postgres://localhost")
	defer os.Unsetenv("APP_DATABASE_URL")

	provider := crypto.NewEnvSecretProvider("APP_")
	ctx := context.Background()

	value, err := provider.GetSecret(ctx, "database-url")
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}

	if value != "postgres://localhost" {
		t.Errorf("expected 'postgres://localhost', got '%s'", value)
	}
}

func TestEnvSecretProvider_NotFound(t *testing.T) {
	provider := crypto.NewEnvSecretProvider("")
	ctx := context.Background()

	_, err := provider.GetSecret(ctx, "NON_EXISTENT_SECRET")
	if err == nil {
		t.Error("expected error for non-existent secret")
	}
}

func TestFileSecretProvider(t *testing.T) {
	// Create temp directory with secret file
	tmpDir := t.TempDir()
	secretPath := filepath.Join(tmpDir, "db-password")
	err := os.WriteFile(secretPath, []byte("super-secret-password\n"), 0600)
	if err != nil {
		t.Fatalf("failed to create secret file: %v", err)
	}

	provider := crypto.NewFileSecretProvider(tmpDir)
	ctx := context.Background()

	value, err := provider.GetSecret(ctx, "db-password")
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}

	// Should trim whitespace
	if value != "super-secret-password" {
		t.Errorf("expected 'super-secret-password', got '%s'", value)
	}
}

func TestFileSecretProvider_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	provider := crypto.NewFileSecretProvider(tmpDir)
	ctx := context.Background()

	_, err := provider.GetSecret(ctx, "non-existent")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestJSONFileSecretProvider(t *testing.T) {
	// Create temp JSON secrets file
	tmpDir := t.TempDir()
	secretsPath := filepath.Join(tmpDir, "secrets.json")
	content := `{
		"db_password": "json-secret-123",
		"api_key": "key-456"
	}`
	err := os.WriteFile(secretsPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create secrets file: %v", err)
	}

	provider := crypto.NewJSONFileSecretProvider(secretsPath)
	ctx := context.Background()

	// Test first secret
	value, err := provider.GetSecret(ctx, "db_password")
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}
	if value != "json-secret-123" {
		t.Errorf("expected 'json-secret-123', got '%s'", value)
	}

	// Test second secret
	value, err = provider.GetSecret(ctx, "api_key")
	if err != nil {
		t.Fatalf("failed to get second secret: %v", err)
	}
	if value != "key-456" {
		t.Errorf("expected 'key-456', got '%s'", value)
	}
}

func TestJSONFileSecretProvider_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	secretsPath := filepath.Join(tmpDir, "secrets.json")
	err := os.WriteFile(secretsPath, []byte(`{"existing": "value"}`), 0600)
	if err != nil {
		t.Fatalf("failed to create secrets file: %v", err)
	}

	provider := crypto.NewJSONFileSecretProvider(secretsPath)
	ctx := context.Background()

	_, err = provider.GetSecret(ctx, "non_existent")
	if err == nil {
		t.Error("expected error for non-existent key")
	}
}

func TestSecretManager_MultipleProviders(t *testing.T) {
	// Create file-based secret
	tmpDir := t.TempDir()
	secretPath := filepath.Join(tmpDir, "file-secret")
	err := os.WriteFile(secretPath, []byte("from-file"), 0600)
	if err != nil {
		t.Fatalf("failed to create secret file: %v", err)
	}

	// Set env secret with same name (should take priority)
	os.Setenv("FILE_SECRET", "from-env")
	defer os.Unsetenv("FILE_SECRET")

	// Create manager with env provider first (higher priority)
	manager := crypto.NewSecretManager(&crypto.SecretManagerConfig{
		CacheTTL: time.Minute,
		Providers: []crypto.SecretProvider{
			crypto.NewEnvSecretProvider(""),
			crypto.NewFileSecretProvider(tmpDir),
		},
	})

	ctx := context.Background()

	// Should get from env (first provider)
	value, err := manager.GetSecret(ctx, "FILE_SECRET")
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}
	if value != "from-env" {
		t.Errorf("expected 'from-env', got '%s'", value)
	}

	// Get secret only in file
	os.Setenv("FILE_SECRET", "") // Clear env var
	value, err = manager.GetSecret(ctx, "file-secret")
	if err != nil {
		t.Fatalf("failed to get file secret: %v", err)
	}
	if value != "from-file" {
		t.Errorf("expected 'from-file', got '%s'", value)
	}
}

func TestSecretManager_Caching(t *testing.T) {
	callCount := 0

	// Create a mock provider that counts calls
	mockProvider := &mockSecretProvider{
		secrets: map[string]string{"cached-secret": "initial-value"},
		onGet: func() {
			callCount++
		},
	}

	manager := crypto.NewSecretManager(&crypto.SecretManagerConfig{
		CacheTTL:  time.Hour, // Long TTL for test
		Providers: []crypto.SecretProvider{mockProvider},
	})

	ctx := context.Background()

	// First call should hit provider
	_, err := manager.GetSecret(ctx, "cached-secret")
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}

	// Second call should use cache
	_, err = manager.GetSecret(ctx, "cached-secret")
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (cached), got %d", callCount)
	}

	// Invalidate cache
	manager.InvalidateCache("cached-secret")

	// Third call should hit provider again
	_, err = manager.GetSecret(ctx, "cached-secret")
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls after invalidation, got %d", callCount)
	}
}

func TestSecretManager_GetSecretBytes(t *testing.T) {
	os.Setenv("BINARY_SECRET", "SGVsbG8gV29ybGQ=") // base64 of "Hello World"
	defer os.Unsetenv("BINARY_SECRET")

	manager := crypto.NewSecretManager(&crypto.SecretManagerConfig{
		Providers: []crypto.SecretProvider{
			crypto.NewEnvSecretProvider(""),
		},
	})

	ctx := context.Background()

	bytes, err := manager.GetSecretBytes(ctx, "BINARY_SECRET")
	if err != nil {
		t.Fatalf("failed to get secret bytes: %v", err)
	}

	if string(bytes) != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", string(bytes))
	}
}

func TestSecretManager_MustGetSecret_Panic(t *testing.T) {
	manager := crypto.NewSecretManager(&crypto.SecretManagerConfig{
		Providers: []crypto.SecretProvider{
			crypto.NewEnvSecretProvider(""),
		},
	})

	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing required secret")
		}
	}()

	manager.MustGetSecret(ctx, "NON_EXISTENT_REQUIRED_SECRET")
}

// mockSecretProvider is a test helper
type mockSecretProvider struct {
	secrets map[string]string
	onGet   func()
}

func (m *mockSecretProvider) Name() string {
	return "mock"
}

func (m *mockSecretProvider) GetSecret(ctx context.Context, name string) (string, error) {
	if m.onGet != nil {
		m.onGet()
	}
	if val, ok := m.secrets[name]; ok {
		return val, nil
	}
	return "", context.DeadlineExceeded // Simulate not found
}

func (m *mockSecretProvider) GetSecretWithVersion(ctx context.Context, name, version string) (string, error) {
	return m.GetSecret(ctx, name)
}
