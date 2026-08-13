#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export SUDA_INSTALL_CADDY="${SUDA_INSTALL_CADDY:-1}"
export SUDA_INSTALL_LXC="${SUDA_INSTALL_LXC:-1}"

bash "$ROOT_DIR/infra/host-deps.sh"
