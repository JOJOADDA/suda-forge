# SUDA FORGE — Integration Audit

## Audit scope

This audit was performed before changing implementation code for the post-Phase-10 integration and hardening pass. It traces the current repository across the Project Intelligence, Environment, Shared Infrastructure, Project Computer, Provisioning, Agent, Model, AI, Orchestration, Verification, Deployment, Knowledge, Design, Event, PostgreSQL, and Frontend boundaries.

The audit distinguishes **code existence** from actual runtime wiring and from availability on the current restricted Docker host.

## Actual dependency graph

```text
Project Intent
  → Project Intelligence Engine
  → Environment Manifest + Resource Plan
  → Shared Tool/Version/Artifact Resolver + Global Cache
  → Project Computer Manager
  → RuntimeProvider
  → Provisioning Manager
  → Agent Service / Adapter
  → Model Router / AI Fabric
  → Orchestration DAG / approvals / recovery
  → Verification Engine / RepairLoop
  → Deployment Manager / health / rollback
  → Event Bus → persisted activity → SSE → Frontend

Knowledge Graph + Design Intelligence + Agent Constitution
  → Context Assembly / Change Impact / Governance
  → Agent and autonomous-loop composition
```

The graph is structurally present and the main packages compile together. The audit found several **contract-only** or **memory-backed** seams that must be closed before the product can claim production-grade restart recovery.

## Integration matrix

| Component | Exists | Persisted | Wired | Runtime-capable | Tested | Gap | Recommended action |
|---|---:|---:|---:|---:|---:|---|---|
| Project Intent | Yes | Yes through Phase 8 stores | Yes | Yes | Yes | No material gap found | Preserve current API and persistence path |
| Project Intelligence | Yes | Yes | Yes | Yes | Yes | Architecture output is not yet the sole source for every downstream product action | Add end-to-end contract test from intent through manifest |
| Environment Manifest | Yes | Yes | Yes | Host-dependent | Yes | Verification and drift repair are runtime-dependent | Keep fail-closed status and add restart recovery test |
| Tool Registry | Yes | No in production bootstrap | Partially | Host-dependent | Yes | `DefaultRegistry()` is in-memory; registry PostgreSQL store is not wired | Load registry from PostgreSQL and fail clearly if required records are unavailable |
| Version Registry | Yes through Shared Infrastructure | No in production bootstrap | Partially | Host-dependent | Yes | Same in-memory bootstrap gap | Make database-backed registry authoritative, retaining deterministic defaults only for tests/bootstrap migration |
| Artifact Registry | Yes | Schema exists | Partially | Host-dependent | Yes | Artifact rows and catalog are not restored into the runtime registry at startup | Add authoritative load and integrity reconciliation |
| Global Cache | Yes, checksum-aware | Schema/store exists | Partially | Host-dependent | Yes | `globalCache` is in-memory and provisioning uses a separate `provisioning.NewMemoryCache()` | Use one shared cache boundary and hydrate it from PostgreSQL; add reference counting/cleanup checks |
| Project Computer | Yes | Yes via `projectcomputer.PostgresStore` | Yes | **Blocked on current Docker host for LXC** | Yes | Capability/readiness restore and runtime reconciliation need startup recovery | Load persisted records, reconcile RuntimeProvider status, preserve `BLOCKED_BY_ENVIRONMENT` |
| RuntimeProvider | Yes | N/A | Yes | **Environment-blocked for LXC** | Yes with fakes/adapters | Real host execution cannot be proven here | Keep all host operations behind provider and test on LXC-capable host |
| Provisioning | Yes | Yes via `provisioning.PostgresStore` | Yes | **Environment-blocked where runtime steps need LXC** | Yes | Uses a separate memory cache and does not yet consume the shared artifact cache as the authoritative reuse path | Inject the shared cache adapter and reconcile interrupted runs at bootstrap |
| Agent Fabric | Yes | Session stores exist | Yes at service/API level | Agent CLI execution is host-dependent | Unit/contract tests | Real adapter-to-Project-Computer execution is not proven in the restricted host | Add runtime-boundary integration tests; never use MockAdapter in production |
| Agent Constitution | Yes | Store/schema exists | Partially | Yes for policy evaluation | Yes | Bootstrap keeps `constitutions` as an empty in-memory map and does not reload PostgreSQL records; execution boundary enforcement needs proof | Load constitution per project/agent and enforce before tool execution |
| PermissionPolicy | Yes | Embedded in constitution/agent stores | Partially | Yes for deterministic evaluation | Yes | Governance API exists, but not every execution path is proven to call it | Add server-side authorization middleware/guard at agent tool and deployment boundaries |
| Model Fabric | Yes | Yes through existing model/routing stores | Yes | Provider-dependent | Existing tests | No material Phase 10 duplication found | Preserve provider-agnostic contracts |
| Model Router | Yes | Yes | Yes | Provider-dependent | Existing tests | Fallback execution under provider failure needs end-to-end test | Add failure/recovery integration test without creating a router duplicate |
| AI Fabric | Yes | Yes | Yes | **Local runtimes/GPU host-dependent** | Existing tests | Real inference/GPU readiness cannot be proven in Docker | Keep explicit runtime health and GPU blocking states |
| Orchestration | Yes | Yes | Yes | Runtime-dependent | Existing tests | Phase 10 autonomous loop currently plans delegates rather than running a persisted loop | Add a persisted composition coordinator over the existing Orchestrator, not a second orchestrator |
| Verification Engine | Yes | Yes | Yes | Runtime-dependent | Yes | Authority is wired for deployment verification; broader agent success gating needs more integration coverage | Add tests proving agent success cannot unlock deployment without passed verification |
| RepairLoop | Yes | Yes | Yes | Runtime-dependent | Yes | Repair execution requires real task executor/runtime | Keep existing RepairLoop and add interruption/reverify tests |
| Knowledge Graph | Yes | Store/schema exists | **Partially** | N/A | Yes | Production bootstrap uses `knowledge.NewMemoryStore`; PostgreSQL store has context-aware methods that do not implement the package `Store` interface | Unify the store contract or add a narrow adapter, then load graph from PostgreSQL |
| Design Intelligence | Yes | Store/schema exists | **Partially** | N/A | Yes | API map and Product Experience map are separate; Context Assembly's design map is not populated by the API handler | Use one shared design repository/service and restore systems at startup |
| Context Assembly | Yes | Snapshot store exists | Partially | N/A | Yes | Current service reads the in-memory graph/constitution maps, so restart recovery is incomplete | Query authoritative PostgreSQL-backed graph, design, and constitution stores |
| Change Impact | Yes | Store/schema exists | Partially | N/A | Yes | It traverses the in-memory graph in production bootstrap | Run against the authoritative graph and persist analysis results |
| Autonomous Product Loop | Yes as plan composition | Plan schema exists | **Contract-only for execution** | Runtime-dependent | Unit tests | No persisted worker/coordinator currently executes every stage and resumes after restart | Implement the smallest persisted coordinator using existing Orchestration/Verification/Deployment APIs |
| Deployment | Yes | Yes | Yes | Caddy/runtime-dependent | Existing tests | Port registry is memory-backed and deployment restart recovery needs verification | Persist/restore port allocations and reconcile active releases/health |
| Event Bus | Yes, one bus | Activity projection exists | Yes | Yes | Existing tests | Some domain events and DB activity projections need coverage for all major state changes | Keep one bus; add event coverage and persistence replay where required |
| SSE | Yes | N/A | Yes | Yes | Existing tests/build | Frontend currently fetches activity but does not visibly consume the stream for all screens | Add reconnecting stream subscription for live product activity |
| PostgreSQL | Yes | Yes for many subsystems | Yes | Environment/config-dependent | Migration/tests | New Phase 10 stores are not all authoritative in bootstrap | Fail clearly for required DB in production mode and hydrate all stateful services |
| Frontend | Yes | Backend-dependent | Yes for core APIs | Browser-dependent | Production build | App still contains a large legacy surface and some fetch-only polling; live state/recovery coverage is incomplete | Keep visual design stable, add contract/E2E tests and SSE reconnect/state mapping |

