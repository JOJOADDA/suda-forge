#!/usr/bin/env bash
set -Eeuo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DNS_SERVER="${SUDA_DNS_SERVER:-1.1.1.1}"
DNS_SERVER_2="${SUDA_DNS_SERVER_2:-8.8.8.8}"
ATTEMPTS="${SUDA_NETWORK_ATTEMPTS:-3}"
WAIT_SECONDS="${SUDA_NETWORK_WAIT:-3}"
FIX_DNS=0
PERSIST_RESOLV_CONF=0
PUSH=0
VERBOSE=0

log() { printf '\n==> %s\n' "$*"; }
info() { printf 'INFO: %s\n' "$*"; }
warn() { printf 'WARN: %s\n' "$*" >&2; }
fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
Usage: infra/check-network-and-push.sh [options]

Checks:
  - default route and basic HTTPS connectivity
  - DNS resolution for github.com and api.github.com
  - HTTPS access to GitHub
  - git ls-remote against the repository origin

Options:
  --repo DIR              Repository directory (default: detected project root)
  --dns-server IP         Primary DNS server for recovery (default: 1.1.1.1)
  --dns-server-2 IP       Secondary DNS server (default: 8.8.8.8)
  --attempts N            Number of check attempts (default: 3)
  --wait N                Seconds between attempts (default: 3)
  --fix-dns               Attempt a temporary DNS fix using resolvectl
  --persist-resolv-conf   If resolvectl is unavailable, allow editing /etc/resolv.conf
  --push                  Run git push origin main only after all checks pass
  --verbose               Show diagnostic command output
  -h, --help              Show this help

Safety:
  --fix-dns requires root or passwordless sudo. resolvectl changes are runtime
  settings and may be reset by the host network manager. Editing resolv.conf is
  disabled unless --persist-resolv-conf is explicitly provided.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --repo) REPO_DIR="${2:?missing repository directory}"; shift 2 ;;
    --dns-server) DNS_SERVER="${2:?missing DNS server}"; shift 2 ;;
    --dns-server-2) DNS_SERVER_2="${2:?missing secondary DNS server}"; shift 2 ;;
    --attempts) ATTEMPTS="${2:?missing attempts}"; shift 2 ;;
    --wait) WAIT_SECONDS="${2:?missing wait seconds}"; shift 2 ;;
    --fix-dns) FIX_DNS=1; shift ;;
    --persist-resolv-conf) PERSIST_RESOLV_CONF=1; shift ;;
    --push) PUSH=1; shift ;;
    --verbose) VERBOSE=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) fail "unknown option: $1" ;;
  esac
done

case "$ATTEMPTS" in ''|*[!0-9]*|0) fail "attempts must be a positive integer" ;; esac
case "$WAIT_SECONDS" in ''|*[!0-9]*) fail "wait must be a non-negative integer" ;; esac
case "$DNS_SERVER" in *[!0-9a-fA-F:.]*) fail "invalid primary DNS address" ;; esac
case "$DNS_SERVER_2" in *[!0-9a-fA-F:.]*) fail "invalid secondary DNS address" ;; esac

[ -d "$REPO_DIR/.git" ] || fail "not a Git repository: $REPO_DIR"
command -v git >/dev/null 2>&1 || fail "git is required"
command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v getent >/dev/null 2>&1 || fail "getent is required"

run_quiet() {
  if [ "$VERBOSE" = "1" ]; then
    "$@"
  else
    "$@" >/dev/null 2>&1
  fi
}

check_route() {
  command -v ip >/dev/null 2>&1 || { warn "ip command is unavailable; skipping route inspection"; return 0; }
  local route iface
  route="$(ip route show default 2>/dev/null | head -1 || true)"
  if [ -z "$route" ]; then
    warn "no default route detected"
    return 1
  fi
  iface="$(awk '{for (i=1; i<=NF; i++) if ($i == "dev") {print $(i+1); exit}}' <<< "$route")"
  info "default route: ${iface:-unknown}"
}

check_raw_internet() {
  # HTTPS to a numeric public IP checks network reachability without depending on DNS.
  run_quiet curl -kfsSI --connect-timeout 5 --max-time 10 https://1.1.1.1
}

check_dns() {
  getent ahosts github.com >/dev/null 2>&1 && getent ahosts api.github.com >/dev/null 2>&1
}

check_https() {
  run_quiet curl -fsSI --connect-timeout 5 --max-time 15 https://github.com
  run_quiet curl -fsSI --connect-timeout 5 --max-time 15 https://api.github.com
}

