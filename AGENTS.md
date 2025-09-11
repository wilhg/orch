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
