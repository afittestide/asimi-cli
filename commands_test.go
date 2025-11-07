package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommandRegistryOrder(t *testing.T) {
	registry := NewCommandRegistry()
	commands := registry.GetAllCommands()
	if len(commands) == 0 {
		t.Fatalf("expected commands to be registered")
	}
	if commands[0].Name != "/help" {
		t.Fatalf("expected first command to be /help, got %s", commands[0].Name)
	}
}

func TestHandleHelpCommandTopic(t *testing.T) {
	model := &TUIModel{}

	cmd := handleHelpCommand(model, nil)
	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	msg := cmd()
	helpMsg, ok := msg.(showHelpMsg)
	if !ok {
		t.Fatalf("expected showHelpMsg got %T", msg)
	}
	if helpMsg.topic != "index" {
		t.Fatalf("expected default topic 'index', got %q", helpMsg.topic)
	}

	cmd = handleHelpCommand(model, []string{"modes"})
	msg = cmd()
	helpMsg, ok = msg.(showHelpMsg)
	if !ok {
		t.Fatalf("expected showHelpMsg got %T", msg)
	}
	if helpMsg.topic != "modes" {
		t.Fatalf("expected topic 'modes', got %q", helpMsg.topic)
	}
}

func TestCommandRegistryColonLookup(t *testing.T) {
	registry := NewCommandRegistry()

	cmd, ok := registry.GetCommand(":help")
	if !ok {
		t.Fatalf("expected :help to resolve to registered command")
	}
	if cmd.Name != "/help" {
		t.Fatalf("expected normalized command name to be /help, got %s", cmd.Name)
	}

	registry.RegisterCommand(":custom", "custom command", func(*TUIModel, []string) tea.Cmd { return nil })
	if _, ok := registry.Commands["/custom"]; !ok {
		t.Fatalf("expected custom command to be normalized at registration")
	}
}
