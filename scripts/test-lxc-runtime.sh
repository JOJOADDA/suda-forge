#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if ! "$ROOT/scripts/check-runtime-host.sh" >/tmp/suda-forge-runtime-check.txt 2>&1; then
  cat /tmp/suda-forge-runtime-check.txt
  echo 'BLOCKED: current host is not a supported LXC runtime host.'
  exit 2
fi

NAME="suda-forge-acceptance-$$"
cleanup() { sudo lxc-stop -n "$NAME" -k >/dev/null 2>&1 || true; sudo lxc-destroy -n "$NAME" -f >/dev/null 2>&1 || true; }
trap cleanup EXIT

sudo lxc-create -n "$NAME" -t download -- -d ubuntu -r jammy -a amd64
sudo lxc-start -n "$NAME" -d
sleep 3
[ "$(sudo lxc-info -n "$NAME" -sH)" = 'RUNNING' ]
sudo lxc-attach -n "$NAME" -- sh -lc 'mkdir -p /workspace; printf hello > /workspace/hello.txt; test "$(cat /workspace/hello.txt)" = hello; git init /workspace/repo; printf "from http.server import HTTPServer,SimpleHTTPRequestHandler\nHTTPServer((""0.0.0.0"", 8765),SimpleHTTPRequestHandler).serve_forever()\n" > /workspace/server.py; nohup python3 /workspace/server.py >/tmp/suda-http.log 2>&1 &'
sudo lxc-attach -n "$NAME" -- sh -lc 'test -f /workspace/hello.txt; git -C /workspace/repo status; kill "$(pgrep -f "python3 /workspace/server.py")" || true'
sudo lxc-stop -n "$NAME" -k
sudo lxc-start -n "$NAME" -d
sleep 3
sudo lxc-attach -n "$NAME" -- test -f /workspace/hello.txt
sudo lxc-stop -n "$NAME" -k
printf 'PASS: real LXC acceptance path completed for %s\n' "$NAME"
