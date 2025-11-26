package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	gogit "github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms/fake"
)

// mockConfig returns a mock configuration for testing
func mockConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 3000,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "asimi",
			Password: "asimi",
			Name:     "asimi_dev",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		LLM: LLMConfig{
			Provider: "fake",
			Model:    "mock-model",
			APIKey:   "",
			BaseURL:  "",
		},
	}
}

// containsMessage checks if any message in the slice contains the given substring
func containsMessage(messages []string, substring string) bool {
	for _, msg := range messages {
		if strings.Contains(msg, substring) {
			return true
		}
	}
	return false
}

// TestTUIModelInit tests the initialization of the TUI model
func TestTUIModelInit(t *testing.T) {
	model := NewTUIModel(mockConfig(), nil, nil, nil, nil, nil)
	cmd := model.Init()

	// Init should return nil as there's no initial command
	require.Nil(t, cmd)
}

// TestTUIModelWindowSizeMsg tests handling of window size messages
func TestTUIModelWindowSizeMsg(t *testing.T) {
	model := NewTUIModel(mockConfig(), nil, nil, nil, nil, nil)

	// Send a window size message
	newModel, cmd := model.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	updatedModel, ok := newModel.(TUIModel)
	require.True(t, ok)
	require.Equal(t, 100, updatedModel.width)
	require.Equal(t, 50, updatedModel.height)
	require.Nil(t, cmd)
}

// newTestModel creates a new TUIModel for testing purposes.
func newTestModel(t *testing.T) (*TUIModel, *fake.LLM) {
	llm := fake.NewFakeLLM([]string{})
	model := NewTUIModel(mockConfig(), nil, nil, nil, nil, nil)
	// Disable persistent history to keep tests hermetic.
	model.persistentPromptHistory = nil
	model.initHistory()
	// Use native session path for tests now that legacy agent is removed.
	sess, err := NewSession(llm, &Config{LLM: LLMConfig{Provider: "fake"}}, RepoInfo{}, func(any) {})
	require.NoError(t, err)
	model.SetSession(sess)
	return model, llm
}

func TestCommandCompletionOrderDefaultsToHelp(t *testing.T) {
	model := NewTUIModel(mockConfig(), nil, nil, nil, nil, nil)
	model.prompt.SetValue(":")
	model.completionMode = "command"
	model.updateCommandCompletions()
	require.NotEmpty(t, model.completions.Options)
	require.Equal(t, ":help", model.completions.Options[0])
}

// TestTUIModelKeyMsg tests quitting the application with 'q' and Ctrl+C
func TestTUIModelKeyMsg(t *testing.T) {
	testCases := []struct {
		name          string
		key           tea.KeyMsg
		expectQuit    bool
		expectCommand bool
	}{
		{
			name:          "Quit with 'q'",
			key:           tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
			expectQuit:    false,
			expectCommand: false,
		},
		{
			name:          "First 'ctrl+c' does not quit",
			key:           tea.KeyMsg{Type: tea.KeyCtrlC},
			expectQuit:    false,
			expectCommand: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := NewTUIModel(mockConfig(), nil, nil, nil, nil, nil)

			// Send a quit key message
			newModel, cmd := model.Update(tc.key)

			if tc.expectCommand {
				require.NotNil(t, cmd)
			} else {
				require.Nil(t, cmd)
			}

			if tc.expectQuit {
				// Execute the command to verify it's a quit command
				result := cmd()
				_, ok := result.(tea.QuitMsg)
				require.True(t, ok)
			}

			// Model should be unchanged
			_, ok := newModel.(TUIModel)
			require.True(t, ok)
		})
	}
}

func TestDoubleCtrlCToQuit(t *testing.T) {
	model := NewTUIModel(mockConfig(), nil, nil, nil, nil, nil)

	// First CTRL-C should not quit
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.Nil(t, cmd)
	tuiModel, ok := newModel.(TUIModel)
	require.True(t, ok)
	require.False(t, tuiModel.ctrlCPressedTime.IsZero())

	// Second CTRL-C should quit (wait slightly longer than debounce time)
	time.Sleep(ctrlCDebounceTime + 10*time.Millisecond)
	newModel, cmd = tuiModel.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.NotNil(t, cmd)
	result := cmd()
	_, ok = result.(tea.QuitMsg)
	require.True(t, ok)
}

func TestTUIModelSubmit(t *testing.T) {
	t.Skip("TODO: fix this test")
	testCases := []struct {
		name                 string
		initialEditorValue   string
		expectedMessageCount int
		expectedLastMessage  string
		expectCommand        bool
	}{
		{
			name:                 "Submit empty message",
			initialEditorValue:   "",
			expectedMessageCount: 1,
			expectedLastMessage:  "Welcome to Asimi CLI! Send a message to start chatting.",
			expectCommand:        false,
		},
		{
			name:                 "Submit command",
			initialEditorValue:   "/help",
			expectedMessageCount: 2,
			expectedLastMessage:  "Available commands:",
			expectCommand:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model, _ := newTestModel(t)

			model.prompt.SetValue(tc.initialEditorValue)

			newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

			if tc.expectCommand {
				require.NotNil(t, cmd)
				msg := cmd()
				newModel, cmd = newModel.Update(msg)
				require.Nil(t, cmd)
			} else {
				require.Nil(t, cmd)
			}

			chat := model.content.GetChat()
			require.Equal(t, tc.expectedMessageCount, len(chat.Messages))
			require.Contains(t, chat.Messages[len(chat.Messages)-1], tc.expectedLastMessage, "prompt", tc.name)
		})
	}
}

