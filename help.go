package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HelpWindow is a simplified component for displaying help documentation
// Navigation is handled by ContentComponent
type HelpWindow struct {
	width  int
	height int
	topic  string
}

// NewHelpWindow creates a new help window
func NewHelpWindow() HelpWindow {
	return HelpWindow{
		width:  80,
		height: 20,
		topic:  "index",
	}
}

// SetSize updates the dimensions of the help window
func (h *HelpWindow) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// SetTopic sets the help topic to display
func (h *HelpWindow) SetTopic(topic string) {
	if topic == "" {
		topic = "index"
	}
	h.topic = topic
}

// GetTopic returns the current topic
func (h *HelpWindow) GetTopic() string {
	return h.topic
}

// RenderContent generates the styled help content for the current topic
func (h *HelpWindow) RenderContent() string {
	return h.renderHelpContent(h.topic)
}

// renderHelpContent generates the help content for a given topic
func (h *HelpWindow) renderHelpContent(topic string) string {
	// Style definitions
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F4DB53")).
		MarginTop(1).
		MarginBottom(1)

	subheaderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#01FAFA")).
		MarginTop(1)

	codeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F952F9")).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F952F9"))

	// Get help content based on topic
	content := h.getHelpTopic(topic)

	// Apply styling to the content
	lines := strings.Split(content, "\n")
	var styledLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Headers (lines starting with #)
		if strings.HasPrefix(trimmed, "# ") {
			styledLines = append(styledLines, headerStyle.Render(strings.TrimPrefix(trimmed, "# ")))
		} else if strings.HasPrefix(trimmed, "## ") {
			styledLines = append(styledLines, subheaderStyle.Render(strings.TrimPrefix(trimmed, "## ")))
		} else if strings.HasPrefix(trimmed, "```") {
			// Code blocks - skip the markers
			continue
		} else if strings.HasPrefix(trimmed, "  ") && strings.Contains(trimmed, "-") {
			// Key bindings (indented lines with dashes)
			parts := strings.SplitN(trimmed, "-", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				desc := strings.TrimSpace(parts[1])
				styledLines = append(styledLines, "  "+keyStyle.Render(key)+" - "+desc)
			} else {
				styledLines = append(styledLines, line)
			}
		} else if strings.HasPrefix(trimmed, ":") || strings.HasPrefix(trimmed, "/") {
			// Commands
			styledLines = append(styledLines, "  "+codeStyle.Render(trimmed))
		} else {
			styledLines = append(styledLines, line)
		}
	}

	return strings.Join(styledLines, "\n")
}

// getHelpTopic returns the help content for a specific topic
func (h *HelpWindow) getHelpTopic(topic string) string {
	topics := map[string]string{
		"index":      helpIndex,
		"":           helpIndex, // Default topic
		"modes":      helpModes,
		"commands":   helpCommands,
		"navigation": helpNavigation,
		"editing":    helpEditing,
		"files":      helpFiles,
		"sessions":   helpSessions,
		"context":    helpContext,
		"config":     helpConfig,
		"quickref":   helpQuickRef,
	}

	if content, ok := topics[strings.ToLower(topic)]; ok {
		return content
	}

	// Topic not found
	return fmt.Sprintf(`# Help Topic Not Found

The help topic '%s' was not found.

## Available Topics

Type :help followed by one of these topics:

  :help index       - Main help index
  :help modes       - Vi modes (INSERT, NORMAL, VISUAL)
  :help commands    - Available commands
  :help navigation  - Navigation keys
  :help editing     - Editing commands
  :help files       - File operations
  :help sessions    - Session management
  :help context     - Context and token usage
  :help config      - Configuration options
  :help quickref    - Quick reference guide

Press 'q' or ESC to close this help window.
`, topic)
}

// Help content definitions

const helpIndex = `# Asimi Help Index

Welcome to Asimi - A safe, opinionated coding agent with vim-like interface.

## Getting Started

Asimi uses vi-style editing by default. You start in INSERT mode where you can
type normally. Press ESC to enter NORMAL mode for navigation and commands.

## Quick Start

  1. Type your question or request in INSERT mode
  2. Press Enter to send
  3. Use @ to reference files (e.g., @main.go)
  4. Press : in NORMAL mode to enter commands
  5. Press ? in NORMAL mode for quick help

## Help Topics

  :help modes       - Vi modes (INSERT, NORMAL, VISUAL, COMMAND-LINE)
  :help commands    - Available commands (:help, :new, :quit, etc.)
  :help navigation  - Navigation keys (h, j, k, l, w, b, etc.)
  :help editing     - Editing commands (i, a, o, d, y, p, etc.)
  :help files       - File operations and @ references
  :help sessions    - Session management and resume
  :help context     - Context and token usage
  :help config      - Configuration options
  :help quickref    - Quick reference guide

## Navigation in Help

  j/k or ↓/↑       - Scroll line by line
  Ctrl+d/Ctrl+u    - Scroll half page
  Ctrl+f/Ctrl+b    - Scroll full page
  g/G              - Go to top/bottom
  q or ESC         - Close help

## Getting Help

For more information on a specific topic, use:
  :help <topic>

Example:
  :help modes
  :help commands
  :help navigation
`

