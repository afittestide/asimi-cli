package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	gokeyring "github.com/zalando/go-keyring"
)

const (
	keyringService = "asimi-cli"
	keyringPrefix  = "oauth_"
)

// TokenData holds OAuth token information
type TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
	Provider     string    `json:"provider"`
}

// SaveTokenToKeyring securely stores OAuth tokens in the OS keyring
func SaveTokenToKeyring(provider, accessToken, refreshToken string, expiry time.Time) error {
	data := TokenData{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       expiry,
		Provider:     provider,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	key := keyringPrefix + provider
	err = gokeyring.Set(keyringService, key, string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to store token in keyring: %w", err)
	}

	return nil
}

// GetOauthToken retrieves OAuth tokens from environment variable or OS keyring
func GetOauthToken(provider string) (*TokenData, error) {
	// First check if the env var `<provider>_OAUTH_TOKEN` exists
	envVarName := strings.ToUpper(provider) + "_OAUTH_TOKEN"
	if envToken := os.Getenv(envVarName); envToken != "" {
		// Env var holds the plain token text
		// Set a far-future expiry since we don't have refresh capability
		return &TokenData{
			AccessToken:  envToken,
			RefreshToken: "",
			Expiry:       time.Now().Add(365 * 24 * time.Hour),
			Provider:     provider,
		}, nil
	}

	// Fall back to keyring
	key := keyringPrefix + provider
	jsonData, err := gokeyring.Get(keyringService, key)
	if err != nil {
		if err == gokeyring.ErrNotFound {
			return nil, nil // Token not found is not an error
		}
		return nil, fmt.Errorf("failed to retrieve token from keyring: %w", err)
	}

	var data TokenData
	err = json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	return &data, nil
}

// GetTokenFromKeyring is an alias for GetOauthToken for backward compatibility
func GetTokenFromKeyring(provider string) (*TokenData, error) {
	return GetOauthToken(provider)
}

// DeleteTokenFromKeyring removes OAuth tokens from the OS keyring on logout
func DeleteTokenFromKeyring(provider string) error {
	key := keyringPrefix + provider
	err := gokeyring.Delete(keyringService, key)
	if err != nil && err != gokeyring.ErrNotFound {
		return fmt.Errorf("failed to delete token from keyring: %w", err)
	}
	return nil
}

// SaveAPIKeyToKeyring securely stores API keys in the OS keyring
func SaveAPIKeyToKeyring(provider, apiKey string) error {
	key := "apikey_" + provider
	err := gokeyring.Set(keyringService, key, apiKey)
	if err != nil {
		return fmt.Errorf("failed to store API key in keyring: %w", err)
	}
	return nil
}

// GetAPIKeyFromKeyring retrieves API keys from the OS keyring
func GetAPIKeyFromKeyring(provider string) (string, error) {
	key := "apikey_" + provider
	apiKey, err := gokeyring.Get(keyringService, key)
	if err != nil {
		if err == gokeyring.ErrNotFound {
			return "", nil // API key not found is not an error
		}
		return "", fmt.Errorf("failed to retrieve API key from keyring: %w", err)
	}
	return apiKey, nil
}

// DeleteAPIKeyFromKeyring removes API keys from the OS keyring
func DeleteAPIKeyFromKeyring(provider string) error {
	key := "apikey_" + provider
	err := gokeyring.Delete(keyringService, key)
	if err != nil && err != gokeyring.ErrNotFound {
		return fmt.Errorf("failed to delete API key from keyring: %w", err)
	}
	return nil
}

// IsTokenExpired checks if the token has expired
func IsTokenExpired(data *TokenData) bool {
	if data == nil {
		return true
	}
	// Add a 5-minute buffer before actual expiry
	return time.Now().After(data.Expiry.Add(-5 * time.Minute))
}
