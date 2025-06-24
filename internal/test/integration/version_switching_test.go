package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kovyrin/prompt-sync/internal/config"
	"github.com/kovyrin/prompt-sync/internal/lock"
	"github.com/kovyrin/prompt-sync/internal/workflow"
)

// MockVersionedGitFetcher simulates different versions of a repository
type MockVersionedGitFetcher struct {
	clonedRepos map[string]string
	versions    map[string]map[string][]string // url -> version -> files
}

func NewMockVersionedGitFetcher() *MockVersionedGitFetcher {
	return &MockVersionedGitFetcher{
		clonedRepos: make(map[string]string),
		versions: map[string]map[string][]string{
			"github.com/test/prompts": {
				"v1.0.0": {
					"prompts/authentication.md",
					"prompts/common.md",
				},
				"v2.0.0": {
					"prompts/auth-patterns.md",    // renamed from authentication.md
					"prompts/common.md",           // kept
					"prompts/breaking-changes.md", // new
				},
			},
		},
	}
}

func (m *MockVersionedGitFetcher) Clone(url, ref string) (string, error) {
	path, _, err := m.CloneOrUpdate(url, ref)
	return path, err
}

func (m *MockVersionedGitFetcher) Update(url, ref string) error {
	_, _, err := m.CloneOrUpdate(url, ref)
	return err
}

func (m *MockVersionedGitFetcher) CachedPath(url, ref string) (string, bool) {
	if path, exists := m.clonedRepos[url+"#"+ref]; exists {
		return path, true
	}
	return "", false
}

func (m *MockVersionedGitFetcher) CloneOrUpdate(url, ref string) (string, string, error) {
	// Create a temp directory for this "clone"
	tmpDir, err := os.MkdirTemp("", "mock-repo-*")
	if err != nil {
		return "", "", err
	}

	// Determine which files to create based on version
	version := ref
	if version == "" {
		version = "v2.0.0" // default to latest
	}

	files, exists := m.versions[url][version]
	if !exists {
		// Default behavior for unknown repos
		files = []string{"prompts/default.md"}
	}

	// Create the files
	for _, file := range files {
		filePath := filepath.Join(tmpDir, file)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", "", err
		}

		content := "# " + filepath.Base(file) + "\nContent for " + file + " at " + version
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return "", "", err
		}
	}

	// Mock commit hash
	commitHash := "abc123-" + version

	m.clonedRepos[url+"#"+ref] = tmpDir
	return tmpDir, commitHash, nil
}

func (m *MockVersionedGitFetcher) Cleanup() {
	for _, dir := range m.clonedRepos {
		os.RemoveAll(dir)
	}
}

func TestVersionSwitchingWithCleanup(t *testing.T) {
	// Create workspace
	workDir := t.TempDir()

	// Create initial Promptsfile with v1.0.0
	cfg := &config.ExtendedConfig{
		Sources: []string{"github.com/test/prompts#v1.0.0"},
		Adapters: config.AdaptersCfg{
			Cursor: config.CursorCfg{Enabled: true},
			Claude: config.ClaudeCfg{Enabled: false},
		},
	}

	configPath := filepath.Join(workDir, "Promptsfile")
	require.NoError(t, writeTestConfig(configPath, cfg))

	// Create mock git fetcher
	mockFetcher := NewMockVersionedGitFetcher()
	defer mockFetcher.Cleanup()

	// First install with v1.0.0
	installer1, err := workflow.New(workflow.InstallOptions{
		WorkspaceDir: workDir,
		AllowUnknown: true,
	})
	require.NoError(t, err)

	// Replace git fetcher with our mock
	installer1.SetGitFetcher(mockFetcher)

	err = installer1.Execute()
	require.NoError(t, err)

	// Verify v1.0.0 files exist
	assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/authentication.md"))
	assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/common.md"))
	assert.NoFileExists(t, filepath.Join(workDir, ".cursor/rules/_active/auth-patterns.md"))
	assert.NoFileExists(t, filepath.Join(workDir, ".cursor/rules/_active/breaking-changes.md"))

	// Update to v2.0.0
	cfg.Sources = []string{"github.com/test/prompts#v2.0.0"}
	require.NoError(t, writeTestConfig(configPath, cfg))

	// Second install with v2.0.0
	installer2, err := workflow.New(workflow.InstallOptions{
		WorkspaceDir: workDir,
		AllowUnknown: true,
	})
	require.NoError(t, err)

	installer2.SetGitFetcher(mockFetcher)

	err = installer2.Execute()
	require.NoError(t, err)

	// Verify v2.0.0 files exist and v1.0.0-only files are removed
	assert.NoFileExists(t, filepath.Join(workDir, ".cursor/rules/_active/authentication.md"), "Old file should be removed")
	assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/auth-patterns.md"), "New file should exist")
	assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/common.md"), "Common file should remain")
	assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/breaking-changes.md"), "New file should exist")

	// Verify lock file is updated correctly
	lockWriter := lock.New(workDir)
	lockData, err := lockWriter.Read()
	require.NoError(t, err)
	require.NotNil(t, lockData)

	assert.Len(t, lockData.Sources, 1)
	assert.Equal(t, "github.com/test/prompts", lockData.Sources[0].URL)
	assert.Equal(t, "v2.0.0", lockData.Sources[0].Ref)
	assert.Len(t, lockData.Sources[0].Files, 3) // auth-patterns, common, breaking-changes
}