const helpModes = `# Vi Modes

Asimi uses vi-style modal editing. Each mode has a different purpose and
different key bindings.

## INSERT Mode (Default)

This is the mode you start in. Type normally to compose your message.

  Status: -- INSERT --
  Border: Green (#00FF00)

  Enter INSERT mode from NORMAL mode:
    i    - Insert at cursor
    I    - Insert at beginning of line
    a    - Append after cursor
    A    - Append at end of line
    o    - Open new line below
    O    - Open new line above

  Exit INSERT mode:
    ESC  - Return to NORMAL mode

## NORMAL Mode

Navigation and command mode. Use this to move around and execute commands.

  Status: -- NORMAL --
  Border: Yellow (#F4DB53)

  Enter NORMAL mode:
    ESC  - From INSERT, VISUAL, or COMMAND-LINE mode

  From NORMAL mode you can:
    - Navigate with h, j, k, l
    - Enter commands with :
    - Enter INSERT mode with i, a, o, etc.
    - Enter VISUAL mode with v
    - Show quick help with ?

## VISUAL Mode

Select text for copying or manipulation.

  Status: -- VISUAL --
  Border: Cyan (#01FAFA)

  Enter VISUAL mode:
    v    - Character-wise visual mode
    V    - Line-wise visual mode

  In VISUAL mode:
    h/j/k/l  - Extend selection
    y        - Yank (copy) selection
    d        - Delete selection
    ESC      - Return to NORMAL mode

## COMMAND-LINE Mode

Execute commands at the bottom of the screen.

  Status: :
  Border: Magenta (#F952F9)

  Enter COMMAND-LINE mode:
    :    - From NORMAL mode

  In COMMAND-LINE mode:
    Type command and press Enter to execute
    ESC  - Cancel and return to NORMAL mode
    Tab  - Command completion (if available)

## LEARNING Mode

Special mode for adding notes to AGENTS.md.

  Status: -- LEARNING --
  Border: Purple

  Enter LEARNING mode:
    #    - From NORMAL mode

  In LEARNING mode:
    Type your note and press Enter to append to AGENTS.md
    ESC  - Cancel and return to NORMAL mode
`

const helpCommands = `# Commands

Commands are executed from COMMAND-LINE mode. Press : in NORMAL mode to enter
COMMAND-LINE mode, then type the command and press Enter.

## Session Management

  :new              - Start a new conversation
  :resume           - Resume a previous session
  :quit             - Quit Asimi (also saves session)

## Configuration

  :login            - Login with OAuth provider selection
  :models           - Select AI model

## Information

  :help [topic]     - Show help (optionally for a specific topic)
  :context          - Show context usage and token information

## History

  :clear-history    - Clear all prompt history

## Export

  :export [type]    - Export conversation to file and open in $EDITOR
                      Types: conversation (default), full

## Project Initialization

  :init             - Initialize project with infrastructure files
                      Creates: AGENTS.md, Justfile, .agents/Sandbox
  :init force       - Force regenerate all infrastructure files

## Examples

  :help modes       - Show help about vi modes
  :new              - Start fresh conversation
  :export           - Export and edit conversation
  :context          - Check token usage
`

const helpNavigation = `# Navigation

Navigation commands work in NORMAL and VISUAL modes.

## Basic Movement

  h        - Move left
  j        - Move down
  k        - Move up
  l        - Move right
  ←↓↑→     - Arrow keys also work

## Word Movement

  w        - Move forward to start of next word
  b        - Move backward to start of previous word
  e        - Move forward to end of word

## Line Movement

  0        - Move to beginning of line
  ^        - Move to first non-blank character
  $        - Move to end of line

## Document Movement

  gg       - Go to first line
  G        - Go to last line

## History Navigation

  ↑        - Previous prompt in history (when on first line)
  ↓        - Next prompt in history (when on last line)
  k        - Previous prompt (in NORMAL mode, when on first line)
  j        - Next prompt (in NORMAL mode, when on last line)

## Chat Scrolling

  Mouse wheel      - Scroll chat history
  Touch gestures   - Scroll on touch devices

## Help Navigation

When viewing help:
  j/k or ↓/↑       - Scroll line by line
  Ctrl+d           - Scroll down half page
  Ctrl+u           - Scroll up half page
  Ctrl+f           - Scroll down full page
  Ctrl+b           - Scroll up full page
  g                - Go to top
  G                - Go to bottom
`

