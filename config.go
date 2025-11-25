package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	koanftoml "github.com/knadh/koanf/parsers/toml/v2"
	koanfenv "github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	koanf "github.com/knadh/koanf/v2"
)

type oauthProviderConfig struct {
	AuthURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// Config represents the application configuration structure
type Config struct {
	Storage    StorageConfig    `koanf:"storage"`
	Logging    LoggingConfig    `koanf:"logging"`
	UI         UIConfig         `koanf:"ui"`
	LLM        LLMConfig        `koanf:"llm"`
	History    HistoryConfig    `koanf:"history"`
	Session    SessionConfig    `koanf:"session"`
	Container  ContainerConfig  `koanf:"container"`
	RunInShell RunInShellConfig `koanf:"run_in_shell"`
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	DatabasePath string `koanf:"database_path"` // Path to SQLite database
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

// LLMConfig holds LLM configuration
type LLMConfig struct {
	Provider                   string `koanf:"provider"`
	Model                      string `koanf:"model"`
	APIKey                     string `koanf:"api_key"`
	BaseURL                    string `koanf:"base_url"`
	MaxThinkingTokens          int    `koanf:"max_thinking_tokens"`
	MaxTurns                   int    `koanf:"max_turns"`
	DisableContextSanitization bool   `koanf:"disable_sanitization"`
	AuthToken                  string `koanf:"auth_token"`
	RefreshToken               string `koanf:"refresh_token"`
}

// HistoryConfig holds persistent session history configuration
type HistoryConfig struct {
	Enabled      bool `koanf:"enabled"`
	MaxSessions  int  `koanf:"max_sessions"`
	MaxAgeDays   int  `koanf:"max_age_days"`
	ListLimit    int  `koanf:"list_limit"`
	AutoSave     bool `koanf:"auto_save"`
	SaveInterval int  `koanf:"save_interval"`
}

// UIConfig holds UI-specific configuration
type UIConfig struct {
	MarkdownEnabled bool `koanf:"markdown_enabled"`
}

// defaultConfig returns the configuration populated with sensible defaults.
func defaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	dbPath := filepath.Join(homeDir, ".local", "share", "asimi", "asimi.sqlite")

	return Config{
		Storage: StorageConfig{
			DatabasePath: dbPath,
		},
		History: HistoryConfig{
			Enabled:      true,
			MaxSessions:  50,
			MaxAgeDays:   30,
			ListLimit:    0,
			AutoSave:     false,
			SaveInterval: 300,
		},
		UI: UIConfig{
			MarkdownEnabled: false,
		},
		Session: SessionConfig{
			Enabled:      true,
			MaxSessions:  50,
			MaxAgeDays:   30,
			ListLimit:    0,
			AutoSave:     true,
			SaveInterval: 300,
		},
		RunInShell: RunInShellConfig{
			RunOnHost: []string{`^gh\s.*`},
		},
	}
}

// SessionConfig holds session persistence configuration
type SessionConfig struct {
	Enabled      bool `koanf:"enabled"`
	MaxSessions  int  `koanf:"max_sessions"`
	MaxAgeDays   int  `koanf:"max_age_days"`
	ListLimit    int  `koanf:"list_limit"`
	AutoSave     bool `koanf:"auto_save"`
	SaveInterval int  `koanf:"save_interval"`
}

// ContainerMount represents a mount point for the container
type ContainerMount struct {
	Source      string `koanf:"source"`
	Destination string `koanf:"destination"`
}

// ContainerConfig holds container configuration
type ContainerConfig struct {
	AdditionalMounts []ContainerMount `koanf:"additional_mounts"`
}

// RunInShellConfig holds configuration for the run_in_shell tool
type RunInShellConfig struct {
	// RunOnHost is a list of regex patterns for commands that should run on the host
	// instead of in the container
	RunOnHost []string `koanf:"run_on_host"`
	// TimeoutMinutes is the timeout for shell commands in minutes (default: 10)
	TimeoutMinutes    int  `koanf:"timeout_minutes"`
	AllowHostFallback bool `koanf:"allow_host_fallback"`
	NoCleanup         bool `koanf:"no_cleanup"`
}

// LoadConfig loads configuration from multiple sources
func LoadConfig() (*Config, error) {
	// Create a new koanf instance
	k := koanf.New(".")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home directory: %v", err)
	} else {
		userConfigPath := filepath.Join(homeDir, ".config", "asimi", "conf.toml")
		if err := k.Load(file.Provider(userConfigPath), koanftoml.Parser()); err != nil {
			log.Printf("Failed to load user config from %s: %v", userConfigPath, err)
		}
	}

	projectConfigPath := filepath.Join(".agents", "asimi.toml")
	if _, err := os.Stat(projectConfigPath); err == nil {
		if err := k.Load(file.Provider(projectConfigPath), koanftoml.Parser()); err != nil {
			log.Printf("Failed to load project config from %s: %v", projectConfigPath, err)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("Unable to stat project config at %s: %v", projectConfigPath, err)
	}

	// 3. Load environment variables
	// Environment variables with prefix "ASIMI_" will override config values
	// e.g., ASIMI_SERVER_PORT=8080 will override the server port
	if err := k.Load(koanfenv.Provider(".", koanfenv.Opt{
		Prefix: "ASIMI_",
		TransformFunc: func(key, value string) (string, any) {
			// Transform environment variable names to match config keys
			// ASIMI_SERVER_PORT becomes "server.port"
			key = strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(key, "ASIMI_")), "_", ".")
			return key, value
		},
	}), nil); err != nil {
		log.Printf("Failed to load environment variables: %v", err)
	}

	// Special handling for API keys from standard environment variables
	// Check for OPENAI_API_KEY if using OpenAI
	if k.String("llm.provider") == "openai" && k.String("llm.api_key") == "" {
		if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
			if err := k.Set("llm.api_key", openaiKey); err != nil {
				log.Printf("Failed to set OpenAI API key from environment: %v", err)
			}
		}
	}

	// Check for ANTHROPIC_API_KEY if using Anthropic
	if k.String("llm.provider") == "anthropic" && k.String("llm.api_key") == "" {
		if anthropicKey := os.Getenv("ANTHROPIC_API_KEY"); anthropicKey != "" {
			if err := k.Set("llm.api_key", anthropicKey); err != nil {
				log.Printf("Failed to set Anthropic API key from environment: %v", err)
			}
		}
	}

	// Unmarshal the configuration into our struct
	config := defaultConfig()
	if err := k.Unmarshal("", &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set default values for session config if not explicitly configured
	// Check if session.enabled was explicitly set in config or environment
	if !k.Exists("session.enabled") {
		config.Session.Enabled = true // Default to enabled
	}

	return &config, nil
}

