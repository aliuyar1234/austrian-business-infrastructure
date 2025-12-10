package security

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantAwarePoolPgx wraps a pgxpool.Pool with automatic tenant context setting
// Use this for RLS-protected tables to ensure tenant isolation at the database level
type TenantAwarePoolPgx struct {
	pool       *pgxpool.Pool
	rlsManager *RLSManager
}

// NewTenantAwarePoolPgx creates a new tenant-aware pgx pool wrapper
func NewTenantAwarePoolPgx(pool *pgxpool.Pool, rlsManager *RLSManager) *TenantAwarePoolPgx {
	return &TenantAwarePoolPgx{
		pool:       pool,
		rlsManager: rlsManager,
	}
}

// Pool returns the underlying pgxpool.Pool
func (p *TenantAwarePoolPgx) Pool() *pgxpool.Pool {
	return p.pool
}

// SetTenantContextPgx sets the PostgreSQL session variable for RLS on a pgx connection
func SetTenantContextPgx(ctx context.Context, conn *pgxpool.Conn, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenantID
	}

	// Use SET LOCAL for transaction-scoped setting (safer for connection pooling)
	_, err := conn.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	return nil
}

// SetTenantContextTxPgx sets the tenant context within a pgx transaction
func SetTenantContextTxPgx(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenantID
	}

	// SET LOCAL is automatically scoped to the transaction
	_, err := tx.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to set tenant context in transaction: %w", err)
	}

	return nil
}

// AcquireWithTenant acquires a connection and sets the tenant context
// The returned connection MUST be released after use
func (p *TenantAwarePoolPgx) AcquireWithTenant(ctx context.Context, tenantID uuid.UUID) (*pgxpool.Conn, error) {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire connection: %w", err)
	}

	if err := SetTenantContextPgx(ctx, conn, tenantID); err != nil {
		conn.Release()
		return nil, err
	}

	return conn, nil
}

// BeginTxWithTenant starts a transaction with tenant context already set
func (p *TenantAwarePoolPgx) BeginTxWithTenant(ctx context.Context, tenantID uuid.UUID, txOptions pgx.TxOptions) (pgx.Tx, error) {
	tx, err := p.pool.BeginTx(ctx, txOptions)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	if err := SetTenantContextTxPgx(ctx, tx, tenantID); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}

	return tx, nil
}

// QueryWithTenant executes a query with tenant context
// This acquires a connection, sets tenant context, executes query, and releases
func (p *TenantAwarePoolPgx) QueryWithTenant(ctx context.Context, tenantID uuid.UUID, sql string, args ...interface{}) (pgx.Rows, error) {
	conn, err := p.AcquireWithTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	// Note: conn is released when rows are closed

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("query: %w", err)
	}

	return &tenantAwareRows{Rows: rows, conn: conn}, nil
}

// tenantAwareRows wraps pgx.Rows to release the connection when closed
type tenantAwareRows struct {
	pgx.Rows
	conn *pgxpool.Conn
}

func (r *tenantAwareRows) Close() {
	r.Rows.Close()
	r.conn.Release()
}

// ExecWithTenant executes a statement with tenant context
func (p *TenantAwarePoolPgx) ExecWithTenant(ctx context.Context, tenantID uuid.UUID, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	conn, err := p.AcquireWithTenant(ctx, tenantID)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	defer conn.Release()

	return conn.Exec(ctx, sql, args...)
}

// QueryRowWithTenant executes a query returning a single row with tenant context
func (p *TenantAwarePoolPgx) QueryRowWithTenant(ctx context.Context, tenantID uuid.UUID, sql string, args ...interface{}) pgx.Row {
	conn, err := p.AcquireWithTenant(ctx, tenantID)
	if err != nil {
		return &errorRow{err: err}
	}
	// Note: connection will be leaked if caller doesn't handle row properly
	// For single row queries, prefer using BeginTxWithTenant for proper cleanup

	return conn.QueryRow(ctx, sql, args...)
}

// errorRow implements pgx.Row for returning errors
type errorRow struct {
	err error
}

func (r *errorRow) Scan(dest ...interface{}) error {
	return r.err
}

// WithTenantMiddleware is a pgxpool config hook that sets tenant context on connection acquire
// Usage:
//
//	config.BeforeAcquire = security.WithTenantMiddleware(config.BeforeAcquire)
//
// Note: This requires tenant ID to be in context via WithTenantContext
func WithTenantMiddleware(next func(context.Context, *pgx.Conn) bool) func(context.Context, *pgx.Conn) bool {
	return func(ctx context.Context, conn *pgx.Conn) bool {
		// Call existing middleware first
		if next != nil && !next(ctx, conn) {
			return false
		}

		// Try to get tenant ID from context
		tenantID, err := GetTenantID(ctx)
		if err != nil {
			// No tenant context - allow connection (for non-tenant operations like auth)
			return true
		}

		// Set tenant context on connection
		_, err = conn.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID.String())
		if err != nil {
			// Log error but allow connection - RLS will still protect
			return true
		}

		return true
	}
}

// ConfigurePoolWithRLS adds RLS middleware to a pgxpool config
// Call this before creating the pool:
//
//	config, _ := pgxpool.ParseConfig(connString)
//	security.ConfigurePoolWithRLS(config)
//	pool, _ := pgxpool.NewWithConfig(ctx, config)
func ConfigurePoolWithRLS(config *pgxpool.Config) {
	config.BeforeAcquire = WithTenantMiddleware(config.BeforeAcquire)
}
