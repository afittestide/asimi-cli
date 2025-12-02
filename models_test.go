package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestMain sets up test environment
func TestMain(m *testing.M) {
	// Use a test-specific keyring service to avoid polluting production credentials
	original := os.Getenv("ASIMI_KEYRING_SERVICE")
	os.Setenv("ASIMI_KEYRING_SERVICE", "dev.asimi.asimi-cli-test")

	// Update the global keyringService variable
	keyringService = getKeyringService()

	defer func() {
		os.Setenv("ASIMI_KEYRING_SERVICE", original)
		keyringService = getKeyringService()
	}()

	// Run tests
	code := m.Run()

	// Exit with test result code
	os.Exit(code)
}

// TestFetchAllModels_ReturnsEmptyWithoutAuth verifies that fetchAllModels returns
// an empty list when no providers are authenticated
func TestFetchAllModels_ReturnsEmptyWithoutAuth(t *testing.T) {
	// Clear any existing credentials
	DeleteTokenFromKeyring("anthropic")
	DeleteAPIKeyFromKeyring("anthropic")
	DeleteTokenFromKeyring("openai")
	DeleteAPIKeyFromKeyring("openai")
	DeleteTokenFromKeyring("googleai")
	DeleteAPIKeyFromKeyring("googleai")

	// Clear environment variables for this test
	originalAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	originalOpenAIKey := os.Getenv("OPENAI_API_KEY")
	originalGeminiKey := os.Getenv("GEMINI_API_KEY")
	originalGoogleKey := os.Getenv("GOOGLE_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("GOOGLE_API_KEY")
	defer func() {
		if originalAnthropicKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", originalAnthropicKey)
		}
		if originalOpenAIKey != "" {
			os.Setenv("OPENAI_API_KEY", originalOpenAIKey)
		}
		if originalGeminiKey != "" {
			os.Setenv("GEMINI_API_KEY", originalGeminiKey)
		}
		if originalGoogleKey != "" {
			os.Setenv("GOOGLE_API_KEY", originalGoogleKey)
		}
	}()

	config := &Config{
		LLM: LLMConfig{
			Provider: "anthropic",
			Model:    "claude-3-5-sonnet-latest",
		},
	}

	models := fetchAllModels(config)

	assert.Equal(t, 1, len(models))
	assert.Contains(t, models[0].DisplayName, "Login")
}

// TestFetchAllModels_WithAPIKey verifies that models show as ready when API key is available
// or error items are added when API fails
func TestFetchAllModels_WithAPIKey(t *testing.T) {
	// Clear any existing credentials
	DeleteTokenFromKeyring("openai")
	DeleteAPIKeyFromKeyring("openai")

	// Set OpenAI API key in environment
	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	config := &Config{
		LLM: LLMConfig{
			Provider: "openai",
			Model:    "gpt-4o",
		},
	}

	models := fetchAllModels(config)

	// With an API key set, we should get either models or an error item
	hasOpenAI := false
	for _, m := range models {
		if m.Provider == "openai" {
			hasOpenAI = true
			// Should be ready, active, or error (if API call failed)
			if m.Status != "ready" && m.Status != "active" && m.Status != "error" {
				t.Errorf("Expected OpenAI model %s to be 'ready', 'active', or 'error', got %s", m.ID, m.Status)
			}
		}
	}

	if !hasOpenAI {
		t.Error("Expected at least one OpenAI item (model or error)")
	}
}

// TestCheckProviderAuth verifies provider authentication detection
func TestCheckProviderAuth(t *testing.T) {
	// Clear credentials
	DeleteTokenFromKeyring("anthropic")
	DeleteAPIKeyFromKeyring("anthropic")

	// Clear environment
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		}
	}()

	// Test with no credentials
	info := checkProviderAuth("anthropic")
	if info.HasAPIKey || info.HasOAuth {
		t.Error("Expected no auth when no credentials are set")
	}

	// Test with environment variable
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	info = checkProviderAuth("anthropic")
	if !info.HasAPIKey {
		t.Error("Expected HasAPIKey to be true when ANTHROPIC_API_KEY is set")
	}
}

