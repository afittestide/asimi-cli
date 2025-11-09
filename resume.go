package main

import (
	"fmt"
	"strings"

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

type SessionSelectionModal struct {
	*BaseModal
	sessions       []Session
	selected       int
	scrollOffset   int
	loading        bool
	loadingSession bool
	err            error
	sessionStore   *SessionStore
}

func NewSessionSelectionModal(store *SessionStore) *SessionSelectionModal {
	baseModal := NewBaseModal("Resume Session", "", 70, 15)

	return &SessionSelectionModal{
		BaseModal:      baseModal,
		sessions:       []Session{},
		selected:       0,
		scrollOffset:   0,
		loading:        true,
		loadingSession: false,
		err:            nil,
		sessionStore:   store,
	}
}

func (m *SessionSelectionModal) visibleSlots() int {
	if m.BaseModal == nil {
		return 1
	}

	contentHeight := m.BaseModal.Height - 4 // account for title and borders
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Reserve lines for instructions and spacing (2 lines for instructions + 1 line for scroll info)
	available := contentHeight - 3
	if available < 1 {
		return 1
	}

	return available
}

func (m *SessionSelectionModal) SetSessions(sessions []Session) {
	m.sessions = sessions
	m.loading = false
	m.loadingSession = false
	m.err = nil
}

func (m *SessionSelectionModal) SetError(err error) {
	m.err = err
	m.loading = false
	m.loadingSession = false
}

func sessionTitlePreview(session Session) string {
	snippet := lastHumanMessage(session.Messages)
	if snippet == "" {
		snippet = session.FirstPrompt
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

func formatMessageCount(messages []llms.MessageContent) string {
	count := 0
	for _, msg := range messages {
		if msg.Role == llms.ChatMessageTypeHuman || msg.Role == llms.ChatMessageTypeAI {
			count++
		}
	}

	if count == 0 {
		return ""
	}
	if count == 1 {
		return "1 msg"
	}
	return fmt.Sprintf("%d msgs", count)
}

func (m *SessionSelectionModal) Render() string {
	var content strings.Builder

	if m.loading {
		content.WriteString("Loading sessions...\n")
		m.BaseModal.Content = content.String()
		return m.BaseModal.Render()
	}

	if m.loadingSession {
		content.WriteString("Loading selected session...\n")
		content.WriteString("Please wait...")
		m.BaseModal.Content = content.String()
		return m.BaseModal.Render()
	}

	if m.err != nil {
		content.WriteString(fmt.Sprintf("Error loading sessions: %v\n\n", m.err))
		content.WriteString("Press Esc to close")
		m.BaseModal.Content = content.String()
		return m.BaseModal.Render()
	}

	if len(m.sessions) == 0 {
		content.WriteString("No previous sessions found.\n")
		content.WriteString("Start chatting to create a new session!\n\n")
		content.WriteString("Press Esc to close")
		m.BaseModal.Content = content.String()
		return m.BaseModal.Render()
	}

	instructionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	content.WriteString(instructionStyle.Render("↑/↓: Navigate • 1-9: Quick select • Enter: Select • Esc/Q: Cancel"))
	content.WriteString("\n\n")

	// Total items = sessions + cancel option
	totalItems := len(m.sessions) + 1

	visible := m.visibleSlots()
	if visible < 1 {
		visible = 1
	}
	if visible > totalItems {
		visible = totalItems
	}

	maxOffset := totalItems - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= totalItems {
		m.selected = totalItems - 1
	}
	if visible > 0 {
		if m.selected < m.scrollOffset {
			m.scrollOffset = m.selected
		} else if m.selected >= m.scrollOffset+visible {
			m.scrollOffset = m.selected - visible + 1
		}
	}

	start := m.scrollOffset
	end := m.scrollOffset + visible
	if end > totalItems {
		end = totalItems
	}

	for i := start; i < end; i++ {
		isSelected := i == m.selected

		// Check if this is the cancel option (last item)
		if i == len(m.sessions) {
			prefix := "   "
			if isSelected {
				prefix = "▶ "
			}

			var line strings.Builder
			line.WriteString(prefix)
			line.WriteString("Cancel")

			lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			if isSelected {
				lineStyle = lineStyle.Foreground(lipgloss.Color("62")).Bold(true)
			}

			content.WriteString(lineStyle.Render(line.String()))
			content.WriteString("\n")
			continue
		}

		session := m.sessions[i]

		prefix := fmt.Sprintf(" %d. ", i+1)
		if isSelected {
			prefix = fmt.Sprintf("▶%d. ", i+1)
		}

		timeStr := formatRelativeTime(session.LastUpdated)
		title := sessionTitlePreview(session)
		messageCount := formatMessageCount(session.Messages)

		var line strings.Builder
		line.WriteString(prefix)
		line.WriteString(fmt.Sprintf("[%s] %s", timeStr, title))

		if messageCount != "" {
			line.WriteString(fmt.Sprintf(" (%s)", messageCount))
		}

		lineStyle := lipgloss.NewStyle()
		if isSelected {
			lineStyle = lineStyle.Foreground(lipgloss.Color("62")).Bold(true)
		}

		content.WriteString(lineStyle.Render(line.String()))
		content.WriteString("\n")
	}

	if totalItems > visible {
		scrollInfo := fmt.Sprintf("\n%d-%d of %d items", start+1, end, totalItems)
		scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
		content.WriteString(scrollStyle.Render(scrollInfo))
	}

	m.BaseModal.Content = content.String()
	return m.BaseModal.Render()
}

func (m *SessionSelectionModal) Update(msg tea.Msg) (*SessionSelectionModal, tea.Cmd) {
	if m.loading || m.loadingSession || m.err != nil || len(m.sessions) == 0 {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "esc" || keyMsg.String() == "q" {
				return m, func() tea.Msg { return modalCancelledMsg{} }
			}
		}
		return m, nil
	}

	// Total items = sessions + cancel option
	totalItems := len(m.sessions) + 1
	visible := m.visibleSlots()
	if visible < 1 {
		visible = 1
	}
	maxOffset := totalItems - visible
	if maxOffset < 0 {
		maxOffset = 0
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
				if m.selected < m.scrollOffset {
					m.scrollOffset = m.selected
				}
				if m.scrollOffset < 0 {
					m.scrollOffset = 0
				}
			}
		case "down", "j":
			if m.selected < totalItems-1 {
				m.selected++
				if m.selected >= m.scrollOffset+visible {
					m.scrollOffset = m.selected - visible + 1
				}
				if m.scrollOffset > maxOffset {
					m.scrollOffset = maxOffset
				}
			}
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			num := int(msg.String()[0] - '1')
			if num < len(m.sessions) {
				m.selected = num
				m.loadingSession = true
				return m, m.loadSelectedSession()
			}
		case "enter":
			// If cancel option is selected (last item)
			if m.selected == len(m.sessions) {
				return m, func() tea.Msg { return modalCancelledMsg{} }
			}
			m.loadingSession = true
			return m, m.loadSelectedSession()
		case "esc", "q":
			return m, func() tea.Msg { return modalCancelledMsg{} }
		}
	}

	return m, nil
}

func (m *SessionSelectionModal) loadSelectedSession() tea.Cmd {
	if len(m.sessions) == 0 || m.selected < 0 || m.selected >= len(m.sessions) {
		return func() tea.Msg { return modalCancelledMsg{} }
	}

	sessionID := m.sessions[m.selected].ID
	store := m.sessionStore
	if store == nil {
		return func() tea.Msg {
			return sessionResumeErrorMsg{err: fmt.Errorf("session store not initialized")}
		}
	}

	return func() tea.Msg {
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