// SaveConfig saves the current config to the project-level asimi.toml file
func SaveConfig(config *Config) error {
	projectConfigPath := filepath.Join(".agents", "asimi.toml")

	if err := os.MkdirAll(".agents", 0o755); err != nil {
		return fmt.Errorf("failed to create .agents directory: %w", err)
	}

	// Create koanf instance and load current project config if it exists
	k := koanf.New(".")
	if _, err := os.Stat(projectConfigPath); err == nil {
		if err := k.Load(file.Provider(projectConfigPath), koanftoml.Parser()); err != nil {
			return fmt.Errorf("failed to load existing project config: %w", err)
		}
	}

	// Update the model setting
	if err := k.Set("llm.model", config.LLM.Model); err != nil {
		return fmt.Errorf("failed to update model in config: %w", err)
	}

	// Save to file
	data, err := k.Marshal(koanftoml.Parser())
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(projectConfigPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// UpdateUserLLMAuth updates or creates ~/.config/asimi/asimi.toml with the given LLM auth settings.
// It saves API keys securely in the keyring and only stores provider/model in the config file.
// TODO: remove this as it removes commands, store in sqlite when it's ready
func UpdateUserLLMAuth(provider, apiKey, model string) error {
	// Save API key securely in keyring
	if err := SaveAPIKeyToKeyring(provider, apiKey); err != nil {
		// Fall back to file storage with warning
		log.Printf("Warning: Failed to save API key to keyring, falling back to file storage: %v", err)
		return updateAPIKeyInFile(provider, apiKey, model)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home dir: %w", err)
	}
	cfgDir := filepath.Join(homeDir, ".config", "asimi")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	cfgPath := filepath.Join(cfgDir, "conf.toml")

	// If file does not exist, create minimal content
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		content := "[llm]\n" +
			fmt.Sprintf("provider = \"%s\"\n", provider) +
			fmt.Sprintf("model = \"%s\"\n", model) +
			"auth_method = \"apikey_keyring\"\n"
		return os.WriteFile(cfgPath, []byte(content), 0o600)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to read user config: %w", err)
	}
	lines := strings.Split(string(data), "\n")

	// Find [llm] section
	llmStart := -1
	llmEnd := len(lines)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[llm]" {
			llmStart = i
			// find end at next section header
			for j := i + 1; j < len(lines); j++ {
				t := strings.TrimSpace(lines[j])
				if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
					llmEnd = j
					break
				}
			}
			break
		}
	}

	// Helper to set or insert a key=value in the [llm] section
	setKey := func(key, value string) {
		quoted := fmt.Sprintf("%s = \"%s\"", key, escapeTOMLString(value))
		if llmStart == -1 {
			return
		}
		found := false
		for i := llmStart + 1; i < llmEnd; i++ {
			t := strings.TrimSpace(lines[i])
			if strings.HasPrefix(t, key+" ") || strings.HasPrefix(t, key+"=") {
				// Replace entire line
				indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
				lines[i] = indent + quoted
				found = true
				break
			}
		}
		if !found {
			// Insert before llmEnd
			if llmEnd > len(lines) {
				llmEnd = len(lines)
			}
			// Ensure there is at least an empty line before end
			insertAt := llmEnd
			newLines := append([]string{}, lines[:insertAt]...)
			newLines = append(newLines, quoted)
			newLines = append(newLines, lines[insertAt:]...)
			lines = newLines
			llmEnd++
		}
	}

	if llmStart == -1 {
		// Append a new [llm] section
		var b strings.Builder
		b.WriteString(string(data))
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			b.WriteString("\n")
		}
		b.WriteString("[llm]\n")
		b.WriteString(fmt.Sprintf("provider = \"%s\"\n", provider))
		b.WriteString(fmt.Sprintf("model = \"%s\"\n", model))
		b.WriteString(fmt.Sprintf("api_key = \"%s\"\n", escapeTOMLString(apiKey)))
		return os.WriteFile(cfgPath, []byte(b.String()), 0o644)
	}

	// Update keys in-place
	setKey("provider", provider)
	setKey("model", model)
	setKey("auth_method", "apikey_keyring")

	// Remove plaintext API key if it exists
	removeKey := func(key string) {
		for i := llmStart + 1; i < llmEnd; i++ {
			t := strings.TrimSpace(lines[i])
			if strings.HasPrefix(t, key+" ") || strings.HasPrefix(t, key+"=") {
				// Remove this line
				newLines := append([]string{}, lines[:i]...)
				newLines = append(newLines, lines[i+1:]...)
				lines = newLines
				llmEnd--
				break
			}
		}
	}
	removeKey("api_key")

	return os.WriteFile(cfgPath, []byte(strings.Join(lines, "\n")), 0o600)
}

