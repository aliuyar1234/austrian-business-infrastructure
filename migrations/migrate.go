package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.sql
var migrationFiles embed.FS

// Migration represents a database migration
type Migration struct {
	Version   string
	Name      string
	SQL       string
	AppliedAt time.Time
}

// Migrator handles database migrations
type Migrator struct {
	pool *pgxpool.Pool
}

// NewMigrator creates a new migrator instance
func NewMigrator(pool *pgxpool.Pool) *Migrator {
	return &Migrator{pool: pool}
}

// Initialize creates the migrations tracking table
func (m *Migrator) Initialize(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`
	_, err := m.pool.Exec(ctx, query)
	return err
}

// AppliedMigrations returns list of already applied migrations
func (m *Migrator) AppliedMigrations(ctx context.Context) (map[string]bool, error) {
	rows, err := m.pool.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

// LoadMigrations loads all migration files from embedded filesystem
func (m *Migrator) LoadMigrations() ([]Migration, error) {
	var migrations []Migration

	err := fs.WalkDir(migrationFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		content, err := migrationFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", path, err)
		}

		// Parse filename: 001_name.sql -> version=001, name=name
		base := filepath.Base(path)
		base = strings.TrimSuffix(base, ".sql")
		parts := strings.SplitN(base, "_", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid migration filename: %s (expected NNN_name.sql)", path)
		}

		migrations = append(migrations, Migration{
			Version: parts[0],
			Name:    parts[1],
			SQL:     string(content),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// Up applies all pending migrations
func (m *Migrator) Up(ctx context.Context) error {
	if err := m.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize migrations table: %w", err)
	}

	applied, err := m.AppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	migrations, err := m.LoadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	for _, migration := range migrations {
		if applied[migration.Version] {
			continue
		}

		if err := m.apply(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %s_%s: %w", migration.Version, migration.Name, err)
		}

		fmt.Printf("Applied migration: %s_%s\n", migration.Version, migration.Name)
	}

	return nil
}

// apply executes a single migration in a transaction
func (m *Migrator) apply(ctx context.Context, migration Migration) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Execute migration SQL
	if _, err := tx.Exec(ctx, migration.SQL); err != nil {
		return fmt.Errorf("migration SQL failed: %w", err)
	}

	// Record migration
	_, err = tx.Exec(ctx,
		"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
		migration.Version, migration.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit(ctx)
}

// Status returns the status of all migrations
func (m *Migrator) Status(ctx context.Context) ([]Migration, error) {
	if err := m.Initialize(ctx); err != nil {
		return nil, err
	}

	applied, err := m.AppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	migrations, err := m.LoadMigrations()
	if err != nil {
		return nil, err
	}

	// Get applied_at times
	rows, err := m.pool.Query(ctx, "SELECT version, applied_at FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appliedTimes := make(map[string]time.Time)
	for rows.Next() {
		var version string
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, err
		}
		appliedTimes[version] = appliedAt
	}

	for i := range migrations {
		if applied[migrations[i].Version] {
			migrations[i].AppliedAt = appliedTimes[migrations[i].Version]
		}
	}

	return migrations, nil
}
