# Changelog

All [Semantic Versions](https://semver.org/spec/v2.0.0.html) of this project and their notable changes are documented in the file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), with
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed

- non-vi mode is no longer supported - vi FTW!
- `/` is just a slash. Use `:` to enter command mode

### Fixed
- Context validation error when interrupting tool execution (issue #37)
- Tests now properly use isolated keyring service to avoid clearing production OAuth tokens
- Command timeout is now returned as command output (with exit code 124) instead of harness error
- Harness errors (connection failures) now trigger automatic container restart and command retry
- Enter now submits prompts directly from vi normal mode when the prompt is non-empty (#32)

### Added
- Support for `ANTHROPIC_OAUTH_TOKEN` environment variable to bypass keyring authentication
  - Accepts raw access token format
  - Accepts full JSON format with refresh token and expiry
  - Accepts base64-encoded JSON (useful when copying from macOS Keychain)
- Configuration option `run_in_shell.timeout_minutes` to set shell command timeout (default: 10 minutes)
- :! <cmd> - running in the container, to verify `:!uname -a`
- :resume to resume session
- :init - analyzes the project and creates a `.agents/asimi.conf`, `
- Each branch has its own prompt & command history
- `ui.markdown_enabled` configuration toggle to re-enable Glamour-based markdown rendering (defaults to off for faster resizing) (#53)
- Ctrl-B SCROLL mode for the chat viewport with vi-style paging and `:1` to jump to the first message without re-pinning

### Changed



## [0.1.0] - 2025/11/1

A development snapshort made for a friend. Not production ready at all.
