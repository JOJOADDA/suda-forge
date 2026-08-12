# SUDA FORGE

SUDA FORGE is a production-oriented, provider-agnostic AI agent platform built as a modular monolith. Every project is modeled as a **Project Computer** with a runtime boundary, while Agents, Models, Runtimes, and Projects remain separate concepts.

The repository currently contains completed Phases 1–8:

| Phase | Capability | Status |
|---|---|---|
| 1 | Project Computer foundation, runtime contracts, LXC adapter, PostgreSQL, React dashboard | Complete |
| 2 | Agent Fabric, adapter SDK, sessions, normalized events, security boundaries | Complete |
| 3 | Model Fabric and deterministic provider/model routing | Complete |
| 4 | Agentic orchestration, task graphs, scheduling, persistence, approvals, SSE | Complete |
| 5 | Verification Engine, authoritative evidence, bounded automatic repair, verification gates | Complete |
| 6 | SUDA AI Fabric, local runtime adapters, hardware/GPU discovery, inference, lifecycle, routing integration | Complete |
| 7 | Hosting and Deployment Fabric, previews, domains, certificates, health, rollback | Complete |
| 8 | Project Intelligence, deterministic architecture selection, versioned environment manifests, resumable provisioning, persistence, APIs, SSE, Create Project wizard | Complete |
| 9 | Shared Infrastructure, Project Computer lifecycle, runtime capabilities, Tool/Version/Artifact registries, integrity-checked global cache, environment resolution, readiness UX | Complete |
| 10 | Product Experience, Design Intelligence, structured Project Knowledge Graph, Agent Constitution, governance, context assembly, autonomous-loop composition, Visual QA boundary, and activity UX | Complete |

## Architecture

The dependency direction is **HTTP/API → application services → domain contracts → infrastructure adapters**. The domain does not know LXC syntax, host shell details, provider-specific response formats, or model-vendor assumptions.

`internal/runtime.Provider` is the Project Computer execution seam. `adapters/runtimes/lxc` owns LXC-specific behavior. `internal/agent` owns Agent Fabric contracts and sessions. `internal/routing` remains the authoritative deterministic Model Router. `internal/aifabric` supplies runtime discovery, model health, resources, normalized inference, streaming, and lifecycle capabilities to the existing registry and router.

The AI Fabric supports real adapters for Ollama and OpenAI-compatible runtimes such as vLLM and llama.cpp. Local runtime endpoints are opt-in through environment variables; SUDA FORGE never installs large models, runtimes, CUDA, or drivers automatically at startup.

## Local development

Start PostgreSQL, apply migrations in numeric order, and run the backend from the repository root:

```bash
for migration in migrations/*.sql; do psql "$DATABASE_URL" -f "$migration"; done
go run ./cmd/server
```

The React application is under `apps/web`:

```bash
cd apps/web
pnpm install
pnpm dev
```

The backend exposes `GET /healthz`, `GET /health`, and `GET /ready`. `/health` reports application health separately from Project Computer runtime-host health. `/ready` returns HTTP 503 when the host cannot support the required LXC Project Computer. This is an explicit environment state, not a fake ready state.

## Optional local AI runtimes

Configure an existing runtime explicitly before starting the server:

```bash
export OLLAMA_URL=http://127.0.0.1:11434
export VLLM_URL=http://127.0.0.1:8000
export LLAMACPP_URL=http://127.0.0.1:8080
```

Only configured endpoints are registered. Runtime discovery and model lifecycle operations use real HTTP adapters. If a runtime is absent or unreachable, the API reports `BLOCKED_BY_ENVIRONMENT` or an explicit offline/error status; it never claims a successful local inference.

## AI Fabric API surface

The AI Fabric exposes runtime health and lifecycle endpoints under `/api/ai/runtimes`, model discovery and lifecycle under `/api/ai/models`, hardware and GPU discovery under `/api/ai/hardware` and `/api/ai/gpus`, normalized inference under `/api/ai/inference`, SSE streaming under `/api/ai/inference/stream`, and project policy settings under `/api/projects/{project}/ai-settings`.

Project settings can restrict providers, runtimes, models, privacy, local-only behavior, budget, and routing policy. These settings are applied before the existing deterministic Model Router chooses a model. Fallback candidates are subject to the same privacy, budget, capability, health, resource, and availability constraints.

