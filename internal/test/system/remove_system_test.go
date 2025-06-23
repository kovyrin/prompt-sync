package system

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveCommand(t *testing.T) {
	// Build binary once for all tests
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "prompt-sync-test-bin")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/prompt-sync")
	buildCmd.Dir = filepath.Join("..", "..", "..") // project root relative to internal/test/system
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build prompt-sync binary: %v\n%s", err, output)
	}

	t.Run("remove source from Promptsfile", func(t *testing.T) {
		workDir := t.TempDir()

		// First init the project
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Add some sources
		cmd = exec.Command(binaryPath, "add", "github.com/org/prompts1", "--no-install")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		cmd = exec.Command(binaryPath, "add", "github.com/org/prompts2", "--no-install")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		cmd = exec.Command(binaryPath, "add", "github.com/org/prompts3", "--no-install")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Remove the middle source
		cmd = exec.Command(binaryPath, "remove", "github.com/org/prompts2")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "remove failed: %s", string(output))

		// Verify the source was removed
		assert.Contains(t, string(output), "✓ Removed source: github.com/org/prompts2")

		// Check Promptsfile
		promptsfileContent, err := os.ReadFile(filepath.Join(workDir, "Promptsfile"))
		require.NoError(t, err)
		assert.Contains(t, string(promptsfileContent), "github.com/org/prompts1")
		assert.NotContains(t, string(promptsfileContent), "github.com/org/prompts2")
		assert.Contains(t, string(promptsfileContent), "github.com/org/prompts3")
	})

	t.Run("remove source with version specification", func(t *testing.T) {
		workDir := t.TempDir()

		// Init
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Add sources with versions
		cmd = exec.Command(binaryPath, "add", "github.com/org/prompts#v1.0.0", "--no-install")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		cmd = exec.Command(binaryPath, "add", "github.com/tools/utils", "--no-install")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Remove by base URL (without version)
		cmd = exec.Command(binaryPath, "remove", "github.com/org/prompts")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "remove failed: %s", string(output))

		// Verify it was removed
		promptsfileContent, err := os.ReadFile(filepath.Join(workDir, "Promptsfile"))
		require.NoError(t, err)
		assert.NotContains(t, string(promptsfileContent), "github.com/org/prompts")
		assert.Contains(t, string(promptsfileContent), "github.com/tools/utils")
	})

	t.Run("error on non-existent source", func(t *testing.T) {
		workDir := t.TempDir()

		// Init
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Add a source
		cmd = exec.Command(binaryPath, "add", "github.com/org/prompts", "--no-install")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Try to remove non-existent source
		cmd = exec.Command(binaryPath, "remove", "github.com/org/nonexistent")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "expected error for non-existent source")
		assert.Contains(t, string(output), "not found")

		// Verify original source is still there
		promptsfileContent, err := os.ReadFile(filepath.Join(workDir, "Promptsfile"))
		require.NoError(t, err)
		assert.Contains(t, string(promptsfileContent), "github.com/org/prompts")
	})

	t.Run("remove without Promptsfile should fail", func(t *testing.T) {
		workDir := t.TempDir()

		// Try to remove without init
		cmd := exec.Command(binaryPath, "remove", "github.com/org/prompts")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "expected error when Promptsfile doesn't exist")
		assert.Contains(t, string(output), "Promptsfile not found")
		assert.Contains(t, string(output), "prompt-sync init")
	})

	t.Run("remove last source shows cleanup hint", func(t *testing.T) {
		workDir := t.TempDir()

		// Init
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Add one source
		cmd = exec.Command(binaryPath, "add", "github.com/org/prompts", "--no-install")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Remove the only source
		cmd = exec.Command(binaryPath, "remove", "github.com/org/prompts")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "remove failed: %s", string(output))

		// Verify cleanup hint is shown
		assert.Contains(t, string(output), "All sources removed")
		assert.Contains(t, string(output), "clean up .gitignore")
	})

	t.Run("remove cleans up rendered files", func(t *testing.T) {
		workDir := t.TempDir()

		// Init
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Get path to test fixture
		projectRoot := filepath.Join("..", "..", "..")
		repoPath := filepath.Join(projectRoot, "testdata", "repos", "acme-prompts")
		absRepoPath, err := filepath.Abs(repoPath)
		require.NoError(t, err)

		// Add and install the fixture repository
		cmd = exec.Command(binaryPath, "add", "file://"+absRepoPath+"#master", "--allow-unknown")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "add with install failed: %s", string(output))

		// Verify files were rendered
		cursorRulesPath := filepath.Join(workDir, ".cursor", "rules", "_active")
		entries, err := os.ReadDir(cursorRulesPath)
		require.NoError(t, err)
		require.Greater(t, len(entries), 0, "Should have rendered files")

		// Remember a rendered file
		renderedFile := filepath.Join(cursorRulesPath, entries[0].Name())
		require.FileExists(t, renderedFile)

		// Remove the source
		cmd = exec.Command(binaryPath, "remove", "file://"+absRepoPath)
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "remove failed: %s", string(output))

		// Verify files were deleted
		assert.NoFileExists(t, renderedFile, "Rendered file should have been deleted")
	})

	t.Run("using rm alias", func(t *testing.T) {
		workDir := t.TempDir()

		// Init
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Add a source
		cmd = exec.Command(binaryPath, "add", "github.com/org/prompts", "--no-install")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Remove using 'rm' alias
		cmd = exec.Command(binaryPath, "rm", "github.com/org/prompts")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "rm alias failed: %s", string(output))
		assert.Contains(t, string(output), "✓ Removed source")

		// Verify it was removed
		promptsfileContent, err := os.ReadFile(filepath.Join(workDir, "Promptsfile"))
		require.NoError(t, err)
		assert.NotContains(t, string(promptsfileContent), "github.com/org/prompts")
	})

	t.Run("prevent removing overlay sources", func(t *testing.T) {
		workDir := t.TempDir()

		// Create a Promptsfile with an overlay manually
		promptsfileContent := `sources:
  - github.com/org/prompts
overlays:
  - scope: personal
    source: github.com/personal/prompts
adapters:
  cursor:
    enabled: true
  claude:
    enabled: false
`
		err := os.WriteFile(filepath.Join(workDir, "Promptsfile"), []byte(promptsfileContent), 0644)
		require.NoError(t, err)

		// Try to remove overlay source
		cmd := exec.Command(binaryPath, "remove", "github.com/personal/prompts")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "expected error when removing overlay source")
		assert.Contains(t, string(output), "overlay")
		assert.Contains(t, string(output), "personal")

		// Verify nothing was changed
		updatedContent, err := os.ReadFile(filepath.Join(workDir, "Promptsfile"))
		require.NoError(t, err)
		assert.Equal(t, promptsfileContent, string(updatedContent))
	})
}

func TestRemoveCommandHelp(t *testing.T) {
	// Build binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "prompt-sync-test-bin")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/prompt-sync")
	buildCmd.Dir = filepath.Join("..", "..", "..")
	require.NoError(t, buildCmd.Run())

	// Test help output
	cmd := exec.Command(binaryPath, "remove", "--help")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err)

	// Verify help content
	helpText := string(output)
	assert.Contains(t, helpText, "Remove a prompt source from your Promptsfile")
	assert.Contains(t, helpText, "clean up rendered files")
	assert.Contains(t, helpText, "github.com/org/prompts")
	assert.Contains(t, helpText, "Aliases")
	assert.Contains(t, helpText, "rm")
}
