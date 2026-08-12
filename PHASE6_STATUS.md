# SUDA FORGE — Phase 6 Final Report

## 1. Architecture changes

Phase 6 introduces `internal/aifabric` as a provider/runtime-independent AI inference fabric inside the existing modular monolith. It does not create a second Model Registry, Model Router, or Agent execution system. The AI Fabric supplies runtime discovery, model availability, health, resource information, inference contracts, and lifecycle operations to the existing Model Fabric and Agent Fabric.

The control-plane path is now:

> Project AI policy → existing deterministic Model Router → selected model/runtime → AI Fabric runtime adapter → normalized inference → Agent/Tool Policy → Project Computer tools.

Models do not execute filesystem, terminal, browser, Git, or database tools directly. Tool calls remain data returned to the Agent layer.

## 2. AI Runtime abstraction

`AIRuntime` defines the normalized lifecycle and inference contract: `Discover`, `Install`, `Remove`, `Start`, `Stop`, `Restart`, `Health`, `ListModels`, `LoadModel`, `UnloadModel`, `Generate`, `Stream`, `Embeddings`, `Capabilities`, and `Resources`. Runtime capabilities are explicit, so unsupported operations return an error instead of being silently simulated.

`RuntimeRegistry` is the single AI runtime registry. Discovered models are stored in this registry and are bridged into the existing `routing.ModelProfile` type through `RoutingProfiles`; no second routing engine is introduced.

## 3. Runtime adapters

The following real adapters are implemented behind the shared contract:

| Runtime | Adapter behavior | Environment status |
|---|---|---|
| Ollama | `/api/tags`, `/api/version`, `/api/generate`, `/api/pull`, `/api/show`, `/api/embeddings`, and streaming generation. | `BLOCKED_BY_ENVIRONMENT` unless an endpoint is explicitly configured and reachable. |
| vLLM | OpenAI-compatible discovery, health, chat generation, streaming, and embeddings. | `BLOCKED_BY_ENVIRONMENT` unless `VLLM_URL` is explicitly configured and reachable. |
| llama.cpp | OpenAI-compatible discovery, health, chat generation, streaming, and embeddings. | `BLOCKED_BY_ENVIRONMENT` unless `LLAMACPP_URL` is explicitly configured and reachable. |
| OpenAI-compatible | Reusable adapter used by vLLM, llama.cpp, and compatible gateways. | Configurable; no credentials are hardcoded. |

Runtime process start/stop/restart is deliberately not faked. The HTTP adapters return an explicit externally-managed-runtime error for process lifecycle operations. Normal server startup does not install Ollama, vLLM, llama.cpp, CUDA, drivers, or models.

Optional endpoint configuration is available through `OLLAMA_URL`, `VLLM_URL`, and `LLAMACPP_URL`.

## 4. Model Registry and profiles

The existing Phase 2 provider/model registry remains authoritative for existing providers and models. AI Fabric models carry runtime ID, provider ID, context window, maximum output, local/remote locality, privacy level, lifecycle state, normalized capabilities, latency, and resource requirements. The bridge maps those descriptors to the existing routing capabilities and health constraints.

The normalized AI capability vocabulary includes coding, reasoning, architecture, fast response, vision, long context, agentic/tool use, structured output, embedding, private, local, cheap, and general capabilities. Profiles are vendor-neutral.

## 5. Hardware and GPU discovery

`DiscoverHostResources` performs real host discovery for CPU cores, `/proc/meminfo` memory, root filesystem capacity, and NVIDIA GPU information through `nvidia-smi` when available. GPU discovery captures vendor, model, memory, driver, CUDA runtime, and compute capability. A missing NVIDIA binary is treated as unavailable hardware rather than successful GPU discovery.

`GPUAllocator` and `ContractGPUAllocator` provide an explicit project/runtime allocation boundary. The initial implementation validates memory, VRAM, and GPU requirements and returns a policy allocation record. It does not expose the host GPU indiscriminately and does not claim GPU execution.

## 6. Runtime and model health

Runtime health uses lightweight HTTP endpoints and records status, version when available, endpoint, latency, available models, last checked time, and errors. Model health snapshots derive model status from runtime health and lifecycle state without making expensive inference calls. Health is included in routing eligibility, and offline or unhealthy runtimes are removed from candidate selection.

## 7. Model lifecycle and installation

