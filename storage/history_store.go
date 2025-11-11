package storage

import (
	"fmt"
	"time"
)

// HistoryStore handles prompt and command history persistence
type HistoryStore struct {
	db  *DB
	cfg *HistoryConfig
}

// NewHistoryStore creates a new history store
func NewHistoryStore(db *DB, cfg *HistoryConfig) *HistoryStore {
	return &HistoryStore{
		db:  db,
		cfg: cfg,
	}
}

// AppendPrompt adds a prompt to the history for a given host/org/project
func (h *HistoryStore) AppendPrompt(host, org, project, prompt string) error {
	// Get or create repository
	repoID, err := h.db.GetOrCreateRepository(host, org, project)
	if err != nil {
		return err
	}

	// Insert prompt
	_, err = h.db.conn.Exec(`
		INSERT INTO prompt_history (repository_id, prompt, timestamp)
		VALUES (?, ?, ?)`,
		repoID,
		prompt,
		time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to append prompt: %w", err)
	}

	// Apply limit if configured
	if h.cfg != nil && h.cfg.MaxSessions > 0 {
		_, err = h.db.conn.Exec(`
			DELETE FROM prompt_history
			WHERE repository_id = ?
			AND id NOT IN (
				SELECT id FROM prompt_history
				WHERE repository_id = ?
				ORDER BY timestamp DESC
				LIMIT ?
			)`,
			repoID, repoID, h.cfg.MaxSessions,
		)
		if err != nil {
			return fmt.Errorf("failed to apply prompt history limit: %w", err)
		}
	}

	return nil
}

// AppendCommand adds a command to the history for a given host/org/project
func (h *HistoryStore) AppendCommand(host, org, project, command string) error {
	// Get or create repository
	repoID, err := h.db.GetOrCreateRepository(host, org, project)
	if err != nil {
		return err
	}

	// Insert command
	_, err = h.db.conn.Exec(`
		INSERT INTO command_history (repository_id, command, timestamp)
		VALUES (?, ?, ?)`,
		repoID,
		command,
		time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to append command: %w", err)
	}

	// Apply limit if configured
	if h.cfg != nil && h.cfg.MaxSessions > 0 {
		_, err = h.db.conn.Exec(`
			DELETE FROM command_history
			WHERE repository_id = ?
			AND id NOT IN (
				SELECT id FROM command_history
				WHERE repository_id = ?
				ORDER BY timestamp DESC
				LIMIT ?
			)`,
			repoID, repoID, h.cfg.MaxSessions,
		)
		if err != nil {
			return fmt.Errorf("failed to apply command history limit: %w", err)
		}
	}

	return nil
}

// LoadPromptHistory loads prompt history for a given host/org/project
func (h *HistoryStore) LoadPromptHistory(host, org, project string, limit int) ([]HistoryEntry, error) {
	// Get repository
	repo, err := h.db.GetRepository(host, org, project)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return []HistoryEntry{}, nil // No repository means no history
	}

	// Query prompts
	query := `
		SELECT prompt, timestamp
		FROM prompt_history
		WHERE repository_id = ?
		ORDER BY timestamp DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := h.db.conn.Query(query, repo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load prompt history: %w", err)
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var prompt string
		var timestamp int64

		if err := rows.Scan(&prompt, &timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan prompt: %w", err)
		}

		entries = append(entries, HistoryEntry{
			Content:   prompt,
			Timestamp: time.Unix(timestamp, 0),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating prompts: %w", err)
	}

	return entries, nil
}

// LoadCommandHistory loads command history for a given host/org/project
func (h *HistoryStore) LoadCommandHistory(host, org, project string, limit int) ([]HistoryEntry, error) {
	// Get repository
	repo, err := h.db.GetRepository(host, org, project)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return []HistoryEntry{}, nil // No repository means no history
	}

	// Query commands
	query := `
		SELECT command, timestamp
		FROM command_history
		WHERE repository_id = ?
		ORDER BY timestamp DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := h.db.conn.Query(query, repo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load command history: %w", err)
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var command string
		var timestamp int64

		if err := rows.Scan(&command, &timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}

		entries = append(entries, HistoryEntry{
			Content:   command,
			Timestamp: time.Unix(timestamp, 0),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating commands: %w", err)
	}

	return entries, nil
}

// ClearPromptHistory clears all prompt history for a given host/org/project
func (h *HistoryStore) ClearPromptHistory(host, org, project string) error {
	// Get repository
	repo, err := h.db.GetRepository(host, org, project)
	if err != nil {
		return err
	}
	if repo == nil {
		return nil // No repository means nothing to clear
	}

	_, err = h.db.conn.Exec("DELETE FROM prompt_history WHERE repository_id = ?", repo.ID)
	if err != nil {
		return fmt.Errorf("failed to clear prompt history: %w", err)
	}

	return nil
}

// ClearCommandHistory clears all command history for a given host/org/project
func (h *HistoryStore) ClearCommandHistory(host, org, project string) error {
	// Get repository
	repo, err := h.db.GetRepository(host, org, project)
	if err != nil {
		return err
	}
	if repo == nil {
		return nil // No repository means nothing to clear
	}

	_, err = h.db.conn.Exec("DELETE FROM command_history WHERE repository_id = ?", repo.ID)
	if err != nil {
		return fmt.Errorf("failed to clear command history: %w", err)
	}

	return nil
}

// HistoryEntry represents a single history item (prompt or command)
type HistoryEntry struct {
	Content   string    // Prompt or command text
	Timestamp time.Time // When it was entered
}

// CleanupOldHistory removes history entries older than configured age
func (h *HistoryStore) CleanupOldHistory() error {
	if h.cfg == nil || h.cfg.MaxAgeDays <= 0 {
		return nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -h.cfg.MaxAgeDays).Unix()

	// Clean prompt history
	_, err := h.db.conn.Exec(
		"DELETE FROM prompt_history WHERE timestamp < ?",
		cutoffTime,
	)
	if err != nil {
		return fmt.Errorf("failed to cleanup old prompt history: %w", err)
	}

	// Clean command history
	_, err = h.db.conn.Exec(
		"DELETE FROM command_history WHERE timestamp < ?",
		cutoffTime,
	)
	if err != nil {
		return fmt.Errorf("failed to cleanup old command history: %w", err)
	}

	return nil
}
