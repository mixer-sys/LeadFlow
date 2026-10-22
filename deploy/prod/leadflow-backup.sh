#!/usr/bin/env bash
set -euo pipefail

# ============================================
# CONFIG
# ============================================

TIMESTAMP="$(date +%F_%H-%M-%S)"
BACKUP_DIR="/var/backups/leadflow"
CONTAINER="leadflow-postgres"
DB_USER="leadflow"
DB_NAME="leadflow"
RETENTION_DAYS=14

# Remote backup
REMOTE_ENABLED=true
REMOTE_USER="backup"
REMOTE_HOST="backup-server"
REMOTE_DIR="/var/backups/leadflow"

# ============================================
# PREPARE
# ============================================

mkdir -p "$BACKUP_DIR"

BACKUP_FILE="$BACKUP_DIR/leadflow_${TIMESTAMP}.dump"
CHECKSUM_FILE="$BACKUP_FILE.sha256"

log() {
  echo "[$(date '+%F %T')] $*"
}

fail() {
  echo "[$(date '+%F %T')] ERROR: $*" >&2
  exit 1
}

# ============================================
# 1. CREATE BACKUP
# ============================================

log "Starting PostgreSQL backup"

docker exec "$CONTAINER" \
  pg_dump \
  -U "$DB_USER" \
  -d "$DB_NAME" \
  -Fc \
  > "$BACKUP_FILE"

if [ ! -s "$BACKUP_FILE" ]; then
  fail "Backup file is empty: $BACKUP_FILE"
fi

log "Backup created: $BACKUP_FILE"

# ============================================
# 2. VERIFY BACKUP IS VALID
# ============================================

log "Verifying backup archive"

if ! docker exec -i "$CONTAINER" \
  pg_restore \
  --list \
  < "$BACKUP_FILE" \
  > /dev/null; then
  fail "Backup archive is invalid: $BACKUP_FILE"
fi

log "Backup archive is valid"

# ============================================
# 3. CREATE CHECKSUM
# ============================================

log "Creating checksum"

sha256sum "$BACKUP_FILE" > "$CHECKSUM_FILE"

LOCAL_CHECKSUM="$(awk '{print $1}' "$CHECKSUM_FILE")"

log "Local checksum: $LOCAL_CHECKSUM"

# ============================================
# 4. SEND TO REMOTE BACKUP SERVER
# ============================================

if [ "$REMOTE_ENABLED" = true ]; then
  log "Sending backup to remote server"

  ssh "$REMOTE_USER@$REMOTE_HOST" \
    "mkdir -p '$REMOTE_DIR'"

  rsync -az \
    "$BACKUP_FILE" \
    "$CHECKSUM_FILE" \
    "$REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/"

  log "Backup transferred to remote server"

  # ============================================
  # 5. VERIFY REMOTE BACKUP
  # ============================================

  log "Verifying remote backup checksum"

  REMOTE_CHECKSUM="$(ssh "$REMOTE_USER@$REMOTE_HOST" \
    "awk '{print \$1}' '$REMOTE_DIR/$(basename "$CHECKSUM_FILE")'")"

  if [ "$LOCAL_CHECKSUM" != "$REMOTE_CHECKSUM" ]; then
    fail "Remote checksum mismatch: local=$LOCAL_CHECKSUM remote=$REMOTE_CHECKSUM"
  fi

  log "Remote backup checksum verified"

  # ============================================
  # 6. VERIFY REMOTE ARCHIVE
  # ============================================

  log "Verifying remote backup archive"

  if ! ssh "$REMOTE_USER@$REMOTE_HOST" \
    "docker exec -i $CONTAINER pg_restore --list < '$REMOTE_DIR/$(basename "$BACKUP_FILE")' > /dev/null"; then
    fail "Remote backup archive is invalid"
  fi

  log "Remote backup archive verified"
fi

# ============================================
# 7. CLEANUP OLD BACKUPS
# ============================================

log "Cleaning old local backups"

find "$BACKUP_DIR" \
  -name 'leadflow_*.dump' \
  -type f \
  -mtime +"$RETENTION_DAYS" \
  -delete

find "$BACKUP_DIR" \
  -name 'leadflow_*.dump.sha256' \
  -type f \
  -mtime +"$RETENTION_DAYS" \
  -delete

log "Backup completed successfully"