package unit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kovyrin/prompt-sync/internal/conflict"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConflictDetector(t *testing.T) {
	t.Run("detects duplicate basenames", func(t *testing.T) {
		dir := t.TempDir()

		// Create files with duplicate basenames in different directories
		subdir1 := filepath.Join(dir, "adapter1")
		subdir2 := filepath.Join(dir, "adapter2")
		require.NoError(t, os.MkdirAll(subdir1, 0755))
		require.NoError(t, os.MkdirAll(subdir2, 0755))

		// Same basename in different directories
		require.NoError(t, os.WriteFile(filepath.Join(subdir1, "coding.md"), []byte("content1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(subdir2, "coding.md"), []byte("content2"), 0644))

		// Unique file
		require.NoError(t, os.WriteFile(filepath.Join(subdir1, "unique.md"), []byte("unique"), 0644))

		detector := conflict.New(false)
		issues, err := detector.ScanDirectory(dir)
		require.NoError(t, err)

		// Should find one duplicate issue
		assert.Len(t, issues, 1)
		assert.Equal(t, "duplicate", issues[0].Type)
		assert.Equal(t, "coding.md", issues[0].Path)
		assert.Contains(t, issues[0].Details, "adapter1/coding.md")
		assert.Contains(t, issues[0].Details, "adapter2/coding.md")
		assert.True(t, issues[0].IsCritical)
	})

	t.Run("no issues when no duplicates", func(t *testing.T) {
		dir := t.TempDir()

		// Create files with unique basenames
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file1.md"), []byte("content1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file2.md"), []byte("content2"), 0644))

		detector := conflict.New(false)
		issues, err := detector.ScanDirectory(dir)
		require.NoError(t, err)

		assert.Empty(t, issues)
	})

	t.Run("detects hash drift", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "file1.md")
		file2 := filepath.Join(dir, "file2.md")

		// Create files
		require.NoError(t, os.WriteFile(file1, []byte("original content"), 0644))
		require.NoError(t, os.WriteFile(file2, []byte("unchanged"), 0644))

		// Expected hashes (file1 has wrong hash, file2 has correct hash)
		expectedHashes := map[string]string{
			file1: "sha256:wronghash1234567890abcdef1234567890abcdef1234567890abcdef12345678",
			file2: "sha256:aaa8d3c8d74ad3e8f6b1772aa9c7e0eaa528cb42fc93599ce2f125b00d4c424c", // actual hash of "unchanged"
		}

		detector := conflict.New(false)
		issues, err := detector.CheckDrift(expectedHashes)
		require.NoError(t, err)

		// Should find one drift issue for file1
		assert.Len(t, issues, 1)
		assert.Equal(t, "drift", issues[0].Type)
		assert.Equal(t, file1, issues[0].Path)
		assert.Contains(t, issues[0].Details, "hash mismatch")
		assert.True(t, issues[0].IsCritical)
	})

	t.Run("detects missing files", func(t *testing.T) {
		dir := t.TempDir()
		missingFile := filepath.Join(dir, "missing.md")

		expectedHashes := map[string]string{
			missingFile: "sha256:somehash",
		}

		detector := conflict.New(false)
		issues, err := detector.CheckDrift(expectedHashes)
		require.NoError(t, err)

		assert.Len(t, issues, 1)
		assert.Equal(t, "drift", issues[0].Type)
		assert.Equal(t, missingFile, issues[0].Path)
		assert.Equal(t, "file missing", issues[0].Details)
		assert.True(t, issues[0].IsCritical)
	})

	t.Run("filters critical issues in strict mode", func(t *testing.T) {
		// Create issues with different criticality
		issues := []conflict.Issue{
			{Type: "duplicate", Path: "file1.md", IsCritical: true},
			{Type: "warning", Path: "file2.md", IsCritical: false},
			{Type: "drift", Path: "file3.md", IsCritical: true},
		}

		// Non-strict mode returns all
		detector := conflict.New(false)
		filtered := detector.FilterCritical(issues)
		assert.Len(t, filtered, 3)

		// Strict mode returns only critical
		strictDetector := conflict.New(true)
		filtered = strictDetector.FilterCritical(issues)
		assert.Len(t, filtered, 2)
		for _, issue := range filtered {
			assert.True(t, issue.IsCritical)
		}
	})
}
