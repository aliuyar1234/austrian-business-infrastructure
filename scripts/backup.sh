#!/bin/bash
# =============================================================================
# Backup script for Austrian Business Platform
# =============================================================================
# Creates timestamped backups of:
#   - PostgreSQL database
#   - Redis data (optional)
#   - Application data volumes
#
# Usage:
#   ./scripts/backup.sh                    # Backup to ./backups/
#   ./scripts/backup.sh /path/to/backups   # Backup to custom directory
#   ./scripts/backup.sh s3://bucket/path   # Backup to S3 (requires aws cli)
#
# Restore:
#   ./scripts/restore.sh backup_20240101_120000.tar.gz
# =============================================================================

set -e

# Configuration
BACKUP_DIR="${1:-./backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="abp_backup_${TIMESTAMP}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.selfhost.yml}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if running from project root
if [ ! -f "$COMPOSE_FILE" ]; then
    log_error "Compose file not found: $COMPOSE_FILE"
    log_error "Run this script from the project root directory"
    exit 1
fi

# Create backup directory
mkdir -p "$BACKUP_DIR"
WORK_DIR=$(mktemp -d)
trap "rm -rf $WORK_DIR" EXIT

log_info "Starting backup: $BACKUP_NAME"

# Backup PostgreSQL
log_info "Backing up PostgreSQL database..."
docker compose -f "$COMPOSE_FILE" exec -T postgres pg_dumpall -U "${POSTGRES_USER:-abp}" > "$WORK_DIR/postgres.sql"
if [ -s "$WORK_DIR/postgres.sql" ]; then
    log_info "PostgreSQL backup complete ($(du -h "$WORK_DIR/postgres.sql" | cut -f1))"
else
    log_error "PostgreSQL backup failed or empty"
    exit 1
fi

# Backup Redis (AOF file)
log_info "Backing up Redis data..."
docker compose -f "$COMPOSE_FILE" exec -T redis redis-cli -a "${REDIS_PASSWORD}" BGSAVE > /dev/null 2>&1 || true
sleep 2
docker compose -f "$COMPOSE_FILE" cp redis:/data/dump.rdb "$WORK_DIR/redis.rdb" 2>/dev/null || \
    log_warn "Redis backup skipped (no dump.rdb found)"

# Backup metadata
log_info "Saving backup metadata..."
cat > "$WORK_DIR/metadata.json" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "backup_name": "$BACKUP_NAME",
    "version": "$(docker compose -f "$COMPOSE_FILE" exec -T server /app/server --version 2>/dev/null || echo 'unknown')",
    "postgres_version": "$(docker compose -f "$COMPOSE_FILE" exec -T postgres psql --version | head -1)",
    "redis_version": "$(docker compose -f "$COMPOSE_FILE" exec -T redis redis-server --version | head -1)"
}
EOF

# Create compressed archive
log_info "Creating backup archive..."
ARCHIVE_PATH="$BACKUP_DIR/${BACKUP_NAME}.tar.gz"
tar -czf "$ARCHIVE_PATH" -C "$WORK_DIR" .

# Handle S3 upload if backup path starts with s3://
if [[ "$BACKUP_DIR" == s3://* ]]; then
    log_info "Uploading to S3..."
    if command -v aws > /dev/null 2>&1; then
        aws s3 cp "$ARCHIVE_PATH" "$BACKUP_DIR/${BACKUP_NAME}.tar.gz"
        rm "$ARCHIVE_PATH"
        log_info "Backup uploaded to S3: $BACKUP_DIR/${BACKUP_NAME}.tar.gz"
    else
        log_error "AWS CLI not installed, backup saved locally: $ARCHIVE_PATH"
    fi
else
    log_info "Backup saved: $ARCHIVE_PATH ($(du -h "$ARCHIVE_PATH" | cut -f1))"
fi

# Cleanup old backups (keep last 7 by default)
KEEP_BACKUPS="${KEEP_BACKUPS:-7}"
if [[ "$BACKUP_DIR" != s3://* ]]; then
    BACKUP_COUNT=$(ls -1 "$BACKUP_DIR"/abp_backup_*.tar.gz 2>/dev/null | wc -l)
    if [ "$BACKUP_COUNT" -gt "$KEEP_BACKUPS" ]; then
        log_info "Cleaning up old backups (keeping last $KEEP_BACKUPS)..."
        ls -1t "$BACKUP_DIR"/abp_backup_*.tar.gz | tail -n +$((KEEP_BACKUPS + 1)) | xargs rm -f
    fi
fi

log_info "Backup complete!"
