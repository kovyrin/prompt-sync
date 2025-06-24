package system_test

import (
	"encoding/json"
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

func TestFullWorkflow(t *testing.T) {
	// Build the binary once for all subtests
	binPath := buildPromptSyncForWorkflow(t)

	// Get project root for accessing test repos
	projectRoot := getProjectRootForWorkflow(t)
	acmeRepo := filepath.Join(projectRoot, "testdata/repos/acme-prompts")
	devRepo := filepath.Join(projectRoot, "testdata/repos/dev-prompts")

	t.Run("complete user journey", func(t *testing.T) {
		workDir := t.TempDir()

		// Step 1: Initialize new project
		t.Run("init project", func(t *testing.T) {
			cmd := exec.Command(binPath, "init")
			cmd.Dir = workDir
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Init failed: %s", string(output))

			// Verify files were created
			assert.FileExists(t, filepath.Join(workDir, "Promptsfile"))
			assert.FileExists(t, filepath.Join(workDir, ".gitignore"))

			// Verify .gitignore has managed block
			gitignoreContent, err := os.ReadFile(filepath.Join(workDir, ".gitignore"))
			require.NoError(t, err)
			assert.Contains(t, string(gitignoreContent), "# BEGIN prompt-sync managed")
		})

		// Step 2: Add sources (both trusted and untrusted)
		t.Run("add sources", func(t *testing.T) {
			// Try to add untrusted source without flag - should fail
			cmd := exec.Command(binPath, "add", "https://github.com/untrusted/prompts")
			cmd.Dir = workDir
			output, err := cmd.CombinedOutput()
			assert.Error(t, err, "Should fail for untrusted source")
			assert.Contains(t, string(output), "untrusted source")

			// Add trusted test sources
			cmd = exec.Command(binPath, "add", "file://"+acmeRepo+"#v1.0.0", "--no-install", "--allow-unknown")
			cmd.Dir = workDir
			output, err = cmd.CombinedOutput()
			require.NoError(t, err, "Add acme failed: %s", string(output))

			cmd = exec.Command(binPath, "add", "file://"+devRepo+"#master", "--no-install", "--allow-unknown")
			cmd.Dir = workDir
			output, err = cmd.CombinedOutput()
			require.NoError(t, err, "Add dev failed: %s", string(output))

			// Verify Promptsfile was updated
			cfg := readWorkflowPromptsfile(t, workDir)
			assert.Len(t, cfg.Sources, 2)
			assert.Contains(t, cfg.Sources, "file://"+acmeRepo+"#v1.0.0")
			assert.Contains(t, cfg.Sources, "file://"+devRepo+"#master")
		})

		// Step 3: Install prompts and verify rendering
		t.Run("install prompts", func(t *testing.T) {
			// First update the Promptsfile to enable both adapters
			cfg := readWorkflowPromptsfile(t, workDir)
			cfg.Adapters = config.AdaptersCfg{
				Cursor: config.CursorCfg{Enabled: true},
				Claude: config.ClaudeCfg{
					Enabled: true,
					Prefix:  "workflow",
				},
			}
			writeWorkflowPromptsfile(t, workDir, cfg)

			cmd := exec.Command(binPath, "install", "--yes", "--allow-unknown")
			cmd.Dir = workDir
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Install failed: %s", string(output))

			// Verify lock file was created
			assert.FileExists(t, filepath.Join(workDir, "Promptsfile.lock"))

			// Verify files were rendered for both adapters
			// Cursor files
			assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/coding.md"))
			assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/testing.md"))

			// Claude files (with prefix)
			claudeFiles, err := filepath.Glob(filepath.Join(workDir, ".claude/commands/workflow-*.md"))
			require.NoError(t, err)
			assert.Len(t, claudeFiles, 2, "Should have 2 Claude command files")

			// Verify .gitignore was updated
			gitignoreContent, err := os.ReadFile(filepath.Join(workDir, ".gitignore"))
			require.NoError(t, err)
			assert.Contains(t, string(gitignoreContent), ".cursor/rules/_active/")
			assert.Contains(t, string(gitignoreContent), ".claude/commands/workflow-*")
		})

		// Step 4: List installed prompts with various flags
		t.Run("list prompts", func(t *testing.T) {
			// Basic list
			cmd := exec.Command(binPath, "list")
			cmd.Dir = workDir
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "List failed: %s", string(output))
			assert.Contains(t, string(output), acmeRepo)
			assert.Contains(t, string(output), devRepo)
			assert.Contains(t, string(output), "v1.0.0")
			assert.Contains(t, string(output), "master")

			// List with files
			cmd = exec.Command(binPath, "list", "--files")
			cmd.Dir = workDir
			output, err = cmd.CombinedOutput()
			require.NoError(t, err, "List --files failed: %s", string(output))
			assert.Contains(t, string(output), ".cursor/rules/_active/coding.md")
			assert.Contains(t, string(output), ".cursor/rules/_active/testing.md")

			// List with JSON output
			cmd = exec.Command(binPath, "list", "--json")
			cmd.Dir = workDir
			output, err = cmd.CombinedOutput()
			require.NoError(t, err, "List --json failed: %s", string(output))

			// The JSON output is an object, not an array
			var jsonOutput map[string][]map[string]interface{}
			err = json.Unmarshal(output, &jsonOutput)
			require.NoError(t, err, "Failed to parse JSON output")
			sources, ok := jsonOutput["sources"]
			require.True(t, ok, "JSON output should have 'sources' field")
			assert.Len(t, sources, 2, "Should have 2 sources in JSON output")
		})

		// Step 5: Update sources
		t.Run("update sources", func(t *testing.T) {
			// First, let's see what would be updated (dry-run)
			cmd := exec.Command(binPath, "update", "--dry-run", "--allow-unknown")
			cmd.Dir = workDir
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Update dry-run failed: %s", string(output))

			// Only dev-prompts should show as updatable (acme is pinned to v1.0.0)
			assert.Contains(t, string(output), "Checking for updates to 1 source(s)")
			assert.Contains(t, string(output), devRepo)
			assert.NotContains(t, string(output), acmeRepo) // Pinned, shouldn't update

			// Force update the pinned source
			cmd = exec.Command(binPath, "update", "file://"+acmeRepo, "--force", "--allow-unknown")
			cmd.Dir = workDir
			output, err = cmd.CombinedOutput()
			require.NoError(t, err, "Force update failed: %s", string(output))
			assert.Contains(t, string(output), "Updated 1 source(s)")
		})

		// Step 6: Remove a source and verify cleanup
		t.Run("remove source", func(t *testing.T) {
			// Remove dev-prompts
			cmd := exec.Command(binPath, "remove", "file://"+devRepo)
			cmd.Dir = workDir
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Remove failed: %s", string(output))

			// Verify it was removed from Promptsfile
			cfg := readWorkflowPromptsfile(t, workDir)
			assert.Len(t, cfg.Sources, 1)
			assert.NotContains(t, cfg.Sources, devRepo)

			// Verify rendered files were cleaned up
			assert.NoFileExists(t, filepath.Join(workDir, ".cursor/rules/_active/testing.md"))

			// But coding.md should still exist (from acme-prompts)
			assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/coding.md"))
		})

		// Step 7: Verify consistency
		t.Run("verify consistency", func(t *testing.T) {
			// First run should pass (everything is consistent)
			cmd := exec.Command(binPath, "verify", "--allow-unknown")
			cmd.Dir = workDir
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Verify failed: %s", string(output))
			assert.Contains(t, string(output), "✓ All files verified successfully")

			// Modify a rendered file to create drift
			codingPath := filepath.Join(workDir, ".cursor/rules/_active/coding.md")
			err = os.WriteFile(codingPath, []byte("Modified content"), 0644)
			require.NoError(t, err)

			// Verify should now detect drift
			cmd = exec.Command(binPath, "verify", "--allow-unknown")
			cmd.Dir = workDir
			output, err = cmd.CombinedOutput()
			assert.Error(t, err, "Verify should fail with drift")
			assert.Contains(t, string(output), "drift detected")

			// Fix by reinstalling
			cmd = exec.Command(binPath, "install", "--yes", "--allow-unknown")
			cmd.Dir = workDir
			output, err = cmd.CombinedOutput()
			require.NoError(t, err, "Reinstall failed: %s", string(output))

			// Verify should pass again
			cmd = exec.Command(binPath, "verify", "--allow-unknown")
			cmd.Dir = workDir
			output, err = cmd.CombinedOutput()
			require.NoError(t, err, "Verify after reinstall failed: %s", string(output))
		})
	})

	t.Run("both adapters workflow", func(t *testing.T) {
		workDir := t.TempDir()

		// Initialize and set up with both adapters
		cmd := exec.Command(binPath, "init")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Init failed: %s", string(output))

		// Manually update Promptsfile to enable both adapters with custom prefix
		cfg := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo + "#master",
			},
			Adapters: config.AdaptersCfg{
				Cursor: config.CursorCfg{Enabled: true},
				Claude: config.ClaudeCfg{
					Enabled: true,
					Prefix:  "test",
				},
			},
		}
		writeWorkflowPromptsfile(t, workDir, cfg)

		// Install
		cmd = exec.Command(binPath, "install", "--yes", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Install failed: %s", string(output))

		// Verify both adapters rendered files
		assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/coding.md"))

		// Check Claude file with prefix
		claudeFiles, err := filepath.Glob(filepath.Join(workDir, ".claude/commands/test-*.md"))
		require.NoError(t, err)
		assert.Len(t, claudeFiles, 1, "Should have 1 Claude command file with 'test' prefix")

		// Verify .gitignore has both patterns
		gitignoreContent, err := os.ReadFile(filepath.Join(workDir, ".gitignore"))
		require.NoError(t, err)
		assert.Contains(t, string(gitignoreContent), ".cursor/rules/_active/")
		assert.Contains(t, string(gitignoreContent), ".claude/commands/test-*")
	})

	t.Run("CI mode workflow", func(t *testing.T) {
		workDir := t.TempDir()

		// Initialize
		cmd := exec.Command(binPath, "init")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Init failed: %s", string(output))

		// Add source
		cmd = exec.Command(binPath, "add", "file://"+acmeRepo+"#master", "--no-install", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Add failed: %s", string(output))

		// Install in CI mode (should be non-interactive and strict)
		cmd = exec.Command(binPath, "install", "--allow-unknown")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "CI=true")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "CI install failed: %s", string(output))

		// CI install should work without --yes flag
		assert.NotContains(t, string(output), "Proceed?") // No prompts in CI mode
	})

	// Edge case scenarios for task 9A.2
	t.Run("conflicting prompts", func(t *testing.T) {
		workDir := t.TempDir()

		// Initialize
		cmd := exec.Command(binPath, "init")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Init failed: %s", string(output))

		// Create two repos with conflicting file names
		repo1 := createTestRepoWithConflict(t, "conflict1", "rules/coding.md", "# Coding Style from Repo 1")
		repo2 := createTestRepoWithConflict(t, "conflict2", "rules/coding.md", "# Coding Style from Repo 2")

		// Add both sources
		cmd = exec.Command(binPath, "add", "file://"+repo1+"#master", "--no-install", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Add repo1 failed: %s", string(output))

		cmd = exec.Command(binPath, "add", "file://"+repo2+"#master", "--no-install", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Add repo2 failed: %s", string(output))

		// Install should fail due to conflict
		cmd = exec.Command(binPath, "install", "--yes", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		assert.Error(t, err, "Install should fail with conflicts")
		assert.Contains(t, string(output), "conflict")

		// Remove one source to resolve conflict
		cmd = exec.Command(binPath, "remove", "file://"+repo2)
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Remove failed: %s", string(output))

		// Install should now succeed
		cmd = exec.Command(binPath, "install", "--yes", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Install after conflict resolution failed: %s", string(output))
	})

	t.Run("version upgrades and downgrades", func(t *testing.T) {
		workDir := t.TempDir()

		// Initialize
		cmd := exec.Command(binPath, "init")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Init failed: %s", string(output))

		// Add acme-prompts with v1.0.0
		cmd = exec.Command(binPath, "add", "file://"+acmeRepo+"#v1.0.0", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Add v1.0.0 failed: %s", string(output))

		// Check current version
		lockData := readWorkflowLockfile(t, workDir)
		assert.Equal(t, "v1.0.0", lockData.Sources[0].Ref)

		// Try to update (should not update pinned version without force)
		cmd = exec.Command(binPath, "update", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Update failed: %s", string(output))
		assert.Contains(t, string(output), "No sources to update")

		// Force update to master
		cmd = exec.Command(binPath, "update", "file://"+acmeRepo, "--force", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Force update failed: %s", string(output))

		// Now change back to v1.0.0 by editing Promptsfile (simulate downgrade)
		cfg := readWorkflowPromptsfile(t, workDir)
		cfg.Sources[0] = "file://" + acmeRepo + "#v1.0.0"
		writeWorkflowPromptsfile(t, workDir, cfg)

		// Reinstall to apply downgrade
		cmd = exec.Command(binPath, "install", "--yes", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Install for downgrade failed: %s", string(output))

		// Verify we're back at v1.0.0
		lockData = readWorkflowLockfile(t, workDir)
		assert.Equal(t, "v1.0.0", lockData.Sources[0].Ref)
	})

	t.Run("offline mode with cache", func(t *testing.T) {
		workDir := t.TempDir()
		cacheDir := t.TempDir()

		// Initialize
		cmd := exec.Command(binPath, "init")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Init failed: %s", string(output))

		// Add source without cache-dir flag (add doesn't support it)
		cmd = exec.Command(binPath, "add", "file://"+acmeRepo+"#master", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Add failed: %s", string(output))

		// First install with custom cache dir to populate cache
		cmd = exec.Command(binPath, "install", "--yes", "--cache-dir", cacheDir, "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Install with cache-dir failed: %s", string(output))

		// Verify cache was populated
		cachedRepos, err := os.ReadDir(cacheDir)
		require.NoError(t, err)
		assert.Greater(t, len(cachedRepos), 0, "Cache directory should have repos")

		// Clean up rendered files to test offline install
		os.RemoveAll(filepath.Join(workDir, ".cursor"))
		os.RemoveAll(filepath.Join(workDir, ".claude"))

		// Now try offline mode - should work from cache
		cmd = exec.Command(binPath, "install", "--yes", "--offline", "--cache-dir", cacheDir, "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Offline install failed: %s", string(output))

		// Verify files were rendered from cache
		assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/coding.md"))
	})

	t.Run("interrupted operation recovery", func(t *testing.T) {
		workDir := t.TempDir()

		// Initialize
		cmd := exec.Command(binPath, "init")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Init failed: %s", string(output))

		// Add sources
		cmd = exec.Command(binPath, "add", "file://"+acmeRepo+"#master", "--no-install", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Add failed: %s", string(output))

		// Install to create initial state
		cmd = exec.Command(binPath, "install", "--yes", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Initial install failed: %s", string(output))

		// Simulate interrupted operation by deleting lock file but keeping rendered files
		lockPath := filepath.Join(workDir, "Promptsfile.lock")
		require.NoError(t, os.Remove(lockPath))

		// Running install again should recover
		cmd = exec.Command(binPath, "install", "--yes", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Recovery install failed: %s", string(output))

		// Lock file should be recreated
		assert.FileExists(t, lockPath)

		// Verify should pass
		cmd = exec.Command(binPath, "verify", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Verify after recovery failed: %s", string(output))
	})

	t.Run("mixed pinned and unpinned sources", func(t *testing.T) {
		workDir := t.TempDir()

		// Initialize
		cmd := exec.Command(binPath, "init")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Init failed: %s", string(output))

		// Create a third test repo for this test
		extraRepo := createTestRepoWithConflict(t, "extra-repo", "rules/extra.md", "# Extra Rules")

		// Add mix of pinned and unpinned sources
		sources := []string{
			"file://" + acmeRepo + "#v1.0.0",  // Pinned to tag
			"file://" + devRepo + "#master",   // Pinned to branch (considered unpinned for updates)
			"file://" + extraRepo + "#master", // Another source at master
		}

		// Update Promptsfile directly to avoid conflicts
		cfg := &config.ExtendedConfig{
			Sources: sources,
			Adapters: config.AdaptersCfg{
				Cursor: config.CursorCfg{Enabled: true},
			},
		}
		writeWorkflowPromptsfile(t, workDir, cfg)

		// Install
		cmd = exec.Command(binPath, "install", "--yes", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Install mixed sources failed: %s", string(output))

		// Update should only affect unpinned sources
		cmd = exec.Command(binPath, "update", "--dry-run", "--allow-unknown")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Update dry-run failed: %s", string(output))

		// Should show 2 sources to update (both master branches are considered unpinned)
		assert.Contains(t, string(output), "Checking for updates to 2 source(s)")
	})
}

// Helper functions
func buildPromptSyncForWorkflow(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "prompt-sync")

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/prompt-sync")
	projectRoot := getProjectRootForWorkflow(t)
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	return binPath
}

func getProjectRootForWorkflow(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}

func readWorkflowPromptsfile(t *testing.T, dir string) *config.ExtendedConfig {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "Promptsfile"))
	require.NoError(t, err)
	var cfg config.ExtendedConfig
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err)
	return &cfg
}

func writeWorkflowPromptsfile(t *testing.T, dir string, cfg *config.ExtendedConfig) {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "Promptsfile"), data, 0644)
	require.NoError(t, err)
}

// Helper functions for edge case tests
func createTestRepoWithConflict(t *testing.T, name, filePath, content string) string {
	t.Helper()

	repoDir := filepath.Join(t.TempDir(), name)
	fileDir := filepath.Join(repoDir, filepath.Dir(filePath))
	require.NoError(t, os.MkdirAll(fileDir, 0755))

	// Create the file
	fullPath := filepath.Join(repoDir, filePath)
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	cmd.Run()

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run())

	// Create master branch (some git versions don't create it by default)
	cmd = exec.Command("git", "branch", "-M", "master")
	cmd.Dir = repoDir
	cmd.Run()

	return repoDir
}

func readWorkflowLockfile(t *testing.T, dir string) *lock.Lock {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "Promptsfile.lock"))
	require.NoError(t, err)
	var lockData lock.Lock
	err = yaml.Unmarshal(data, &lockData)
	require.NoError(t, err)
	return &lockData
}
