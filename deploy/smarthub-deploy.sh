#!/bin/bash
# smarthub-deploy.sh — deploy Smarthub v3 + health-gate + otomatis rollback.
set -uo pipefail
BIN=/opt/smarthub/bin/smarthub-api
BACKUP=/opt/smarthub/backup/smarthub-api.prev
LOG=/var/log/smarthub-deploy.log
HEALTH_URL="http://127.0.0.1:8096/api/health"
MAX_RETRIES=12
RETRY_SEC=5
log() { echo "$(date '+%F %T') $*" | tee -a "$LOG"; }
if [ -f "$BIN" ]; then
  mkdir -p "$(dirname "$BACKUP")"
  cp -f "$BIN" "$BACKUP"
  log "backup binary -> $BACKUP"
fi
if [ ! -f "${1:-}" ]; then echo "usage: $0 <path-to-new-binary>"; exit 2; fi
cp -f "$1" "$BIN"
chmod 755 "$BIN"
log "binary baru terpasang: $1"
systemctl daemon-reload && systemctl restart smarthub-api
log "service reloaded"
for i in $(seq 1 $MAX_RETRIES); do
  sleep "$RETRY_SEC"
  code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 6 "$HEALTH_URL" 2>/dev/null)
  log "health attempt $i: HTTP $code"
  if [ "$code" = "200" ]; then log "HEALTH_OK"; exit 0; fi
done
log "HEALTH_FAIL — rollback"
if [ -f "$BACKUP" ]; then
  cp -f "$BACKUP" "$BIN"
  chmod 755 "$BIN"
  systemctl restart smarthub-api
  sleep "$RETRY_SEC"
  rb=$(curl -s -o /dev/null -w '%{http_code}' --max-time 6 "$HEALTH_URL" 2>/dev/null)
  log "setelah rollback: HTTP $rb"
  [ "$rb" = "200" ] && log "ROLLBACK_OK" && exit 0
  log "ROLLBACK_JUGA_GAGAL"; exit 1
fi
log "TIDAK ADA BACKUP"; exit 1