Model states include `DISCOVERED`, `AVAILABLE`, `INSTALLING`, `INSTALLED`, `LOADING`, `READY`, `UNLOADING`, `REMOVING`, and `FAILED`. Installation, loading, unloading, and discovery are explicit API operations. Ollama installation is implemented through its pull endpoint; OpenAI-compatible adapters return unsupported installation rather than downloading content through an assumed provider API.

No large model download occurs during normal SUDA FORGE startup. Installation requires an explicit API request.

## 8. Routing integration

The existing deterministic Router was extended, not replaced. It now considers runtime health, model memory and VRAM requirements, GPU availability, offline mode, local/cloud locality, and existing privacy, availability, capability, context, agent compatibility, and budget constraints.

New policy support includes `CLOUD_FIRST` in addition to the existing balanced, best, cheapest, fastest, privacy-first, local-first, and local-only controls. `DecideWithFallbacks` applies the same hard constraints to every fallback candidate, so a fallback cannot bypass privacy, budget, capability, resource, or availability rules.

Project AI settings are persisted and applied before routing. They can restrict providers, runtimes, models, local-only behavior, privacy policy, budget, preferred model, and policy.

## 9. Privacy and offline enforcement

Privacy constraints are applied before model selection. `PRIVATE` and `LOCAL_ONLY` requests reject remote models deterministically. Offline mode rejects remote candidates. Fallback selection reuses the same constraints and cannot elevate a remote model into a local-only project.

Credential references remain in the existing server-side credential infrastructure. AI Fabric does not place provider keys in the frontend or runtime configuration payloads.

## 10. Inference contract and streaming

`InferenceRequest` supports project, agent, task, model, runtime, messages, system prompt, tools, temperature, maximum tokens, structured-output metadata, vision inputs, context, and streaming. `InferenceResponse` normalizes content, tool calls, usage, latency, finish reason, request/model/runtime identity, and raw metadata.

Streaming is first-class. Runtime adapters return token/message/tool/completion events through a channel, and the API exposes SSE at `POST /api/ai/inference/stream`. Buffered inference is available at `POST /api/ai/inference`. Tool calls are preserved in the normalized response and are not executed by the model runtime.

Vision, structured-output, and embeddings are explicit capability boundaries. Unsupported capabilities return controlled errors rather than being silently emulated. Embeddings are exposed as a boundary only; no RAG or vector database was implemented.

## 11. Usage and resource accounting

Inference usage records provider/runtime/model/project/agent/task/request identity, input and output tokens, total tokens, duration, estimated cost, CPU time, GPU time, and tokens per second where available. Local inference can be recorded with zero estimated monetary cost while retaining resource usage fields. PostgreSQL persistence is provided for inference requests and usage.

## 12. Benchmarking and capability verification

`Benchmark` provides explicit, user-triggered latency, success, and token-rate measurement. It does not automatically run at startup or modify production routing state. `VerifyCapability` provides explicit `VERIFIED`, `UNVERIFIED`, and `FAILED` outcomes for capability probes such as tool calling.

## 13. APIs and realtime events

The following APIs are implemented:

| Endpoint | Purpose |
|---|---|
| `GET /api/ai/runtimes` | List configured runtimes with capabilities and health. |
| `GET /api/ai/runtimes/{id}` | Inspect one runtime. |
| `POST /api/ai/runtimes/{id}/start` | Request externally-managed runtime start. |
| `POST /api/ai/runtimes/{id}/stop` | Request externally-managed runtime stop. |
| `POST /api/ai/runtimes/{id}/health` | Perform and persist a lightweight health check. |
| `GET /api/ai/models` | List discovered AI models. |
| `GET /api/ai/models/{id}` | Retrieve one model descriptor. |
| `POST /api/ai/models/discover` | Discover models through configured runtimes. |
| `POST /api/ai/models/install` | Explicitly install a model where supported. |
| `POST /api/ai/models/load` | Load/warm a model where supported. |
| `POST /api/ai/models/unload` | Unload a model where supported. |
| `GET /api/ai/hardware` | Discover host resources. |
| `GET /api/ai/gpus` | Discover GPU resources. |
| `GET /api/ai/health` | Return runtime and model health. |
| `POST /api/ai/inference` | Execute normalized buffered inference. |
| `POST /api/ai/inference/stream` | Execute normalized SSE streaming inference. |
| `GET /api/projects/{project}/ai-settings` | Read project AI policy. |
| `PUT /api/projects/{project}/ai-settings` | Persist project AI policy. |

