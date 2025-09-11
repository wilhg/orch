## Go 1.25 AI Agent Framework: Architecture, Standards, and Extensibility

This document defines the project standards, architecture, extensibility model, supported adapters and protocols, and strategies for developer adoption. It complements `12-factor-agent-framework-requirements.md` and is normative for all Go code in this repository.

### Language & Runtime
- Go 1.25 (minimum). Use modules. Enforce `GO111MODULE=on`.
- Supported platforms: linux/amd64, linux/arm64, darwin/arm64. Windows best-effort.
- Use `context.Context` pervasively. All I/O and long-running operations accept `ctx`.
- Avoid goroutine leaks: always pair goroutines with cancellation or wait groups.

### Repository Layout
```
cmd/                # Binaries (orchestrator, worker, cli)
pkg/
  agent/            # Agent contracts: reducer, effects, tools, prompts, schemas
  runtime/          # Execution engine, scheduler, checkpoints, resumptions
  transport/        # Protocol servers/clients (HTTP, gRPC, WS, MQ)
  adapters/         # Model, vector, datastore, queue, observability
  store/            # State stores and snapshots
  prompt/           # Prompt artifact management and evaluation
  eval/             # Offline eval harness
  otel/             # Observability instrumentation
internal/           # Private helpers, not for public import
examples/           # End-to-end examples and templates
scripts/            # Dev scripts
configs/            # Sample configs
docs/               # Additional documentation
```

### Project Standards
- Dependency management: `go.mod`, use minimal required versions, avoid heavy transitive deps.
- Linting/format: `gofmt`, `go vet`, `staticcheck`, `golangci-lint` (with sane defaults).
- Testing: `go test -race -shuffle=on`. Aim for >80% package coverage for core (`pkg/runtime`, `pkg/agent`).
- Benchmarks for hot paths: reducers, effect dispatch, context assembly, tool call marshaling.
- Errors: use `%w` wrapping, sentinel errors for categories, structured fields for context.
- Logging: structured, leveled logging with `slog` (stdlib) and OTel attributes.
- Configuration: env-first with config providers; never embed secrets in code.
- Schema-first: versioned JSON Schemas for events, state, tools, errors; enforce validation at boundaries.

### API Design: Convention over Configuration

- Default-first APIs: provide sensible defaults so most users need zero config.
- Explicit overrides: advanced behavior is enabled via functional options, not required params.
- Predictable naming and behavior: follow Go idioms and consistent naming across packages.
- Minimal knobs: avoid unnecessary flags; add them only when there is a proven need.
- Safe defaults: concurrency, timeouts, retries, and idempotency are enabled with conservative values.
- Discoverability: defaults and available options are documented in Godoc and examples.

### Core Architectural Concepts
- Reducer-centric runtime: `nextState = reducer(currentState, event)`; reducers are pure.
- Effect handlers execute intents emitted by reducers; idempotent and retry-aware.
- Event-sourced state: append-only event log with snapshots for fast recovery.
- Scheduler: manages steps, timeouts, retries, backoff, and concurrency limits.
- Checkpoints: durable restore points for pause/resume and crash recovery.
- Tool calls: schema-validated inputs/outputs, permission-scoped, auditable.
- Prompt artifacts: versioned, testable, and independently deployed.
- Observability first: traces across runs, steps, tool calls, model invocations.

Flow orchestration (planned refactor)
- Explicit flow layer (graph/state machine) above reducers for complex control flow: branching, fan-out/fan-in, halting conditions, and human interrupts.
- First-class checkpointers and resumability on flow steps (inspired by LangGraph/Eino).
- Deterministic step function with typed transitions and explainable branch rationale.

Exactly-once semantics (runtime)
- Idempotent triggers; dedup via idempotency keys.
- Intent execution claims (deterministic claim event) + completion markers to prevent duplicate side effects under concurrency.

Data model & stores
- Postgres: prefer JSONB columns for `payload`/`state`.
- SQLite: text JSON for `payload`/`state` (SQLite JSON functions work on text; JSONB optional in 3.45+).

### Public Interfaces (minimal signatures)