## Runtime host validation

Run `scripts/check-runtime-host.sh` before attempting LXC integration. It is read-only and does not install packages, change firewall rules, disable AppArmor, alter namespaces, or convert the runtime to Docker. Run `scripts/test-lxc-runtime.sh` only on a supported Linux host; in the current Docker-backed development environment it must return:

```text
BLOCKED: current host is not a supported LXC runtime host.
```

The target host must be Ubuntu 24.04 LTS x86_64 on a real VM or bare-metal Linux host, with compatible LXC, unprivileged user namespaces, configured subuid/subgid, delegated cgroup v2, controlled LXC networking, AppArmor/seccomp, and outbound HTTPS.

The same environment-aware rule applies to GPU, CUDA, Ollama, vLLM, and llama.cpp execution. Their adapters and contracts are implemented, but live execution is **BLOCKED_BY_ENVIRONMENT** when those capabilities are not present.

## Tests

Run the complete backend suite and frontend production build:

```bash
go test ./...
pnpm --dir apps/web build
bash scripts/check-architecture.sh
```

The real LXC and local-AI runtime checks are environment-dependent. Deterministic adapter tests use test-only HTTP servers and never create fake production inference, fake GPU success, or fake runtime success.

Phase reports and raw verification outputs are kept in `PHASE4_STATUS.md`, `PHASE5_STATUS.md`, `PHASE6_STATUS.md`, [`PHASE8_STATUS.md`](./PHASE8_STATUS.md), [`PHASE9_STATUS.md`](./PHASE9_STATUS.md), [`PHASE10_STATUS.md`](./PHASE10_STATUS.md), [`PHASE10_IMPLEMENTATION_MAP.md`](./PHASE10_IMPLEMENTATION_MAP.md), [`SUDA_FORGE_INTEGRATION_AUDIT.md`](./SUDA_FORGE_INTEGRATION_AUDIT.md), [`SUDA_FORGE_INTEGRATION_HARDENING_REPORT.md`](./SUDA_FORGE_INTEGRATION_HARDENING_REPORT.md), [`SUDA_FORGE_PRODUCTION_READINESS.md`](./SUDA_FORGE_PRODUCTION_READINESS.md), and `tests/`.

## Security boundaries

Models do not execute tools directly. Tool calls are normalized and returned to the Agent layer, where tool policy and Project Computer boundaries decide whether filesystem, terminal, browser, Git, database, or other actions may occur. Provider credentials remain server-side through the existing credential-reference infrastructure. Host GPU access is not exposed indiscriminately; allocation is capability- and policy-based.

## Phase 7 hosting and deployment

Phase 7 adds the local-first Hosting and Deployment Fabric without changing the Project Computer, Agent Fabric, Model Fabric, Orchestrator, or Verification Engine foundations. The deployment path is modeled as `CODE → BUILD → TEST → VERIFY → RELEASE → DEPLOY → HEALTH → TRAFFIC → ROLLBACK`.

The provider-neutral deployment contracts are in `internal/deployment`. They cover service discovery, port allocation, deployment, network validation, proxy routing, certificates, health checks, and storage. The first local adapters are runtime-scoped service discovery through `RuntimeProvider`, Caddy route administration through its HTTP admin API, Caddy/Let's Encrypt certificate boundaries, and local filesystem storage with traversal protection. Deployment build, test, health, and release operations are executed through the Project Computer runtime contract; HTTP handlers never run host shell commands.

The Phase 7 API includes project-scoped services, service discovery, ports, deployments, releases, rollback, previews, domains, certificates, and health checks. The dashboard includes a Deployment Workspace showing environment, revision, lifecycle state, health, failure reason, and rollback controls. Deployment activation requires an authoritative Phase 5 verification run ID, explicit build/test command arrays, and a successful runtime-scoped health check.

Caddy and real LXC deployment remain environment-dependent. Configure `CADDY_ADMIN_URL` only when a controlled Caddy admin endpoint is available. In the current restricted Docker environment, real LXC, Caddy, external certificate issuance, and runtime deployment are reported as blocked rather than converted into fake success.

## Phase 8 Project Intelligence and provisioning

