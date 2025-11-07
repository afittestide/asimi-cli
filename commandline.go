package main

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Toast represents a single toast notification
type Toast struct {
	ID      string
	Message string
	Type    string // info, success, warning, error
	Created time.Time
	Timeout time.Duration
}

// CommandLineMode represents the state of the command line
type CommandLineMode int

const (
	CommandLineIdle CommandLineMode = iota
	CommandLineCommand
	CommandLineToast
)

// CommandLineComponent manages the bottom command line
// Handles both : commands and toast notifications
type CommandLineComponent struct {
	mode         CommandLineMode
	toasts       []Toast
	command      string
	cursorPos    int // Cursor position in command string
	width        int
	style        lipgloss.Style
	showCursor   bool
	cursorBlink  bool
	lastBlink    time.Time
	blinkRate    time.Duration
}

// NewCommandLineComponent creates a new command line component
func NewCommandLineComponent() *CommandLineComponent {
	return &CommandLineComponent{
		mode:        CommandLineIdle,
		toasts:      make([]Toast, 0),
		cursorPos:   0,
		showCursor:  true,
		cursorBlink: true,
		blinkRate:   500 * time.Millisecond,
		lastBlink:   time.Now(),
		style: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1).
			MaxWidth(50),
	}
}

// AddToast adds a new toast notification
func (cl *CommandLineComponent) AddToast(message, toastType string, timeout time.Duration) {
	toast := Toast{
		ID:      time.Now().String(),
		Message: message,
		Type:    toastType,
		Created: time.Now(),
		Timeout: timeout,
	}
	cl.toasts = append(cl.toasts, toast)
}

// RemoveToast removes a toast by ID
func (cl *CommandLineComponent) RemoveToast(id string) {
	for i, toast := range cl.toasts {
		if toast.ID == id {
			cl.toasts = append(cl.toasts[:i], cl.toasts[i+1:]...)
			break
		}
	}
}

// ClearToasts removes all existing toast notifications
func (cl *CommandLineComponent) ClearToasts() {
	cl.toasts = nil
}

// EnterCommandMode enters command mode with optional initial text
func (cl *CommandLineComponent) EnterCommandMode(initialText string) {
	cl.mode = CommandLineCommand
	cl.command = initialText
	cl.cursorPos = len(initialText)
	cl.showCursor = true
	cl.cursorBlink = true
}

// ExitCommandMode exits command mode and returns to idle
func (cl *CommandLineComponent) ExitCommandMode() {
	cl.mode = CommandLineIdle
	cl.command = ""
	cl.cursorPos = 0
}

// IsInCommandMode returns true if in command mode
func (cl *CommandLineComponent) IsInCommandMode() bool {
	return cl.mode == CommandLineCommand
}

// SetCommand sets the current command being entered
func (cl *CommandLineComponent) SetCommand(cmd string) {
	cl.command = cmd
	cl.cursorPos = len(cmd)
	if cmd != "" {
		cl.mode = CommandLineCommand
	} else {
		cl.mode = CommandLineIdle
	}
}

// InsertRune inserts a character at cursor position
func (cl *CommandLineComponent) InsertRune(r rune) {
	if cl.mode != CommandLineCommand {
		return
	}
	before := cl.command[:cl.cursorPos]
	after := cl.command[cl.cursorPos:]
	cl.command = before + string(r) + after
	cl.cursorPos++
}

// DeleteCharBackward deletes character before cursor (backspace)
func (cl *CommandLineComponent) DeleteCharBackward() {
	if cl.mode != CommandLineCommand || cl.cursorPos == 0 {
		return
	}
	before := cl.command[:cl.cursorPos-1]
	after := cl.command[cl.cursorPos:]
	cl.command = before + after
	cl.cursorPos--
}

