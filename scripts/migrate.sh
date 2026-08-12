#!/usr/bin/env bash
set -Eeuo pipefail

DATABASE_URL="${DATABASE_URL:-}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
[ -n "$DATABASE_URL" ] || { echo 'DATABASE_URL is required' >&2; exit 1; }
command -v psql >/dev/null || { echo 'psql is required' >&2; exit 1; }

psql "$DATABASE_URL" -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
);
SQL

while IFS= read -r migration; do
  version="$(basename "$migration" .sql)"
  applied="$(psql "$DATABASE_URL" -Atqc "SELECT 1 FROM schema_migrations WHERE version = '$version' LIMIT 1")"
  if [ "$applied" = "1" ]; then
    continue
  fi
  echo "applying $version"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$migration"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations(version) VALUES ('$version')"
done < <(find "$ROOT/migrations" -maxdepth 1 -type f -name '*.sql' -printf '%f\n' | sort | sed "s#^#$ROOT/migrations/#")

echo 'migrations complete'
