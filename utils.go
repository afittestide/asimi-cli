package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	gogit "github.com/go-git/go-git/v5"
)

// RepoInfo contains information about the git repository and worktree
type RepoInfo struct {
	ProjectRoot  string
	WorktreePath string
	Branch       string
	IsWorktree   bool
	IsMain       bool
	status       string // Cached git status
}

// GetStatus returns a short git status string (e.g., "[!+]")
// This is cached at the time RepoInfo is created and does not update dynamically
func (r *RepoInfo) GetStatus() string {
	return r.status
}

// GetRepoInfo returns information about the current git repository and worktree
func GetRepoInfo() RepoInfo {
	cwd, err := os.Getwd()
	if err != nil {
		return RepoInfo{}
	}

	// Detect if we're in a worktree by checking if .git is a file vs directory
	gitPath := filepath.Join(cwd, ".git")
	info, err := os.Stat(gitPath)
	isWorktree := err == nil && !info.IsDir()

	// Find project root - for worktrees, find the main repo root
	var projectRoot string
	var worktreePath string

	if isWorktree {
		// Read .git file to find the main repository
		mainRepoRoot, err := findMainRepoRoot(cwd)
		if err == nil && mainRepoRoot != "" {
			projectRoot = mainRepoRoot
			// Calculate worktree path relative to main repo
			relPath, err := filepath.Rel(projectRoot, cwd)
			if err == nil && relPath != "." {
				worktreePath = relPath
			}
		} else {
			// Fallback to current directory if we can't find main repo
			projectRoot = cwd
		}
	} else {
		// Not a worktree, use standard project root finding
		projectRoot = findProjectRoot(cwd)
	}

	// Get current branch and status using go-git
	branch := ""
	status := ""
	repo, err := gogit.PlainOpenWithOptions(cwd, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err == nil {
		ref, err := repo.Head()
		if err == nil {
			if ref.Name().IsBranch() {
				branch = ref.Name().Short()
			} else {
				branch = ref.Hash().String()[:7]
			}
		} else if isWorktree {
			// go-git doesn't fully support worktrees, try reading HEAD directly
			branch = readBranchFromWorktree()
		}
		// Get git status - only read once at startup
		// Skip in tests to avoid slow git operations
		if os.Getenv("ASIMI_SKIP_GIT_STATUS") == "" {
			status = readShortStatus(repo)
		}
	} else if isWorktree {
		// go-git failed, try reading branch directly from worktree
		branch = readBranchFromWorktree()
	}

	// Detect if branch is main/master
	isMain := branch == "main" || branch == "master"

	return RepoInfo{
		ProjectRoot:  projectRoot,
		WorktreePath: worktreePath,
		Branch:       branch,
		IsWorktree:   isWorktree,
		IsMain:       isMain,
		status:       status,
	}
}

func getFileTree(root string) ([]string, error) {
	var files []string
	// Directories to ignore at any level
	ignoreDirs := map[string]bool{
		".git":      true,
		"vendor":    true,
		"worktrees": true,
		"archive":   true,
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if ignoreDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// We only want files.
		// Let's make sure the path is relative to the root.
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

// findProjectRoot returns the nearest ancestor directory (including start)
// that contains a project marker like .git or go.mod. Falls back to start.
// findMainRepoRoot finds the main repository root when in a worktree
// by reading the .git file and extracting the main repo path from the gitdir
func findMainRepoRoot(worktreeDir string) (string, error) {
	gitPath := filepath.Join(worktreeDir, ".git")

	// Read the .git file
	content, err := os.ReadFile(gitPath)
	if err != nil {
		return "", err
	}

	// Parse gitdir: path
	// Example: gitdir: /Users/daonb/src/asimi-cli/.git/worktrees/GH33-fix-worktrees
	gitdirLine := strings.TrimSpace(string(content))
	if !strings.HasPrefix(gitdirLine, "gitdir: ") {
		return "", fmt.Errorf("invalid .git file format")
	}

	gitdir := strings.TrimPrefix(gitdirLine, "gitdir: ")

	// The gitdir points to: <main-repo>/.git/worktrees/<worktree-name>
	// We need to extract <main-repo> from this path
	// Split by "/.git/worktrees/" to get the main repo path
	parts := strings.Split(gitdir, "/.git/worktrees/")
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected gitdir format: %s", gitdir)
	}

	mainRepoRoot := parts[0]
	return mainRepoRoot, nil
}

// getCurrentGitBranch returns the current git branch name
func getCurrentGitBranch() string {
	return defaultGitInfoManager.CurrentBranch()
}

// getGitStatus returns a shortened git status string
func getGitStatus() string {
	return defaultGitInfoManager.ShortStatus()
}

// isGitRepository checks if the current directory is a git repository
func isGitRepository() bool {
	return defaultGitInfoManager.IsRepository()
}

var defaultGitInfoManager = newGitInfoManager()

type gitInfoManager struct {
	mu         sync.RWMutex
	branch     string
	status     string
	repo       *gogit.Repository
	repoPath   string
	isRepo     bool
	lastUpdate time.Time
	updateCh   chan struct{}
	startOnce  sync.Once
}

func newGitInfoManager() *gitInfoManager {
	return &gitInfoManager{
		updateCh: make(chan struct{}, 1),
	}
}

func (m *gitInfoManager) start() {
	m.startOnce.Do(func() {
		m.refresh()
		go m.loop()
	})
}

func (m *gitInfoManager) loop() {
	for range m.updateCh {
		m.refresh()
	}
}

func (m *gitInfoManager) refresh() {
	branch, status, repo, repoPath, err := m.readRepositoryState()
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		m.branch = ""
		m.status = ""
		m.isRepo = false
		m.repo = nil
		m.repoPath = ""
		m.lastUpdate = now
		return
	}

	m.branch = branch
	m.status = status
	m.isRepo = true
	m.repo = repo
	m.repoPath = repoPath
	m.lastUpdate = now
}

func (m *gitInfoManager) readRepositoryState() (string, string, *gogit.Repository, string, error) {
	repo, repoPath, err := m.ensureRepository()
	if err != nil {
		return "", "", nil, "", err
	}

	branch := readCurrentBranch(repo)
	status := readShortStatus(repo)

	return branch, status, repo, repoPath, nil
}

func (m *gitInfoManager) ensureRepository() (*gogit.Repository, string, error) {
	// Get current working directory to find project root
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	root := findProjectRoot(cwd)

	m.mu.RLock()
	repo := m.repo
	repoPath := m.repoPath
	m.mu.RUnlock()

	if repo != nil && repoPath == root {
		return repo, repoPath, nil
	}

	repo, err = gogit.PlainOpenWithOptions(root, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, "", err
	}

	m.mu.Lock()
	m.repo = repo
	m.repoPath = root
	m.mu.Unlock()

	return repo, root, nil
}

func (m *gitInfoManager) requestRefresh() {
	select {
	case m.updateCh <- struct{}{}:
	default:
	}
}

func (m *gitInfoManager) CurrentBranch() string {
	m.start()

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.branch
}

func (m *gitInfoManager) ShortStatus() string {
	m.start()

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *gitInfoManager) IsRepository() bool {
	m.start()

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRepo
}

func refreshGitInfo() {
	defaultGitInfoManager.start()
	defaultGitInfoManager.requestRefresh()
}

func readCurrentBranch(repo *gogit.Repository) string {
	if repo == nil {
		return ""
	}

	ref, err := repo.Head()
	if err != nil {
		// go-git doesn't fully support worktrees, try reading HEAD directly
		branch := readBranchFromWorktree()
		if branch != "" {
			return branch
		}
		return ""
	}

	if ref.Name().IsBranch() {
		return ref.Name().Short()
	}

	return ref.Hash().String()[:7]
}

// readBranchFromWorktree reads the branch name directly from a git worktree
func readBranchFromWorktree() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	gitPath := filepath.Join(cwd, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return ""
	}

	// If .git is a directory, not a worktree
	if info.IsDir() {
		return ""
	}

	// Read the .git file to get the actual git directory
	content, err := os.ReadFile(gitPath)
	if err != nil {
		return ""
	}

	// Parse gitdir: path
	gitdirLine := strings.TrimSpace(string(content))
	if !strings.HasPrefix(gitdirLine, "gitdir: ") {
		return ""
	}

	gitdir := strings.TrimPrefix(gitdirLine, "gitdir: ")

	// Read HEAD from the worktree git directory
	headPath := filepath.Join(gitdir, "HEAD")
	headContent, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}

	// Parse ref: refs/heads/branch
	headLine := strings.TrimSpace(string(headContent))
	if strings.HasPrefix(headLine, "ref: ") {
		ref := strings.TrimPrefix(headLine, "ref: ")
		if strings.HasPrefix(ref, "refs/heads/") {
			return strings.TrimPrefix(ref, "refs/heads/")
		}
	}

	// If HEAD is detached, return the short hash
	if len(headLine) >= 7 {
		return headLine[:7]
	}

	return ""
}

