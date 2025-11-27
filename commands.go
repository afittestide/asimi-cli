package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/template"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

//go:embed prompts/initialize.txt
var initializePrompt string

//go:embed prompts/compact.txt
var compactPrompt string

//go:embed docs/asimi.default.conf
var sandboxDefaultConfig string

//go:embed dotagents/sandbox/bashrc
var sandboxBashrc string

// GuardrailPrefix is the prefix for all guardrail messages
const GuardrailPrefix = "🛠️ "

// InitTemplateData holds data for the initialization prompt template
type InitTemplateData struct {
	ProjectName  string
	ProjectSlug  string
	MissingFiles []string
	ClearMode    bool
}

// Command represents a slash command
type Command struct {
	Name        string
	Description string
	Handler     func(*TUIModel, []string) tea.Cmd
}

// compactConversationMsg is sent when the compact command is executed
type compactConversationMsg struct{}

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
	registry.RegisterCommand("/resume", "Resume a previous session", handleResumeCommand)
	registry.RegisterCommand("/export", "Export conversation to file and open in $EDITOR (usage: /export [full|conversation])", handleExportCommand)
	registry.RegisterCommand("/init", "Init project to work with asimi (usage: /init [clear])", handleInitCommand)
	registry.RegisterCommand("/compact", "Compact conversation history to reduce context usage", handleCompactCommand)

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

// agentAskLLMMsg is a message sent by the agent to trigger a new LLM conversation
type agentAskLLMMsg struct {
	prompt string
}

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

	markdownEnabled := false
	if model != nil && model.config != nil {
		markdownEnabled = model.config.UI.MarkdownEnabled
	}
	chat := model.content.Chat
	model.content.Chat = NewChatComponent(chat.Width, chat.Height, markdownEnabled)

	// Use the generic startConversationMsg to reset the session
	return func() tea.Msg {
		return startConversationMsg{
			clearHistory: true,
		}
	}
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

