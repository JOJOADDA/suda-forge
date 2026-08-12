# SUDA FORGE

SUDA FORGE is a production-oriented, provider-agnostic AI agent platform built as a modular monolith. Every project is modeled as a **Project Computer** with a runtime boundary, while Agents, Models, Runtimes, and Projects remain separate concepts.

The repository currently contains completed Phases 1–6:

| Phase | Capability | Status |
|---|---|---|
| 1 | Project Computer foundation, runtime contracts, LXC adapter, PostgreSQL, React dashboard | Complete |
| 2 | Agent Fabric, adapter SDK, sessions, normalized events, security boundaries | Complete |
| 3 | Model Fabric and deterministic provider/model routing | Complete |
| 4 | Agentic orchestration, task graphs, scheduling, persistence, approvals, SSE | Complete |
| 5 | Verification Engine, authoritative evidence, bounded automatic repair, verification gates | Complete |
| 6 | SUDA AI Fabric, local runtime adapters, hardware/GPU discovery, inference, lifecycle, routing integration | Complete |

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

Phase reports and raw verification outputs are kept in `PHASE4_STATUS.md`, `PHASE5_STATUS.md`, `PHASE6_STATUS.md`, and `tests/`.

## Security boundaries

Models do not execute tools directly. Tool calls are normalized and returned to the Agent layer, where tool policy and Project Computer boundaries decide whether filesystem, terminal, browser, Git, database, or other actions may occur. Provider credentials remain server-side through the existing credential-reference infrastructure. Host GPU access is not exposed indiscriminately; allocation is capability- and policy-based.

## Out of scope until later phases

RAG, vector databases, model training, fine-tuning, distributed GPU clusters, multi-node inference, Kubernetes, microservices, agent councils, advanced deployment, autonomous maintenance, and Phase 7 work are intentionally not implemented.
