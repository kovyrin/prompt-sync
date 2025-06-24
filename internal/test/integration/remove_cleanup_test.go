package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test verifies that the `prompt-sync remove` command reliably removes
// rendered files for a source.
func TestRemoveCommandCleansRenderedFiles(t *testing.T) {
	// Build binary
	binPath := buildPromptSync(t)

	// Workspace
	workDir := t.TempDir()

	// Path to sample repo (uses local testdata fixture)
	projectRoot := getProjectRoot(t)
	repoPath := filepath.Join(projectRoot, "testdata/repos/enterprise-prompts")

	// Init project
	runCmd(t, workDir, binPath, "init")

	// Add and install prompts
	runCmd(t, workDir, binPath, "add", "file://"+repoPath+"#v2.0.0", "--allow-unknown")

	// Verify multiple files exist
	cursorFiles := []string{
		filepath.Join(workDir, ".cursor/rules/_active/auth-patterns.md"),
		filepath.Join(workDir, ".cursor/rules/_active/event-driven.md"),
		filepath.Join(workDir, ".cursor/rules/_active/microservices.mdc"),
		filepath.Join(workDir, ".cursor/rules/_active/unit-testing.md"),
	}

	for _, file := range cursorFiles {
		assert.FileExists(t, file, "Expected file to exist after install: %s", file)
	}

	// Remove source
	runCmd(t, workDir, binPath, "remove", "file://"+repoPath)

	// All rendered files should be gone
	for _, file := range cursorFiles {
		assert.NoFileExists(t, file, "Expected file to be removed: %s", file)
	}

	// Verify Promptsfile no longer contains the source
	data, err := os.ReadFile(filepath.Join(workDir, "Promptsfile"))
	require.NoError(t, err)
	assert.NotContains(t, string(data), repoPath)
}

func TestRemoveCommandWithMultipleSources(t *testing.T) {
	// Build binary
	binPath := buildPromptSync(t)
	workDir := t.TempDir()
	projectRoot := getProjectRoot(t)

	// Init project
	runCmd(t, workDir, binPath, "init")

	// Add two different sources
	repoPath1 := filepath.Join(projectRoot, "testdata/repos/enterprise-prompts")
	repoPath2 := filepath.Join(projectRoot, "testdata/repos/acme-prompts")

	runCmd(t, workDir, binPath, "add", "file://"+repoPath1+"#v2.0.0", "--allow-unknown", "--no-install")
	runCmd(t, workDir, binPath, "add", "file://"+repoPath2+"#master", "--allow-unknown")

	// Check files from both sources exist
	file1 := filepath.Join(workDir, ".cursor/rules/_active/auth-patterns.md")
	file2 := filepath.Join(workDir, ".cursor/rules/_active/coding.md")
	assert.FileExists(t, file1, "File from source 1")
	assert.FileExists(t, file2, "File from source 2")

	// Remove only the first source
	runCmd(t, workDir, binPath, "remove", "file://"+repoPath1)

	// Files from source 1 should be gone, files from source 2 should remain
	assert.NoFileExists(t, file1, "File from removed source should be gone")
	assert.FileExists(t, file2, "File from other source should remain")
}

func TestRemoveCommandEdgeCases(t *testing.T) {
	binPath := buildPromptSync(t)
	workDir := t.TempDir()

	// Init project
	runCmd(t, workDir, binPath, "init")

	t.Run("remove non-existent source", func(t *testing.T) {
		cmd := exec.Command(binPath, "remove", "github.com/fake/repo")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "not found")
	})

	t.Run("remove without lock file", func(t *testing.T) {
		// Add source without installing
		projectRoot := getProjectRoot(t)
		repoPath := filepath.Join(projectRoot, "testdata/repos/acme-prompts")
		runCmd(t, workDir, binPath, "add", "file://"+repoPath, "--allow-unknown", "--no-install")

		// Remove should still work (just removes from Promptsfile)
		runCmd(t, workDir, binPath, "remove", "file://"+repoPath)

		// Verify it's removed from Promptsfile
		data, err := os.ReadFile(filepath.Join(workDir, "Promptsfile"))
		require.NoError(t, err)
		assert.NotContains(t, string(data), repoPath)
	})
}

// Helpers copied from other integration tests
func buildPromptSync(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "prompt-sync")

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/prompt-sync")
	cmd.Dir = getProjectRoot(t)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return binPath
}

func getProjectRoot(t *testing.T) string {
	t.Helper()
	dir, _ := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	return string(bytes.TrimSpace(dir))
}

func runCmd(t *testing.T, dir, bin string, args ...string) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
