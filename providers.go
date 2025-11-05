package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/fx"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// multiHandler wraps multiple handlers and writes to all of them
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Enable if any handler is enabled
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if err := h.Handle(ctx, r.Clone()); err != nil {
			return err
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

// LoggerResult holds the configured logger
type LoggerResult struct {
	fx.Out
	Logger *slog.Logger
}

// ProvideLogger creates and returns a logger instance
func ProvideLogger() (LoggerResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return LoggerResult{}, fmt.Errorf("failed to get user home directory: %w", err)
	}

	logDir := filepath.Join(homeDir, ".local", "share", "asimi")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return LoggerResult{}, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// Set up lumberjack for log rotation
	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "asimi.log"),
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	}

	// Set log level based on debug flag
	logLevel := slog.LevelInfo
	if cli.Debug {
		logLevel = slog.LevelDebug
	}

	// File handler (logs everything at configured level)
	fileHandler := slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: logLevel})

	logger := slog.New(fileHandler)
	slog.SetDefault(logger)

	return LoggerResult{
		Logger: logger,
	}, nil
}

// ProvideConfig loads and returns the application configuration
func ProvideConfig(logger *slog.Logger) (*Config, error) {
	logger.Info("loading configuration")
	config, err := LoadConfig()
	if err != nil {
		logger.Info("using default configuration due to load failure")
		logger.Debug("Warning: Using defaults due to config load failure", "error", err)
		// Continue with default config
		config = &Config{
			Server: ServerConfig{
				Host: "localhost",
				Port: 3000,
			},
			Database: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "asimi",
				Password: "asimi",
				Name:     "asimi_dev",
			},
			Logging: LoggingConfig{
				Level:  "info",
				Format: "text",
			},
			LLM: LLMConfig{
				Provider: "openai",
				Model:    "gpt-3.5-turbo",
				APIKey:   "",
				BaseURL:  "",
			},
		}
	}
	logger.Info("configuration loaded")
	return config, nil
}

// ProvideRepoInfo returns information about the git repository
func ProvideRepoInfo(config *Config, logger *slog.Logger) RepoInfo {
	logger.Info("detecting git repository")
	repoInfo := GetRepoInfo()
	if repoInfo.ProjectRoot != "" {
		logger.Info("git repository detected", "root", repoInfo.ProjectRoot, "branch", repoInfo.Branch)
	} else {
		logger.Info("no git repository found")
	}
	return repoInfo
}

// ProvideShellRunner creates and returns a shell runner
func ProvideShellRunner(config *Config, repoInfo RepoInfo, logger *slog.Logger) shellRunner {
	logger.Info("initializing shell runner")
	return newPodmanShellRunner(config.LLM.PodmanAllowHostFallback, config, repoInfo)
}

// ModelClientParams holds parameters for async LLM client initialization
type ModelClientParams struct {
	fx.In
	Lifecycle fx.Lifecycle
	Config    *Config
	RepoInfo  RepoInfo
	Logger    *slog.Logger
}

// ProvideModelClient sets up async LLM client initialization
// The model client will be initialized in a goroutine and send a message when ready
func ProvideModelClient(params ModelClientParams) {
	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Launch async initialization
			go func() {
				params.Logger.Info("connecting to LLM", "provider", params.Config.LLM.Provider)
				llm, err := getModelClient(params.Config)
				if cli.Debug {
					params.Logger.Debug("[TIMING] getModelClient() completed")
				}

				if err != nil {
					params.Logger.Warn("failed to connect to LLM, running without AI capabilities", "error", err)
					if program != nil {
						program.Send(llmInitErrorMsg{err: err})
					}
				} else {
					params.Logger.Info("LLM client connected")
					params.Logger.Info("creating session")
					sess, sessErr := NewSession(llm, params.Config, params.RepoInfo, func(m any) {
						if program != nil {
							program.Send(m)
						}
					})
					if cli.Debug {
						params.Logger.Debug("[TIMING] NewSession() completed")
					}

					if sessErr != nil {
						params.Logger.Error("failed to create session", "error", sessErr)
						if program != nil {
							program.Send(llmInitErrorMsg{err: sessErr})
						}
					} else {
						if program != nil {
							program.Send(llmInitSuccessMsg{session: sess})
						}
					}
				}
			}()
			return nil
		},
	})
}

// ProvideCommandHistory creates and returns the command history store
func ProvideCommandHistory(repoInfo RepoInfo, logger *slog.Logger) (*HistoryStore, error) {
	logger.Info("loading command history")
	historyStore, err := NewHistoryStore(repoInfo)
	if err != nil {
		logger.Warn("failed to initialize history store", "error", err)
		return nil, nil // Don't fail, just return nil
	}
	return historyStore, nil
}

// ProvideSessionHistory creates and returns the session history store
func ProvideSessionHistory(config *Config, repoInfo RepoInfo, logger *slog.Logger) (*SessionStore, error) {
	if !config.Session.Enabled {
		return nil, nil // Session storage is disabled
	}

	logger.Info("loading session history")
	maxSessions := 50
	maxAgeDays := 30
	if config.Session.MaxSessions > 0 {
		maxSessions = config.Session.MaxSessions
	}
	if config.Session.MaxAgeDays > 0 {
		maxAgeDays = config.Session.MaxAgeDays
	}

	store, err := NewSessionStore(repoInfo, maxSessions, maxAgeDays)
	if err != nil {
		logger.Error("failed to create session store", "error", err)
		return nil, nil // Don't fail startup
	}
	return store, nil
}

// ProvideTUIModel creates and returns the TUI model
func ProvideTUIModel(config *Config, repoInfo RepoInfo, historyStore *HistoryStore, sessionStore *SessionStore, logger *slog.Logger) *TUIModel {
	return NewTUIModel(config, &repoInfo, historyStore, sessionStore)
}

// TUIProgramParams holds parameters for TUI program initialization
type TUIProgramParams struct {
	fx.In
	Model     *TUIModel
	Lifecycle fx.Lifecycle
	Logger    *slog.Logger
}

// StartTUI creates the TUI program
func StartTUI(params TUIProgramParams) *tea.Program {
	params.Logger.Info("creating TUI program")

	// Create the bubbletea program with alt screen and mouse support
	prog := tea.NewProgram(params.Model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Set global program reference so async operations can send messages
	program = prog

	return prog
}
