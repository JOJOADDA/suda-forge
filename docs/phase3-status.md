# SUDA FORGE — Phase 3 Status

**Phase:** Intelligent Model Fabric and Deterministic Routing Engine  
**Status:** Implemented and verified. Phase 4 is not started.

## Architecture

Phase 2’s Model Registry remains intact and is complemented by a provider-independent `internal/routing` layer. A `ModelProfile` is independent from Agent, Provider, Runtime, and Project. The router receives a normalized `RoutingRequest` and returns a `RoutingDecision`; it never calls an LLM and contains no provider SDK or API logic.

The routing pipeline is deterministic: validate the request, apply privacy and local-policy constraints, remove unavailable or incompatible models, enforce context/tool/budget/runtime constraints, apply a compatible user override, score remaining candidates, sort deterministically, and return a primary model with alternatives, rejected candidates, constraints, cost, confidence, and explanation.

## Capability and task system

The capability matrix is represented as a typed capability map rather than scattered booleans. It includes coding, reasoning, architecture, debugging, refactoring, frontend, backend, database, DevOps, security, testing, documentation, vision, tool use, structured output, long context, and fast response. `TaskProfile` provides deterministic structured task input for code, refactor, debug, architecture, UI, database, DevOps, security, testing, documentation, and general work.

## Health, availability, pricing, and cost

Provider health is represented by a cacheable `ProviderHealth` abstraction with status, latency, rate limits, authentication status, and timestamp. Model availability is separate from registration and supports `AVAILABLE`, `DEGRADED`, `UNAVAILABLE`, `DISABLED`, and `UNKNOWN`. Pricing includes input/output cost, currency, pricing unit, and effective date. `TokenCostEstimator` provides a replaceable estimate based on input and output token estimates; pricing is not hardcoded into policy logic.

## Policies and privacy enforcement

The router supports `BEST`, `CHEAPEST`, `FASTEST`, `PRIVACY_FIRST`, `BALANCED`, and `CUSTOM`, plus `LOCAL_FIRST`, `LOCAL_ONLY`, and `REMOTE_ALLOWED`. Task, project, organization, and global policy fields are represented with the documented precedence foundation. Private projects and private tasks cannot route to remote models. `LOCAL_ONLY` returns `NO_COMPATIBLE_MODEL` when no local model satisfies the hard constraints. User overrides are accepted only when the selected model passes all hard constraints.

Hard constraints eliminate candidates, including privacy violations, unavailable models, runtime unavailability, local-only violations, insufficient context, missing vision/tool capability, agent incompatibility, and budget limits. Soft preferences affect ranking and explanation. The score is transparent and combines capability match, policy preference, reliability, latency preference, and cost preference.

## Database and API

Migration `migrations/003_model_routing.sql` adds `model_pricing`, `model_health`, `routing_policies`, `routing_decisions`, and `model_usage_events`. Routing decisions are persisted for auditability without credentials or secret values. The API exposes `GET /api/models`, `GET /api/models/{id}`, `GET /api/providers`, `GET /api/providers/{id}`, and `POST /api/model-routing/decide`.

## Frontend and simulator

The frontend now contains a functional Model Center showing registered models, provider, local/remote state, context, coding, and tool-use metadata. It includes a routing preview that submits a deterministic refactor task and displays the selected model, alternatives, estimated cost, and explanation. `cmd/routing-sim` provides a development/testing simulator and emits a JSON decision with reasons and alternatives.

The live simulator selected `ollama/local-code` for a balanced, local-first refactor preview and retained `openai/cloud-best` as an alternative. A live API call produced the same deterministic selection and persisted one routing decision in PostgreSQL.

## Verification

| Area | Result |
|---|---|
| Go unit tests, including prior phases | PASS |
| Routing determinism and policy matrix | PASS |
| Privacy, local-only, budget, context, availability constraints | PASS |
| Provider health and cost abstractions | PASS |
| PostgreSQL migrations | PASS |
| Routing decision persistence | PASS |
| Live routing API | PASS |
| Routing simulator | PASS |
| Architecture boundary check | PASS |
| React + TypeScript production build | PASS |
| Real provider API calls | NOT REQUIRED / NOT CLAIMED |
| LXC-backed model execution | ENVIRONMENT-DEPENDENT |

## Remaining limitations

Provider discovery adapters are contracts only; no live Ollama, vLLM, llama.cpp, OpenAI, Anthropic, or Google discovery call is enabled. Health values and routing profiles are currently registry/bootstrap metadata rather than continuously refreshed provider observations. The first cost estimator uses fixed token estimates supplied by the caller. Project and organization policy persistence is schema-ready but not yet exposed as a complete policy management UI. The router does not perform autonomous task classification, model execution, multi-agent orchestration, repair loops, memory, RAG, or GPU management.

## Next phase boundary

Phase 4 must not begin until this report is accepted. The next recommended technical step is to validate provider discovery and cached health adapters against configured local/cloud endpoints, then connect routing decisions to a real model/provider execution boundary without changing the deterministic router.
