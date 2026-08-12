# SUDA FORGE — Phase 4 Status Report

## Outcome

Phase 4, **Agentic Orchestration Engine**, is implemented in the modular monolith. The system now models workflows as persisted dependency graphs, validates and plans them deterministically, selects compatible agents, routes model work through the existing routing fabric, executes tasks through an explicit executor contract, and exposes workflow operations through the REST API and dashboard.

The implementation does not introduce microservices, Kubernetes, host-shell access, or mock UI operations. LXC remains an explicit runtime capability that is blocked by the current restricted Docker environment; the orchestration layer remains testable through contract tests and the simulator.

## Implemented surface

| Area | Delivered behavior |
|---|---|
| Task domain | Task, TaskRun, TaskResult, artifacts, retry policy, deadlines, task status lifecycle, required capabilities, model hints, and agent hints. |
| Workflow domain | Workflow identity, project scoping, execution strategy, failure policy, concurrency limit, workflow status, and audit timestamps. |
| Graph validation | Duplicate IDs, missing dependencies, self-dependencies, cycles, and invalid task references are rejected deterministically. |
| Planning | Goal-to-task planning produces a stable DAG with analysis, implementation, verification, and delivery stages. |
| Scheduling | Dependency-driven readiness, bounded concurrency, retries, deadlines, cancellation, failure propagation, continue-independent policy, and terminal workflow status. |
| Agent selection | Project-scoped candidate filtering by enabled state, adapter availability, runtime availability, required capabilities, and deterministic tie-breaking. |
| Task execution | `TaskExecutor` composes agent selection, model routing, and adapter execution. The orchestrator never invokes a host shell directly. |
| Persistence | PostgreSQL migration and store methods persist workflows, tasks, dependencies, runs, artifacts, and approvals. |
| API | Workflow create, get, plan, run, cancel, approval resolve, and project-scoped SSE event endpoints are exposed alongside existing project, agent, and model APIs. |
| UI | The dashboard includes workflow planning, status display, task graph visibility, and project-scoped activity presentation backed by real API calls. |
| Verification | A runnable orchestration simulator exercises a multi-stage DAG and emits JSON evidence. |

## Verification performed

The backend test suite passes with `go test ./...`. The frontend production build passes with `pnpm --dir apps/web build`. The orchestration simulator completes a dependency chain and now reports `SUCCEEDED`; its output is stored in `tests/orchestration-simulator-result.json`. Existing routing, runtime, filesystem, auth, event, agent, and project tests also remain green.

The scheduler lifecycle bug discovered during final verification was corrected: after the final task batch completes, the scheduler now transitions the workflow to `SUCCEEDED` or `FAILED` instead of returning while leaving it `RUNNING`. Approval resolution was also corrected to load the existing persisted approval first, preserving the valid task and workflow references required by PostgreSQL constraints.

## Environment limitation

The current execution environment is a restricted Docker container and cannot provide the required unprivileged LXC kernel capabilities. This is not hidden or emulated. Runtime validation therefore remains split between host-validation scripts, provider contract tests, and the orchestration simulator. A host with user namespaces, cgroups, LXC tooling, and the configured storage/network profile is required for real Project Computer execution.

## Phase boundary

Phase 4 is complete. Phase 5 has **not** been started. The next phase may focus on production hardening, richer adapter-backed execution, operational observability, and host-level LXC acceptance testing, but those activities are intentionally outside this report.
