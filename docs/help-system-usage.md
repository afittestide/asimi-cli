# Help System Usage Guide

## Getting Started with Help

### First Time Users

When you first start Asimi, the help system is your best friend:

```bash
# Start Asimi
./asimi

# Open help
:help
```

This shows the main help index with links to all topics.

### Learning Path

We recommend learning in this order:

1. **Start with the index** (`:help`)
   - Get an overview of all available topics
   - Understand the help system navigation

2. **Learn the modes** (`:help modes`)
   - Understand INSERT, NORMAL, VISUAL modes
   - Learn how to switch between modes
   - Practice mode transitions

3. **Master navigation** (`:help navigation`)
   - Learn basic movement (h, j, k, l)
   - Practice word movement (w, b)
   - Try line movement (0, $)

4. **Explore commands** (`:help commands`)
   - See all available commands
   - Try a few basic ones (:new, :context)
   - Bookmark this for reference

5. **Keep quickref handy** (`:help quickref`)
   - One-page cheat sheet
   - Quick reminders when needed
   - Print or keep open in another terminal

## Common Use Cases

### "I forgot a command"

```bash
# Show all commands
:help commands

# Or use quick reference
:help quickref
```

### "How do I navigate in vi mode?"

```bash
# Detailed navigation help
:help navigation

# Quick reminder
? (press in NORMAL mode)
```

### "What are all these modes?"

```bash
# Complete mode documentation
:help modes
```

### "How do I reference files?"

```bash
# File operations guide
:help files
```

### "I need to configure something"

```bash
# Configuration documentation
:help config
```

### "How do I manage sessions?"

```bash
# Session management guide
:help sessions
```

### "What's using up my tokens?"

```bash
# Context and token usage
:help context
```

## Tips and Tricks

### 1. Quick Help in NORMAL Mode

Press `?` in NORMAL mode for instant help without leaving your work:

```
[In NORMAL mode]
? → Shows quick help overlay
ESC → Returns to work
```

### 2. Topic Shortcuts

Remember these common topics:

- `modes` - When confused about modes
- `commands` - When you forgot a command
- `quickref` - When you need a quick reminder
- `navigation` - When you're lost in the editor

### 3. Navigation Mastery

Learn these navigation keys for efficient help browsing:

```
j/k       - Line by line (most common)
Ctrl+d/u  - Half page (for scanning)
g/G       - Jump to top/bottom (for overview)
```

### 4. Keep Help Open

You can open help in one terminal and work in another:

```bash
# Terminal 1
./asimi
:help quickref

# Terminal 2
./asimi
[work normally]
```

### 5. Print the Quick Reference

The quickref is designed to be printable:

```bash
# Export quickref to file
:help quickref
# (manually copy or screenshot)
```

## Best Practices

### For New Users

1. **Start with `:help`** - Don't skip the index
2. **Read `:help modes`** - Understanding modes is crucial
3. **Practice navigation** - Try all the keys in `:help navigation`
4. **Use `?` often** - Quick help is your friend
5. **Refer to `:help quickref`** - Keep it handy

### For Experienced Users

1. **Use specific topics** - `:help <topic>` is faster
2. **Master navigation** - Learn Ctrl+d/u for speed
3. **Teach others** - Share help topics with teammates
4. **Contribute** - Suggest improvements to help content
5. **Customize** - Add your own notes to AGENTS.md

### For Teams

1. **Share help topics** - "Check `:help sessions`"
2. **Create team guides** - Reference help in documentation
3. **Onboard with help** - New members start with `:help`
4. **Report issues** - If help is unclear, report it
5. **Extend help** - Add team-specific topics

## Keyboard Shortcuts Summary

### Opening Help

| Command | Description |
|---------|-------------|
| `:help` | Main help index |
| `:help <topic>` | Specific topic |
| `?` (NORMAL) | Quick help overlay |

### Navigating Help