func TestTUIModelKeyboardInteraction(t *testing.T) {
	testCases := []struct {
		name   string
		key    tea.KeyMsg
		setup  func(model *TUIModel)
		verify func(t *testing.T, model *TUIModel, cmd tea.Cmd)
	}{
		{
			name: "Escape key",
			key:  tea.KeyMsg{Type: tea.KeyEsc},
			setup: func(model *TUIModel) {
				model.modal = NewBaseModal("Test", "Test content", 30, 10)
				model.showCompletionDialog = true
			},
			verify: func(t *testing.T, model *TUIModel, cmd tea.Cmd) {
				require.Nil(t, cmd)
				require.Nil(t, model.modal)
				require.False(t, model.showCompletionDialog)
			},
		},
		{
			name: "Down arrow in completion dialog",
			key:  tea.KeyMsg{Type: tea.KeyDown},
			setup: func(model *TUIModel) {
				model.showCompletionDialog = true
				model.completions.SetOptions([]string{"option1", "option2", "option3"})
				model.completions.Show()
			},
			verify: func(t *testing.T, model *TUIModel, cmd tea.Cmd) {
				require.Nil(t, cmd)
				require.Equal(t, 1, model.completions.Selected)
			},
		},
		{
			name: "Up arrow in completion dialog",
			key:  tea.KeyMsg{Type: tea.KeyUp},
			setup: func(model *TUIModel) {
				model.showCompletionDialog = true
				model.completions.SetOptions([]string{"option1", "option2", "option3"})
				model.completions.Show()
				model.completions.Selected = 1
			},
			verify: func(t *testing.T, model *TUIModel, cmd tea.Cmd) {
				require.Nil(t, cmd)
				require.Equal(t, 0, model.completions.Selected)
			},
		},
		{
			name: "Tab to select in completion dialog",
			key:  tea.KeyMsg{Type: tea.KeyTab},
			setup: func(model *TUIModel) {
				model.showCompletionDialog = true
				model.completionMode = "command"
				model.completions.SetOptions([]string{":help", "option2", "option3"})
				model.completions.Show()
			},
			verify: func(t *testing.T, model *TUIModel, cmd tea.Cmd) {
				require.Nil(t, cmd)
				require.Equal(t, ViewHelp, model.content.GetActiveView())
				require.Equal(t, "index", model.content.help.GetTopic())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model, _ := newTestModel(t)
			if tc.setup != nil {
				tc.setup(model)
			}

			newModel, cmd := model.Update(tc.key)
			updatedModel, ok := newModel.(TUIModel)
			require.True(t, ok)

			for cmd != nil {
				msg := cmd()
				newModel, cmd = updatedModel.Update(msg)
				updatedModel, ok = newModel.(TUIModel)
				require.True(t, ok)
			}

			tc.verify(t, &updatedModel, cmd)
		})
	}
}

// TestTUIModelView tests the view rendering
func TestTUIModelView(t *testing.T) {
	model := NewTUIModel(mockConfig(), nil, nil, nil, nil, nil)

	// Test view rendering with zero dimensions (should show initializing)
	view := model.View()
	require.NotEmpty(t, view)
	require.Contains(t, view, "Initializing...")

	// Set dimensions and test proper rendering
	model.width = 80
	model.height = 24
	view = model.View()
	require.NotEmpty(t, view)
	require.NotContains(t, view, "Initializing...")

}

// TestPromptComponent tests the prompt component
func TestPromptComponent(t *testing.T) {
	prompt := NewPromptComponent(50, 10)

	// Test setting and getting value
	testValue := "Test content"
	prompt.SetValue(testValue)
	require.Equal(t, testValue, prompt.Value())

	// Test dimensions
	prompt.SetWidth(60)
	require.Equal(t, 60, prompt.Width)

	prompt.SetHeight(15)
	require.Equal(t, 15, prompt.Height)
}

// TestChatComponent tests the chat component
func TestChatComponent(t *testing.T) {
	chat := NewChatComponent(50, 10, false)

	// Should have initial welcome message
	require.Equal(t, 1, len(chat.Messages))
	require.Equal(t, "Welcome to Asimi CLI! Send a message to start chatting.", chat.Messages[0])

	// Test adding a message
	testMessage := "Test message"
	chat.AddMessage(testMessage)
	require.Equal(t, 2, len(chat.Messages))
	require.Equal(t, testMessage, chat.Messages[1])

	// Test dimensions
	chat.SetWidth(60)
	require.Equal(t, 60, chat.Width)

	chat.SetHeight(15)
	require.Equal(t, 15, chat.Height)
}

// TestCompletionDialog tests the completion dialog
func TestCompletionDialog(t *testing.T) {
	dialog := NewCompletionDialog()

	// Initially should be invisible
	require.False(t, dialog.Visible)
	require.Empty(t, dialog.Options)
	require.Equal(t, 0, dialog.Selected)

	// Test setting options
	options := []string{"option1", "option2", "option3"}
	dialog.SetOptions(options)
	require.Equal(t, options, dialog.Options)

	// Test showing and hiding
	dialog.Show()
	require.True(t, dialog.Visible)

	dialog.Hide()
	require.False(t, dialog.Visible)

	// Test selection navigation
	dialog.Show()
	dialog.SetOptions(options)

	dialog.SelectNext()
	require.Equal(t, 1, dialog.Selected)

	dialog.SelectNext()
	require.Equal(t, 2, dialog.Selected)

	dialog.SelectNext()
	require.Equal(t, 2, dialog.Selected)

	dialog.SelectPrev()
	require.Equal(t, 1, dialog.Selected)

	// Test getting selected option
	dialog.Selected = 1
	require.Equal(t, "option2", dialog.GetSelected())

	// Test view rendering
	// When not visible, should be empty
	dialog.Hide()
	view := dialog.View()
	require.Empty(t, view)

	// When visible but no options, should be empty
	dialog.Show()
	dialog.SetOptions([]string{})
	view = dialog.View()
	require.Empty(t, view)

	// When visible with options, should contain the options
	dialog.SetOptions(options)
	view = dialog.View()
	require.NotEmpty(t, view)
	// When visible, should contain the options
	for _, option := range options {
		require.Contains(t, view, option)
	}
}

