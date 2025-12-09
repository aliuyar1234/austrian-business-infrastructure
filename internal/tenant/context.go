package tenant

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNoTenantContext is returned when a tenant-scoped operation is attempted
	// without a tenant context. This should cause operations to fail closed.
	ErrNoTenantContext = errors.New("no tenant context: operation requires tenant scope")

	// ErrCrossTenantAccess is returned when an operation attempts to access
	// resources belonging to a different tenant.
	ErrCrossTenantAccess = errors.New("cross-tenant access denied")
)

// contextKey is a private type for context keys to prevent collisions
type contextKey int

const (
	tenantIDKey contextKey = iota
	userIDKey
	roleKey
)

// WithTenantID returns a new context with the tenant ID set.
// This should be called in middleware after JWT validation.
func WithTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// GetTenantID retrieves the tenant ID from context.
// Returns uuid.Nil if no tenant ID is set.
func GetTenantID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(tenantIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// MustGetTenantID retrieves the tenant ID from context or panics.
// Use this only in contexts where tenant ID is guaranteed to exist.
func MustGetTenantID(ctx context.Context) uuid.UUID {
	id := GetTenantID(ctx)
	if id == uuid.Nil {
		panic("tenant ID not in context")
	}
	return id
}

// RequireTenantID retrieves the tenant ID from context or returns an error.
// This is the recommended way to get tenant ID in request handlers.
func RequireTenantID(ctx context.Context) (uuid.UUID, error) {
	id := GetTenantID(ctx)
	if id == uuid.Nil {
		return uuid.Nil, ErrNoTenantContext
	}
	return id, nil
}

// WithUserID returns a new context with the user ID set.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID retrieves the user ID from context.
// Returns uuid.Nil if no user ID is set.
func GetUserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(userIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// WithRole returns a new context with the user role set.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

// GetRole retrieves the user role from context.
// Returns empty string if no role is set.
func GetRole(ctx context.Context) string {
	if role, ok := ctx.Value(roleKey).(string); ok {
		return role
	}
	return ""
}

// SetTenantIDForRLS sets the app.tenant_id session variable on a database connection.
// This enables PostgreSQL Row Level Security to filter by tenant.
//
// Must be called on each database connection before executing tenant-scoped queries.
// The RLS policies use: tenant_id = current_setting('app.tenant_id')::uuid
func SetTenantIDForRLS(ctx context.Context, conn *pgx.Conn, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrNoTenantContext
	}

	_, err := conn.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	return nil
}

// SetTenantIDForPool sets the app.tenant_id session variable for a pooled connection.
// Use this with pgxpool when executing tenant-scoped queries.
func SetTenantIDForPool(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrNoTenantContext
	}

	_, err := pool.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	return nil
}

// ClearTenantIDForRLS clears the app.tenant_id session variable.
// This should be called after completing tenant-scoped operations.
func ClearTenantIDForRLS(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, "RESET app.tenant_id")
	return err
}

// WithTenantContext is a helper that sets up the RLS context for a connection
// and ensures cleanup. It acquires a connection, sets the tenant context,
// executes the function, and releases the connection.
func WithTenantContext(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(conn *pgxpool.Conn) error) error {
	if tenantID == uuid.Nil {
		return ErrNoTenantContext
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Set tenant context for RLS
	_, err = conn.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	return fn(conn)
}

// ValidateTenantAccess checks if the tenant ID from context matches the expected tenant.
// Returns ErrCrossTenantAccess if there's a mismatch.
func ValidateTenantAccess(ctx context.Context, resourceTenantID uuid.UUID) error {
	contextTenantID := GetTenantID(ctx)
	if contextTenantID == uuid.Nil {
		return ErrNoTenantContext
	}
	if contextTenantID != resourceTenantID {
		return ErrCrossTenantAccess
	}
	return nil
}

// TenantContextMiddleware provides a database-level tenant context setup.
// This is used to ensure all queries within a request are properly scoped.
type TenantContextMiddleware struct {
	pool *pgxpool.Pool
}

// NewTenantContextMiddleware creates a new tenant context middleware
func NewTenantContextMiddleware(pool *pgxpool.Pool) *TenantContextMiddleware {
	return &TenantContextMiddleware{pool: pool}
}

// WrapConn wraps a connection with tenant context setup.
// The returned connection has RLS enabled for the specified tenant.
func (m *TenantContextMiddleware) WrapConn(ctx context.Context, tenantID uuid.UUID) (*pgxpool.Conn, error) {
	if tenantID == uuid.Nil {
		return nil, ErrNoTenantContext
	}

	conn, err := m.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	_, err = conn.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID.String())
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	return conn, nil
}
