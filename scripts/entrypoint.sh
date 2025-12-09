#!/bin/sh
set -e

# =============================================================================
# Entrypoint script for Austrian Business Platform
# =============================================================================
# Handles:
#   - Auto database migrations (if AUTO_MIGRATE=true)
#   - Graceful startup with dependency checks
# =============================================================================

echo "Starting Austrian Business Platform..."

# Wait for database to be ready
wait_for_db() {
    echo "Waiting for database..."
    max_attempts=30
    attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if pg_isready -h "${DB_HOST:-postgres}" -p "${DB_PORT:-5432}" -U "${POSTGRES_USER:-abp}" > /dev/null 2>&1; then
            echo "Database is ready!"
            return 0
        fi

        attempt=$((attempt + 1))
        echo "Database not ready (attempt $attempt/$max_attempts), waiting..."
        sleep 2
    done

    echo "ERROR: Database not ready after $max_attempts attempts"
    exit 1
}

# Run database migrations
run_migrations() {
    echo "Running database migrations..."

    # Check if migrations directory exists
    if [ ! -d "/app/migrations" ]; then
        echo "WARNING: No migrations directory found, skipping migrations"
        return 0
    fi

    # Run migrations using golang-migrate or built-in migrator
    if command -v migrate > /dev/null 2>&1; then
        migrate -path /app/migrations -database "$DATABASE_URL" up
    elif [ -f "/app/migrate" ]; then
        /app/migrate
    else
        echo "WARNING: No migration tool found, skipping migrations"
        echo "Install golang-migrate or include migration binary in image"
    fi

    echo "Migrations complete!"
}

# Main entrypoint logic
main() {
    # Parse DATABASE_URL to extract host for pg_isready
    if [ -n "$DATABASE_URL" ]; then
        # Extract host from postgres://user:pass@host:port/db
        DB_HOST=$(echo "$DATABASE_URL" | sed -E 's/.*@([^:\/]+).*/\1/')
        export DB_HOST
    fi

    # Wait for database
    if [ "${SKIP_DB_WAIT:-false}" != "true" ]; then
        wait_for_db
    fi

    # Run migrations if enabled
    if [ "${AUTO_MIGRATE:-false}" = "true" ]; then
        run_migrations
    fi

    # Execute the main application
    echo "Starting server..."
    exec /app/server "$@"
}

main "$@"
