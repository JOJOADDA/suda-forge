#!/usr/bin/env bash
set -u

pass=0
fail=0
check() { local name="$1"; shift; if "$@" >/dev/null 2>&1; then printf '%-24s PASS\n' "$name"; pass=$((pass+1)); else printf '%-24s FAIL\n' "$name"; fail=$((fail+1)); fi; }
value() { command -v "$1" >/dev/null 2>&1; }

echo 'SUDA FORGE Runtime Host Check'
echo '============================'
check 'OS' test -f /etc/os-release
check 'Kernel' test -n "$(uname -r)"
check 'Architecture' test "$(uname -m)" = x86_64
check 'LXC tools' value lxc-create
check 'User namespaces' unshare -Ur true
check 'SubUID/SubGID' test -s /etc/subuid
check 'Cgroups' test -e /sys/fs/cgroup/cgroup.controllers
check 'Network namespace' unshare -Urn true
check 'Storage' test "$(df -Pk / | awk 'NR==2 {print $4}')" -gt 10485760
check 'Privileges' sudo -n true
if systemd-detect-virt 2>/dev/null | grep -Eq 'docker|podman|lxc|container'; then printf '%-24s FAIL (nested/restricted: %s)\n' 'Nested container detection' "$(systemd-detect-virt 2>/dev/null)"; fail=$((fail+1)); else printf '%-24s PASS\n' 'Nested container detection'; pass=$((pass+1)); fi
if [ -f /etc/lxc/lxc-usernet ]; then printf '%-24s PASS\n' 'LXC network policy'; pass=$((pass+1)); else printf '%-24s FAIL\n' 'LXC network policy'; fail=$((fail+1)); fi

echo '----------------------------'
if [ "$fail" -eq 0 ]; then echo 'READY FOR LXC: YES'; exit 0; else echo 'READY FOR LXC: NO'; exit 2; fi
