#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_DIR="${SUDA_ENV_DIR:-/etc/suda-forge}"
ENV_FILE="$ENV_DIR/suda-forge.env"
DB_NAME="${SUDA_DB_NAME:-suda_forge}"
DB_USER="${SUDA_DB_USER:-suda_forge}"

log() { printf '\n==> %s\n' "$*"; }
fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
pg_as_postgres() { runuser -u postgres -- "$@"; }

[ "$(id -u)" -eq 0 ] || fail "database bootstrap must run as root"
command -v apt-get >/dev/null 2>&1 || fail "apt-get is required"
mkdir -p "$ENV_DIR"
chmod 0750 "$ENV_DIR"
umask 077

postgres_service_present=0
if command -v systemctl >/dev/null 2>&1 && systemctl list-unit-files postgresql.service 2>/dev/null | grep -q '^postgresql\.service'; then
  postgres_service_present=1
fi

if [ "$postgres_service_present" -eq 0 ]; then
  log "installing PostgreSQL server (client tools are not sufficient)"
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq
  apt-get install -y -qq postgresql postgresql-contrib postgresql-client
fi

command -v psql >/dev/null 2>&1 || fail "psql is not installed after PostgreSQL setup"
command -v pg_isready >/dev/null 2>&1 || fail "pg_isready is not installed after PostgreSQL setup"

systemctl enable --now postgresql
for _ in $(seq 1 30); do
  pg_isready -q && break
  sleep 1
done
pg_isready -q || fail "PostgreSQL did not become ready"

if [ ! -f "$ENV_FILE" ]; then
  cp "$ROOT_DIR/infra/templates/suda-forge.env.example" "$ENV_FILE"
  chmod 0600 "$ENV_FILE"
fi

current_url="$(sed -n 's/^DATABASE_URL=//p' "$ENV_FILE" | tail -1 || true)"
if [ -n "$current_url" ] && [[ "$current_url" != *change-me* ]]; then
  log "preserving existing DATABASE_URL"
  exit 0
fi

command -v openssl >/dev/null 2>&1 || apt-get install -y -qq openssl
DB_PASSWORD="$(openssl rand -hex 24)"
DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@127.0.0.1:5432/${DB_NAME}?sslmode=disable"

log "creating PostgreSQL role and database"
role_exists="$(pg_as_postgres psql -tAc "SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = '$DB_USER'" | tr -d '[:space:]')"
if [ "$role_exists" = "1" ]; then
  pg_as_postgres psql -v ON_ERROR_STOP=1 \
    -c "ALTER ROLE \"$DB_USER\" LOGIN PASSWORD '$DB_PASSWORD';"
else
  pg_as_postgres psql -v ON_ERROR_STOP=1 \
    -c "CREATE ROLE \"$DB_USER\" LOGIN PASSWORD '$DB_PASSWORD';"
fi

if ! pg_as_postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1; then
  pg_as_postgres createdb -O "$DB_USER" "$DB_NAME"
fi

python3 - "$ENV_FILE" "$DB_URL" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
url = sys.argv[2]
lines = path.read_text().splitlines()
out = []
replaced = False
for line in lines:
    if line.startswith("DATABASE_URL="):
        out.append("DATABASE_URL=" + url)
        replaced = True
    else:
        out.append(line)
if not replaced:
    out.insert(0, "DATABASE_URL=" + url)
path.write_text("\n".join(out) + "\n")
PY
chmod 0600 "$ENV_FILE"
printf 'PostgreSQL ready; DATABASE_URL configured in %s\n' "$ENV_FILE"
