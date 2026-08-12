# SUDA FORGE — Post-Phase-10 Integration & Production Hardening Report

## Scope and boundary

This document reports the post-Phase-10 integration and hardening pass. It is **not a new architectural phase**. No microservices, Kubernetes, duplicate orchestrator, duplicate router, duplicate event bus, duplicate verification system, vector database, or speculative product surface was introduced.

The objective was to make Phases 1–10 operate as one coherent modular monolith, to make PostgreSQL authoritative for production-critical state wherever a persistence contract already exists, and to distinguish code-level readiness from capabilities unavailable on the restricted Docker host.

## 1. Architecture audit

The complete audit is recorded in [`SUDA_FORGE_INTEGRATION_AUDIT.md`](./SUDA_FORGE_INTEGRATION_AUDIT.md). The actual product dependency graph is:

```text
Intent
 → Project Intelligence
 → Environment Manifest / Resources
 → Tool + Version + Artifact + Cache Resolution
 → Project Computer / RuntimeProvider
 → Provisioning
 → Agent Fabric + Model Router + AI Fabric
 → Orchestration
 → Verification + RepairLoop
 → Deployment + Health / Rollback
 → Event Bus
 → PostgreSQL activity projection + SSE
 → Frontend state
```

Design Intelligence, Knowledge Graph, Constitution, Context Assembly, and Change Impact are composition services attached to the same graph rather than independent execution frameworks.

## 2. Production bootstrap status

Production bootstrap now fails clearly when PostgreSQL cannot be created or pinged. The server initializes the PostgreSQL Project Computer store, Provisioning store, Knowledge Graph store, Design Intelligence store, Constitution store, Product Experience store, Agent store, Verification store, Deployment store, and routing/AI stores.

The production bootstrap was corrected in the following areas:

| Area | Result |
|---|---|
| Knowledge Graph | `knowledge.PostgresStore` is now wired into the server instead of `knowledge.NewMemoryStore` |
| Design Intelligence | One shared design-system projection is used by API handlers and Context Assembly; PostgreSQL loading is available after restart |
| Constitution | PostgreSQL-backed lazy recovery is available in Context Assembly and Constitution GET; Agent Service has a server-side execution guard |
| Tool Registry | PostgreSQL records are loaded at startup; deterministic defaults seed the authoritative store only when it is empty |
| Deployment ports | Production uses `deployment.PostgresPortRegistry` instead of `MemoryPortRegistry` |
| Provisioning | Plans are persisted at creation and runs are loaded from the persistent store before in-process projections |
| Product Experience | Autonomous-loop plans are persisted through the existing Product Experience PostgreSQL store |
| Cache | The production bootstrap no longer injects a second `provisioning.MemoryCache`; Shared Infrastructure remains the cache authority for resolution |

## 3. PostgreSQL persistence status

The database is authoritative for the stateful paths that now have production wiring. In-memory maps remain limited to projections and hot-session state.

| State | Classification |
|---|---|
| Project Computer lifecycle | `IMPLEMENTED_AND_VERIFIED` at the persistence contract level; real runtime reconciliation is `IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED` on this host |
| Provisioning runs and steps | `IMPLEMENTED_AND_VERIFIED` for store-first restart recovery and persisted plan creation |
| Intent, requirements, manifests | `IMPLEMENTED_AND_VERIFIED` |
| Knowledge Graph | `IMPLEMENTED_AND_VERIFIED` for PostgreSQL store and context-aware production contract |
| Design System | `IMPLEMENTED_AND_VERIFIED` for PostgreSQL persistence and restart-time loading of systems, tokens, and components |
| Agent Constitutions | `IMPLEMENTED_AND_VERIFIED` for persistence, lazy recovery, and server-side guard wiring |
| Activity and audit projections | `IMPLEMENTED_AND_VERIFIED` for event-bus projection and PostgreSQL activity persistence |
| Deployment port bindings | `IMPLEMENTED_AND_VERIFIED` for persistent reservation, release, listing, and conflict behavior |
| Autonomous-loop execution state | `IMPLEMENTED_BUT_INTEGRATION_GAP`; plan persistence exists, but a complete persisted worker that executes every loop stage is not yet present |
| Shared cache blobs | `IMPLEMENTED_BUT_INTEGRATION_GAP`; resolver/cache integrity behavior is verified, but durable blob hydration and reference-count cleanup across process restarts need a runtime storage adapter |

## 4. Project Computer and Provisioning

The intended flow is wired as:

