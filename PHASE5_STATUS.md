# SUDA FORGE — Phase 5 Final Report

## 1. Architecture changes

Phase 5 adds a provider-agnostic `internal/verification` package without redesigning Phases 1–4. The Verification Engine is authoritative: agent results and repair explanations are inputs, while only normalized verification outcomes determine pass or failure. Verification adapters receive a `ProjectContext` containing the runtime provider and runtime ID; they do not execute host commands directly. The existing realtime event bus is reused for verification and repair events.

The design preserves the existing separation between **Agent**, **Model**, **Runtime**, and **Project**. The repair bridge delegates execution to the existing orchestration `TaskExecutor` contract rather than introducing a second agent execution system.

## 2. Verification domain

The new domain contains `VerificationRun`, `VerificationCheck`, `VerificationResult`, `VerificationArtifact`, `FailureReport`, `RepairPlan`, `RepairAttempt`, `VerificationProfile`, structured evidence, retry policies, project state, and explicit task/workflow gates. Verification statuses are `PENDING`, `RUNNING`, `PASSED`, `FAILED`, `CANCELLED`, and `TIMED_OUT`.

Check types include build, typecheck, lint, format, unit, integration, E2E, runtime health, HTTP health, database health, browser, security, dependency, and custom checks. Check graphs are validated for duplicate IDs, missing dependencies, and cycles before execution.

## 3. Verification profiles

The engine provides `STRICT`, `STANDARD`, `FAST`, and `CUSTOM` policies. Profiles are defaults and can be fully overridden through the API with project-defined checks and commands. The engine does not infer or reject projects based on language. Commands are supplied as structured `argv` or a project-specific command configuration.

## 4. Check implementations and boundaries

Command-backed checks execute through `runtime.Provider.Exec` with a runtime ID, working directory, environment, and timeout. The command adapter covers build, typecheck, lint, format, unit, integration, E2E, custom, dependency, and security command checks. Runtime, HTTP, database, and browser adapters are explicit boundaries represented by an unsupported-environment adapter until their corresponding Project Computer capabilities are available. This produces an explicit environment failure rather than a false pass.

Known `NO_TESTS_FOUND` output is classified as a test failure. Runtime health and browser verification are not faked in the restricted Docker environment.

## 5. Evidence model

Results contain structured exit code, stdout, stderr, duration, test counters when available, HTTP status, URL, artifact path, commit SHA, and extensible structured metadata. Output is sanitized when obvious credential markers are present. Configured artifact paths are associated with the project, task, verification run, and check. The engine captures commit SHA and clean/dirty/unknown worktree state through runtime-scoped Git commands when the state is not supplied by the caller.

## 6. Failure classification and analyzer

Failures are normalized into build, test, type, lint, runtime, browser, database, security, timeout, environment, and unknown categories. Each report includes the check, command, exit code, output, affected files, suspected cause, severity, retryability, and structured identity. `DeterministicFailureAnalyzer` generates a structured repair plan without executing commands. The analyzer contract is replaceable by a Model Router-backed implementation without coupling the verification decision to model output.

## 7. Repair loop

`RepairLoop` implements a bounded `VERIFY -> ANALYZE -> REPAIR -> VERIFY` cycle. Maximum attempts are configurable and cannot loop indefinitely. Repair execution delegates to the existing orchestration `TaskExecutor` through `OrchestrationRepairExecutor`. Repair history is retained across re-verification runs, and `repair.started`, `repair.completed`, `repair.failed`, and `repair.max_attempts` events are emitted.

## 8. Regression strategy and gates

After a repair attempt, the failed verification profile is executed again, which reruns the configured failed check and its regression targets. `TaskVerified` and `WorkflowVerified` helpers prevent an agent-success result from being treated as proof. `RequiredVerificationGate` provides a deployment gate abstraction and does not implement deployment itself.

## 9. Commit and state association

Every run stores project ID, task/workflow references, commit SHA when available, worktree state, and changed files. When a Project Computer runtime is present, the engine obtains this state using runtime-scoped `git rev-parse HEAD` and `git status --porcelain` calls. If the runtime is unavailable, the state remains explicitly `unknown`; it is never fabricated.

## 10. API and realtime events

The following endpoints are implemented:

| Endpoint | Purpose |
|---|---|
| `POST /api/verifications` | Execute a project-scoped verification run and persist it. |
| `GET /api/verifications/{id}` | Retrieve a run with checks, results, failures, repairs, and state. |
| `GET /api/tasks/{id}/verifications` | List verification history for a task. |
| `POST /api/verifications/{id}/cancel` | Cancel a pending or running persisted run. |
| `POST /api/verifications/{id}/repair` | Run bounded failure analysis, repair delegation, and re-verification. |
| `GET /api/verifications/{id}/artifacts` | List artifacts associated with a verification run. |

The existing SSE infrastructure receives `verification.started`, `verification.check.started`, `verification.check.passed`, `verification.check.failed`, `verification.completed`, `verification.cancelled`, `repair.started`, `repair.completed`, `repair.failed`, and `repair.max_attempts` events.

## 11. Frontend changes

The dashboard now includes a real Verification Engine panel. It invokes the verification API, displays per-check statuses, required-pass/fail counts, runtime evidence, commit/worktree state, failure causes, and repair attempts. The panel does not display an agent claim as proof of completion. It reports the explicit environment failure produced when LXC or the Project Computer runtime is unavailable.

## 12. Database migration

`migrations/005_verification.sql` adds tables for verification runs, checks, failures, repairs, and artifacts with project, task, run, and check lineage. The PostgreSQL store supports save, get, task history, and artifact retrieval, including reconstruction of persisted check results and repair history.

## 13. Test results

The final verification matrix passed:

| Check | Result |
|---|---|
| `go test ./...` | Passed. |
| `go test ./internal/verification` | Passed, including build pass/fail, blocked dependencies, optional failures, repair bounds, unsupported browser boundary, cancellation, and evidence state tests. |
| `pnpm --dir apps/web build` | Passed. |
| `scripts/check-architecture.sh` | Passed. It emitted only the existing advisory about frontend adapter labels. |
| `git diff --check` | Passed. |

Raw command outputs are retained under `tests/phase5-go-test.txt`, `tests/phase5-web-build.txt`, and `tests/phase5-architecture-check.txt`.

## 14. Known limitations

The current environment is a restricted Docker container and cannot provide a reliable real unprivileged LXC Project Computer. Therefore, runtime, browser, and real agent CLI execution remain provider boundaries and are not claimed as operationally verified here. The command adapter and API are production-shaped and runtime-scoped, while deterministic tests cover the behavior without pretending that LXC exists.

The current default repair wiring deliberately fails explicitly when no runtime-backed agent executor is available. It does not bypass security boundaries or execute commands on the host. A production host with the existing Agent Fabric, Model Router, and Project Computer runtime must supply the concrete adapter-backed executor.

GPU management, local-model installation, advanced deployment, production hosting, autonomous maintenance, distributed orchestration, Kubernetes, microservices migration, and Phase 6 work were not started.

## Phase boundary

**Phase 5 is complete. Phase 6 has not been started.**