// updateAPIKeyInFile is the fallback method for storing API keys in file (less secure)
func updateAPIKeyInFile(provider, apiKey, model string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home dir: %w", err)
	}
	cfgDir := filepath.Join(homeDir, ".config", "asimi")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	cfgPath := filepath.Join(cfgDir, "conf.toml")

	// If file does not exist, create minimal content
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		content := "[llm]\n" +
			fmt.Sprintf("provider = \"%s\"\n", provider) +
			fmt.Sprintf("model = \"%s\"\n", model) +
			fmt.Sprintf("api_key = \"%s\"\n", escapeTOMLString(apiKey)) +
			"auth_method = \"apikey_file\"\n"
		return os.WriteFile(cfgPath, []byte(content), 0o600)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to read user config: %w", err)
	}
	lines := strings.Split(string(data), "\n")

	// Find [llm] section
	llmStart := -1
	llmEnd := len(lines)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[llm]" {
			llmStart = i
			// find end at next section header
			for j := i + 1; j < len(lines); j++ {
				t := strings.TrimSpace(lines[j])
				if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
					llmEnd = j
					break
				}
			}
			break
		}
	}

	// Helper to set or insert a key=value in the [llm] section
	setKey := func(key, value string) {
		quoted := fmt.Sprintf("%s = \"%s\"", key, escapeTOMLString(value))
		if llmStart == -1 {
			return
		}
		found := false
		for i := llmStart + 1; i < llmEnd; i++ {
			t := strings.TrimSpace(lines[i])
			if strings.HasPrefix(t, key+" ") || strings.HasPrefix(t, key+"=") {
				// Replace entire line
				indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
				lines[i] = indent + quoted
				found = true
				break
			}
		}
		if !found {
			// Insert before llmEnd
			if llmEnd > len(lines) {
				llmEnd = len(lines)
			}
			// Ensure there is at least an empty line before end
			insertAt := llmEnd
			newLines := append([]string{}, lines[:insertAt]...)
			newLines = append(newLines, quoted)
			newLines = append(newLines, lines[insertAt:]...)
			lines = newLines
			llmEnd++
		}
	}

	if llmStart == -1 {
		// Append a new [llm] section
		var b strings.Builder
		b.WriteString(string(data))
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			b.WriteString("\n")
		}
		b.WriteString("[llm]\n")
		b.WriteString(fmt.Sprintf("provider = \"%s\"\n", provider))
		b.WriteString(fmt.Sprintf("model = \"%s\"\n", model))
		b.WriteString(fmt.Sprintf("api_key = \"%s\"\n", escapeTOMLString(apiKey)))
		b.WriteString("auth_method = \"apikey_file\"\n")
		return os.WriteFile(cfgPath, []byte(b.String()), 0o600)
	}

	// Update keys in-place
	setKey("provider", provider)
	setKey("model", model)
	setKey("api_key", apiKey)
	setKey("auth_method", "apikey_file")

	return os.WriteFile(cfgPath, []byte(strings.Join(lines, "\n")), 0o600)
}