| Key | Action |
|-----|--------|
| `j` or `↓` | Scroll down one line |
| `k` or `↑` | Scroll up one line |
| `Ctrl+d` | Scroll down half page |
| `Ctrl+u` | Scroll up half page |
| `Ctrl+f` | Scroll down full page |
| `Ctrl+b` | Scroll up full page |
| `g` | Go to top |
| `G` | Go to bottom |
| `q` or `ESC` | Close help |

## Help Topics Reference

### Core Topics

- **index** - Main help index, start here
- **quickref** - One-page cheat sheet
- **modes** - Vi modes explained
- **commands** - All available commands

### Feature Topics

- **navigation** - Movement and navigation
- **editing** - Editing commands
- **files** - File operations
- **sessions** - Session management
- **context** - Token usage and context

### Configuration

- **config** - Configuration options

## Troubleshooting

### Help won't open

```bash
# Check if you're in the right mode
ESC  # Enter NORMAL mode
:help  # Try again
```

### Can't navigate in help

```bash
# Make sure help viewer has focus
# Try pressing j or k
# If nothing happens, close and reopen:
ESC
:help
```

### Help content is cut off

```bash
# Resize your terminal
# Or use Ctrl+d/u to scroll
```

### Wrong topic shown

```bash
# Close and reopen with correct topic
ESC
:help <correct-topic>
```

## Advanced Usage

### Combining with Other Features

```bash
# Check context before asking
:context
# Read about context management
:help context
# Ask your question with context in mind
```

### Learning Workflow

```bash
# 1. Read help
:help editing

# 2. Try commands in NORMAL mode
ESC
i  # Try insert mode
ESC
o  # Try open line

# 3. Refer back to help
:help editing
```

### Teaching Others

```bash
# Share specific help topics
"Check :help modes for mode explanations"
"See :help commands for all commands"
"Use :help quickref as a cheat sheet"
```

## Integration with Workflow

### Morning Routine

```bash
# Start Asimi
./asimi

# Quick review
:help quickref

# Start working
ESC
i
```

### When Stuck

```bash
# Check relevant help
:help <topic>

# Try the command
ESC
:<command>

# If still stuck, check context
:context
```

### Before Asking for Help

```bash
# 1. Check help first
:help <topic>

# 2. Try the solution
# 3. If still stuck, ask with context
```

## Customization

### Adding Your Own Notes

Use LEARNING mode to add notes:

```bash
# In NORMAL mode
#
# Type your note
This project uses snake_case for functions
# Press Enter
```

Notes are saved to AGENTS.md and help Asimi learn your preferences.

### Team Documentation

Create a team help file:

```bash
# Create team-help.md
# Reference Asimi help topics
# Add team-specific workflows
# Share with team
```

## Performance Tips

### Fast Help Access

```bash
# Memorize common topics
:help modes      # m for modes
:help commands   # c for commands
:help quickref   # q for quickref
```

### Efficient Navigation

```bash
# Use half-page scrolling
Ctrl+d  # Down
Ctrl+u  # Up

# Jump to sections
g   # Top
G   # Bottom
```

### Quick Reference

Keep `:help quickref` open in a separate terminal for instant reference.

## Conclusion

The help system is designed to make Asimi self-documenting. Use it liberally:

- **When learning**: Start with `:help`
- **When stuck**: Check `:help <topic>`
- **When teaching**: Share help topics
- **When working**: Use `?` for quick help

Remember: The help system is always available, always up-to-date, and always free. Use it!

## Quick Start Checklist

- [ ] Open help: `:help`
- [ ] Read modes: `:help modes`
- [ ] Try navigation: `:help navigation`
- [ ] Check commands: `:help commands`
- [ ] Save quickref: `:help quickref`
- [ ] Practice with `?` in NORMAL mode
- [ ] Try all navigation keys (j/k/g/G/Ctrl+d/u)
- [ ] Close help with `q` or `ESC`
- [ ] Open help for specific topics
- [ ] Share help with teammates

Happy learning! 🚀
