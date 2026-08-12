# SUDA FORGE — Phase 2 Status

**Phase:** Agent Fabric Kernel and Model Fabric foundation  
**Status:** Implemented and tested within the current environment. Phase 3 is not started.

## Implemented

The project now separates **Agent**, **Model**, **Provider**, **AgentConfiguration**, **CredentialReference**, **PermissionPolicy**, **AgentSession**, and normalized **AgentEvent** as independent concepts. Agents are represented by declarative definitions and adapter implementations; models belong to providers and are referenced through model configuration rather than a single `agent.model_id` field.

The Agent Adapter SDK covers start, stop, resume, message dispatch, event streaming, cancellation, status, capabilities, installation, credential resolution, and runtime-scoped process management. Codex, Claude Code, and Kimi have explicit CLI adapter contracts. Their adapters delegate process execution to a runtime-scoped `ProcessManager`; they do not execute host commands. Actual CLI execution remains environment-dependent and is not claimed as passing.

A `MockAdapter` exists as **TEST ONLY**. It emits normalized events and is not wired as a production runtime. A shared normalizer preserves raw provider data while exposing normalized SUDA FORGE events such as messages, thinking, tool activity, errors, and approval-required actions.

Agent sessions have an explicit persisted lifecycle: `CREATED`, `STARTING`, `RUNNING`, `PAUSED`, `WAITING`, `COMPLETING`, `COMPLETED`, `FAILED`, `CANCELLED`, and `RECOVERING`. Invalid transitions are rejected. A failed adapter/process start is persisted as `FAILED`, never `RUNNING`.

The API now exposes provider-neutral endpoints for Agent Registry, Provider Registry, Model Registry, project-scoped session creation and listing, start, messages, cancellation, and session events. The frontend includes the first functional Agent surface for project and agent selection, session creation, message submission, and normalized event display.

## Database changes

Migration `migrations/002_agent_fabric.sql` adds `agent_definitions`, `agent_adapters`, `agent_sessions`, `agent_session_events`, `providers`, `models`, `model_capabilities`, `agent_configurations`, and `credential_references`, with explicit foreign keys and project-scoped session/event queries. It seeds declarative Codex, Claude Code, Kimi, and Custom provider entries without storing raw credentials.

## Security boundaries

Agent processes can only be started through the runtime-scoped process contract, whose intended path is `AgentAdapter → RuntimeProcessManager → RuntimeProvider.Exec → LXCProvider → Project Computer → CLI`. The Agent core imports no LXC implementation or host process executor. Credential records contain references such as secret names, not raw secret values. Permission policies are project-scoped and support approval-required actions.

## Test results

| Area | Result |
|---|---|
| Go unit tests | PASS |
| Agent session lifecycle tests | PASS |
| Mock adapter normalized-event tests | PASS |
| Codex/Claude/Kimi adapter contract tests | PASS |
| Raw plus normalized event tests | PASS |
| Agent permission and credential-reference tests | PASS |
| Provider/model registry tests | PASS |
| PostgreSQL migration and AgentSession integration tests | PASS |
| Static architecture boundary check | PASS |
| React + TypeScript production build | PASS |
| Live Agent Registry endpoint | PASS |
| Live project-scoped session creation | PASS |
| Failed runtime process start persisted as `FAILED` | PASS |
| Real CLI execution in LXC | BLOCKED BY ENVIRONMENT |
| Real Agent PTY/process execution | BLOCKED BY ENVIRONMENT |

The current Docker-backed host is deliberately not used to fake LXC or CLI success. Phase 2 therefore validates contracts, persistence, normalization, API behavior, and security boundaries only.

## Repository changes

The main additions are under `internal/agent`, `internal/model`, `internal/httpapi`, `migrations/002_agent_fabric.sql`, `scripts/check-architecture.sh`, and the Phase 2 test files. The repository remains `/home/ubuntu/suda-forge`; the next commit will contain this Phase 2 milestone.

## Not implemented by design

Agent Council, Task Graph, autonomous orchestration, automatic model routing, repair loops, sophisticated memory, RAG, local GPU management, deployment automation, advanced browser verification, self-healing, and autonomous production deployment remain outside Phase 2.

## Recommended next phase

Before Phase 3, run the real LXC Project Computer acceptance suite on the supported Linux VM/host from Phase 1. After that runtime is proven, the next architectural step is runtime-backed AgentProcessManager validation and real CLI adapter installation/authentication tests. No provider-specific model router or multi-agent orchestration should be introduced before that boundary is verified.
