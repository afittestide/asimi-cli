# Startup Performance & UX Improvement Plan

## Overview

This document outlines a two-stage refactoring to improve asimi-cli's startup experience:
1. **Stage 1**: Refactor to uber/fx dependency injection for clean architecture
2. **Stage 2**: Add startup progress messages for early user feedback

## Current Problems

- Users see a blank screen during initialization for over 7 seconds
- Sequential initialization blocks startup unnecessarily
- Global state makes testing difficult
- No visibility into what's happening during startup

## Stage 1: Refactor to uber/fx (Clean Architecture)

### Phase 1.1: Add fx and Create Core Providers

**Goal**: Set up fx framework and migrate core services

**Tasks**:
- Add `go.uber.org/fx` dependency to `go.mod`
- Create new file: `providers.go`
- Implement core provider functions:
  - `ProvideLogger() *slog.Logger` - logger initialization
  - `ProvideConfig(*slog.Logger) *Config` - config loading
  - `ProvideGitInfo(*Config) *GitInfo` - git repository info
  - `ProvideShellRunner(*Config) ShellRunner` - podman shell runner

**Reference**: Current implementations in `main.go:53-83` (logger), `config.go:207-278` (config), `utils.go:25-91` (git info)

### Phase 1.2: Model and Session Providers

**Goal**: Migrate LLM and session management to providers

**Tasks**:
- Implement provider functions:
  - `ProvideModelClient(*Config, *slog.Logger) llms.Model` - async model initialization
  - `ProvideSession(llms.Model, *Config) *Session` - session with system prompt
  - `ProvideCommandHistory(*Config) *CommandHistory` - `CommandHistory` is currently named `HistoryStore`
  - `ProvideSessionHistory(*Config, *GitInfo) *SessionHistory` - `SessionHistory` is currently known as `SessionStore`
- Handle async LLM initialization properly with fx lifecycle

**Reference**: Current implementations in `main.go:577-727` (LLM), `session.go:125-207` (session), `tui.go:98-121` (stores)

### Phase 1.3: UI Providers

**Goal**: Migrate TUI components to providers

**Tasks**:
- Implement provider functions:
  - `ProvideMarkdownRenderer(*Config) *glamour.TermRenderer` - async renderer
  - `ProvideTUIModel(*Config, *HistoryStore, *SessionStore) *TUIModel` - UI model
  - `ProvideBubbletea(*TUIModel) *tea.Program` - bubbletea program
- Ensure proper dependency ordering

**Reference**: Current implementations in `main.go:210-213` (renderer), `tui.go:86-162` (TUI model)

### Phase 1.4: Integrate fx.App

**Goal**: Replace sequential initialization with fx dependency injection

**Tasks**:
- Modify `main.go:90-258` (runCmd.Run()) to use fx.New()
- Define fx application with all providers
- Set up lifecycle hooks:
  - `OnStart`: trigger async operations (LLM connection, markdown renderer)
  - `OnStop`: cleanup resources (close connections, flush logs)
- Remove global variables:
  - `program` (line 259)
  - `shellRunner` (defined in tools.go)
- Pass dependencies through function parameters instead
- Keep Kong CLI parsing outside fx (still in main())

**Example structure**:
```go
app := fx.New(
    fx.Provide(
        ProvideLogger,
        ProvideConfig,
        ProvideGitInfo,
        ProvideShellRunner,
        ProvideLLMClient,
        ProvideSession,
        ProvideHistoryStore,
        ProvideSessionStore,
        ProvideMarkdownRenderer,
        ProvideTUIModel,
        ProvideBubbletea,
    ),
    fx.Invoke(RunTUI),
)
```

### Phase 1.5: Testing & Validation

**Goal**: Ensure no regressions and improve testability

**Tasks**:
- Update existing tests to use `fxtest` utilities
- Add unit tests for individual providers
- Test isolated components with mock dependencies
- Run full test suite: `just test`
- Run with coverage: `just test-coverage`
- Manual testing with different configs
- Verify startup time doesn't increase
- Test with `--debug` flag for logging

**Validation criteria**:
- All existing tests pass
- No change in startup time
- Clean dependency graph (no cycles)
- No global state remaining

## Stage 2: Early UI Feedback (Startup Progress Messages)

### Phase 2.1: Create TUI-Aware Writer

