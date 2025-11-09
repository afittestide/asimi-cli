package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

//go:embed prompts/initialize.txt
var initializePrompt string

// Command represents a slash command
type Command struct {
	Name        string
	Description string
	Handler     func(*TUIModel, []string) tea.Cmd
}

// CommandRegistry holds all available commands
type CommandRegistry struct {
	Commands map[string]Command
	order    []string
}

func normalizeCommandName(name string) string {
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, ":") {
		return "/" + strings.TrimPrefix(name, ":")
	}
	return name
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() CommandRegistry {
	registry := CommandRegistry{
		Commands: make(map[string]Command),
	}

	// Register built-in commands
	registry.RegisterCommand("/help", "Show help (usage: :help [topic])", handleHelpCommand)
	registry.RegisterCommand("/new", "Start a new session", handleNewSessionCommand)
	registry.RegisterCommand("/quit", "Quit the application", handleQuitCommand)
	registry.RegisterCommand("/login", "Login with OAuth provider selection", handleLoginCommand)
	registry.RegisterCommand("/models", "Select AI model", handleModelsCommand)
	registry.RegisterCommand("/context", "Show context usage details", handleContextCommand)
	registry.RegisterCommand("/clear-history", "Clear all prompt history", handleClearHistoryCommand)
	registry.RegisterCommand("/resume", "Resume a previous session", handleResumeCommand)
	registry.RegisterCommand("/export", "Export conversation to file and open in $EDITOR (usage: /export [full|conversation])", handleExportCommand)
	registry.RegisterCommand("/init", "Initialize project with missing infrastructure files (AGENTS.md, Justfile, Dockerfile)", handleInitCommand)

	return registry
}

// RegisterCommand registers a new command
func (cr *CommandRegistry) RegisterCommand(name, description string, handler func(*TUIModel, []string) tea.Cmd) {
	normalized := normalizeCommandName(name)
	if normalized == "" {
		return
	}
	if _, exists := cr.Commands[normalized]; !exists {
		cr.order = append(cr.order, normalized)
	}
	cr.Commands[normalized] = Command{
		Name:        normalized,
		Description: description,
		Handler:     handler,
	}
}

// GetCommand gets a command by name
func (cr CommandRegistry) GetCommand(name string) (Command, bool) {
	normalized := normalizeCommandName(name)
	cmd, exists := cr.Commands[normalized]
	return cmd, exists
}

// FindCommand finds commands by prefix (like vim).
// Returns:
// - exactMatch: the matched command if exactly one match is found
// - matches: all commands that start with the prefix
// - found: true if exactly one match was found
func (cr CommandRegistry) FindCommand(prefix string) (exactMatch Command, matches []string, found bool) {
	normalized := normalizeCommandName(prefix)
	if normalized == "" {
		return Command{}, nil, false
	}

	// First try exact match
	if cmd, exists := cr.Commands[normalized]; exists {
		return cmd, []string{normalized}, true
	}

	// Try prefix matching
	var matchedCommands []string
	searchPrefix := strings.TrimPrefix(normalized, "/")

	for _, cmdName := range cr.order {
		cmdNameWithoutSlash := strings.TrimPrefix(cmdName, "/")
		if strings.HasPrefix(cmdNameWithoutSlash, searchPrefix) {
			matchedCommands = append(matchedCommands, cmdName)
		}
	}

	if len(matchedCommands) == 1 {
		cmd := cr.Commands[matchedCommands[0]]
		return cmd, matchedCommands, true
	}

	return Command{}, matchedCommands, false
}

// GetAllCommands returns all registered commands
func (cr CommandRegistry) GetAllCommands() []Command {
	var commands []Command
	for _, name := range cr.order {
		if cmd, ok := cr.Commands[name]; ok {
			commands = append(commands, cmd)
		}
	}
	return commands
}

// Command handlers

type showHelpMsg struct {
	topic string
}
type showContextMsg struct{ content string }

func handleHelpCommand(model *TUIModel, args []string) tea.Cmd {
	// Determine the help topic from args
	topic := "index" // Default topic
	if len(args) > 0 {
		topic = args[0]
	}

	return func() tea.Msg {
		return showHelpMsg{topic: topic}
	}
}

func handleNewSessionCommand(model *TUIModel, args []string) tea.Cmd {
	model.saveSession()
	model.sessionActive = true
	chat := model.content.GetChat()
	*chat = NewChatComponent(chat.Width, chat.Height)

	model.rawSessionHistory = make([]string, 0)

	model.toolCallMessageIndex = make(map[string]int)

	// Reset prompt history and waiting state
	model.initHistory()
	model.cancelStreaming()
	model.stopStreaming()

	// If we have an active session, reset its conversation history
	if model.session != nil {
		model.session.ClearHistory()
	}
	return nil
}

func handleQuitCommand(model *TUIModel, args []string) tea.Cmd {
	// Check if we're in a non-chat view (help, models, resume)
	// If so, return to chat instead of quitting the application
	if model.content.GetActiveView() != ViewChat {
		model.content.ShowChat()
		return nil
	}

	// Save the session before quitting the application
	model.saveSession()
	// Quit the application
	return tea.Quit
}

func handleContextCommand(model *TUIModel, args []string) tea.Cmd {
	return func() tea.Msg {
		if model.session == nil {
			return showContextMsg{content: "No active session. Use :login to configure a provider and start chatting."}
		}
		info := model.session.GetContextInfo()
		return showContextMsg{content: renderContextInfo(info)}
	}
}

