# TUI Architecture Documentation

This document describes the architecture of the Terminal User Interface (TUI) for Claude Code, focusing on two major refactorings that established clean component boundaries and message-passing patterns.

## Table of Contents

1. [Overview](#overview)
2. [Component Architecture](#component-architecture)
3. [Mode Management System](#mode-management-system)
4. [Command Line Component](#command-line-component)
5. [Mouse Event Handling](#mouse-event-handling)
6. [Message Flow Patterns](#message-flow-patterns)
7. [Future Improvements](#future-improvements)

---

## Overview

### Screen Layout

```
┌─────────────────────────────────────────┐
│                                         │
│         CONTENT AREA                    │  ← Mouse wheel works here
│         (Chat/Help/Models/Resume)       │
│                                         │
│         Y = 0 to contentHeight          │
│                                         │
├─────────────────────────────────────────┤  ← contentHeight boundary
│         EMPTY LINE                      │
├─────────────────────────────────────────┤
│  ┌───────────────────────────────────┐  │
│  │    PROMPT AREA                    │  │  ← Mouse wheel ignored here
│  └───────────────────────────────────┘  │
├─────────────────────────────────────────┤
│  STATUS LINE                            │
├─────────────────────────────────────────┤
│  COMMAND LINE                           │
└─────────────────────────────────────────┘
```

The TUI follows the [Bubbletea](https://github.com/charmbracelet/bubbletea) architecture pattern:
- **Model**: `TUIModel` holds application state
- **Update**: Message handlers update state and return commands
- **View**: Renders the current state (no logic, just rendering)

### Core Principles

1. **Single Source of Truth**: State lives in one place
2. **Message Passing**: Components communicate via messages, not direct calls
3. **Separation of Concerns**: Components handle their own input and state
4. **No Polling**: View() renders state without checking component internals
5. **Centralized Coordination**: TUI coordinates, components execute

---

## Component Architecture

### TUIModel (tui.go)

The main model that coordinates all components:

```go
type TUIModel struct {
    Mode           string              // Current UI mode (single source of truth)
    prompt         *PromptComponent    // User input
    content        *ContentComponent   // Chat/Help/Models/Resume views
    commandLine    *CommandLineComponent // Command line input
    status         *StatusComponent    // Status bar
    completion     *CompletionComponent // Completion dialog
    // ... other fields
}
```

**Responsibilities:**
- Coordinate component interactions
- Handle high-level key bindings
- Route messages to appropriate handlers
- Maintain global state (mode, focus, etc.)

### Components

Each component is self-contained and follows these patterns:

1. **Handles its own input** via `HandleKey()` or `Update()`
2. **Returns messages** to communicate state changes
3. **Exposes minimal API** for coordination
4. **Doesn't know about other components**

---

## Mode Management System

### Problem

Previously, `TUIModel.View()` had to poll multiple components to determine what mode to display:

```go
// OLD - Scattered, coupled logic
if m.commandLine.IsInCommandMode() {
    m.status.SetViMode(viEnabled, "COMMAND", "")
} else if activeView != ViewChat {
    m.status.SetViMode(viEnabled, viewName, "")
} else {
    m.status.SetViMode(viEnabled, viMode, viPending)
}
```

This violated separation of concerns and created tight coupling.

### Solution: Centralized Mode Management

**Core Concept:**
- **Single Source of Truth**: `TUIModel.Mode string` field
- **Single Message**: `ChangeModeMsg{NewMode: "insert"}` for ALL mode changes
- **Centralized Handler**: Updates `m.Mode` and status component in one place
- **Components Send Messages**: When they change state
- **View() Just Renders**: No polling, no logic

### The Message

```go
type ChangeModeMsg struct {
    NewMode string // "insert", "normal", "visual", "command", "help", "models", "resume", "scroll"
}
```

### Centralized Handler

```go
case ChangeModeMsg:
    // Centralized mode management - update Mode and status
    m.Mode = msg.NewMode
    
    // Update status component
    viEnabled, _, viPending := m.prompt.ViModeStatus()
    
    // Map mode to display string
    var displayMode string
    switch msg.NewMode {
    case "insert":
        displayMode = ViModeInsert
    case "normal":
        displayMode = ViModeNormal
    case "visual":
        displayMode = ViModeVisual
    case "command":
        displayMode = "COMMAND"
    case "help":
        displayMode = "HELP"
    case "models":
        displayMode = "MODELS"
    case "resume":
        displayMode = "RESUME"
    case "scroll":
        displayMode = "SCROLL"
    default:
        displayMode = msg.NewMode
    }
    
    m.status.SetViMode(viEnabled, displayMode, viPending)
    return m, nil
```

- `scroll` is a dedicated chat-navigation mode entered with `Ctrl-B`. It locks the viewport in place (no auto-scroll) and provides vi-style paging:
  - `Ctrl-F` / `Ctrl-B` - Page down/up
  - `Ctrl-D` / `Ctrl-U` - Half page down/up
  - `j` / `k` / `↓` / `↑` - Half page down/up (vi-style)
  - `G` - Jump to bottom
  - `:` - Enter command mode without snapping back
  - `Esc` / `i` - Return to insert mode

### Component Integration

Components return `ChangeModeMsg` when their state changes:

```go
// CommandLineComponent
func (cl *CommandLineComponent) EnterCommandMode(initialText string) tea.Cmd {
    cl.mode = CommandLineCommand
    // ... setup ...
    return func() tea.Msg {
        return ChangeModeMsg{NewMode: "command"}
    }
}

// ContentComponent
func (c *ContentComponent) ShowHelp(topic string) tea.Cmd {
    c.activeView = ViewHelp
    // ... setup ...
    return func() tea.Msg {
        return ChangeModeMsg{NewMode: "help"}
    }
}

// TUI vi mode changes
case "i":
    m.prompt.EnterViInsertMode()
    return m, func() tea.Msg { return ChangeModeMsg{NewMode: "insert"} }
```

### Message Flow Example

```
User presses ':'
  ↓
TUI calls m.commandLine.EnterCommandMode("")
  ↓
Returns ChangeModeMsg{NewMode: "command"}
  ↓
TUI receives message in handleCustomMessages
  ↓
Sets m.Mode = "command"
  ↓
Maps to displayMode = "COMMAND"
  ↓
Calls m.status.SetViMode(viEnabled, "COMMAND", viPending)
  ↓
View() renders with updated status
```

### Supported Modes

- `"insert"` → `<INSERT>`
- `"normal"` → `<NORMAL>`
- `"visual"` → `<VISUAL>`
- `"command"` → `<COMMAND>`
- `"help"` → `<HELP>`
- `"models"` → `<MODELS>`
- `"resume"` → `<RESUME>`
- `"learning"` → (maps to itself)

### Benefits

✅ **Single Source of Truth** - `m.Mode` is the only place tracking mode  
✅ **Single Message** - `ChangeModeMsg` for ALL mode changes  
✅ **Centralized Logic** - One handler updates everything  
✅ **No Polling** - View() doesn't check component state  
✅ **Flexible** - Easy to add modes (just strings)  
✅ **Testable** - Verify mode changes via messages  
✅ **Decoupled** - Components don't know about status  
✅ **Consistent** - Same pattern everywhere  

---

## Command Line Component

### Problem

Previously, `TUIModel.handleCommandLineInput()` handled all keyboard input for the command line (266 lines), which violated separation of concerns.

### Solution: Component Self-Management

The `CommandLineComponent` now handles its own input and communicates via messages.

### Component Messages

```go
type (
    commandReadyMsg       struct{ command string }  // Command ready to execute
    commandCancelledMsg   struct{}                   // User pressed ESC
    commandTextChangedMsg struct{}                   // Text changed, update completions
    navigateCompletionMsg struct{ direction int }    // Navigate completion list
    navigateHistoryMsg    struct{ direction int }    // Navigate history
    acceptCompletionMsg   struct{}                   // Accept selected completion
)
```

### HandleKey Method

The component handles its own keyboard input:

```go
func (cl *CommandLineComponent) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool)
```

**Handles:**
- Basic editing (backspace, delete, cursor movement, space, typing)
- Enter (returns `commandReadyMsg`)
- ESC (returns `commandCancelledMsg` + `ChangeModeMsg`)
- Up/Down/Tab (returns navigation messages)

**Returns:**
- `tea.Cmd` - Command to execute (message for TUI)
- `bool` - Whether the key was handled

### TUI Integration

```go
if m.commandLine.IsInCommandMode() {
    cmd, handled := m.commandLine.HandleKey(msg)
    if handled {
        return m, cmd  // Process returned message
    }
    // Fallback (shouldn't happen)
}
```

### Message Handlers

**`commandReadyMsg`**:
- Saves to persistent history
- Parses and executes command using `FindCommand()` (vim-style partial matching)
- Hides completions
- Restores focus

**`commandCancelledMsg`**:
- Hides completions
- Restores focus to prompt

**`commandTextChangedMsg`**:
- Updates command line completions via `updateCommandLineCompletions()`

**`navigateCompletionMsg`**:
- Navigates completion dialog up/down

**`acceptCompletionMsg`**:
- Accepts selected completion

### Architecture Evolution

#### Before (Monolithic)
```
TUIModel.handleKeyMsg()
    └── TUIModel.handleCommandLineInput()  [handles everything - 266 lines]
            ├── Editing (backspace, delete, cursor)
            ├── History navigation
            ├── Completion navigation
            └── Command execution
```

#### After (Component-Based)
```
TUIModel.handleKeyMsg()
    └── CommandLineComponent.HandleKey()  [handles ALL keys]
            ├── Returns commandReadyMsg → TUIModel.handleCustomMessages()
            ├── Returns commandCancelledMsg → TUIModel.handleCustomMessages()
            ├── Returns commandTextChangedMsg → TUIModel.handleCustomMessages()
            ├── Returns navigateCompletionMsg → TUIModel.handleCustomMessages()
            ├── Returns navigateHistoryMsg → TUIModel.handleCustomMessages()
            └── Returns acceptCompletionMsg → TUIModel.handleCustomMessages()
```

### Benefits

✅ **Complete Separation** - Component handles ALL its own input  
✅ **Clean Message Passing** - Uses bubbletea pattern throughout  
✅ **Reduced Code** - Removed 266 lines from TUI  
✅ **Better Encapsulation** - All command line logic in one place  
✅ **Easier Testing** - Component can be tested independently  
✅ **Maintainability** - Clear responsibilities  

---

## Mouse Event Handling

### Problem

Mouse wheel events were being processed twice, causing duplicate scrolling and potentially affecting areas outside the content window:

1. **Duplicate Processing**: Events were handled by both `content.Update(msg)` and `handleMouseMsg()`
2. **No Position Checking**: Events were processed regardless of mouse cursor position
3. **Unintended Side Effects**: Mouse wheel could affect prompt history navigation

### Solution: Position-Aware Single Processing

The mouse event handling now follows these principles:

1. **Position Checking**: Only process events within the content area
2. **Single Handler**: Events are processed exactly once
3. **Component Delegation**: Content component handles its own scrolling

### Implementation

**Position Checking** (`tui.go`):
```go
case tea.MouseMsg:
    // Only handle mouse events if they're within the content area
    // Content area is from top of screen to just above the prompt
    contentHeight := m.height - 6 // Subtract prompt, status, command line, etc.
    if msg.Y < contentHeight {
        // Mouse is in content area - let content component handle it
        var contentCmd tea.Cmd
        m.content, contentCmd = m.content.Update(msg)
        return m, contentCmd
    }
    // Mouse is outside content area - ignore it
    return m, nil
```

**Content Height Calculation**:
```go
contentHeight := m.height - 6

// Breakdown:
// - Command line: 1 line
// - Status line: 1 line
// - Prompt (with borders): ~2-3 lines
// - Empty line: 1 line
// = 6 lines reserved for UI chrome
```

### Event Flow

```
Mouse Wheel Event (Y position)
       ↓
   TUI Update()
       ↓
   Check: msg.Y < contentHeight?
       ↓
       ├─→ YES (in content area)
       │        ↓
       │   content.Update(msg)
       │        ↓
       │   Chat.Update(msg)
       │        ↓
       │   Viewport.ScrollUp/Down()  ← Single scroll ✓
       │
       └─→ NO (outside content area)
                ↓
           Ignore event  ← No scrolling ✓
```

### Component Responsibilities

**TUIModel** (`tui.go`):
- Checks mouse event position
- Routes events to content component only if in content area
- Ignores events outside content area

**ContentComponent** (`content.go`):
- Receives mouse events from TUI
- Delegates to active view (chat, help, models, resume)
- Handles view-specific mouse behavior

**ChatComponent** (`chat.go`):
- Handles mouse wheel scrolling
- Updates viewport position
- Tracks user scrolling vs auto-scrolling
- Supports touch gestures (drag scrolling)

**PromptComponent** (`prompt.go`):
- Does NOT handle mouse events
- Only responds to keyboard input
- History navigation via up/down arrow keys

### Supported Mouse Events

**In Content Area**:
- `MouseWheelUp` - Scroll content up
- `MouseWheelDown` - Scroll content down
- `MouseLeft` + `MouseActionPress` - Start touch drag
- `MouseMotion` - Touch drag scrolling
- `MouseLeft` + `MouseActionRelease` - End touch drag

**Outside Content Area**:
- All mouse events are ignored

### Benefits

✅ **No Duplicate Scrolling** - Each mouse event processed exactly once  
✅ **Position Awareness** - Events only affect area under cursor  
✅ **Clean Separation** - Each component handles its own input  
✅ **Predictable Behavior** - Users get expected scrolling  
✅ **No Side Effects** - Prompt history navigation unaffected  
✅ **Touch Support** - Drag scrolling works in content area  

### Testing Scenarios

**Scenario 1: Scroll in Chat Area**
```
User: Scrolls mouse wheel over chat messages
Expected: Chat scrolls smoothly
Result: ✓ Works correctly
```

**Scenario 2: Scroll over Prompt**
```
User: Scrolls mouse wheel over prompt input
Expected: Nothing happens
Result: ✓ Works correctly (event ignored)
```

**Scenario 3: Keyboard History Navigation**
```
User: Presses up/down arrows in prompt
Expected: Navigate through prompt history
Result: ✓ Works correctly (unaffected by mouse fix)
```

**Scenario 4: Touch Drag Scrolling**
```
User: Drags finger/mouse in chat area
Expected: Chat scrolls based on drag distance
Result: ✓ Works correctly (handled by chat component)
```

---

## Message Flow Patterns

### Pattern 1: Component State Change

```
User action
  ↓
Component method called
  ↓
Component updates internal state
  ↓
Returns ChangeModeMsg (or other message)
  ↓
TUI receives in handleCustomMessages
  ↓
TUI updates global state
  ↓
View() renders
```

### Pattern 2: Component Input Handling

```
User presses key
  ↓
TUI.handleKeyMsg() checks active component
  ↓
Calls Component.HandleKey(msg)
  ↓
Component processes key
  ↓
Returns (tea.Cmd, bool)
  ↓
TUI executes command (if any)
  ↓
Message flows back to handleCustomMessages
```

### Pattern 3: Batched Messages

Components can return multiple messages at once:

```go
case "esc":
    exitCmd := cl.ExitCommandMode()  // ChangeModeMsg{NewMode: "insert"}
    return tea.Batch(
        exitCmd,
        func() tea.Msg { return commandCancelledMsg{} },
    ), true
```

### Pattern 4: Command Chaining

Methods return commands that can be chained:

```go
// Before
m.content.ShowHelp(msg.topic)
return m, nil

// After
return m, m.content.ShowHelp(msg.topic)  // Returns ChangeModeMsg
```

---

## Future Improvements

### Mode Management
- Add mode transition validation (e.g., can't go from help → command directly)
- Track mode history for undo/debugging
- Add mode change hooks for logging/analytics
- Make mode changes async if needed
- Add mode-specific state (e.g., help topic, selected model)

### Command Line Component
- Extract completion handling to a separate component
- Make history navigation a reusable component
- Add more unit tests for `CommandLineComponent.HandleKey()`

### General Architecture
- Consider extracting more components (e.g., StatusComponent could handle its own updates)
- Add component lifecycle hooks (Init, Cleanup)
- Implement component-level error handling
- Add telemetry/metrics for component interactions

---

## Files Reference

### Core Files
- `tui.go` - Main TUI model and coordination logic
- `commandline.go` - Command line component
- `content.go` - Content view component (chat/help/models/resume)
- `prompt.go` - User input prompt component
- `status.go` - Status bar component

### Key Changes Made

**commandline.go**:
- Added message types (`ChangeModeMsg`, `commandReadyMsg`, etc.)
- Added `HandleKey()` method
- Updated `EnterCommandMode()` and `ExitCommandMode()` to return messages

**content.go**:
- Updated `ShowChat()`, `ShowHelp()`, `ShowModels()`, `ShowResume()` to return `ChangeModeMsg`
- Updated exit handlers to return commands
- Updated selection handlers to batch commands

**tui.go**:
- Added `Mode` field to `TUIModel`
- Added centralized `ChangeModeMsg` handler
- Removed mode polling logic from `View()`
- Updated all vi mode changes to send `ChangeModeMsg`
- Updated all component call sites to use returned commands
- Removed 266 lines of command line input handling

---

## Testing

All refactorings maintain backward compatibility:

✅ All existing tests pass  
✅ No behavior changes for users  
✅ Mode display works correctly for all states  
✅ Completion dialog works in command line mode  
✅ History navigation works correctly  
✅ Vim-style partial command matching integrated  

---

## Summary

These refactorings established a clean, maintainable architecture:

1. **Mode Management**: Single source of truth with centralized message handling
2. **Component Boundaries**: Each component handles its own input and state
3. **Message Passing**: Clean communication via bubbletea messages
4. **No Polling**: View() just renders, no logic
5. **Testability**: Components can be tested in isolation

The result is a more maintainable, extensible, and testable codebase that follows bubbletea best practices.
