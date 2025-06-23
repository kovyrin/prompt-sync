package system_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/kovyrin/prompt-sync/internal/config"
	"github.com/kovyrin/prompt-sync/internal/lock"
)

func TestUpdateSystemCommand(t *testing.T) {
	// Build the binary
	binPath := buildPromptSyncBinary(t)

	// Get project root for accessing test repos
	projectRoot := getProjectRootForSystemTest(t)
	acmeRepo := filepath.Join(projectRoot, "testdata/repos/acme-prompts")
	devRepo := filepath.Join(projectRoot, "testdata/repos/dev-prompts")

	t.Run("update all unpinned sources", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create Promptsfile with test sources
		cfg := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo,            // Unpinned
				"file://" + devRepo + "#master", // Pinned to branch (not considered pinned for updates)
			},
			Adapters: config.AdaptersCfg{
				Cursor: config.CursorCfg{Enabled: true},
			},
		}
		writeSystemTestPromptsfile(t, tmpDir, cfg)

		// Create a lock file
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "oldcommit"},
				{URL: "file://" + devRepo, Commit: "oldcommit", Ref: "master"},
			},
		}
		writeSystemTestLockfile(t, tmpDir, lockData)

		// Run update with dry-run
		cmd := exec.Command(binPath, "update", "--dry-run", "--allow-unknown")
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Command failed: %s", string(output))

		// Check output
		assert.Contains(t, string(output), "Checking for updates")
		assert.Contains(t, string(output), "Available updates:")
		assert.Contains(t, string(output), "Dry run mode - no changes made")
	})

	t.Run("update specific source", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create Promptsfile
		cfg := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo,
				"file://" + devRepo,
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writeSystemTestPromptsfile(t, tmpDir, cfg)

		// Create lock file
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "abc123"},
				{URL: "file://" + devRepo, Commit: "def456"},
			},
		}
		writeSystemTestLockfile(t, tmpDir, lockData)

		// Update only one source
		cmd := exec.Command(binPath, "update", "file://"+acmeRepo, "--dry-run", "--allow-unknown")
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Command failed: %s", string(output))

		// Should only check one source
		assert.Contains(t, string(output), "Checking for updates to 1 source(s)")
		assert.Contains(t, string(output), acmeRepo)
		assert.NotContains(t, string(output), devRepo)
	})

	t.Run("force update pinned source", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create Promptsfile with pinned source
		cfg := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo + "#v1.0.0", // Pinned
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writeSystemTestPromptsfile(t, tmpDir, cfg)

		// Create lock file
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "abc123", Ref: "v1.0.0"},
			},
		}
		writeSystemTestLockfile(t, tmpDir, lockData)

		// Try without force - should fail
		cmd := exec.Command(binPath, "update", "file://"+acmeRepo)
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "pinned to a specific version")
		assert.Contains(t, string(output), "--force")

		// Try with force
		cmd = exec.Command(binPath, "update", "file://"+acmeRepo, "--force", "--dry-run", "--allow-unknown")
		cmd.Dir = tmpDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Command failed: %s", string(output))
		assert.Contains(t, string(output), "force update pinned source")
	})

	t.Run("update requires Promptsfile", func(t *testing.T) {
		tmpDir := t.TempDir()

		cmd := exec.Command(binPath, "update")
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "Promptsfile not found")
	})

	t.Run("update requires lock file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create only Promptsfile, no lock file
		cfg := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo,
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writeSystemTestPromptsfile(t, tmpDir, cfg)

		cmd := exec.Command(binPath, "update")
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "no lock file found")
		assert.Contains(t, string(output), "prompt-sync install")
	})

	t.Run("update in CI mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create Promptsfile
		cfg := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo,
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writeSystemTestPromptsfile(t, tmpDir, cfg)

		// Create lock file
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "abc123"},
			},
		}
		writeSystemTestLockfile(t, tmpDir, lockData)

		// Run with CI mode
		cmd := exec.Command(binPath, "update", "--dry-run", "--allow-unknown")
		cmd.Dir = tmpDir
		cmd.Env = append(os.Environ(), "CI=true")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Command failed: %s", string(output))

		// CI mode should work (strict mode is enabled)
		assert.Contains(t, string(output), "Checking for updates")
	})

	t.Run("actual update with install", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create Promptsfile
		cfg := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo + "#master",
			},
			Adapters: config.AdaptersCfg{
				Cursor: config.CursorCfg{Enabled: true},
			},
		}
		writeSystemTestPromptsfile(t, tmpDir, cfg)

		// First do an install to create initial state
		cmd := exec.Command(binPath, "install", "--yes", "--allow-unknown")
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Install failed: %s", string(output))

		// Verify lock file was created
		lockPath := filepath.Join(tmpDir, "Promptsfile.lock")
		assert.FileExists(t, lockPath)

		// Read initial lock file
		initialLock := readSystemTestLockfile(t, tmpDir)
		initialCommit := initialLock.Sources[0].Commit

		// Now run update (it won't actually update since we're at HEAD, but it should work)
		cmd = exec.Command(binPath, "update", "--allow-unknown")
		cmd.Dir = tmpDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Update failed: %s", string(output))

		// Lock file should still exist
		assert.FileExists(t, lockPath)

		// For this test repo, commit shouldn't change since we're already at HEAD
		updatedLock := readSystemTestLockfile(t, tmpDir)
		assert.Equal(t, initialCommit, updatedLock.Sources[0].Commit)
	})
}

// Helper functions
func writeSystemTestPromptsfile(t *testing.T, dir string, cfg *config.ExtendedConfig) {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "Promptsfile"), data, 0644)
	require.NoError(t, err)
}

func writeSystemTestLockfile(t *testing.T, dir string, lockData *lock.Lock) {
	t.Helper()
	data, err := yaml.Marshal(lockData)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "Promptsfile.lock"), data, 0644)
	require.NoError(t, err)
}

func readSystemTestLockfile(t *testing.T, dir string) *lock.Lock {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "Promptsfile.lock"))
	require.NoError(t, err)
	var lockData lock.Lock
	err = yaml.Unmarshal(data, &lockData)
	require.NoError(t, err)
	return &lockData
}

func getProjectRootForSystemTest(t *testing.T) string {
	t.Helper()
	// Start from current directory and walk up until we find go.mod
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}

// buildPromptSyncBinary builds the prompt-sync binary and returns its path.
// It reuses the existing helper function pattern from other system tests.
func buildPromptSyncBinary(t *testing.T) string {
	t.Helper()

	// Build to a temporary directory
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "prompt-sync")

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/prompt-sync")

	// Find the project root
	projectRoot := getProjectRootForSystemTest(t)
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	// Verify the binary exists
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("Binary not found at %s: %v", binPath, err)
	}

	return binPath
}
