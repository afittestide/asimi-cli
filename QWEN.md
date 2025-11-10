# Asimi CLI - Agent Guide

**Language**: Go

## Quick Commands

```bash
# Development
just run              # Run the application
just build            # Build binary
just test             # Run all tests
just test-coverage    # Run tests with coverage
just lint             # Run golangci-lint
just fmt              # Format code
just modules          # Vendor dependencies

# Infrastructure
just bootstrap        # Install dev tools (golangci-lint, podman)
just infrabuild       # Build dev container
just infraclean       # Clean container resources

# Profiling
just profile          # Profile startup performance
just measure          # Measure run_in_shell performance
```

## Code Style

**Write idiomatic Go** - Keep it simple, flat, and direct.

- **Naming**: Short and meaningful
- **No wrappers**: Avoid unnecessary abstractions
- **No build tags**: Keep builds simple
- **Inline comments**: Only for non-trivial code
- **Flat structure**: Avoid creating new directories/files

## Libraries

- `slog` - Logging (use `--debug` flag)
- `bubbletea` - Terminal UI
- `koanf` - Configuration
- `kong` - CLI parsing
- `langchaingo` - LLM communications
- `go-git` - Git operations (NEVER shell out to git)
- `podman/docker` - Container management in `podman_runner.go` (NEVER shell out)

## Testing

- Run tests: `go test ./...`
- Coverage: `just test-coverage`
- Search code: `rg <pattern>`

## Release Management

- Follow **SemVer**
- Update **CHANGELOG.md** with: Fixed, Changed, Added
- Use git tags to sync code and changelog
- Commit messages: Use present progressive ("adding feature", not "added feature")

## Workflow

Check the `docs` when getting started and make sure to update any relevant docs.
Once a change is finished and ALL tests pass update the CHANGELOG.md
**DON'T MERGE** - ask user for approval


## Logs & Config

- Logs: ./asimi.log
- Config: `~/.config/asimi/conf.toml` or `.asimi/conf.toml`