```go
// pkg/agent/contracts.go
type Event struct {
  ID string
  Type string
  Timestamp time.Time
  Payload any
}

type State interface {
  RunID() string
}

type Intent struct {
  Name string
  Args map[string]any
}

type Reducer interface {
  Reduce(ctx context.Context, current State, event Event) (next State, intents []Intent, err error)
}

type EffectHandler interface {
  CanHandle(intent Intent) bool
  Handle(ctx context.Context, s State, intent Intent) (events []Event, err error)
}

type Tool interface {
  Name() string
  InputSchema() []byte // JSON Schema
  OutputSchema() []byte
  Invoke(ctx context.Context, args map[string]any) (result map[string]any, err error)
}
```

Error model (compact)
- Errors are structured: `{ category, code, message, context, cause[] }` with a max payload size and truncation policy.
- Categories include: validation, tool, network, model, policy, system.
- Policies: retry/backoff, dead-letter, alert routing by category.

Tool permissions & MCP unification
- Tool descriptors include JSON Schemas (in/out) and capability/permission descriptors.
- All tool calls (local or MCP) go through the same validation/permission path with identical error semantics.

Security & secrets
- Credential providers (env, Vault, cloud SM); no secrets in code/logs.
- Least-privilege permissions per tool; RBAC at control plane endpoints.

### Extensibility Model
- Registry pattern: each pluggable type registers via `RegisterX(name string, factory)`.
- Dependency injection: constructor-based injection with functional options (no global singletons).
- Plugin loading: build-time (Go modules) and runtime (HashiCorp go-plugin for out-of-process isolation) for untrusted providers.
- Capability descriptors: plugins declare capabilities and required permissions; enforced at runtime.

```go
// pkg/runtime/registry.go
type ProviderFactory[T any] func(ctx context.Context, cfg map[string]any) (T, error)

func Register[T any](name string, f ProviderFactory[T])
func Resolve[T any](name string) (ProviderFactory[T], bool)
```

### Components and Adapters

- Models (LLMs): OpenAI-compatible, AWS Bedrock, Vertex AI, Ollama (local), vLLM.
- Embeddings/Vectors: OpenAI, Jina, Voyage; vector stores: pgvector, Qdrant, Milvus, Weaviate.
- Data Stores: PostgreSQL (primary), SQLite (dev), DynamoDB (optional), Redis (cache).
- Message Queues: NATS JetStream, Kafka, RabbitMQ, SQS.
- Observability: OpenTelemetry (OTLP), Prometheus metrics export, structured logs to stdout.
- Secrets/Config: env, AWS Secrets Manager, GCP Secret Manager, HashiCorp Vault.
- Transports: HTTP/JSON, gRPC, WebSocket; webhooks for triggers; CRON/scheduler; MCP (client and server roles).

Minimal adapter interfaces
```go
type LLM interface {
  Generate(ctx context.Context, prompt string, opts map[string]any) (text string, tokens int, err error)
}

type Embedder interface {
  Embed(ctx context.Context, input []string, opts map[string]any) (vectors [][]float32, err error)
}

type VectorStore interface {
  Upsert(ctx context.Context, vectors []VectorItem) error
  Query(ctx context.Context, vector []float32, k int, filter map[string]any) ([]VectorItem, error)
}

type EventBus interface {
  Publish(ctx context.Context, topic string, data []byte, opts ...Option) error
  Subscribe(ctx context.Context, topic string, handler func(context.Context, []byte) error) (func() error, error)
}

type CheckpointStore interface {
  Save(ctx context.Context, runID string, checkpoint []byte) error
  Load(ctx context.Context, runID string) ([]byte, error)
}
```

### MCP Interfaces

```go
type ToolDescriptor struct {
  Name string
  InputSchema []byte
  OutputSchema []byte
}

type ResourceDescriptor struct {
  URI string
  Description string
}

type MCPClient interface {
  Handshake(ctx context.Context) error
  ListTools(ctx context.Context) ([]ToolDescriptor, error)
  CallTool(ctx context.Context, name string, args map[string]any) (map[string]any, error)
  ListResources(ctx context.Context) ([]ResourceDescriptor, error)
  ReadResource(ctx context.Context, uri string) ([]byte, error)
}

type MCPServer interface {
  RegisterTool(t Tool) error
  RegisterResource(r Resource) error
  Serve(ctx context.Context, conn net.Conn) error // JSON-RPC 2.0 over WS/HTTP
}
```

