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
