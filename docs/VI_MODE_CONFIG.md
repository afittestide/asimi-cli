# Vi Mode Configuration

Asimi CLI supports vi-style modal editing for power users. This feature is **enabled by default** but can be disabled via configuration.

## Disabling Vi Mode

### Option 1: Configuration File

Add the following to your configuration file (`~/.config/asimi/conf.toml` or `.asimi/conf.toml`):

```toml
[llm]
vi_mode = false
```

### Option 2: Environment Variable

Set the environment variable before running Asimi:

```bash
export ASIMI_LLM_VI_MODE=false
asimi chat
```

Or inline:

```bash
ASIMI_LLM_VI_MODE=false asimi chat
```

## Default Behavior

When `vi_mode` is not explicitly set in the configuration:
- **Default**: Vi mode is **enabled**
- Users get the full vi modal editing experience
- Border colors indicate the current mode (green for insert, yellow for normal)

## Vi Mode Features

When vi mode is enabled:

### Insert Mode (Green Border)
- Type normally
- Press `Esc` to enter normal mode

### Normal Mode (Yellow Border)
- **Navigation**: `h/j/k/l`, `w/b`, `0/$`, `gg/G`
- **Editing**: `x`, `dw`, `D`, `p`
- **Enter Insert**: `i`, `a`, `I`, `A`, `o`, `O`
- **Commands**: Use `:` instead of `/` (e.g., `:help`, `:new`)

### Mode Indicators (Status Bar)
- Status bar shows `-- INSERT --` (green) when insert mode is active
- Status bar shows `-- NORMAL --` (yellow) when normal mode is active

## When to Disable Vi Mode

Consider disabling vi mode if:
- You're not familiar with vi/vim keybindings
- You prefer standard text editing behavior
- You want to use `/` for commands without modal switching
- You're teaching Asimi to beginners

## Re-enabling Vi Mode

To re-enable vi mode:

1. Remove or comment out the `vi_mode = false` line from your config
2. Or set `vi_mode = true` explicitly
3. Or use the `/vi` command during a session to toggle it on

## Runtime Toggle

You can also toggle vi mode during a session using the `/vi` command:

```
/vi
```

This will enable/disable vi mode for the current session only. The configuration file setting will be used for future sessions.
