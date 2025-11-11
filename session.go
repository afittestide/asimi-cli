package main

import (
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	debug "runtime/debug"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
	lctools "github.com/tmc/langchaingo/tools"
)

// NotifyFunc is a function that handles notifications
type NotifyFunc func(any)

// Session is a lightweight chat loop that uses llms.Model directly
// and native provider tool/function-calling. It executes tools via the
// existing CoreToolScheduler and keeps conversation state locally.
type Session struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	LastUpdated time.Time `json:"last_updated"`
	FirstPrompt string    `json:"first_prompt"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	WorkingDir  string    `json:"working_dir"`
	ProjectSlug string    `json:"project_slug,omitempty"`

	Messages     []llms.MessageContent `json:"messages"`
	ContextFiles map[string]string     `json:"context_files"`

	llm                     llms.Model              `json:"-"`
	toolCatalog             map[string]lctools.Tool `json:"-"`
	toolDefs                []llms.Tool             `json:"-"`
	lastToolCallKey         string                  `json:"-"`
	toolCallRepetitionCount int                     `json:"-"`
	scheduler               *CoreToolScheduler      `json:"-"`
	notify                  NotifyFunc              `json:"-"`
	accumulatedContent      strings.Builder         `json:"-"`
	config                  *LLMConfig              `json:"-"`
	startTime               time.Time               `json:"-"`
}

// formatMetadata returns the metadata header used by export helpers.
func (s *Session) formatMetadata(exportType ExportType, exportedAt time.Time) string {
	var b strings.Builder
	exported := exportedAt.Format("2006-01-02 15:04:05")
	version := asimiVersion()

	b.WriteString(fmt.Sprintf("**Asimi Version:** %s \n", version))
	b.WriteString(fmt.Sprintf("**Export Type:** %s\n", exportType))
	b.WriteString(fmt.Sprintf("**Session ID:** %s | **Working Directory:** %s\n", s.ID, s.WorkingDir))
	b.WriteString(fmt.Sprintf("**Provider:** %s | **Model:** %s\n", s.Provider, s.Model))
	b.WriteString(fmt.Sprintf("**Created:** %s | **Last Updated:** %s | **Exported:** %s\n",
		s.CreatedAt.Format("2006-01-02 15:04:05"),
		s.LastUpdated.Format("2006-01-02 15:04:05"),
		exported))
	if s.ProjectSlug != "" {
		b.WriteString(fmt.Sprintf("**Project:** %s\n", s.ProjectSlug))
	}

	return b.String()
}

// No syncMessages method needed anymore - we only use Messages

// resetStreamBuffer safely resets the accumulated content buffer
func (s *Session) resetStreamBuffer() {
	s.accumulatedContent.Reset()
}

// getStreamBuffer returns the current accumulated content and optionally resets it
func (s *Session) getStreamBuffer(reset bool) string {
	content := s.accumulatedContent.String()
	if reset {
		s.accumulatedContent.Reset()
	}
	return content
}

// notification messages
type streamChunkMsg string
type streamStartMsg struct{}
type streamCompleteMsg struct{}
type streamInterruptedMsg struct{ partialContent string }
type streamErrorMsg struct{ err error }
type streamMaxTurnsExceededMsg struct{ maxTurns int }
type streamMaxTokensReachedMsg struct{ content string }

// Local copies of prompt partials and template used by the session, to decouple from agent.go.
var sessPromptPartials = map[string]any{
	"SandboxStatus": "none",
	"UserMemory":    "",
	"Env":           "",
	"ReadFile":      "read_file",
	"WriteFile":     "write_file",
	"Grep":          "grep",
	"Glob":          "glob",
	"Edit":          "replace_text",
	"Shell":         "run_in_shell",
	"ReadManyFiles": "read_many_files",
	"Memory":        "",
	"LS":            "list_files",
	"history":       "",
}

//go:embed prompts/system_prompt.tmpl
var sessSystemPromptTemplate string

// NewSession creates a new Session instance with a system prompt and tools.
func NewSession(llm llms.Model, cfg *Config, repoInfo RepoInfo, toolNotify NotifyFunc) (*Session, error) {
	now := time.Now()
	workingDir, _ := os.Getwd()

	s := &Session{
		ID:          generateSessionID(),
		CreatedAt:   now,
		LastUpdated: now,
		WorkingDir:  workingDir,
		llm:         llm,
		toolCatalog: map[string]lctools.Tool{},
		notify:      toolNotify,
	}
	if cfg != nil {
		s.config = &cfg.LLM
		s.Provider = cfg.LLM.Provider
		s.Model = cfg.LLM.Model
		// Set default maxTurns if not configured
	} else {
		// Create default config if none provided
		s.config = &LLMConfig{}
	}
	if s.config.MaxTurns <= 0 {
		s.config.MaxTurns = 999
	}

	// Build system prompt from the existing template and partials, same as the agent.
	partials := make(map[string]any, len(sessPromptPartials))
	for k, v := range sessPromptPartials {
		partials[k] = v
	}
	partials["Env"] = sessBuildEnvBlock(repoInfo)

	pt := prompts.PromptTemplate{
		Template:         sessSystemPromptTemplate,
		TemplateFormat:   prompts.TemplateFormatGoTemplate,
		InputVariables:   []string{"input", "agent_scratchpad"},
		PartialVariables: partials,
	}

	// Render with empty input/scratchpad since this is a system message.
	sys, err := pt.Format(map[string]any{"input": "", "agent_scratchpad": ""})
	if err != nil {
		return nil, fmt.Errorf("formatting system prompt: %w", err)
	}
	var parts []llms.ContentPart
	if s.config != nil && s.config.Provider == "anthropic" {
		parts = append(parts, llms.TextPart("You are Claude Code, Anthropic's official CLI for Claude."))
	}
	parts = append(parts, llms.TextPart(sys))

	// Add AGENTS.md to system message if it exists
	projectContext := readProjectContext()
	if projectContext != "" {
		parts = append(parts, llms.TextPart(fmt.Sprintf("\n--- Project specific directions from: AGENTS.md ---\n%s\n--- End of Directions from: AGENTS.md ---", projectContext)))
	}

	if s.config != nil && s.config.Provider == "ollama" {
		var builder strings.Builder
		for _, part := range parts {
			if textPart, ok := part.(llms.TextContent); ok {
				if builder.Len() > 0 {
					builder.WriteString("\n\n")
				}
				builder.WriteString(textPart.Text)
			}
		}
		parts = []llms.ContentPart{llms.TextPart(builder.String())}
	}

	s.Messages = append(s.Messages, llms.MessageContent{
		Role:  llms.ChatMessageTypeSystem,
		Parts: parts,
	})

	// Build tool schema for the model and execution catalog for the scheduler.
	s.toolDefs, s.toolCatalog = buildLLMTools(cfg)
	s.scheduler = NewCoreToolScheduler(s.notify)
	s.ContextFiles = make(map[string]string)
	s.startTime = time.Now()
	return s, nil
}

// AddContextFile adds file content to the context for the next prompt
func (s *Session) AddContextFile(path, content string) {
	s.ContextFiles[path] = content
}

// ClearContext removes all dynamically added file content from the context
func (s *Session) ClearContext() {
	s.ContextFiles = make(map[string]string)
}

// ClearHistory clears the conversation history but keeps the system message
func (s *Session) ClearHistory() {
	// Keep only the system message (first message)
	if len(s.Messages) > 0 && s.Messages[0].Role == llms.ChatMessageTypeSystem {
		s.Messages = s.Messages[:1]
	} else {
		s.Messages = []llms.MessageContent{}
	}

	// Reset tool call tracking
	s.lastToolCallKey = ""
	s.toolCallRepetitionCount = 0

	// Reset session start time
	s.startTime = time.Now()

	s.ClearContext()
}

// HasContextFiles returns true if there are files in the context
func (s *Session) HasContextFiles() bool {
	return len(s.ContextFiles) > 0
}

// GetContextFiles returns a copy of the context files map
func (s *Session) GetContextFiles() map[string]string {
	result := make(map[string]string)
	for k, v := range s.ContextFiles {
		result[k] = v
	}
	return result
}

// buildPromptWithContext builds a prompt that includes all file content
func (s *Session) buildPromptWithContext(userPrompt string) string {
	if len(s.ContextFiles) == 0 {
		return userPrompt
	}

	var fileContents []string
	for path, content := range s.ContextFiles {
		fileContents = append(fileContents, fmt.Sprintf("--- Context from: %s ---\n%s\n--- End of Context from: %s ---", path, content, path))
	}

	return strings.Join(fileContents, "\n\n") + "\n" + userPrompt
}

// getToolCallKey generates a unique key for a tool call based on name and arguments
func (s *Session) getToolCallKey(name, argsJSON string) string {
	keyString := fmt.Sprintf("%s:%s", name, argsJSON)
	hash := sha256.Sum256([]byte(keyString))
	return hex.EncodeToString(hash[:])
}

// checkToolCallLoop detects if the same tool call is being repeated
func (s *Session) checkToolCallLoop(name, argsJSON string) bool {
	const toolCallLoopThreshold = 3 // More conservative than gemini-cli's 5

	key := s.getToolCallKey(name, argsJSON)
	if s.lastToolCallKey == key {
		s.toolCallRepetitionCount++
	} else {
		s.lastToolCallKey = key
		s.toolCallRepetitionCount = 1
	}

	if s.toolCallRepetitionCount >= toolCallLoopThreshold {
		slog.Warn("tool call loop detected", "tool", name, "count", s.toolCallRepetitionCount)
		return true
	}

	return false
}

// removeUnmatchedToolCalls removes any trailing assistant messages with tool calls
// that don't have corresponding tool responses. This prevents errors when the agent
// is interrupted mid-execution.
func (s *Session) removeUnmatchedToolCalls() {
	if len(s.Messages) == 0 {
		return
	}

	for len(s.Messages) > 0 {
		lastIdx := len(s.Messages) - 1
		lastMsg := s.Messages[lastIdx]

		if lastMsg.Role == llms.ChatMessageTypeAI {
			hasToolCalls := false
			for _, part := range lastMsg.Parts {
				if _, ok := part.(llms.ToolCall); ok {
					hasToolCalls = true
					break
				}
			}

			if hasToolCalls {
				slog.Debug("removing unmatched tool call from context")
				s.Messages = s.Messages[:lastIdx]
				continue
			}
		}

		if lastMsg.Role == llms.ChatMessageTypeTool {
			if lastIdx == 0 {
				slog.Debug("removing tool result without prior messages")
				s.Messages = s.Messages[:lastIdx]
				continue
			}

			// Look backwards past other tool messages to find the AI message with tool calls
			var aiMsg *llms.MessageContent
			for i := lastIdx - 1; i >= 0; i-- {
				if s.Messages[i].Role == llms.ChatMessageTypeAI {
					aiMsg = &s.Messages[i]
					break
				}
				// Stop if we encounter a non-tool message that isn't AI
				if s.Messages[i].Role != llms.ChatMessageTypeTool {
					break
				}
			}

			if aiMsg == nil {
				slog.Debug("removing tool result without prior AI message")
				s.Messages = s.Messages[:lastIdx]
				continue
			}

			toolCallIDs := make(map[string]struct{})
			for _, part := range aiMsg.Parts {
				if tc, ok := part.(llms.ToolCall); ok && tc.ID != "" {
					toolCallIDs[tc.ID] = struct{}{}
				}
			}

			valid := len(toolCallIDs) > 0
			for _, part := range lastMsg.Parts {
				if resp, ok := part.(llms.ToolCallResponse); ok {
					if _, exists := toolCallIDs[resp.ToolCallID]; !exists || resp.ToolCallID == "" {
						valid = false
						break
					}
				}
			}

			if !valid {
				slog.Debug("removing dangling tool result from context")
				s.Messages = s.Messages[:lastIdx]
				continue
			}
		}

		return
	}
}

// prepareUserMessage builds the prompt with context and adds it to the message history
func (s *Session) prepareUserMessage(prompt string) {
	// Before adding a new user message, check for and remove any unmatched tool calls
	s.removeUnmatchedToolCalls()

	fullPrompt := s.buildPromptWithContext(prompt)
	s.Messages = append(s.Messages, llms.MessageContent{
		Role:  llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{llms.TextPart(fullPrompt)},
	})
}

func (s *Session) generateLLMResponse(ctx context.Context, streamingFunc func(ctx context.Context, chunk []byte) error) (*llms.ContentChoice, error) {
	// Build call options; try with explicit tool choice first, then without, then no tools.
	var callOptsWithChoice []llms.CallOption
	var callOptsNoChoice []llms.CallOption
	if len(s.toolDefs) > 0 {
		callOptsNoChoice = []llms.CallOption{llms.WithTools(s.toolDefs), llms.WithMaxTokens(64000)}
		callOptsWithChoice = append([]llms.CallOption{}, callOptsNoChoice...)
		callOptsWithChoice = append(callOptsWithChoice, llms.WithToolChoice("auto"))
	}

	// Add streaming option if requested
	if streamingFunc != nil {
		callOptsWithChoice = append(callOptsWithChoice, llms.WithStreamingFunc(streamingFunc))
	}

	// Remove any unmatched tool calls from context before sending to API
	s.removeUnmatchedToolCalls()

	// Attempt with explicit tool choice first.
	resp, err := s.llm.GenerateContent(ctx, s.Messages, callOptsWithChoice...)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response choices")
	}
	return resp.Choices[0], nil
}

// appendMessages adds LLM response content and tool calls to the message history
func (s *Session) appendMessages(content string, toolCalls []llms.ToolCall) {
	// Build the assistant message parts
	var parts []llms.ContentPart

	// Add text content if present
	if strings.TrimSpace(content) != "" {
		parts = append(parts, llms.TextPart(content))
	}

	// Add tool calls if present
	for _, toolCall := range toolCalls {
		parts = append(parts, llms.ToolCall{
			ID:           toolCall.ID,
			Type:         toolCall.Type,
			FunctionCall: toolCall.FunctionCall,
		})
	}

	// Only add the assistant message if we have content or tool calls
	if len(parts) > 0 {
		s.Messages = append(s.Messages, llms.MessageContent{
			Role:  llms.ChatMessageTypeAI,
			Parts: parts,
		})
	}
}

// executeToolCall executes a single tool call and returns the response content
func (s *Session) executeToolCall(ctx context.Context, tool lctools.Tool, tc llms.ToolCall, argsJSON string) llms.ToolCallResponse {
	var out string
	var callErr error

	if s.scheduler != nil {
		ch := s.scheduler.Schedule(tool, argsJSON)
		res := <-ch
		out, callErr = res.Output, res.Error
	} else {
		out, callErr = tool.Call(ctx, argsJSON)
	}

	if callErr != nil {
		return llms.ToolCallResponse{
			ToolCallID: tc.ID,
			Name:       tc.FunctionCall.Name,
			Content:    fmt.Sprintf("Error: %v", callErr),
		}
	}

	return llms.ToolCallResponse{
		ToolCallID: tc.ID,
		Name:       tc.FunctionCall.Name,
		Content:    out,
	}
}

// GetMessageSnapshot returns the current size of the message history for rollback purposes
func (s *Session) GetMessageSnapshot() int {
	return len(s.Messages)
}

// RollbackTo truncates the message history back to the provided snapshot index
func (s *Session) RollbackTo(snapshot int) {
	if snapshot < 1 {
		snapshot = 1 // always preserve the system prompt
	}
	if snapshot > len(s.Messages) {
		snapshot = len(s.Messages)
	}
	if snapshot < len(s.Messages) {
		s.Messages = s.Messages[:snapshot]
	}

	// Reset tool loop detection state when rolling back
	s.lastToolCallKey = ""
	s.toolCallRepetitionCount = 0
}

// hasToolCallResponse checks if toolMessages already contains a response for the given tool call ID
// TODO: test to ensure we need this and the loops that use it
func hasToolCallResponse(toolMessages []llms.MessageContent, toolCallID string) bool {
	for _, msg := range toolMessages {
		if msg.Role != llms.ChatMessageTypeTool {
			continue
		}
		for _, part := range msg.Parts {
			if resp, ok := part.(llms.ToolCallResponse); ok && resp.ToolCallID == toolCallID {
				return true
			}
		}
	}
	return false
}

// processToolCalls handles executing tool calls and building response messages
func (s *Session) processToolCalls(ctx context.Context, toolCalls []llms.ToolCall) ([]llms.MessageContent, bool) {
	toolMessages := make([]llms.MessageContent, 0, len(toolCalls))

	for i, tc := range toolCalls {
		if tc.FunctionCall == nil {
			continue
		}
		name := tc.FunctionCall.Name
		argsJSON := tc.FunctionCall.Arguments

		// Check for context cancellation before processing each tool call
		select {
		case <-ctx.Done():
			// Context was cancelled - provide "session aborted" responses for remaining tool calls
			slog.Debug("context cancelled during tool execution, aborting remaining tool calls", "completed", i, "total", len(toolCalls))

			// Add abort responses for all remaining tool calls (including current one)
			for _, remainingTC := range toolCalls {
				if remainingTC.FunctionCall == nil {
					continue
				}
				if !hasToolCallResponse(toolMessages, remainingTC.ID) {
					toolMessages = append(toolMessages, llms.MessageContent{
						Role: llms.ChatMessageTypeTool,
						Parts: []llms.ContentPart{llms.ToolCallResponse{
							ToolCallID: remainingTC.ID,
							Name:       remainingTC.FunctionCall.Name,
							Content:    "error: session aborted by user",
						}},
					})
				}
			}

			return toolMessages, true // shouldReturn = true
		default:
			// Continue with normal processing
		}

		// Check for tool call loops
		if s.checkToolCallLoop(name, argsJSON) {
			toolMessages = append(toolMessages, llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{llms.ToolCallResponse{
					ToolCallID: tc.ID,
					Name:       name,
					Content:    fmt.Sprintf("error: tool call loop detected after %d attempts", s.toolCallRepetitionCount),
				}},
			})
			return toolMessages, true // shouldReturn = true
		}

		tool, ok := s.toolCatalog[name]
		if !ok {
			// If the model requested an unknown tool, feed an error response back.
			toolMessages = append(toolMessages, llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{llms.ToolCallResponse{
					ToolCallID: tc.ID,
					Name:       name,
					Content:    fmt.Sprintf("error: unknown tool %q", name),
				}},
			})
			continue
		}

		// Execute tool and add response
		response := s.executeToolCall(ctx, tool, tc, argsJSON)
		slog.Debug("Called a tool", "tool", name, "args", argsJSON, "response", response)
		toolMessages = append(toolMessages, llms.MessageContent{
			Role:  llms.ChatMessageTypeTool,
			Parts: []llms.ContentPart{response},
		})
	}

	return toolMessages, false // shouldReturn = false
}

// Ask sends a user prompt through the native loop. It returns the final assistant text.
// It handles provider-native tool calls by executing them and feeding results back.
func (s *Session) Ask(ctx context.Context, prompt string) (string, error) {
	// Build prompt with context if available and add to messages
	s.prepareUserMessage(prompt)
	// Clear context after building the prompt
	defer s.ClearContext()

	// A simple loop: generate -> maybe tool calls -> tool responses -> generate.
	var finalText string
	var lastAssistant string
	var hadAnyToolCall bool
	var i int
	maxTurns := s.config.MaxTurns
	for i = 0; i < maxTurns; i++ {
		choice, err := s.generateLLMResponse(ctx, nil)
		if err != nil {
			return "", err
		}

		// Check if response was truncated due to max tokens
		if choice.StopReason == "max_tokens" {
			return choice.Content + "\n\n[Response truncated due to length limit]", nil
		}

		// Build response with reasoning content if available
		responseText := choice.Content
		if choice.ReasoningContent != "" {
			responseText = "<thinking>\n" + choice.ReasoningContent + "\n</thinking>\n\n" + choice.Content
		}

		// Record assistant response in message history
		if strings.TrimSpace(responseText) != "" {
			finalText = responseText
		}
		s.appendMessages(responseText, choice.ToolCalls)

		// Handle tool calls, if any.
		if len(choice.ToolCalls) == 0 {
			// Give the model another turn to issue tool calls if it only planned.
			// Stop if it repeats the same assistant content.
			if hadAnyToolCall || strings.TrimSpace(choice.Content) == strings.TrimSpace(lastAssistant) {
				break
			}
			lastAssistant = choice.Content
			continue
		}
		hadAnyToolCall = true

		// Process tool calls and add responses
		toolMessages, shouldReturn := s.processToolCalls(ctx, choice.ToolCalls)
		if len(toolMessages) > 0 {
			s.Messages = append(s.Messages, toolMessages...)
		}

		if shouldReturn {
			return finalText, nil
		}

		// Continue to next iteration to let the model incorporate tool results.
		if len(toolMessages) > 0 {
			continue
		}

		// No tool responses to send; break.
		break
	}
	if i < maxTurns {
		return finalText, nil
	}
	return fmt.Sprintf("%s\n\nEnded after %d interation", finalText, maxTurns), nil
}

// AskStream sends a user prompt through the native loop with streaming support.
// It launches the streaming process in a goroutine and returns immediately.
// Uses the notify callback to send streaming chunks as they arrive.
// Supports cancellation via the provided context.
func (s *Session) AskStream(ctx context.Context, prompt string) {
	// Launch streaming in a goroutine to avoid blocking the UI
	go func() {
		// Ensure cleanup on exit
		defer func() {
			s.ClearContext()
		}()

		// Build prompt with context if available and add to messages
		s.prepareUserMessage(prompt)

		// Notify UI that streaming has started
		if s.notify != nil {
			s.notify(streamStartMsg{})
		}

		// A simple loop: generate -> maybe tool calls -> tool responses -> generate.
		// Cap at a few iterations to avoid infinite loops.
		var i int
		maxTurns := s.config.MaxTurns
		for i = 0; i < maxTurns; i++ {
			s.resetStreamBuffer()

			// Check for cancellation
			select {
			case <-ctx.Done():
				// Streaming was cancelled - add any accumulated content to message history
				accumulatedText := s.getStreamBuffer(false)
				if strings.TrimSpace(accumulatedText) != "" {
					s.appendMessages(accumulatedText, nil)
				}
				if s.notify != nil {
					s.notify(streamInterruptedMsg{partialContent: accumulatedText})
				}
				return
			default:
				// Continue with streaming
			}

			// Create streaming function that accumulates content and notifies UI
			streamingFunc := func(ctx context.Context, chunk []byte) error {
				// Check for cancellation in streaming callback
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				chunkStr := string(chunk)
				s.accumulatedContent.WriteString(chunkStr)
				if s.notify != nil {
					s.notify(streamChunkMsg(chunkStr))
				}
				return nil
			}

			choice, err := s.generateLLMResponse(ctx, streamingFunc)
			if err != nil {
				// Check if this was a cancellation
				if ctx.Err() != nil {
					accumulatedText := s.getStreamBuffer(false)
					if strings.TrimSpace(accumulatedText) != "" {
						s.appendMessages(accumulatedText, nil)
					}
					if s.notify != nil {
						s.notify(streamInterruptedMsg{partialContent: accumulatedText})
					}
					return
				}

				// Regular error
				if s.notify != nil {
					s.notify(streamErrorMsg{err: err})
				}
				return
			}

			// Use accumulated content as the response
			responseContent := s.getStreamBuffer(false)

			// Check if response was truncated due to max tokens
			if choice.StopReason == "max_tokens" {
				if s.notify != nil {
					s.notify(streamMaxTokensReachedMsg{content: responseContent})
				}
				s.appendMessages(responseContent, choice.ToolCalls)
				break
			}

			// Add reasoning content if available (for models like deepseek-reasoner)
			if choice.ReasoningContent != "" && s.notify != nil {
				s.notify(streamChunkMsg("\n\n<thinking>\n" + choice.ReasoningContent + "\n</thinking>\n\n"))
			}

			// Add the assistant message with content and tool calls to message history
			s.appendMessages(responseContent, choice.ToolCalls)

			// Handle tool calls, if any.
			if len(choice.ToolCalls) == 0 {
				// No tool calls - streaming is complete
				break
			}

			// Process tool calls and add responses
			toolMessages, shouldReturn := s.processToolCalls(ctx, choice.ToolCalls)
			if len(toolMessages) > 0 {
				s.Messages = append(s.Messages, toolMessages...)
			}

			if shouldReturn {
				break
			}

			// Continue to next iteration to let the model incorporate tool results.
			if len(toolMessages) > 0 {
				continue
			}

			// No tool responses to send; break.
			break
		}

		// Check if we exceeded max turns and send appropriate notification
		if s.notify != nil {
			if i >= maxTurns {
				s.notify(streamMaxTurnsExceededMsg{maxTurns: maxTurns})
			} else {
				s.notify(streamCompleteMsg{})
			}
		}
	}()
}

// sessBuildEnvBlock constructs a markdown summary of the OS, shell, and key paths.
func sessBuildEnvBlock(repoInfo RepoInfo) string {
	cwd, _ := os.Getwd()
	if cwd == "" {
		cwd = "(unknown)"
	}

	home, _ := os.UserHomeDir()
	if home == "" {
		home = "(unknown)"
	}

	root := repoInfo.ProjectRoot
	if root == "" {
		root = "(unknown)"
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf(`- **OS:** debian
- **Shell:** %s
- **Paths:**
  - **cwd:** %s
  - **project root:** %s
  - **home:** %s`,
		shell,
		cwd,
		root,
		home))

	// Add branch information if available
	if repoInfo.Branch != "" {
		result.WriteString(fmt.Sprintf("\n- **Branch:** %s", repoInfo.Branch))
	}

	// Add worktree information if we're in a worktree
	if repoInfo.IsWorktree && repoInfo.Branch != "" {
		result.WriteString(fmt.Sprintf("\n\n**IMPORTANT:** We're working on worktree '%s' at '%s'. Changes will be squashed before merging so commit frequently.",
			repoInfo.Branch,
			cwd))
	}

	return result.String()
}

func asimiVersion() string {
	if strings.TrimSpace(version) != "" {
		return strings.TrimSpace(version)
	}

	if v := os.Getenv("ASIMI_VERSION"); v != "" {
		return v
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if normalized := normalizeBuildVersion(info.Main.Version); normalized != "" {
			return normalized
		}

		var revision string
		var modified bool
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.modified":
				modified = setting.Value == "true"
			}
		}

		if revision != "" {
			shortRev := revision
			if len(shortRev) > 7 {
				shortRev = shortRev[:7]
			}
			if modified {
				return fmt.Sprintf("dev-%s-dirty", shortRev)
			}
			return fmt.Sprintf("dev-%s", shortRev)
		}
	}

	return "dev"
}

func normalizeBuildVersion(v string) string {
	if v == "" || v == "(devel)" {
		return ""
	}
	return strings.TrimPrefix(v, "v")
}

// readProjectContext reads the contents of AGENTS.md from the current working directory.
func readProjectContext() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	path := filepath.Join(wd, "AGENTS.md")
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

// buildLLMTools returns the LLM tool/function definitions and a catalog by name for execution.
func buildLLMTools(cfg *Config) ([]llms.Tool, map[string]lctools.Tool) {
	// Get tools with config
	tools := getAvailableTools(cfg)

	// Map our concrete tools by name for execution.
	execCatalog := map[string]lctools.Tool{}
	for i := range tools {
		tool := tools[i]
		//nolint:typecheck // Tool interface is correctly defined in tools.go
		execCatalog[tool.Name()] = tool
	}

	// Helper to produce a basic JSON schema for function parameters.
	obj := func(props map[string]any, required []string) map[string]any {
		m := map[string]any{
			"type":       "object",
			"properties": props,
		}
		if len(required) > 0 {
			m["required"] = required
		}
		return m
	}

	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	boolean := func(desc string) map[string]any { return map[string]any{"type": "boolean", "description": desc} }

	defs := []llms.Tool{
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "read_file",
				Description: "Reads a file and returns its content.",
				Parameters: obj(map[string]any{
					"path": str("Absolute or relative path to the file"),
				}, []string{"path"}),
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "write_file",
				Description: "Writes content to a file, creating or overwriting it.",
				Parameters: obj(map[string]any{
					"path":    str("Target file path"),
					"content": str("File contents to write"),
				}, []string{"path", "content"}),
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "list_files",
				Description: "Lists the contents of a directory.",
				Parameters: obj(map[string]any{
					"path": str("Directory path (defaults to '.')"),
				}, nil),
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "replace_text",
				Description: "Replaces all occurrences of a string in a file with another string.",
				Parameters: obj(map[string]any{
					"path":     str("File path"),
					"old_text": str("Text to replace"),
					"new_text": str("Replacement text"),
				}, []string{"path", "old_text", "new_text"}),
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "run_in_shell",
				Description: "Executes a shell command in a persistent shell session.",
				Parameters: obj(map[string]any{
					"command":     str("Shell command to run"),
					"description": str("Short description of the command"),
				}, []string{"command"}),
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "read_many_files",
				Description: "Reads content from multiple files specified by wildcard paths.",
				Parameters: obj(map[string]any{
					"paths": map[string]any{
						"type":        "array",
						"description": "Array of file paths or glob patterns to read",
						"items": map[string]any{
							"type":        "string",
							"description": "A file path or glob pattern",
						},
					},
				}, []string{"paths"}),
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "merge",
				Description: "Squashes a worktree-backed branch onto the main branch after user approval, then cleans up the worktree.",
				Parameters: obj(map[string]any{
					"worktree_path":  str("Absolute path to the worktree directory"),
					"branch":         str("Name of the branch associated with the worktree"),
					"main_branch":    str("Name of the trunk branch to merge into (defaults to main)"),
					"commit_message": str("Optional squash commit message to use"),
					"auto_approve":   boolean("Set to true to skip interactive approval (requires commit_message)"),
					"skip_review":    boolean("Set to true to skip launching lazygit"),
					"push":           boolean("Push the updated main branch to origin after merging"),
				}, []string{"worktree_path", "branch"}),
			},
		},
	}

	return defs, execCatalog
}

// GetSessionDuration returns the duration since the session started
func (s *Session) GetSessionDuration() time.Duration {
	return time.Since(s.startTime)
}

// GetContextUsagePercent returns the percentage of context used (0-100)
func (s *Session) GetContextUsagePercent() float64 {
	info := s.GetContextInfo()
	if info.TotalTokens <= 0 {
		return 0
	}
	return (float64(info.UsedTokens) / float64(info.TotalTokens)) * 100
}

// CompactHistory summarizes the conversation history to reduce context usage
// It uses the high-end model to create a comprehensive summary that includes:
// - All diffs/changes made to files
// - Key decisions and outcomes
// - Important technical details
// The summary replaces the conversation history while preserving the system message
func (s *Session) CompactHistory(ctx context.Context, compactPrompt string) (string, error) {
	if len(s.Messages) <= 2 {
		return "", fmt.Errorf("not enough conversation history to compact")
	}

	// Build the content to summarize
	var contentBuilder strings.Builder

	// Collect all diffs and file changes
	contentBuilder.WriteString("## File Changes and Diffs\n\n")
	fileChanges := s.extractFileChanges()
	if len(fileChanges) > 0 {
		for path, changes := range fileChanges {
			contentBuilder.WriteString(fmt.Sprintf("### %s\n\n", path))
			for _, change := range changes {
				contentBuilder.WriteString(change)
				contentBuilder.WriteString("\n\n")
			}
		}
	} else {
		contentBuilder.WriteString("No file changes recorded.\n\n")
	}

	// Collect conversation messages (excluding tool calls)
	contentBuilder.WriteString("## Conversation History\n\n")
	for i := 1; i < len(s.Messages); i++ {
		msg := s.Messages[i]

		switch msg.Role {
		case llms.ChatMessageTypeHuman:
			contentBuilder.WriteString("**User:**\n")
			for _, part := range msg.Parts {
				if textPart, ok := part.(llms.TextContent); ok {
					contentBuilder.WriteString(textPart.Text)
					contentBuilder.WriteString("\n\n")
				}
			}

		case llms.ChatMessageTypeAI:
			contentBuilder.WriteString("**Assistant:**\n")
			// Only include text content, skip tool calls
			for _, part := range msg.Parts {
				if textPart, ok := part.(llms.TextContent); ok {
					contentBuilder.WriteString(textPart.Text)
					contentBuilder.WriteString("\n\n")
				}
			}
		}
	}

	// Build the compaction request
	fullPrompt := fmt.Sprintf("%s\n\n---\n\n%s", compactPrompt, contentBuilder.String())

	// Save the current messages
	originalMessages := s.Messages
	systemMessage := s.Messages[0]

	// Create a temporary message history with just the system message and compaction request
	s.Messages = []llms.MessageContent{
		systemMessage,
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(fullPrompt)},
		},
	}

	// Generate the summary using the LLM
	choice, err := s.generateLLMResponse(ctx, nil)
	if err != nil {
		// Restore original messages on error
		s.Messages = originalMessages
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	summary := choice.Content
	if choice.ReasoningContent != "" {
		summary = choice.ReasoningContent + "\n\n" + choice.Content
	}

	// Replace the conversation history with the summary
	s.Messages = []llms.MessageContent{
		systemMessage,
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart("Previous conversation summary:\n\n" + summary)},
		},
		{
			Role:  llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{llms.TextPart("I understand. I have the context from the previous conversation and am ready to continue.")},
		},
	}

	// Reset tool call tracking
	s.lastToolCallKey = ""
	s.toolCallRepetitionCount = 0

	return summary, nil
}

// extractFileChanges extracts all file changes from tool call responses
func (s *Session) extractFileChanges() map[string][]string {
	changes := make(map[string][]string)

	for _, msg := range s.Messages {
		if msg.Role != llms.ChatMessageTypeTool {
			continue
		}

		for _, part := range msg.Parts {
			if toolResp, ok := part.(llms.ToolCallResponse); ok {
				// Track write_file and replace_text operations
				if toolResp.Name == "write_file" || toolResp.Name == "replace_text" {
					// Try to extract the file path from the response
					// The response format varies, but we can try to parse it
					content := toolResp.Content
					if strings.Contains(content, "Successfully") || strings.Contains(content, "wrote") {
						// Extract file path - this is a simple heuristic
						lines := strings.Split(content, "\n")
						for _, line := range lines {
							if strings.Contains(line, "Successfully") || strings.Contains(line, "wrote") {
								changes["file-changes"] = append(changes["file-changes"], content)
								break
							}
						}
					}
				}
			}
		}
	}

	return changes
}

type SessionIndex struct {
	Sessions []Session `json:"sessions"`
}

func generateSessionID() string {
	timestamp := time.Now().Format("2006-01-02-150405")

	randomBytes := make([]byte, 4)
	crand.Read(randomBytes)
	suffix := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("%s-%s", timestamp, suffix)
}

func branchSlugOrDefault(branch string) string {
	if branch == "" {
		branch = "main"
	}

	slug := sanitizeSegment(branch)
	if slug == "" {
		return "main"
	}

	return slug
}

const defaultProjectSlug = "project-unknown"

func projectSlug(workingDir string) string {
	if workingDir == "" {
		cwd, err := os.Getwd()
		if err == nil {
			workingDir = cwd
		}
	}

	root := findProjectRoot(workingDir)
	if slug := remoteRepoSlug(root); slug != "" {
		return slug
	}

	return fallbackProjectSlug(root)
}

func findProjectRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == "/" || parent == dir {
			return start
		}
		dir = parent
	}
}

func remoteRepoSlug(workingDir string) string {
	remote, err := gitRemoteOriginURL(workingDir)
	if err != nil || remote == "" {
		return ""
	}

	owner, repo := parseGitRemote(remote)
	owner = sanitizeSegment(owner)
	repo = sanitizeSegment(repo)
	if owner == "" || repo == "" {
		return ""
	}

	return owner + "/" + repo
}

func gitRemoteOriginURL(workingDir string) (string, error) {
	cmd := exec.Command("git", "-C", workingDir, "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func parseGitRemote(remote string) (owner, repo string) {
	remote = strings.TrimSpace(remote)
	remote = strings.TrimSuffix(remote, ".git")
	if remote == "" {
		return "", ""
	}

	if strings.Contains(remote, "://") {
		if u, err := url.Parse(remote); err == nil {
			segments := strings.Split(strings.Trim(u.Path, "/"), "/")
			if len(segments) >= 2 {
				owner = segments[len(segments)-2]
				repo = segments[len(segments)-1]
			}
			return owner, repo
		}
	}

	if strings.Contains(remote, ":") {
		parts := strings.SplitN(remote, ":", 2)
		if len(parts) == 2 {
			path := strings.Trim(parts[1], "/")
			segments := strings.Split(path, "/")
			if len(segments) >= 2 {
				owner = segments[len(segments)-2]
				repo = segments[len(segments)-1]
			}
		}
	}

	return owner, repo
}

func fallbackProjectSlug(workingDir string) string {
	cleaned := filepath.Clean(workingDir)
	if cleaned == "" || cleaned == "." {
		cleaned = workingDir
	}

	base := strings.ToLower(filepath.Base(cleaned))
	if base == "." || base == string(os.PathSeparator) || base == "" {
		base = "project"
	}

	slugBase := sanitizeSegment(base)
	if slugBase == "" {
		slugBase = "project"
	}

	hash := sha256.Sum256([]byte(cleaned))
	return fmt.Sprintf("%s-%s", slugBase, hex.EncodeToString(hash[:])[:6])
}

func sanitizeSegment(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	prevHyphen := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
			continue
		}
		if !prevHyphen {
			b.WriteRune('-')
			prevHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

