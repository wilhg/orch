Read agents/12-factor-agent-framework-requirements.md
Read agents/ARCHITECTURE-AND-STANDARDS.md
Read agents/ROADMAP.md

## Go 1.25 Guidance

- **Target version**: Use Go 1.25.x (see `go.mod`).
- **OS support**: macOS 12+ required. See the official release notes.
- **Containers**: Go 1.25 introduces container-aware `GOMAXPROCS` defaults; prefer relying on runtime defaults unless you have measured reasons to override.
- **Spec cleanup**: The language spec removes the “core types” notion; no code changes needed for typical projects.

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
- Coverage gates are enforced in CI; raise thresholds over time.
- Unit tests SHOULD avoid external deps; prefer in-memory SQLite for store logic.
- Integration tests MUST use `testcontainers-go` and are guarded by a build tag:
  - Run: `go test -tags=integration ./...`.
  - Examples: Postgres container for store; networked tools.
  - Keep tests hermetic and self-cleaning.

## Git commits

- After formatting and tests pass, stage changes from the repository root:

```bash
git add .
git commit -m "<type>: <summary>"
```
