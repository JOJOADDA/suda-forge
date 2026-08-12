# SUDA FORGE — Frontend Integration Report

## Scope

This update implements the frontend portion of the attached product directive within the frozen Phase-10 architecture. It does not create another architectural phase, replace the backend, or introduce duplicate routers, orchestrators, stores, event buses, or runtime boundaries.

The frontend remains a React/Vite client of the existing Go modular monolith. PostgreSQL-backed backend state remains authoritative; the UI only renders responses, persisted checkpoints, SSE projections, and explicit environment outcomes.

## Product navigation

A feature-oriented workspace navigation component was added at `apps/web/src/features/navigation/WorkspaceNavigation.tsx`. It provides stable anchors for Create Project, Project Computer, Agents, Orchestration, Autonomous Loop, Verification, AI Control, Deployment, Model Routing, Knowledge/Impact, and Activity. The navigation is a presentation layer only and does not create a second routing or orchestration subsystem.

## Connected frontend surfaces

| Surface | Frontend behavior | Backend contract |
|---|---|---|
| Create Project | Intent textarea, platform hints, analysis, manifest, provisioning, cache/runtime states, reset/recovery | Project Intelligence, Environment Manifest, Provisioning, Project Computer, SSE |
| Project Computer | Readiness, runtime, capabilities, environment resolutions, cache status | `/api/project-computers`, environment resolve, cache stats |
| Agent Workspace | Worker cards, session creation, normalized events, runtime, verification context | Agent registry, agent sessions, event endpoints, activity SSE |
| Governance | Permission/risk/resource preflight and decision rendering | Constitution lookup and `/governance/evaluate`; frontend cannot override policy |
| Orchestration | Workflow goal and task graph | Existing `/workflows` orchestration endpoint |
| Autonomous Loop | Start, persisted current stage, result cards, polling, resume after BLOCKED/FAILED | Persisted Product Experience coordinator start/status/resume endpoints |
| Verification | Run, evidence cards, failed check cause, repair attempts | Verification Engine and RepairLoop endpoints |
| AI Control Center | Runtime health, hardware, models, policy and local-only setting | AI runtime/model/hardware APIs and project AI settings |
| Deployment | Release inputs, verification-gated deployment, health, status, rollback | Deployment and rollback endpoints |
| Model Routing | Policy preview, selected model, alternatives, reason, estimated cost | Deterministic Model Router endpoint |
| Knowledge Graph | Structured graph preview with clickable node IDs | PostgreSQL-backed Knowledge Graph endpoint |
| Change Impact | Node selection, affected files/APIs/components/tests, risk, approval requirement | Existing `/impact/analyze` Product Experience endpoint |
| Visual QA | Run action and viewport result states; shows blocked/reason without fake evidence | Existing Visual QA endpoint and Project Computer capability boundary |
| Activity | Project-filtered historical activity plus live events and normalized state labels | Existing activity API and `/activity/stream` SSE projection |

## Contract verification

A saved path-based contract audit script is included at `scripts/frontend_backend_contract_audit.mjs`. It compared every frontend `${API}` path in `App.tsx` with registered backend routes in `internal/httpapi/server.go`.

The result was:

```text
backend route count: 112
unresolved frontend paths: 0
SSE contract: true
```

The audit intentionally checks route coverage without inventing method semantics from complex inline fetch expressions. The frontend calls use the existing endpoint methods and request payloads already accepted by the backend handlers.

## Truthful state handling

The UI distinguishes `EXECUTING`, `WAITING`, `VERIFYING`, `BLOCKED`, `COMPLETED`, and `FAILED` where backend state is available. Visual QA does not claim screenshots or passes when the browser capability is unavailable. Project Computer, GPU, browser, deployment, and local-agent limitations remain visible as environment or backend states rather than being hidden behind permanent loading indicators.

Governance decisions are displayed from the backend evaluator as `ALLOW`, `APPROVAL_REQUIRED`, `DENY`, or `UNAVAILABLE`. The frontend does not approve or bypass an action. Deployment remains disabled until an authoritative verification object is selected.

## Validation

The following checks passed after the frontend integration:

```text
node scripts/frontend_backend_contract_audit.mjs
pnpm --dir apps/web build
go test ./...
bash scripts/check-architecture.sh
git diff --check
```

A sandbox browser smoke check loaded the frontend and exposed the expected navigation and controls without a JavaScript runtime error. Screenshot upload was unavailable in the sandbox browser, so no visual pass is claimed. The frontend Visual QA panel remains environment-aware and will report blocked until a real browser capability is available.

## Environment blockers

The following are intentionally not simulated: real LXC Project Computers, GPU/CUDA, browser screenshots and DOM analysis, unavailable Agent CLI runtimes, Caddy, and live deployment targets. These remain `BLOCKED_BY_ENVIRONMENT` or unavailable according to backend responses.

## Files

| File | Purpose |
|---|---|
| `apps/web/src/features/navigation/WorkspaceNavigation.tsx` | Feature-oriented control-plane navigation |
| `apps/web/src/features/productexperience/ProductExperiencePanel.tsx` | Design, Knowledge, Impact, Visual QA, Environment, and Activity product experience |
| `apps/web/src/features/project/ProjectProvisioningWizard.tsx` | Create Project provisioning workflow |
| `apps/web/src/App.tsx` | Existing frontend composition, now connected to the new actions and panels |
| `apps/web/src/App.css` | Navigation, worker, governance, impact, Visual QA, and loop state styling |
| `scripts/frontend_backend_contract_audit.mjs` | Deterministic frontend/backend path and SSE contract audit |
| `FRONTEND_BROWSER_SMOKE_FINDINGS.md` | Browser smoke-check evidence and limitations |

## Conclusion

The frontend now presents the requested SUDA FORGE product experience while remaining a truthful client of the existing PostgreSQL-authoritative backend. Every added action maps to an existing backend route, every environment-dependent capability preserves its blocked state, and no frontend-only implementation can bypass Governance, RuntimeProvider, Verification, Deployment, or persistence boundaries.
