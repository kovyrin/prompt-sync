package system

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCommand(t *testing.T) {
	// Build binary once for all tests
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "prompt-sync-test-bin")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/prompt-sync")
	buildCmd.Dir = filepath.Join("..", "..", "..") // project root relative to internal/test/system
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build prompt-sync binary: %v\n%s", err, output)
	}

	t.Run("add source to existing Promptsfile", func(t *testing.T) {
		workDir := t.TempDir()

		// First init the project
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Add a trusted source
		cmd = exec.Command(binaryPath, "add", "github.com/org/prompts", "--no-install")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "add failed: %s", string(output))

		// Verify the source was added to Promptsfile
		promptsfileContent, err := os.ReadFile(filepath.Join(workDir, "Promptsfile"))
		require.NoError(t, err)
		assert.Contains(t, string(promptsfileContent), "github.com/org/prompts")
		assert.Contains(t, string(output), "✓ Added source: github.com/org/prompts")
	})

	t.Run("add source with version specification", func(t *testing.T) {
		workDir := t.TempDir()

		// Init
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Add source with version
		cmd = exec.Command(binaryPath, "add", "github.com/org/prompts#v1.0.0", "--no-install")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "add failed: %s", string(output))

		// Verify version was preserved
		promptsfileContent, err := os.ReadFile(filepath.Join(workDir, "Promptsfile"))
		require.NoError(t, err)
		assert.Contains(t, string(promptsfileContent), "github.com/org/prompts#v1.0.0")
	})

	t.Run("reject untrusted source without --allow-unknown", func(t *testing.T) {
		workDir := t.TempDir()

		// Init
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Try to add untrusted source
		cmd = exec.Command(binaryPath, "add", "github.com/untrusted/prompts")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "expected error for untrusted source")
		assert.Contains(t, string(output), "untrusted source")

		// Verify source was not added
		promptsfileContent, err := os.ReadFile(filepath.Join(workDir, "Promptsfile"))
		require.NoError(t, err)
		assert.NotContains(t, string(promptsfileContent), "github.com/untrusted/prompts")
	})

	t.Run("allow untrusted source with --allow-unknown", func(t *testing.T) {
		workDir := t.TempDir()

		// Init
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Add untrusted source with flag
		cmd = exec.Command(binaryPath, "add", "github.com/untrusted/prompts", "--allow-unknown", "--no-install")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "add failed: %s", string(output))

		// Verify source was added
		promptsfileContent, err := os.ReadFile(filepath.Join(workDir, "Promptsfile"))
		require.NoError(t, err)
		assert.Contains(t, string(promptsfileContent), "github.com/untrusted/prompts")
	})

	t.Run("reject duplicate sources", func(t *testing.T) {
		workDir := t.TempDir()

		// Init
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Add source first time
		cmd = exec.Command(binaryPath, "add", "github.com/org/prompts", "--no-install")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Try to add same source again
		cmd = exec.Command(binaryPath, "add", "github.com/org/prompts", "--no-install")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "expected error for duplicate source")
		assert.Contains(t, string(output), "already exists")
	})

	t.Run("add without Promptsfile should fail", func(t *testing.T) {
		workDir := t.TempDir()

		// Try to add without init
		cmd := exec.Command(binaryPath, "add", "github.com/org/prompts")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "expected error when Promptsfile doesn't exist")
		assert.Contains(t, string(output), "Promptsfile not found")
		assert.Contains(t, string(output), ".ai/Promptsfile") // The error now shows searched paths
	})

	t.Run("add and install from fixture repository", func(t *testing.T) {
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

		// Create a git repo in the fixture if it doesn't exist
		if _, err := os.Stat(filepath.Join(absRepoPath, ".git")); os.IsNotExist(err) {
			// Initialize the test repo
			gitCmd := exec.Command("git", "init")
			gitCmd.Dir = absRepoPath
			require.NoError(t, gitCmd.Run())

			gitCmd = exec.Command("git", "config", "user.email", "test@example.com")
			gitCmd.Dir = absRepoPath
			gitCmd.Run()

			gitCmd = exec.Command("git", "config", "user.name", "Test User")
			gitCmd.Dir = absRepoPath
			gitCmd.Run()

			gitCmd = exec.Command("git", "add", ".")
			gitCmd.Dir = absRepoPath
			require.NoError(t, gitCmd.Run())

			gitCmd = exec.Command("git", "commit", "-m", "Initial commit")
			gitCmd.Dir = absRepoPath
			gitCmd.Run()

			// Create a master branch
			gitCmd = exec.Command("git", "branch", "-M", "master")
			gitCmd.Dir = absRepoPath
			gitCmd.Run()
		}

		// Add and install the fixture repository
		cmd = exec.Command(binaryPath, "add", "file://"+absRepoPath+"#master", "--allow-unknown")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "add with install failed: %s", string(output))

		// Verify source was added
		assert.Contains(t, string(output), "✓ Added source")
		assert.Contains(t, string(output), "✓ Installation complete")

		// Verify lock file was created
		lockPath := filepath.Join(workDir, "Promptsfile.lock")
		assert.FileExists(t, lockPath)

		// Verify files were rendered
		cursorRulesPath := filepath.Join(workDir, ".cursor", "rules", "_active")
		_, err = os.Stat(cursorRulesPath)
		assert.NoError(t, err, "cursor rules directory should exist")
	})

	t.Run("validate source URL format", func(t *testing.T) {
		workDir := t.TempDir()

		// Init
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = workDir
		require.NoError(t, cmd.Run())

		// Test various invalid URLs
		invalidURLs := []struct {
			url           string
			expectedError string
		}{
			{"", "source URL cannot be empty"},
			{"not-a-url", "invalid repository format"},
			{"http://example.com", "repository path format"},
			{"github.com/", "should not end with /"},
		}

		for _, tc := range invalidURLs {
			cmd = exec.Command(binaryPath, "add", tc.url)
			cmd.Dir = workDir
			output, err := cmd.CombinedOutput()
			require.Error(t, err, "expected error for URL: %s", tc.url)
			assert.Contains(t, string(output), tc.expectedError, "URL: %s", tc.url)
		}
	})
}