// TestCompletionDialogScrolling tests the scrolling functionality of the completion dialog
func TestCompletionDialogScrolling(t *testing.T) {
	dialog := NewCompletionDialog()
	dialog.MaxHeight = 5
	dialog.ScrollMargin = 1
	options := []string{"a", "b", "c", "d", "e", "f", "g"}
	dialog.SetOptions(options)

	// Initial state
	require.Equal(t, 0, dialog.Selected)
	require.Equal(t, 0, dialog.Offset)

	// Scroll down
	dialog.SelectNext() // b
	require.Equal(t, 1, dialog.Selected)
	require.Equal(t, 0, dialog.Offset)

	dialog.SelectNext() // c
	require.Equal(t, 2, dialog.Selected)
	require.Equal(t, 0, dialog.Offset)

	dialog.SelectNext() // d
	require.Equal(t, 3, dialog.Selected)
	require.Equal(t, 0, dialog.Offset)

	dialog.SelectNext() // e, enters scroll margin
	require.Equal(t, 4, dialog.Selected)
	require.Equal(t, 1, dialog.Offset) // scrolled

	dialog.SelectNext() // f
	require.Equal(t, 5, dialog.Selected)
	require.Equal(t, 2, dialog.Offset)

	dialog.SelectNext() // g, at the end
	require.Equal(t, 6, dialog.Selected)
	require.Equal(t, 2, dialog.Offset) // offset is maxed out

	// Try to scroll past the end
	dialog.SelectNext() // g
	require.Equal(t, 6, dialog.Selected)
	require.Equal(t, 2, dialog.Offset)

	// Scroll up
	dialog.SelectPrev() // f
	require.Equal(t, 5, dialog.Selected)
	require.Equal(t, 2, dialog.Offset)

	dialog.SelectPrev() // e
	require.Equal(t, 4, dialog.Selected)
	require.Equal(t, 2, dialog.Offset)

	dialog.SelectPrev() // d
	require.Equal(t, 3, dialog.Selected)
	require.Equal(t, 2, dialog.Offset)

	dialog.SelectPrev() // c, enters scroll margin
	require.Equal(t, 2, dialog.Selected)
	require.Equal(t, 1, dialog.Offset)

	dialog.SelectPrev() // b, enters scroll margin
	require.Equal(t, 1, dialog.Selected)
	require.Equal(t, 0, dialog.Offset)

	dialog.SelectPrev() // a
	require.Equal(t, 0, dialog.Selected)
	require.Equal(t, 0, dialog.Offset)

	// Try to scroll past the beginning
	dialog.SelectPrev() // a
	require.Equal(t, 0, dialog.Selected)
	require.Equal(t, 0, dialog.Offset)
}

// TestStatusComponent tests the status component
func TestStatusComponent(t *testing.T) {
	status := NewStatusComponent(50)

	// Test setting properties with new API
	status.SetProvider("test", "model", true)

	// Set repo info to test branch rendering
	repoInfo := &RepoInfo{
		Branch: "main",
	}
	status.SetRepoInfo(repoInfo)

	// Test width
	status.SetWidth(60)
	require.Equal(t, 60, status.Width)

	// Test view rendering
	view := status.View()
	require.NotEmpty(t, view)
	// The new status format includes git branch and provider info
	require.Contains(t, view, "main")       // Should contain branch name
	require.Contains(t, view, "test-model") // Should contain provider-model
	require.Contains(t, view, "✅")          // Should contain connected icon
}

func TestSummarizeStatus(t *testing.T) {
	cases := []struct {
		name     string
		status   gogit.Status
		expected string
	}{
		{
			name:     "empty status",
			status:   gogit.Status{},
			expected: "",
		},
		{
			name: "mixed indicators",
			status: gogit.Status{
				"modified.go": &gogit.FileStatus{
					Staging:  gogit.Modified,
					Worktree: gogit.Unmodified,
				},
				"staged_added.go": &gogit.FileStatus{
					Staging:  gogit.Added,
					Worktree: gogit.Unmodified,
				},
				"deleted.txt": &gogit.FileStatus{
					Staging:  gogit.Deleted,
					Worktree: gogit.Unmodified,
				},
				"renamed.txt": &gogit.FileStatus{
					Staging:  gogit.Renamed,
					Worktree: gogit.Unmodified,
				},
				"untracked.md": &gogit.FileStatus{
					Staging:  gogit.Untracked,
					Worktree: gogit.Untracked,
				},
				"worktree_modified.go": &gogit.FileStatus{
					Staging:  gogit.Unmodified,
					Worktree: gogit.Modified,
				},
			},
			expected: "[!+-→?]",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, summarizeStatus(tc.status))
		})
	}
}

// TestBaseModal tests the base modal component
func TestBaseModal(t *testing.T) {
	title := "Test Modal"
	content := "This is a test modal"
	modal := NewBaseModal(title, content, 30, 10)

	require.Equal(t, title, modal.Title)
	require.Equal(t, content, modal.Content)
	require.Equal(t, 30, modal.Width)
	require.Equal(t, 10, modal.Height)

	// Test rendering
	view := modal.Render()
	require.NotEmpty(t, view)
	require.Contains(t, view, title)
	require.Contains(t, view, content)
}

// TestCommandLine tests the command line component (including toast functionality)
func TestCommandLine(t *testing.T) {
	commandLine := NewCommandLineComponent()

	// Initially should have no toasts
	require.Empty(t, commandLine.toasts)

	// Test adding a toast
	message := "Test toast message"
	toastType := "info"
	timeout := 5 * time.Second

	commandLine.AddToast(message, toastType, timeout)
	require.Equal(t, 1, len(commandLine.toasts))

	// Test view rendering with toast
	view := commandLine.View()
	require.NotEmpty(t, view)
	require.Contains(t, view, message)

	// Test clearing toasts
	commandLine.ClearToasts()
	require.Empty(t, commandLine.toasts)

	// Re-add toast to verify removal still works
	commandLine.AddToast(message, toastType, timeout)
	require.Equal(t, 1, len(commandLine.toasts))

	// Test removing a toast
	toastID := commandLine.toasts[0].ID
	commandLine.RemoveToast(toastID)
	require.Empty(t, commandLine.toasts)

	// Test updating (removing expired toasts)
	commandLine.AddToast(message, toastType, 1*time.Millisecond)
	time.Sleep(2 * time.Millisecond) // Wait for toast to expire
	commandLine.Update()
	require.Empty(t, commandLine.toasts)
}

