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

// TestTUIModelInit tests the initialization of the TUI model
func TestTUIModelInit(t *testing.T) {
	model := NewTUIModel(mockConfig(), nil, nil, nil)
	cmd := model.Init()

	// Init should return nil as there's no initial command
	require.Nil(t, cmd)
}

// TestTUIModelWindowSizeMsg tests handling of window size messages
func TestTUIModelWindowSizeMsg(t *testing.T) {
	model := NewTUIModel(mockConfig(), nil, nil, nil)

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
	model := NewTUIModel(mockConfig(), nil, nil, nil)
	// Disable persistent history to keep tests hermetic.
	model.historyStore = nil
	model.initHistory()
	// Use native session path for tests now that legacy agent is removed.
	repoInfo := GetRepoInfo()
	sess, err := NewSession(llm, &Config{LLM: LLMConfig{Provider: "fake"}}, repoInfo, func(any) {})
	require.NoError(t, err)
	model.SetSession(sess)
	return model, llm
}

func TestCommandCompletionOrderDefaultsToHelp(t *testing.T) {
	model := NewTUIModel(mockConfig(), nil, nil, nil)
	model.prompt.SetValue("/")
	model.completionMode = "command"
	model.updateCommandCompletions()
	require.NotEmpty(t, model.completions.Options)
	require.Equal(t, "/help", model.completions.Options[0])
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
			model := NewTUIModel(mockConfig(), nil, nil, nil)

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
	model := NewTUIModel(mockConfig(), nil, nil, nil)

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

			require.Equal(t, tc.expectedMessageCount, len(model.chat.Messages))
			require.Contains(t, model.chat.Messages[len(model.chat.Messages)-1], tc.expectedLastMessage, "prompt", tc.name)
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
				// Disable vi mode so escape clears the modal
				model.prompt.SetViMode(false)
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
				model.completions.SetOptions([]string{"/help", "option2", "option3"})
				model.completions.Show()
			},
			verify: func(t *testing.T, model *TUIModel, cmd tea.Cmd) {
				require.NotNil(t, cmd)
				msg := cmd()
				newModel, cmd := model.Update(msg)
				require.Nil(t, cmd)
				updatedModel, ok := newModel.(TUIModel)
				require.True(t, ok)
				require.Contains(t, updatedModel.chat.Messages[len(updatedModel.chat.Messages)-1], "Available commands:")
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

			tc.verify(t, &updatedModel, cmd)
		})
	}
}

