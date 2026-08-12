# SUDA FORGE — Phase 7 Final Report

## Implementation status

Phase 7 adds the **Hosting, Deployment & Local Infrastructure Fabric** on top of frozen Phases 1–6. The implementation keeps the modular-monolith architecture, reuses the existing Project Computer `RuntimeProvider`, uses the existing event bus and SSE stream, and does not introduce microservices, Kubernetes, a second orchestrator, a second model registry, or a second agent execution system.

The deployment path is represented as:

> CODE → BUILD → TEST → VERIFY → RELEASE → DEPLOY → HEALTH → TRAFFIC → MONITOR → ROLLBACK

## Files created

| File | Purpose |
|---|---|
| `internal/deployment/types.go` | Service, port, preview, domain, certificate, environment, release, deployment, health, snapshot, and provider-neutral interfaces. |
| `internal/deployment/infrastructure.go` | Runtime-scoped service discovery, port registry, port conflict prevention, hostname validation, and SSRF-safe target validation. |
| `internal/deployment/providers.go` | Runtime health checker, Caddy ProxyProvider, Caddy certificate boundary, and local storage provider. |
| `internal/deployment/runtime_adapter.go` | RuntimeProvider-backed deployment adapter and authoritative verification gate adapter. |
| `internal/deployment/service.go` | Release/deployment state machine, pipeline, resource checks, verification gate, health gate, activation, and rollback. |
| `internal/deployment/store.go` | PostgreSQL persistence for services, ports, releases, and deployments. |
| `internal/deployment/events.go` | Existing event-bus adapter. |
| `internal/deployment/audit.go` | Composite event sink for realtime SSE plus durable PostgreSQL deployment audit events. |
| `internal/deployment/catalog.go` | Project-scoped in-process catalog for domains, certificates, previews, and related lifecycle state. |
| `internal/deployment/deployment_test.go` | Unit and contract-style tests for port conflicts, SSRF validation, verification gates, health gates, and activation. |
| `internal/httpapi/deployment.go` | Project-scoped deployment, service, port, preview, domain, certificate, and health-check handlers. |
| `migrations/007_deployment.sql` | PostgreSQL schema for Phase 7 entities and deployment events. |
| `tests/phase7-go-test.txt` | Final backend test output. |
| `tests/phase7-web-build.txt` | Final frontend production build output. |
| `tests/phase7-architecture-check.txt` | Final architecture boundary output. |

## Files modified

The server bootstrap, HTTP server contract, configuration loader, dashboard component, dashboard stylesheet, and README were updated. The existing runtime, agent, model, orchestration, verification, and AI Fabric implementations were not replaced or redesigned.

## Database migration

Migration `007_deployment.sql` creates project-scoped tables for environments, services, ports, previews, domains, certificates, releases, deployments, deployment events, health checks, and snapshots. Existing `projects` are reused as the owning project boundary. Foreign keys and a unique `(protocol, external_port)` constraint prevent cross-project port conflicts at the persistence layer.

## New domain entities

Phase 7 introduces `Service`, `PortBinding`, `Preview`, `Domain`, `Certificate`, `EnvironmentConfig`, `Release`, `Deployment`, `HealthCheck`, and `Snapshot`. Deployment statuses cover `PENDING`, `PREPARING`, `BUILDING`, `TESTING`, `VERIFYING`, `DEPLOYING`, `HEALTH_CHECK`, `ACTIVE`, `FAILED`, `ROLLED_BACK`, and `CANCELLED`.

Deployments are separate from immutable releases. A release records the Git revision, artifacts, build metadata, environment, configuration version, and verification reference. A deployment points to a release and records runtime target, strategy, status, health, failure, timestamps, and audit metadata.

## New interfaces

The provider-neutral interfaces are `DeploymentProvider`, `RuntimeProvider`, `NetworkProvider`, `ProxyProvider`, `CertificateProvider`, `StorageProvider`, `ServiceDiscovery`, `PortRegistry`, `HealthChecker`, and `AuditSink`.

The implementation keeps provider-specific configuration inside adapters. The Caddy adapter owns Caddy admin API paths and reverse-proxy route payloads. Deployment commands and health commands execute through `RuntimeProvider.Exec` against the Project Computer; no HTTP handler executes host shell commands.

## New adapters

The following adapters are implemented:

| Adapter | Behavior | Current environment state |
|---|---|---|
| `RuntimeServiceDiscovery` | Runs a fixed `ss -ltnH` discovery command through the Project Computer runtime. | Contract implemented; live LXC execution is blocked in the Docker sandbox. |
| `RuntimeHealthChecker` | Executes health commands through the Project Computer runtime and records pass/fail evidence. | Contract implemented; live runtime execution is environment-dependent. |
| `CaddyProxy` | Uses Caddy's HTTP admin API for route creation, update, deletion, and URL rendering. | Requires an explicitly configured `CADDY_ADMIN_URL`; unavailable here. |
| `CaddyCertificate` | Provides Caddy/Let's Encrypt issuance, renewal, and status boundaries. | Requires controlled Caddy infrastructure and DNS/HTTPS reachability. |
| `RuntimeDeploymentProvider` | Performs deployment file/revision operations through the Project Computer runtime. | Requires a real supported runtime. |
| `LocalStorage` | Provides local volume and snapshot boundaries under a configured root with path traversal protection. | Implementation is local-first and does not claim remote backup. |

