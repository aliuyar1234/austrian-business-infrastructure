package database

import (
	"context"
	"fmt"
	"time"

	"austrian-business-infrastructure/internal/security"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresConfig holds database connection configuration
type PostgresConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// DefaultPostgresConfig returns sensible defaults for PostgreSQL connection pool
func DefaultPostgresConfig(url string) *PostgresConfig {
	return &PostgresConfig{
		URL:             url,
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
}

// Pool wraps pgxpool.Pool with additional functionality
type Pool struct {
	*pgxpool.Pool
}

// NewPool creates a new PostgreSQL connection pool
func NewPool(ctx context.Context, cfg *PostgresConfig) (*Pool, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Apply pool settings
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	// Configure RLS middleware for automatic tenant context setting
	// This ensures app.tenant_id is set on each connection when tenant ID is in context
	security.ConfigurePoolWithRLS(poolConfig)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Pool{Pool: pool}, nil
}

// Close closes the connection pool
func (p *Pool) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}

// Health checks if the database connection is healthy
func (p *Pool) Health(ctx context.Context) error {
	return p.Ping(ctx)
}

// Stats returns connection pool statistics
func (p *Pool) Stats() *pgxpool.Stat {
	return p.Pool.Stat()
}
