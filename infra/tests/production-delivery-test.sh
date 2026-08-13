#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

bash -n infra/install.sh infra/bootstrap-install.sh infra/deploy.sh infra/remote-activate.sh infra/check-network-and-push.sh infra/steps/*.sh infra/tests/deployment-logic-test.sh infra/host-deps.sh infra/preflight.sh infra/lib/health-check.sh scripts/migrate.sh
for file in \
  infra/install.sh \
  infra/bootstrap-install.sh \
  infra/steps/01-host-deps.sh \
  infra/steps/02-preflight.sh \
  infra/steps/03-database.sh \
  infra/steps/04-application.sh \
  infra/steps/05-health.sh \
  infra/deploy.sh \
  infra/remote-activate.sh \
  infra/check-network-and-push.sh \
  infra/tests/deployment-logic-test.sh \
  infra/host-deps.sh \
  infra/preflight.sh \
  infra/lib/health-check.sh \
  scripts/migrate.sh \
  infra/templates/suda-forge.env.example \
  infra/templates/suda-forge.service.tmpl \
  infra/templates/Caddyfile.tmpl \
  apps/web/vite.config.ts; do
  test -f "$file" || { echo "missing $file" >&2; exit 1; }
done

grep -q "const API = import.meta.env.VITE_API_URL ?? ''" apps/web/src/App.tsx
grep -q "credentials: 'include'" apps/web/src/App.tsx
grep -q "'/auth'" apps/web/vite.config.ts

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
sed 's#{{INSTALL_DIR}}#/opt/suda-forge#g' infra/templates/suda-forge.service.tmpl > "$tmpdir/suda-forge.service"
sed -e 's#{{HOSTNAME}}#suda.example.com#g' -e 's#{{INSTALL_DIR}}#/opt/suda-forge#g' infra/templates/Caddyfile.tmpl > "$tmpdir/Caddyfile"
! grep -q '{{' "$tmpdir/suda-forge.service" "$tmpdir/Caddyfile"
grep -q 'ExecStart=/opt/suda-forge/bin/suda-forge' "$tmpdir/suda-forge.service"
grep -q 'reverse_proxy @backend 127.0.0.1:8080' "$tmpdir/Caddyfile"
grep -q '/auth/\*' "$tmpdir/Caddyfile"
grep -q 'root \* /opt/suda-forge/apps/web/dist' "$tmpdir/Caddyfile"

echo 'production delivery tests passed'
