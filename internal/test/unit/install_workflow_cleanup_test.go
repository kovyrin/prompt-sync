package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kovyrin/prompt-sync/internal/lock"
	"github.com/kovyrin/prompt-sync/internal/workflow"
)

func TestInstallWorkflow_VersionSwitchingCleanup(t *testing.T) {
	t.Run("removes orphaned files when updating source version", func(t *testing.T) {
		// Create test workspace
		workDir := t.TempDir()

		// Create initial lock file with v1.0.0 files
		oldLock := &lock.Lock{
			Version: "1.0",
			Sources: []lock.Source{
				{
					URL:    "github.com/test/prompts",
					Ref:    "v1.0.0",
					Commit: "abc123",
					Files: []lock.File{
						{Path: ".cursor/rules/_active/authentication.md", Hash: "sha256:oldauth"},
						{Path: ".cursor/rules/_active/common.md", Hash: "sha256:common1"},
						{Path: ".claude/commands/test-authentication.md", Hash: "sha256:claudeauth"},
					},
				},
			},
		}

		// Create the old files that should be removed
		cursorDir := filepath.Join(workDir, ".cursor/rules/_active")
		claudeDir := filepath.Join(workDir, ".claude/commands")
		require.NoError(t, os.MkdirAll(cursorDir, 0755))
		require.NoError(t, os.MkdirAll(claudeDir, 0755))

		// Write old files
		require.NoError(t, os.WriteFile(filepath.Join(workDir, ".cursor/rules/_active/authentication.md"), []byte("old auth"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(workDir, ".cursor/rules/_active/common.md"), []byte("common content"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(workDir, ".claude/commands/test-authentication.md"), []byte("claude auth"), 0644))

		// Write lock file
		lockWriter := lock.New(workDir)
		require.NoError(t, lockWriter.Write(oldLock.Sources))

		// Create Promptsfile with updated version
		promptsfileContent := `sources:
  - github.com/test/prompts#v2.0.0
adapters:
  cursor:
    enabled: true
  claude:
    enabled: true
    prefix: test
`
		require.NoError(t, os.WriteFile(filepath.Join(workDir, "Promptsfile"), []byte(promptsfileContent), 0644))

		// Mock git fetcher that simulates v2.0.0 with different files
		mockFetcher := &MockGitFetcherForCleanup{
			version: "v2.0.0",
			files: map[string][]string{
				"github.com/test/prompts": {
					"prompts/auth-patterns.md",    // new file (renamed from authentication)
					"prompts/common.md",           // kept file
					"prompts/breaking-changes.md", // new file
				},
			},
		}

		// Run install with v2.0.0
		installer, err := workflow.New(workflow.InstallOptions{
			WorkspaceDir: workDir,
			AllowUnknown: true,
		})
		require.NoError(t, err)

		installer.SetGitFetcher(mockFetcher)
		err = installer.Execute()
		require.NoError(t, err)

		// Verify cleanup worked correctly
		assert.NoFileExists(t, filepath.Join(workDir, ".cursor/rules/_active/authentication.md"), "Old file should be removed")
		assert.NoFileExists(t, filepath.Join(workDir, ".claude/commands/test-authentication.md"), "Old Claude file should be removed")
		assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/common.md"), "Common file should remain")
		assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/auth-patterns.md"), "New file should exist")
		assert.FileExists(t, filepath.Join(workDir, ".cursor/rules/_active/breaking-changes.md"), "New file should exist")
	})

	t.Run("handles multiple sources correctly", func(t *testing.T) {
		// When updating one source, should only clean up files from that source
		workDir := t.TempDir()

		// Create lock with multiple sources
		oldLock := &lock.Lock{
			Version: "1.0",
			Sources: []lock.Source{
				{
					URL:    "github.com/test/prompts",
					Ref:    "v1.0.0",
					Commit: "abc123",
					Files: []lock.File{
						{Path: ".cursor/rules/_active/test.md", Hash: "sha256:test1"},
					},
				},
				{
					URL:    "github.com/other/prompts",
					Ref:    "v1.0.0",
					Commit: "def456",
					Files: []lock.File{
						{Path: ".cursor/rules/_active/other.md", Hash: "sha256:other1"},
					},
				},
			},
		}

		// Create files
		cursorDir := filepath.Join(workDir, ".cursor/rules/_active")
		require.NoError(t, os.MkdirAll(cursorDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(cursorDir, "test.md"), []byte("test"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(cursorDir, "other.md"), []byte("other"), 0644))

		lockWriter := lock.New(workDir)
		require.NoError(t, lockWriter.Write(oldLock.Sources))

		// Update only the first source
		promptsfileContent := `sources:
  - github.com/test/prompts#v2.0.0
  - github.com/other/prompts#v1.0.0
adapters:
  cursor:
    enabled: true
`
		require.NoError(t, os.WriteFile(filepath.Join(workDir, "Promptsfile"), []byte(promptsfileContent), 0644))

		// TODO: After fix is implemented
		// - test.md might be removed if not in v2.0.0
		// - other.md should remain (source not updated)
	})

	t.Run("preserves files that exist in both versions", func(t *testing.T) {
		workDir := t.TempDir()

		// Create lock with files that exist in both versions
		oldLock := &lock.Lock{
			Version: "1.0",
			Sources: []lock.Source{
				{
					URL:    "github.com/test/prompts",
					Ref:    "v1.0.0",
					Commit: "abc123",
					Files: []lock.File{
						{Path: ".cursor/rules/_active/stable.md", Hash: "sha256:stable1"},
						{Path: ".cursor/rules/_active/changing.md", Hash: "sha256:change1"},
					},
				},
			},
		}

		cursorDir := filepath.Join(workDir, ".cursor/rules/_active")
		require.NoError(t, os.MkdirAll(cursorDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(cursorDir, "stable.md"), []byte("stable content"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(cursorDir, "changing.md"), []byte("will be removed"), 0644))

		lockWriter := lock.New(workDir)
		require.NoError(t, lockWriter.Write(oldLock.Sources))

		promptsfileContent := `sources:
  - github.com/test/prompts#v2.0.0
adapters:
  cursor:
    enabled: true
`
		require.NoError(t, os.WriteFile(filepath.Join(workDir, "Promptsfile"), []byte(promptsfileContent), 0644))

		// TODO: After fix
		// - stable.md should remain (exists in both versions)
		// - changing.md should be removed (not in v2.0.0)
	})
}

// Helper function to track orphaned files during version updates
func findOrphanedFiles(oldFiles, newFiles []lock.File) []string {
	// Create a set of new file paths
	newPaths := make(map[string]bool)
	for _, f := range newFiles {
		newPaths[f.Path] = true
	}

	// Find files that exist in old but not in new
	var orphaned []string
	for _, f := range oldFiles {
		if !newPaths[f.Path] {
			orphaned = append(orphaned, f.Path)
		}
	}

	return orphaned
}

func TestFindOrphanedFiles(t *testing.T) {
	oldFiles := []lock.File{
		{Path: "a.md", Hash: "hash1"},
		{Path: "b.md", Hash: "hash2"},
		{Path: "c.md", Hash: "hash3"},
	}

	newFiles := []lock.File{
		{Path: "a.md", Hash: "hash1new"},
		{Path: "c.md", Hash: "hash3new"},
		{Path: "d.md", Hash: "hash4"},
	}

	orphaned := findOrphanedFiles(oldFiles, newFiles)

	assert.Equal(t, []string{"b.md"}, orphaned)
}

// MockGitFetcherForCleanup is a simple mock for testing cleanup functionality
type MockGitFetcherForCleanup struct {
	version string
	files   map[string][]string
}

func (m *MockGitFetcherForCleanup) Clone(url, ref string) (string, error) {
	path, _, err := m.CloneOrUpdate(url, ref)
	return path, err
}

func (m *MockGitFetcherForCleanup) Update(url, ref string) error {
	return nil
}

func (m *MockGitFetcherForCleanup) CachedPath(url, ref string) (string, bool) {
	return "", false
}

func (m *MockGitFetcherForCleanup) CloneOrUpdate(url, ref string) (string, string, error) {
	tmpDir, err := os.MkdirTemp("", "mock-repo-*")
	if err != nil {
		return "", "", err
	}

	// Create files based on the mock configuration
	if files, ok := m.files[url]; ok {
		for _, file := range files {
			filePath := filepath.Join(tmpDir, file)
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", "", err
			}
			content := "# " + filepath.Base(file) + "\nMock content"
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return "", "", err
			}
		}
	}

	return tmpDir, "mock-commit-" + m.version, nil
}
