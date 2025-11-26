package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
)

func TestGitHelpersReturnRepositoryState(t *testing.T) {
	tempDir := t.TempDir()

	repo, worktree := initTempRepo(t, tempDir)

	// Switch to the temporary repository directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalDir))
	})
	require.NoError(t, os.Chdir(tempDir))

	// Reset the git info manager so it loads this repository
	t.Cleanup(func() {
		defaultGitInfoManager = newGitInfoManager()
	})
	defaultGitInfoManager = newGitInfoManager()

	require.True(t, isGitRepository(), "expected temporary directory to be detected as git repository")

	expectedBranch := currentBranchName(t, repo)
	require.Equal(t, expectedBranch, getCurrentGitBranch())

	require.Empty(t, getGitStatus(), "freshly committed repository should report clean status")

	// Create an untracked and a modified file to trigger status updates
	untrackedFile := filepath.Join(tempDir, "untracked.txt")
	require.NoError(t, os.WriteFile(untrackedFile, []byte("hello"), 0o644))

	trackedFile := filepath.Join(tempDir, "tracked.txt")
	require.NoError(t, os.WriteFile(trackedFile, []byte("first\n"), 0o644))
	_, err = worktree.Add("tracked.txt")
	require.NoError(t, err)
	_, err = worktree.Commit("add tracked file", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(trackedFile, []byte("first\nsecond\n"), 0o644))

	// Explicitly refresh git info to pick up the changes
	refreshGitInfo()

	// Wait for the async refresh to complete
	require.Eventually(t, func() bool {
		return getGitStatus() == "[!?]"
	}, 2*time.Second, 50*time.Millisecond, "status should reflect modified tracked file and untracked file")
}

func initTempRepo(t *testing.T, dir string) (*gogit.Repository, *gogit.Worktree) {
	repo, err := gogit.PlainInit(dir, false)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	initialFile := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(initialFile, []byte("temp repo\n"), 0o644))

	_, err = worktree.Add("README.md")
	require.NoError(t, err)

	_, err = worktree.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	require.NoError(t, err)

	// Create and checkout a "main" branch so our helpers see a familiar name
	require.NoError(t, worktree.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("main"),
		Create: true,
	}))

	return repo, worktree
}

func currentBranchName(t *testing.T, repo *gogit.Repository) string {
	head, err := repo.Head()
	require.NoError(t, err)
	return head.Name().Short()
}

func TestEnsureOllamaConfiguredMissingBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ollama is not expected on Windows hosts")
	}

	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	err := ensureOllamaConfigured("http://127.0.0.1:12345")
	require.Error(t, err)
	require.Contains(t, err.Error(), "ollama CLI not found")
}

func TestEnsureOllamaConfiguredSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ollama is not expected on Windows hosts")
	}

	fakePath := prepareFakeOllama(t)
	t.Setenv("PATH", fakePath)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/version", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, writeErr := w.Write([]byte(`{"version":"test"}`))
		require.NoError(t, writeErr)
	}))
	t.Cleanup(server.Close)

	err := ensureOllamaConfigured(server.URL)
	require.NoError(t, err)
}

func TestEnsureOllamaConfiguredServerError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ollama is not expected on Windows hosts")
	}

	fakePath := prepareFakeOllama(t)
	t.Setenv("PATH", fakePath)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/version", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	err := ensureOllamaConfigured(server.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "returned status 500")
}

func prepareFakeOllama(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "ollama")
	require.NoError(t, os.WriteFile(binaryPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	return dir
}

func TestDiffLines(t *testing.T) {
	tests := []struct {
		name            string
		original        []string
		current         []string
		expectedAdded   int
		expectedDeleted int
	}{
		{
			name:            "no changes",
			original:        []string{"line1", "line2", "line3"},
			current:         []string{"line1", "line2", "line3"},
			expectedAdded:   0,
			expectedDeleted: 0,
		},
		{
			name:            "only additions",
			original:        []string{"line1", "line2"},
			current:         []string{"line1", "line2", "line3", "line4"},
			expectedAdded:   2,
			expectedDeleted: 0,
		},
		{
			name:            "only deletions",
			original:        []string{"line1", "line2", "line3", "line4"},
			current:         []string{"line1", "line2"},
			expectedAdded:   0,
			expectedDeleted: 2,
		},
		{
			name:            "balanced changes (should be modifications)",
			original:        []string{"line1", "line2", "line3"},
			current:         []string{"line1", "newline2", "newline3"},
			expectedAdded:   2,
			expectedDeleted: 2,
		},
		{
			name:            "more additions than deletions",
			original:        []string{"line1", "line2"},
			current:         []string{"line1", "newline2", "line3", "line4"},
			expectedAdded:   3,
			expectedDeleted: 1,
		},
		{
			name:            "more deletions than additions",
			original:        []string{"line1", "line2", "line3", "line4"},
			current:         []string{"line1", "newline2"},
			expectedAdded:   1,
			expectedDeleted: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, deleted := diffLines(tt.original, tt.current)
			require.Equal(t, tt.expectedAdded, added, "added lines mismatch")
			require.Equal(t, tt.expectedDeleted, deleted, "deleted lines mismatch")
		})
	}
}

func TestParseGitNumstat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		expectedAdded int
		expectedDel   int
	}{
		{
			name: "mixed changes",
			input: "1\t1\tmain.go\n" +
				"34\t0\ttui.go\n",
			expectedAdded: 35,
			expectedDel:   1,
		},
		{
			name:          "binary files ignored",
			input:         "-\t-\timage.png\n",
			expectedAdded: 0,
			expectedDel:   0,
		},
		{
			name:          "unbalanced changes stay separate",
			input:         "5\t2\treport.md\n",
			expectedAdded: 5,
			expectedDel:   2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			added, deleted := parseGitNumstat([]byte(tt.input))
			require.Equal(t, tt.expectedAdded, added)
			require.Equal(t, tt.expectedDel, deleted)
		})
	}
}
