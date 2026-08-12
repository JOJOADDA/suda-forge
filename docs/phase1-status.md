# SUDA FORGE — Phase 1 Acceptance Status

**Status:** Phase 1 implementation complete within the capabilities of the current host. Phase 2 is not started.

| Component | Classification | Evidence |
|---|---|---|
| Go backend | PASS | `go test ./...` |
| React + TypeScript frontend | PASS | `pnpm build` |
| PostgreSQL migrations and CRUD | PASS | `go test -tags=integration ./internal/postgres` |
| Project domain and lifecycle | PASS | domain unit tests and invariant checks |
| Filesystem traversal and symlink boundary | PASS | malicious-path and symlink regression tests |
| RuntimeProvider abstraction | PASS | provider contract harness |
| Application health/readiness distinction | PASS | live `/health` = application READY, runtime BLOCKED; `/ready` = HTTP 503 |
| Event bus and SSE boundary | PASS | in-process event bus tests and live route wiring |
| Authentication boundary | PASS | boundary unit tests; provider integration remains configurable |
| Runtime-scoped process/Git/ports/preview contracts | PASS | provider-neutral service contracts compile; real execution depends on runtime |
| LXC host validator | PASS | `scripts/check-runtime-host.sh` reports the host accurately |
| Real LXC integration | BLOCKED BY ENVIRONMENT | current host is Docker; network/mount/cgroup delegation unavailable |
| Real PTY | BLOCKED BY ENVIRONMENT | requires a running Project Computer |
| Real project runtime | BLOCKED BY ENVIRONMENT | requires supported unprivileged LXC host |
| Real preview | BLOCKED BY ENVIRONMENT | requires a runtime process and discovered port |
| AI/model/agent features | NOT IMPLEMENTED BY DESIGN | Phase 2 is explicitly frozen |

## Commands

```bash
go test ./...
DATABASE_URL='postgres://suda:suda@localhost:5432/suda_forge?sslmode=disable' go test -tags=integration ./internal/postgres
(cd apps/web && pnpm build)
./scripts/check-runtime-host.sh
./scripts/test-lxc-runtime.sh
```

On the current environment the last command must return exit code 2 and print `BLOCKED: current host is not a supported LXC runtime host.` It must not create a privileged or fake runtime.

## Runtime boundary

The intended execution chain remains `HTTP/API → application services → domain → RuntimeProvider → LXCProvider → Project Computer`. The domain does not contain LXC syntax, host commands, Caddy details, or shell concatenation. Process, Git, port, preview, filesystem, terminal, and event paths remain runtime-scoped.

## Exact next action

Run `scripts/check-runtime-host.sh` and then `scripts/test-lxc-runtime.sh` on an Ubuntu 24.04 x86_64 real VM or bare-metal host with LXC 5.x, unprivileged user namespaces, configured subuid/subgid, delegated cgroup v2, controlled LXC network policy, AppArmor/seccomp, and outbound HTTPS. Only after the real acceptance test passes should Phase 2 begin.
