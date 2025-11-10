package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
)

func TestNewResumeWindowDefaults(t *testing.T) {
	window := NewResumeWindow()

	assert.Equal(t, 70, window.width)
	assert.Equal(t, 15, window.height)
	assert.Equal(t, 8, window.maxVisible)
	assert.False(t, window.loading)
	assert.Empty(t, window.sessions)
	assert.Nil(t, window.errorMsg)
}

func TestResumeWindowSetSizeAdjustsVisibleSlots(t *testing.T) {
	window := NewResumeWindow()

	window.SetSize(80, 10)
	assert.Equal(t, 80, window.width)
	assert.Equal(t, 10, window.height)
	assert.Equal(t, 6, window.maxVisible)

	window.SetSize(50, 2)
	assert.Equal(t, 1, window.maxVisible) // min clamp
}

func TestResumeWindowSetSessionsAndRender(t *testing.T) {
	window := NewResumeWindow()
	now := time.Now()

	sessions := []Session{
		testSession("s-1", "Refactor prompt", now, "Need to refactor"),
		testSession("s-2", "Investigate bug", now.Add(-2*time.Hour), "Bug details"),
	}

	window.SetSessions(sessions)
	assert.False(t, window.loading)
	assert.Equal(t, 2, window.GetItemCount())

	render := window.RenderList(0, 0, window.GetVisibleSlots())
	assert.Contains(t, render, sessionTitlePreview(sessions[0]))
	assert.Contains(t, render, sessionTitlePreview(sessions[1]))
	assert.Contains(t, render, "▶ ")
	assert.Contains(t, render, "1 msg")
}

func TestResumeWindowLoadingAndErrorStates(t *testing.T) {
	window := NewResumeWindow()

	window.SetLoading(true)
	assert.Contains(t, window.RenderList(0, 0, window.GetVisibleSlots()), "Loading sessions")

	window.SetLoading(false)
	window.SetError(assert.AnError)
	render := window.RenderList(0, 0, window.GetVisibleSlots())
	assert.Contains(t, render, "Error loading sessions")
	assert.NotContains(t, render, "Loading sessions")
}

func TestResumeWindowEmptyState(t *testing.T) {
	window := NewResumeWindow()
	window.SetSessions(nil)

	render := window.RenderList(0, 0, window.GetVisibleSlots())
	assert.Contains(t, render, "No previous sessions found")
	assert.Contains(t, render, "Start chatting to create a new session")
}

func TestResumeWindowScrollInfo(t *testing.T) {
	window := NewResumeWindow()
	now := time.Now()

	var sessions []Session
	for i := 0; i < 20; i++ {
		sessions = append(sessions, testSession(
			fmt.Sprintf("s-%d", i+1),
			fmt.Sprintf("Prompt %d", i+1),
			now.Add(-time.Duration(i)*time.Minute),
			fmt.Sprintf("Message %d", i+1),
		))
	}

	window.SetSessions(sessions)
	render := window.RenderList(5, 5, 5)

	assert.Contains(t, render, "6-10 of 20 sessions")
	assert.Contains(t, render, "Message 6")
	assert.NotContains(t, render, "Message 2") // scrolled past
}

func TestResumeWindowGetSelectedSession(t *testing.T) {
	window := NewResumeWindow()
	window.SetSessions([]Session{
		testSession("one", "First", time.Now(), "msg"),
	})

	assert.Nil(t, window.GetSelectedSession(-1))
	assert.Nil(t, window.GetSelectedSession(2))

	session := window.GetSelectedSession(0)
	assert.NotNil(t, session)
	assert.Equal(t, "one", session.ID)
}

func TestSessionTitlePreviewFallbacks(t *testing.T) {
	session := testSession("s-1", "", time.Now(), "")
	session.Messages = nil
	assert.Equal(t, "Recent activity", sessionTitlePreview(session))

	session.FirstPrompt = " initial "
	assert.Equal(t, "initial", sessionTitlePreview(session))

	session.Messages = []llms.MessageContent{
		textMessage(llms.ChatMessageTypeHuman, "User question"),
	}
	assert.Equal(t, "User question", sessionTitlePreview(session))
}

func TestFormatMessageCount(t *testing.T) {
	assert.Equal(t, "", formatMessageCount(nil))

	msgs := []llms.MessageContent{
		textMessage(llms.ChatMessageTypeHuman, "hi"),
		textMessage(llms.ChatMessageTypeAI, "hello"),
	}
	assert.Equal(t, "2 msgs", formatMessageCount(msgs))

	one := []llms.MessageContent{
		textMessage(llms.ChatMessageTypeHuman, "single"),
	}
	assert.Equal(t, "1 msg", formatMessageCount(one))
}

func testSession(id, prompt string, updated time.Time, messageTexts ...string) Session {
	var messages []llms.MessageContent
	for _, text := range messageTexts {
		messages = append(messages, textMessage(llms.ChatMessageTypeHuman, text))
	}

	return Session{
		ID:          id,
		FirstPrompt: prompt,
		LastUpdated: updated,
		Messages:    messages,
		Model:       "test",
	}
}

func textMessage(role llms.ChatMessageType, text string) llms.MessageContent {
	return llms.MessageContent{
		Role: role,
		Parts: []llms.ContentPart{
			llms.TextContent{Text: text},
		},
	}
}