Phase 8 transforms non-technical project intent into a deterministic architecture proposal, an auditable versioned Environment Manifest, and a resumable Project Computer provisioning run. The implementation is deliberately provider-agnostic: every host operation remains behind `internal/runtime.Provider`, while LXC, GPU, browser, and unavailable local-tool capabilities return explicit `BLOCKED_BY_ENVIRONMENT` or failure states instead of fake success.

The Phase 8 API includes `POST /api/projects/{project}/intelligence/analyze`, `POST /api/projects/{project}/environment/manifests`, and provisioning plan/start/status/cancel/resume/cleanup endpoints. Provisioning progress is adapted into the existing shared SSE event bus. PostgreSQL migration `008_project_intelligence.sql` stores intents, extracted requirements, architecture decisions, manifests, environment versions, provisioning runs and steps, installations, verification records, and environment events. The React dashboard now contains an eight-stage Create Project experience: intent understanding, architecture planning, Project Computer preparation, tool preparation, dependency installation, AI-agent preparation, environment verification, and readiness. It exposes progress, cache status, warnings, errors, blocked states, and reset/recovery actions from backend state.

Project Intelligence is deterministic by design. It does not silently replace an incompatible user override, and it records selected candidates, rejected alternatives, reasons, manifest fingerprints, and required verification evidence. Phase 8 composes with the existing Orchestration and Verification layers; it does not introduce microservices, Kubernetes, or a second runtime boundary.

## Phase 9 Shared Infrastructure and Project Computer

Phase 9 makes Project Computer a first-class persisted entity with lifecycle operations through `RuntimeProvider`, explicit runtime capability checks, progressive readiness states, resource insufficiency rejection, environment fingerprints, and environment drift comparison. It adds a shared Tool Registry, version-aware resolution, Artifact identity and checksum verification, a global deduplicated cache boundary, cache invalidation, and manifest-to-tool resolution events. The API includes Project Computer lifecycle endpoints, shared tool/version endpoints, cache statistics, and project-scoped environment resolve/verify/repair operations. The existing Create Project wizard now surfaces cache reasoning, environment resolutions, Project Computer status, readiness, and capability evidence.

The current restricted Docker host still cannot provide real LXC, GPU, browser, or unavailable agent binaries. Phase 9 therefore preserves explicit `BLOCKED_BY_ENVIRONMENT`, `UNSUPPORTED`, `FAILED`, and `INSUFFICIENT_RESOURCES` outcomes. It does not claim a Project Computer is ready merely because a database record or installation step exists. Migrations `012_autonomous_loop_execution.sql` and `013_global_cache_blobs.sql` add restart-safe loop checkpoints and durable shared-cache blobs.

## Phase 10 Product Experience and Intelligence Composition

Phase 10 adds the product experience and intelligence composition layer without rebuilding any Phase 1–9 subsystem. It provides Design Intelligence and structured Design System persistence, a queryable Project Knowledge Graph, Agent Constitution and policy evaluation, layered Context Assembly, session recovery projections, graph-based impact analysis, autonomous-loop delegation to existing orchestration and verification, a truthful Visual QA boundary, and project activity projection through the existing event bus and SSE.

The Phase 10 status report and implementation map are available in [`PHASE10_STATUS.md`](./PHASE10_STATUS.md) and [`PHASE10_IMPLEMENTATION_MAP.md`](./PHASE10_IMPLEMENTATION_MAP.md). The post-Phase-10 integration audit, hardening report, and frozen-architecture Production Readiness record are available in [`SUDA_FORGE_INTEGRATION_AUDIT.md`](./SUDA_FORGE_INTEGRATION_AUDIT.md), [`SUDA_FORGE_INTEGRATION_HARDENING_REPORT.md`](./SUDA_FORGE_INTEGRATION_HARDENING_REPORT.md), and [`SUDA_FORGE_PRODUCTION_READINESS.md`](./SUDA_FORGE_PRODUCTION_READINESS.md).

## Out of scope until later phases

RAG, vector databases, model training, fine-tuning, distributed GPU clusters, multi-node inference, Kubernetes, microservices, agent councils, advanced deployment, autonomous maintenance, full IDE replacement, and Phase 11 work are intentionally not implemented.
