package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kovyrin/prompt-sync/internal/config"
	"github.com/kovyrin/prompt-sync/internal/lock"
	"github.com/kovyrin/prompt-sync/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestFileTrackingIntegration(t *testing.T) {
	t.Run("lock file tracks source paths for all rendered files", func(t *testing.T) {
		// Setup
		workspace := t.TempDir()
		cacheDir := t.TempDir()

		// Create test repository structure
		testRepo := filepath.Join(t.TempDir(), "test-repo")
		require.NoError(t, os.MkdirAll(filepath.Join(testRepo, "prompts"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(testRepo, "rules"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(testRepo, "commands"), 0755))

		// Create test files
		testFiles := map[string]string{
			"prompts/auth.md":      "# Authentication Rules",
			"rules/security.md":    "# Security Guidelines",
			"commands/validate.md": "# Validation Commands",
		}

		for path, content := range testFiles {
			fullPath := filepath.Join(testRepo, path)
			require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
		}

		// Initialize as git repo
		initGitRepo(t, testRepo)

		// Create Promptsfile
		promptsfile := &config.ExtendedConfig{
			Sources: []string{testRepo},
			Adapters: config.AdaptersCfg{
				Cursor: config.CursorCfg{Enabled: true},
				Claude: config.ClaudeCfg{
					Enabled: true,
					Prefix:  "test",
				},
			},
		}

		promptsPath := filepath.Join(workspace, "Promptsfile")
		writePromptsfile(t, promptsPath, promptsfile)

		// Create mock git fetcher
		mockFetcher := &mockGitFetcher{
			repos: map[string]string{
				testRepo: testRepo,
			},
		}

		// Run installation
		opts := workflow.InstallOptions{
			WorkspaceDir: workspace,
			CacheDir:     cacheDir,
			AllowUnknown: true, // Allow test repos from temp directories
		}

		installer, err := workflow.New(opts)
		require.NoError(t, err)
		installer.SetGitFetcher(mockFetcher)

		err = installer.Execute()
		require.NoError(t, err)

		// Read lock file
		lockWriter := lock.New(workspace)
		lockData, err := lockWriter.Read()
		require.NoError(t, err)
		require.NotNil(t, lockData)

		// Verify source path tracking
		assert.Len(t, lockData.Sources, 1)
		source := lockData.Sources[0]
		assert.Equal(t, testRepo, source.URL)

		// Build expected mappings
		expectedMappings := map[string]string{
			// Cursor adapter outputs
			".cursor/rules/_active/auth.md":     "prompts/auth.md",
			".cursor/rules/_active/security.md": "rules/security.md",
			// Claude adapter outputs
			".claude/commands/test-validate.md": "commands/validate.md",
		}

		// Verify all expected files are tracked with source paths
		actualMappings := make(map[string]string)
		for _, file := range source.Files {
			actualMappings[file.Path] = file.SourcePath
		}

		for outputPath, sourcePath := range expectedMappings {
			assert.Equal(t, sourcePath, actualMappings[outputPath],
				"Expected %s to map to %s", outputPath, sourcePath)
		}
	})

	t.Run("source file changes are detected through mapping", func(t *testing.T) {
		// Setup
		workspace := t.TempDir()
		cacheDir := t.TempDir()

		// Create initial repository
		testRepo := filepath.Join(t.TempDir(), "change-test-repo")
		require.NoError(t, os.MkdirAll(filepath.Join(testRepo, "prompts"), 0755))

		// Initial file
		initialPath := filepath.Join(testRepo, "prompts", "initial.md")
		require.NoError(t, os.WriteFile(initialPath, []byte("# Initial"), 0644))

		initGitRepo(t, testRepo)

		// Create Promptsfile
		promptsfile := &config.ExtendedConfig{
			Sources: []string{testRepo},
			Adapters: config.AdaptersCfg{
				Cursor: config.CursorCfg{Enabled: true},
			},
		}

		promptsPath := filepath.Join(workspace, "Promptsfile")
		writePromptsfile(t, promptsPath, promptsfile)

		// Mock fetcher
		mockFetcher := &mockGitFetcher{
			repos: map[string]string{
				testRepo: testRepo,
			},
			commits: make(map[string]string),
		}

		// Initial installation
		opts := workflow.InstallOptions{
			WorkspaceDir: workspace,
			CacheDir:     cacheDir,
			AllowUnknown: true, // Allow test repos from temp directories
		}

		installer, err := workflow.New(opts)
		require.NoError(t, err)
		installer.SetGitFetcher(mockFetcher)

		err = installer.Execute()
		require.NoError(t, err)

		// Verify initial state
		lockWriter := lock.New(workspace)
		files, err := lockWriter.GetFilesBySource(testRepo)
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Equal(t, "prompts/initial.md", files[0].SourcePath)

		// Simulate source file rename
		newPath := filepath.Join(testRepo, "prompts", "renamed.md")
		require.NoError(t, os.Rename(initialPath, newPath))

		// Update commit to simulate repository change
		mockFetcher.commits[testRepo] = "newcommit"

		// Re-run installation
		err = installer.Execute()
		require.NoError(t, err)

		// Verify the change is reflected
		files, err = lockWriter.GetFilesBySource(testRepo)
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Equal(t, "prompts/renamed.md", files[0].SourcePath)

		// Old file should be cleaned up
		oldFile := filepath.Join(workspace, ".cursor/rules/_active/initial.md")
		assert.NoFileExists(t, oldFile)

		// New file should exist
		newFile := filepath.Join(workspace, ".cursor/rules/_active/renamed.md")
		assert.FileExists(t, newFile)
	})
}

// Mock git fetcher for testing
type mockGitFetcher struct {
	repos   map[string]string // URL -> local path
	commits map[string]string // URL -> commit hash
}

func (m *mockGitFetcher) CloneOrUpdate(url, ref string) (string, string, error) {
	localPath, ok := m.repos[url]
	if !ok {
		return "", "", fmt.Errorf("repository not found: %s", url)
	}

	commit := "testcommit"
	if c, ok := m.commits[url]; ok {
		commit = c
	}

	return localPath, commit, nil
}

func (m *mockGitFetcher) Clone(url, ref string) (string, error) {
	localPath, ok := m.repos[url]
	if !ok {
		return "", fmt.Errorf("repository not found: %s", url)
	}
	return localPath, nil
}

func (m *mockGitFetcher) Update(url, ref string) error {
	if _, ok := m.repos[url]; !ok {
		return fmt.Errorf("repository not found: %s", url)
	}
	return nil
}

func (m *mockGitFetcher) CachedPath(url, ref string) (string, bool) {
	localPath, ok := m.repos[url]
	return localPath, ok
}

// Helper to initialize a git repo
func initGitRepo(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = path
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = path
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = path
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = path
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = path
	require.NoError(t, cmd.Run())
}

// Helper to write Promptsfile
func writePromptsfile(t *testing.T, path string, cfg *config.ExtendedConfig) {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))
}
