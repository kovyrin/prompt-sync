package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/kovyrin/prompt-sync/internal/cmd"
	"github.com/kovyrin/prompt-sync/internal/config"
	"github.com/kovyrin/prompt-sync/internal/lock"
)

func TestUpdateCommand(t *testing.T) {
	// Get project root for accessing test repos
	projectRoot := getProjectRootForUpdate(t)
	acmeRepo := filepath.Join(projectRoot, "testdata/repos/acme-prompts")
	devRepo := filepath.Join(projectRoot, "testdata/repos/dev-prompts")

	t.Run("updating all prompts", func(t *testing.T) {
		// Create temp directory
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create initial Promptsfile with unpinned sources
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo,
				"file://" + devRepo,
			},
			Adapters: config.AdaptersCfg{
				Cursor: config.CursorCfg{Enabled: true},
			},
		}
		writePromptsfileForUpdate(t, tmpDir, initialConfig)

		// Create lock file with old commits
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "abc123"},
				{URL: "file://" + devRepo, Commit: "def456"},
			},
		}
		writeLockfileForUpdate(t, tmpDir, lockData)

		// Run update all with dry-run to avoid actual installation
		err := runUpdateCommandWithFlags([]string{}, map[string]interface{}{
			"dry-run":       true,
			"allow-unknown": true,
		})
		assert.NoError(t, err)
	})

	t.Run("updating specific prompts", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo,
				"file://" + devRepo,
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForUpdate(t, tmpDir, initialConfig)

		// Create lock file
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "abc123"},
				{URL: "file://" + devRepo, Commit: "def456"},
			},
		}
		writeLockfileForUpdate(t, tmpDir, lockData)

		// Update only specific sources with dry-run
		err := runUpdateCommandWithFlags([]string{"file://" + devRepo}, map[string]interface{}{
			"dry-run":       true,
			"allow-unknown": true,
		})
		assert.NoError(t, err)
	})

	t.Run("respecting version constraints", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile with pinned and unpinned sources
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo + "#v1.0.0", // Pinned to specific version
				"file://" + devRepo + "#master",  // Pinned to branch
				"file://" + acmeRepo,             // Unpinned (duplicate for testing)
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForUpdate(t, tmpDir, initialConfig)

		// Create lock file
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "abc123", Ref: "v1.0.0"},
				{URL: "file://" + devRepo, Commit: "def456", Ref: "master"},
				{URL: "file://" + acmeRepo, Commit: "ghi789"},
			},
		}
		writeLockfileForUpdate(t, tmpDir, lockData)

		// Update all with dry-run
		err := runUpdateCommandWithFlags([]string{}, map[string]interface{}{
			"dry-run":       true,
			"allow-unknown": true,
		})
		assert.NoError(t, err)

		// Verify Promptsfile wasn't changed (pinned versions should remain)
		cfg := readPromptsfileForUpdate(t, tmpDir)
		assert.Contains(t, cfg.Sources, "file://"+acmeRepo+"#v1.0.0")
		assert.Contains(t, cfg.Sources, "file://"+devRepo+"#master")
	})

	t.Run("handling breaking changes warnings", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo,
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForUpdate(t, tmpDir, initialConfig)

		// Create lock file with old version
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "oldcommit", Ref: "v1.0.0"},
			},
		}
		writeLockfileForUpdate(t, tmpDir, lockData)

		// In a real implementation, this would check for breaking changes
		// For now, just run the update with dry-run
		err := runUpdateCommandWithFlags([]string{}, map[string]interface{}{
			"dry-run":       true,
			"allow-unknown": true,
		})
		assert.NoError(t, err)
	})

	t.Run("update command requires Promptsfile to exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Don't create a Promptsfile

		err := runUpdateCommand([]string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Promptsfile not found")
	})

	t.Run("update command requires lock file to exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile but no lock file
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo,
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForUpdate(t, tmpDir, initialConfig)

		// Try to update without lock file
		err := runUpdateCommand([]string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no lock file found")
	})

	t.Run("dry run mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo,
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForUpdate(t, tmpDir, initialConfig)

		// Create lock file
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "abc123"},
			},
		}
		writeLockfileForUpdate(t, tmpDir, lockData)

		// Run with dry-run flag
		err := runUpdateCommandWithFlags([]string{}, map[string]interface{}{
			"dry-run":       true,
			"allow-unknown": true,
		})
		assert.NoError(t, err)

		// Verify lock file wasn't changed
		updatedLock := readLockfileForUpdate(t, tmpDir)
		assert.Equal(t, "abc123", updatedLock.Sources[0].Commit)
	})

	t.Run("error on non-existent source", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo,
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForUpdate(t, tmpDir, initialConfig)

		// Create lock file
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "abc123"},
			},
		}
		writeLockfileForUpdate(t, tmpDir, lockData)

		// Try to update non-existent source
		err := runUpdateCommand([]string{"github.com/org/nonexistent"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("updating with version change", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile with version that can be updated
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo + "#v1", // Major version constraint
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForUpdate(t, tmpDir, initialConfig)

		// Create lock file with old version
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "abc123", Ref: "v1.0.0"},
			},
		}
		writeLockfileForUpdate(t, tmpDir, lockData)

		// Update - should update to latest v1.x.x (dry-run to avoid actual update)
		err := runUpdateCommandWithFlags([]string{}, map[string]interface{}{
			"dry-run":       true,
			"allow-unknown": true,
		})
		assert.NoError(t, err)
	})

	t.Run("force update pinned sources", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile with pinned source
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo + "#v1.0.0", // Pinned to specific version
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForUpdate(t, tmpDir, initialConfig)

		// Create lock file
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "abc123", Ref: "v1.0.0"},
			},
		}
		writeLockfileForUpdate(t, tmpDir, lockData)

		// Force update with dry-run
		err := runUpdateCommandWithFlags([]string{"file://" + acmeRepo}, map[string]interface{}{
			"force":         true,
			"dry-run":       true,
			"allow-unknown": true,
		})
		assert.NoError(t, err)
	})

	t.Run("update without force fails for pinned source", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile with pinned source
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"file://" + acmeRepo + "#v1.0.0", // Pinned to specific version
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForUpdate(t, tmpDir, initialConfig)

		// Create lock file
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{URL: "file://" + acmeRepo, Commit: "abc123", Ref: "v1.0.0"},
			},
		}
		writeLockfileForUpdate(t, tmpDir, lockData)

		// Try to update without force - should fail
		err := runUpdateCommand([]string{"file://" + acmeRepo})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pinned to a specific version")
		assert.Contains(t, err.Error(), "--force")
	})
}