func handleClearHistoryCommand(model *TUIModel, args []string) tea.Cmd {
	// Clear persistent history
	if model.historyStore != nil {
		if err := model.historyStore.Clear(); err != nil {
			model.commandLine.AddToast("Failed to clear history", "error", 3000)
			return nil
		}
	}

	// Clear in-memory history
	model.promptHistory = make([]promptHistoryEntry, 0)
	model.historyCursor = 0
	model.historySaved = false
	model.historyPendingPrompt = ""

	model.commandLine.AddToast("Prompt history cleared", "success", 3000)
	return nil
}

func handleResumeCommand(model *TUIModel, args []string) tea.Cmd {
	return func() tea.Msg {
		if model == nil || model.config == nil {
			return sessionResumeErrorMsg{err: fmt.Errorf("resume unavailable: missing configuration")}
		}

		if !model.config.Session.Enabled {
			return showContextMsg{content: "Session resume is disabled in configuration."}
		}

		repoInfo := GetRepoInfo()

		currentBranch := branchSlugOrDefault(repoInfo.Branch)
		if model.sessionStore == nil ||
			model.sessionStore.projectRoot != repoInfo.ProjectRoot ||
			model.sessionStore.branchSlug != currentBranch {

			maxSessions := 50
			if model.config.Session.MaxSessions > 0 {
				maxSessions = model.config.Session.MaxSessions
			}

			maxAgeDays := 30
			if model.config.Session.MaxAgeDays > 0 {
				maxAgeDays = model.config.Session.MaxAgeDays
			}

			store, err := NewSessionStore(repoInfo, maxSessions, maxAgeDays)
			if err != nil {
				return sessionResumeErrorMsg{err: fmt.Errorf("failed to initialize session store: %w", err)}
			}

			if model.sessionStore != nil {
				model.sessionStore.Close()
			}
			model.sessionStore = store
		}

		if model.sessionStore == nil {
			return sessionResumeErrorMsg{err: fmt.Errorf("session store not initialized")}
		}

		listLimit := 0
		if model.config.Session.ListLimit >= 0 {
			listLimit = model.config.Session.ListLimit
		}

		sessions, err := model.sessionStore.ListSessions(listLimit)
		if err != nil {
			return sessionResumeErrorMsg{err: fmt.Errorf("failed to list sessions: %w", err)}
		}

		return sessionsLoadedMsg{sessions: sessions}
	}
}

func handleExportCommand(model *TUIModel, args []string) tea.Cmd {
	if model.session == nil {
		return func() tea.Msg {
			return showContextMsg{content: "No active session to export. Start a conversation first."}
		}
	}

	// Determine export type from args, default to conversation
	exportType := ExportTypeConversation
	if len(args) > 0 {
		switch args[0] {
		case "full":
			exportType = ExportTypeFull
		case "conversation":
			exportType = ExportTypeConversation
		default:
			model.commandLine.AddToast(fmt.Sprintf("Unknown export type '%s'. Use 'full' or 'conversation'", args[0]), "error", 3000)
			return nil
		}
	}

	// Export the session to a file
	filepath, err := exportSession(model.session, exportType)
	if err != nil {
		return func() tea.Msg {
			return showContextMsg{content: fmt.Sprintf("Export failed: %v", err)}
		}
	}

	// Open the file in the editor using ExecProcess
	cmd := openInEditor(filepath)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return showContextMsg{content: fmt.Sprintf("Editor exited with error: %v", err)}
		}
		model.commandLine.AddToast(fmt.Sprintf("Conversation exported successfully (%s).", exportType), "success", 3000)
		return nil
	})
}

func handleInitCommand(model *TUIModel, args []string) tea.Cmd {
	if model.session == nil {
		return func() tea.Msg {
			return showContextMsg{content: "No active session. Use :login to configure a provider and start chatting."}
		}
	}

	return func() tea.Msg {
		// Check if user wants to force regeneration
		forceMode := len(args) > 0 && args[0] == "force"

		// Check for missing infrastructure files
		missingFiles := checkMissingInfraFiles()

		if len(missingFiles) == 0 {
			if !forceMode {
				return showContextMsg{content: "All infrastructure files already exist:\n✓ AGENTS.md\n✓ Justfile\n✓ .asimi/Dockerfile\n\nUse `:init force` to regenerate them."}
			}

			if program != nil {
				program.Send(showContextMsg{content: "Force regenerating infrastructure files..."})
			}
		} else if !forceMode {
			var message strings.Builder
			message.WriteString("Missing infrastructure files detected:\n")
			for _, file := range missingFiles {
				message.WriteString(fmt.Sprintf("✗ %s\n", file))
			}
			message.WriteString("\nStarting initialization process...")

			// Show the missing files message first
			if program != nil {
				program.Send(showContextMsg{content: message.String()})
			}
		}

		// Use the embedded initialization prompt
		initPrompt := initializePrompt

		if forceMode {
			initPrompt += "\nNote: Force mode enabled - regenerate all infrastructure files even if they exist.\n"
		}

		// Send the initialization prompt to the session
		return initializeProjectMsg{prompt: initPrompt}
	}
}

// initializeProjectMsg is sent when the init command is executed
type initializeProjectMsg struct {
	prompt string
}

// checkMissingInfraFiles checks which infrastructure files are missing
func checkMissingInfraFiles() []string {
	var missing []string

	// Check for AGENTS.md
	if _, err := os.Stat("AGENTS.md"); os.IsNotExist(err) {
		missing = append(missing, "AGENTS.md")
	}

	// Check for Justfile
	if _, err := os.Stat("Justfile"); os.IsNotExist(err) {
		missing = append(missing, "Justfile")
	}

	// Check for .asimi/Dockerfile
	if _, err := os.Stat(".asimi/Dockerfile"); os.IsNotExist(err) {
		missing = append(missing, ".asimi/Dockerfile")
	}

	return missing
}
