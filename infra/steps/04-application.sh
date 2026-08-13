#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HOSTNAME_VALUE="${1:?public hostname is required}"

SUDA_SKIP_HOST_DEPS=1 \
  SUDA_SKIP_MIGRATIONS=0 \
  SUDA_INSTALL_DIR="${SUDA_INSTALL_DIR:-/opt/suda-forge}" \
  bash "$ROOT_DIR/infra/install.sh" "$HOSTNAME_VALUE"