check_git_remote() {
  (cd "$REPO_DIR" && git ls-remote origin HEAD >/dev/null)
}

run_checks() {
  log "network diagnostics"
  check_route || return 1
  if ! check_raw_internet; then
    warn "basic HTTPS connectivity failed"
    return 1
  fi
  if ! check_dns; then
    warn "DNS resolution failed for GitHub"
    return 1
  fi
  if ! check_https; then
    warn "HTTPS access to GitHub failed"
    return 1
  fi
  if ! check_git_remote; then
    warn "git cannot access origin"
    return 1
  fi
  info "internet, DNS, HTTPS, and GitHub checks passed"
}

get_default_iface() {
  command -v ip >/dev/null 2>&1 || return 0
  ip route show default 2>/dev/null | awk 'NR==1 {for (i=1; i<=NF; i++) if ($i == "dev") {print $(i+1); exit}}'
}

as_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  elif command -v sudo >/dev/null 2>&1 && sudo -n true >/dev/null 2>&1; then
    sudo -n "$@"
  else
    return 1
  fi
}

backup_resolv_conf=""
restore_resolv_conf() {
  if [ -n "$backup_resolv_conf" ] && [ -f "$backup_resolv_conf" ]; then
    cp -f "$backup_resolv_conf" /etc/resolv.conf || true
    rm -f "$backup_resolv_conf" || true
    info "restored original /etc/resolv.conf"
  fi
}
trap restore_resolv_conf EXIT

fix_dns() {
  [ "$FIX_DNS" = "1" ] || return 1
  log "attempting DNS recovery"
  local iface
  iface="$(get_default_iface)"

  if command -v resolvectl >/dev/null 2>&1 && [ -n "$iface" ]; then
    as_root resolvectl dns "$iface" "$DNS_SERVER" "$DNS_SERVER_2" || {
      warn "resolvectl DNS update failed"
    }
    as_root resolvectl flush-caches >/dev/null 2>&1 || true
    info "applied runtime DNS servers to interface $iface"
    return 0
  fi

  if [ "$PERSIST_RESOLV_CONF" != "1" ]; then
    warn "resolvectl is unavailable; refusing to edit /etc/resolv.conf without --persist-resolv-conf"
    return 1
  fi

  as_root test -w /etc/resolv.conf || fail "cannot write /etc/resolv.conf; run as root or configure passwordless sudo"
  backup_resolv_conf="/etc/resolv.conf.suda-forge-backup.$$"
  as_root cp -L /etc/resolv.conf "$backup_resolv_conf"
  printf 'nameserver %s\nnameserver %s\n' "$DNS_SERVER" "$DNS_SERVER_2" | as_root tee /etc/resolv.conf >/dev/null
  info "temporarily replaced /etc/resolv.conf with $DNS_SERVER and $DNS_SERVER_2"
  return 0
}

log "repository: $REPO_DIR"
info "origin: $(cd "$REPO_DIR" && git remote get-url origin)"

for ((attempt=1; attempt<=ATTEMPTS; attempt++)); do
  if run_checks; then
    if [ "$PUSH" = "1" ]; then
      log "pushing main to origin"
      (cd "$REPO_DIR" && git push origin main)
      info "git push completed"
    fi
    exit 0
  fi
  if [ "$attempt" -lt "$ATTEMPTS" ]; then
    info "check attempt $attempt/$ATTEMPTS failed; waiting ${WAIT_SECONDS}s"
    sleep "$WAIT_SECONDS"
  fi
done

if [ "$FIX_DNS" = "1" ]; then
  if fix_dns; then
    for ((attempt=1; attempt<=ATTEMPTS; attempt++)); do
      if run_checks; then
        if [ "$PUSH" = "1" ]; then
          log "pushing main to origin"
          (cd "$REPO_DIR" && git push origin main)
          info "git push completed"
        fi
        exit 0
      fi
      [ "$attempt" -lt "$ATTEMPTS" ] && sleep "$WAIT_SECONDS"
    done
  fi
fi

cat >&2 <<'EOF'

DNS/Internet recovery failed. Diagnostic actions:
  1. Check the default route and host firewall.
  2. Check whether the environment blocks outbound DNS or HTTPS.
  3. Run with --fix-dns, and use --persist-resolv-conf only on a host where editing resolv.conf is appropriate.
  4. If DNS works in another terminal, rerun without --fix-dns and inspect the Git remote/authentication.
EOF
exit 1