```text
USER INTENT
 → PROJECT INTELLIGENCE
 → ENVIRONMENT MANIFEST
 → RESOURCE PLAN
 → TOOL/VERSION RESOLUTION
 → ARTIFACT/CACHE RESOLUTION
 → PROJECT COMPUTER
 → PROVISIONING
 → ENVIRONMENT VERIFICATION
 → READY / BLOCKED / FAILED
```

All project runtime operations continue to pass through `RuntimeProvider`. HTTP handlers do not execute host commands. Provisioning plan creation is now persisted immediately, and manager reads prefer the persistent store before consulting the in-process map. A restart-recovery regression test creates a plan with one manager and successfully resumes it through a new manager instance.

Real LXC lifecycle execution remains classified as `IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED` because the current Docker environment cannot provide the required host capability.

## 5. Global cache and shared infrastructure

The existing Shared Infrastructure resolver remains the only resolver and cache boundary. It performs deterministic tool/version selection, artifact identity checks, checksum verification, corruption detection, invalidation, cache hit/miss reporting, and duplicate requirement elimination.

The production registry now hydrates from PostgreSQL and seeds deterministic defaults only when the database has no tool records. The separate unused provisioning memory-cache injection was removed. The remaining genuine gap is durable binary/blob hydration and reference-count cleanup across process restarts; this is documented rather than represented as a false cache hit.

Classification: **`IMPLEMENTED_BUT_INTEGRATION_GAP`** for restart-persistent blob reuse, and **`IMPLEMENTED_AND_VERIFIED`** for deterministic resolution and in-process integrity behavior.

## 6. Agent execution status

Agent sessions and events use the existing PostgreSQL-backed Agent Store and adapter registry. A provider-neutral `ExecutionGuard` hook was added to Agent Service. Production bootstrap connects it to Constitution and PermissionPolicy evaluation before an agent adapter starts execution.

Agent CLI execution through Codex, Claude Code, and Kimi remains dependent on the actual Project Computer runtime and installed CLIs. On this host that path is **`IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED`**. Test-only fakes remain test-only and are not used as production adapters.

## 7. Autonomous loop status

The existing Product Experience loop plan composes the authoritative delegates:

```text
Intent → Plan → Architecture → Design → Implementation → Build → Test
→ Verification → Visual QA → Security → Repair → Re-verify
→ Commit → Deploy → Post-deploy verification
```

Loop plans are now persisted at API creation time. The plan correctly delegates to Project Intelligence, Design Intelligence, Orchestration, Verification, RepairLoop, Deployment, and Project Computer capabilities.

A complete background coordinator that executes each stage, persists the current stage, resumes after restart, and gates later stages on authoritative verification remains **`IMPLEMENTED_BUT_INTEGRATION_GAP`**. No second orchestrator was introduced; the remaining work must be a small coordinator over the existing Orchestration Engine and Scheduler.

## 8. Governance enforcement

Constitution and PermissionPolicy evaluation remains server-side. Agent Service now invokes an execution guard before adapter start. The guard loads the project-agent Constitution from PostgreSQL when it is not present in the projection map and rejects non-allowed execution.

Dangerous actions remain fail-closed through the existing Constitution evaluator as `DENY` or `APPROVAL_REQUIRED`. UI restrictions are not treated as authorization. Classification: **`IMPLEMENTED_AND_VERIFIED`** for policy evaluation and the agent-start boundary; individual terminal-tool and deployment action paths require additional end-to-end coverage before being classified beyond this boundary.

## 9. Knowledge Graph and context recovery

Context Assembly now requires the context-aware `knowledge.Store` contract. Both the PostgreSQL and memory implementations satisfy the same interface, while production bootstrap selects PostgreSQL. Design Systems and Constitutions are loaded from PostgreSQL when absent from projections.

Structured graph relationships are used by context recovery and change-impact traversal. The system does not claim to resend an entire project to the model. Classification: **`IMPLEMENTED_AND_VERIFIED`** for the store contract and context traversal; larger cross-domain reconstruction from every historical task, Git, verification, and deployment record remains **`IMPLEMENTED_BUT_INTEGRATION_GAP`** where those records are not yet normalized into graph nodes.

## 10. Change Impact

Change Impact traverses structured graph edges and returns affected nodes, files, APIs, components, tests, risk, reasons, and approval requirement. Database/table impact is classified as high-risk and requires approval.