func TestAddCommandHelp(t *testing.T) {
	// Build binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "prompt-sync-test-bin")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/prompt-sync")
	buildCmd.Dir = filepath.Join("..", "..", "..")
	require.NoError(t, buildCmd.Run())

	// Test help output
	cmd := exec.Command(binaryPath, "add", "--help")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err)

	// Verify help content
	helpText := string(output)
	assert.Contains(t, helpText, "Add a new prompt source")
	assert.Contains(t, helpText, "--no-install")
	assert.Contains(t, helpText, "--allow-unknown")
	assert.Contains(t, helpText, "github.com/org/prompts")
}

func TestAddCommandInCIMode(t *testing.T) {
	// Build binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "prompt-sync-test-bin")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/prompt-sync")
	buildCmd.Dir = filepath.Join("..", "..", "..")
	require.NoError(t, buildCmd.Run())

	workDir := t.TempDir()

	// Init
	cmd := exec.Command(binaryPath, "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Try to add untrusted source in CI mode
	cmd = exec.Command(binaryPath, "add", "github.com/untrusted/prompts", "--no-install")
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "CI=true")
	output, err := cmd.CombinedOutput()
	require.Error(t, err, "expected error in CI mode for untrusted source")
	assert.Contains(t, string(output), "untrusted source")

	// CI mode should still allow trusted sources
	cmd = exec.Command(binaryPath, "add", "github.com/org/prompts", "--no-install")
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "CI=true")
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "should allow trusted source in CI mode: %s", string(output))
	assert.Contains(t, string(output), "✓ Added source")
}
