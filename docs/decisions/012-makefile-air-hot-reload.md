# ADR 012: Makefile + Air for Go Development Workflow

## Status

Accepted

## Context

During development, the backend Go server needs to be restarted after every code change. The current workflow (`go run cmd/api/main.go`) requires manual restart, which creates friction when iterating on features.

The frontend Angular dev workflow (`ng serve`) automatically detects changes and hot-reloads, providing a significantly better developer experience. The backend should match this experience to maintain consistency across the stack.

## Decision

Adopt a two-tool approach for backend development:

1. **Makefile** for command shortcuts, making the backend CLI match the simplicity of the frontend
   - `make run` — run the API once (non-interactive)
   - `make dev` — run with hot-reload (interactive development)
   - `make test` / `make test-integration` — test shortcuts
   - `make lint` / `make vet` / `make fmt` — code quality tools

2. **air** ([air-verse/air](https://github.com/air-verse/air)) for live-reload
   - Watches `.go` files in `internal/` and `cmd/` directories
   - Automatically rebuilds and restarts on file changes
   - Configured via `.air.toml` to ignore migrations and vendor directories
   - Requires one-time installation: `go install github.com/air-verse/air@latest`

## Rationale

### Developer Experience

- Frontend: `ng serve` (hot-reload by default)
- Backend: `make dev` (hot-reload, same conceptual simplicity)
- Reduces context switching and friction when moving between frontend/backend work
- Faster iteration cycles — no manual server restart needed

### Command Simplicity

- Standard Go projects use Makefiles for convenience
- Single entry point: `make help` to discover all commands
- Aligns with Unix conventions — make is ubiquitous on macOS and Linux
- No additional language/tool to learn beyond make

### Tooling

- **air** is a lightweight, Go-specific tool (not a general-purpose task runner)
- Configuration is simple TOML — easy to maintain and extend
- Minimal overhead — watches files efficiently
- No impact on production — used for development only

## Consequences

**Positive:**
- Backend DX now matches frontend (hot-reload, discoverable commands)
- Makefile is version-controlled and self-documenting
- `air` adds <100ms to iteration cycles compared to manual restart
- Easy to onboard new contributors — `make help` explains everything

**Negative:**
- Requires `go install` of air on first setup (one-time cost)
- Windows developers must use WSL or Git Bash to run make (Windows has no native make)
- Adds `.air.toml` configuration to maintain (but complexity is very low)
- Two dependency sources for development: Go modules + system tools

**Risks:**
- air file watching may be unreliable on some filesystems (rare)
- macOS/Linux-only without workarounds for Windows

## Alternatives Considered

1. **Task (taskfile.dev)** — YAML-based make alternative
   - Slightly more readable than Makefile syntax
   - Cross-platform (no WSL needed on Windows)
   - Rejected: Adds another binary to install; make is more standard

2. **go-task / Go-based runners** — Custom Go tool
   - Could be compiled into the project
   - Rejected: Overengineering for simple command grouping

3. **Shell scripts** — Simple `run.sh`, `dev.sh` wrappers
   - No discovery mechanism (`make help`)
   - Less standard
   - Rejected: make is the Go community standard

## Implementation

Backend root now contains:
- `Makefile` — Command definitions
- `.air.toml` — air configuration (watches internal/, cmd/, excludes migrations/)

Updated documentation:
- `CLAUDE.md` — Commands section updated with `make dev` and installation steps
- `README.md` — Run section updated with new command recommendations

## Related ADRs

- ADR 002 (Go + Gin) — Backend architecture
