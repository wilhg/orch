Read all .md docs under ./agents/ folder

## Go 1.25 Guidance

- **Target version**: Use Go 1.25.x (see `go.mod`).
- **OS support**: macOS 12+ required. See the official release notes.
- **Containers**: Go 1.25 introduces container-aware `GOMAXPROCS` defaults; prefer relying on runtime defaults unless you have measured reasons to override.
- **Spec cleanup**: The language spec removes the “core types” notion; no code changes needed for typical projects.

### Always-on Go 1.25 features in this repo

- **GreenTea GC (experimental)**: Enabled by default via `GOEXPERIMENT=greenteagc` in CI and Docker images. Measure in your environment before overriding.
- **JSON v2 (experimental)**: Enabled by default via `GOEXPERIMENT=jsonv2` so the standard `encoding/json` uses the new faster implementation. The API remains `encoding/json`; only the implementation switches. Subject to change across Go versions.

Local dev tips

```bash
# enable same defaults locally (compiler experiments)
export GOEXPERIMENT=jsonv2,greenteagc

go test ./... -race -shuffle=on
```

Notes
- Both features are experimental; if you hit regressions, unset them locally:
  - `unset GOEXPERIMENT; export GODEBUG=`
- Container-aware `GOMAXPROCS` is automatic; avoid manual overrides unless measured.

