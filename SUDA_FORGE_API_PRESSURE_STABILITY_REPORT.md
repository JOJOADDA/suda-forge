# SUDA FORGE — API Pressure Stability Report

## Scope and safety boundary

This test covered the sensitive Autonomous Loop and Verification Engine HTTP paths against the current SUDA FORGE server build on a local PostgreSQL instance. The test used an isolated database project fixture and did not attempt to create an LXC Project Computer, execute arbitrary host commands, access a GPU, or claim production capacity.

The LXC-backed project creation endpoint was tested first and correctly returned an internal runtime failure because the current host is a restricted Docker environment. To exercise the PostgreSQL foreign-key and recovery paths without pretending that LXC was available, a project row was inserted directly into the local test database. All loop and verification requests then passed through the real HTTP server, coordinator, engine, PostgreSQL stores, and route handlers.

## Test setup

| Item | Value |
|---|---|
| Server | Current SUDA FORGE build on `127.0.0.1:18080` |
| Database | Local PostgreSQL `suda_forge` |
| Project fixture | `pressure-live-1786561384` |
| Pressure harness | `scripts/api_pressure_test.py` |
| Concurrent workers | 32 |
| Main batch | 256 requests per scenario |
| Recovery batch | 128 requests per scenario |
| Evidence | `tests_api_pressure_256.json`, `tests_api_pressure_resume_128.json`, `tests_api_pressure_final_loop.json` |

## Main 256-request results

| Scenario | Requests | Result | Latency p95 | Maximum | Interpretation |
|---|---:|---|---:|---:|---|
| Loop status reads | 256 | 256 × `200` | 18.08 ms | 26.27 ms | Stable persisted reads |
| Duplicate loop starts | 256 | 256 × `202` | 21.47 ms | 33.71 ms | Coordinator remains responsive; PostgreSQL retained one loop row |
| Malformed loop JSON | 256 | 256 × `400` | 21.81 ms | 31.82 ms | Input rejection is deterministic |
| Verification creates | 256 | 256 × `201` | 124.32 ms | 155.33 ms | All valid requests persisted failed runtime-scoped runs without 5xx responses |
| Malformed verification JSON | 256 | 256 × `400` | 23.08 ms | 28.56 ms | Input rejection is deterministic |
| Verification missing project ID | 256 | 256 × `400` | 24.14 ms | 33.37 ms | Required-field validation is stable |
| Unknown loop resume | 256 | 256 × `409` | 21.37 ms | 27.48 ms | Missing persisted loop is reported as conflict, not success |

The 256-request batch produced **zero transport errors and zero 5xx responses** for the valid fixture scenarios. Verification latency was higher than loop reads because each valid request exercised runtime-scoped verification and PostgreSQL persistence.

## Concurrent recovery results

After stopping and restarting the server, the previously persisted loop was readable with `200 OK` and retained `status: BLOCKED`, `current_stage: INTENT`, and the `BLOCKED_BY_ENVIRONMENT` result. This proves that the checkpoint survives process recovery.

A second 128-request concurrent batch then produced:

| Scenario | Requests | Result | Latency p95 | Maximum |
|---|---:|---|---:|---:|
| Loop status reads | 128 | 128 × `200` | 9.06 ms | 16.62 ms |
| Duplicate loop starts | 128 | 128 × `202` | 16.25 ms | 22.61 ms |
| Valid loop resumes | 128 | 128 × `202` | 14.77 ms | 17.65 ms |
| Malformed loop JSON | 128 | 128 × `400` | 15.84 ms | 19.10 ms |
| Verification creates | 128 | 128 × `201` | 109.09 ms | 142.18 ms |
| Malformed verification JSON | 128 | 128 × `400` | 16.82 ms | 22.38 ms |
| Verification missing project ID | 128 | 128 × `400` | 15.60 ms | 19.53 ms |
| Unknown loop resume | 128 | 128 × `409` | 15.48 ms | 18.65 ms |

The final loop remained one persisted row with `BLOCKED` status and `INTENT` as the current stage. No duplicate loop rows were created by concurrent starts or resumes. The database contained 416 persisted verification runs from the valid pressure batches, all with the expected runtime-blocked failure outcome.

## Race and process checks

The following package-level race tests passed:

```text
go test -race ./internal/productexperience ./internal/verification
```

The live server log contained no panic, fatal error, or unhandled server failure during the pressure batches. PostgreSQL activity returned to a small steady-state connection count after traffic, with no visible connection explosion.

## Stability conclusion

Within the tested local envelope of 32 concurrent workers and 256-request batches, the Autonomous Loop and Verification Engine HTTP paths were stable. Valid requests remained responsive, malformed requests were rejected consistently, unknown recovery targets returned conflicts, and persisted loop state remained recoverable after a server restart.

The test does **not** establish a production capacity limit, because the environment does not provide the real LXC runtime, browser, GPU, external agent CLI, Caddy, or production network topology. It also does not claim that a blocked loop can complete its later stages. The observed behavior is the correct truthful outcome: runtime-dependent stages stop at `BLOCKED_BY_ENVIRONMENT` and remain persisted for future recovery.

## Reproducibility

From the repository root, after preparing a local PostgreSQL database and starting the server on port 18080:

```bash
python3 scripts/api_pressure_test.py http://127.0.0.1:18080 <project-id> 256 32
python3 scripts/api_pressure_test.py http://127.0.0.1:18080 <project-id> 128 32

go test -race ./internal/productexperience ./internal/verification
```

The raw JSON evidence is included with this report in `tests_api_pressure_256.json`, `tests_api_pressure_resume_128.json`, and `tests_api_pressure_final_loop.json`.