// DeleteCharForward deletes character at cursor (delete key)
func (cl *CommandLineComponent) DeleteCharForward() {
	if cl.mode != CommandLineCommand || cl.cursorPos >= len(cl.command) {
		return
	}
	before := cl.command[:cl.cursorPos]
	after := cl.command[cl.cursorPos+1:]
	cl.command = before + after
}

// MoveCursorLeft moves cursor one position left
func (cl *CommandLineComponent) MoveCursorLeft() {
	if cl.cursorPos > 0 {
		cl.cursorPos--
	}
}

// MoveCursorRight moves cursor one position right
func (cl *CommandLineComponent) MoveCursorRight() {
	if cl.cursorPos < len(cl.command) {
		cl.cursorPos++
	}
}

// MoveCursorHome moves cursor to start
func (cl *CommandLineComponent) MoveCursorHome() {
	cl.cursorPos = 0
}

// MoveCursorEnd moves cursor to end
func (cl *CommandLineComponent) MoveCursorEnd() {
	cl.cursorPos = len(cl.command)
}

// GetCommand returns the current command
func (cl *CommandLineComponent) GetCommand() string {
	return cl.command
}

// ClearCommand clears the current command
func (cl *CommandLineComponent) ClearCommand() {
	cl.command = ""
	cl.mode = CommandLineIdle
}

// SetWidth sets the width for rendering
func (cl *CommandLineComponent) SetWidth(width int) {
	cl.width = width
}

// Update handles updating the command line (e.g., removing expired toasts, cursor blink)
func (cl *CommandLineComponent) Update() {
	now := time.Now()

	// Update cursor blink
	if cl.mode == CommandLineCommand && now.Sub(cl.lastBlink) >= cl.blinkRate {
		cl.cursorBlink = !cl.cursorBlink
		cl.lastBlink = now
	}

	// Remove expired toasts
	activeToasts := make([]Toast, 0)
	for _, toast := range cl.toasts {
		if now.Sub(toast.Created) < toast.Timeout {
			activeToasts = append(activeToasts, toast)
		}
	}
	cl.toasts = activeToasts
}

// View renders the command line
func (cl *CommandLineComponent) View() string {
	// Priority 1: Show command if in command mode
	if cl.mode == CommandLineCommand {
		// Build command text with cursor
		cmdText := ":" + cl.command

		// Insert cursor at position (account for leading ":")
		displayPos := cl.cursorPos + 1
		var displayText string
		if cl.showCursor && cl.cursorBlink {
			if displayPos < len(cmdText) {
				// Cursor in middle of text
				before := cmdText[:displayPos]
				cursorChar := string(cmdText[displayPos])
				after := cmdText[displayPos+1:]
				cursorStyle := lipgloss.NewStyle().Reverse(true)
				displayText = before + cursorStyle.Render(cursorChar) + after
			} else {
				// Cursor at end
				cursorStyle := lipgloss.NewStyle().Reverse(true)
				displayText = cmdText + cursorStyle.Render(" ")
			}
		} else {
			displayText = cmdText
		}

		cmdStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Width(cl.width)
		return cmdStyle.Render(displayText)
	}

	// Priority 2: Show toast if active
	if len(cl.toasts) > 0 {
		toast := cl.toasts[len(cl.toasts)-1]
		style := cl.style

		contentWidth := lipgloss.Width(toast.Message)
		frameWidth, _ := style.GetFrameSize()
		maxWidth := style.GetMaxWidth()
		if maxWidth > 0 && contentWidth+frameWidth > maxWidth {
			style = style.Copy().MaxWidth(contentWidth + frameWidth)
		}

		switch toast.Type {
		case "info":
			style = style.Background(lipgloss.NoColor{})
		case "success":
			style = style.Background(lipgloss.Color("76")) // Green
		case "warning":
			style = style.Background(lipgloss.Color("11")) // Yellow
		case "error":
			style = style.Background(lipgloss.Color("124")) // Red
		}

		return style.Render(toast.Message)
	}

	// Default: Show blank line
	return ""
}
