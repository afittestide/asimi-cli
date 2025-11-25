# Asimi CLI

[![Tests](https://img.shields.io/github/actions/workflow/status/afittestide/asimi-cli/ci.yml?branch=dev&label=tests)](https://github.com/afittestide/asimi-cli/actions/workflows/ci.yml?query=branch%3Adev)

> A safe, opinionated coding agent

Asimi is an opinionated command-line interface that brings AI-powered coding assistance directly to your terminal. Built with modern Go libraries and a focus on developer experience, Asimi helps you write better code faster.

## ✨ Features

  📦 **Integrated Podman Sandboxes** - Agent's shell run in its own container
- 🎨 **vi mimicry** - Asimi is based on the fittest dev tool and its reincarnations vim an neovim
- 🤖 **Multiple AI Providers** - Support for Ollama and Claude foe now. More are comming [TODO: Add a link to issues with a label of "new providers"]
-  **Less Clutter** - Asimi's special files are under `.agents` directory and TOML is used for .conf
- 🔧 Fast Shell** - Asimi's persistent, containerized shell is >100 times faster than the others
- 📊 Context Awareness** - Smart token counting and context visualization
- 🎯 Session Management** - Save and resume your coding sessions

We're still missing MCP support. If it's critical for you, please consider helping out #


## 🚀 Quick Start

Please choose your installer flavor:

### Go

```bash
go install github.com/affitestide/asimi-cli
```

### Binaries

TODO: Release 0.2.0

Download the binary from your platform from [latest releases](https://github.com/afittestide/asimi-cli/releases/latest) and copy to your favorite bin directory e.g, `/usr/local/bin`.

### One liner

TODO: add a one line installer probably at https://asimi.dev/installer

### First Steps

After To start Asimi in interactive mode type `asimi`. 
1. **Add the infrastructure to your project**
   `:init` To add:

   
2. **Login to your AI provider:**
   `:login`


2. **Initialize your repo:**
    `:init` - Creates a AGENTS.md and Justfile if missing and `.agents/Sandbox` for the container 

3. **Check your container:***
    `:!uname -a` - runs shell commands is a persistent, containrized bash
    `:!pwd` - should be the same path as on your host

## ⌨️ vi FTW


Asimi mimics the vi/vim/neovim interface:

### Modes

- **Insert Mode** Type normally
- **Normal Mode** Navigation and editing only
- **Command Mode** Entering agents commands
   - `:help` - Show help 
   - `:context` - View token usage and context
   - `:new` - Start a new conversation
   - `:resume` - resume an old session

### Quick Reference

**Entering Insert Mode:**
- `i` - Insert at cursor
- `I` - Insert at line start
- `a` - Append after cursor
- `A` - Append at line end
- `o` - Open line below
- `O` - Open line above

**Entering Command Mode:**
- ':' - In normal mode or as first character is Visual

**Navigation (Normal Mode):**
- `h/j/k/l` - Left/Down/Up/Right
- `w/b` - Word forward/backward
- `0/$` - Line start/end
- `gg/G` - Input start/end

**Editing (Normal Mode):**
- `x` - Delete character
- `dw` - Delete word
- `D` - Delete to line end
- `p` - Paste

## 🛠️ Development

### Prerequisites

- Go 1.25 or higher
- [Just](https://github.com/casey/just) command runner

### Common Tasks

We're using a `Justfile` to collect all our script.
If you need a new script please add a recipe in the Justfile.

```bash
# List recipes
just

# run Asimi
just run

# Run tests
just test

# measure shell's  performance
just measure
```

### Project Structure

Flat. Please refrain from adding directories and files.

## 📦 Libraries

- **[LangChainGo](https://github.com/tmc/langchaingo)** - LLM communications and tools
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** - Terminal UI framework
- **[Koanf](https://github.com/knadh/koanf)** - Configuration management
- **[Kong](https://github.com/alecthomas/kong)** - CLI argument parser
- **[Glamour](https://github.com/charmbracelet/glamour)** - Markdown rendering

## 🔒 Security

Asimi takes security seriously:

- API keys are stored securely in your system keyring
- No data is sent to third parties except your chosen AI provider
- Shell commands are executed in a containerized sandbox

## 🤝 Contributing

We welcome contributions! Here are some ways you can help:

1. **Report bugs** - Open an issue with details
2. **Suggest features** - Share your ideas
3. **Submit PRs** - Fix bugs or add features

### Commit Message Style

We use present progressive tense for commit messages:

```bash
# Good
git commit -m "feat: adding markdown support"
git commit -m "bug: fixing context overflow bug"

# Avoid
git commit -m "added markdown support"
git commit -m "fixed context overflow bug"
```

## 📝 Configuration

Asimi stores its configuration in `~/.config/asimi/asimi.toml` (user-level) or `.agents/asimi.toml` (project-level):

```toml
[llm]
provider = "anthropic"
model = "claude-sonnet-4-20250514"
vi_mode = true  # Enable vi-style keybindings (default: true)
max_output_tokens = 4096

[ui]
# Toggle Glamour-based markdown rendering (default: false). Set to true for full markdown.
markdown_enabled = false

[run_in_shell]
# Commands regex to run on the host instead of the container
run_on_host = ['^gh ']
```

### Configuration Options

- **`vi_mode`** - Enable/disable vi-style keybindings (default: `true`)
  - Set to `false` to use standard editing mode
  - Can also be set via environment variable: `ASIMI_LLM_VI_MODE=false`
- **`provider`** - AI provider (anthropic, openai, googleai, qwen)
- **`model`** - Model name (provider-specific)
- **`max_output_tokens`** - Maximum tokens in AI responses
- **`max_turns`** - Maximum conversation turns before stopping
- **`ui.markdown_enabled`** - Enable Glamour markdown rendering when `true` (default: `false`); when `false`, Asimi uses lightweight word wrapping for faster UI resizing. Override via `ASIMI_UI_MARKDOWN_ENABLED=true|false`.

### Environment Variables

#### General Configuration

All configuration options can be set via environment variables using the `ASIMI_` prefix. The variable name should match the config path with underscores instead of dots. For example:
- `ASIMI_LLM_PROVIDER=anthropic` sets `llm.provider`
- `ASIMI_LLM_MODEL=claude-sonnet-4-20250514` sets `llm.model`
- `ASIMI_UI_MARKDOWN_ENABLED=true` sets `ui.markdown_enabled`

#### System Variables

- **`EDITOR`** - Preferred text editor for `:export` commands (default: system default)
- **`SHELL`** - Shell to use in container sessions (default: `/bin/bash`)

#### API Keys & Authentication

- **`ANTHROPIC_API_KEY`** - API key for Anthropic Claude models (alternative to OAuth)
- **`ANTHROPIC_OAUTH_TOKEN`** - OAuth token for Anthropic API (takes priority over keyring). Supports three formats:
  - Raw access token: `sk-ant-...`
  - JSON format: `{"access_token":"...", "refresh_token":"...", "expiry":"...", "provider":"anthropic"}`
  - Base64-encoded JSON (useful if copying from keychain)
- **`ANTHROPIC_BASE_URL`** - Custom base URL for Anthropic API (e.g., for proxy or custom endpoint)
- **`OPENAI_API_KEY`** - API key for OpenAI GPT models
- **`GEMINI_API_KEY`** - API key for Google Gemini models

#### OAuth Configuration (Advanced)

For custom OAuth setups, you can override the default OAuth endpoints:

**Google/Gemini:**
- `GOOGLE_CLIENT_ID` - OAuth client ID
- `GOOGLE_CLIENT_SECRET` - OAuth client secret
- `GOOGLE_AUTH_URL` - Authorization URL (optional, default: `https://accounts.google.com/o/oauth2/v2/auth`)
- `GOOGLE_TOKEN_URL` - Token URL (optional, default: `https://oauth2.googleapis.com/token`)
- `GOOGLE_OAUTH_SCOPES` - Comma-separated list of scopes (optional, default: generative-language scope)

**OpenAI:**
- `OPENAI_CLIENT_ID` - OAuth client ID
- `OPENAI_CLIENT_SECRET` - OAuth client secret
- `OPENAI_AUTH_URL` - Authorization URL
- `OPENAI_TOKEN_URL` - Token URL
- `OPENAI_OAUTH_SCOPES` - Comma-separated list of scopes (optional)

**Anthropic:**
- `ANTHROPIC_CLIENT_ID` - OAuth client ID
- `ANTHROPIC_CLIENT_SECRET` - OAuth client secret
- `ANTHROPIC_AUTH_URL` - Authorization URL
- `ANTHROPIC_TOKEN_URL` - Token URL
- `ANTHROPIC_OAUTH_SCOPES` - Comma-separated list of scopes (optional)

#### Development & Testing

- **`ASIMI_KEYRING_SERVICE`** - Override keyring service name (default: `asimi-cli`)
- **`ASIMI_SKIP_GIT_STATUS`** - Skip git status checks (set to any value to enable)
- **`ASIMI_VERSION`** - Override version string for testing


Logs are rotated and stored in `~/.local/share/asimi/`. When running with `--debug`, logs are instead written to `asimi.log` in the project root for quick inspection.

## 🐛 Troubleshooting

### Common Issues

**Q: Asimi won't start**
```bash
# Check if the binary is executable
chmod +x asimi

# Try running with verbose logging
./asimi --debug
```

**Q: API key not working**
# Re-login to refresh credentials

use `:login`

**Q: Context overflow errors**
```bash
# Check your context usage
:context

# Start a new conversation
/new
```

## 📊 Roadmap

See issues for planned issues

### Upcoming Features

- [ ] MCP Support
- [ ] Task delegation with sub-agents
- [ ] 

## 📄 License

[Add your license here]

## 🙏 Acknowledgments

- Built with ❤️ using Go
- Inspired by vi and his great grandchildren - the coding agents
- Special thanks to the Bubble Tea and LangChainGo communities

---

**Made with 🪾 by the Asimi team**

*Safe, fun, and high-quality code generation*

------