func handleResumeCommand(model *TUIModel, args []string) tea.Cmd {
	// Immediately show the resume view with loading state
	showResumeCmd := model.content.ShowResume([]Session{})
	model.content.resume.SetLoading(true)

	// Load sessions in the background
	loadCmd := func() tea.Msg {
		if model == nil || model.config == nil {
			return sessionResumeErrorMsg{err: fmt.Errorf("resume unavailable: missing configuration")}
		}

		if !model.config.Session.Enabled {
			return showContextMsg{content: "Session resume is disabled in configuration."}
		}

		repoInfo := GetRepoInfo()

		currentBranch := branchSlugOrDefault(repoInfo.Branch)
		if model.sessionStore == nil ||
			model.sessionStore.ProjectRoot != repoInfo.ProjectRoot ||
			model.sessionStore.Branch != currentBranch {

			maxSessions := 50
			if model.config.Session.MaxSessions > 0 {
				maxSessions = model.config.Session.MaxSessions
			}

			maxAgeDays := 30
			if model.config.Session.MaxAgeDays > 0 {
				maxAgeDays = model.config.Session.MaxAgeDays
			}

			store, err := NewSessionStore(model.db, repoInfo, maxSessions, maxAgeDays)
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

	// Return both commands - show view immediately, then load data
	return tea.Batch(showResumeCmd, loadCmd)
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
			return showContextMsg{content: "No model connection. Use :login to configure a provider and start chatting."}
		}
	}

	return func() tea.Msg {
		// Check if user wants to clear and regenerate everything
		clearMode := len(args) > 0 && args[0] == "clear"

		// Ensure .agents directory exists
		if err := os.MkdirAll(".agents/sandbox", 0o755); err != nil {
			return showContextMsg{content: fmt.Sprintf("Error creating .agents directory: %v", err)}
		}

		// In clear mode, remove all infrastructure files first
		if clearMode {
			filesToRemove := []string{
				"AGENTS.md",
				"Justfile",
				".agents/asimi.toml",
				".agents/sandbox/bashrc",
				".agents/sandbox/Dockerfile",
			}
			for _, file := range filesToRemove {
				os.Remove(file) // Ignore errors - file might not exist
			}
			if program != nil {
				program.Send(showContextMsg{content: "Cleared existing infrastructure files. Starting fresh initialization...\n"})
			}
		}

		// Always write embedded files (asimi.toml and bashrc)
		// These are simple files we can provide directly
		projectConfigPath := ".agents/asimi.toml"
		if _, err := os.Stat(projectConfigPath); os.IsNotExist(err) || clearMode {
			if err := os.WriteFile(projectConfigPath, []byte(sandboxDefaultConfig), 0o644); err != nil {
				return showContextMsg{content: fmt.Sprintf("Error writing project config file %s: %v", projectConfigPath, err)}
			}
			if program != nil && !clearMode {
				program.Send(showContextMsg{content: fmt.Sprintf("Initialized %s from embedded default\n", projectConfigPath)})
			}
		}

		bashrcPath := ".agents/sandbox/bashrc"
		if _, err := os.Stat(bashrcPath); os.IsNotExist(err) || clearMode {
			if err := os.WriteFile(bashrcPath, []byte(sandboxBashrc), 0o644); err != nil {
				return showContextMsg{content: fmt.Sprintf("Error writing bashrc file %s: %v", bashrcPath, err)}
			}
			if program != nil && !clearMode {
				program.Send(showContextMsg{content: fmt.Sprintf("Initialized %s from embedded default\n", bashrcPath)})
			}
		}

		// Check for missing infrastructure files (AGENTS.md, Justfile, Dockerfile)
		missingFiles := checkMissingInfraFiles()

		if len(missingFiles) == 0 && !clearMode {
			return showContextMsg{content: strings.Join([]string{"All infrastructure files already exist:",
				"✓ AGENTS.md",
				"✓ Justfile",
				"✓ .agents/sandbox/Dockerfile",
				"✓ .agents/sandbox/bashrc",
				"✓ .agents/asimi.toml",
				"",
				"Use `:init clear` to remove and regenerate them."}, "\n")}
		}

		// Show missing files message (if not in clear mode)
		if len(missingFiles) > 0 && !clearMode {
			var message strings.Builder
			message.WriteString("Missing infrastructure files detected:\n")
			for _, file := range missingFiles {
				message.WriteString(fmt.Sprintf("✗ %s\n", file))
			}
			message.WriteString("\nStarting initialization process...")

			if program != nil {
				program.Send(showContextMsg{content: message.String()})
			}
		}

		// Get the project slug from RepoInfo
		slug := model.session.repoInfo.Slug

		// Extract just the project name from the slug (last part after /)
		// For "owner/repo" or "host/owner/repo", we want just "repo"
		projectName := slug
		if idx := strings.LastIndex(slug, "/"); idx >= 0 {
			projectName = slug[idx+1:]
		}

		// Prepare template data
		templateData := InitTemplateData{
			ProjectName:  projectName,
			ProjectSlug:  slug,
			MissingFiles: missingFiles,
			ClearMode:    clearMode,
		}

		// Parse and execute the template
		tmpl, err := template.New("init").Parse(initializePrompt)
		if err != nil {
			return showContextMsg{content: fmt.Sprintf("Error parsing initialization template: %v", err)}
		}

		var initPrompt bytes.Buffer
		if err := tmpl.Execute(&initPrompt, templateData); err != nil {
			return showContextMsg{content: fmt.Sprintf("Error executing initialization template: %v", err)}
		}

		// Capture the original shell runner before switching to host mode
		// This will be the container runner that we'll use for running tests
		shellRunnerMu.RLock()
		originalRunner := currentShellRunner
		shellRunnerMu.RUnlock()

		// Send the initialization prompt to the session with guardrails
		// Use host shell runner for init to avoid container issues
		return startConversationMsg{
			prompt:       initPrompt.String(),
			clearHistory: true,
			onStreamComplete: func(model *TUIModel) tea.Cmd {
				return verifyInit(model, originalRunner)
			},
			RunOnHost: true,
		}
	}
}

// startConversationMsg is sent to start a new conversation with optional guardrails
type startConversationMsg struct {
	prompt           string
	clearHistory     bool
	onStreamComplete func(*TUIModel) tea.Cmd // Optional guardrail function to run after stream completes
	RunOnHost        bool                    // When true, use host shell runner instead of podman
}

// verifyInit runs validation checks after init completes
// It accepts a containerRunner parameter to run tests in the container
func verifyInit(model *TUIModel, containerRunner shellRunner) tea.Cmd {
	return verifyInitWithRetry(model, containerRunner, 0)
}