const helpEditing = `# Editing Commands

Editing commands work in NORMAL mode.

## Entering INSERT Mode

  i        - Insert before cursor
  I        - Insert at beginning of line
  a        - Append after cursor
  A        - Append at end of line
  o        - Open new line below and insert
  O        - Open new line above and insert

## Deletion

  x        - Delete character under cursor
  X        - Delete character before cursor
  dw       - Delete word
  dd       - Delete line
  D        - Delete to end of line

## Copying and Pasting

  y        - Yank (copy) in VISUAL mode
  yy       - Yank current line
  p        - Paste after cursor
  P        - Paste before cursor

## Undo and Redo

  u        - Undo last change
  Ctrl+r   - Redo

## Special Features

  @        - Start file reference (triggers file completion)
  #        - Enter LEARNING mode (add note to AGENTS.md)

## File References

Type @ followed by a filename to reference a file in your prompt:

  @main.go          - Reference main.go
  @src/utils.go     - Reference file in subdirectory

A completion dialog will appear showing matching files. Use:
  ↓/↑ or Tab       - Navigate completions
  Enter            - Select file
  ESC              - Cancel

## Learning Mode

Press # in NORMAL mode to enter LEARNING mode. Type a note and press Enter
to append it to AGENTS.md. This is useful for teaching Asimi about your
project conventions and preferences.

Example:
  # We use snake_case for function names in this project
`

const helpFiles = `# File Operations

Asimi provides several ways to work with files in your project.

## File References with @

Use @ to reference files in your prompts. This loads the file content into
the conversation context.

  @filename         - Reference a file
  @path/to/file     - Reference file in subdirectory

When you type @, a completion dialog appears showing available files:
  ↓/↑              - Navigate through files
  Enter            - Select file
  ESC              - Cancel

Example:
  Can you review @main.go and suggest improvements?

## File Completion

The file completion dialog shows:
  - All files in your project
  - Filtered by your search query
  - Sorted by relevance

Type to filter:
  @mai             - Shows files matching "mai" (e.g., main.go)
  @src/            - Shows files in src/ directory

## Context Management

Files you reference are added to the conversation context. Use :context to
see what's currently in context:

  :context         - Show context usage and loaded files

## File Tools

Asimi has built-in tools for file operations:
  - read_file      - Read file contents
  - write_file     - Write or update files
  - list_files     - List directory contents

These tools are used automatically by the AI when needed.

## Best Practices

1. Reference only the files you need for the current task
2. Use :context to monitor token usage
3. Start a :new session if context gets too large
4. Use specific file paths to avoid ambiguity
`

const helpSessions = `# Session Management

Asimi can save and resume your coding sessions, preserving the entire
conversation history and context.

## Starting a New Session

  :new             - Start a fresh conversation
                     Clears chat history and context

## Resuming Sessions

  :resume          - Show list of recent sessions
                     Select one to resume

The session list shows:
  - First prompt from each session
  - Time since last update
  - Project/directory

Navigation in session list:
  ↓/↑              - Navigate sessions
  Enter            - Resume selected session
  ESC              - Cancel

## Auto-Save

Sessions are automatically saved when:
  - You send a message
  - You quit Asimi
  - You start a new session

## Session Configuration

Configure session behavior in ~/.config/asimi/conf.toml:

  [session]
  enabled = true           # Enable session persistence
  auto_save = true         # Auto-save after each message
  max_sessions = 50        # Maximum sessions to keep
  max_age_days = 30        # Delete sessions older than this
  list_limit = 20          # Number of sessions to show in :resume

## Session Storage

Sessions are stored in:
  ~/.local/share/asimi/sessions/

Each session includes:
  - Full conversation history
  - Context files
  - Model and provider information
  - Timestamps

## Exporting Sessions

  :export              - Export conversation to file
  :export conversation - Export just the conversation
  :export full         - Export with full context

The exported file opens in your $EDITOR for review or sharing.
`

