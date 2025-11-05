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

2. **Try some commands:**
   - `:help` - Show available commands
   - `:context` - View token usage and context
   - `:new` - Start a new conversation
   - `:resume` - resume an old session


## ⌨️ Vi Mode

Asimi comes with default proper vi editing.

### Modes

- **Insert Mode** (Green border; status bar shows `-- INSERT --`): Type normally
- **Normal Mode** (Yellow border; status bar shows `-- NORMAL --`): Navigation and editing only

### Quick Reference

**Entering Insert Mode:**
- `i` - Insert at cursor
- `I` - Insert at line start
- `a` - Append after cursor
- `A` - Append at line end
- `o` - Open line below
- `O` - Open line above

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

**Commands:**
Use `:` instead of `/` in vi mode (e.g., `:help`, `:new`, `:quit`)

**Exit Vi Mode:**
Press `Esc` to go from insert to normal mode. Run `/vi` or `:vi` to disable vi mode entirely.

## 🛠️ Development

### Prerequisites

- Go 1.21 or higher
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

## 🎨 Theme

Asimi uses a custom color scheme inspired by Terminal7:

- **Prompt Border**: `#F952F9` (Magenta)
- **Chat Border**: `#F4DB53` (Yellow)
- **Text Color**: `#01FAFA` (Cyan)
- **Warning**: `#F4DB53` (Yellow)
- **Error**: `#F54545` (Red)

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

### Authentication Environment Variables

Asimi supports authentication via environment variables to bypass the keyring:

- **`ANTHROPIC_API_KEY`** - Set your Anthropic API key directly
- **`ANTHROPIC_OAUTH_TOKEN`** - Set your Anthropic OAuth token directly (bypasses keyring)
- **`OPENAI_API_KEY`** - Set your OpenAI API key directly

These environment variables are useful for CI/CD environments or when keyring access is not available.

Logs are stored in `~/.local/share/asimi/log/`

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
```bash
# Re-login to refresh credentials
asimi login
```

**Q: Context overflow errors**
```bash
# Check your context usage
/context

# Start a new conversation
/new
```

## 📊 Roadmap

See [CHANGELOG.md](CHANGELOG.md) for planned features and recent changes.

### Upcoming Features

- [ ] Init command
- [ ] Task delegation with sub-agents
- [x] Session resume with history

## 📄 License

[Add your license here]

## 🙏 Acknowledgments

- Built with ❤️ using Go
- Inspired by modern CLI tools and AI assistants
- Special thanks to the Bubble Tea and LangChain communities

---

**Made with 🪾 by the Asimi team**

*Safe, fun, and high-quality code generation*