// verifyInitWithRetry is the internal implementation with retry tracking
func verifyInitWithRetry(model *TUIModel, containerRunner shellRunner, retryCount int) tea.Cmd {
	const maxRetries = 5 // Maximum number of retry attempts

	return func() tea.Msg {
		var results []string
		var hasErrors bool

		report := func(message string) {
			results = append(results, message)
			if program != nil {
				program.Send(showContextMsg{content: shellOutputMidPrefix + " " + message})
			}
		}
		// Send initial message
		if program != nil {
			msg := "\n" + GuardrailPrefix + "Verifying initialization"
			if retryCount > 0 {
				msg += fmt.Sprintf(" (attempt %d/%d)", retryCount+1, maxRetries+1)
			}
			program.Send(showContextMsg{content: msg})
		}

		if _, err := os.Stat("AGENTS.md"); os.IsNotExist(err) {
			report("❌ AGENTS.md was not created")
			hasErrors = true
		} else {
			report("✅ AGENTS.md created")
		}

		if _, err := os.Stat("Justfile"); os.IsNotExist(err) {
			report("❌ Justfile was not created")
			hasErrors = true
		} else {
			report("✅ Justfile created")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			report("running `just build-sandbox`")
			infraResult, err := hostRun(ctx, RunInShellInput{
				Command:     "just build-sandbox",
				Description: "Building infrastructure files",
			})

			if err != nil || infraResult.ExitCode != "0" {
				report(fmt.Sprintf("❌ just build-sandbox failed (exit code: %s)", infraResult.ExitCode))
				if infraResult.Output != "" {
					results = append(results, fmt.Sprintf("   Output: %s", strings.TrimSpace(infraResult.Output)))
				}
				hasErrors = true
			} else {
				report("✅ just build-sandbox completed successfully")
				unameResult, err := containerRunner.Run(ctx, RunInShellInput{
					Command:     "uname",
					Description: "Running tests in container",
				})
				isLinux := strings.Contains(unameResult.Output, "Linux")

				if err != nil || unameResult.ExitCode != "0" || !isLinux {
					report(fmt.Sprintf("❌ Sandbox smoke test failed (exit code: %s)", unameResult.ExitCode))
					hasErrors = true
				} else {
					if program != nil {
						program.Send(showContextMsg{content: `✅ sandbox smoke test completed
Please review your recipes using ':!just' or start fresh with ':new'`})
					}
					return nil
				}
			}
		}

		// Prepare the message to the model
		slog.Debug("In verifyInit", "hasErrors", hasErrors, "messages", results, "retryCount", retryCount)
		if hasErrors {
			// Check if we've exceeded the maximum retry count
			if retryCount >= maxRetries {
				var failureMsg strings.Builder
				failureMsg.WriteString(fmt.Sprintf("\n❌ Initialization failed after %d attempts.\n", maxRetries+1))
				failureMsg.WriteString("The following issues could not be resolved:\n")
				for _, result := range results {
					failureMsg.WriteString(result + "\n")
				}
				failureMsg.WriteString("\nPlease review the errors and try running ':init' again, or manually fix the issues.")
				return showContextMsg{content: failureMsg.String()}
			}

			// Stop and remove the container so the next attempt will rebuild with fixes
			if containerRunner != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := containerRunner.Close(ctx); err != nil {
					slog.Warn("Failed to close container during verifyInit", "error", err)
				}
			}
			var message strings.Builder
			message.WriteString("Issues found verifying initialization.\n" +
				"Please review the failures below and provide a fix.\n" +
				"If files need to be modified, use the appropriate tools.\n")
			for _, result := range results {
				message.WriteString(result + "\n")
			}
			s := message.String()
			// Return a startConversationMsg to send this message to the LLM session
			return startConversationMsg{
				prompt:       s,
				clearHistory: false,
				RunOnHost:    true,
				onStreamComplete: func(model *TUIModel) tea.Cmd {
					return verifyInitWithRetry(model, containerRunner, retryCount+1)
				},
			}
		} else {
			return showContextMsg{content: "\n🎉 All tests passed!\nProject initialization completed successfully"}
		}

	}
}

// checkMissingInfraFiles checks which infrastructure files are missing
func checkMissingInfraFiles() []string {
	var missing []string

	for _, file := range []string{
		"AGENTS.md",
		"Justfile",
		".agents/asimi.toml",
		".agents/sandbox/Dockerfile",
		".agents/sandbox/bashrc",
	} {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			missing = append(missing, file)
		}
	}

	return missing
}

func handleCompactCommand(model *TUIModel, args []string) tea.Cmd {
	if model.session == nil {
		return func() tea.Msg {
			return showContextMsg{content: "No active session to compact. Start a conversation first."}
		}
	}

	return func() tea.Msg {
		// Check if there's enough conversation to compact
		if len(model.session.Messages) <= 2 {
			return showContextMsg{content: "Not enough conversation history to compact. Continue chatting first."}
		}

		// Show compacting message
		if program != nil {
			program.Send(showContextMsg{content: "Compacting conversation history...\nThis may take a moment as we summarize the conversation."})
		}

		// Send the compact request
		return compactConversationMsg{}
	}
}