func readShortStatus(repo *gogit.Repository) string {
	if repo == nil {
		return ""
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return ""
	}

	status, err := worktree.Status()
	if err != nil {
		return ""
	}

	return summarizeStatus(status)
}

func summarizeStatus(status gogit.Status) string {
	if len(status) == 0 {
		return ""
	}

	var modified, added, deleted, untracked, renamed int
	for _, entry := range status {
		switch entry.Staging {
		case gogit.Modified, gogit.UpdatedButUnmerged:
			modified++
		case gogit.Added, gogit.Copied:
			added++
		case gogit.Deleted:
			deleted++
		case gogit.Renamed:
			renamed++
		case gogit.Untracked:
			untracked++
		}
		switch entry.Worktree {
		case gogit.Modified, gogit.UpdatedButUnmerged:
			modified++
		case gogit.Added, gogit.Copied:
			added++
		case gogit.Deleted:
			deleted++
		case gogit.Renamed:
			renamed++
		case gogit.Untracked:
			untracked++
		}
	}

	var builder strings.Builder
	builder.WriteString("[")
	if modified > 0 {
		builder.WriteString("!")
	}
	if added > 0 {
		builder.WriteString("+")
	}
	if deleted > 0 {
		builder.WriteString("-")
	}
	if renamed > 0 {
		builder.WriteString("→")
	}
	if untracked > 0 {
		builder.WriteString("?")
	}
	builder.WriteString("]")

	result := builder.String()
	if result == "[]" {
		return ""
	}
	return result
}

