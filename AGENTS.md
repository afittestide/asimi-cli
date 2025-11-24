# Asimi CLI - Agent Guide

**Language**: Go 1.25.3 | **Build**: `just build` | **Test**: `just test` | **Lint**: `just lint`

## Essential Commands

```bash
just run              # Run with debug logging
just build            # Build binary
just test             # Run all tests
just test-coverage    # Tests with coverage report
just lint             # Run golangci-lint
just fmt              # Format code
just bootstrap        # Install dev tools (golangci-lint, podman)
just sandbox-build    # Build agent sandbox container
just measure          # Measure run_in_shell performance
```

## Code Conventions

**Write idiomatic Go** - Simple, flat, direct.

- **Naming**: Short, meaningful names (Go style)
- **Structure**: Flat - avoid new directories/files
- **Comments**: Only for non-trivial logic (why, not what)
- **No abstractions**: Avoid unnecessary wrappers
- **No build tags**: Keep builds simple

## Key Libraries

- `slog` - Logging (use `--debug` flag)
- `bubbletea` - Terminal UI framework
- `koanf` - Configuration management
- `kong` - CLI argument parsing
- `langchaingo` - LLM communications
- `go-git` - Git operations (NEVER shell out to git)
- `podman` - Container management in `podman_runner.go` (NEVER shell out)

## Testing

- Run: `go test ./...` or `just test`
- Coverage: `just test-coverage` (generates `coverage.html`)
- Search: `rg <pattern>` (ripgrep installed)

## Commit Style

- Use **present progressive**: "adding feature" not "added feature"
- Follow **SemVer** for releases
- Update **CHANGELOG.md** with: Fixed, Changed, Added
- **DON'T MERGE** - ask user for approval

## Project Files

- Logs: `./asimi.log` (with `--debug`)
- Config: `~/.config/asimi/conf.toml` or `.asimi/conf.toml`
- Storage: SQLite schema at `storage/schema.go`
- Docs: Check `docs/` when starting, update when finished
