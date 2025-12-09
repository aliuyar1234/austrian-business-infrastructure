package platform

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// TestEnvironment holds the test infrastructure
type TestEnvironment struct {
	PostgresURL string
	RedisURL    string
	DB          *pgxpool.Pool
	Redis       *redis.Client
	cleanup     []func()
}

// Setup creates a test environment with PostgreSQL and Redis
// For CI/CD, expects POSTGRES_URL and REDIS_URL environment variables
// For local development, uses testcontainers if available
func Setup(t *testing.T) *TestEnvironment {
	t.Helper()

	env := &TestEnvironment{}

	// Use environment variables for test databases
	// In production tests, these would point to testcontainers or docker-compose services
	postgresURL := getEnvOrDefault("TEST_POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/platform_test?sslmode=disable")
	redisURL := getEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/1")

	env.PostgresURL = postgresURL
	env.RedisURL = redisURL

	// Connect to PostgreSQL
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, postgresURL)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
		return nil
	}

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("PostgreSQL not reachable: %v", err)
		return nil
	}

	env.DB = pool
	env.cleanup = append(env.cleanup, func() { pool.Close() })

	// Connect to Redis
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Skipf("Invalid Redis URL: %v", err)
		return nil
	}

	rdb := redis.NewClient(opt)
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
		return nil
	}

	env.Redis = rdb
	env.cleanup = append(env.cleanup, func() { rdb.Close() })

	// Run migrations
	if err := env.runMigrations(ctx); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return env
}

// Cleanup releases all test resources
func (e *TestEnvironment) Cleanup() {
	for i := len(e.cleanup) - 1; i >= 0; i-- {
		e.cleanup[i]()
	}
}

// Reset clears all test data between tests
func (e *TestEnvironment) Reset(ctx context.Context) error {
	// Clear tables in reverse dependency order
	tables := []string{
		"audit_logs",
		"sessions",
		"api_keys",
		"invitations",
		"users",
		"tenants",
	}

	for _, table := range tables {
		if _, err := e.DB.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}

	// Clear Redis
	if err := e.Redis.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("failed to flush redis: %w", err)
	}

	return nil
}

// runMigrations applies database migrations
func (e *TestEnvironment) runMigrations(ctx context.Context) error {
	// Create tables if they don't exist
	schema := `
	-- Tenants
	CREATE TABLE IF NOT EXISTS tenants (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		slug VARCHAR(100) UNIQUE NOT NULL,
		settings JSONB DEFAULT '{}',
		status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	-- Users
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
		email VARCHAR(255) NOT NULL,
		password_hash VARCHAR(255),
		name VARCHAR(255),
		role VARCHAR(20) NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
		status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'pending')),
		email_verified BOOLEAN DEFAULT FALSE,
		totp_secret VARCHAR(255),
		totp_enabled BOOLEAN DEFAULT FALSE,
		oauth_provider VARCHAR(50),
		oauth_id VARCHAR(255),
		last_login_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(tenant_id, email)
	);

	-- Sessions
	CREATE TABLE IF NOT EXISTS sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		refresh_token_hash VARCHAR(255) NOT NULL,
		user_agent TEXT,
		ip_address VARCHAR(45),
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	-- API Keys
	CREATE TABLE IF NOT EXISTS api_keys (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		key_hash VARCHAR(255) NOT NULL,
		key_prefix VARCHAR(12) NOT NULL,
		scopes TEXT[] DEFAULT '{}',
		last_used_at TIMESTAMPTZ,
		expires_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	-- Invitations
	CREATE TABLE IF NOT EXISTS invitations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
		email VARCHAR(255) NOT NULL,
		role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'member', 'viewer')),
		token_hash VARCHAR(255) NOT NULL,
		invited_by UUID NOT NULL REFERENCES users(id),
		status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'expired')),
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	-- Audit Logs
	CREATE TABLE IF NOT EXISTS audit_logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
		user_id UUID REFERENCES users(id) ON DELETE SET NULL,
		action VARCHAR(100) NOT NULL,
		resource_type VARCHAR(100),
		resource_id UUID,
		details JSONB DEFAULT '{}',
		ip_address VARCHAR(45),
		user_agent TEXT,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
	CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
	CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix);
	CREATE INDEX IF NOT EXISTS idx_invitations_tenant_id ON invitations(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_invitations_token_hash ON invitations(token_hash);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_id ON audit_logs(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
	`

	_, err := e.DB.Exec(ctx, schema)
	return err
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
