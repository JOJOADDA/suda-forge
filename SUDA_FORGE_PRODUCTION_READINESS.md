# SUDA FORGE — Production Readiness

## Decision

SUDA FORGE is **frozen at Phase 10**. This document is a production-readiness record, not a new architectural phase. No Phase 11 was created, no duplicate subsystem was introduced, and the modular-monolith boundaries remain authoritative.

The work executed here follows:

> Integration → Hardening → Real Runtime Validation → Production Readiness → Product UX

## Frozen architecture

The following remain the only authoritative boundaries: `RuntimeProvider`, Project Computer, Project Intelligence, Tool/Version Registry, Global Cache, Agent Fabric, Agent Constitution, Model Fabric, Deterministic Model Router, AI Fabric, Orchestration Engine, Verification Engine, RepairLoop, Deployment Fabric, Design Intelligence, Project Knowledge Graph, Context Assembly, Product Experience, Event Bus, PostgreSQL persistence, and SSE.

No second router, orchestrator, agent executor, event bus, knowledge system, verification engine, cache, microservice, or Kubernetes layer was added.

## Completed integration work

| Area | Readiness classification | Evidence |
|---|---|---|
| PostgreSQL authority | `IMPLEMENTED_AND_VERIFIED` | Product Experience loop execution, cache metadata/blobs, Knowledge Graph, and existing Phase 1–10 stores are wired through PostgreSQL in production bootstrap |
| Autonomous Loop | `IMPLEMENTED_AND_VERIFIED` for coordination and recovery | Existing loop plan now has persisted current stage, per-stage results, error, timestamps, start/resume/status APIs, and startup recovery |
| Orchestration delegation | `IMPLEMENTED_AND_VERIFIED` for planning delegation | Loop stages delegate to the existing Orchestrator and persist the resulting Workflow; no second orchestrator exists |
| Verification boundary | `IMPLEMENTED_BUT_INTEGRATION_GAP` for full live stage execution | Missing verification delegate or unavailable runtime yields `BLOCKED_BY_ENVIRONMENT`; no fake pass is emitted |
| Global Cache | `IMPLEMENTED_AND_VERIFIED` for durable blob/metadata storage | PostgreSQL `global_cache_blobs`, restart hydration, checksum validation, cache hit reuse, metadata persistence, and invalidation cleanup |
| Knowledge Graph | `IMPLEMENTED_AND_VERIFIED` for canonical population contract | Existing authoritative store now has idempotent structured `knowledge.Populate` over project-domain snapshots |
| Create Project UX | `IMPLEMENTED_AND_VERIFIED` for frontend contract/build | Wizard is feature-oriented and presents eight explainable stages, cache status, resource/runtime state, blocked state, errors, progress, and reset/retry affordance |
| Activity/SSE | `IMPLEMENTED_AND_VERIFIED` for contract | Existing project-filtered activity SSE remains the single event projection and is consumed by the frontend |
| Governance | `IMPLEMENTED_AND_VERIFIED` at the agent-start boundary | Constitution and PermissionPolicy are evaluated server-side before adapter execution |
| Deployment | `IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED` for live target | Verification gate and PostgreSQL port bindings are wired; real LXC/Caddy/health target is unavailable in the current host |

## Autonomous Loop execution contract

The existing Product Experience loop now follows this persisted state machine:

```text
PLAN CREATED
   ↓
RUNNING + current_stage persisted
   ↓
stage delegate executes
   ↓
result persisted
   ↓
next stage checkpoint
   ↓
COMPLETED / FAILED / BLOCKED
   ↓
server restart
   ↓
startup recovery resumes RUNNING or BLOCKED execution
```

The HTTP surface is:

| Endpoint | Purpose |
|---|---|
| `POST /api/projects/{project}/autonomous-loop/plan` | Persist a loop plan without starting it |
| `POST /api/projects/{project}/autonomous-loop/start` | Persist and start a loop execution asynchronously |
| `GET /api/projects/{project}/autonomous-loop/{loop}` | Read authoritative checkpoint and stage results |
| `POST /api/projects/{project}/autonomous-loop/{loop}/resume` | Resume a persisted incomplete loop |

