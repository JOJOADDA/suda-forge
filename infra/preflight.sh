#!/usr/bin/env bash
set -Eeuo pipefail

HOSTNAME_VALUE="${1:-${SUDA_HOSTNAME:-}}"
SKIP_DNS_CHECK="${SUDA_SKIP_DNS_CHECK:-0}"

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

warn() {
  printf 'WARN: %s\n' "$*" >&2
}

[ "${EUID}" -eq 0 ] || fail "the installer must run as root (use sudo)"
[ -r /etc/os-release ] || fail "/etc/os-release is missing"
# shellcheck disable=SC1091
. /etc/os-release
case "${ID:-}" in
  ubuntu|debian) ;;
  *) fail "Ubuntu or Debian is required; detected ${ID:-unknown}" ;;
esac

[ -n "$HOSTNAME_VALUE" ] || fail "a public DNS hostname is required"
case "$HOSTNAME_VALUE" in
  *[!a-zA-Z0-9.-]*) fail "hostname contains unsupported characters: $HOSTNAME_VALUE" ;;
esac
[[ "$HOSTNAME_VALUE" != .* && "$HOSTNAME_VALUE" != *..* && "$HOSTNAME_VALUE" != *. ]] || fail "invalid hostname: $HOSTNAME_VALUE"

command -v systemctl >/dev/null || fail "systemd is required"
command -v curl >/dev/null || fail "curl is required"
command -v git >/dev/null || fail "git is required"
command -v go >/dev/null || fail "Go is required to build SUDA FORGE"
command -v node >/dev/null || fail "Node.js is required to build the frontend"
command -v pnpm >/dev/null || fail "pnpm is required to build the frontend"
command -v psql >/dev/null || fail "PostgreSQL client (psql) is required"
command -v caddy >/dev/null || warn "caddy is not installed yet; the installer will install it when available"
command -v lxc >/dev/null || warn "lxc is not installed; Project Computer readiness will remain blocked until LXC is provisioned"

if ss -H -ltn 2>/dev/null | awk '{print $4}' | grep -Eq '(^|:)80$|(^|:)443$'; then
  warn "port 80 or 443 is already in use; confirm that Caddy can own the public ports"
fi

if [ "$SKIP_DNS_CHECK" != "1" ]; then
  resolved="$(getent ahostsv4 "$HOSTNAME_VALUE" 2>/dev/null | awk 'NR==1 {print $1}')" || true
  public_ip="$(curl -4fsS --max-time 5 https://api.ipify.org 2>/dev/null || true)"
  if [ -z "$resolved" ]; then
    warn "$HOSTNAME_VALUE does not resolve locally; use SUDA_SKIP_DNS_CHECK=1 only for a controlled bootstrap"
  elif [ -n "$public_ip" ] && [ "$resolved" != "$public_ip" ]; then
    warn "$HOSTNAME_VALUE resolves to $resolved while this host reports $public_ip"
  fi
fi

printf 'preflight OK: %s on %s (%s)\n' "$HOSTNAME_VALUE" "${PRETTY_NAME:-$ID}" "$(uname -m)"