### Protocols
- HTTP/JSON: public control plane (create runs, pause/resume, fetch state), tool servers.
- gRPC: high-performance internal APIs (runtime <-> workers <-> adapters).
- WebSocket: interactive sessions and human-in-the-loop channels.
- Webhooks: triggers from external systems; idempotent with signatures (HMAC) and replay protection.
- MCP: Model Context Protocol over WebSocket and HTTP with JSON-RPC 2.0 semantics for tools/resources/prompts.
- OpenTelemetry OTLP/HTTP or gRPC for traces/metrics/logs.

### Scaling & Performance
- Horizontal scale: stateless workers; state in Postgres/Redis; queues for work distribution.
- Concurrency controls: per-run, per-agent, per-tool; rate limiters and circuit breakers.
- Backpressure: bounded queues; shed load on saturation; priority scheduling.
- Caching: request/result cache with TTL and key strategies; warm-up via pre-fetch hooks.
- Sharding: runID-based partitioning for event and state processing.

### Reliability
- Exactly-once effects via idempotency keys and durable checkpoints.
- Retries with exponential backoff and jitter; dead-letter queues for poison messages.
- Health checks: liveness/readiness; graceful shutdown with draining.

### Security
- Principle of least privilege for tools and plugins; scoped tokens/credentials.
- Secrets never logged; redaction middleware; encrypted at rest and in transit.
- Signed webhooks and request authentication (OAuth2/JWT/API keys) with RBAC.

### Developer Experience (DX)
- CLI: `orch` for init, run, resume, inspect, trace, eval.
- Scaffolding: `orch init agent` generates reducer/effects/tests and sample configs.
- Hot-reload dev server with in-memory stores for rapid iteration.
- Example gallery (in `examples/`) mirroring common patterns (RAG, tool-use, approval flows).
- First-class docs and code snippets; minimal working examples in each package.

### Learning from Existing Tools