// Helper functions

func getProjectRootForUpdate(t *testing.T) string {
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

func writePromptsfileForUpdate(t *testing.T, dir string, cfg *config.ExtendedConfig) {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "Promptsfile"), data, 0644)
	require.NoError(t, err)
}

func readPromptsfileForUpdate(t *testing.T, dir string) *config.ExtendedConfig {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "Promptsfile"))
	require.NoError(t, err)
	var cfg config.ExtendedConfig
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err)
	return &cfg
}

func writeLockfileForUpdate(t *testing.T, dir string, lockData *lock.Lock) {
	t.Helper()
	data, err := yaml.Marshal(lockData)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "Promptsfile.lock"), data, 0644)
	require.NoError(t, err)
}

func readLockfileForUpdate(t *testing.T, dir string) *lock.Lock {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "Promptsfile.lock"))
	require.NoError(t, err)
	var lockData lock.Lock
	err = yaml.Unmarshal(data, &lockData)
	require.NoError(t, err)
	return &lockData
}

func runUpdateCommand(args []string) error {
	// Create a new command instance to avoid flag conflicts
	rootCmd := &cobra.Command{Use: "prompt-sync"}
	updateCmd := cmd.NewUpdateCommand()
	rootCmd.AddCommand(updateCmd)

	rootCmd.SetArgs(append([]string{"update"}, args...))
	return rootCmd.Execute()
}

func runUpdateCommandWithFlags(args []string, flags map[string]interface{}) error {
	// Create a new command instance
	rootCmd := &cobra.Command{Use: "prompt-sync"}
	updateCmd := cmd.NewUpdateCommand()
	rootCmd.AddCommand(updateCmd)

	// Build args with flags
	cmdArgs := []string{"update"}
	for flag, value := range flags {
		switch v := value.(type) {
		case bool:
			if v {
				cmdArgs = append(cmdArgs, "--"+flag)
			}
		case string:
			cmdArgs = append(cmdArgs, "--"+flag, v)
		}
	}
	cmdArgs = append(cmdArgs, args...)

	rootCmd.SetArgs(cmdArgs)
	return rootCmd.Execute()
}
