# SUDA FORGE — Phase 9 Status

## Outcome

Phase 9, **Shared Infrastructure, Project Computer, and Productization Foundation**, is implemented in the modular monolith. Project Computer is now a first-class persisted entity with lifecycle operations, runtime capability checks, progressive readiness, resource validation, environment drift comparison, shared tool/version/artifact registries, and an integrity-checked global cache.

The implementation preserves the existing Project Intelligence, Provisioning, Agent Fabric, AI Fabric, Orchestration, Verification, and Deployment boundaries. It does not introduce microservices, Kubernetes, a second orchestrator, a second model router, a second agent execution system, direct host-shell execution, fake LXC, fake browser, fake GPU, or hardcoded credentials.

## Implemented surface

| Area | Delivered behavior |
|---|---|
| Project Computer | First-class entity with project ownership, runtime provider, runtime ID, image identity, resources, status, fingerprint, readiness, capabilities, metadata, timestamps, and lifecycle operations. |
| Lifecycle | Create, start, stop, restart, destroy, verify, and bounded rebuild operations with audit-compatible state transitions. |
| Runtime capabilities | Filesystem, process, network, ports, PTY, browser, Git, containers, GPU, and snapshots with `SUPPORTED`, `UNSUPPORTED`, `BLOCKED_BY_ENVIRONMENT`, and `FAILED` outcomes. |
| Readiness | Progressive `CORE_READY`, `AGENT_READY`, `BROWSER_READY`, `BUILD_READY`, `DEPLOY_READY`, and `FULLY_READY` states. |
| Resource validation | Early rejection with `INSUFFICIENT_RESOURCES` for CPU, memory, disk, GPU, and GPU memory requirements. |
| Tool Registry | Shared registry for languages, frameworks, SDKs, build tools, package managers, browsers, testing tools, AI agents, CLIs, and utilities. |
| Version Registry | Platform- and architecture-aware versions with compatibility, installation, verification, dependencies, and artifact identity. |
| Artifact integrity | Artifact identity, version, platform, architecture, size, source, storage location, checksum, and metadata. |
| Global Cache | Deduplicated cache with `HIT`, `MISS`, `CORRUPT`, and `INVALID` states, checksum verification, corruption detection, invalidation, reference counts, and statistics. |
| Environment resolution | Manifest → Tool Registry → Version Registry → Artifact → Cache resolution with explicit reasons and cache outcomes. |
| Fingerprints and drift | Fingerprints include OS, image, runtimes, tools, SDKs, system packages, agents, browser versions, variables, ports, resources, and profile; drift comparison returns `ENVIRONMENT_DRIFT`. |
| Persistence | PostgreSQL migration `009_infrastructure.sql` stores Project Computers, lifecycle operations, runtime capabilities, tools, versions, artifacts, cache entries, fingerprints, capabilities, resources, and shared infrastructure events. |
| API | Project Computer lifecycle endpoints, tool and version endpoints, cache statistics/artifacts, environment resolve/verify/repair endpoints, and existing Phase 8 APIs. |
| SSE | Existing event bus carries Project Computer, environment resolution, cache, and provisioning events; no second event system was created. |
| UI | Create Project now connects to Project Computer creation, reuses its runtime for provisioning, and displays readiness, capability evidence, cache hits/misses, and resolution reasons. |

## Verification performed

The backend test suite passes with `go test ./...`. The frontend production build passes with `pnpm --dir apps/web build`. The architecture boundary check passes with `bash scripts/check-architecture.sh`, and `git diff --check` reports no whitespace errors. Tests cover Project Computer blocked runtime behavior, resource insufficiency, readiness, lifecycle verification, cache hit/miss, duplicate artifact reuse, version mismatch, checksum failure, corruption detection, and manifest resolution.

## Environment limitation

The current execution environment is a restricted Docker container and cannot provide the required unprivileged LXC kernel capabilities. This is not hidden or emulated. Project Computer creation, browser checks, GPU checks, and unavailable tools therefore remain `BLOCKED_BY_ENVIRONMENT`, `UNSUPPORTED`, or explicit failure states with evidence. Installation records, cache records, or database rows are not treated as proof of usable runtime capability.

## Phase boundary

Phase 9 is complete. Phase 10 has **not** been started. The implementation was committed as `53aba1e` and tagged `phase-9-complete`.