// TestTUIModelView tests the view rendering
func TestTUIModelView(t *testing.T) {
	model := NewTUIModel(mockConfig(), nil, nil, nil)

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
	chat := NewChatComponent(50, 10)

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

// TestToastManager tests the toast manager
func TestToastManager(t *testing.T) {
	toastManager := NewToastManager()

	// Initially should have no toasts
	require.Empty(t, toastManager.Toasts)

	// Test adding a toast
	message := "Test toast message"
	tostType := "info"
	timeout := 5 * time.Second

	toastManager.AddToast(message, tostType, timeout)
	require.Equal(t, 1, len(toastManager.Toasts))

	// Test view rendering
	view := toastManager.View()
	require.NotEmpty(t, view)
	require.Contains(t, view, message)

	// Test clearing toasts
	toastManager.Clear()
	require.Empty(t, toastManager.Toasts)

	// Re-add toast to verify removal still works
	toastManager.AddToast(message, tostType, timeout)
	require.Equal(t, 1, len(toastManager.Toasts))

	// Test removing a toast
	toastID := toastManager.Toasts[0].ID
	toastManager.RemoveToast(toastID)
	require.Empty(t, toastManager.Toasts)

	// Test updating (removing expired toasts)
	toastManager.AddToast(message, tostType, 1*time.Millisecond)
	time.Sleep(2 * time.Millisecond) // Wait for toast to expire
	toastManager.Update()
	require.Empty(t, toastManager.Toasts)
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
	model := NewTUIModel(mockConfig(), nil, nil, nil)
	model.width = 80
	model.height = 24

	// Test regular input mode (vi mode disabled)
	model.prompt.SetViMode(false)
	view := model.renderHomeView(80, 24)
	require.NotEmpty(t, view)
	require.Contains(t, view, "Asimi CLI - Interactive Coding Agent")
	require.Contains(t, view, "Your AI-powered coding assistant")
	require.Contains(t, view, "Use / to access commands")
	require.Contains(t, view, "Use /vi to enable vi mode")
	require.NotContains(t, view, "Vi mode is enabled")

	// Test vi mode enabled
	model.prompt.SetViMode(true)
	view = model.renderHomeView(80, 24)
	require.NotEmpty(t, view)
	require.Contains(t, view, "Asimi CLI - Interactive Coding Agent")
	require.Contains(t, view, "Your AI-powered coding assistant")
	require.Contains(t, view, "Vi mode is enabled")
	require.Contains(t, view, "Press Esc to enter NORMAL mode")
	require.Contains(t, view, "In NORMAL mode, press : to enter COMMAND-LINE mode")
	require.NotContains(t, view, "Use / to access commands")
}

// TestColonCommandCompletion tests command completion with colon prefix in vi mode
func TestColonCommandCompletion(t *testing.T) {
	model, _ := newTestModel(t)
	model.prompt.SetViMode(true)

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

	// Test that slash commands still work
	model.prompt.SetValue("/he")
	model.updateCommandCompletions()
	require.NotEmpty(t, model.completions.Options)
	require.Contains(t, model.completions.Options, "/help")

	// Test that slash commands show with slash prefix
	model.prompt.SetValue("/")
	model.updateCommandCompletions()
	require.NotEmpty(t, model.completions.Options)
	for _, opt := range model.completions.Options {
		require.True(t, strings.HasPrefix(opt, "/"), "Command should start with / but got: %s", opt)
	}
}

// TestColonInNormalModeShowsCompletion tests that pressing : in normal mode shows completion dialog
func TestColonInNormalModeShowsCompletion(t *testing.T) {
	model, _ := newTestModel(t)
	model.prompt.SetViMode(true)

	// Start in insert mode
	require.True(t, model.prompt.IsViInsertMode())

	// Press Esc to enter normal mode
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedModel := newModel.(TUIModel)
	require.True(t, updatedModel.prompt.IsViNormalMode())

	// Press : to enter command-line mode
	newModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	updatedModel = newModel.(TUIModel)

	// Should be in command-line mode
	require.True(t, updatedModel.prompt.IsViCommandLineMode())

	// Completion dialog should be shown
	require.True(t, updatedModel.showCompletionDialog)
	require.Equal(t, "command", updatedModel.completionMode)
	require.True(t, updatedModel.completions.Visible)

	// Completions should have : prefix
	require.NotEmpty(t, updatedModel.completions.Options)
	for _, opt := range updatedModel.completions.Options {
		require.True(t, strings.HasPrefix(opt, ":"), "Command should start with : but got: %s", opt)
	}

	// Prompt value should be ":"
	require.Equal(t, ":", updatedModel.prompt.Value())
}

func TestShowHelpMsgUsesActiveLeader(t *testing.T) {
	model := TUIModel{
		commandRegistry: NewCommandRegistry(),
		chat:            NewChatComponent(80, 20),
	}

	withColon, _ := model.handleCustomMessages(showHelpMsg{leader: ":"})
	colonModel, ok := withColon.(TUIModel)
	require.True(t, ok)
	require.NotEmpty(t, colonModel.chat.Messages)
	colonHelp := colonModel.chat.Messages[len(colonModel.chat.Messages)-1]
	require.Contains(t, colonHelp, "Active command leader: :")
	require.Contains(t, colonHelp, ":help - Show help information")

	withSlash, _ := colonModel.handleCustomMessages(showHelpMsg{leader: "/"})
	slashModel, ok := withSlash.(TUIModel)
	require.True(t, ok)
	require.NotEmpty(t, slashModel.chat.Messages)
	slashHelp := slashModel.chat.Messages[len(slashModel.chat.Messages)-1]
	require.Contains(t, slashHelp, "Active command leader: /")
	require.Contains(t, slashHelp, "/help - Show help information")
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
	model.promptHistory = []promptHistoryEntry{
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
	model.promptHistory = []promptHistoryEntry{
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

	model.promptHistory = []promptHistoryEntry{
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

	model.promptHistory = []promptHistoryEntry{
		{Prompt: "first", SessionSnapshot: 1, ChatSnapshot: 0},
		{Prompt: "second", SessionSnapshot: 3, ChatSnapshot: 2},
	}
	model.historyCursor = len(model.promptHistory) // At present

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

	// Clear the welcome message for cleaner testing
	model.chat.Messages = []string{}
	model.chat.UpdateContent()

	// Simulate a conversation
	model.chat.AddMessage("You: first")
	model.chat.AddMessage("Asimi: response1")
	model.promptHistory = append(model.promptHistory, promptHistoryEntry{
		Prompt:          "first",
		SessionSnapshot: 1,
		ChatSnapshot:    0, // Before adding messages
	})

	model.chat.AddMessage("You: second")
	model.chat.AddMessage("Asimi: response2")
	model.promptHistory = append(model.promptHistory, promptHistoryEntry{
		Prompt:          "second",
		SessionSnapshot: 1, // Session hasn't changed (no actual LLM calls)
		ChatSnapshot:    2, // After first conversation
	})

	model.historyCursor = len(model.promptHistory)

	// Navigate to first prompt
	model.handleHistoryNavigation(-1) // to "second"
	model.handleHistoryNavigation(-1) // to "first"

	require.Equal(t, 0, model.historyCursor)
	require.Equal(t, "first", model.prompt.Value())
	require.True(t, model.historySaved)

	// Simulate submitting the historical prompt
	chatLenBefore := len(model.chat.Messages)
	sessionLenBefore := len(model.session.Messages)

	// The handleEnterKey function should detect historySaved and roll back
	// We'll test the rollback logic directly
	if model.historySaved && model.historyCursor < len(model.promptHistory) {
		entry := model.promptHistory[model.historyCursor]
		model.session.RollbackTo(entry.SessionSnapshot)
		model.chat.TruncateTo(entry.ChatSnapshot)
	}

	// Verify rollback occurred
	require.Equal(t, 1, len(model.session.Messages), "Session should be rolled back to system message")
	require.Equal(t, 0, len(model.chat.Messages), "Chat should be rolled back to empty")
	require.Less(t, len(model.chat.Messages), chatLenBefore)
	require.Equal(t, len(model.session.Messages), sessionLenBefore) // Session didn't change in this test
}

// TestNewSessionCommand_ResetsHistory tests that /new command resets history
func TestNewSessionCommand_ResetsHistory(t *testing.T) {
	model, _ := newTestModel(t)

	// Add some history
	model.promptHistory = []promptHistoryEntry{
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
	handleNewSessionCommand(model, []string{})

	// Verify history was reset
	require.Empty(t, model.promptHistory)
	require.Equal(t, 0, model.historyCursor)
	require.False(t, model.historySaved)
	require.Empty(t, model.historyPendingPrompt)
	require.False(t, model.waitingForResponse)
}

// TestHistoryNavigation_WithArrowKeys tests arrow key handling
func TestHistoryNavigation_WithArrowKeys(t *testing.T) {
	model, _ := newTestModel(t)

	// Add history
	model.promptHistory = []promptHistoryEntry{
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
	model.chat.AddMessage("message 1")
	model.chat.AddMessage("message 2")

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
	repoInfo := GetRepoInfo()
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
		model.promptHistory = append(model.promptHistory, promptHistoryEntry{
			Prompt:          "prompt " + string(rune('0'+i)),
			SessionSnapshot: i*2 + 1,
			ChatSnapshot:    i * 2,
		})
	}
	model.historyCursor = len(model.promptHistory)
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

// Tests from tui_vi_config_test.go

func TestTUIRespectsViModeConfig(t *testing.T) {
	tests := []struct {
		name           string
		viMode         *bool
		expectedViMode bool
	}{
		{
			name:           "default config enables vi mode",
			viMode:         nil,
			expectedViMode: true,
		},
		{
			name:           "explicitly enabled",
			viMode:         boolPtr(true),
			expectedViMode: true,
		},
		{
			name:           "explicitly disabled",
			viMode:         boolPtr(false),
			expectedViMode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				LLM: LLMConfig{
					ViMode: tt.viMode,
				},
			}

			model := NewTUIModel(config, nil, nil, nil)

			if model.prompt.ViMode != tt.expectedViMode {
				t.Errorf("TUI prompt.ViMode = %v, want %v", model.prompt.ViMode, tt.expectedViMode)
			}
		})
	}
}

// Tests from tui_e2e_test.go

func TestFileCompletion(t *testing.T) {
	// Create a new TUI model for testing
	config := mockConfig()
	model := NewTUIModel(config, nil, nil, nil)

	// Set up a mock session for the test
	llm := fake.NewFakeLLM([]string{})
	repoInfo := GetRepoInfo()
	sess, err := NewSession(llm, &Config{LLM: LLMConfig{Provider: "fake"}}, repoInfo, func(any) {})
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
	require.NotEmpty(t, tuiModel.chat.Messages)
	require.Contains(t, tuiModel.chat.Messages[len(tuiModel.chat.Messages)-1], "Loaded file: main.go")
	require.True(t, tuiModel.prompt.TextArea.Focused(), "The editor should remain focused")
}

func TestSlashCommandCompletion(t *testing.T) {
	// Create a new TUI model for testing
	config := mockConfig()
	model := NewTUIModel(config, nil, nil, nil)

	// Set up a mock session for the test
	llm := fake.NewFakeLLM([]string{})
	repoInfo := GetRepoInfo()
	sess, err := NewSession(llm, &Config{LLM: LLMConfig{Provider: "fake"}}, repoInfo, func(any) {})
	require.NoError(t, err)
	model.SetSession(sess)

	// Create a new test model
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(200, 200))

	// Simulate typing "/"
	tm.Type("/")

	// Wait for the completion dialog to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "/help")
	}, teatest.WithCheckInterval(time.Millisecond*100), teatest.WithDuration(time.Second*3))

	// Simulate pressing enter to select the first command
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

	// Assert that the messages contain the help text
	require.Contains(t, tuiModel.chat.Messages[len(tuiModel.chat.Messages)-1], "Available commands:")
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