The coordinator is an adapter over the existing Orchestrator and Verification boundary. It does not execute host operations directly and cannot turn missing runtime capabilities into success.

## Global Cache recovery contract

The cache remains a single shared infrastructure cache with a memory hot path and PostgreSQL durability:

```text
Project A → resolve artifact → checksum verify → save blob + metadata in PostgreSQL
Project B → hydrate metadata → load blob by artifact identity → checksum verify → Cache HIT
```

Invalidation removes the durable blob and marks metadata invalid. A corrupted or missing blob produces a miss/corrupt outcome rather than a false hit.

## Knowledge Graph population contract

The existing graph is populated through the `knowledge.Populate` helper using structured snapshots. The helper validates project scope, node and edge identity, timestamps missing records, upserts idempotently, and returns the authoritative graph. It is intentionally generic so Project Intelligence, Design, Agents, Tasks, Tests, Verification, Git, Releases, and Deployments can contribute through the same store without creating another graph system.

## Canonical acceptance scenario

The requested product intent remains suitable for acceptance testing:

> Build a children's educational mobile application containing interactive lessons for letters, numbers, animals, and basic exercises.

The expected decision chain is preserved:

```text
Intent → Requirements → Architecture → Technology selection
→ Environment Manifest → Resource Plan → Tools / Versions
→ Project Computer → Provisioning → Implementation → Testing
→ Verification → Repair → Deployment
```

Every architecture decision must remain explainable through selection, reason, confidence, constraints, and alternatives. The existing Project Intelligence and Environment Manifest contracts provide the decision record; the frontend now surfaces the state and reasons rather than hiding them behind a spinner.

## Validation results

The following checks passed:

```text
go test ./...
pnpm --dir apps/web build
bash scripts/check-architecture.sh
git diff --check
```

The architecture checker passed with its existing advisory warning about frontend adapter labels. The frontend TypeScript/Vite production build passed after extracting the Create Project wizard into `features/project/ProjectProvisioningWizard.tsx`.

The new autonomous-loop regression test proves that a loop checkpoints completed stages, stops honestly at `BLOCKED_BY_ENVIRONMENT`, and resumes from its persisted current stage after the blocker is removed.

## Real-runtime blockers

The current Docker-backed development host does not expose the capabilities required to prove the following live behaviors:

| Capability | Truthful result |
|---|---|
| Unprivileged LXC, namespaces, cgroups, networking, storage, PTY | `BLOCKED_BY_ENVIRONMENT` |
| Real browser DOM/screenshot/visual analysis | `BLOCKED_BY_ENVIRONMENT` |
| GPU/CUDA/local inference | `BLOCKED_BY_ENVIRONMENT` |
| Runtime-scoped Claude Code, Codex, Kimi, or other CLI execution | `BLOCKED_BY_ENVIRONMENT` when binaries/runtime credentials are unavailable |
| Caddy, real process health, ports, deployment, rollback | `BLOCKED_BY_ENVIRONMENT` without a supported deployment target |

No fake screenshot, fake browser pass, fake GPU pass, fake agent execution, fake deployment, or fake readiness state is emitted.

## Residual gaps

The remaining work is validation and operational completion, not architectural expansion. The full live loop still needs a supported host to execute implementation, build, verification, repair, deployment, and post-deployment health delegates. The Knowledge Graph population helper is ready, but individual domain projections must call it as their records become available. The cache is restart-safe for blobs and metadata, while long-term garbage collection policy beyond invalidation remains an operational policy decision.

## Conclusion

SUDA FORGE is now in **Production Readiness** rather than feature-development mode. The existing Phase 1–10 architecture is integrated, restart-aware, PostgreSQL-backed, governance-bound, cache-durable, graph-populatable, and reflected by a truthful product UX. Real runtime acceptance remains blocked only where the current environment lacks the required LXC, Browser, GPU, CLI, or deployment capabilities.