func escapeTOMLString(s string) string {
	// Basic escaping for quotes and backslashes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// UpdateUserOAuthTokens saves OAuth tokens securely in the OS keyring and updates provider in config.
func UpdateUserOAuthTokens(provider, accessToken, refreshToken string, expiry time.Time) error {
	// Save tokens securely in keyring
	if err := SaveTokenToKeyring(provider, accessToken, refreshToken, expiry); err != nil {
		// Fall back to file storage with warning
		log.Printf("Warning: Failed to save tokens to keyring, falling back to file storage: %v", err)
		return updateOAuthTokensInFile(provider, accessToken, refreshToken, expiry)
	}

	// Only save provider info in the config file (not the tokens)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home dir: %w", err)
	}
	cfgDir := filepath.Join(homeDir, ".config", "asimi")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	cfgPath := filepath.Join(cfgDir, "conf.toml")

	data := []byte{}
	if b, err := os.ReadFile(cfgPath); err == nil {
		data = b
	}
	lines := strings.Split(string(data), "\n")

	// Ensure we have an [llm] section
	llmStart := -1
	llmEnd := len(lines)
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if t == "[llm]" {
			llmStart = i
			for j := i + 1; j < len(lines); j++ {
				tt := strings.TrimSpace(lines[j])
				if strings.HasPrefix(tt, "[") && strings.HasSuffix(tt, "]") {
					llmEnd = j
					break
				}
			}
			break
		}
	}
	if llmStart == -1 {
		// append
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "[llm]")
		llmStart = len(lines) - 1
		llmEnd = len(lines)
	}

	setKey := func(key, value string) {
		quoted := fmt.Sprintf("%s = \"%s\"", key, escapeTOMLString(value))
		found := false
		for i := llmStart + 1; i < llmEnd; i++ {
			t := strings.TrimSpace(lines[i])
			if strings.HasPrefix(t, key+" ") || strings.HasPrefix(t, key+"=") {
				indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
				lines[i] = indent + quoted
				found = true
				break
			}
		}
		if !found {
			insertAt := llmEnd
			newLines := append([]string{}, lines[:insertAt]...)
			newLines = append(newLines, quoted)
			newLines = append(newLines, lines[insertAt:]...)
			lines = newLines
			llmEnd++
		}
	}

	// Only set provider and a note about secure storage
	setKey("provider", provider)
	setKey("auth_method", "oauth_keyring")

	// Remove any plaintext tokens from config if they exist
	removeKey := func(key string) {
		for i := llmStart + 1; i < llmEnd; i++ {
			t := strings.TrimSpace(lines[i])
			if strings.HasPrefix(t, key+" ") || strings.HasPrefix(t, key+"=") {
				// Remove this line
				newLines := append([]string{}, lines[:i]...)
				newLines = append(newLines, lines[i+1:]...)
				lines = newLines
				llmEnd--
				break
			}
		}
	}
	removeKey("auth_token")
	removeKey("refresh_token")

	return os.WriteFile(cfgPath, []byte(strings.Join(lines, "\n")), 0o600) // More restrictive permissions
}

