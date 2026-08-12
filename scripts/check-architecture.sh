#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fail=0
for forbidden in 'adapters/runtimes/lxc' 'net/http' 'os/exec'; do
  if grep -R "$forbidden" "$ROOT/domain" >/dev/null 2>&1; then echo "FAIL domain imports $forbidden"; fail=1; fi
done
if grep -R 'os/exec' "$ROOT/internal/agent" >/dev/null 2>&1; then echo 'FAIL agent core executes host commands'; fail=1; fi
if grep -R -E 'os/exec|exec.Command|/bin/sh|/bin/bash' "$ROOT/internal/orchestration" >/dev/null 2>&1; then echo 'FAIL orchestrator contains direct host command execution'; fail=1; fi
if grep -R -E 'claude|codex|kimi' "$ROOT/apps/web/src" >/dev/null 2>&1; then echo 'WARN frontend contains adapter labels; verify it does not parse provider output'; fi
if [ "$fail" -ne 0 ]; then exit 1; fi
echo 'PASS architecture dependency checks'
