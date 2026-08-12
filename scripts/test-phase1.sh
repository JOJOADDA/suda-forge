#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
go test ./...
if [ -n "${DATABASE_URL:-}" ]; then go test -tags=integration ./internal/postgres; else echo 'DATABASE_URL not set: PostgreSQL integration suite not run'; fi
(cd apps/web && pnpm build)
"$ROOT/scripts/check-runtime-host.sh" || true
printf '%s\n' 'Phase 1 verification completed; inspect each command result and runtime-host status separately.'