**Goal**: Create an io.Writer that safely writes to stderr around TUI lifecycle

**Tasks**:
- Create new file: `tui_writer.go`
- Implement `TUIAwareWriter`:
  ```go
  type TUIAwareWriter struct {
      mu          sync.Mutex
      tuiActive   bool
      queuedLogs  []string
  }

  func (w *TUIAwareWriter) Write(p []byte) (n int, err error) {
      w.mu.Lock()
      defer w.mu.Unlock()

      if w.tuiActive {
          // Queue the message while TUI is running
          w.queuedLogs = append(w.queuedLogs, string(p))
          return len(p), nil
      }

      // Write directly to stderr before/after TUI
      return os.Stderr.Write(p)
  }

  func (w *TUIAwareWriter) SetTUIActive(active bool) {
      w.mu.Lock()
      defer w.mu.Unlock()
      w.tuiActive = active
  }

  func (w *TUIAwareWriter) Flush() {
      w.mu.Lock()
      defer w.mu.Unlock()

      for _, msg := range w.queuedLogs {
          os.Stderr.WriteString(msg)
      }
      w.queuedLogs = nil
  }
  ```
- Modify `initLogger()` to use dual handlers with TUIAwareWriter
- No config changes needed
- Verbosity controlled by existing `--debug` flag

### Phase 2.2: Configure slog with Dual Handlers

**Goal**: Configure slog to write to both file and TUIAwareWriter

**Tasks**:
- Modify `initLogger()` in `main.go`:
  ```go
  func initLogger(debug bool) (*TUIAwareWriter, error) {
      // Existing file handler setup
      logFile := &lumberjack.Logger{...}
      logLevel := slog.LevelInfo
      if debug {
          logLevel = slog.LevelDebug
      }
      fileHandler := slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: logLevel})

      // New: TUI-aware stderr handler
      tuiWriter := &TUIAwareWriter{}
      stderrHandler := slog.NewTextHandler(tuiWriter, &slog.HandlerOptions{Level: slog.LevelInfo})

      // Combine both handlers
      multiHandler := newMultiHandler(fileHandler, stderrHandler)
      logger := slog.New(multiHandler)
      slog.SetDefault(logger)

      return tuiWriter, nil
  }
  ```
- Implement simple `multiHandler` that writes to both handlers
- Return `tuiWriter` so it can be controlled during TUI lifecycle
- Keep existing log format and behavior

**Design rationale**:
- Uses existing slog infrastructure
- No custom abstractions - just an io.Writer
- File logging unchanged
- Stderr output controlled by TUI lifecycle

### Phase 2.3: Add slog.Info() Calls to Providers

**Goal**: Add progress logging using slog.Info() throughout initialization

**Tasks**:
- Add `slog.Info()` calls at key initialization points in each provider:

  **Info-level messages** (appear on stderr during startup):
  - `slog.Info("loading configuration")`
  - `slog.Info("configuration loaded")`
  - `slog.Info("detecting git repository")`
  - `slog.Info("git repository detected")` or `slog.Info("no git repository found")`
  - `slog.Info("initializing shell runner")`
  - `slog.Info("loading command history")`
  - `slog.Info("loading session history")`
  - `slog.Info("connecting to LLM", "provider", providerName)`
  - `slog.Info("LLM client connected")`
  - `slog.Info("creating session")`
  - `slog.Info("initializing markdown renderer")`
  - `slog.Info("building TUI")`
  - `slog.Info("starting TUI")`

  **Debug-level messages** (only in log file when `--debug`):
  - Keep existing `slog.Debug()` calls for detailed timing
  - OAuth token validation/refresh details
  - Container connection details
  - Any other verbose debugging info

- Use structured logging fields where relevant (e.g., "provider", "duration")
- Handle errors with `slog.Error()` or `slog.Warn()`
- Messages automatically go to:
  - File: always (at configured level)
  - Stderr: only during startup (before TUI) and after exit (queued messages)

### Phase 2.4: Manage TUIAwareWriter Lifecycle

**Goal**: Control when logs go to stderr vs get queued

