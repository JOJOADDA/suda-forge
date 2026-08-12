# SUDA FORGE

SUDA FORGE is a runtime-agnostic Project Computer control plane. Phase 1 keeps the production runtime target as **unprivileged LXC** and uses the current Docker-backed sandbox only for backend, frontend, PostgreSQL, contract, and security testing.

## Local development

Start PostgreSQL, apply `migrations/001_initial.sql`, then run the backend from the repository root with `go run ./cmd/server`. The React application is under `apps/web`; run `pnpm install` once and `pnpm dev` for local development. The backend exposes `GET /healthz`, `GET /health`, and `GET /ready`.

`/health` reports application health separately from runtime-host health. `/ready` returns HTTP 503 when the host cannot support the required LXC Project Computer. The frontend displays **LXC unavailable** in that case and never converts it into a fake READY state.

## Runtime host validation

Run `scripts/check-runtime-host.sh` before attempting LXC integration. It is read-only and does not install packages, change firewall rules, disable AppArmor, alter namespaces, or convert the runtime to Docker. Run `scripts/test-lxc-runtime.sh` only on a supported Linux host; on the current environment it must return:

```text
BLOCKED: current host is not a supported LXC runtime host.
```

The target host must be Ubuntu 24.04 LTS x86_64 on a real VM or bare-metal Linux host, with LXC 5.x or compatible, unprivileged user namespaces, configured subuid/subgid, delegated cgroup v2, controlled LXC networking, AppArmor/seccomp, and outbound HTTPS.

## Tests

Run `go test ./...` for backend unit and contract tests. Run `cd apps/web && pnpm build` for the TypeScript production build. The real LXC test is intentionally environment-dependent and must not be relabeled as a unit or contract test.

## Architecture decisions

The dependency direction is HTTP/API → application services → domain → provider-neutral interfaces → infrastructure adapters. The domain does not know LXC command syntax, container names, filesystem paths, Caddy, shell implementation, or model providers.

`internal/runtime.Provider` is the seam for Project Computer execution. `adapters/runtimes/lxc` owns classic LXC commands. Filesystem, terminal, process, Git, port discovery, preview, and event services depend on runtime-level contracts rather than host processes. The in-process event bus and SSE endpoint use the same event shape later required for terminal, process, runtime, project, and agent events.

Phase 2 is intentionally not started. Model adapters, model registry, router, agent council, autonomous repair, and local model management remain out of scope until the real LXC Project Computer acceptance test passes on a supported Linux host.
