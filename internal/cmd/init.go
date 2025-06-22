package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const (
	promptsfileName = "Promptsfile"
	gitignoreName   = ".gitignore"

	managedBegin = "# BEGIN prompt-sync managed\n"
	managedEnd   = "# END prompt-sync managed\n"
)

var force bool

func init() {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Prompt-Sync in the current project",
		RunE:  runInit,
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing Promptsfile if present")
	RootCmd.AddCommand(cmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	// Create Promptsfile
	if err := createPromptsfile(); err != nil {
		return err
	}
	// Ensure .gitignore managed block exists
	if err := ensureGitignoreBlock(); err != nil {
		return err
	}
	return nil
}

func createPromptsfile() error {
	if _, err := os.Stat(promptsfileName); err == nil && !force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", promptsfileName)
	}
	template := `# Promptsfile – managed by prompt-sync

# Define your prompt sources here. Example:
# sources:
#   - name: myteam/common-prompts
#     url:  git@github.com:myteam/common-prompts.git
#     ref:  main
#`
	return os.WriteFile(promptsfileName, []byte(template), 0o644)
}

func ensureGitignoreBlock() error {
	contents := managedBegin + ".cursor/rules/\n" + managedEnd

	var existing []byte
	if b, err := os.ReadFile(gitignoreName); err == nil {
		existing = b
		if bytes.Contains(existing, []byte(managedBegin)) {
			// Managed block already present – nothing to do.
			return nil
		}
	}

	updated := append(existing, []byte(contents)...)
	// Ensure directory exists for .gitignore path (usually project root)
	if err := os.WriteFile(filepath.Clean(gitignoreName), updated, 0o644); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}
	return nil
}
