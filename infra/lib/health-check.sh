#!/usr/bin/env bash
set -Eeuo pipefail

URL="${1:-http://127.0.0.1:8080/healthz}"
TIMEOUT="${SUDA_HEALTH_TIMEOUT:-30}"
SERVICE_NAME="${SUDA_SERVICE_NAME:-suda-forge}"

for ((attempt=1; attempt<=TIMEOUT; attempt++)); do
  if curl -fsS --max-time 2 "$URL" >/tmp/suda-forge-health.json 2>/tmp/suda-forge-health.err; then
    printf 'health OK: %s\n' "$URL"
    cat /tmp/suda-forge-health.json
    exit 0
  fi
  sleep 1
done

echo "health check failed: $URL" >&2
if command -v systemctl >/dev/null 2>&1; then
  systemctl --no-pager --full status "$SERVICE_NAME" >&2 || true
fi
if command -v journalctl >/dev/null 2>&1; then
  journalctl -u "$SERVICE_NAME" -n 40 --no-pager >&2 || true
fi
cat /tmp/suda-forge-health.err >&2 || true
exit 1