**Tasks**:
- In `main.go` (or provider that runs TUI):
  ```go
  tuiWriter, _ := initLogger(cli.Debug)

  // During initialization: logs go to stderr
  // (slog.Info() calls from providers appear on user's terminal)

  // Before starting TUI
  slog.Info("starting TUI")
  tuiWriter.SetTUIActive(true)  // Start queueing logs

  // Run TUI
  finalModel, err := program.Run()

  // After TUI exits
  tuiWriter.SetTUIActive(false)
  tuiWriter.Flush()  // Write queued messages to stderr

  // Any final cleanup logs go to stderr again
  ```

- Pass `tuiWriter` through fx providers or keep as package-level var
- Ensure SetTUIActive(true) is called just before `program.Run()`
- Ensure SetTUIActive(false) and Flush() are called after TUI exits
- Use defer for cleanup if appropriate

**Implementation considerations**:
- Startup messages appear on stderr before TUI
- While TUI runs, logs are queued (not displayed)
- After TUI exits, queued logs are flushed to stderr
- User sees: startup progress → TUI → queued logs → exit
- No interference with bubbletea alt screen

## Expected Outcomes

### Stage 1 Benefits

**Clean Architecture**:
- Clear dependency graph (no circular dependencies)
- Explicit dependencies via function parameters
- Easy to understand initialization order

**Better Testability**:
- Components can be tested in isolation
- Mock dependencies easily injected
- Use `fxtest` for integration tests
- No global state to manage

**Foundation for Parallelization**:
- fx can parallelize independent providers
- Async operations handled cleanly
- Lifecycle hooks for startup/shutdown

**Maintainability**:
- Easy to add new services
- Clear where to add initialization code
- Dependencies documented in code

### Stage 2 Benefits

**Immediate User Feedback**:
- No blank screen during startup
- User sees progress immediately
- Reduces perceived latency

**Transparency**:
- User knows what's happening
- Can identify slow operations
- Helpful for debugging startup issues
- Simple, readable progress messages

**Simplicity**:
- Uses existing slog infrastructure
- Single io.Writer abstraction (TUIAwareWriter)
- No custom logging framework
- All logging goes through slog

**Configurable**:
- Users can redirect stderr to disable: `asimi 2>/dev/null`
- Debug flag shows additional detail in log file
- Standard Unix approach

## Files to Modify

### Stage 1

**New files**:
- `providers.go` - all fx provider functions

**Modified files**:
- `go.mod` - add `go.uber.org/fx`
- `main.go` - replace sequential init with fx.App
- `tools.go` - remove global shellRunner, pass as parameter
- `*_test.go` - update tests to use fxtest

### Stage 2

**New files**:
- `tui_writer.go` - TUIAwareWriter implementation

**Modified files**:
- `main.go` - modify initLogger() to use dual handlers, manage TUIAwareWriter lifecycle
- `providers.go` - add slog.Info() calls for startup progress

## Testing Strategy

### Unit Tests
- Test each provider in isolation
- Mock dependencies using interfaces
- Test error handling in providers
- Test TUIAwareWriter behavior:
  - Direct writes when TUI inactive
  - Queueing when TUI active
  - Flush after TUI deactivated

### Integration Tests
- Use `fxtest.New()` for integration testing
- Test full application startup
- Test with different configurations
- Test async operations (LLM, renderer)

### Manual Testing
- Verify startup messages appear on stderr before TUI
- Test with `--debug` flag for detailed file logging
- Verify queued messages flush after TUI exit
- Test with/without git repository
- Test with different LLM providers
- Test with slow network (simulate latency)
- Test error scenarios (invalid config, no internet, etc.)
- Test on different terminals (verify stderr/stdout separation)
- Verify no log interference with TUI rendering

### Performance Testing
- Measure startup time before/after
- Ensure no regression
- Identify opportunities for parallelization
- Profile with `just profile`

## Migration Risks & Mitigation

### Risks

1. **Breaking existing behavior**: fx changes initialization order
   - *Mitigation*: Comprehensive testing, careful dependency ordering

2. **Increased complexity**: fx adds abstraction layer
   - *Mitigation*: Good documentation, simple provider pattern

3. **Async initialization issues**: LLM client ready before TUI
   - *Mitigation*: Use fx lifecycle hooks, test thoroughly

4. **Log message interference with TUI**: stderr writes corrupting display
   - *Mitigation*: TUIAwareWriter queues messages during TUI active period, comprehensive testing
