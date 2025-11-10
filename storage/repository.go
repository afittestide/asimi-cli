package storage

import (
	"database/sql"
	"fmt"
)

// GetOrCreateRepository gets an existing repository or creates a new one
// Returns the repository ID
func (db *DB) GetOrCreateRepository(host, org, project string) (int64, error) {
	// Try to get existing repository
	var id int64
	err := db.conn.QueryRow(
		"SELECT id FROM repositories WHERE host = ? AND org = ? AND project = ?",
		host, org, project,
	).Scan(&id)

	if err == nil {
		return id, nil
	}

	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query repository: %w", err)
	}

	// Repository doesn't exist, create it
	result, err := db.conn.Exec(
		"INSERT INTO repositories (host, org, project) VALUES (?, ?, ?)",
		host, org, project,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create repository: %w", err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get repository ID: %w", err)
	}

	return id, nil
}

// GetOrCreateBranch gets an existing branch or creates a new one
// Returns the branch ID
func (db *DB) GetOrCreateBranch(repositoryID int64, name string) (int64, error) {
	// Try to get existing branch
	var id int64
	err := db.conn.QueryRow(
		"SELECT id FROM branches WHERE repository_id = ? AND name = ?",
		repositoryID, name,
	).Scan(&id)

	if err == nil {
		return id, nil
	}

	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query branch: %w", err)
	}

	// Branch doesn't exist, create it
	result, err := db.conn.Exec(
		"INSERT INTO branches (repository_id, name) VALUES (?, ?)",
		repositoryID, name,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create branch: %w", err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get branch ID: %w", err)
	}

	return id, nil
}

// GetRepository retrieves a repository by host, org and project
func (db *DB) GetRepository(host, org, project string) (*Repository, error) {
	var repo Repository
	err := db.conn.QueryRow(
		"SELECT id, host, org, project FROM repositories WHERE host = ? AND org = ? AND project = ?",
		host, org, project,
	).Scan(&repo.ID, &repo.Host, &repo.Org, &repo.Project)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return &repo, nil
}

// GetBranch retrieves a branch by repository ID and name
func (db *DB) GetBranch(repositoryID int64, name string) (*Branch, error) {
	var branch Branch
	err := db.conn.QueryRow(
		"SELECT id, repository_id, name FROM branches WHERE repository_id = ? AND name = ?",
		repositoryID, name,
	).Scan(&branch.ID, &branch.RepositoryID, &branch.Name)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}

	return &branch, nil
}

// ListRepositories returns all repositories
func (db *DB) ListRepositories() ([]Repository, error) {
	rows, err := db.conn.Query("SELECT id, host, org, project FROM repositories ORDER BY host, org, project")
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		var repo Repository
		if err := rows.Scan(&repo.ID, &repo.Host, &repo.Org, &repo.Project); err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}
		repos = append(repos, repo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating repositories: %w", err)
	}

	return repos, nil
}

// ListBranches returns all branches for a repository
func (db *DB) ListBranches(repositoryID int64) ([]Branch, error) {
	rows, err := db.conn.Query(
		"SELECT id, repository_id, name FROM branches WHERE repository_id = ? ORDER BY name",
		repositoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}
	defer rows.Close()

	var branches []Branch
	for rows.Next() {
		var branch Branch
		if err := rows.Scan(&branch.ID, &branch.RepositoryID, &branch.Name); err != nil {
			return nil, fmt.Errorf("failed to scan branch: %w", err)
		}
		branches = append(branches, branch)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating branches: %w", err)
	}

	return branches, nil
}

// DeleteRepository deletes a repository and all its branches/sessions (CASCADE)
func (db *DB) DeleteRepository(id int64) error {
	result, err := db.conn.Exec("DELETE FROM repositories WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("repository not found")
	}

	return nil
}

// DeleteBranch deletes a branch and all its sessions (CASCADE)
func (db *DB) DeleteBranch(id int64) error {
	result, err := db.conn.Exec("DELETE FROM branches WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("branch not found")
	}

	return nil
}
