# 🪾 Asimi CLI

> A safe, opinionated coding agent

Asimi is an opinionated command-line interface that brings AI-powered coding assistance directly to your terminal. Built with modern Go libraries and a focus on developer experience, Asimi helps you write better code faster.

> TLDR; `just run`, `:login`, `:init`

## ✨ Features

- **🎨 Beautiful TUI** - Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for a smooth, interactive experience
- **📝 Markdown Support** - Rich markdown rendering with syntax highlighting using [Glamour](https://github.com/charmbracelet/glamour)
- **🤖 Multiple AI Providers** - Support for ollama, Claude, OpenAI, Gemini, and Qwen models
- **🔧 Powerful Tools** - Integrated file operations, shell commands, and context management
- **⌨️ Vi Mode** - Using vi-style line editing by default
- **📊 Context Awareness** - Smart token counting and context visualization
- **🎯 Session Management** - Save and resume your coding sessions
## 🚀 Quick Start

### Installation

```bash
go install github.com/asimi/asimi-cli
./asimi
```

### First Steps

1. **Login to your AI provider:**
   `:login`

2. **Initialize your repo:**
    `:init` - Creates a AGENTS.md and Justfile if missing and `.agents/Sandbox` for the container 


## ⌨️ Vi Mode

Asimi comes with proper vi editing.

### Modes

- **Insert Mode** Type normally
- **Normal Mode** Navigation and editing only
- **Command Mode** Entering agents commands

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

# measure harness performance
just measure
```

### Project Structure

Flat. Please refrain from adding directories and files.

## 📦 Libraries

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** - Terminal UI framework
- **[Koanf](https://github.com/knadh/koanf)** - Configuration management
- **[Kong](https://github.com/alecthomas/kong)** - CLI argument parser
- **[LangChainGo](https://github.com/tmc/langchaingo)** - LLM communications and tools
- **[Glamour](https://github.com/charmbracelet/glamour)** - Markdown rendering

## 🔒 Security

Asimi takes security seriously:

- API keys are stored securely in your system keyring
- No data is sent to third parties except your chosen AI provider
- All file operations require explicit confirmation
- Shell commands are executed with proper sandboxing

## 🤝 Contributing

We welcome contributions! Here are some ways you can help:

1. **Report bugs** - Open an issue with details
2. **Suggest features** - Share your ideas
3. **Submit PRs** - Fix bugs or add features
4. **Improve docs** - Help others understand Asimi

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

Asimi stores its configuration in `~/.config/asimi/conf.toml` (user-level) or `.asimi/conf.toml` (project-level):

```toml
[llm]
provider = "anthropic"
model = "claude-sonnet-4-20250514"
vi_mode = true  # Enable vi-style keybindings (default: true)
max_output_tokens = 4096
[run_in_shell]
# Commands regex to run on the host instead of the container
run_on_host = [
    '^gh ',  # GitHub CLI commands
]
```

### Configuration Options

- **`vi_mode`** - Enable/disable vi-style keybindings (default: `true`)
  - Set to `false` to use standard editing mode
  - Can also be set via environment variable: `ASIMI_LLM_VI_MODE=false`
- **`provider`** - AI provider (anthropic, openai, googleai, qwen)
- **`model`** - Model name (provider-specific)
- **`max_output_tokens`** - Maximum tokens in AI responses
- **`max_turns`** - Maximum conversation turns before stopping

See `conf.toml.example` for a complete list of configuration options.

### Environment Variables

- **`EDITOR`** - Preferred text editor for export commands (e.g., `nvim`, `emacs`, `code`)
- **`ASIMI_LAZYGIT_CMD`** - Custom lazygit command path
- **`ANTHROPIC_OAUTH_TOKEN`** - OAuth token for Anthropic API (takes priority over keyring). Supports three formats:
  - Raw access token: `sk-ant-...`
  - JSON format: `{"access_token":"...", "refresh_token":"...", "expiry":"...", "provider":"anthropic"}`
  - Base64-encoded JSON (useful if copying from keychain)
- **`ANTHROPIC_API_KEY`** - API key for Anthropic (alternative to OAuth)
- **`ANTHROPIC_BASE_URL`** - Custom base URL for Anthropic API (e.g., for proxy or custom endpoint)


Logs are rotated and stored in `~/.local/share/asimi/`

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
- Inspired by modern CLI tools and AI assistants
- Special thanks to the Bubble Tea and LangChain communities

---

**Made with 🪾 by the Asimi team**

*Safe, fun, and high-quality code generation*
