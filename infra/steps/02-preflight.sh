#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HOSTNAME_VALUE="${1:?public hostname is required}"

SUDA_SKIP_DNS_CHECK="${SUDA_SKIP_DNS_CHECK:-0}" \
  bash "$ROOT_DIR/infra/preflight.sh" "$HOSTNAME_VALUE"
