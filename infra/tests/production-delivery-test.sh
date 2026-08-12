#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

bash -n infra/install.sh infra/host-deps.sh infra/preflight.sh infra/lib/health-check.sh scripts/migrate.sh
for file in \
  infra/install.sh \
  infra/host-deps.sh \
  infra/preflight.sh \
  infra/lib/health-check.sh \
  scripts/migrate.sh \
  infra/templates/suda-forge.env.example \
  infra/templates/suda-forge.service.tmpl \
  infra/templates/Caddyfile.tmpl; do
  test -f "$file" || { echo "missing $file" >&2; exit 1; }
done

test "$(grep -c "VITE_API_URL.*import.meta.env.DEV" apps/web/src/App.tsx)" -eq 1

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
sed 's#{{INSTALL_DIR}}#/opt/suda-forge#g' infra/templates/suda-forge.service.tmpl > "$tmpdir/suda-forge.service"
sed -e 's#{{HOSTNAME}}#suda.example.com#g' -e 's#{{INSTALL_DIR}}#/opt/suda-forge#g' infra/templates/Caddyfile.tmpl > "$tmpdir/Caddyfile"
! grep -q '{{' "$tmpdir/suda-forge.service" "$tmpdir/Caddyfile"
grep -q 'ExecStart=/opt/suda-forge/bin/suda-forge' "$tmpdir/suda-forge.service"
grep -q 'reverse_proxy @backend 127.0.0.1:8080' "$tmpdir/Caddyfile"
grep -q 'root \* /opt/suda-forge/apps/web/dist' "$tmpdir/Caddyfile"

echo 'production delivery tests passed'
