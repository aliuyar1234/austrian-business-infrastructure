#!/bin/bash
# =============================================================================
# Restore script for Austrian Business Platform
# =============================================================================
# Restores from a backup created by backup.sh
#
# Usage:
#   ./scripts/restore.sh backup_20240101_120000.tar.gz
#
# WARNING: This will OVERWRITE your current database!
# =============================================================================

set -e

BACKUP_FILE="$1"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.selfhost.yml}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

if [ -z "$BACKUP_FILE" ]; then
    log_error "Usage: $0 <backup_file.tar.gz>"
    exit 1
fi

if [ ! -f "$BACKUP_FILE" ]; then
    log_error "Backup file not found: $BACKUP_FILE"
    exit 1
fi

# Confirmation prompt
echo ""
log_warn "WARNING: This will OVERWRITE your current database!"
log_warn "Backup file: $BACKUP_FILE"
echo ""
read -p "Are you sure you want to continue? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    log_info "Restore cancelled"
    exit 0
fi

# Extract backup
WORK_DIR=$(mktemp -d)
trap "rm -rf $WORK_DIR" EXIT

log_info "Extracting backup..."
tar -xzf "$BACKUP_FILE" -C "$WORK_DIR"

# Show metadata
if [ -f "$WORK_DIR/metadata.json" ]; then
    log_info "Backup metadata:"
    cat "$WORK_DIR/metadata.json"
    echo ""
fi

# Stop server to prevent writes during restore
log_info "Stopping server..."
docker compose -f "$COMPOSE_FILE" stop server || true

# Restore PostgreSQL
if [ -f "$WORK_DIR/postgres.sql" ]; then
    log_info "Restoring PostgreSQL database..."

    # Drop and recreate database
    docker compose -f "$COMPOSE_FILE" exec -T postgres psql -U "${POSTGRES_USER:-abp}" -c "DROP DATABASE IF EXISTS ${POSTGRES_DB:-abp};" postgres || true
    docker compose -f "$COMPOSE_FILE" exec -T postgres psql -U "${POSTGRES_USER:-abp}" -c "CREATE DATABASE ${POSTGRES_DB:-abp};" postgres

    # Restore data
    docker compose -f "$COMPOSE_FILE" exec -T postgres psql -U "${POSTGRES_USER:-abp}" < "$WORK_DIR/postgres.sql"
    log_info "PostgreSQL restore complete"
else
    log_warn "No PostgreSQL backup found in archive"
fi

# Restore Redis (optional)
if [ -f "$WORK_DIR/redis.rdb" ]; then
    log_info "Restoring Redis data..."
    docker compose -f "$COMPOSE_FILE" stop redis
    docker compose -f "$COMPOSE_FILE" cp "$WORK_DIR/redis.rdb" redis:/data/dump.rdb
    docker compose -f "$COMPOSE_FILE" start redis
    log_info "Redis restore complete"
else
    log_info "No Redis backup found (this is normal for fresh installs)"
fi

# Restart services
log_info "Starting services..."
docker compose -f "$COMPOSE_FILE" up -d

log_info "Restore complete!"
log_info "Please verify your data is correct"
