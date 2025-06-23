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

func TestRemoveCommand(t *testing.T) {
	t.Run("removing existing prompts", func(t *testing.T) {
		// Create temp directory
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create initial Promptsfile with multiple sources
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"github.com/org/prompts1",
				"github.com/org/prompts2",
				"github.com/org/prompts3",
			},
			Adapters: config.AdaptersCfg{
				Cursor: config.CursorCfg{Enabled: true},
			},
		}
		writePromptsfileForRemove(t, tmpDir, initialConfig)

		// Remove the middle source
		err := runRemoveCommand([]string{"github.com/org/prompts2"})
		assert.NoError(t, err)

		// Verify the source was removed
		cfg := readPromptsfileForRemove(t, tmpDir)
		assert.NotContains(t, cfg.Sources, "github.com/org/prompts2")
		assert.Contains(t, cfg.Sources, "github.com/org/prompts1")
		assert.Contains(t, cfg.Sources, "github.com/org/prompts3")
		assert.Len(t, cfg.Sources, 2)
	})

	t.Run("removing source with version specification", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile with versioned source
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"github.com/org/prompts#v1.0.0",
				"github.com/org/other-prompts",
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForRemove(t, tmpDir, initialConfig)

		// Remove by base URL (without version)
		err := runRemoveCommand([]string{"github.com/org/prompts"})
		assert.NoError(t, err)

		// Verify it was removed
		cfg := readPromptsfileForRemove(t, tmpDir)
		assert.NotContains(t, cfg.Sources, "github.com/org/prompts#v1.0.0")
		assert.Contains(t, cfg.Sources, "github.com/org/other-prompts")
	})

	t.Run("handling non-existent prompts gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"github.com/org/prompts",
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForRemove(t, tmpDir, initialConfig)

		// Try to remove non-existent source
		err := runRemoveCommand([]string{"github.com/org/nonexistent"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Verify nothing was changed
		cfg := readPromptsfileForRemove(t, tmpDir)
		assert.Len(t, cfg.Sources, 1)
		assert.Equal(t, "github.com/org/prompts", cfg.Sources[0])
	})

	t.Run("cleaning up rendered files", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"github.com/org/prompts",
			},
			Adapters: config.AdaptersCfg{
				Cursor: config.CursorCfg{Enabled: true},
				Claude: config.ClaudeCfg{Enabled: true, Prefix: "test"},
			},
		}
		writePromptsfileForRemove(t, tmpDir, initialConfig)

		// Create lock file with rendered file paths
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{
					URL:    "github.com/org/prompts",
					Commit: "abc123",
					Files: []lock.File{
						{Path: ".cursor/rules/_active/rule1.md", Hash: "hash1"},
						{Path: ".cursor/rules/_active/rule2.md", Hash: "hash2"},
						{Path: ".claude/commands/test-cmd1.md", Hash: "hash3"},
					},
				},
			},
		}
		writeLockfile(t, tmpDir, lockData)

		// Create the rendered files
		cursorDir := filepath.Join(tmpDir, ".cursor/rules/_active")
		claudeDir := filepath.Join(tmpDir, ".claude/commands")
		require.NoError(t, os.MkdirAll(cursorDir, 0755))
		require.NoError(t, os.MkdirAll(claudeDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(cursorDir, "rule1.md"), []byte("content1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(cursorDir, "rule2.md"), []byte("content2"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "test-cmd1.md"), []byte("content3"), 0644))

		// Remove the source
		err := runRemoveCommand([]string{"github.com/org/prompts"})
		assert.NoError(t, err)

		// Verify files were deleted
		assert.NoFileExists(t, filepath.Join(cursorDir, "rule1.md"))
		assert.NoFileExists(t, filepath.Join(cursorDir, "rule2.md"))
		assert.NoFileExists(t, filepath.Join(claudeDir, "test-cmd1.md"))
	})

	t.Run("updating lock file after removal", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile with multiple sources
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"github.com/org/prompts1",
				"github.com/org/prompts2",
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForRemove(t, tmpDir, initialConfig)

		// Create lock file
		lockData := &lock.Lock{
			Sources: []lock.Source{
				{
					URL:    "github.com/org/prompts1",
					Commit: "abc123",
					Files:  []lock.File{{Path: ".cursor/rules/_active/rule1.md", Hash: "hash1"}},
				},
				{
					URL:    "github.com/org/prompts2",
					Commit: "def456",
					Files:  []lock.File{{Path: ".cursor/rules/_active/rule2.md", Hash: "hash2"}},
				},
			},
		}
		writeLockfile(t, tmpDir, lockData)

		// Remove one source
		err := runRemoveCommand([]string{"github.com/org/prompts2"})
		assert.NoError(t, err)

		// Verify lock file was updated
		updatedLock := readLockfile(t, tmpDir)
		assert.Len(t, updatedLock.Sources, 1)
		assert.Equal(t, "github.com/org/prompts1", updatedLock.Sources[0].URL)
		assert.NotContains(t, updatedLock.Sources, lock.Source{URL: "github.com/org/prompts2"})
	})

	t.Run("remove command requires Promptsfile to exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Don't create a Promptsfile

		err := runRemoveCommand([]string{"github.com/org/prompts"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Promptsfile not found")
	})

	t.Run("removing last source", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile with one source
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"github.com/org/prompts",
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForRemove(t, tmpDir, initialConfig)

		// Remove the only source
		err := runRemoveCommand([]string{"github.com/org/prompts"})
		assert.NoError(t, err)

		// Verify sources is empty
		cfg := readPromptsfileForRemove(t, tmpDir)
		assert.Empty(t, cfg.Sources)
	})

	t.Run("removing source from overlays", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile with overlay
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"github.com/org/prompts",
			},
			Overlays: []config.Overlay{
				{
					Scope:  "personal",
					Source: "github.com/personal/prompts",
				},
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForRemove(t, tmpDir, initialConfig)

		// Try to remove overlay source
		err := runRemoveCommand([]string{"github.com/personal/prompts"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "overlay")

		// Verify nothing was changed
		cfg := readPromptsfileForRemove(t, tmpDir)
		assert.Len(t, cfg.Overlays, 1)
	})

	t.Run("partial URL matching", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"github.com/org/prompts#v1.0.0",
				"github.com/org/prompts-utils",
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfileForRemove(t, tmpDir, initialConfig)

		// Remove by exact base URL
		err := runRemoveCommand([]string{"github.com/org/prompts"})
		assert.NoError(t, err)

		// Verify only the exact match was removed
		cfg := readPromptsfileForRemove(t, tmpDir)
		assert.NotContains(t, cfg.Sources, "github.com/org/prompts#v1.0.0")
		assert.Contains(t, cfg.Sources, "github.com/org/prompts-utils")
	})
}