## Important discovered gaps

The most significant production gap is in bootstrap authority: `cmd/server/main.go` currently initializes PostgreSQL stores for several Phase 1–9 domains, but initializes Phase 10 Knowledge Graph state with `knowledge.NewMemoryStore`, initializes an empty in-memory constitution map, initializes design systems in an in-memory map, and creates a separate provisioning memory cache from the Shared Infrastructure cache. These are not architectural duplicates, but they are inconsistent persistence boundaries.

The second major gap is execution composition. Phase 10 currently exposes an autonomous-loop plan whose delegates correctly point to existing subsystems, but it is not yet a persisted coordinator that executes, gates, resumes, and re-verifies the complete loop. This must be added as the smallest composition fix, not as a new orchestrator.

The third gap is enforcement proof. Governance evaluation exists as an API and deterministic policy evaluator, but the audit must prove that dangerous actions are blocked at the actual agent tool, terminal, Git, and deployment execution boundaries rather than only at the UI or evaluation endpoint.

## Security observations

The repository's direct host-operation boundary is concentrated behind `RuntimeProvider` and existing adapters. The current audit found no reason to introduce host shell execution in HTTP handlers or Phase 10 composition packages. Production review must still test project isolation, credential references, filesystem/symlink boundaries, SSRF-safe proxy behavior, deployment authorization, and cache artifact integrity.

## Audit classification summary

| Classification | Meaning in this repository |
|---|---|
| `IMPLEMENTED_AND_VERIFIED` | Code exists, is wired through the intended contract, and has passing automated coverage in the available environment. |
| `IMPLEMENTED_BUT_ENVIRONMENT_BLOCKED` | Code and contracts are present, but LXC/GPU/Browser/agent CLI/runtime capability cannot be exercised on the current host. |
| `IMPLEMENTED_BUT_INTEGRATION_GAP` | Code exists and may have unit coverage, but production bootstrap, persistence, restart recovery, or execution-boundary wiring is incomplete. |
| `UNSUPPORTED` | The current system explicitly does not provide the capability and does not claim it. |
| `FAILED` | A required check or operation fails for a code or configuration reason, not merely because the host lacks a capability. |

## Audit conclusion

Phases 1–10 are not isolated islands: they share the intended modular-monolith contracts, database, runtime boundary, event bus, orchestration, verification, and frontend API surface. However, the product is not yet fully production-hardened because several Phase 10 stores remain memory-backed in bootstrap and the autonomous loop is currently a plan rather than a resumable executor. The next implementation work should close these integration gaps only, preserve all existing subsystem ownership, and retain explicit environment-blocked outcomes.
