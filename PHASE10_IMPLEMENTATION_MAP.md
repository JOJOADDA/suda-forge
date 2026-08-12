# SUDA FORGE — Phase 10 Implementation Map

## Rule

Phase 10 is an experience and intelligence composition layer. It must obey **no duplicate intelligence, no duplicate orchestration, and no duplicate execution**.

| Existing capability | Reuse | Extension | New Phase 10 component |
|---|---|---|---|
| Project Intelligence | `internal/projectintelligence.Engine`, requirements, classification, architecture decisions | Add design-oriented output only where it belongs in the product composition boundary | Design Intelligence service and design analysis contracts |
| Environment Manifest | `internal/environment.Manifest`, planner, verifier, fingerprints | Add design/tool/context references only if required without changing manifest authority | None unless a narrow projection is needed |
| Provisioning | `internal/provisioning.Manager`, graph, resumability, cache boundary | Add resolution metadata and truthful UX projection | None |
| Project Computer | `internal/projectcomputer.Manager`, lifecycle, capabilities, readiness | Add product-facing diagnostics and activity projection | None |
| Tool/Version/Artifact Registry | `internal/sharedinfra.Registry`, resolver, cache, checksum verification | Add design/test/agent context projections | None |
| Agent Fabric | `internal/agent.Service`, session and event contracts, permission types | Constitution/policy evaluation composes before existing agent actions | Agent Constitution and Governance composition service |
| Model Fabric/Router | Existing model registry and router | Context Assembly provides task/project context; router remains authoritative | None |
| AI Fabric | Existing local/remote runtime and hardware policy | Context and product activity consume its existing events/status | None |
| Orchestration | `internal/orchestration.Orchestrator`, DAG, scheduler, approvals, recovery, worktrees | Autonomous Product Loop creates a composed workflow plan and delegates execution | Product Loop coordinator, not a second orchestrator |
| Verification | `internal/verification.Engine`, checks, evidence, `RepairLoop` | Product Loop invokes build/test/visual/security verification through existing contracts | Visual QA boundary adapter only where existing browser capability supports it |
| Deployment | Existing deployment manager, health, rollback | Product Loop delegates deployment and post-deploy verification | None |
| Event bus/SSE | `internal/events.Bus` and current HTTP SSE handlers | Add Phase 10 event projections and activity grouping | None |
| PostgreSQL | Existing `pgxpool` stores and migrations | Add migration `010_product_experience.sql` only for new Phase 10 data | Design, knowledge, constitution, impact, and session projection stores |
| Frontend | Current `apps/web/src/App.tsx` and CSS, existing real API calls | Refactor incrementally into feature-oriented components without breaking contracts | Product workspace shell, design/knowledge/context/activity panels |

## Phase 10 domains

The minimum new backend domains are:

1. `internal/designintelligence`: structured design system analysis, tokens, component definitions, variants, patterns, layout rules, motion rules, and accessibility rules.
2. `internal/knowledge`: authoritative structured project nodes and edges with deterministic upsert/query behavior.
3. `internal/constitution`: agent identity, mission, authority, restrictions, decision/tool/verification/security/collaboration rules, and policy evaluation through existing permission/approval contracts.
4. `internal/productexperience`: context assembly, session recovery projection, change-impact analysis, autonomous-loop composition, and activity/readiness projections.

## Non-goals

No new model router, agent executor, provisioning engine, verification engine, runtime provider, event bus, fake browser, fake LXC, fake GPU, fake deployment, Kubernetes, microservices, RAG replacement for structural graph relationships, or rewrite of Phases 1–9 is allowed.
