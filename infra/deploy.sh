#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_HOST="${SUDA_DEPLOY_HOST:-}"
DEPLOY_USER="${SUDA_DEPLOY_USER:-ubuntu}"
DEPLOY_PORT="${SUDA_DEPLOY_PORT:-22}"
REMOTE_DIR="${SUDA_REMOTE_DIR:-/opt/suda-forge}"
RELEASE_ROOT="${SUDA_RELEASE_ROOT:-/opt/suda-forge-releases}"
SERVICE_NAME="${SUDA_SERVICE_NAME:-suda-forge}"
HOSTNAME_VALUE="${SUDA_HOSTNAME:-}"
HEALTH_URL="${SUDA_HEALTH_URL:-http://127.0.0.1:8080/healthz}"
DRY_RUN=0
SKIP_TESTS="${SUDA_SKIP_TESTS:-0}"
SKIP_MIGRATIONS="${SUDA_SKIP_MIGRATIONS:-0}"
KEEP_RELEASES="${SUDA_KEEP_RELEASES:-3}"
REF="HEAD"

log() { printf '\n==> %s\n' "$*"; }
fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
usage() {
  cat <<'EOF'
Usage: infra/deploy.sh --host deploy.example.com [options]

Required:
  --host HOST              SSH host or IP address

Options:
  --user USER              SSH user (default: ubuntu)
  --port PORT              SSH port (default: 22)
  --ref REF                Git ref to deploy (default: HEAD)
  --hostname HOSTNAME      Public hostname used by Caddy
  --remote-dir DIR        Current release symlink (default: /opt/suda-forge)
  --dry-run                Build and validate without uploading or activating
  --skip-tests             Skip go test and frontend typecheck
  --skip-migrations        Do not apply database migrations on the server
  --keep-releases N        Number of old releases to retain (default: 3)

Environment variables mirror the options with SUDA_ prefixes.
SSH keys and sudo access must already be configured on the target server.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --host) DEPLOY_HOST="${2:?missing host}"; shift 2 ;;
    --user) DEPLOY_USER="${2:?missing user}"; shift 2 ;;
    --port) DEPLOY_PORT="${2:?missing port}"; shift 2 ;;
    --ref) REF="${2:?missing ref}"; shift 2 ;;
    --hostname) HOSTNAME_VALUE="${2:?missing hostname}"; shift 2 ;;
    --remote-dir) REMOTE_DIR="${2:?missing remote dir}"; shift 2 ;;
    --dry-run) DRY_RUN=1; shift ;;
    --skip-tests) SKIP_TESTS=1; shift ;;
    --skip-migrations) SKIP_MIGRATIONS=1; shift ;;
    --keep-releases) KEEP_RELEASES="${2:?missing release count}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) fail "unknown option: $1" ;;
  esac
done

if [ -z "$DEPLOY_HOST" ] && [ "$DRY_RUN" != "1" ]; then
  usage >&2
  fail "--host is required unless --dry-run is used"
fi
[ -n "$DEPLOY_HOST" ] || DEPLOY_HOST="dry-run.local"
[ -n "$HOSTNAME_VALUE" ] || HOSTNAME_VALUE="$DEPLOY_HOST"
case "$DEPLOY_PORT" in ''|*[!0-9]*) fail "SSH port must be numeric" ;; esac
case "$KEEP_RELEASES" in ''|*[!0-9]*) fail "--keep-releases must be a non-negative integer" ;; esac

command -v git >/dev/null 2>&1 || fail "git is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"
command -v sha256sum >/dev/null 2>&1 || fail "sha256sum is required"
command -v ssh >/dev/null 2>&1 || fail "ssh is required"
command -v scp >/dev/null 2>&1 || fail "scp is required"

cd "$ROOT_DIR"
[ -z "$(git status --porcelain)" ] || fail "working tree is dirty; commit changes before deployment"
COMMIT_SHA="$(git rev-parse "$REF")"
SHORT_SHA="$(git rev-parse --short "$COMMIT_SHA")"
RELEASE_ID="$(date -u +%Y%m%d%H%M%S)-$SHORT_SHA"

