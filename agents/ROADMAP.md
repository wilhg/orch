## Roadmap (Go 1.25 AI Agent Framework)

This roadmap prioritizes a production-quality MVP, then incremental milestones to broaden capabilities. Each milestone includes verifiable exit criteria. Cross-references refer to `12-factor-agent-framework-requirements.md` and `ARCHITECTURE-AND-STANDARDS.md`.

### Legend
- [M] Milestone
- [T] Task (deliverable)
- Exit Criteria: verifiable checks required for milestone completion

---

## [M1] MVP Runtime Core (Reducer, Effects, State, HTTP, OTel)

Scope: Minimal, reliable runtime that executes reducer/effect cycles with durable state, HTTP control plane, and observability.

[T1] Initialize module and repo layout (cmd/, pkg/, examples/, docs/)
[T2] Define core contracts: Event, State, Intent, Reducer, EffectHandler (F12, F05)
[T3] Implement event-sourced state with Postgres store and snapshots (F05)
[T4] Implement runtime loop: apply event -> reducer -> intents -> effect handlers (F08, F12)
[T5] Idempotency keys in intents/effects; exactly-once semantics (F00, F11)
[T6] Checkpoints and resume semantics (F06)
[T7] HTTP/JSON control plane: create run, get state, pause/resume (F11)
[T8] OpenTelemetry traces/metrics/logs; structured logging (F00)
[T9] Example agent: simple todo or RAG-stub with mock tool (F01, F04)
[T10] CI with vet, lint, race, unit tests
[T11] Compact error model scaffold in runtime and HTTP control plane (F09)

Refactors (carry into M2/M3)
- Promote a flow orchestration layer (graph/state machine) atop reducers for complex control flow and resumable steps.
- Compact error model: structured categories/codes and policies (retry, DLQ, alert) enforced uniformly across runtime and tools.
- Unify tool permission model for local and MCP tools with identical validation and error semantics.

Exit Criteria
- `go test ./... -race` passes with core packages >80% coverage
- End-to-end example run succeeds via HTTP trigger; pause/resume verified without duplicate effects
- OTel traces visible in a local collector; logs/metrics emitted
- Deterministic replay test passes on captured run (F00-AC1)
 - Errors returned by runtime and control plane conform to compact error model (F09-AC1)

Verification Plan
- Unit: reducers/effects with golden snapshot diffs; registry and idempotency guards.
- Integration: Postgres-backed event store; HTTP control plane; pause/resume; duplicate trigger coalescence.
- Replay: capture run fixtures and assert state/output parity.
- Observability: validate spans for run→step→tool and required attributes; metrics presence.

---

## [M2] Tooling & Prompts (Typed Tools, Prompt Artifacts, Validation)

Scope: First-class tool calls and prompt management with schema validation and basic offline evaluation.

[T1] Tool interface with JSON Schema input/output and permission model (F01, F04)
[T2] Tool registry and safe invocation path with validation errors surfaced (F04)
[T3] Prompt artifact store with versioning and linting (F02)
[T4] Offline eval harness with fixtures for prompts (F02)
[T5] Extend example to call 1–2 real tools (HTTP call, filesystem sandbox)
[T6] MCP client support: handshake, listTools/resources/prompts, callTool
[T7] MCP server support: expose local tools/resources/prompts to external MCP clients
[T8] ReAct agent template and docs (inspired by Eino) with typed tools and checkpoints
[T8] ReAct agent template and docs (inspired by Eino) with typed tools and checkpoints

Exit Criteria
- Invalid tool inputs/outputs produce structured errors in traces (F04-AC1)
- Prompt update increments version, diff visible; lint blocks bad templates (F02-AC1/2)
- Offline eval runs on fixtures and reports scores in CI (F02-AC3)
 - MCP conformance tests: client and server modes pass handshake and tool invocation flows (F00-AC6)
 - ReAct template compiles and passes tests; example run demonstrates reasoning-act-observe loop
 - ReAct template compiles and passes tests; example run demonstrates reasoning-act-observe loop

Verification Plan
- Tool schema validation unit tests; permission-denied path tests.
- Prompt linting and version bump tests; diff rendering.
- Offline eval job on fixtures with thresholds; CI artifact of scores.
 - MCP tests using recorded fixtures and a mock server/client to validate JSON-RPC, error semantics, and parity with local tools.

---

## [M3] Context Assembly & Caching (Own Your Context Window)

Scope: Deterministic, observable context assembly with retrieval and caching hooks.

[T1] Context assembler with pinning, deduplication, and token budgeting (F03)
[T2] Embeddings/Vector adapters (OpenAI, pgvector) with pluggable interface
[T3] Cache layer for retrieval and tool results with TTL and metrics
[T4] Update example to demonstrate citations and deterministic assembly logs
[T5] Flow orchestration API (beta): branching, fan-out/fan-in, halt conditions, human interrupts with checkpoints
[T5] Flow orchestration API (beta): branching, fan-out/fan-in, halt conditions, human interrupts with checkpoints

Exit Criteria
- Context logs enumerate sources, chunk IDs, token counts (F03-AC1)
- Dedup works deterministically on repeated sources (F03-AC2)
- Cache reduces p95 latency vs baseline by documented % (F03-AC3)
 - Flow orchestration demo shows resumable steps and explainable branch rationale
- Flow orchestration demo shows resumable steps and explainable branch rationale

