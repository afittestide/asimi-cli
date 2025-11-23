package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/afittestide/asimi/storage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tmc/langchaingo/llms"
)

type sessionsLoadedMsg struct {
	sessions []Session
}

type sessionSelectedMsg struct {
	session *Session
}

type sessionResumeErrorMsg struct {
	err error
}

// ResumeWindow is a simplified component for displaying session selection
// Navigation is handled by ContentComponent
type ResumeWindow struct {
	width          int
	height         int
	sessions       []Session
	loading        bool
	loadingSession bool
	errorMsg       error
	maxVisible     int
}

func NewResumeWindow() ResumeWindow {
	return ResumeWindow{
		width:          70,
		height:         15,
		sessions:       []Session{},
		loading:        false,
		loadingSession: false,
		errorMsg:       nil,
		maxVisible:     8,
	}
}

func (r *ResumeWindow) SetSize(width, height int) {
	r.width = width
	r.height = height
	// Adjust maxVisible based on trial & error
	r.maxVisible = height - 3
	if r.maxVisible < 1 {
		r.maxVisible = 1
	}
}

func (r *ResumeWindow) GetVisibleSlots() int {
	return r.maxVisible
}

func (r *ResumeWindow) SetSessions(sessions []Session) {
	r.sessions = sessions
	r.loading = false
	r.loadingSession = false
	r.errorMsg = nil
}

func (r *ResumeWindow) SetLoading(loading bool) {
	r.loading = loading
	if loading {
		r.errorMsg = nil
	}
}

func (r *ResumeWindow) SetError(err error) {
	r.errorMsg = err
	r.loading = false
	r.loadingSession = false
}

func (r *ResumeWindow) GetItemCount() int {
	return len(r.sessions)
}

func (r *ResumeWindow) GetSelectedSession(index int) *Session {
	if index < 0 || index >= len(r.sessions) {
		return nil
	}
	return &r.sessions[index]
}

func sessionTitlePreview(session Session) string {

	snippet := session.FirstPrompt
	if len(session.Messages) > 0 {
		snippet = lastHumanMessage(session.Messages)
	}

	snippet = cleanSnippet(snippet)
	if snippet == "" {
		return "Recent activity"
	}
	return truncateSnippet(snippet, 60)
}

func lastHumanMessage(messages []llms.MessageContent) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != llms.ChatMessageTypeHuman {
			continue
		}
		for _, part := range messages[i].Parts {
			if textPart, ok := part.(llms.TextContent); ok {
				text := strings.TrimSpace(textPart.Text)
				if text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func cleanSnippet(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "---") && strings.HasSuffix(trimmed, "---") {
			continue
		}
		if strings.HasPrefix(trimmed, "Context from:") {
			continue
		}
		return trimmed
	}

	return strings.TrimSpace(lines[0])
}

func truncateSnippet(text string, limit int) string {
	if limit <= 0 {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}

	if limit <= 3 {
		return string(runes[:limit])
	}

	return string(runes[:limit-3]) + "..."
}

// RenderList renders the session list with the given selection
func (r *ResumeWindow) RenderList(selectedIndex, scrollOffset, visibleSlots int) string {
	var content strings.Builder

	if r.loading {
		content.WriteString("Loading sessions...\n")
		return content.String()
	}

	if r.loadingSession {
		content.WriteString("Loading selected session...\n")
		content.WriteString("Please wait...")
		return content.String()
	}

	if r.errorMsg != nil {
		content.WriteString(fmt.Sprintf("Error loading sessions: %v\n\n", r.errorMsg))
		return content.String()
	}

	if len(r.sessions) == 0 {
		content.WriteString("No previous sessions found.\n")
		content.WriteString("Start chatting to create a new session!\n\n")
		return content.String()
	}

	totalItems := len(r.sessions)
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	maxOffset := totalItems - visibleSlots
	if maxOffset < 0 {
		maxOffset = 0
	}
	if scrollOffset > maxOffset {
		scrollOffset = maxOffset
	}
	start := scrollOffset
	end := scrollOffset + visibleSlots
	if end > totalItems {
		end = totalItems
	}

	for i := start; i < end; i++ {
		isSelected := i == selectedIndex
		session := r.sessions[i]

		prefix := "  "
		if isSelected {
			prefix = "▶ "
		}

		timeStr := formatRelativeTime(session.LastUpdated)
		title := sessionTitlePreview(session)

		var line strings.Builder
		line.WriteString(prefix)
		line.WriteString(fmt.Sprintf("[%s] %4d %s", timeStr, session.MessageCount, title))

		lineStyle := lipgloss.NewStyle()
		if isSelected {
			lineStyle = lineStyle.Foreground(lipgloss.Color("62")).Bold(true)
		}

		content.WriteString(lineStyle.Render(line.String()))
		content.WriteString("\n")
	}

	return content.String()
}

// LoadSession loads a session by ID
func (r *ResumeWindow) LoadSession(sessionID string) tea.Cmd {
	r.loadingSession = true

	return func() tea.Msg {
		config, err := LoadConfig()
		if err != nil {
			return sessionResumeErrorMsg{err: fmt.Errorf("failed to load config: %w", err)}
		}

		// Initialize storage
		db, err := storage.InitDB(config.Storage.DatabasePath)
		if err != nil {
			return sessionResumeErrorMsg{err: fmt.Errorf("failed to initialize storage: %w", err)}
		}
		defer db.Close()

		maxSessions := 50
		maxAgeDays := 30
		if config.Session.MaxSessions > 0 {
			maxSessions = config.Session.MaxSessions
		}
		if config.Session.MaxAgeDays > 0 {
			maxAgeDays = config.Session.MaxAgeDays
		}

		repoInfo := GetRepoInfo()
		store, err := NewSessionStore(db, repoInfo, maxSessions, maxAgeDays)
		if err != nil {
			return sessionResumeErrorMsg{err: fmt.Errorf("failed to create session store: %w", err)}
		}
		defer store.Close()

		// Load the session
		session, err := store.LoadSession(sessionID)
		if err != nil {
			return sessionResumeErrorMsg{err: fmt.Errorf("failed to load session: %w", err)}
		}

		if session == nil {
			return sessionResumeErrorMsg{err: fmt.Errorf("session %s not found", sessionID)}
		}

		return sessionSelectedMsg{session: session}
	}
}

func formatRelativeTime(t time.Time) string {
	now := time.Now()

	// Normalize to midnight for calendar day comparison
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tMidnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	daysDiff := int(todayMidnight.Sub(tMidnight).Hours() / 24)

	// Today
	if daysDiff == 0 {
		return fmt.Sprintf("Today %s", t.Format("15:04"))
	}

	// Yesterday
	if daysDiff == 1 {
		return fmt.Sprintf("Yesterday %s", t.Format("15:04"))
	}

	// This year - show month and day
	if t.Year() == now.Year() {
		return t.Format("Jan 2 15:04")
	}

	// Older - show full date
	return t.Format("Jan 2, 2006")
}