// updateOAuthTokensInFile is the fallback method for storing tokens in file (less secure)
func updateOAuthTokensInFile(provider, accessToken, refreshToken string, expiry time.Time) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home dir: %w", err)
	}
	cfgDir := filepath.Join(homeDir, ".config", "asimi")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	cfgPath := filepath.Join(cfgDir, "conf.toml")

	data := []byte{}
	if b, err := os.ReadFile(cfgPath); err == nil {
		data = b
	}
	lines := strings.Split(string(data), "\n")

	// Ensure we have an [llm] section
	llmStart := -1
	llmEnd := len(lines)
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if t == "[llm]" {
			llmStart = i
			for j := i + 1; j < len(lines); j++ {
				tt := strings.TrimSpace(lines[j])
				if strings.HasPrefix(tt, "[") && strings.HasSuffix(tt, "]") {
					llmEnd = j
					break
				}
			}
			break
		}
	}
	if llmStart == -1 {
		// append
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "[llm]")
		llmStart = len(lines) - 1
		llmEnd = len(lines)
	}

	setKey := func(key, value string) {
		quoted := fmt.Sprintf("%s = \"%s\"", key, escapeTOMLString(value))
		found := false
		for i := llmStart + 1; i < llmEnd; i++ {
			t := strings.TrimSpace(lines[i])
			if strings.HasPrefix(t, key+" ") || strings.HasPrefix(t, key+"=") {
				indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
				lines[i] = indent + quoted
				found = true
				break
			}
		}
		if !found {
			insertAt := llmEnd
			newLines := append([]string{}, lines[:insertAt]...)
			newLines = append(newLines, quoted)
			newLines = append(newLines, lines[insertAt:]...)
			lines = newLines
			llmEnd++
		}
	}

	// Set provider to ensure consistency
	setKey("provider", provider)
	setKey("auth_method", "oauth_file")
	setKey("auth_token", accessToken)
	if refreshToken != "" {
		setKey("refresh_token", refreshToken)
	}

	return os.WriteFile(cfgPath, []byte(strings.Join(lines, "\n")), 0o600) // More restrictive permissions
}
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getOAuthConfig(provider string) (oauthProviderConfig, error) {
	p := oauthProviderConfig{}
	switch provider {
	case "googleai":
		// Defaults for Google accounts (Gemini)
		p.AuthURL = getEnv(os.Getenv("ASIMI_OAUTH_GOOGLE_AUTH_URL"), "https://accounts.google.com/o/oauth2/v2/auth")
		p.TokenURL = getEnv(os.Getenv("ASIMI_OAUTH_GOOGLE_TOKEN_URL"), "https://oauth2.googleapis.com/token")
		p.ClientID = os.Getenv("ASIMI_OAUTH_GOOGLE_CLIENT_ID")
		p.ClientSecret = os.Getenv("ASIMI_OAUTH_GOOGLE_CLIENT_SECRET")
		scopes := os.Getenv("ASIMI_OAUTH_GOOGLE_SCOPES")
		if scopes == "" {
			// Default to the Generative Language scope
			p.Scopes = []string{"https://www.googleapis.com/auth/generative-language"}
		} else {
			p.Scopes = strings.Split(scopes, ",")
		}
	case "openai":
		p.AuthURL = os.Getenv("ASIMI_OAUTH_OPENAI_AUTH_URL")
		p.TokenURL = os.Getenv("ASIMI_OAUTH_OPENAI_TOKEN_URL")
		p.ClientID = os.Getenv("ASIMI_OAUTH_OPENAI_CLIENT_ID")
		p.ClientSecret = os.Getenv("ASIMI_OAUTH_OPENAI_CLIENT_SECRET")
		scopes := os.Getenv("ASIMI_OAUTH_OPENAI_SCOPES")
		if scopes != "" {
			p.Scopes = strings.Split(scopes, ",")
		}
	case "anthropic":
		p.AuthURL = os.Getenv("ASIMI_OAUTH_ANTHROPIC_AUTH_URL")
		p.TokenURL = os.Getenv("ASIMI_OAUTH_ANTHROPIC_TOKEN_URL")
		p.ClientID = os.Getenv("ASIMI_OAUTH_ANTHROPIC_CLIENT_ID")
		p.ClientSecret = os.Getenv("ASIMI_OAUTH_ANTHROPIC_CLIENT_SECRET")
		scopes := os.Getenv("ASIMI_OAUTH_ANTHROPIC_SCOPES")
		if scopes != "" {
			p.Scopes = strings.Split(scopes, ",")
		}
	default:
		return p, fmt.Errorf("unsupported provider for oauth: %s", provider)
	}
	if p.AuthURL == "" || p.TokenURL == "" || p.ClientID == "" {
		return p, fmt.Errorf("OAuth not configured. Set ASIMI_OAUTH_* env vars for %s", provider)
	}
	return p, nil
}