- LangChain / LangGraph ([repo](https://github.com/langchain-ai/langgraph), [docs](https://github.com/langchain-ai/langgraph/tree/main/docs))
  - **Graph-based orchestration**: explicit DAG/graph of nodes with state channels; deterministic step function and resumability.
  - **Checkpointers & interrupts**: first-class pause/resume and human-in-the-loop interruption/continuations; persistent graph state.
  - **Typed tools & schema validation**: tool calling with structured inputs/outputs; safer execution path.
  - **Streaming and concurrency**: streaming tokens/events through the graph; controlled concurrency and backpressure.
  - What we adopt: reducer-centric purity with explicit state transitions, typed tools, checkpoint interfaces, resumable steps.

- Microsoft AutoGen ([repo](https://github.com/microsoft/autogen))
  - **Multi-agent conversation patterns**: reusable templates (e.g., Assistant <-> User Proxy, GroupChat) for collaboration and role specialization.
  - **Tool/Code execution loop**: built-in code execution and tool invocation patterns to close the loop on tasks.
  - **Human feedback**: easy insertion of human messages/approvals into agent conversations.
  - **Extensibility**: compose custom agents, termination conditions, and routing strategies.
  - What we adopt: message-passing patterns, termination conditions, human approvals, and permissive tool routing.

- n8n ([repo](https://github.com/n8n-io/n8n), [docs](https://github.com/n8n-io/n8n-docs))
  - **Node-based low-code UX**: clear node abstraction with triggers/actions; composability encourages ecosystem growth.
  - **Marketplace & versioned nodes**: contribution pipeline for third-party nodes, semantic versioning, and upgrade paths.
  - **Credentials & secrets**: centralized credential management and scoped permissions per node.
  - **Executions, retries, and backoff**: operational visibility, retry strategies, and failure handling out of the box.
  - What we adopt: plugin registry + capability descriptors, credential scoping, retries/backoff, and execution logs/inspectability.

- LlamaIndex ([repo](https://github.com/run-llama/llama_index))
  - **Data framework and indices**: composable indices, retrievers, and query engines; modular storage.
  - **Graph/RAG patterns**: graph-based retrieval (GraphRAG), routing, and structured context assembly.
  - **Observability & eval**: tracing hooks and evaluation utilities for RAG quality.
  - What we adopt: deterministic context assembly with pinning/dedup/token budgeting and observability hooks.

- Microsoft Semantic Kernel ([repo](https://github.com/microsoft/semantic-kernel))
  - **Plugins/functions model**: strongly-typed functions/plugins with parameter schemas and planners.
  - **Orchestration**: planners, connectors, and multi-agent collaboration patterns across .NET/Python.
  - **Enterprise alignment**: RBAC, connectors, and production-friendly patterns.
  - What we adopt: typed plugin interfaces, planners as optional schedulers, and enterprise-grade connectors and RBAC patterns.

- CloudWeGo Eino (ByteDance) ([repo](https://github.com/cloudwego/eino), [docs](https://www.cloudwego.io/docs/eino/))
  - **Componentized architecture**: clear component definitions (LLM, tools, memory, retrieval) enabling provider swap and composition.
  - **Flow orchestration**: first-class orchestration primitives to compose complex AI pipelines cleanly.
  - **ReAct agent implementation**: standardized reasoning/acting loop with flexible tool/memory integration ([React Agent Manual](https://www.cloudwego.io/docs/eino/core_modules/flow_integration_components/react_agent_manual/)).
  - **Production-hardened**: evolved from internal ByteDance usage with attention to maintainability and scale ([open-source announcement](https://cloudwego.cn/docs/eino/overview/eino_open_source/)).
  - What we adopt: strong component contracts, orchestration separated from components, ReAct template, scalable enterprise practices.

References (GitHub):
- LangGraph: https://github.com/langchain-ai/langgraph
- AutoGen: https://github.com/microsoft/autogen
- n8n: https://github.com/n8n-io/n8n, https://github.com/n8n-io/n8n-docs
- LlamaIndex: https://github.com/run-llama/llama_index
- Semantic Kernel: https://github.com/microsoft/semantic-kernel
- Eino: https://github.com/cloudwego/eino, https://www.cloudwego.io/docs/eino/

### Versioning & Compatibility
- Semantic versioning for public Go APIs in `pkg/`.
- Adapters declare compatibility matrix (framework version -> adapter version).
- Migration guides for breaking changes; deprecation period >= one minor release.

### Testing Strategy
- Unit tests for reducers, effect handlers, adapters.
- Integration tests: runtime with Postgres/NATS using testcontainers.
- Deterministic replay tests from captured runs (golden files).
- Load tests for scheduler and tool dispatch.

### Release & CI/CD
- CI: `go vet`, `golangci-lint`, unit/integration tests, race detector.
- Artifacts: multi-arch binaries, Docker images (distroless), SBOM and provenance (SLSA level 2+).
- Versioned docs, changelog, release notes with upgrade steps.

### Making It Popular Among Programmers
- Minimal onboarding: one-command example runs; scaffolded agents in <5 minutes.
- Stable, small core interfaces; batteries-included adapters for popular providers.
- Excellent docs and diagrams; copy-paste snippets; cookbook of patterns.
- Marketplace for community adapters and tools; clear contribution guide.
- Benchmarks and transparency: publish p95 latency and throughput on reference flows.

### Roadmap (initial)
1. MVP runtime (reducers/effects, checkpoints, HTTP API, Postgres store, OTel).
2. LLM adapters (OpenAI, Ollama) and vector stores (pgvector, Qdrant).
3. Event bus (NATS) and worker pool; CLI and scaffolding.
4. gRPC and WebSocket transports; human-in-the-loop Slack adapter.
5. Plugin SDK and marketplace; docs and examples expansion.



### Verification Tooling (applies to all milestones)

The detailed verification plans live in `ROADMAP.md` under each milestone. This section captures the common tooling and gates reused across milestones.

General principles
- Hermetic by default: integration tests use Testcontainers (Postgres, NATS, OTel Collector, Mock Slack).
- Deterministic first: record-and-replay harness captures events, tool I/O, and model outputs; seeded randomness.
- Structured evidence: CI uploads traces, logs, coverage, latency summaries, and golden snapshots as artifacts.
- Gated merges: PRs must pass milestone-specific targets before merging.

Common harness
- Unit tests for reducers, effects, registries, adapters (golden state diffs).
- Integration tests: end-to-end via HTTP/CLI with Postgres and NATS.
- Replay tests: `pkg/eval/replay` asserts identical states/outputs.
- Offline evals: prompt fixtures + scoring with thresholds.
- Observability checks: ephemeral OTel Collector validating spans/metrics/logs.
- Benchmarks: `go test -bench` hot paths; `benchstat` comparisons.

CI gates (examples)
- Lint/vet/staticcheck; race detector enabled for core packages.
- Replay parity on designated scenarios; latency gates per milestone.
- Security gates: secret scanning, dependency audit; webhook signature verification tests.