const helpContext = `# Context and Token Usage

Asimi tracks token usage to help you stay within model limits and manage
conversation context effectively.

## Viewing Context

  :context         - Show detailed context information

The context view shows:
  - Current token usage
  - Maximum token limit
  - Percentage used
  - Number of messages
  - Loaded files and their sizes

## Token Limits

Different models have different context limits:
  - Claude Sonnet: 200K tokens
  - GPT-4: 128K tokens
  - Gemini Pro: 1M tokens

Asimi automatically tracks usage and warns when approaching limits.

## Managing Context

When context gets too large:

1. Start a new session:
   :new

2. Export current conversation:
   :export

3. Reference only essential files:
   Use @ selectively

4. Use shorter prompts:
   Be concise in your requests

## Context Files

Files referenced with @ are loaded into context:
  @main.go         - Adds main.go to context

View loaded files:
  :context         - Shows all files in context

## Token Counting

Asimi counts tokens for:
  - Your prompts
  - AI responses
  - System messages
  - File contents
  - Tool calls and results

## Best Practices

1. Monitor context usage regularly with :context
2. Start fresh sessions for new tasks
3. Reference files only when needed
4. Export important conversations before starting new sessions
5. Use :clear-history to reset prompt history (doesn't affect context)
`

const helpConfig = `# Configuration

Asimi can be configured through configuration files and environment variables.

## Configuration Files

Asimi looks for configuration in this order:
  1. .agents/asimi.toml        (project-level)
  2. ~/.config/asimi/conf.toml (user-level)

## Basic Configuration

[llm]
provider = "anthropic"           # AI provider
model = "claude-sonnet-4-20250514"  # Model name
vi_mode = true                   # Enable vi mode (default: true)
max_output_tokens = 4096         # Max tokens in responses
max_turns = 50                   # Max conversation turns

[session]
enabled = true                   # Enable session persistence
auto_save = true                 # Auto-save sessions
max_sessions = 50                # Max sessions to keep
max_age_days = 30                # Delete old sessions
list_limit = 20                  # Sessions shown in :resume

## Providers

Supported providers:
  - anthropic      (Claude models)
  - openai         (GPT models)
  - googleai       (Gemini models)
  - qwen           (Qwen models)
  - ollama         (Local models)

## Environment Variables

EDITOR                    - Text editor for :export
ASIMI_LAZYGIT_CMD        - Custom lazygit path
ANTHROPIC_OAUTH_TOKEN    - Anthropic OAuth token
ANTHROPIC_API_KEY        - Anthropic API key
ANTHROPIC_BASE_URL       - Custom Anthropic endpoint
ASIMI_LLM_VI_MODE        - Enable/disable vi mode

## Vi Mode

Vi mode is enabled by default. To disable:

In config file:
  [llm]
  vi_mode = false

Or with environment variable:
  ASIMI_LLM_VI_MODE=false asimi

## Logging

Logs are stored in:
  ~/.local/share/asimi/asimi.log

Logs are automatically rotated.

## Example Configuration

[llm]
provider = "anthropic"
model = "claude-sonnet-4-20250514"
vi_mode = true
max_output_tokens = 8192
max_turns = 100

[session]
enabled = true
auto_save = true
max_sessions = 100
max_age_days = 60
list_limit = 30

## Changing Configuration

After editing configuration:
  1. Restart Asimi for changes to take effect
  2. Use :login to reconfigure provider
  3. Use :models to change model
`

const helpQuickRef = `# Quick Reference

## Modes

  ESC      - NORMAL mode (from INSERT/VISUAL/COMMAND-LINE)
  i        - INSERT mode at cursor
  a        - INSERT mode after cursor
  o        - INSERT mode on new line below
  v        - VISUAL mode
  :        - COMMAND-LINE mode
  #        - LEARNING mode

## Navigation (NORMAL mode)

  h j k l  - Left, down, up, right
  w b      - Word forward/backward
  0 $      - Line start/end
  gg G     - Document start/end
  ↑ ↓      - History navigation

## Editing (NORMAL mode)

  x        - Delete character
  dw dd D  - Delete word/line/to-end
  y p      - Yank (copy) and paste
  u Ctrl+r - Undo/redo

## Commands (type : then command)

  :help [topic]    - Show help
  :new             - New session
  :resume          - Resume session
  :quit            - Quit
  :login           - Login to provider
  :models          - Select model
  :context         - Show context info
  :export          - Export conversation
  :init            - Initialize project

## Special Features

  @filename        - Reference file (triggers completion)
  #note            - Add note to AGENTS.md
  Ctrl+C (2x)      - Quit (press twice quickly)
  Ctrl+Z           - Background Asimi
  Ctrl+O           - Toggle raw session view
  ?                - Quick help (in NORMAL mode)

## File Completion

  @        - Start file reference
  ↓↑       - Navigate files
  Enter    - Select file
  ESC      - Cancel

## Help Navigation

  j k      - Scroll line by line
  Ctrl+d u - Half page down/up
  Ctrl+f b - Full page down/up
  g G      - Top/bottom
  q ESC    - Close help

## Tips

  - Start in INSERT mode, press ESC for NORMAL mode
  - Use : for commands, @ for files, # for learning
  - Check :context to monitor token usage
  - Use :export to save conversations
  - Press ? in NORMAL mode for quick help
`
