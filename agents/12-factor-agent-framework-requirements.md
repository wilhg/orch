## 12-Factor Agent Framework: Requirements and Acceptance Criteria

Source: [humanlayer/12-factor-agents](https://github.com/humanlayer/12-factor-agents)

### Purpose
Define actionable, testable requirements for an AI agent framework aligned to the 12-Factor Agents principles. All future implementation tasks must reference these requirement IDs and meet the acceptance criteria.

### Scope
Applies to core runtime, SDKs, dev tools, and ops components supporting design, execution, monitoring, and lifecycle management of AI agents.

### Conventions
- Requirement IDs: Fxx-Ry (factor x, requirement y)
- Acceptance Criteria: Fxx-ACz
- MUST/SHOULD/MAY per RFC 2119 semantics

---

## Cross-Cutting Requirements

- F00-R1 (Determinism & Replay): The framework MUST provide deterministic replays of agent runs given the same inputs, tool responses, and model outputs captured as fixtures.
- F00-R2 (Schema-first): All agent I/O, tool calls, events, state, and errors MUST be described with versioned JSON Schemas.
- F00-R2.1 (Compact Error Model): Errors MUST be compact and structured with `{category, code, message, context, cause[]}` and a max payload size; truncation policies MUST be defined.
- F00-R3 (Observability): The framework MUST emit structured logs, metrics, and traces across runs, steps, tool calls, and model invocations.
- F00-R4 (Idempotency): All triggers and tool integrations MUST support idempotency keys to prevent duplicate side effects.
- F00-R5 (Security): Secrets MUST be externalized; tools MUST define least-privilege scopes; PII handling MUST be configurable and auditable.
- F00-R6 (Extensibility): Factors MUST be implementable via extension points (adapters, middlewares, reducers, stores) with stable interfaces.
- F00-R6.1 (Flow Orchestration): The framework SHOULD provide an optional flow orchestration layer (graphs/state machines) atop reducers for complex control-flow and resumable steps.
- F00-R7 (MCP Interoperability): The framework MUST implement Model Context Protocol (MCP) client and server modes to interoperate with MCP tool/resource/prompt providers over WebSocket/HTTP.

Acceptance Criteria
- F00-AC1: Re-run of a captured execution produces identical state transitions and outputs.
- F00-AC2: JSON Schemas exist for run, step, event, state, tool-call, and error objects with semantic versions.
- F00-AC2.1: Error objects adhere to the compact error model with category/code and are enforced in CI.
- F00-AC3: A sample agent exposes logs, metrics, and distributed traces visible in a demo dashboard.
- F00-AC4: Duplicate delivery of the same trigger does not create duplicate side effects (verified via idempotency key).
- F00-AC5: Static analysis flags use of secrets in code; secrets are resolved from configuration providers at runtime.
- F00-AC6: MCP handshake, listTools/resources/prompts, and tool invocation conformance tests pass for both server and client modes.

---

## Factor 01 — Natural Language to Tool Calls

Requirements
- F01-R1: The framework MUST translate natural language inputs into strongly-typed tool calls using JSON Schema-validated arguments.
- F01-R2: Tool invocation MUST be first-class with audit logs including tool name, version, arguments, and outputs.
- F01-R3: Partial/ambiguous parses MUST surface as actionable validation errors rather than free-form text.

Acceptance Criteria
- F01-AC1: A sample NL prompt yields a validated tool-call payload that passes schema validation; invalid arguments are rejected with actionable messages.
- F01-AC2: Tool calls are recorded with name, version, args, output, duration, and correlation IDs in traces.
- F01-AC3: Ambiguous NL produces a structured validation error with missing/invalid fields enumerated.

## Factor 02 — Own Your Prompts

Requirements
- F02-R1: Prompts MUST be versioned artifacts with metadata (id, version, description, owners, change log).
- F02-R2: Prompt templating MUST support variables, conditionals, and composition with linting and unit tests.
- F02-R3: Offline prompt evaluation MUST be supported with fixtures and regression thresholds.

Acceptance Criteria
- F02-AC1: A prompt change increments a version and surfaces a diff; rollbacks are possible.
- F02-AC2: Prompt templates fail CI on unbound variables or lint errors.
- F02-AC3: Offline eval runs on a fixture set and gates merges when scores drop below thresholds.

## Factor 03 — Own Your Context Window

Requirements
- F03-R1: Retrieval and context assembly MUST be explicit, configurable, and observable.
- F03-R2: The framework MUST support pinning, deduplication, chunking, and citation of context items.
- F03-R3: Caching and pre-fetch hooks MUST be available to reduce latency (see Appendix 13).

Acceptance Criteria
- F03-AC1: A demo agent shows deterministic context assembly logs listing sources, chunk IDs, and token budgets.
- F03-AC2: Duplicate context items are deduplicated deterministically.
- F03-AC3: Enabling cache reduces p95 latency vs. baseline by a documented percentage on the sample workflow.

## Factor 04 — Tools Are Structured Outputs

Requirements
- F04-R1: Tool outputs MUST be typed objects validated against schemas; parsing errors MUST be first-class.
- F04-R2: The runtime MUST prevent tool execution on unvalidated or unsafe arguments.
- F04-R3: Tools MUST declare side-effect categories and required permissions.
- F04-R4: MCP tool calls MUST route through the same validation/permission checks as local tools with identical error semantics.

Acceptance Criteria
- F04-AC1: Invalid tool outputs raise structured parse errors captured in traces and error streams.
- F04-AC2: A tool cannot execute when required permissions are not granted; the run fails safely with a permission error.
- F04-AC3: Tool catalogs include JSON Schemas for inputs/outputs and a machine-readable permission model.
- F04-AC4: MCP tool invocations route through the same validation and permission checks as local tools, with identical error semantics.

## Factor 05 — Unify Execution State

Requirements
- F05-R1: A single canonical execution state model MUST represent goals, steps, pending actions, and results.
- F05-R2: All mutations MUST occur via events/actions; direct state mutation is forbidden.
- F05-R3: State persistence MUST be pluggable (memory, file, database) with identical semantics.

Acceptance Criteria
- F05-AC1: State snapshots before/after each step exist; diffs show only reducer-driven changes.
- F05-AC2: The same run produces identical states across different stores (memory vs. database).
- F05-AC3: Direct mutation attempts are blocked by lint/TypeScript/typing rules or runtime guards.

## Factor 06 — Launch, Pause, Resume

Requirements
- F06-R1: The runtime MUST support checkpoints and resumption from any checkpoint idempotently.
- F06-R2: Long-running steps MUST support heartbeats, timeouts, and backoff retries.
- F06-R3: Manual pause/resume MUST be exposed via API/CLI/UI.

Acceptance Criteria
- F06-AC1: A run paused mid-step resumes to completion with no duplicate side effects.
- F06-AC2: Steps exceeding timeout are retried with exponential backoff and jitter; max-attempts are enforced.
- F06-AC3: Operators can list, pause, and resume runs via CLI or HTTP endpoints.

## Factor 07 — Contact Humans With Tools

Requirements
- F07-R1: The framework MUST integrate with human channels (e.g., email, Slack, chat UI) as tools with schemas.
- F07-R2: Clarifying questions and approvals MUST be modeled as structured requests/responses.
- F07-R3: Human feedback MUST be appended to execution state as events.

Acceptance Criteria
- F07-AC1: A sample flow asks a clarifying question via Slack; the human reply unblocks the run.
- F07-AC2: Approval tools produce auditable records with approver id, timestamp, and decision.
- F07-AC3: Human events are visible in the state timeline and traces.

## Factor 08 — Own Your Control Flow

Requirements
- F08-R1: Control flow MUST be authored explicitly (state machines/reducers/graphs), not implicitly via raw LLM loops.
- F08-R2: The runtime MUST support branching, looping, parallelism, and halting criteria.
- F08-R2.1: Flow orchestration API SHOULD provide resumable steps and human interrupts with checkpoints.
- F08-R3: Control-flow decisions MUST be explainable with inputs and rationale recorded.

Acceptance Criteria
- F08-AC1: A sample agent declares its control flow; traces show branch decisions with inputs and reasons.
- F08-AC2: Parallel branches execute with fan-out/fan-in and aggregated results.
- F08-AC3: Halting conditions terminate runs deterministically and are visible in logs.

## Factor 09 — Compact Errors

Requirements
- F09-R1: Errors MUST be compact, structured, and categorized (validation, tool, network, model, policy, etc.).
- F09-R2: The framework MUST support root-cause attachment and remediation hints.
- F09-R3: Error policies (retry, dead-letter, alert) MUST be configurable per category.

Acceptance Criteria
- F09-AC1: Errors include category, code, message, context, and cause chain; max payload size enforced.
- F09-AC2: Policy config transforms a tool error into a single retry with dead-letter on failure.
- F09-AC3: An alert is emitted on repeated failures with a compact payload and deep link to traces.

## Factor 10 — Small, Focused Agents

Requirements
- F10-R1: Agents SHOULD be single-responsibility units with well-defined inputs/outputs and tool scopes.
- F10-R2: Composition MUST be supported via message-passing/events rather than shared mutable state.
- F10-R3: Agent registries MUST include metadata: purpose, owners, dependencies, SLAs.

Acceptance Criteria
- F10-AC1: A complex workflow is implemented as multiple small agents communicating via events.
- F10-AC2: Registry shows each agent’s contract and dependency graph.
- F10-AC3: Replacing one agent implementation does not affect others if the contract is unchanged.

## Factor 11 — Trigger From Anywhere

Requirements
- F11-R1: The framework MUST support triggers via HTTP, CLI, message queues, schedules, and webhooks.
- F11-R2: All triggers MUST share a canonical envelope with idempotency keys and correlation IDs.
- F11-R3: Replay and test harnesses MUST be able to inject triggers programmatically.

Acceptance Criteria
- F11-AC1: The same sample workflow runs when triggered via HTTP, CLI, and queue using the same payload schema.
- F11-AC2: Duplicate trigger delivery is coalesced by idempotency key.
- F11-AC3: A test harness invokes triggers and asserts final state and outputs deterministically.

## Factor 12 — Stateless Reducer

Requirements
- F12-R1: Next-state derivation MUST be implemented as a pure reducer: nextState = reducer(currentState, event).
- F12-R2: Reducers MUST be unit-testable with fixtures and golden snapshots.
- F12-R3: Side effects MUST be emitted as intents separate from state mutation.

Acceptance Criteria
- F12-AC1: Reducer unit tests pass against a fixture suite; snapshots verify deterministic updates.
- F12-AC2: Side effects are emitted as intents and executed by effect handlers; reducers remain pure.
- F12-AC3: Time-travel debugging replays events to reconstruct state.

## Appendix 13 — Pre-Fetch (Optional but Recommended)

Requirements
- F13-R1: Pre-fetch hooks MAY proactively retrieve likely-needed data and warm caches.
- F13-R2: Pre-fetch operations MUST respect privacy/policy constraints and be cancelable.

Acceptance Criteria
- F13-AC1: Enabling pre-fetch reduces end-to-end p95 on the sample by a documented percentage.
- F13-AC2: Pre-fetches are visible in traces and cancel when the plan changes.

---

## Operational Requirements

- FOP-R1: Config via environment and providers; no secrets in code.
- FOP-R2: Zero-downtime deploys; runs survive restarts via checkpoints.
- FOP-R3: Backpressure and concurrency controls at run/step/tool levels.
- FOP-R4: Multi-tenant isolation for data and rate limits.

Acceptance Criteria
- FOP-AC1: Restarting the runtime during a run does not lose progress; the run resumes from last checkpoint.
- FOP-AC2: Concurrency limits cap parallel tool calls; excess work is queued with backpressure.
- FOP-AC3: Tenants cannot access each other’s state, logs, or artifacts.

---

## Deliverables Checklist (for each new agent or feature)

- Requirements mapping: list of Fxx-Ry addressed.
- Schemas for inputs, outputs, tools, events, and state.
- Prompt artifacts with versions and tests.
- Reducer and effect handler unit tests with fixtures.
- Observability: logs, metrics, traces with correlation IDs.
- Operational docs: configs, permissions, SLAs, runbooks.

---

## Quality Gates

- CI blocks on schema validation, prompt lint/eval, reducer unit tests, and deterministic replay tests.
- Changes that alter control flow require trace diffs review.
- Error budgets and p95/p99 SLOs are defined and monitored for critical agents.

---

## References

- 12-Factor Agents (HumanLayer): https://github.com/humanlayer/12-factor-agents
- Related concepts: event sourcing, reducers, RAG, idempotency, circuit breaking, saga patterns.
- MCP specification: https://github.com/modelcontextprotocol/specification