func TestCommandLineBackspaceAtLineStartExitsCommandMode(t *testing.T) {
	commandLine := NewCommandLineComponent()
	commandLine.EnterCommandMode("")
	require.True(t, commandLine.IsInCommandMode())

	cmd, handled := commandLine.HandleKey(tea.KeyMsg{Type: tea.KeyBackspace})
	require.True(t, handled)
	require.NotNil(t, cmd)
	require.False(t, commandLine.IsInCommandMode())
	require.Equal(t, "", commandLine.GetCommand())
	require.Equal(t, 0, commandLine.cursorPos)
}

// TestTUIModelUpdateFileCompletions tests the file completion functionality with multiple files
func TestTUIModelUpdateFileCompletions(t *testing.T) {
	model, _ := newTestModel(t)

	// Set up mock file list
	files := []string{
		"main.go",
		"utils.go",
		"config.json",
		"README.md",
		"docs/guide.md",
		"test/utils_test.go",
	}

	// Test single file completion
	model.prompt.SetValue("@mai")
	model.updateFileCompletions(files)
	require.Equal(t, 1, len(model.completions.Options))
	require.Contains(t, model.completions.Options[0], "main.go")

	// Test multiple matching files
	model.prompt.SetValue("@util")
	model.updateFileCompletions(files)
	require.Equal(t, 2, len(model.completions.Options))
	require.True(t,
		(strings.Contains(model.completions.Options[0], "utils.go") && strings.Contains(model.completions.Options[1], "utils_test.go")) ||
			(strings.Contains(model.completions.Options[1], "utils.go") && strings.Contains(model.completions.Options[0], "utils_test.go")))

	// Test multiple file references in one input
	model.prompt.SetValue("Check these files: @main.go and @config")
	model.updateFileCompletions(files)
	require.Equal(t, 1, len(model.completions.Options))
	require.Contains(t, model.completions.Options[0], "config.json")

}

// TestRenderHomeView tests the home view rendering
func TestRenderHomeView(t *testing.T) {
	model := NewTUIModel(mockConfig(), nil, nil, nil, nil, nil)
	model.width = 80
	model.height = 24

	view := model.renderHomeView(80, 24)
	require.NotEmpty(t, view)
	require.Contains(t, view, "vi")
	require.Contains(t, view, "Asimi")
}

// TestColonCommandCompletion tests command completion with colon prefix in vi mode
func TestColonCommandCompletion(t *testing.T) {
	model, _ := newTestModel(t)

	// Test initial colon shows all commands with colon prefix
	model.prompt.SetValue(":")
	model.completionMode = "command"
	model.updateCommandCompletions()
	require.NotEmpty(t, model.completions.Options)
	// All options should start with ":"
	for _, opt := range model.completions.Options {
		require.True(t, strings.HasPrefix(opt, ":"), "Command should start with : but got: %s", opt)
	}

	// Test filtering with partial command
	model.prompt.SetValue(":he")
	model.updateCommandCompletions()
	require.NotEmpty(t, model.completions.Options)
	require.Contains(t, model.completions.Options, ":help")

	// Test filtering with more specific command
	model.prompt.SetValue(":new")
	model.updateCommandCompletions()
	require.NotEmpty(t, model.completions.Options)
	require.Contains(t, model.completions.Options, ":new")
}

// TestColonInNormalModeShowsCompletion tests that pressing : in normal mode shows completion dialog
func TestColonInNormalModeActivatesCommandLine(t *testing.T) {
	model, _ := newTestModel(t)

	// Start in insert mode
	require.True(t, model.prompt.IsViInsertMode())

	// Press Esc to enter normal mode
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedModel := newModel.(TUIModel)
	require.True(t, updatedModel.prompt.IsViNormalMode())

	// Press : to enter command-line mode
	newModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	updatedModel = newModel.(TUIModel)

	require.True(t, updatedModel.commandLine.IsInCommandMode(), "command line should enter command mode")
	require.False(t, updatedModel.showCompletionDialog, "completion dialog should not be shown automatically")
	require.Equal(t, "", updatedModel.commandLine.GetCommand(), "command buffer should be empty")
	require.False(t, updatedModel.prompt.TextArea.Focused(), "prompt should lose focus while command line active")
}

func TestShowHelpMsgDisplaysRequestedTopic(t *testing.T) {
	model := NewTUIModel(mockConfig(), nil, nil, nil, nil, nil)
	require.Equal(t, ViewChat, model.content.GetActiveView())

	newModel, _ := model.handleCustomMessages(showHelpMsg{topic: "modes"})
	updatedModel, ok := newModel.(TUIModel)
	require.True(t, ok)
	require.Equal(t, ViewHelp, updatedModel.content.GetActiveView())
	require.Equal(t, "modes", updatedModel.content.help.GetTopic())
}

// Tests from tui_history_test.go

// TestHistoryNavigation_EmptyHistory tests navigation with no history
func TestHistoryNavigation_EmptyHistory(t *testing.T) {
	model, _ := newTestModel(t)

	// Press up arrow with empty history
	handled, cmd := model.handleHistoryNavigation(-1)
	require.False(t, handled, "Should not handle navigation with empty history")
	require.Nil(t, cmd)

	// Press down arrow with empty history
	handled, cmd = model.handleHistoryNavigation(1)
	require.False(t, handled, "Should not handle navigation with empty history")
	require.Nil(t, cmd)
}

// TestHistoryNavigation_SingleEntry tests navigation with one history entry
func TestHistoryNavigation_SingleEntry(t *testing.T) {
	model, _ := newTestModel(t)

	// Add one history entry
	model.sessionPromptHistory = []promptHistoryEntry{
		{Prompt: "first prompt", SessionSnapshot: 1, ChatSnapshot: 0},
	}
	model.historyCursor = 1 // At present

	// Navigate up (to first entry)
	handled, cmd := model.handleHistoryNavigation(-1)
	require.True(t, handled)
	require.Nil(t, cmd)
	require.Equal(t, 0, model.historyCursor)
	require.Equal(t, "first prompt", model.prompt.Value())
	require.True(t, model.historySaved, "Should save present state")

	// Try to navigate up again (should stay at first entry)
	handled, cmd = model.handleHistoryNavigation(-1)
	require.True(t, handled)
	require.Nil(t, cmd)
	require.Equal(t, 0, model.historyCursor)

	// Navigate down (back to present)
	handled, cmd = model.handleHistoryNavigation(1)
	require.True(t, handled)
	require.Nil(t, cmd)
	require.Equal(t, 1, model.historyCursor)
	require.False(t, model.historySaved, "Should clear saved state when returning to present")
}