// Helper functions

func writePromptsfileForRemove(t *testing.T, dir string, cfg *config.ExtendedConfig) {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "Promptsfile"), data, 0644)
	require.NoError(t, err)
}

func readPromptsfileForRemove(t *testing.T, dir string) *config.ExtendedConfig {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "Promptsfile"))
	require.NoError(t, err)
	var cfg config.ExtendedConfig
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err)
	return &cfg
}

func writeLockfile(t *testing.T, dir string, lockData *lock.Lock) {
	t.Helper()
	data, err := yaml.Marshal(lockData)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "Promptsfile.lock"), data, 0644)
	require.NoError(t, err)
}

func readLockfile(t *testing.T, dir string) *lock.Lock {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "Promptsfile.lock"))
	require.NoError(t, err)
	var lockData lock.Lock
	err = yaml.Unmarshal(data, &lockData)
	require.NoError(t, err)
	return &lockData
}

func runRemoveCommand(args []string) error {
	// Create a new command instance to avoid flag conflicts
	rootCmd := &cobra.Command{Use: "prompt-sync"}
	removeCmd := cmd.NewRemoveCommand()
	rootCmd.AddCommand(removeCmd)

	rootCmd.SetArgs(append([]string{"remove"}, args...))
	return rootCmd.Execute()
}

func runRemoveCommandWithFlags(args []string, flags map[string]interface{}) error {
	// Create a new command instance
	rootCmd := &cobra.Command{Use: "prompt-sync"}
	removeCmd := cmd.NewRemoveCommand()
	rootCmd.AddCommand(removeCmd)

	// Build args with flags
	cmdArgs := []string{"remove"}
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
