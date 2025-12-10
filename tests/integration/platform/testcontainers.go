package platform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	// Clear all tables with data (excluding schema_migrations)
	// Uses CASCADE to handle foreign key dependencies automatically
	_, err := e.DB.Exec(ctx, `
		DO $$
		DECLARE
			r RECORD;
		BEGIN
			-- Disable triggers temporarily for faster truncation
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename != 'schema_migrations')
			LOOP
				EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("failed to truncate tables: %w", err)
	}

	// Clear Redis
	if err := e.Redis.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("failed to flush redis: %w", err)
	}

	return nil
}

// runMigrations applies database migrations from the migrations/ folder
// This ensures integration tests exercise the same schema as production,
// including RLS policies, triggers, and constraints
func (e *TestEnvironment) runMigrations(ctx context.Context) error {
	// Find migrations directory - try multiple paths for different test contexts
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		return fmt.Errorf("migrations directory not found")
	}

	// Read all migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	// Filter and sort SQL files
	var migrations []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			migrations = append(migrations, f.Name())
		}
	}
	sort.Strings(migrations) // Ensures 001_, 002_, etc. order

	// Create schema_migrations table to track applied migrations
	_, err = e.DB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	// Apply each migration
	for _, migration := range migrations {
		// Check if already applied
		var exists bool
		err := e.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", migration).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", migration, err)
		}
		if exists {
			continue
		}

		// Read migration file
		content, err := os.ReadFile(filepath.Join(migrationsDir, migration))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", migration, err)
		}

		// Execute migration
		_, err = e.DB.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("execute migration %s: %w", migration, err)
		}

		// Record migration
		_, err = e.DB.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", migration)
		if err != nil {
			return fmt.Errorf("record migration %s: %w", migration, err)
		}
	}

	return nil
}

// findMigrationsDir locates the migrations directory from various test contexts
func findMigrationsDir() string {
	// Try common paths relative to different test execution contexts
	candidates := []string{
		"migrations",                          // From repo root
		"../../../migrations",                 // From tests/integration/platform/
		"../../migrations",                    // From tests/integration/
		os.Getenv("MIGRATIONS_DIR"),           // Explicit env var
	}

	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		if _, err := os.Stat(dir); err == nil {
			absPath, _ := filepath.Abs(dir)
			return absPath
		}
	}

	// Try to find from current working directory
	cwd, _ := os.Getwd()
	for i := 0; i < 5; i++ { // Walk up to 5 levels
		candidate := filepath.Join(cwd, "migrations")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		cwd = filepath.Dir(cwd)
	}

	return ""
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
