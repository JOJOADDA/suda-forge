#!/usr/bin/env bash
set -Eeuo pipefail

ARTIFACT_PATH="${1:?artifact path required}"
RELEASE_ID="${2:?release id required}"
CURRENT_DIR="${3:?current dir required}"
RELEASE_ROOT="${4:?release root required}"
SERVICE_NAME="${5:?service name required}"
HOSTNAME_VALUE="${6:?hostname required}"
HEALTH_URL="${7:?health url required}"
SKIP_MIGRATIONS="${8:-0}"
KEEP_RELEASES="${9:-3}"
ENV_DIR="/etc/suda-forge"
ENV_FILE="$ENV_DIR/suda-forge.env"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
CADDY_FILE="/etc/caddy/Caddyfile"
RELEASE_DIR="$RELEASE_ROOT/$RELEASE_ID"
PREVIOUS_TARGET=""
CADDY_BACKUP=""

log() { printf '\n==> %s\n' "$*"; }
fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || fail "remote activation must run as root"
[ -f "$ARTIFACT_PATH" ] || fail "artifact not found: $ARTIFACT_PATH"
[ -f "$ENV_FILE" ] || fail "missing $ENV_FILE; run infra/install.sh once or create it from the example"
mkdir -p "$RELEASE_ROOT" "$ENV_DIR" /var/lib/suda-forge/deployments /var/lib/suda-forge/runtime

if [ -L "$CURRENT_DIR" ] || [ -d "$CURRENT_DIR" ]; then
  PREVIOUS_TARGET="$(readlink -f "$CURRENT_DIR" 2>/dev/null || true)"
  if [ -n "$PREVIOUS_TARGET" ] && [ -d "$CURRENT_DIR" ] && [ ! -L "$CURRENT_DIR" ]; then
    LEGACY_TARGET="$RELEASE_ROOT/legacy-$(date -u +%Y%m%d%H%M%S)"
    mv "$CURRENT_DIR" "$LEGACY_TARGET"
    PREVIOUS_TARGET="$LEGACY_TARGET"
  fi
fi

rollback() {
  set +e
  printf '\nERROR: deployment failed; attempting rollback\n' >&2
  systemctl stop "$SERVICE_NAME" >/dev/null 2>&1 || true
  if [ -n "$PREVIOUS_TARGET" ] && [ -d "$PREVIOUS_TARGET" ]; then
    ln -sfn "$PREVIOUS_TARGET" "$CURRENT_DIR"
    sed "s#{{INSTALL_DIR}}#$CURRENT_DIR#g" "$PREVIOUS_TARGET/infra/templates/suda-forge.service.tmpl" > "$SERVICE_FILE"
    systemctl daemon-reload
    systemctl restart "$SERVICE_NAME" >/dev/null 2>&1 || true
    if [ -n "$CADDY_BACKUP" ] && [ -f "$CADDY_BACKUP" ]; then
      cp "$CADDY_BACKUP" "$CADDY_FILE"
      caddy validate --config "$CADDY_FILE" >/dev/null 2>&1 && systemctl reload caddy >/dev/null 2>&1 || true
    fi
    printf 'rollback target: %s\n' "$PREVIOUS_TARGET" >&2
  else
    printf 'no previous release available; service remains stopped\n' >&2
  fi
  exit 1
}
trap rollback ERR

log "verifying artifact checksum"
if [ -f "${ARTIFACT_PATH}.sha256" ]; then
  (cd "$(dirname "$ARTIFACT_PATH")" && sha256sum -c "$(basename "$ARTIFACT_PATH").sha256")
fi

log "extracting release $RELEASE_ID"
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"
tar -xzf "$ARTIFACT_PATH" -C "$RELEASE_DIR"
chmod 0755 "$RELEASE_DIR/bin/suda-forge"

if [ "$SKIP_MIGRATIONS" != "1" ]; then
  log "applying database migrations"
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
  DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}" bash "$RELEASE_DIR/scripts/migrate.sh"
fi

log "switching current release"
ln -sfn "$RELEASE_DIR" "$CURRENT_DIR"

log "installing systemd unit"
sed "s#{{INSTALL_DIR}}#$CURRENT_DIR#g" "$RELEASE_DIR/infra/templates/suda-forge.service.tmpl" > "$SERVICE_FILE"
chmod 0644 "$SERVICE_FILE"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME" >/dev/null
systemctl restart "$SERVICE_NAME"

if command -v caddy >/dev/null 2>&1 && [ -f "$RELEASE_DIR/infra/templates/Caddyfile.tmpl" ]; then
  log "validating Caddy configuration"
  mkdir -p /etc/caddy
  CADDY_BACKUP="/etc/caddy/Caddyfile.suda-forge.previous.$RELEASE_ID"
  [ -f "$CADDY_FILE" ] && cp "$CADDY_FILE" "$CADDY_BACKUP" || true
  sed -e "s#{{HOSTNAME}}#$HOSTNAME_VALUE#g" -e "s#{{INSTALL_DIR}}#$CURRENT_DIR#g" "$RELEASE_DIR/infra/templates/Caddyfile.tmpl" > "$CADDY_FILE"
  caddy validate --config "$CADDY_FILE"
  systemctl enable caddy >/dev/null 2>&1 || true
  systemctl reload caddy || systemctl restart caddy
fi

log "health gate"
SUDA_SERVICE_NAME="$SERVICE_NAME" bash "$CURRENT_DIR/infra/lib/health-check.sh" "$HEALTH_URL"

log "retaining last $KEEP_RELEASES releases"
find "$RELEASE_ROOT" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort -r | tail -n +$((KEEP_RELEASES + 1)) | while read -r old; do
  [ -n "$old" ] && [ "$RELEASE_ROOT/$old" != "$RELEASE_DIR" ] && rm -rf "$RELEASE_ROOT/$old"
done
rm -f "$ARTIFACT_PATH" "${ARTIFACT_PATH}.sha256"
trap - ERR
printf '\nSUDA FORGE release active: %s\n' "$RELEASE_ID"
