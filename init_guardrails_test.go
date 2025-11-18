package main

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckMissingInfraFiles(t *testing.T) {
	// This test just verifies the function runs without error
	// The actual files may or may not exist depending on the test environment
	missing := checkMissingInfraFiles()

	// Should return a slice (may be empty or have items)
	// Note: An empty slice is valid when all files exist
	_ = missing // Just verify it doesn't panic
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists",
			slice:    []string{"foo", "bar", "baz"},
			item:     "bar",
			expected: true,
		},
		{
			name:     "item does not exist",
			slice:    []string{"foo", "bar", "baz"},
			item:     "qux",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "foo",
			expected: false,
		},
		{
			name:     "nil slice",
			slice:    nil,
			item:     "foo",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			require.Equal(t, tt.expected, result, "contains(%v, %q)", tt.slice, tt.item)
		})
	}
}

func TestRunInitGuardrails(t *testing.T) {
	// Create a temporary directory to act as a clean project environment
	tempDir := t.TempDir()

	// Change the working directory to the temporary directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	// Ensure we change back to the original directory after the test
	t.Cleanup(func() {
		os.Chdir(originalWd)
	})

	// Initialize a git repository
	_, err = exec.Command("git", "init").CombinedOutput()
	require.NoError(t, err)

	// Create a dummy Justfile
	justfileContent := "\nlint:\n\techo \"Linting passed\"\n\ntest:\n\techo \"Tests passed\"\n\ninfrabuild:\n\techo \"Infra build successful\"\n\nmeasure:\n\techo \"Measure successful\"\n"
	err = os.WriteFile("Justfile", []byte(justfileContent), 0644)
	require.NoError(t, err)

	// Create and commit a dummy Go file
	goFileContent := "package main\n\nfunc main() {}"
	err = os.WriteFile("main.go", []byte(goFileContent), 0644)
	require.NoError(t, err)
	_, err = exec.Command("git", "add", "main.go", "Justfile").CombinedOutput()
	require.NoError(t, err)
	// Need to set user for commit
	_, err = exec.Command("git", "config", "user.email", "test@example.com").CombinedOutput()
	require.NoError(t, err)
	_, err = exec.Command("git", "config", "user.name", "Test User").CombinedOutput()
	require.NoError(t, err)
	_, err = exec.Command("git", "commit", "-m", "initial commit").CombinedOutput()
	require.NoError(t, err)

	// Create a new file to be detected by the git purity check
	newFileContent := "new file"
	err = os.WriteFile("new.txt", []byte(newFileContent), 0644)
	require.NoError(t, err)
	_, err = exec.Command("git", "add", "new.txt").CombinedOutput()
	require.NoError(t, err)

	// Create a spy function to capture update messages
	var updateMessages []string
	updateSpy := func(msg string) {
		updateMessages = append(updateMessages, msg)
	}

	// Create guardrails service (without session for this test)
	config := &Config{}
	logger := slog.Default()
	guardrails := NewGuardrailsService(nil, config, logger)

	// Now, run the guardrails within this controlled environment
	report := runInitGuardrails(guardrails, updateSpy)

	// Verify update messages were called
	require.NotEmpty(t, updateMessages, "Update function should have been called")

	// Check that progress messages were sent
	allUpdates := strings.Join(updateMessages, "")
	require.Contains(t, allUpdates, "Running initialization guardrails", "Should show initialization message")
	require.Contains(t, allUpdates, "Step 1/4: Linting", "Should show linting step")
	require.Contains(t, allUpdates, "Step 2/4: Testing", "Should show testing step")
	require.Contains(t, allUpdates, "Step 3/4: Building and testing sandbox", "Should show sandbox step")
	require.Contains(t, allUpdates, "Step 4/4: Ensuring git purity", "Should show git purity step")

	// Assert that the report indicates success from our dummy commands
	require.Contains(t, report, "✅ Linting passed", "Report should show linting passed")
	require.Contains(t, report, "✅ All tests passed", "Report should show tests passed")
	require.Contains(t, report, "✅ Sandbox built and tested successfully", "Report should show sandbox success")
	require.Contains(t, report, "✅ Git purity check passed", "Report should show git purity passed")
	require.Contains(t, report, "new.txt", "Report should list the new file")
}

func TestInitGuardrailsCompleteMsg(t *testing.T) {
	// Test that message type can be created
	msg := initGuardrailsCompleteMsg{
		report: "Test report",
	}

	require.Equal(t, "Test report", msg.report)
}

func TestGuardrailsService(t *testing.T) {
	config := &Config{}
	logger := slog.Default()
	service := NewGuardrailsService(nil, config, logger)

	require.NotNil(t, service)
	require.NotNil(t, service.checks)
	require.Contains(t, service.checks, "lint")
	require.Contains(t, service.checks, "test")
	require.Contains(t, service.checks, "sandbox")
	require.Contains(t, service.checks, "git-purity")
}

func TestGuardrailRun(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		os.Chdir(originalWd)
	})

	// Initialize git
	_, err = exec.Command("git", "init").CombinedOutput()
	require.NoError(t, err)
	_, err = exec.Command("git", "config", "user.email", "test@example.com").CombinedOutput()
	require.NoError(t, err)
	_, err = exec.Command("git", "config", "user.name", "Test User").CombinedOutput()
	require.NoError(t, err)

	config := &Config{}
	logger := slog.Default()
	service := NewGuardrailsService(nil, config, logger)

	ctx := context.Background()
	updateSpy := func(msg string) {}

	// Test git-purity guardrail
	result, err := service.Run(ctx, "git-purity", updateSpy)
	require.NoError(t, err)
	require.Equal(t, "git-purity", result.Name)
	require.True(t, result.Passed)

	// Test non-existent guardrail
	_, err = service.Run(ctx, "non-existent", updateSpy)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