// TestHistoryNavigation_MultipleEntries tests navigation through multiple entries
func TestHistoryNavigation_MultipleEntries(t *testing.T) {
	model, _ := newTestModel(t)
	model.prompt.SetValue("current input")

	// Add multiple history entries
	model.sessionPromptHistory = []promptHistoryEntry{
		{Prompt: "first prompt", SessionSnapshot: 1, ChatSnapshot: 0},
		{Prompt: "second prompt", SessionSnapshot: 3, ChatSnapshot: 2},
		{Prompt: "third prompt", SessionSnapshot: 5, ChatSnapshot: 4},
	}
	model.historyCursor = 3 // At present

	// Navigate up once
	handled, cmd := model.handleHistoryNavigation(-1)
	require.True(t, handled)
	require.Nil(t, cmd)
	require.Equal(t, 2, model.historyCursor)
	require.Equal(t, "third prompt", model.prompt.Value())
	require.True(t, model.historySaved)
	require.Equal(t, "current input", model.historyPendingPrompt)

	// Navigate up again
	handled, cmd = model.handleHistoryNavigation(-1)
	require.True(t, handled)
	require.Nil(t, cmd)
	require.Equal(t, 1, model.historyCursor)
	require.Equal(t, "second prompt", model.prompt.Value())

	// Navigate up to first
	handled, cmd = model.handleHistoryNavigation(-1)
	require.True(t, handled)
	require.Nil(t, cmd)
	require.Equal(t, 0, model.historyCursor)
	require.Equal(t, "first prompt", model.prompt.Value())

	// Try to navigate up past first (should stay at first)
	handled, cmd = model.handleHistoryNavigation(-1)
	require.True(t, handled)
	require.Nil(t, cmd)
	require.Equal(t, 0, model.historyCursor)
	require.Equal(t, "first prompt", model.prompt.Value())

	// Navigate down
	handled, cmd = model.handleHistoryNavigation(1)
	require.True(t, handled)
	require.Nil(t, cmd)
	require.Equal(t, 1, model.historyCursor)
	require.Equal(t, "second prompt", model.prompt.Value())

	// Navigate down to third
	handled, cmd = model.handleHistoryNavigation(1)
	require.True(t, handled)
	require.Nil(t, cmd)
	require.Equal(t, 2, model.historyCursor)
	require.Equal(t, "third prompt", model.prompt.Value())

	// Navigate down to present
	handled, cmd = model.handleHistoryNavigation(1)
	require.True(t, handled)
	require.Nil(t, cmd)
	require.Equal(t, 3, model.historyCursor)
	require.Equal(t, "current input", model.prompt.Value())
	require.False(t, model.historySaved)
}

// TestHistoryNavigation_DownWithoutSavedState tests down navigation without saved state
func TestHistoryNavigation_DownWithoutSavedState(t *testing.T) {
	model, _ := newTestModel(t)

	model.sessionPromptHistory = []promptHistoryEntry{
		{Prompt: "first prompt", SessionSnapshot: 1, ChatSnapshot: 0},
	}
	model.historyCursor = 1 // At present
	model.historySaved = false

	// Try to navigate down when already at present
	handled, cmd := model.handleHistoryNavigation(1)
	require.False(t, handled, "Should not handle down when not in history")
	require.Nil(t, cmd)
}

// TestHistoryNavigation_CursorInitialization tests cursor initialization from present
func TestHistoryNavigation_CursorInitialization(t *testing.T) {
	model, _ := newTestModel(t)
	model.prompt.SetValue("current")

	model.sessionPromptHistory = []promptHistoryEntry{
		{Prompt: "first", SessionSnapshot: 1, ChatSnapshot: 0},
		{Prompt: "second", SessionSnapshot: 3, ChatSnapshot: 2},
	}
	model.historyCursor = len(model.sessionPromptHistory) // At present

	// First up navigation should go to last entry
	handled, cmd := model.handleHistoryNavigation(-1)
	require.True(t, handled)
	require.Nil(t, cmd)
	require.Equal(t, 1, model.historyCursor)
	require.Equal(t, "second", model.prompt.Value())
}

// TestWaitingIndicator_StartStop tests the waiting indicator lifecycle
func TestWaitingIndicator_StartStop(t *testing.T) {
	model, _ := newTestModel(t)

	// Initially not waiting
	require.False(t, model.waitingForResponse)
	require.True(t, model.waitingStart.IsZero())

	// Start waiting
	cmd := model.startWaitingForResponse()
	require.True(t, model.waitingForResponse)
	require.False(t, model.waitingStart.IsZero())
	require.NotNil(t, cmd, "Should return tick command")

	// Verify status component was updated
	require.True(t, model.status.waitingForResponse)

	// Stop waiting
	model.stopStreaming()
	require.False(t, model.waitingForResponse)
	require.False(t, model.status.waitingForResponse)
}

// TestWaitingIndicator_DoubleStart tests starting waiting when already waiting
func TestWaitingIndicator_DoubleStart(t *testing.T) {
	model, _ := newTestModel(t)

	// Start waiting
	cmd1 := model.startWaitingForResponse()
	require.NotNil(t, cmd1)
	startTime := model.waitingStart

	// Try to start again
	cmd2 := model.startWaitingForResponse()
	require.Nil(t, cmd2, "Should not return command when already waiting")
	require.Equal(t, startTime, model.waitingStart, "Start time should not change")
}

// TestWaitingIndicator_DoubleStop tests stopping when not waiting
func TestWaitingIndicator_DoubleStop(t *testing.T) {
	model, _ := newTestModel(t)

	// Stop when not waiting (should not panic)
	model.stopStreaming()
	require.False(t, model.waitingForResponse)
}