// TestFetchAnthropicModels_LoadsFromKeyring verifies that fetchAnthropicModels
// loads credentials from the keyring when they're not in the config
func TestFetchAnthropicModels_LoadsFromKeyring(t *testing.T) {
	// This test verifies the bug fix for worktrees
	// When running in a worktree, the config file might not have auth tokens,
	// but they should be loaded from the OS keyring

	// Create a config without auth tokens
	config := &Config{
		LLM: LLMConfig{
			Provider: "anthropic",
			Model:    "claude-3-5-sonnet-20241022",
			// Intentionally leave AuthToken and APIKey empty
		},
	}

	// Save a test token to the keyring
	testToken := "test-token-12345"
	testRefreshToken := "test-refresh-token"
	expiry := time.Now().Add(24 * time.Hour)

	err := SaveTokenToKeyring("anthropic", testToken, testRefreshToken, expiry)
	if err != nil {
		t.Skipf("Skipping test: keyring not available: %v", err)
	}
	defer DeleteTokenFromKeyring("anthropic")

	// Call fetchAnthropicModels - it should load the token from keyring
	// Note: This will fail with a network error since we're using a fake token,
	// but we're just testing that it loads the token from keyring
	_, err = fetchAnthropicModels(config)

	// Verify that the config was updated with the token from keyring
	if config.LLM.AuthToken != testToken {
		t.Errorf("Expected AuthToken to be loaded from keyring, got %q, want %q",
			config.LLM.AuthToken, testToken)
	}

	if config.LLM.RefreshToken != testRefreshToken {
		t.Errorf("Expected RefreshToken to be loaded from keyring, got %q, want %q",
			config.LLM.RefreshToken, testRefreshToken)
	}
}

// TestFetchAnthropicModels_LoadsAPIKeyFromKeyring verifies that fetchAnthropicModels
// loads API keys from the keyring as a fallback
func TestFetchAnthropicModels_LoadsAPIKeyFromKeyring(t *testing.T) {
	// Create a config without auth tokens
	config := &Config{
		LLM: LLMConfig{
			Provider: "anthropic",
			Model:    "claude-3-5-sonnet-20241022",
		},
	}

	// Save a test API key to the keyring
	testAPIKey := "sk-ant-test-key-12345"

	err := SaveAPIKeyToKeyring("anthropic", testAPIKey)
	if err != nil {
		t.Skipf("Skipping test: keyring not available: %v", err)
	}
	defer DeleteAPIKeyFromKeyring("anthropic")

	// Call fetchAnthropicModels - it should load the API key from keyring
	_, err = fetchAnthropicModels(config)

	// Verify that the config was updated with the API key from keyring
	if config.LLM.APIKey != testAPIKey {
		t.Errorf("Expected APIKey to be loaded from keyring, got %q, want %q",
			config.LLM.APIKey, testAPIKey)
	}
}

// TestFetchAnthropicModels_NoCredentials verifies error when no credentials available
func TestFetchAnthropicModels_NoCredentials(t *testing.T) {
	// Create a config without auth tokens
	config := &Config{
		LLM: LLMConfig{
			Provider: "anthropic",
			Model:    "claude-3-5-sonnet-20241022",
		},
	}

	// Make sure no credentials are in keyring
	DeleteTokenFromKeyring("anthropic")
	DeleteAPIKeyFromKeyring("anthropic")

	// Clear environment variable
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		}
	}()

	// Call fetchAnthropicModels - it should return an error
	_, err := fetchAnthropicModels(config)

	if err == nil {
		t.Error("Expected error when no credentials available, got nil")
	}

	expectedError := "no authentication configured for anthropic provider"
	if err.Error() != expectedError {
		t.Errorf("Expected error %q, got %q", expectedError, err.Error())
	}
}