// shortenProviderModel shortens provider and model names for display
func shortenProviderModel(provider, model string) string {
	// Shorten common provider names
	switch strings.ToLower(provider) {
	case "anthropic":
		provider = "Claude"
	case "openai":
		provider = "GPT"
	case "google", "googleai":
		provider = "Gemini"
	case "ollama":
		provider = "Ollama"
	}

	// Shorten common model names
	modelShort := model
	lowerModel := strings.ToLower(model)
	if strings.Contains(lowerModel, "claude") {
		// Handle models like "Claude-Haiku-4.5", "Claude 3.5 Sonnet", etc.
		// Extract the meaningful part after "claude"
		parts := strings.FieldsFunc(lowerModel, func(r rune) bool {
			return r == '-' || r == ' ' || r == '_'
		})

		// Skip "claude" prefix and build the short name
		if len(parts) > 1 {
			// For "claude-3-5-haiku-20240307" -> "3.5-Haiku"
			// For "claude-haiku-4.5" -> "Haiku-4.5"
			var shortParts []string
			for i := 1; i < len(parts); i++ {
				part := parts[i]
				// Skip date suffixes like "20240307"
				if len(part) == 8 && strings.ContainsAny(part, "0123456789") {
					continue
				}
				// Skip "latest" suffix
				if part == "latest" {
					continue
				}
				shortParts = append(shortParts, part)
			}

			if len(shortParts) > 0 {
				// Join parts and capitalize first letter of each word
				result := strings.Join(shortParts, "-")
				// Capitalize: "3-5-haiku" -> "3.5-Haiku"
				result = strings.ReplaceAll(result, "-5-", ".5-")
				// Capitalize model names
				result = strings.ReplaceAll(result, "haiku", "Haiku")
				result = strings.ReplaceAll(result, "sonnet", "Sonnet")
				result = strings.ReplaceAll(result, "opus", "Opus")
				modelShort = result
			}
		} else if strings.Contains(lowerModel, "instant") {
			modelShort = "Instant"
		}
	} else if strings.Contains(lowerModel, "gpt") {
		if strings.Contains(model, "4") {
			if strings.Contains(model, "turbo") {
				modelShort = "4T"
			} else {
				modelShort = "4"
			}
		} else if strings.Contains(model, "3.5") {
			modelShort = "3.5"
		}
	} else if strings.Contains(lowerModel, "gemini") {
		if strings.Contains(model, "pro") {
			modelShort = "Pro"
		} else if strings.Contains(model, "flash") {
			modelShort = "Flash"
		}
	}

	return fmt.Sprintf("%s-%s", provider, modelShort)
}

// getProviderStatusIcon returns an icon for the provider status
func getProviderStatusIcon(connected bool) string {
	if connected {
		return "✅"
	}
	return "🔌"
}