Verification Plan
- Determinism tests for pinning/dedup/token budget; citation presence.
- Perf harness to measure baseline vs cache-enabled p95 on RAG flow; enforce threshold.

---

## [M4] Triggers & Transports (Trigger From Anywhere)

Scope: Multiple trigger pathways and internal transport for scale-out workers.

[T1] CLI trigger (stdin/JSON) and replay runner (F11)
[T2] Webhooks with HMAC signatures and idempotency (F11)
[T3] gRPC internal API between runtime and workers
[T4] WebSocket session server for interactive runs (F07)
[T5] Scheduler for retries, backoff, and concurrency limits (F06, F08)

Exit Criteria
- Same example runs via HTTP, CLI, and webhook with canonical envelope (F11-AC1)
- Duplicate deliveries coalesced by idempotency key (F11-AC2)
- Parallel branches demonstrated with fan-out/fan-in; retries/backoff visible in traces (F08-AC2)

Verification Plan
- Canonical envelope conformance tests across transports.
- Idempotency tests with duplicate delivery.
- gRPC worker integration tests; WebSocket interactive session mock.

---

## [M5] Human-in-the-Loop & Approvals (Contact Humans With Tools)

Scope: Structured human interactions for clarifications/approvals integrated into runs.

[T1] Slack adapter/tool for questions and approvals with schema (F07)
[T2] Human response mapped to events and state timeline (F07)
[T3] Audit trail for approvals (who/when/decision) (F07)

Exit Criteria
- Example flow pauses for Slack question and resumes on reply (F07-AC1)
- Approval records are auditable and visible in state/traces (F07-AC2/3)

Verification Plan
- Mock Slack adapter tests for question/approval flows.
- Audit artifact checks (actor, time, decision) and timeline visibility.

---

## [M6] Marketplace-Ready Extensibility (Adapters & Plugins)

Scope: Stable plugin APIs, registries, and contribution guide for adapters.

[T1] Generic registry with factory pattern and capability descriptors
[T2] Adapter interfaces: LLM, Embedder, VectorStore, EventBus, CheckpointStore
[T3] Out-of-process plugin support (optional) for untrusted providers
[T4] Contribution guide and example adapter templates

Exit Criteria
- Two third-party adapters built using templates without core changes
- Compatibility matrix documented; semantic versioning adopted

Verification Plan
- Registry resolution and capability/permission checks.
- Build two adapters from template and run conformance tests; no core diffs.

---

## [M7] Reliability & Ops Hardening

Scope: Production concerns—backpressure, health, graceful shutdown, SLOs, alerts.

[T1] Health endpoints and readiness gates; graceful draining
[T2] Backpressure controls and bounded queues with priority scheduling
[T3] Dead-letter queues and alerting hooks; compact error payloads (F09)
[T4] Multi-tenant isolation and RBAC for control plane

Exit Criteria
- Load test meets defined p95/p99 SLOs on reference flows
- Repeated failures route to DLQ with alerts and compact errors (F09-AC3)
- Tenants isolated in data and observability artifacts (FOP-AC3)

Verification Plan
- Load tests with backpressure and bounded queues; graceful shutdown tests.
- DLQ and alert hooks under repeated failures; compact error payload checks.
- Multi-tenant isolation tests across data and OTel attributes.

---

## [M8] Docs, DX, and Examples

Scope: First-class developer experience, docs, and example gallery.

[T1] CLI `orch` with init, run, resume, inspect, trace, eval
[T2] Scaffolding: `orch init agent` generates reducer/effects/tests
[T3] Example gallery: RAG, tool-use workflow, approval flow, multi-agent composition
[T4] Docs site: quickstart, tutorials, API references, cookbook

Exit Criteria
- New user can scaffold and run an agent in <5 minutes
- Examples cover 80% of common patterns and pass CI
- Docs include copy-paste snippets and architecture diagrams

Verification Plan
- CLI E2E: init→run→resume→inspect scripted test.
- Scaffolding generates compiling agent with tests; example gallery runs in CI.
- Link checker and snippet compile tests on docs.

---

## [M9] Security & Compliance

Scope: Secrets, redaction, authn/z, and audit trails.

[T1] Secrets providers (env, Vault, cloud SM); zero secrets in code
[T2] Request auth (API keys/JWT/OAuth2) with RBAC roles
[T3] Log redaction and PII handling policies; audit logs for privileged actions

Exit Criteria
- Secrets resolved from providers; scanners find no hardcoded secrets (F00-AC5)
- Control plane protected; RBAC enforced; audit trails complete

Verification Plan
- Secret provider integration tests; hardcoded secret scanner in CI.
- Webhook HMAC verification tests; RBAC policy tests; audit log assertions.

---

## [M10] Enterprise Scale (Optional Advanced)

Scope: Large-scale deployment patterns and performance optimization.

[T1] Sharding strategy for runs and queues
[T2] Horizontal autoscaling policies based on OTel metrics
[T3] Performance tuning playbook and benchmarks publication

Exit Criteria
- Reference deployment scales to target throughput with cost/latency benchmarks
- Autoscaling responds within target time to load spikes and stabilizes queues

Verification Plan
- Sharding correctness tests (no cross-shard leakage).
- Autoscaling simulation driven by OTel metrics; queue stabilization check.
- Benchmark suite publishing with `benchstat` comparisons.


