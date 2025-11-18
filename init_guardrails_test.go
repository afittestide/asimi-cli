package main

import (
	"os"
	"os/exec"
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

	// Now, run the guardrails within this controlled environment
	report := runInitGuardrails()

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