Classification: **`IMPLEMENTED_AND_VERIFIED`** for graph traversal and risk classification; complete repository-wide indexing remains dependent on additional graph population from existing task, Git, and deployment records.

## 11. Verification authority and deployment

Deployment verification continues to use the existing Verification Store and Verification Engine. Deployment refuses to proceed when the referenced verification run is not `Passed`. Runtime health and Caddy integration remain behind deployment abstractions.

Deployment port bindings are now PostgreSQL-backed with conflict detection and release/list operations. No fake Caddy, certificate, health, or deployment success was added. LXC/Caddy runtime execution on the current host is **`IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED`**.

## 12. Event Bus, activity, and SSE

There remains one shared Event Bus. Provisioning, Project Computer, deployment, verification, repair, AI, governance/product activity, and other lifecycle events continue to use it. Activity persistence is an event projection, not a second event system.

The frontend Product Experience panel now subscribes to the named `project.activity` SSE event and cleans up the subscription on project changes. Classification: **`IMPLEMENTED_AND_VERIFIED`** for event protocol and frontend build; long-running reconnect and browser delivery are **`IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED`** until exercised in a real browser-capable runtime.

## 13. Frontend API integrity

The frontend build passes TypeScript and Vite production compilation. Product Experience state is loaded through backend APIs, and activity updates use the project-filtered SSE endpoint. The frontend does not claim blocked runtime, browser, GPU, verification, or deployment states as ready.

The existing dashboard was not visually redesigned during this pass. Classification: **`IMPLEMENTED_AND_VERIFIED`** for build and API contract compilation; full browser E2E is **`IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED`** in the current environment.

## 14. Canonical E2E and recovery results

The canonical pipeline is executable at contract/test level through Project Intelligence, manifests, resolution, Project Computer, Provisioning, activity, and verification boundaries. Real LXC, browser, agent CLI, GPU, Caddy, and deployment health execution cannot be completed on the current Docker host and must remain explicitly reported as `BLOCKED_BY_ENVIRONMENT`.

A persisted provisioning recovery regression test passed. The following recovery surfaces remain covered by existing stores/contracts but need expanded live integration tests: interrupted agent sessions, workflow scheduler restart, verification interruption, deployment interruption, and full autonomous-loop resume.

## 15. Security review

The audit reviewed host command boundaries, RuntimeProvider enforcement, project ownership checks, artifact checksums, cache invalidation, deployment port conflicts, trusted route validation, server-side Constitution checks, and database-backed project scoping.

The repository still contains host-level probes in runtime readiness and GPU hardware detection. These are infrastructure capability probes, not HTTP-handler execution paths, and they should remain isolated behind runtime/infrastructure boundaries. No new direct host command path was added.

Classification: **`IMPLEMENTED_AND_VERIFIED`** for the reviewed application boundaries; filesystem symlink, SSRF, credential, and browser-level security require live runtime security testing before a stronger claim.

## 16. Environment-blocked capabilities

| Capability | Classification | Reason |
|---|---|---|
| LXC Project Computer creation and lifecycle | `IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED` | Restricted Docker host does not expose the required LXC capability |
| GPU detection/inference | `IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED` | No usable GPU runtime is available |
| Browser Visual QA | `IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED` | Browser capability is unavailable; system returns blocked rather than pass |
| Real Agent CLI execution | `IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED` | Required runtime-scoped CLIs are not guaranteed in the host |
| Caddy/runtime deployment health | `IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED` | Real deployment target and health runtime are unavailable |

## 17. Verification commands

The following checks passed during this hardening pass:

```text
go test ./...
pnpm --dir apps/web build
bash scripts/check-architecture.sh
git diff --check
go test ./internal/provisioning ./internal/productexperience ./internal/constitution ./internal/agent ./internal/deployment ./internal/knowledge ./internal/sharedinfra
```

The architecture script emitted only its existing advisory warning about frontend adapter labels and passed the dependency checks.

## Final conclusion

SUDA FORGE Phases 1–10 now operate through one coherent modular-monolith composition with PostgreSQL-backed Knowledge Graph, Constitution recovery, registry hydration, persisted provisioning plans, persisted deployment port bindings, server-side agent-start governance, shared design projections, and real SSE activity consumption.

The remaining gaps are explicitly classified rather than hidden: durable binary cache hydration, a persisted autonomous-loop executor, complete cross-domain graph population, and live LXC/Browser/GPU/CLI/Caddy execution. No unsupported capability is reported as successful, and no Phase 11 or new architectural subsystem was created.
