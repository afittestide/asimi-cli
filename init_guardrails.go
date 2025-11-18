package main

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

// runInitGuardrails executes all guardrail checks and returns a comprehensive report
// This is a wrapper that uses the GuardrailsService
func runInitGuardrails(guardrails *GuardrailsService, update func(string)) string {
	ctx := context.Background()
	return guardrails.RunAll(ctx, update)
}

// initGuardrailsCompleteMsg is sent when guardrails complete
type initGuardrailsCompleteMsg struct {
	report string
}

// runInitGuardrailsAsync runs guardrails in a goroutine and sends result
func runInitGuardrailsAsync(guardrails *GuardrailsService, update func(string)) tea.Cmd {
	return func() tea.Msg {
		report := runInitGuardrails(guardrails, update)
		return initGuardrailsCompleteMsg{report: report}
	}
}

// contains checks if a string slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
