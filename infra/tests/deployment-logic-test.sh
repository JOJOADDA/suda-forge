#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

mkdir -p "$TMP/releases/old/infra/templates" "$TMP/new/infra/templates" "$TMP/etc" "$TMP/state"
cat > "$TMP/releases/old/infra/templates/suda-forge.service.tmpl" <<'EOF'
[Service]
WorkingDirectory={{INSTALL_DIR}}
ExecStart={{INSTALL_DIR}}/bin/suda-forge
EOF
cp "$TMP/releases/old/infra/templates/suda-forge.service.tmpl" "$TMP/new/infra/templates/suda-forge.service.tmpl"
mkdir -p "$TMP/releases/old/bin" "$TMP/new/bin"
printf '#!/bin/sh\n' > "$TMP/releases/old/bin/suda-forge"
printf '#!/bin/sh\n' > "$TMP/new/bin/suda-forge"
chmod +x "$TMP/releases/old/bin/suda-forge" "$TMP/new/bin/suda-forge"
ln -s "$TMP/releases/old" "$TMP/current"
printf 'DATABASE_URL=postgres://test\n' > "$TMP/etc/suda-forge.env"
printf '#!/usr/bin/env bash\nexit 0\n' > "$TMP/health-ok.sh"
printf '#!/usr/bin/env bash\nexit 1\n' > "$TMP/health-fail.sh"
printf '#!/usr/bin/env bash\nexit 0\n' > "$TMP/systemctl"
chmod +x "$TMP/health-ok.sh" "$TMP/health-fail.sh" "$TMP/systemctl"

(cd "$TMP/new" && tar -czf "$TMP/release-ok.tar.gz" .)
sha256sum "$TMP/release-ok.tar.gz" > "$TMP/release-ok.tar.gz.sha256"
sudo -n env \
  SUDA_ENV_DIR="$TMP/etc" \
  SUDA_SERVICE_FILE="$TMP/service" \
  SUDA_STATE_DIR="$TMP/state" \
  SUDA_SYSTEMCTL_CMD="$TMP/systemctl" \
  SUDA_HEALTH_CHECK_SCRIPT="$TMP/health-ok.sh" \
  bash "$ROOT/infra/remote-activate.sh" \
  "$TMP/release-ok.tar.gz" release-ok "$TMP/current" "$TMP/releases" suda-forge test.local http://health 1 3 >/dev/null
[ "$(readlink -f "$TMP/current")" = "$TMP/releases/release-ok" ]

tar -C "$TMP/new" -czf "$TMP/release-fail.tar.gz" .
sha256sum "$TMP/release-fail.tar.gz" > "$TMP/release-fail.tar.gz.sha256"
if sudo -n env \
  SUDA_ENV_DIR="$TMP/etc" \
  SUDA_SERVICE_FILE="$TMP/service" \
  SUDA_STATE_DIR="$TMP/state" \
  SUDA_SYSTEMCTL_CMD="$TMP/systemctl" \
  SUDA_HEALTH_CHECK_SCRIPT="$TMP/health-fail.sh" \
  bash "$ROOT/infra/remote-activate.sh" \
  "$TMP/release-fail.tar.gz" release-fail "$TMP/current" "$TMP/releases" suda-forge test.local http://health 1 3 >/dev/null 2>&1; then
  echo 'expected health-gate failure' >&2
  exit 1
fi
[ "$(readlink -f "$TMP/current")" = "$TMP/releases/release-ok" ]

echo 'deployment logic tests passed'
