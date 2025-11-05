package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms/fake"
)

func TestFileCompletion(t *testing.T) {
	// Create a new TUI model for testing
	config := mockConfig()
	model := NewTUIModel(config)

	// Set up a mock session for the test
	llm := fake.NewFakeLLM([]string{})
	sess, err := NewSession(llm, &Config{LLM: LLMConfig{Provider: "fake"}}, func(any) {})
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
	//TODO: verify the application is down or final mode will hang if the "10" above is too short

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
	foundLoadedFileMsg := false
	for _, msg := range tuiModel.chat.Messages {
		if strings.Contains(msg, "Loaded file: main.go") {
			foundLoadedFileMsg = true
			break
		}
	}
	require.True(t, foundLoadedFileMsg, "messages=%+v", tuiModel.chat.Messages)
	require.True(t, tuiModel.prompt.TextArea.Focused(), "The editor should remain focused")
}

func TestSlashCommandCompletion(t *testing.T) {
	// Create a new TUI model for testing
	config := mockConfig()
	model := NewTUIModel(config)

	// Set up a mock session for the test
	llm := fake.NewFakeLLM([]string{})
	sess, err := NewSession(llm, &Config{LLM: LLMConfig{Provider: "fake"}}, func(any) {})
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
	//TODO: verify the application is down or final mode will hang if the "10" above is too short
	finalModel := tm.FinalModel(t)
	tuiModel, ok := finalModel.(TUIModel)
	require.True(t, ok)

	// Assert that the messages contain the help text
	foundHelpMsg := false
	for _, msg := range tuiModel.chat.Messages {
		if strings.Contains(msg, "Available commands:") {
			foundHelpMsg = true
			break
		}
	}
	require.True(t, foundHelpMsg, "messages=%+v", tuiModel.chat.Messages)
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