Normalized event types include `ai.runtime.health_changed`, `ai.runtime.started`, `ai.runtime.stopped`, `ai.model.discovered`, `ai.model.install_started`, `ai.model.install_completed`, `ai.model.load_started`, `ai.model.ready`, `ai.model.unloaded`, `ai.model.failed`, `ai.request.started`, `ai.request.token`, `ai.request.completed`, and `ai.request.failed`.

## 14. Database migration

`migrations/006_ai_fabric.sql` adds `ai_runtimes`, `ai_runtime_health`, `ai_hardware_resources`, `ai_gpu_resources`, `ai_model_installations`, `ai_model_health`, `ai_inference_requests`, `ai_inference_usage`, `project_ai_settings`, and `ai_model_capability_checks`. Existing `providers` and `models` tables are reused rather than duplicated.

## 15. Frontend changes

The dashboard now includes an AI Control Center showing configured runtime health, discovered models, context and locality, CPU/RAM/GPU hardware information, and project-level routing/privacy controls. It uses the live AI Fabric APIs and reports an empty/unavailable state when no local runtime is configured; it does not display simulated online or GPU status.

## 16. Tests executed

| Test | Result |
|---|---|
| `go test ./...` | Passed. |
| AI Fabric HTTP adapter tests | Passed, covering Ollama discovery, health, generation, resource validation, and registry registration. |
| Router regression tests | Passed, including offline exclusion, GPU/VRAM resource rejection, privacy, and deterministic routing. |
| `pnpm --dir apps/web build` | Passed. |
| `scripts/check-architecture.sh` | Passed with the pre-existing frontend adapter-label advisory. |
| `git diff --check` | Passed. |

Raw outputs are retained under `tests/phase6-go-test.txt`, `tests/phase6-web-build.txt`, and `tests/phase6-architecture-check.txt`.

## 17. Environment limitations and known gaps

The current development container does not expose reliable real LXC, NVIDIA GPU, CUDA, Ollama, vLLM, or llama.cpp services. These runtime executions are therefore **BLOCKED_BY_ENVIRONMENT**, not marked as PASS. The adapters, hardware probes, contracts, APIs, and deterministic tests are real, but this environment cannot validate external runtime success.

The current implementation does not include a production process supervisor for local runtime daemons. Start/stop/restart remains an explicit external-management boundary. It also does not implement automatic model installation, automatic benchmarking, full remote-provider inference adapters, full browser/vision execution, or provider credential resolution beyond the existing credential-reference seam. These are deliberate limitations rather than simulated behavior.

RAG, vector databases, model training, fine-tuning, distributed GPU clusters, multi-node inference, Kubernetes, microservices, agent councils, advanced deployment, and autonomous maintenance were not implemented.

## 18. Acceptance checklist

| Requirement | Status |
|---|---|
| AI Runtime abstraction | Implemented. |
| Ollama adapter | Implemented; live execution `BLOCKED_BY_ENVIRONMENT` here. |
| vLLM adapter | Implemented through OpenAI-compatible runtime; live execution `BLOCKED_BY_ENVIRONMENT` here. |
| llama.cpp adapter | Implemented through OpenAI-compatible runtime; live execution `BLOCKED_BY_ENVIRONMENT` here. |
| Existing remote providers preserved | Existing registry and provider seams preserved. |
| Existing Model Registry and Router extended | Implemented without duplication. |
| Model profiles and model/runtime health | Implemented. |
| Hardware, GPU, and resource model | Implemented with real detection and explicit unavailable states. |
| Model discovery and lifecycle | Implemented through explicit runtime operations. |
| Resource-aware, privacy-aware routing | Implemented in existing Router. |
| Local-first, cloud-first, cost, balanced, fallback, offline | Implemented. |
| Normalized inference and streaming | Implemented, including SSE. |
| Tool-call preservation | Implemented; model never directly executes tools. |
| Vision, structured output, embedding boundaries | Implemented as explicit capabilities/boundaries. |
| Usage and local resource accounting | Implemented as normalized persistence fields. |
| Benchmark and capability verification | Implemented as explicit user-triggered abstractions. |
| AI APIs and realtime events | Implemented. |
| AI Control Center and project AI settings | Implemented. |
| PostgreSQL migration | Implemented in migration 006. |
| Automated tests | Passed. |
| Fake production inference/GPU/runtime success | Not introduced. |
| Phase 7 | **NOT STARTED**. |

**Phase 6 is complete. The implementation stops here as required.**
