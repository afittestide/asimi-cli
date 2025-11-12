package storage

import (
	"time"

	"github.com/tmc/langchaingo/llms"
)

// Schema version for migrations
const SchemaVersion = 1

// SessionConfig holds session persistence configuration
type SessionConfig struct {
	Enabled      bool
	MaxSessions  int
	MaxAgeDays   int
	ListLimit    int
	AutoSave     bool
	SaveInterval int
}

// HistoryConfig holds persistent history configuration
type HistoryConfig struct {
	Enabled      bool
	MaxSessions  int  // Used as max entries for history
	MaxAgeDays   int
	ListLimit    int
	AutoSave     bool
	SaveInterval int
}

// DBSession maps directly to the sessions table with db tags
// This is used for database operations only
type DBSession struct {
	ID          string    `db:"id"`
	BranchID    int64     `db:"branch_id"`
	CreatedAt   time.Time `db:"created_at"`
	LastUpdated time.Time `db:"last_updated"`
	FirstPrompt string    `db:"first_prompt"`
	Provider    string    `db:"provider"`
	Model       string    `db:"model"`
	WorkingDir  string    `db:"working_dir"`
}

// SessionData contains the persistable session fields
// Note: The main Session type in the main package includes runtime fields
// like llm, toolCatalog, etc. that are not persisted
type SessionData struct {
	ID           string
	CreatedAt    time.Time
	LastUpdated  time.Time
	FirstPrompt  string
	Provider     string
	Model        string
	WorkingDir   string
	ProjectSlug  string
	Messages     []llms.MessageContent
	ContextFiles map[string]string
}

// Repository represents a Git repository (host/org/project)
type Repository struct {
	ID      int64  `db:"id"`      // Auto-increment primary key
	Host    string `db:"host"`    // e.g., "github.com", "gitlab.com", "bitbucket.org"
	Org     string `db:"org"`     // e.g., "tuzig"
	Project string `db:"project"` // e.g., "asimi-cli"
}

// Branch represents a Git branch within a repository
type Branch struct {
	ID           int64  `db:"id"`            // Auto-increment primary key
	RepositoryID int64  `db:"repository_id"` // Foreign key to repositories.id
	Name         string `db:"name"`          // e.g., "main", "feature/sqlite"
}

// Message represents a single message in a conversation
type Message struct {
	ID        int64     `db:"id"`         // Auto-increment primary key
	SessionID string    `db:"session_id"` // Foreign key to sessions.id
	Sequence  int       `db:"sequence"`   // Message order in conversation
	Role      string    `db:"role"`       // "human", "ai", "system", "tool"
	Content   string    `db:"content"`    // JSON-encoded MessageContent.Parts
	CreatedAt time.Time `db:"created_at"` // Stored as Unix timestamp
}

// PromptHistory represents a user prompt for autocomplete
type PromptHistory struct {
	ID       int64     `db:"id"`        // Auto-increment primary key
	BranchID int64     `db:"branch_id"` // Foreign key to branches.id
	Prompt   string    `db:"prompt"`    // User's prompt text
	Timestamp time.Time `db:"timestamp"` // Stored as Unix timestamp
}

// CommandHistory represents a slash command for history
type CommandHistory struct {
	ID        int64     `db:"id"`        // Auto-increment primary key
	BranchID  int64     `db:"branch_id"` // Foreign key to branches.id
	Command   string    `db:"command"`   // Command text
	Timestamp time.Time `db:"timestamp"` // Stored as Unix timestamp
}

// SchemaVersionRecord tracks schema migrations
type SchemaVersionRecord struct {
	Version   int       `db:"version"`
	AppliedAt time.Time `db:"applied_at"`
}

// Schema is the SQL DDL for creating all tables
const Schema = `
-- Repositories table (host + org + project)
CREATE TABLE IF NOT EXISTS repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    host TEXT NOT NULL,
    org TEXT NOT NULL,
    project TEXT NOT NULL,
    UNIQUE(host, org, project)
);

CREATE INDEX IF NOT EXISTS idx_repositories_lookup ON repositories(host, org, project);

-- Branches table
CREATE TABLE IF NOT EXISTS branches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    UNIQUE(repository_id, name),
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_branches_repo ON branches(repository_id, name);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    branch_id INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    last_updated INTEGER NOT NULL,
    first_prompt TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    working_dir TEXT NOT NULL,
    FOREIGN KEY (branch_id) REFERENCES branches(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_branch ON sessions(branch_id, last_updated DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_updated ON sessions(last_updated DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_created ON sessions(created_at DESC);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, sequence);
CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at DESC);

-- Prompt history table
CREATE TABLE IF NOT EXISTS prompt_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    branch_id INTEGER NOT NULL,
    prompt TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    FOREIGN KEY (branch_id) REFERENCES branches(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_prompt_history_branch ON prompt_history(branch_id, timestamp DESC);

-- Command history table
CREATE TABLE IF NOT EXISTS command_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    branch_id INTEGER NOT NULL,
    command TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    FOREIGN KEY (branch_id) REFERENCES branches(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_command_history_branch ON command_history(branch_id, timestamp DESC);

-- Schema version table
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);

INSERT OR IGNORE INTO schema_version (version, applied_at) VALUES (1, unixepoch());
`
