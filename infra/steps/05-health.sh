#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HOSTNAME_VALUE="${1:?public hostname is required}"
INSTALL_DIR="${SUDA_INSTALL_DIR:-/opt/suda-forge}"

SUDA_SERVICE_NAME=suda-forge \
  bash "$ROOT_DIR/infra/lib/health-check.sh" "http://127.0.0.1:8080/healthz"

cat <<EOF

SUDA FORGE bootstrap completed.
  App:       https://${HOSTNAME_VALUE}
  Bootstrap: https://${HOSTNAME_VALUE}/auth/status
  Service:   systemctl status suda-forge
  Logs:      journalctl -u suda-forge -f
  Env:       /etc/suda-forge/suda-forge.env
  Install:   ${INSTALL_DIR}
EOF
