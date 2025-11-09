# 🪾 Asimi CLI

> A safe, opinionated coding agent

Asimi is an opinionated command-line interface that brings AI-powered coding assistance directly to your terminal. Built with modern Go libraries and a focus on developer experience, Asimi helps you write better code faster.

> TLDR; `just run`, `:login`, `:init`

## ✨ Features

  **📦 **Integrated Podman Sandboxes** - Agent's shell run in its own container
- **🎨 vi mimicry ** - Asimi is based on the fittest dev tool and its reincarnations vim an neovim
- **🤖 Multiple AI Providers** - Support for ollama, Claude, Gemini and soon, more
- **🔧 Fast Shell** - Asimi's persistent, containerized shell is running more than 100 times faster 
- **📊 Context Awareness** - Smart token counting and context visualization
- **🎯 Session Management** - Save and resume your coding sessions

We're still missing MCP support. If it's critical for you, please consider helping out #


## 🚀 Quick Start

Please choose your installer flavor:

### Brew

```
brew install asimi
```

### Download Binaries

TODO: add link to latest release page

### One line installer

TODO: add a one line installer . The script will be part installed at https://asimi.dev/installer

### Go


```bash
go install github.com/asimi/asimi-cli
./asimi
```


### First Steps

1. **Add the infrastructure to your project**
   `:init` To add:

   
2. **Login to your AI provider:**
   `:login`

<<<<<<< Updated upstream
2. **Get help:**
   `:help` - Comprehensive help system with vim-like navigation
   `:help quickref` - Quick reference guide
   `?` (in NORMAL mode) - Quick help overlay

3. **Try some commands:**
   - `:help` - Show help system
   - `:context` - View token usage and context
   - `:new` - Start a new conversation
   - `:resume` - resume an old session
=======
3. **Chat**

## ⌨️ vi

Asimi aims to looks and feels like a reincarnation of vi for the AI Age.
This means the `/` is out. The commands are still there, but now you use `:`
to move to a special command line to enter your command.
>>>>>>> Stashed changes



### Modes

- **Insert Mode** (Green border; status bar shows `-- INSERT --`): For entering long text
- **Normal Mode** (Yellow border; status bar shows `-- NORMAL --`): Navigation and editing only
- **Command Mode** (Yellow border; status bar shows `-- NORMAL --`): Navigation and editing only

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

# measure shell's  performance
just measure
```

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

### Environment Variables

- **`EDITOR`** - Preferred text editor for export commands
- **`ASIMI_LAZYGIT_CMD`** - Custom lazygit command path
- **`ANTHROPIC_OAUTH_TOKEN`** - OAuth token for Anthropic API (takes priority over keyring). Supports three formats:
  - Raw access token: `sk-ant-...`
  - JSON format: `{"access_token":"...", "refresh_token":"...", "expiry":"...", "provider":"anthropic"}`
  - Base64-encoded JSON (useful if copying from keychain)
- **`ANTHROPIC_API_KEY`** - API key for Anthropic (alternative to OAuth)
- **`ANTHROPIC_BASE_URL`** - Custom base URL for Anthropic API (e.g., for proxy or custom endpoint)


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

See [CHANGELOG.md](CHANGELOG.md) for planned features and recent changes.

### Upcoming Features

- [ ] Init command
- [ ] Task delegation with sub-agents
- [x] Session resume with history

## 📄 License

[Add your license here]

## 🙏 Acknowledgments

- Built with ❤️ using Go
- Inspired by vi and modern coding agents
- Special thanks to the Bubble Tea and LangChainGo communities

---

**Made with 🪾 by the Asimi team**

*Safe, fun, and high-quality code generation*

------
