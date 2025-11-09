package main

import (
	"testing"
	"time"
)

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
