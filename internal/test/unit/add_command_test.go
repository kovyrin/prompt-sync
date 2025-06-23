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
)

func TestAddCommand(t *testing.T) {
	t.Run("adding valid prompt sources", func(t *testing.T) {
		// Create temp directory
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create initial Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"github.com/org/existing-prompts",
			},
			Adapters: config.AdaptersCfg{
				Cursor: config.CursorCfg{Enabled: true},
			},
		}
		writePromptsfile(t, tmpDir, initialConfig)

		// Test adding a new source (with --no-install to avoid cloning)
		err := runAddCommandWithFlags([]string{"github.com/org/new-prompts"}, map[string]interface{}{
			"no-install": true,
		})
		assert.NoError(t, err)

		// Verify the source was added
		cfg := readPromptsfile(t, tmpDir)
		assert.Contains(t, cfg.Sources, "github.com/org/existing-prompts")
		assert.Contains(t, cfg.Sources, "github.com/org/new-prompts")
		assert.Len(t, cfg.Sources, 2)
	})

	t.Run("adding source with version specification", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create initial Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources:  []string{},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfile(t, tmpDir, initialConfig)

		// Test adding sources with different ref specifications
		testCases := []struct {
			input    string
			expected string
		}{
			{
				input:    "github.com/org/prompts#v1.0.0",
				expected: "github.com/org/prompts#v1.0.0",
			},
			{
				input:    "github.com/org/prompts#main",
				expected: "github.com/org/prompts#main",
			},
			{
				input:    "github.com/org/prompts#feature/new-prompt",
				expected: "github.com/org/prompts#feature/new-prompt",
			},
		}

		for _, tc := range testCases {
			// Reset sources
			writePromptsfile(t, tmpDir, initialConfig)

			err := runAddCommandWithFlags([]string{tc.input}, map[string]interface{}{
				"no-install": true,
			})
			assert.NoError(t, err)

			cfg := readPromptsfile(t, tmpDir)
			assert.Contains(t, cfg.Sources, tc.expected)
		}
	})

	t.Run("rejecting untrusted sources without --allow-unknown", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create initial Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources:  []string{},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfile(t, tmpDir, initialConfig)

		// Try to add untrusted source
		err := runAddCommand([]string{"github.com/untrusted/prompts"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "untrusted source")

		// Verify source was not added
		cfg := readPromptsfile(t, tmpDir)
		assert.Empty(t, cfg.Sources)
	})

	t.Run("allowing untrusted sources with --allow-unknown", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create initial Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources:  []string{},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfile(t, tmpDir, initialConfig)

		// Add untrusted source with flag
		err := runAddCommandWithFlags([]string{"github.com/untrusted/prompts"}, map[string]interface{}{
			"allow-unknown": true,
			"no-install":    true,
		})
		assert.NoError(t, err)

		// Verify source was added
		cfg := readPromptsfile(t, tmpDir)
		assert.Contains(t, cfg.Sources, "github.com/untrusted/prompts")
	})

	t.Run("handling duplicate prompt names", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create initial Promptsfile with a source
		initialConfig := &config.ExtendedConfig{
			Sources: []string{
				"github.com/org/prompts",
			},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfile(t, tmpDir, initialConfig)

		// Try to add the same source again
		err := runAddCommand([]string{"github.com/org/prompts"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")

		// Verify no duplicate was added
		cfg := readPromptsfile(t, tmpDir)
		assert.Len(t, cfg.Sources, 1)
		assert.Equal(t, "github.com/org/prompts", cfg.Sources[0])
	})

	t.Run("adding source with --no-install flag", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create initial Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources:  []string{},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfile(t, tmpDir, initialConfig)

		// Add source with --no-install
		err := runAddCommandWithFlags([]string{"github.com/org/prompts"}, map[string]interface{}{
			"no-install": true,
		})
		assert.NoError(t, err)

		// Verify source was added
		cfg := readPromptsfile(t, tmpDir)
		assert.Contains(t, cfg.Sources, "github.com/org/prompts")

		// Verify no lock file was created (install was not triggered)
		_, err = os.Stat(filepath.Join(tmpDir, "Promptsfile.lock"))
		assert.True(t, os.IsNotExist(err), "Lock file should not exist with --no-install")
	})

	t.Run("invalid source URL format", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Create initial Promptsfile
		initialConfig := &config.ExtendedConfig{
			Sources:  []string{},
			Adapters: config.AdaptersCfg{Cursor: config.CursorCfg{Enabled: true}},
		}
		writePromptsfile(t, tmpDir, initialConfig)

		// Test invalid URLs
		invalidURLs := []string{
			"not-a-url",
			"http://example.com", // Not a git URL
			"",                   // Empty
			"github.com/",        // Incomplete
		}

		for _, url := range invalidURLs {
			err := runAddCommand([]string{url})
			assert.Error(t, err, "Should error on invalid URL: %s", url)

			// Verify nothing was added
			cfg := readPromptsfile(t, tmpDir)
			assert.Empty(t, cfg.Sources)
		}
	})

	t.Run("add command requires Promptsfile to exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(oldWd)

		// Don't create a Promptsfile

		err := runAddCommand([]string{"github.com/org/prompts"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Promptsfile not found")
	})
}

// Helper functions

func writePromptsfile(t *testing.T, dir string, cfg *config.ExtendedConfig) {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "Promptsfile"), data, 0644)
	require.NoError(t, err)
}

func readPromptsfile(t *testing.T, dir string) *config.ExtendedConfig {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "Promptsfile"))
	require.NoError(t, err)
	var cfg config.ExtendedConfig
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err)
	return &cfg
}

func runAddCommand(args []string) error {
	// Create a new command instance to avoid flag conflicts
	rootCmd := &cobra.Command{Use: "prompt-sync"}
	addCmd := cmd.NewAddCommand()
	rootCmd.AddCommand(addCmd)

	rootCmd.SetArgs(append([]string{"add"}, args...))
	return rootCmd.Execute()
}

func runAddCommandWithFlags(args []string, flags map[string]interface{}) error {
	// Create a new command instance
	rootCmd := &cobra.Command{Use: "prompt-sync"}
	addCmd := cmd.NewAddCommand()
	rootCmd.AddCommand(addCmd)

	// Build args with flags
	cmdArgs := []string{"add"}
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