### References
- **Go 1.25 Release Notes**: [go.dev/doc/go1.25](https://go.dev/doc/go1.25)
- **Go 1.25 Blog**: [go.dev/blog/go1.25](https://go.dev/blog/go1.25)
- **Container-aware GOMAXPROCS**: [go.dev/blog/container-aware-gomaxprocs](https://go.dev/blog/container-aware-gomaxprocs)
- **Core types spec note**: [go.dev/blog/coretypes](https://go.dev/blog/coretypes)

## Code formatting

- Always format code before committing.
- Run: `gofmt -s -w .` (or `go fmt ./...`) before `git commit`.
- Optional pre-commit hook (`.git/hooks/pre-commit`):

```bash
#!/usr/bin/env bash
set -euo pipefail
unformatted=$(gofmt -s -l .)
if [ -n "$unformatted" ]; then
  echo "These files need gofmt:" >&2
  echo "$unformatted" >&2
  exit 1
fi
```

## Testing & TDD

- Practice TDD: write unit tests first, then implement code.
- Default command: `go test ./... -race -shuffle=on`.
- Unit tests SHOULD avoid external deps; prefer in-memory SQLite for store logic.
- Integration tests MUST use `testcontainers-go` and are guarded by a build tag:
  - Run: `go test -tags=integration ./...`.
  - Examples: Postgres container for store; networked tools.
  - Keep tests hermetic and self-cleaning.

## You should ALWAYS make Git commits after tests pass

- Commit only after local tests pass. Then run lint before committing:

```bash
go test ./... -race -shuffle=on
golangci-lint fmt
golangci-lint run
```

And then stage changes from the repository root:
```bash
git add .
git commit -m "<type>: <summary>"
```

## Quickstart

See `README.md` for the 5-minute quickstart. To drive the example todo agent:

```bash
RUN_ID=$(curl -sX POST http://localhost:8080/api/runs | jq -r .run_id)
curl -sX POST http://localhost:8080/api/examples/todo -H 'content-type: application/json' \
  -d '{"RunID":"'"$RUN_ID"'","Type":"complete_task","Payload":{"title":"demo"}}'
```

### Model Context Protocol (MCP) Go SDK

- **SDK**: We use the official MCP Go SDK to implement client/server interoperability.
  - Repository: `https://github.com/modelcontextprotocol/go-sdk`
  - Core package: `mcp/` (client, server, tool/resource APIs, protocol types).
- **Integration in this repo**
  - Client wrapper: `pkg/mcpclient/`
    - Default build (no tags): no-op client so normal builds/tests don’t require MCP.
    - With `-tags mcp`: wraps the official SDK to provide `Dial/Handshake`, `ListTools`, `CallTool`, and `ListResources`.
  - Tool registry: `pkg/agent/registry.go` provides `RegisterTool`, `ResolveTool`, and `RangeTools` to export local tools to an MCP server.
- **Build/run notes**
  - Enable Go 1.25 experiments as usual: `export GOEXPERIMENT=jsonv2,greenteagc`.
  - Build with MCP client enabled:
    - `go test -tags mcp ./... -race -shuffle=on`
  - The real MCP client expects a ws:// or wss:// endpoint to `Dial` and then `Handshake`.
- **Planned server support (M2:T7)**
  - Expose registered tools via an `mcp.Server`, map our `agent.Tool` to MCP tool descriptors, and forward `CallTool` to `SafeInvoke`.
  - Add an integration test (tagged) that spins up a local MCP server and verifies client handshake/tool calls.

#### Additional notes

- **Build tags and defaults**
  - Default build: MCP disabled (no-op wrappers).
  - Enable real client/server with: `-tags mcp` (e.g., `go test -tags mcp ./...`).
  - Consider a separate tagged CI job for MCP.

- **Transports**
  - SDK supports JSON-RPC over multiple transports (WS, SSE, raw net.Conn).
  - We provide `Serve(ctx, addr)` (simple TCP accept loop) and `ServeConn` for pre-established connections. For production, prefer WS/WSS.

- **Tool schema types**
  - SDK tool descriptors use `jsonschema-go` types. If we move beyond `[]byte` schemas locally, add a converter when exporting tools.

- **Client/server API alignment**
  - Client flow: create → wire to connection (Serve) → `Handshake` → `ListTools`/`ListResources` → `CallTool`.
  - Map SDK responses to our local `ToolDescriptor`/resource structs.

- **Error semantics**
  - Convert MCP errors to our compact error model (`pkg/errmodel`) with category/code.

- **Security and auth**
  - Use SDK auth hooks for tokens/headers if the peer requires auth. Use TLS (WSS/HTTPS) at transport.

- **Observability**
  - Add OTel spans around handshake and tool calls. Consider an MCP→OTel bridge for richer instrumentation.

- **Testing patterns**
  - Use `net.Pipe()` for deterministic in-memory client/server tests; keep behind `-tags mcp`.

- **Versioning**
  - Pin a compatible `modelcontextprotocol/go-sdk` version; monitor upstream changes in `mcp/*` APIs.

- **References**
  - SDK repo: `https://github.com/modelcontextprotocol/go-sdk`
  - MCP package: `https://github.com/modelcontextprotocol/go-sdk/tree/main/mcp`

### Token Counting (tiktoken-go)

- We use `tiktoken-go` for accurate token counting in the context assembler by default.
  - Default encoding: `o200k_base` (optimized for GPT-4o / 4.1 / 4.5 families) per upstream guidance.
  - Library: `github.com/pkoukk/tiktoken-go` ([repo](https://github.com/pkoukk/tiktoken-go)).
- Usage in code
  - The assembler tries `o200k_base` via `GetEncoding` and falls back to rune-length estimation if unavailable.
  - You can override with a specific model/encoding:

```go
import "github.com/wilhg/orch/pkg/runtime/assembler"

// Model-based estimator
est, _ := assembler.NewTikTokenEstimator("gpt-4o")

// Encoding-based estimator (e.g., cl100k_base)
est2, _ := assembler.NewTikTokenEncodingEstimator("cl100k_base")

asm := assembler.New(assembler.WithTokenEstimator(est))
```

- Notes
  - See the repo’s README for mapping between models and encodings (e.g., `o200k_base`, `cl100k_base`).
  - If tokenization fails (e.g., missing encoding files), we transparently fall back to rune-length so development isn’t blocked. For production determinism, prefer explicit estimators.

- Environment overrides
  - **ORCH_TOKEN_ENCODING**: Set to a tiktoken encoding name (e.g., `o200k_base`, `cl100k_base`).
  - **ORCH_TOKEN_MODEL**: Alternative to specify a model name (e.g., `gpt-4o`, `gpt-4`).
  - Precedence: `ORCH_TOKEN_ENCODING` > `ORCH_TOKEN_MODEL` > default `o200k_base` > rune-length fallback.