// TestWaitingTickMsg_WhileWaiting tests waiting tick message handling
func TestWaitingTickMsg_WhileWaiting(t *testing.T) {
	model, _ := newTestModel(t)

	// Start waiting
	model.startWaitingForResponse()

	// Handle tick message
	newModel, cmd := model.handleCustomMessages(waitingTickMsg{})
	updatedModel, ok := newModel.(TUIModel)
	require.True(t, ok)
	require.NotNil(t, cmd, "Should return next tick command")

	// Verify still waiting
	require.True(t, updatedModel.waitingForResponse)
}

// TestWaitingTickMsg_NotWaiting tests waiting tick when not waiting
func TestWaitingTickMsg_NotWaiting(t *testing.T) {
	model, _ := newTestModel(t)

	// Handle tick message when not waiting
	newModel, cmd := model.handleCustomMessages(waitingTickMsg{})
	updatedModel, ok := newModel.(TUIModel)
	require.True(t, ok)
	require.Nil(t, cmd, "Should not return tick command when not waiting")
	require.False(t, updatedModel.waitingForResponse)
}

// TestHistoryRollback_OnSubmit tests that submitting a historical prompt rolls back state
func TestHistoryRollback_OnSubmit(t *testing.T) {
	model, _ := newTestModel(t)
	chat := model.content.GetChat()

	// Clear the welcome message for cleaner testing
	chat.Messages = []string{}
	chat.UpdateContent()

	// Simulate a conversation
	chat.AddMessage("You: first")
	chat.AddMessage("Asimi: response1")
	model.sessionPromptHistory = append(model.sessionPromptHistory, promptHistoryEntry{
		Prompt:          "first",
		SessionSnapshot: 1,
		ChatSnapshot:    0, // Before adding messages
	})

	chat.AddMessage("You: second")
	chat.AddMessage("Asimi: response2")
	model.sessionPromptHistory = append(model.sessionPromptHistory, promptHistoryEntry{
		Prompt:          "second",
		SessionSnapshot: 1, // Session hasn't changed (no actual LLM calls)
		ChatSnapshot:    2, // After first conversation
	})

	model.historyCursor = len(model.sessionPromptHistory)

	// Navigate to first prompt
	model.handleHistoryNavigation(-1) // to "second"
	model.handleHistoryNavigation(-1) // to "first"

	require.Equal(t, 0, model.historyCursor)
	require.Equal(t, "first", model.prompt.Value())
	require.True(t, model.historySaved)

	// Simulate submitting the historical prompt
	chatLenBefore := len(chat.Messages)
	sessionLenBefore := len(model.session.Messages)

	// The handleEnterKey function should detect historySaved and roll back
	// We'll test the rollback logic directly
	if model.historySaved && model.historyCursor < len(model.sessionPromptHistory) {
		entry := model.sessionPromptHistory[model.historyCursor]
		model.session.RollbackTo(entry.SessionSnapshot)
		chat.TruncateTo(entry.ChatSnapshot)
	}

	// Verify rollback occurred
	require.Equal(t, 1, len(model.session.Messages), "Session should be rolled back to system message")
	require.Equal(t, 0, len(chat.Messages), "Chat should be rolled back to empty")
	require.Less(t, len(chat.Messages), chatLenBefore)
	require.Equal(t, len(model.session.Messages), sessionLenBefore) // Session didn't change in this test
}

// TestNewSessionCommand_ResetsHistory tests that /new command resets history
func TestNewSessionCommand_ResetsHistory(t *testing.T) {
	model, _ := newTestModel(t)

	// Add some history
	model.sessionPromptHistory = []promptHistoryEntry{
		{Prompt: "first", SessionSnapshot: 1, ChatSnapshot: 0},
		{Prompt: "second", SessionSnapshot: 3, ChatSnapshot: 2},
	}
	model.historyCursor = 1
	model.historySaved = true
	model.historyPendingPrompt = "pending"

	// Start waiting
	model.startWaitingForResponse()
	require.True(t, model.waitingForResponse)

	// Execute /new command
	cmd := handleNewSessionCommand(model, []string{})

	// Process the returned message
	msg := cmd()
	startMsg, ok := msg.(startConversationMsg)
	require.True(t, ok, "Expected startConversationMsg")
	require.True(t, startMsg.clearHistory)

	// Simulate the message being processed by Update
	updatedModel, _ := model.Update(startMsg)
	updatedModelValue, ok := updatedModel.(TUIModel)
	require.True(t, ok, "Expected TUIModel")

	// Verify history was reset
	require.Empty(t, updatedModelValue.sessionPromptHistory)
	require.Equal(t, 0, updatedModelValue.historyCursor)
	require.False(t, updatedModelValue.historySaved)
	require.Empty(t, updatedModelValue.historyPendingPrompt)
	require.False(t, updatedModelValue.waitingForResponse)
}

// TestHistoryNavigation_WithArrowKeys tests arrow key handling
func TestHistoryNavigation_WithArrowKeys(t *testing.T) {
	model, _ := newTestModel(t)

	// Add history
	model.sessionPromptHistory = []promptHistoryEntry{
		{Prompt: "first", SessionSnapshot: 1, ChatSnapshot: 0},
		{Prompt: "second", SessionSnapshot: 3, ChatSnapshot: 2},
	}
	model.historyCursor = 2
	model.prompt.SetValue("current")

	// Simulate up arrow on first line (cursor at start)
	model.prompt.TextArea.CursorStart()
	newModel, cmd := model.handleKeyMsg(tea.KeyMsg{Type: tea.KeyUp})
	updatedModel, ok := newModel.(TUIModel)
	require.True(t, ok)
	require.Nil(t, cmd)
	require.Equal(t, 1, updatedModel.historyCursor)
	require.Equal(t, "second", updatedModel.prompt.Value())

	// Simulate down arrow on last line (cursor at end)
	updatedModel.prompt.TextArea.CursorEnd()
	newModel, cmd = updatedModel.handleKeyMsg(tea.KeyMsg{Type: tea.KeyDown})
	updatedModel, ok = newModel.(TUIModel)
	require.True(t, ok)
	require.Nil(t, cmd)
	require.Equal(t, 2, updatedModel.historyCursor)
	require.Equal(t, "current", updatedModel.prompt.Value())
}