func TestVersionSwitchingWithMultipleSources(t *testing.T) {
	// Create workspace
	workDir := t.TempDir()

	// Create initial Promptsfile with two sources
	cfg := &config.ExtendedConfig{
		Sources: []string{
			"github.com/test/prompts#v1.0.0",
			"github.com/other/prompts#v1.0.0",
		},
		Adapters: config.AdaptersCfg{
			Cursor: config.CursorCfg{Enabled: true},
		},
	}

	configPath := filepath.Join(workDir, "Promptsfile")
	require.NoError(t, writeTestConfig(configPath, cfg))

	// Create mock git fetcher with multiple repos
	mockFetcher := &MockVersionedGitFetcher{
		clonedRepos: make(map[string]string),
		versions: map[string]map[string][]string{
			"github.com/test/prompts": {
				"v1.0.0": {"prompts/test-old.md"},
				"v2.0.0": {"prompts/test-new.md"},
			},
			"github.com/other/prompts": {
				"v1.0.0": {"prompts/other.md"},
			},
		},
	}
	defer mockFetcher.Cleanup()

	// First install
	installer1, err := workflow.New(workflow.InstallOptions{
		WorkspaceDir: workDir,
		AllowUnknown: true,
	})
	require.NoError(t, err)
	installer1.SetGitFetcher(mockFetcher)

	err = installer1.Execute()
	require.NoError(t, err)

	// Verify initial files
	assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/test-old.md"))
	assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/other.md"))

	// Update only the first source
	cfg.Sources = []string{
		"github.com/test/prompts#v2.0.0",
		"github.com/other/prompts#v1.0.0",
	}
	require.NoError(t, writeTestConfig(configPath, cfg))

	// Second install
	installer2, err := workflow.New(workflow.InstallOptions{
		WorkspaceDir: workDir,
		AllowUnknown: true,
	})
	require.NoError(t, err)
	installer2.SetGitFetcher(mockFetcher)

	err = installer2.Execute()
	require.NoError(t, err)

	// Verify only test source files were updated
	assert.NoFileExists(t, filepath.Join(workDir, ".cursor/rules/_active/test-old.md"), "Old test file should be removed")
	assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/test-new.md"), "New test file should exist")
	assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/other.md"), "Other source file should remain")
}

// Helper function to write test config
func writeTestConfig(path string, cfg *config.ExtendedConfig) error {
	// Simple YAML writer for test
	content := "sources:\n"
	for _, s := range cfg.Sources {
		content += "  - " + s + "\n"
	}
	content += "adapters:\n"
	content += "  cursor:\n"
	content += "    enabled: " + fmt.Sprintf("%v", cfg.Adapters.Cursor.Enabled) + "\n"
	content += "  claude:\n"
	content += "    enabled: " + fmt.Sprintf("%v", cfg.Adapters.Claude.Enabled) + "\n"
	if cfg.Adapters.Claude.Prefix != "" {
		content += "    prefix: " + cfg.Adapters.Claude.Prefix + "\n"
	}

	return os.WriteFile(path, []byte(content), 0644)
}
