package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kovyrin/prompt-sync/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallWorkflow(t *testing.T) {
	t.Run("installs prompt packs from Promptsfile", func(t *testing.T) {
		// Create a test workspace
		workspace := t.TempDir()

		// Get absolute paths to test repos
		acmeRepo := filepath.Join(getProjectRoot(t), "testdata/repos/acme-prompts")
		devRepo := filepath.Join(getProjectRoot(t), "testdata/repos/dev-prompts")

		// Create a Promptsfile with test prompt packs
		promptsfile := fmt.Sprintf(`# Test Promptsfile
sources:
  - file://%s#v1.0.0
  - file://%s#master

adapters:
  cursor:
    enabled: true
  claude:
    enabled: true
    prefix: "acme"
`, acmeRepo, devRepo)
		err := os.WriteFile(filepath.Join(workspace, "Promptsfile"), []byte(promptsfile), 0644)
		require.NoError(t, err)

		// Run the install workflow
		installer, err := workflow.New(workflow.InstallOptions{
			WorkspaceDir: workspace,
			StrictMode:   false,
			AllowUnknown: true, // Allow test URLs
		})
		require.NoError(t, err)

		err = installer.Execute()
		require.NoError(t, err)

		// Verify lock file was created
		lockPath := filepath.Join(workspace, "Promptsfile.lock")
		assert.FileExists(t, lockPath)

		// Verify .gitignore was updated with managed block
		gitignorePath := filepath.Join(workspace, ".gitignore")
		assert.FileExists(t, gitignorePath)
		gitignoreContent, err := os.ReadFile(gitignorePath)
		require.NoError(t, err)
		assert.Contains(t, string(gitignoreContent), "# BEGIN PROMPT-SYNC MANAGED")
		assert.Contains(t, string(gitignoreContent), ".cursor/rules/_active/")
		assert.Contains(t, string(gitignoreContent), ".claude/commands/acme-*")

		// Verify rendered files exist
		cursorRulesDir := filepath.Join(workspace, ".cursor/rules/_active")
		assert.DirExists(t, cursorRulesDir)

		claudeCommandsDir := filepath.Join(workspace, ".claude/commands")
		assert.DirExists(t, claudeCommandsDir)
	})

	t.Run("applies overlay precedence correctly", func(t *testing.T) {
		workspace := t.TempDir()

		// Create Promptsfile with overlapping packs
		promptsfile := `sources:
  - https://github.com/org/org-prompts.git
  - https://github.com/project/project-prompts.git
  - https://github.com/personal/personal-prompts.git

overlays:
  - scope: org
    source: https://github.com/org/org-prompts.git
  - scope: project
    source: https://github.com/project/project-prompts.git
  - scope: personal
    source: https://github.com/personal/personal-prompts.git

adapters:
  cursor:
    enabled: true
`
		err := os.WriteFile(filepath.Join(workspace, "Promptsfile"), []byte(promptsfile), 0644)
		require.NoError(t, err)

		// TODO: Run install and verify personal > project > org precedence
	})

	t.Run("preserves frontmatter in MDC files", func(t *testing.T) {
		workspace := t.TempDir()

		// Create test repo with MDC file containing frontmatter
		testRepo := filepath.Join(t.TempDir(), "frontmatter-test")
		require.NoError(t, os.MkdirAll(filepath.Join(testRepo, "prompts"), 0755))

		// Create MDC file with frontmatter
		mdcContent := `---
title: Test Rule with Frontmatter
priority: high
tags:
  - testing
  - mdc
custom_settings:
  enabled: true
  level: strict
---

# Test Rule

This MDC file has frontmatter that should be preserved.

::alert{type="info"}
MDC components should also work
::`

		mdcPath := filepath.Join(testRepo, "prompts", "test-rule.mdc")
		require.NoError(t, os.WriteFile(mdcPath, []byte(mdcContent), 0644))

		// Initialize git repo
		cmd := exec.Command("git", "init")
		cmd.Dir = testRepo
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = testRepo
		cmd.Run()

		cmd = exec.Command("git", "config", "user.name", "Test User")
		cmd.Dir = testRepo
		cmd.Run()

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = testRepo
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "commit", "-m", "Initial commit")
		cmd.Dir = testRepo
		require.NoError(t, cmd.Run())

		// Create Promptsfile
		promptsfile := fmt.Sprintf(`sources:
  - file://%s#master

adapters:
  cursor:
    enabled: true
`, testRepo)

		require.NoError(t, os.WriteFile(filepath.Join(workspace, "Promptsfile"), []byte(promptsfile), 0644))

		// Run install
		installer, err := workflow.New(workflow.InstallOptions{
			WorkspaceDir: workspace,
			AllowUnknown: true,
		})
		require.NoError(t, err)

		err = installer.Execute()
		require.NoError(t, err)

		// Verify the rendered file preserves frontmatter
		renderedPath := filepath.Join(workspace, ".cursor/rules/_active/test-rule.mdc")
		assert.FileExists(t, renderedPath)

		renderedContent, err := os.ReadFile(renderedPath)
		require.NoError(t, err)

		// Should contain all the frontmatter
		assert.Contains(t, string(renderedContent), "title: Test Rule with Frontmatter")
		assert.Contains(t, string(renderedContent), "priority: high")
		assert.Contains(t, string(renderedContent), "tags:")
		assert.Contains(t, string(renderedContent), "- testing")
		assert.Contains(t, string(renderedContent), "- mdc")
		assert.Contains(t, string(renderedContent), "custom_settings:")
		assert.Contains(t, string(renderedContent), "enabled: true")
		assert.Contains(t, string(renderedContent), "level: strict")

		// Should also contain the body content
		assert.Contains(t, string(renderedContent), "# Test Rule")
		assert.Contains(t, string(renderedContent), "::alert{type=\"info\"}")
	})

	t.Run("detects and reports conflicts", func(t *testing.T) {
		workspace := t.TempDir()

		// Create Promptsfile with conflicting packs
		promptsfile := `sources:
  - https://github.com/pack1/prompts.git
  - https://github.com/pack2/prompts.git  # Has same filename as pack1

adapters:
  cursor:
    enabled: true
`
		err := os.WriteFile(filepath.Join(workspace, "Promptsfile"), []byte(promptsfile), 0644)
		require.NoError(t, err)

		// TODO: Run install and verify conflict detection
		// Should warn about duplicate basenames across adapters
	})

	t.Run("respects strict mode", func(t *testing.T) {
		workspace := t.TempDir()

		// Create Promptsfile with conflict-prone setup
		promptsfile := `sources:
  - https://github.com/untrusted/prompts.git  # Not in trusted sources

adapters:
  cursor:
    enabled: true
`
		err := os.WriteFile(filepath.Join(workspace, "Promptsfile"), []byte(promptsfile), 0644)
		require.NoError(t, err)

		// TODO: Run install with --strict flag
		// Should fail due to untrusted source
	})

	t.Run("verify mode detects drift", func(t *testing.T) {
		workspace := t.TempDir()

		// Create initial state
		promptsfile := `sources:
  - https://github.com/acme/prompts.git#v1.0.0

adapters:
  cursor:
    enabled: true
`
		err := os.WriteFile(filepath.Join(workspace, "Promptsfile"), []byte(promptsfile), 0644)
		require.NoError(t, err)

		// Create a lock file
		lockfile := `# Promptsfile.lock
# Generated by prompt-sync
sources:
  - url: https://github.com/acme/prompts.git
    ref: v1.0.0
    commit: abc123def456
    files:
      - path: rules/coding.md
        hash: sha256:oldhashabc123
`
		err = os.WriteFile(filepath.Join(workspace, "Promptsfile.lock"), []byte(lockfile), 0644)
		require.NoError(t, err)

		// TODO: Modify rendered file to create drift
		// Run verify and ensure it detects the drift
	})
}

// getProjectRoot returns the absolute path to the project root
func getProjectRoot(t *testing.T) string {
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
