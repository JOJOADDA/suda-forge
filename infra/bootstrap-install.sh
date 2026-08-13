#!/usr/bin/env bash
set -Eeuo pipefail

REPO_URL="${SUDA_REPO_URL:-https://github.com/JOJOADDA/suda-forge.git}"
REF="${SUDA_REF:-main}"
INSTALL_DIR="${SUDA_INSTALL_DIR:-/opt/suda-forge}"
HOSTNAME_VALUE=""
SKIP_DNS_CHECK="${SUDA_SKIP_DNS_CHECK:-0}"
INSTALL_CADDY="${SUDA_INSTALL_CADDY:-1}"
INSTALL_LXC="${SUDA_INSTALL_LXC:-1}"
DB_NAME="${SUDA_DB_NAME:-suda_forge}"
DB_USER="${SUDA_DB_USER:-suda_forge}"
FROM_BOOTSTRAP=0

log() { printf '\n==> %s\n' "$*"; }
fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
usage() {
  cat <<'EOF'
Usage:
  curl -fsSL https://raw.githubusercontent.com/JOJOADDA/suda-forge/main/infra/bootstrap-install.sh \
    | sudo bash -s -- suda.example.com

Options:
  --ref REF                 Git ref to install (default: main)
  --install-dir DIR         Install directory (default: /opt/suda-forge)
  --repo-url URL            Repository URL override
  --skip-dns-check          Skip public DNS preflight while propagation is pending
  --skip-caddy              Do not install Caddy automatically
  --skip-lxc                Do not install/initialize LXC automatically
  --db-name NAME            PostgreSQL database name (default: suda_forge)
  --db-user USER            PostgreSQL role name (default: suda_forge)
EOF
}

ARGS=()
while [ "$#" -gt 0 ]; do
  case "$1" in
    --ref) REF="${2:?missing ref}"; shift 2 ;;
    --install-dir) INSTALL_DIR="${2:?missing install dir}"; shift 2 ;;
    --repo-url) REPO_URL="${2:?missing repository URL}"; shift 2 ;;
    --skip-dns-check) SKIP_DNS_CHECK=1; shift ;;
    --skip-caddy) INSTALL_CADDY=0; shift ;;
    --skip-lxc) INSTALL_LXC=0; shift ;;
    --db-name) DB_NAME="${2:?missing database name}"; shift 2 ;;
    --db-user) DB_USER="${2:?missing database user}"; shift 2 ;;
    --_from-bootstrap) FROM_BOOTSTRAP=1; shift ;;
    -h|--help) usage; exit 0 ;;
    --*) fail "unknown option: $1" ;;
    *)
      [ -z "$HOSTNAME_VALUE" ] || fail "only one public hostname is allowed"
      HOSTNAME_VALUE="$1"
      shift
      ;;
  esac
done

[ -n "$HOSTNAME_VALUE" ] || { usage >&2; fail "public hostname is required"; }
[ "$(id -u)" -eq 0 ] || fail "run with sudo on the target server"

case "$HOSTNAME_VALUE" in *[!A-Za-z0-9._-]*) fail "invalid hostname: $HOSTNAME_VALUE" ;; esac
case "$DB_NAME" in *[!A-Za-z0-9_]*|'') fail "invalid database name" ;; esac
case "$DB_USER" in *[!A-Za-z0-9_]*|'') fail "invalid database user" ;; esac

if [ "$FROM_BOOTSTRAP" -eq 0 ]; then
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-bootstrap-install.sh}")" 2>/dev/null && pwd || true)"
  if [ ! -f "$SCRIPT_DIR/install.sh" ]; then
    export DEBIAN_FRONTEND=noninteractive
    command -v git >/dev/null 2>&1 || {
      log "installing Git bootstrap dependency"
      apt-get update -qq
      apt-get install -y -qq git ca-certificates curl
    }

    if [ -e "$INSTALL_DIR" ] && [ ! -d "$INSTALL_DIR/.git" ]; then
      [ -z "$(find "$INSTALL_DIR" -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null || true)" ] || \
        fail "$INSTALL_DIR exists and is not an empty Git checkout"
    fi

    if [ -d "$INSTALL_DIR/.git" ]; then
      log "updating existing checkout at $INSTALL_DIR"
      git -C "$INSTALL_DIR" fetch --quiet --tags origin
      git -C "$INSTALL_DIR" checkout --quiet "$REF" 2>/dev/null || git -C "$INSTALL_DIR" checkout --quiet -B "$REF" "origin/$REF"
      git -C "$INSTALL_DIR" reset --hard --quiet "origin/$REF" 2>/dev/null || git -C "$INSTALL_DIR" reset --hard --quiet "$REF"
    else
      log "cloning SUDA FORGE into $INSTALL_DIR"
      mkdir -p "$(dirname "$INSTALL_DIR")"
      git clone --depth 1 --branch "$REF" "$REPO_URL" "$INSTALL_DIR"
    fi

    exec env \
      SUDA_INSTALL_DIR="$INSTALL_DIR" \
      SUDA_REPO_URL="$REPO_URL" \
      SUDA_REF="$REF" \
      SUDA_SKIP_DNS_CHECK="$SKIP_DNS_CHECK" \
      SUDA_INSTALL_CADDY="$INSTALL_CADDY" \
      SUDA_INSTALL_LXC="$INSTALL_LXC" \
      SUDA_DB_NAME="$DB_NAME" \
      SUDA_DB_USER="$DB_USER" \
      bash "$INSTALL_DIR/infra/bootstrap-install.sh" \
        "$HOSTNAME_VALUE" --_from-bootstrap
  fi
fi

SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export SUDA_INSTALL_DIR="$INSTALL_DIR"
export SUDA_SKIP_DNS_CHECK="$SKIP_DNS_CHECK"
export SUDA_INSTALL_CADDY="$INSTALL_CADDY"
export SUDA_INSTALL_LXC="$INSTALL_LXC"
export SUDA_DB_NAME="$DB_NAME"
export SUDA_DB_USER="$DB_USER"

log "step 1/5: host dependencies"
bash "$SCRIPT_ROOT/infra/steps/01-host-deps.sh"

log "step 2/5: preflight and public DNS"
bash "$SCRIPT_ROOT/infra/steps/02-preflight.sh" "$HOSTNAME_VALUE"

log "step 3/5: PostgreSQL and environment"
bash "$SCRIPT_ROOT/infra/steps/03-database.sh"

log "step 4/5: application, systemd, Caddy, and migrations"
bash "$SCRIPT_ROOT/infra/steps/04-application.sh" "$HOSTNAME_VALUE"

log "step 5/5: health gate"
bash "$SCRIPT_ROOT/infra/steps/05-health.sh" "$HOSTNAME_VALUE"
