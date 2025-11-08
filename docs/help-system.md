# Help System Implementation

## Overview

Asimi now includes a comprehensive help system modeled after Vim's help viewer. The help system provides detailed documentation on all aspects of using Asimi, with vim-like navigation and organization.

## Features

### 1. **Full-Screen Help Viewer**
- Dedicated help viewer component that takes over the entire screen
- Vim-like navigation (j/k, Ctrl+d/u, Ctrl+f/b, g/G)
- Press 'q' or ESC to close

### 2. **Multiple Help Topics**
The help system is organized into topics:

- **index** - Main help index and getting started
- **modes** - Vi modes (INSERT, NORMAL, VISUAL, COMMAND-LINE, LEARNING)
- **commands** - Available commands (:help, :new, :quit, etc.)
- **navigation** - Navigation keys and movement
- **editing** - Editing commands and operations
- **files** - File operations and @ references
- **sessions** - Session management and resume
- **context** - Context and token usage
- **config** - Configuration options
- **quickref** - Quick reference guide

### 3. **Vim-like Navigation**

When viewing help:
- `j/k` or `↓/↑` - Scroll line by line
- `Ctrl+d` - Scroll down half page
- `Ctrl+u` - Scroll up half page
- `Ctrl+f` - Scroll down full page
- `Ctrl+b` - Scroll up full page
- `g` - Go to top
- `G` - Go to bottom
- `q` or `ESC` - Close help

### 4. **Styled Content**
- Headers in bold yellow
- Subheaders in bold cyan
- Key bindings highlighted in magenta
- Code blocks with background
- Consistent color scheme matching Asimi's theme

## Usage

### Basic Help
```
:help
```
Shows the main help index.

### Topic-Specific Help
```
:help modes
:help commands
:help navigation
:help editing
:help files
:help sessions
:help context
:help config
:help quickref
```

### Quick Help in Normal Mode
Press `?` in NORMAL mode to show a quick reference of key bindings.

## Implementation Details

### Components

1. **HelpViewer** (`help.go`)
   - Main help viewer component
   - Uses `bubbles/viewport` for scrolling
   - Handles keyboard navigation
   - Renders styled help content

2. **Help Content**
   - Embedded as constants in `help.go`
   - Organized by topic
   - Markdown-like formatting
   - Styled during rendering

3. **Integration** (`tui.go`)
   - Help viewer added to TUIModel
   - Handles showHelpMsg
   - Manages focus and visibility
   - Renders as full-screen overlay

4. **Command Handler** (`commands.go`)
   - `:help [topic]` command
   - Parses topic argument
   - Sends showHelpMsg

### Message Flow

```
User types: :help modes
    ↓
handleEnterKey() processes command
    ↓
handleHelpCommand() creates showHelpMsg{topic: "modes"}
    ↓
handleCustomMessages() receives showHelpMsg
    ↓
helpViewer.Show("modes") displays help
    ↓
User navigates with j/k/g/G/Ctrl+d/etc
    ↓
User presses q or ESC
    ↓
helpViewer.Hide() closes help
```

### Styling

The help viewer uses Asimi's color scheme:
- **Headers**: `#F4DB53` (Yellow) - Bold
- **Subheaders**: `#01FAFA` (Cyan) - Bold
- **Keys**: `#F952F9` (Magenta) - Bold
- **Code**: `#F952F9` on `#1a1a1a` background
- **Border**: `#F952F9` (Magenta)
- **Text**: Default terminal color

## Adding New Help Topics

To add a new help topic:

1. Add a constant in `help.go`:
```go
const helpNewTopic = `# New Topic Title

Content goes here...
`
```

2. Add to the topics map in `getHelpTopic()`:
```go
topics := map[string]string{
    // ... existing topics ...
    "newtopic": helpNewTopic,
}
```

3. Update the help index to reference the new topic.

## Testing

Run help tests:
```bash
go test -v -run TestHelp
```

Tests cover:
- Help viewer creation
- Show/hide functionality
- All help topics
- Unknown topic handling

## Future Enhancements

Possible improvements:
- [ ] Cross-references between help topics (like Vim's |topic| links)
- [ ] Search within help (/)
- [ ] Help history (navigate back/forward between topics)
- [ ] Context-sensitive help (F1 key shows help for current mode)
- [ ] Help tags for quick topic jumping
- [ ] Export help to markdown files
- [ ] Custom user help topics

## Comparison with Vim's Help

### Similarities
- Full-screen viewer
- Topic-based organization
- Vim-like navigation
- Quick reference available
- Comprehensive coverage

### Differences
- No hyperlinks (yet)
- Simpler tag system
- Fewer topics (focused on Asimi features)
- Styled with colors (Vim help is plain text)
- No help search (yet)

## Examples

### Getting Started
```
:help
```
Shows the main index with links to all topics.

### Learning Vi Modes
```
:help modes
```
Detailed explanation of INSERT, NORMAL, VISUAL, COMMAND-LINE, and LEARNING modes.

### Command Reference
```
:help commands
```
Complete list of all available commands with descriptions.

### Quick Reference
```
:help quickref
```
One-page cheat sheet of the most common operations.

### Configuration
```
:help config
```
Details on configuration files, environment variables, and options.

## Tips

1. **Start with the index**: `:help` shows all available topics
2. **Use quickref for reminders**: `:help quickref` is a great cheat sheet
3. **Learn one topic at a time**: Focus on modes, then navigation, then editing
4. **Press ? in NORMAL mode**: Quick help without leaving your work
5. **Help is always available**: No internet connection needed

## Keyboard Shortcuts Summary

| Key | Action |
|-----|--------|
| `:help` | Open help index |
| `:help <topic>` | Open specific topic |
| `?` (NORMAL mode) | Quick help |
| `j/k` or `↓/↑` | Scroll line by line |
| `Ctrl+d/u` | Half page scroll |
| `Ctrl+f/b` | Full page scroll |
| `g/G` | Top/bottom |
| `q` or `ESC` | Close help |