// TestCancelActiveStreaming tests the streaming cancellation helper
func TestCancelActiveStreaming(t *testing.T) {
	model, _ := newTestModel(t)

	// Set up active streaming
	model.streamingActive = true
	cancelCalled := false
	model.streamingCancel = func() {
		cancelCalled = true
	}

	// Cancel streaming
	model.cancelStreaming()

	require.True(t, cancelCalled, "Cancel function should be called")
	require.False(t, model.streamingActive)
	require.Nil(t, model.streamingCancel)
}

// TestCancelActiveStreaming_NotActive tests cancellation when not streaming
func TestCancelActiveStreaming_NotActive(t *testing.T) {
	model, _ := newTestModel(t)

	// Not streaming
	model.streamingActive = false
	model.streamingCancel = nil

	// Should not panic
	model.cancelStreaming()

	require.False(t, model.streamingActive)
	require.Nil(t, model.streamingCancel)
}

// TestSaveHistoryPresentState tests saving the present state
func TestSaveHistoryPresentState(t *testing.T) {
	model, _ := newTestModel(t)
	model.prompt.SetValue("current prompt")
	chat := model.content.GetChat()
	chat.AddMessage("message 1")
	chat.AddMessage("message 2")

	// Save present state
	model.saveHistoryPresentState()

	require.True(t, model.historySaved)
	require.Equal(t, "current prompt", model.historyPendingPrompt)
	// Chat has welcome message + 2 added messages = 3 total
	require.Equal(t, 3, model.historyPresentChatSnapshot)
	require.Equal(t, 1, model.historyPresentSessionSnapshot) // System message only

	// Try to save again (should not change)
	model.prompt.SetValue("different")
	model.saveHistoryPresentState()
	require.Equal(t, "current prompt", model.historyPendingPrompt, "Should not update when already saved")
}

// TestRestoreHistoryPresent tests restoring the present state
func TestRestoreHistoryPresent(t *testing.T) {
	model, _ := newTestModel(t)
	model.prompt.SetValue("current")
	model.historyPendingPrompt = "pending"
	model.historySaved = true

	// Restore present
	model.restoreHistoryPresent()

	require.Equal(t, "pending", model.prompt.Value())
	require.False(t, model.historySaved)
}

// TestApplyHistoryEntry tests applying a history entry
func TestApplyHistoryEntry(t *testing.T) {
	model, _ := newTestModel(t)
	model.prompt.SetValue("current")

	entry := promptHistoryEntry{
		Prompt:          "historical prompt",
		SessionSnapshot: 5,
		ChatSnapshot:    3,
	}

	// Apply entry
	model.applyHistoryEntry(entry)

	require.Equal(t, "historical prompt", model.prompt.Value())
}

// TestStatusComponent_WaitingIndicator tests the status component waiting indicator
func TestStatusComponent_WaitingIndicator(t *testing.T) {
	status := NewStatusComponent(80)

	// Initially not waiting
	require.False(t, status.waitingForResponse)

	// Start waiting
	status.StartWaiting()
	require.True(t, status.waitingForResponse)
	require.False(t, status.waitingSince.IsZero())

	// Stop waiting
	status.StopWaiting()
	require.False(t, status.waitingForResponse)
}

// TestStatusComponent_WaitingIndicatorView tests the waiting indicator in the view
func TestStatusComponent_WaitingIndicatorView(t *testing.T) {
	status := NewStatusComponent(200) // Use very wide width to avoid truncation
	status.SetProvider("test", "model", true)

	// Create a mock session to provide usage data
	llm := &mockLLMNoTools{}
	repoInfo := RepoInfo{}
	sess, err := NewSession(llm, &Config{}, repoInfo, func(any) {})
	require.NoError(t, err)
	status.SetSession(sess)

	// View without waiting
	middleSection := status.renderMiddleSection()
	require.NotContains(t, middleSection, "⏳")

	// Start waiting (less than 3 seconds ago)
	status.StartWaiting()
	status.waitingSince = time.Now().Add(-2 * time.Second)
	middleSection = status.renderMiddleSection()
	require.NotContains(t, middleSection, "⏳", "Should not show indicator before 3 seconds")

	// Waiting for more than 3 seconds - check middle section directly
	status.StartWaiting()
	status.waitingSince = time.Now().Add(-5 * time.Second)
	middleSection = status.renderMiddleSection()
	require.Contains(t, middleSection, "⏳", "Middle section should contain waiting indicator")
	require.Contains(t, middleSection, "5s", "Middle section should show elapsed time")
}

// TestEscapeDuringStreaming_StopsWaiting tests that ESC during streaming stops waiting
func TestEscapeDuringStreaming_StopsWaiting(t *testing.T) {
	model, _ := newTestModel(t)

	// Set up streaming
	model.streamingActive = true
	cancelCalled := false
	model.streamingCancel = func() {
		cancelCalled = true
	}

	// Start waiting
	model.startWaitingForResponse()
	require.True(t, model.waitingForResponse)

	// Press escape
	newModel, _ := model.handleEscape()
	updatedModel, ok := newModel.(TUIModel)
	require.True(t, ok)

	require.True(t, cancelCalled)
	require.False(t, updatedModel.waitingForResponse)
}

// TestStreamChunkMsg_StopsWaiting tests that receiving a stream chunk resets the quiet time timer
func TestStreamChunkMsg_StopsWaiting(t *testing.T) {
	model, _ := newTestModel(t)

	// Start waiting and mark as streaming
	model.startWaitingForResponse()
	model.streamingActive = true
	require.True(t, model.waitingForResponse)

	// Record the initial wait start time
	initialWaitStart := model.waitingStart

	// Wait a bit to ensure time passes
	time.Sleep(10 * time.Millisecond)

	// Receive stream chunk - should reset the waiting timer
	newModel, _ := model.handleCustomMessages(streamChunkMsg("chunk"))
	updatedModel, ok := newModel.(TUIModel)
	require.True(t, ok)

	// Waiting should still be active (for tracking quiet time)
	require.True(t, updatedModel.waitingForResponse)
	// But the timer should have been reset (waitingStart should be newer)
	require.True(t, updatedModel.waitingStart.After(initialWaitStart), "Waiting timer should be reset when chunk arrives")
}

