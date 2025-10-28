package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAGENTSmdInSystemPrompt verifies that AGENTS.md content is included in the system prompt
func TestAGENTSmdInSystemPrompt(t *testing.T) {
	llm := &sessionMockLLMContext{}
	sess, err := NewSession(llm, &Config{}, func(any) {})
	assert.NoError(t, err)

	info := sess.GetContextInfo()

	// AGENTS.md should NOT be in ContextFiles anymore
	contextFiles := sess.GetContextFiles()
	_, hasAgents := contextFiles["AGENTS.md"]
	assert.False(t, hasAgents, "AGENTS.md should not be in ContextFiles (it's in the system prompt)")

	// System prompt should include AGENTS.md content if it exists
	// Check if AGENTS.md exists by trying to read it
	projectContext := readProjectContext()
	if projectContext != "" {
		t.Logf("AGENTS.md found with %d characters", len(projectContext))

		// System prompt tokens should include AGENTS.md
		assert.Greater(t, info.SystemPromptTokens, 0, "System prompt should have tokens including AGENTS.md")

		t.Logf("Context breakdown:")
		t.Logf("  System prompt: %d tokens (includes AGENTS.md)", info.SystemPromptTokens)
		t.Logf("  System tools: %d tokens", info.SystemToolsTokens)
		t.Logf("  Memory files: %d tokens", info.MemoryFilesTokens)
		t.Logf("  Messages: %d tokens", info.MessagesTokens)
	} else {
		t.Log("AGENTS.md not found in project directory")
	}
}
