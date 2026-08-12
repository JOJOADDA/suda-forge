#!/usr/bin/env bash
set -Eeuo pipefail

HOSTNAME_VALUE="${1:-${SUDA_HOSTNAME:-}}"
SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_DIR="${SUDA_INSTALL_DIR:-/opt/suda-forge}"
SERVICE_NAME="suda-forge"
SERVICE_USER="root"
ENV_DIR="/etc/suda-forge"
ENV_FILE="$ENV_DIR/suda-forge.env"
SERVICE_TEMPLATE="$SCRIPT_ROOT/infra/templates/suda-forge.service.tmpl"
CADDY_TEMPLATE="$SCRIPT_ROOT/infra/templates/Caddyfile.tmpl"

log() { printf '\n==> %s\n' "$*"; }
fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

[ "$EUID" -eq 0 ] || fail "run as root: sudo bash infra/install.sh <hostname>"
[ -n "$HOSTNAME_VALUE" ] || fail "hostname is required"

if [ "${SUDA_SKIP_HOST_DEPS:-0}" != "1" ]; then
  log "host dependencies"
  bash "$SCRIPT_ROOT/infra/host-deps.sh"
fi

log "preflight"
SUDA_SKIP_DNS_CHECK="${SUDA_SKIP_DNS_CHECK:-0}" bash "$SCRIPT_ROOT/infra/preflight.sh" "$HOSTNAME_VALUE"

if [ "$SCRIPT_ROOT" != "$INSTALL_DIR" ]; then
  log "syncing source into $INSTALL_DIR"
  mkdir -p "$INSTALL_DIR"
  cp -a "$SCRIPT_ROOT"/. "$INSTALL_DIR"/
fi

mkdir -p "$ENV_DIR" /var/lib/suda-forge/deployments /var/lib/suda-forge/runtime
chmod 0750 "$ENV_DIR" /var/lib/suda-forge

if [ ! -f "$ENV_FILE" ]; then
  log "creating environment file"
  cp "$INSTALL_DIR/infra/templates/suda-forge.env.example" "$ENV_FILE"
  chmod 0600 "$ENV_FILE"
  printf 'Edit %s before production use, especially DATABASE_URL.\n' "$ENV_FILE"
fi

if [ "${SUDA_SKIP_MIGRATIONS:-0}" != "1" ]; then
  log "applying database migrations"
  DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' "$ENV_FILE" | tail -1)" \
    bash "$INSTALL_DIR/scripts/migrate.sh"
fi

log "building frontend"
(cd "$INSTALL_DIR/apps/web" && pnpm install --frozen-lockfile --prod=false && pnpm build)

log "building backend"
mkdir -p "$INSTALL_DIR/bin"
(cd "$INSTALL_DIR" && go build -trimpath -ldflags="-s -w" -o "$INSTALL_DIR/bin/suda-forge" ./cmd/server)
chmod 0755 "$INSTALL_DIR/bin/suda-forge"

log "installing systemd unit"
sed "s#{{INSTALL_DIR}}#$INSTALL_DIR#g" "$SERVICE_TEMPLATE" > "/etc/systemd/system/${SERVICE_NAME}.service"
chmod 0644 "/etc/systemd/system/${SERVICE_NAME}.service"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME" >/dev/null
systemctl restart "$SERVICE_NAME"

if command -v caddy >/dev/null 2>&1; then
  log "installing Caddy configuration"
  mkdir -p /etc/caddy
  sed -e "s#{{HOSTNAME}}#$HOSTNAME_VALUE#g" -e "s#{{INSTALL_DIR}}#$INSTALL_DIR#g" "$CADDY_TEMPLATE" > /etc/caddy/Caddyfile
  caddy validate --config /etc/caddy/Caddyfile
  systemctl enable caddy >/dev/null 2>&1 || true
  systemctl reload caddy || systemctl restart caddy
else
  printf 'WARN: Caddy is not installed; install it and render %s before exposing the service.\n' "$CADDY_TEMPLATE" >&2
fi

log "health check"
bash "$INSTALL_DIR/infra/lib/health-check.sh" "http://127.0.0.1:8080/healthz"

cat <<EOF

SUDA FORGE installation complete.
  App:      https://$HOSTNAME_VALUE
  Service:  systemctl status $SERVICE_NAME
  Logs:     journalctl -u $SERVICE_NAME -f
  Env:      $ENV_FILE
  Install:  $INSTALL_DIR
EOF
