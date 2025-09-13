# orch

[![CI](https://github.com/wilhg/orch/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/wilhg/orch/actions/workflows/ci.yml)

## Quickstart (5 minutes)

Prereqs: Go 1.25+, Docker (optional for Postgres), macOS 12+ or Linux.

1) Clone and build

```bash
git clone https://github.com/wilhg/orch.git
cd orch
go build ./cmd/orch
```

2) Run with in-memory SQLite and stdout tracing

```bash
export DATABASE_URL="sqlite:file:orch.sqlite?cache=shared&_pragma=busy_timeout(5000)&_fk=1"
export ORCH_OTEL_STDOUT=1
./orch -addr :8080
```

3) Create a run

```bash
curl -sX POST http://localhost:8080/api/runs | jq
# => { "run_id": "<uuid>" }
RUN_ID=$(curl -sX POST http://localhost:8080/api/runs | jq -r .run_id)
```

4) Drive the example todo agent

```bash
# Add a task (pure reducer, no side-effects)
curl -sX POST http://localhost:8080/api/examples/todo \
  -H 'content-type: application/json' \
  -d '{"RunID":"'"$RUN_ID"'","Type":"add_task","Payload":{"title":"demo"}}' | jq

# Complete the task (emits an effect and a logged event)
curl -sX POST http://localhost:8080/api/examples/todo \
  -H 'content-type: application/json' \
  -d '{"RunID":"'"$RUN_ID"'","Type":"complete_task","Payload":{"title":"demo"}}' | jq

# Inspect events
curl -sS "http://localhost:8080/api/events?run=$RUN_ID" | jq '.[].type'
```

5) Pause/Resume a run

```bash
curl -sX POST http://localhost:8080/api/runs/pause -H 'content-type: application/json' -d '{"run_id":"'"$RUN_ID"'"}'
curl -sX POST http://localhost:8080/api/runs/resume -H 'content-type: application/json' -d '{"run_id":"'"$RUN_ID"'"}'
```

Notes:
- Set `ORCH_OTEL_STDOUT=1` to print spans; integrate OTLP later.
- For PostgreSQL, set `DATABASE_URL` to a Postgres DSN and rerun.

## Database

Configure the database with `DATABASE_URL`. Both PostgreSQL and SQLite are supported.

- PostgreSQL example:

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/orch?sslmode=disable"
```

- SQLite example (file-based):

```bash
export DATABASE_URL="sqlite:///./orch.sqlite?cache=shared&_pragma=busy_timeout(5000)"
```

The ent-backed store provides schema migration via code. Call `entstore.Open(ctx, os.Getenv("DATABASE_URL"))` and `store.Migrate(ctx)` during initialization.

## Example Agent

- Source: `examples/todo/agent.go`
- HTTP demo endpoint: `POST /api/examples/todo` with body:

```json
{
  "RunID": "<run-id>",
  "Type": "add_task",
  "Payload": {"title": "demo"}
}
```

Types: `add_task`, `complete_task`. Completing a task emits a `logged` event via an effect handler.

## Observability

- Traces: enabled via `pkg/otel`. Set `ORCH_OTEL_STDOUT=1` to pretty-print spans to stdout.
- HTTP spans: enabled using `otelhttp` wrapper in `cmd/orch`.

## Models (LLMs & Embeddings)

We ship first-class adapters for OpenAI and Gemini. Each provider includes an LLM (chat) and Embeddings client with sensible defaults.

- OpenAI
  - Default chat model: `gpt-5-nano`
  - Default embedding model: `text-embedding-3-small`
  - Auth: set `OPENAI_API_KEY`

- Gemini (go-genai)
  - Default chat model: `gemini-2.5-flash-lite`
  - Default embedding model: `gemini-embedding-001`
  - Auth: set `GOOGLE_API_KEY`
  - Backend: set `GOOGLE_GENAI_USE_VERTEXAI=false` to use the Gemini API backend (default when using our adapter factory via API key)

### Local env (.env)

Use a local `.env` for convenience (see `.env.example`). Load it with `set -a && source .env && set +a`.

```bash
# .env example
OPENAI_API_KEY=sk-...
GOOGLE_API_KEY=ai-...
GOEXPERIMENT=jsonv2,greenteagc
```

### Integration tests

Integration tests are tagged and skipped unless the respective API key is present.

```bash
# OpenAI
set -a && source .env && set +a
go test -tags=integration ./... -run TestOpenAI -v

# Gemini
set -a && source .env && set +a
go test -tags=integration ./... -run TestGemini -v
```

Notes
- We enable Go 1.25 experiments locally for parity with CI: `GOEXPERIMENT=jsonv2,greenteagc`.
- Some container-based tests (e.g., ChromaDB, Postgres) require Docker.

### CI secrets

GitHub Actions will run unit tests for all PRs. Integration jobs run when secrets are configured:

- Set `OPENAI_API_KEY` (repository secret) to enable the OpenAI integration job
- Set `GOOGLE_API_KEY` (repository secret) to enable the Gemini integration job

Both jobs respect `GOEXPERIMENT=jsonv2,greenteagc` and execute `go test -tags=integration`.