## Deployment pipeline and safety

The deployment manager requires explicit build and test command arrays in deployment metadata. Commands are validated as argv arrays and executed only through `RuntimeProvider`. The pipeline stops on build, test, verification, deployment, or health failure.

Activation is fail-closed. A deployment must contain a `verification_run_id`, and the server checks the persisted Phase 5 verification run status for `PASSED`; a client-provided success flag is not trusted. A deployment must also provide a health target and pass a runtime-scoped health check before entering `ACTIVE`.

Rollback is first-class and does not destroy the previous active release. It requests deployment of the selected previous release and records a `deployment.rolled_back` event. The implementation supports the strategy enum `RECREATE`, `BLUE_GREEN`, and `ROLLING`; the current safe local implementation uses the reliable recreate path unless a future runtime provider adds stronger traffic-switching capabilities.

## Service discovery and routing

Service discovery is runtime-scoped and language/framework agnostic. It records runtime, protocol, host, port, status, environment, process identity where available, and metadata. The in-memory port registry prevents conflicts before persistence, while migration 007 adds a database uniqueness constraint.

Preview routes are project-scoped. The preview API requires a service identity and hostname validation, and routes are created through `ProxyProvider`. The route layer rejects localhost, loopback, private-network, metadata-style, `.internal`, IP-literal, path-traversal, and malformed hostname/target inputs. No arbitrary host service can be exposed through a project preview.

## API endpoints

| Endpoint | Purpose |
|---|---|
| `GET /api/projects/{project}/services` | List persisted project services. |
| `POST /api/projects/{project}/services/discover` | Discover services inside a specified Project Computer runtime. |
| `GET /api/projects/{project}/ports` | List project port bindings. |
| `POST /api/projects/{project}/ports` | Reserve an external port with conflict validation. |
| `POST /api/projects/{project}/previews` | Create a proxy-backed preview route. |
| `GET /api/projects/{project}/previews` | List project previews. |
| `DELETE /api/projects/{project}/previews/{preview}` | Delete a proxy-backed preview route. |
| `POST /api/projects/{project}/domains` | Validate and create a project domain. |
| `GET /api/projects/{project}/domains` | List project domains. |
| `POST /api/projects/{project}/domains/{domain}/certificate` | Request provider-backed certificate issuance. |
| `POST /api/projects/{project}/health-checks` | Run a runtime-scoped health check. |
| `POST /api/projects/{project}/deployments` | Create and asynchronously start a deployment pipeline. |
| `GET /api/projects/{project}/deployments` | List deployment history from PostgreSQL. |
| `GET /api/projects/{project}/deployments/{deployment}` | Read an in-memory deployment state with project ownership validation. |
| `POST /api/projects/{project}/deployments/{deployment}/rollback` | Request rollback to a deployment release. |
| `GET /api/projects/{project}/releases` | List releases for a project. |

All events use the existing `/api/v1/events` SSE stream and the existing event bus. Deployment events are also inserted into `deployment_events` for auditability.

## UI changes

The dashboard now includes a real **Deployment Workspace**. It supports environment selection, version, source revision, runtime target, health target, deployment submission after authoritative verification, lifecycle status, health status, failure reason, deployment history, and rollback control. It uses live API state and displays environment-blocked failures rather than fabricating active deployments.

## Tests executed

| Command | Result |
|---|---|
| `go test ./...` | Passed. |
| `pnpm --dir apps/web build` | Passed. |
| `bash scripts/check-architecture.sh` | Passed with the existing advisory about frontend adapter labels. |
| `git diff --check` | Passed. |
| Deployment package tests | Passed: port conflicts, SSRF validation, verification gating, health gating, and activation lifecycle. |

## Tests blocked by environment

A real LXC deployment, real Caddy admin operation, Let's Encrypt issuance, and runtime service discovery require infrastructure unavailable inside the current restricted Docker environment. These are reported as **ENVIRONMENT-BLOCKED**. No fake deployment, fake Caddy success, fake certificate, or fake LXC success was generated.

The exact manual acceptance procedure on a supported host is: configure a real unprivileged LXC runtime, run migrations 001–007, start a controlled Caddy admin endpoint, configure `CADDY_ADMIN_URL`, create a Project Computer, run Phase 5 verification, discover services through the runtime, reserve ports, create a preview, issue a certificate, submit a deployment with explicit build/test commands and `verification_run_id`, observe `ACTIVE` only after the runtime health check passes, then stop the service and confirm the deployment remains auditable and rollbackable.

## Security considerations

Project-provided filesystem paths are not used as storage roots; storage roots are server configuration. Runtime commands execute against explicit runtime IDs through `RuntimeProvider`. Proxy routes require trusted project service IDs and validated hostnames. Private, loopback, metadata, and arbitrary host targets are rejected. Deployment activation cannot be authorized by a client success flag. Provider credentials and certificate private keys are not stored as ordinary database strings.

## Remaining Phase 7 limitations

The current environment prevents live LXC/Caddy/certificate execution. The deployment catalog for domains and previews is currently in-process while service, port, release, and deployment history are PostgreSQL-backed; migration 007 provides the persistence foundation for completing catalog persistence. Full blue-green/rolling traffic switching remains a provider capability boundary, with recreate as the safe initial strategy. Backup/restore contracts are present through `StorageProvider`, while a production snapshot backend requires a supported Project Computer and storage root.

Phase 8 has not been started.
