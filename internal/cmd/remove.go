package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/kovyrin/prompt-sync/internal/config"
	"github.com/kovyrin/prompt-sync/internal/lock"
)

// NewRemoveCommand creates a new remove command
func NewRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <source>",
		Short: "Remove a prompt source from Promptsfile",
		Long: `Remove a prompt source from your Promptsfile and clean up rendered files.

The source can be specified with or without version specification:
  - github.com/org/prompts (removes any version)
  - github.com/org/prompts#v1.0.0 (removes specific version)

This command will:
  - Remove the source from Promptsfile
  - Delete all rendered files from the removed source
  - Update the lock file`,
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"rm"},
		RunE:    runRemove,
	}

	return cmd
}

// RemoveCmd is the exported remove command for backward compatibility
var RemoveCmd = NewRemoveCommand()

func init() {
	RootCmd.AddCommand(RemoveCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	source := args[0]

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Locate Promptsfile
	promptsfilePath, err := config.FindPromptsfilePath(workDir)
	if err != nil {
		return err
	}
	promptsDir := filepath.Dir(promptsfilePath)

	// Load current configuration
	loader := config.NewLoader(promptsDir)
	cfg, err := loader.Load()
	if err != nil {
		return fmt.Errorf("loading Promptsfile: %w", err)
	}

	// Find and remove the source
	removed, updatedSources, err := removeSource(cfg.Sources, source)
	if err != nil {
		return err
	}
	if !removed {
		// Check if it's in overlays
		for _, overlay := range cfg.Overlays {
			if matchesSource(overlay.Source, source) {
				return fmt.Errorf("cannot remove '%s': it's part of an overlay (scope: %s)", overlay.Source, overlay.Scope)
			}
		}
		return fmt.Errorf("source '%s' not found in Promptsfile", source)
	}

	// Update configuration
	cfg.Sources = updatedSources

	// Load lock file to get file paths for cleanup
	lockWriter := lock.New(promptsDir)
	lockData, err := lockWriter.Read()
	if err == nil && lockData != nil {
		// Clean up rendered files
		if err := cleanupRenderedFiles(workDir, source, lockData); err != nil {
			// Log warning but don't fail
			fmt.Fprintf(cmd.OutOrStderr(), "Warning: failed to clean up some files: %v\n", err)
		}

		// Update lock file
		if err := updateLockFile(lockWriter, lockData, source); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Warning: failed to update lock file: %v\n", err)
		}
	}

	// Write updated configuration
	if err := writePromptsfile(promptsfilePath, cfg); err != nil {
		return fmt.Errorf("writing Promptsfile: %w", err)
	}

	fmt.Printf("✓ Removed source: %s\n", source)

	// Check if gitignore needs updating
	if len(cfg.Sources) == 0 && len(cfg.Overlays) == 0 {
		fmt.Println("Note: All sources removed. You may want to clean up .gitignore")
	}

	return nil
}

func removeSource(sources []string, target string) (bool, []string, error) {
	// Extract base URL from target (remove version spec)
	targetBase := strings.Split(target, "#")[0]

	var updatedSources []string
	found := false

	for _, source := range sources {
		sourceBase := strings.Split(source, "#")[0]
		if sourceBase == targetBase {
			found = true
			// Skip this source (remove it)
		} else {
			updatedSources = append(updatedSources, source)
		}
	}

	return found, updatedSources, nil
}

func matchesSource(source, target string) bool {
	sourceBase := strings.Split(source, "#")[0]
	targetBase := strings.Split(target, "#")[0]
	return sourceBase == targetBase
}

func cleanupRenderedFiles(workDir, source string, lockData *lock.Lock) error {
	// Extract base URL
	sourceBase := strings.Split(source, "#")[0]

	// Find the source in lock data
	for _, lockedSource := range lockData.Sources {
		if strings.Split(lockedSource.URL, "#")[0] == sourceBase {
			// Delete all files for this source
			for _, file := range lockedSource.Files {
				filePath := filepath.Join(workDir, file.Path)
				if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
					// Continue trying to delete other files
					fmt.Printf("Warning: could not delete %s: %v\n", filePath, err)
				} else if err == nil && file.SourcePath != "" {
					// Log successful removal with source mapping
					fmt.Printf("  Removed %s (from %s)\n", file.Path, file.SourcePath)
				}
			}
			// Try to clean up empty directories
			cleanupEmptyDirs(workDir, lockedSource.Files)
			break
		}
	}

	return nil
}

// cleanupEmptyDirs attempts to remove empty directories after file cleanup
func cleanupEmptyDirs(workDir string, files []lock.File) {
	dirs := make(map[string]bool)

	// Collect all directories that contained files
	for _, file := range files {
		dir := filepath.Dir(filepath.Join(workDir, file.Path))
		for dir != workDir && dir != "." {
			dirs[dir] = true
			dir = filepath.Dir(dir)
		}
	}

	// Try to remove directories (will fail if not empty)
	for dir := range dirs {
		_ = os.Remove(dir) // Ignore errors - directory might not be empty
	}
}

func updateLockFile(lockWriter *lock.Writer, lockData *lock.Lock, source string) error {
	// Extract base URL
	sourceBase := strings.Split(source, "#")[0]

	// Filter out the removed source
	var updatedSources []lock.Source
	for _, lockedSource := range lockData.Sources {
		if strings.Split(lockedSource.URL, "#")[0] != sourceBase {
			updatedSources = append(updatedSources, lockedSource)
		}
	}

	// Update lock data
	lockData.Sources = updatedSources

	// Write updated lock file
	return lockWriter.Write(lockData.Sources)
}

func writePromptsfile(path string, cfg *config.ExtendedConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
