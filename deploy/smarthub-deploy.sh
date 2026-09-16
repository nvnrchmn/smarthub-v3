#!/bin/bash
# smarthub-deploy.sh — health-gated restart + auto rollback.
# Dijalanin dari CI sesudah rsync binary baru ke /opt/smarthub/bin/.
set -uo pipefail
BIN=/opt/smarthub/bin/smarthub-api
BACKUP=/opt/smarthub/bin/smarthub-api.bak
LOG=/var/log/smarthub-deploy.log
MAX_RETRIES=12
RETRY_SEC=5
log() { echo "$(date '+%F %T') $*" | tee -a "$LOG"; }
if [ ! -f "$BIN" ]; then log "BINARY_TIDAK_ADA: $BIN"; exit 2; fi
cp -f "$BIN" "$BACKUP"
log "backup binary -> $BACKUP"
systemctl daemon-reload && systemctl restart smarthub-api
log "service restarted"
for i in $(seq 1 $MAX_RETRIES); do
  sleep "$RETRY_SEC"
  code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 6 http://127.0.0.1:8082/api/health 2>/dev/null)
  log "health attempt $i: HTTP $code"
  if [ "$code" = "200" ]; then log "HEALTH_OK"; exit 0; fi
done
log "HEALTH_FAIL — rollback"
cp -f "$BACKUP" "$BIN"
systemctl restart smarthub-api
sleep "$RETRY_SEC"
rb=$(curl -s -o /dev/null -w '%{http_code}' --max-time 6 http://127.0.0.1:8082/api/health 2>/dev/null)
log "setelah rollback: HTTP $rb"
[ "$rb" = "200" ] && log "ROLLBACK_OK" && exit 0
log "ROLLBACK_JUGA_GAGAL"; exit 1