log "preflight: commit $COMMIT_SHA"
[ -x "$ROOT_DIR/infra/preflight.sh" ] || fail "infra/preflight.sh is missing or not executable"
[ -x "$ROOT_DIR/scripts/migrate.sh" ] || fail "scripts/migrate.sh is missing or not executable"

if [ "$SKIP_TESTS" != "1" ]; then
  log "backend tests"
  go test ./...
  log "frontend typecheck and build"
  (cd "$ROOT_DIR/apps/web" && pnpm exec tsc --noEmit && pnpm build)
else
  log "tests skipped by SUDA_SKIP_TESTS/--skip-tests"
  (cd "$ROOT_DIR/apps/web" && pnpm build)
fi

STAGE_DIR="$(mktemp -d)"
cleanup() { rm -rf "$STAGE_DIR"; }
trap cleanup EXIT
mkdir -p "$STAGE_DIR/release/bin"

log "building backend artifact"
go build -trimpath -ldflags="-s -w" -o "$STAGE_DIR/release/bin/suda-forge" ./cmd/server
chmod 0755 "$STAGE_DIR/release/bin/suda-forge"

log "assembling release $RELEASE_ID"
tar -C "$ROOT_DIR" --exclude='./.git' --exclude='./apps/web/node_modules' --exclude='./apps/web/dist' --exclude='./bin' -cf - . | tar -C "$STAGE_DIR/release" -xf -
cp -a "$ROOT_DIR/apps/web/dist" "$STAGE_DIR/release/apps/web/dist"
printf '%s\n' "$COMMIT_SHA" > "$STAGE_DIR/release/RELEASE_COMMIT"
printf '%s\n' "$RELEASE_ID" > "$STAGE_DIR/release/RELEASE_ID"
printf '%s  %s\n' "$COMMIT_SHA" "$RELEASE_ID" > "$STAGE_DIR/release/RELEASE_METADATA"
(cd "$STAGE_DIR/release" && tar -czf "$STAGE_DIR/suda-forge-$RELEASE_ID.tar.gz" .)
sha256sum "$STAGE_DIR/suda-forge-$RELEASE_ID.tar.gz" | tee "$STAGE_DIR/suda-forge-$RELEASE_ID.tar.gz.sha256"

if [ "$DRY_RUN" = "1" ]; then
  log "dry-run complete"
  printf 'artifact: %s\n' "$STAGE_DIR/suda-forge-$RELEASE_ID.tar.gz"
  printf 'commit:   %s\n' "$COMMIT_SHA"
  exit 0
fi

ARTIFACT="$STAGE_DIR/suda-forge-$RELEASE_ID.tar.gz"
CHECKSUM="$ARTIFACT.sha256"
REMOTE_TMP="/tmp/suda-forge-$RELEASE_ID"

log "checking SSH connectivity"
ssh -p "$DEPLOY_PORT" -o BatchMode=yes -o ConnectTimeout=10 "$DEPLOY_USER@$DEPLOY_HOST" true
log "uploading release artifact"
scp -P "$DEPLOY_PORT" "$ARTIFACT" "$DEPLOY_USER@$DEPLOY_HOST:$REMOTE_TMP.tar.gz"
scp -P "$DEPLOY_PORT" "$CHECKSUM" "$DEPLOY_USER@$DEPLOY_HOST:$REMOTE_TMP.tar.gz.sha256"

log "activating release on $DEPLOY_HOST"
ssh -p "$DEPLOY_PORT" "$DEPLOY_USER@$DEPLOY_HOST" sudo -n bash -s -- \
  "$REMOTE_TMP.tar.gz" "$RELEASE_ID" "$REMOTE_DIR" "$RELEASE_ROOT" "$SERVICE_NAME" "$HOSTNAME_VALUE" "$HEALTH_URL" "$SKIP_MIGRATIONS" "$KEEP_RELEASES" \
  < "$ROOT_DIR/infra/remote-activate.sh"

log "deployment complete"
printf 'commit:  %s\n' "$COMMIT_SHA"
printf 'release: %s\n' "$RELEASE_ID"
printf 'server:  %s@%s:%s\n' "$DEPLOY_USER" "$DEPLOY_HOST" "$DEPLOY_PORT"
