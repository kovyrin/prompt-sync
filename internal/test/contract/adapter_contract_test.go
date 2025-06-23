package contract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kovyrin/prompt-sync/internal/adapter"
	"github.com/kovyrin/prompt-sync/internal/adapter/claude"
	"github.com/kovyrin/prompt-sync/internal/adapter/cursor"
)

// TestAdapterContract defines the expected behavior for any AgentAdapter
// implementation. This test ensures all adapters follow the same interface.
func TestAdapterContract(t *testing.T) {
	// Test data - a simple prompt pack
	pack := adapter.PromptPack{
		Name: "test-pack",
		Path: createTestPromptPack(t),
	}

	adapters := []adapter.AgentAdapter{
		// We'll add actual adapters here as we implement them
		cursor.NewAdapter(),
		claude.NewAdapter("test"),
	}

	for _, a := range adapters {
		t.Run(a.Name(), func(t *testing.T) {
			testAdapterContract(t, a, pack)
		})
	}
}

func testAdapterContract(t *testing.T, a adapter.AgentAdapter, pack adapter.PromptPack) {
	t.Run("Name returns non-empty string", func(t *testing.T) {
		name := a.Name()
		if name == "" {
			t.Fatal("Name() returned empty string")
		}
	})

	t.Run("Detect returns boolean", func(t *testing.T) {
		// Detect should not panic
		_ = a.Detect()
	})

	t.Run("TargetDir returns valid path", func(t *testing.T) {
		dir := a.TargetDir(adapter.ScopeProject)
		if dir == "" {
			t.Fatal("TargetDir() returned empty string")
		}
		// Should be a relative path
		if filepath.IsAbs(dir) {
			t.Fatalf("TargetDir() returned absolute path: %s", dir)
		}
	})

	t.Run("Render produces files", func(t *testing.T) {
		files, err := a.Render(pack, adapter.ScopeProject)
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}

		if len(files) == 0 {
			t.Fatal("Render() produced no files")
		}

		// Check each rendered file
		for _, f := range files {
			if f.Path == "" {
				t.Error("Rendered file has empty path")
			}
			if len(f.Content) == 0 {
				t.Errorf("Rendered file %s has no content", f.Path)
			}
			if f.Hash == "" {
				t.Errorf("Rendered file %s has no hash", f.Path)
			}
		}
	})

	t.Run("Verify checks file integrity", func(t *testing.T) {
		files, err := a.Render(pack, adapter.ScopeProject)
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}

		// Verify should pass with correct files
		err = a.Verify(files, adapter.StrictnessNormal)
		if err != nil {
			t.Fatalf("Verify() failed on valid files: %v", err)
		}

		// Modify a file's content
		if len(files) > 0 {
			files[0].Content = []byte("modified content")
			err = a.Verify(files, adapter.StrictnessNormal)
			if err == nil {
				t.Fatal("Verify() should fail on modified content")
			}
		}
	})
}

// createTestPromptPack creates a temporary directory with sample prompt files
func createTestPromptPack(t *testing.T) string {
	t.Helper()

	packDir := t.TempDir()
	promptsDir := filepath.Join(packDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	// Create a simple prompt file with front-matter
	promptFile := filepath.Join(promptsDir, "test-rule.md")
	content := `---
title: Test Rule
description: A test prompt for adapter testing
alwaysApply: true
globs: ["**/*.go"]
---

This is a test prompt that helps with Go development.
`
	if err := os.WriteFile(promptFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	// Create a metadata.yaml file
	metadataFile := filepath.Join(promptsDir, "metadata.yaml")
	metadataContent := `defaults:
  alwaysApply: false
  globs: ["**/*"]

files:
  test-rule.md:
    priority: high
`
	if err := os.WriteFile(metadataFile, []byte(metadataContent), 0644); err != nil {
		t.Fatalf("failed to write metadata file: %v", err)
	}

	return packDir
}
