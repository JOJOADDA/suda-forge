#!/usr/bin/env bash
set -Eeuo pipefail

[ "${EUID}" -eq 0 ] || { echo 'host-deps must run as root' >&2; exit 1; }
command -v apt-get >/dev/null || { echo 'apt-get is required on Ubuntu/Debian' >&2; exit 1; }

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq ca-certificates curl git build-essential postgresql-client golang-go

node_major=0
if command -v node >/dev/null 2>&1; then
  node_major="$(node -p 'process.versions.node.split(".")[0]' 2>/dev/null || echo 0)"
fi
if [ "$node_major" -lt 20 ]; then
  curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
  apt-get install -y -qq nodejs
fi

if ! command -v pnpm >/dev/null 2>&1; then
  npm install --global --no-audit --no-fund pnpm@9.15.0
fi

if [ "${SUDA_INSTALL_CADDY:-0}" = "1" ] && ! command -v caddy >/dev/null 2>&1; then
  log_caddy='installing Caddy'
  printf '\n==> %s\n' "$log_caddy"
  apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https gnupg
  install -d -m 0755 /usr/share/keyrings /etc/apt/sources.list.d
  curl -1sLf https://dl.cloudsmith.io/public/caddy/stable/gpg.key \
    | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  curl -1sLf https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt \
    > /etc/apt/sources.list.d/caddy-stable.list
  apt-get update -qq
  apt-get install -y -qq caddy
fi

if [ "${SUDA_INSTALL_LXC:-0}" = "1" ] && ! command -v lxc >/dev/null 2>&1; then
  command -v snap >/dev/null 2>&1 || apt-get install -y -qq snapd
  snap install lxd
  lxd init --auto
fi

printf 'host dependencies ready\n'