// TestStreamCompleteMsg_StopsWaiting tests that stream completion stops waiting
func TestStreamCompleteMsg_StopsWaiting(t *testing.T) {
	model, _ := newTestModel(t)

	// Start waiting
	model.startWaitingForResponse()
	require.True(t, model.waitingForResponse)

	// Stream completes
	newModel, _ := model.handleCustomMessages(streamCompleteMsg{})
	updatedModel, ok := newModel.(TUIModel)
	require.True(t, ok)

	require.False(t, updatedModel.waitingForResponse)
}

// TestStreamErrorMsg_StopsWaiting tests that stream error stops waiting
func TestStreamErrorMsg_StopsWaiting(t *testing.T) {
	model, _ := newTestModel(t)

	// Start waiting
	model.startWaitingForResponse()
	require.True(t, model.waitingForResponse)

	// Stream error
	testErr := errors.New("test error")
	newModel, _ := model.handleCustomMessages(streamErrorMsg{err: testErr})
	updatedModel, ok := newModel.(TUIModel)
	require.True(t, ok)

	require.False(t, updatedModel.waitingForResponse)
}

// TestHistoryNavigation_RapidNavigation tests rapid navigation through history
func TestHistoryNavigation_RapidNavigation(t *testing.T) {
	model, _ := newTestModel(t)

	// Add many history entries
	for i := 0; i < 10; i++ {
		model.sessionPromptHistory = append(model.sessionPromptHistory, promptHistoryEntry{
			Prompt:          "prompt " + string(rune('0'+i)),
			SessionSnapshot: i*2 + 1,
			ChatSnapshot:    i * 2,
		})
	}
	model.historyCursor = len(model.sessionPromptHistory)
	model.prompt.SetValue("current")

	// Rapidly navigate up
	for i := 0; i < 10; i++ {
		model.handleHistoryNavigation(-1)
	}
	require.Equal(t, 0, model.historyCursor)
	require.Equal(t, "prompt 0", model.prompt.Value())

	// Rapidly navigate down
	for i := 0; i < 10; i++ {
		model.handleHistoryNavigation(1)
	}
	require.Equal(t, 10, model.historyCursor)
	require.Equal(t, "current", model.prompt.Value())
	require.False(t, model.historySaved)
}

// Tests from tui_e2e_test.go

func TestFileCompletion(t *testing.T) {
	// Create a new TUI model for testing
	config := mockConfig()
	model := NewTUIModel(config, nil, nil, nil, nil, nil)

	// Set up a mock session for the test
	llm := fake.NewFakeLLM([]string{})
	sess, err := NewSession(llm, &Config{LLM: LLMConfig{Provider: "fake"}}, RepoInfo{}, func(any) {})
	require.NoError(t, err)
	model.SetSession(sess)

	// Create a new test model
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(200, 200))

	// Get file list and find the inex of main.go
	files, err := getFileTree(".")
	require.NoError(t, err)
	mainGoIndex := -1
	for i, f := range files {
		if f == "main.go" {
			mainGoIndex = i
			break
		}
	}
	require.NotEqual(t, -1, mainGoIndex, "main.go not found in file tree")

	// Simulate typing "@"
	tm.Type("@main.")

	// Wait for the completion dialog to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "main.go")
	}, teatest.WithCheckInterval(time.Millisecond*100), teatest.WithDuration(time.Second*3))

	// Simulate pressing enter to select the file
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for a bit to let the file be read
	time.Sleep(100 * time.Millisecond)

	// Quit the application (requires double CTRL-C)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	time.Sleep(ctrlCDebounceTime + 10*time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	// Get the final model
	finalModel := tm.FinalModel(t)
	tuiModel, ok := finalModel.(TUIModel)
	require.True(t, ok)

	// Assert that the prompt was completed to @main.go
	require.Equal(t, "@main.go ", tuiModel.prompt.TextArea.Value())

	// Assert that the file viewer contains the file content
	contextFiles := tuiModel.session.GetContextFiles()
	require.Contains(t, contextFiles["main.go"], "package main")

	// Assert that the prompt was not sent and the editor is still focused
	chat := tuiModel.content.GetChat()
	require.NotEmpty(t, chat.Messages)
	require.True(t, containsMessage(chat.Messages, "Loaded file: main.go"),
		"messages", chat.Messages)
	require.True(t, tuiModel.prompt.TextArea.Focused(), "The editor should remain focused")
}

func TestColonCommandCompletionE2E(t *testing.T) {
	// Create a new TUI model for testing
	config := mockConfig()
	model := NewTUIModel(config, nil, nil, nil, nil, nil)

	// Set up a mock session for the test
	llm := fake.NewFakeLLM([]string{})
	sess, err := NewSession(llm, &Config{LLM: LLMConfig{Provider: "fake"}}, RepoInfo{}, func(any) {})
	require.NoError(t, err)
	model.SetSession(sess)

	// Create a new test model
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(200, 200))

	// Simulate typing ":" to enter command mode
	tm.Type(":")

	// Wait for the command line to show ":"
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), ":")
	}, teatest.WithCheckInterval(time.Millisecond*100), teatest.WithDuration(time.Second*3))

	// Type "help" command
	tm.Type("help")

	// Press enter to execute the command
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for a bit to let the command be executed
	time.Sleep(100 * time.Millisecond)

	// Quit the application (requires double CTRL-C)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	time.Sleep(ctrlCDebounceTime + 10*time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	// Get the final model
	finalModel := tm.FinalModel(t)
	tuiModel, ok := finalModel.(TUIModel)
	require.True(t, ok)

	require.Equal(t, ViewHelp, tuiModel.content.GetActiveView())
	require.Equal(t, "index", tuiModel.content.help.GetTopic())
}

func TestLiveAgentE2E(t *testing.T) {
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping live agent test")
	}

	t.Skip("E2E test is skipped for now")
	cmd := exec.Command("go", "run", ".", "-p", "who are you?")
	output, err := cmd.CombinedOutput()
	// In case of error, report the output
	require.NoError(t, err, "output", string(output))

	// Assert the output
	require.Contains(t, string(output), "I am ")
	require.NotContains(t, string(output), "Error")
}
