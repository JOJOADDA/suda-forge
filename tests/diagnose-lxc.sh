#!/usr/bin/env bash
set +e
out="/home/ubuntu/suda-forge/tests/lxc-environment-diagnostic.txt"
exec > >(tee "$out") 2>&1
section() { printf '\n===== %s =====\n' "$1"; }
section OS; cat /etc/os-release
section KERNEL; uname -a
section ARCH; uname -m
section USER; whoami; id
section SUDO; sudo -n true; echo "sudo_exit=$?"
section LXC_INSTALLATION; command -v lxc; lxc --version; lxc-checkconfig || true; command -v lxc-create; lxc-create --version 2>&1 || true
section SERVICES; systemctl status lxc --no-pager || true; systemctl status lxcfs --no-pager || true
section CONTAINERS; sudo lxc list 2>&1 || true; sudo lxc-ls --fancy 2>&1 || true
section STORAGE; sudo lxc storage list 2>&1 || true; sudo lxc storage show default 2>&1 || true; df -h
section NETWORKS; sudo lxc network list 2>&1 || true; ip addr 2>&1 | head -120; ip route 2>&1 || true
section INTERNET; curl --connect-timeout 10 --max-time 20 -I https://images.linuxcontainers.org 2>&1 || true; curl --connect-timeout 10 --max-time 20 -I https://cloud-images.ubuntu.com 2>&1 || true
section DNS; getent hosts images.linuxcontainers.org; getent hosts cloud-images.ubuntu.com
section TLS; curl --connect-timeout 10 --max-time 20 -vI https://images.linuxcontainers.org 2>&1 | tail -100
section RESOURCES; free -h
section KERNEL_FEATURES; lsmod 2>/dev/null | grep -E 'lxc|overlay|br_netfilter' || true; sysctl kernel.unprivileged_userns_clone 2>&1 || true
section SECURITY; sudo aa-status 2>/dev/null || true; sudo ufw status 2>&1 || true; sudo iptables -L -n 2>/dev/null | head -100 || true
section PROXY; env | grep -i proxy || true
section VIRTUALIZATION; systemd-detect-virt 2>&1 || true; virt-what 2>/dev/null || true; cat /proc/1/cgroup; cat /proc/self/status | grep -E 'Cap|Seccomp|NoNewPrivs'
section SUBIDS; cat /etc/subuid; cat /etc/subgid
section LXC_CONFIG; find /etc/lxc -maxdepth 3 -type f -print -exec sed -n '1,160p' {} \; 2>/dev/null || true
section END; date -Is
