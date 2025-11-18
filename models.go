package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AnthropicModel represents a model from the Anthropic API
type AnthropicModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
	Type        string `json:"type"`
}

// AnthropicModelsResponse represents the response from /v1/models endpoint
type AnthropicModelsResponse struct {
	Data    []AnthropicModel `json:"data"`
	FirstID string           `json:"first_id,omitempty"`
	LastID  string           `json:"last_id,omitempty"`
	HasMore bool             `json:"has_more"`
}

// fetchAnthropicModels fetches available models from the Anthropic API
func fetchAnthropicModels(config *Config) ([]AnthropicModel, error) {
	// Load credentials from keyring if not already in config
	// This ensures worktrees can access credentials stored in the OS keyring
	if config.LLM.AuthToken == "" && config.LLM.APIKey == "" {
		// Try OAuth tokens first
		tokenData, err := GetOauthToken(config.LLM.Provider)
		if err == nil && tokenData != nil {
			if !IsTokenExpired(tokenData) {
				// Token is still valid - use it
				config.LLM.AuthToken = tokenData.AccessToken
				config.LLM.RefreshToken = tokenData.RefreshToken
			} else {
				// Token expired - try to refresh for Anthropic
				if config.LLM.Provider == "anthropic" {
					auth := &AuthAnthropic{}
					newAccessToken, refreshErr := auth.access()
					if refreshErr == nil {
						config.LLM.AuthToken = newAccessToken
					}
				}
			}
		}

		// If still no auth token, try API key from keyring
		if config.LLM.AuthToken == "" {
			apiKey, err := GetAPIKeyFromKeyring(config.LLM.Provider)
			if err == nil && apiKey != "" {
				config.LLM.APIKey = apiKey
			}
		}
	}

	if config.LLM.AuthToken == "" && config.LLM.APIKey == "" {
		return nil, fmt.Errorf("no authentication configured for anthropic provider")
	}

	// Create HTTP client with appropriate authentication
	client := &http.Client{}
	if config.LLM.AuthToken != "" {
		// Use OAuth authentication
		client.Transport = &anthropicOAuthTransport{
			token:  config.LLM.AuthToken,
			config: config,
			base:   http.DefaultTransport,
		}
	} else {
		// Use API key authentication
		client.Transport = &anthropicAPIKeyTransport{
			base: http.DefaultTransport,
		}
	}

	// Determine base URL
	baseURL := "https://api.anthropic.com"
	if config.LLM.BaseURL != "" {
		baseURL = strings.TrimSuffix(config.LLM.BaseURL, "/")
	}
	if envBaseURL := os.Getenv("ANTHROPIC_BASE_URL"); envBaseURL != "" {
		baseURL = strings.TrimSuffix(envBaseURL, "/")
	}

	// Create request
	req, err := http.NewRequest("GET", baseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("anthropic-version", "2023-06-01")
	if config.LLM.APIKey != "" {
		req.Header.Set("x-api-key", config.LLM.APIKey)
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var modelsResponse AnthropicModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return modelsResponse.Data, nil
}

// ModelsWindow is a simplified component for displaying model selection
// Navigation is handled by ContentComponent
type ModelsWindow struct {
	width        int
	height       int
	models       []AnthropicModel
	currentModel string
	loading      bool
	errorMsg     string
	maxVisible   int
}

// NewModelsWindow creates a new models window
func NewModelsWindow() ModelsWindow {
	return ModelsWindow{
		width:        70,
		height:       15,
		models:       []AnthropicModel{},
		currentModel: "",
		loading:      false,
		errorMsg:     "",
		maxVisible:   8,
	}
}

// SetSize updates the dimensions
func (m *ModelsWindow) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Adjust maxVisible based on height
	m.maxVisible = height - 4 // Account for title, footer, instructions
	if m.maxVisible < 1 {
		m.maxVisible = 1
	}
}

// SetModels updates the models list
func (m *ModelsWindow) SetModels(models []AnthropicModel, currentModel string) {
	m.models = models
	m.currentModel = currentModel
	m.loading = false
	m.errorMsg = ""
}

// SetLoading sets loading state
func (m *ModelsWindow) SetLoading(loading bool) {
	m.loading = loading
	if loading {
		m.errorMsg = ""
	}
}

// SetError sets error state
func (m *ModelsWindow) SetError(err string) {
	m.errorMsg = err
	m.loading = false
}

// GetItemCount returns the number of models
func (m *ModelsWindow) GetItemCount() int {
	return len(m.models)
}

// GetVisibleSlots returns how many items can be shown at once
func (m *ModelsWindow) GetVisibleSlots() int {
	return m.maxVisible
}

// GetInitialSelection returns the index of the current model (or 0)
func (m *ModelsWindow) GetInitialSelection() int {
	for i, model := range m.models {
		if model.ID == m.currentModel {
			return i
		}
	}
	return 0
}

// GetSelectedModel returns the model at the given index
func (m *ModelsWindow) GetSelectedModel(index int) *AnthropicModel {
	if index < 0 || index >= len(m.models) {
		return nil
	}
	return &m.models[index]
}

// RenderList renders the models list with the given selection
func (m *ModelsWindow) RenderList(selectedIndex, scrollOffset, visibleSlots int) string {
	var content strings.Builder

	if m.loading {
		content.WriteString("Loading models...\n\n")
		content.WriteString("⏳ Fetching available models from Anthropic API...")
		return content.String()
	}

	if m.errorMsg != "" {
		content.WriteString("Error loading models:\n\n")
		content.WriteString(m.errorMsg + "\n\n")
		return content.String()
	}

	if len(m.models) == 0 {
		content.WriteString("No models available\n")
		return content.String()
	}

	start := scrollOffset
	end := scrollOffset + visibleSlots
	if end > len(m.models) {
		end = len(m.models)
	}

	for i := start; i < end; i++ {
		model := m.models[i]
		prefix := "  "
		suffix := ""

		// Mark current model
		if model.ID == m.currentModel {
			suffix = " (current)"
		}

		// Mark selected model
		if i == selectedIndex {
			prefix = "▶ "
		}

		style := lipgloss.NewStyle()
		if i == selectedIndex {
			style = style.Foreground(lipgloss.Color("62")).Bold(true)
		}

		displayText := model.DisplayName
		if displayText == "" {
			displayText = model.ID
		}

		line := fmt.Sprintf("%s%s%s", prefix, displayText, suffix)
		content.WriteString(style.Render(line) + "\n")
	}

	// Show scroll indicator if needed
	if len(m.models) > visibleSlots {
		scrollInfo := fmt.Sprintf("\n%d-%d of %d models", start+1, end, len(m.models))
		scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
		content.WriteString(scrollStyle.Render(scrollInfo))
	}

	return content.String()
}

// Message types for model selection
type modelSelectedMsg struct {
	model *AnthropicModel
}

// Message types for model loading
type modelsLoadedMsg struct {
	models []AnthropicModel
}

type modelsLoadErrorMsg struct {
	error string
}

type showModelSelectionMsg struct{}

// Command handler
func handleModelsCommand(model *TUIModel, args []string) tea.Cmd {
	// Only allow model selection for Anthropic provider
	name := model.config.LLM.Provider
	if name != "anthropic" {
		model.commandLine.AddToast("Model selection is not available for "+name, "error", 3000)
		return nil
	}

	return func() tea.Msg {
		return showModelSelectionMsg{}
	}
}

// TUI command to fetch models
func (m *TUIModel) fetchModelsCommand() tea.Cmd {
	return func() tea.Msg {
		models, err := fetchAnthropicModels(m.config)
		if err != nil {
			return modelsLoadErrorMsg{error: err.Error()}
		}
		return modelsLoadedMsg{models: models}
	}
}
