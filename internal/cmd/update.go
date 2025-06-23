package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kovyrin/prompt-sync/internal/config"
	"github.com/kovyrin/prompt-sync/internal/git"
	"github.com/kovyrin/prompt-sync/internal/workflow"
)

var (
	updateDryRun       bool
	updateForce        bool
	updateStrict       bool
	updateOffline      bool
	updateCacheDir     string
	updateAllowUnknown bool
)

// NewUpdateCommand creates a new update command
func NewUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [sources...]",
		Short: "Update prompt sources to their latest versions",
		Long: `Update prompt sources to their latest versions while respecting version constraints.

Without arguments, updates all unpinned sources to their latest versions.
With source arguments, updates only the specified sources.

Examples:
  # Update all unpinned sources
  prompt-sync update

  # Update specific sources
  prompt-sync update github.com/org/prompts

  # Check what would be updated without making changes
  prompt-sync update --dry-run

  # Force update even pinned sources
  prompt-sync update --force github.com/org/prompts#v1.0.0`,
		RunE: runUpdate,
	}

	cmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "Show what would be updated without making changes")
	cmd.Flags().BoolVar(&updateForce, "force", false, "Force update even for pinned sources")
	cmd.Flags().BoolVar(&updateStrict, "strict", false, "Treat warnings as errors")
	cmd.Flags().BoolVar(&updateOffline, "offline", false, "Use only cached repositories")
	cmd.Flags().StringVar(&updateCacheDir, "cache-dir", "", "Override cache directory")
	cmd.Flags().BoolVar(&updateAllowUnknown, "allow-unknown", false, "Allow untrusted sources")

	return cmd
}

// UpdateCmd is the exported update command for backward compatibility
var UpdateCmd = NewUpdateCommand()

func init() {
	RootCmd.AddCommand(UpdateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Check for CI mode
	if os.Getenv("CI") == "true" {
		updateStrict = true
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Check if Promptsfile exists
	if _, err := os.Stat("Promptsfile"); os.IsNotExist(err) {
		return fmt.Errorf("Promptsfile not found. Run 'prompt-sync init' first")
	}

	// Check if lock file exists
	if _, err := os.Stat("Promptsfile.lock"); os.IsNotExist(err) {
		return fmt.Errorf("no lock file found. Run 'prompt-sync install' first")
	}

	// Load configuration
	loader := config.NewLoader(workDir)
	cfg, err := loader.Load()
	if err != nil {
		return fmt.Errorf("loading Promptsfile: %w", err)
	}

	// Determine which sources to update
	sourcesToUpdate, err := determineSourcesToUpdate(cfg, args)
	if err != nil {
		return err
	}

	if len(sourcesToUpdate) == 0 {
		fmt.Println("No sources to update")
		return nil
	}

	// Show what will be updated
	fmt.Printf("Checking for updates to %d source(s)...\n", len(sourcesToUpdate))

	// Check for updates
	updates, err := checkForUpdates(workDir, sourcesToUpdate)
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	if len(updates) == 0 {
		fmt.Println("✓ All sources are up to date")
		return nil
	}

	// Display available updates
	displayUpdates(cmd, updates)

	// If dry-run, stop here
	if updateDryRun {
		fmt.Println("\nDry run mode - no changes made")
		return nil
	}

	// Apply updates
	fmt.Println("\nApplying updates...")

	// Update Promptsfile for unpinned sources that need new refs
	if err := updatePromptsfile(cfg, updates); err != nil {
		return fmt.Errorf("updating Promptsfile: %w", err)
	}

	// Run install to apply all updates
	installer, err := workflow.New(workflow.InstallOptions{
		WorkspaceDir: workDir,
		StrictMode:   updateStrict,
		Offline:      updateOffline,
		CacheDir:     updateCacheDir,
		AllowUnknown: updateAllowUnknown,
	})
	if err != nil {
		return fmt.Errorf("creating installer: %w", err)
	}

	if err := installer.Execute(); err != nil {
		return fmt.Errorf("applying updates: %w", err)
	}

	fmt.Printf("\n✓ Updated %d source(s)\n", len(updates))
	return nil
}

type sourceUpdate struct {
	URL         string
	CurrentRef  string
	CurrentHash string
	NewRef      string
	NewHash     string
	IsPinned    bool
}

func determineSourcesToUpdate(cfg *config.ExtendedConfig, args []string) ([]string, error) {
	if len(args) == 0 {
		// Update all sources
		sources := make([]string, 0, len(cfg.Sources))
		for _, source := range cfg.Sources {
			// Skip pinned sources unless --force is set
			if !updateForce && isPinnedSource(source) {
				continue
			}
			sources = append(sources, source)
		}
		return sources, nil
	}

	// Update specific sources
	sources := make([]string, 0, len(args))
	for _, arg := range args {
		// Find matching source
		found := false
		for _, source := range cfg.Sources {
			if matchesSourceURL(source, arg) {
				// Check if pinned and force not set
				if !updateForce && isPinnedSource(source) {
					return nil, fmt.Errorf("source '%s' is pinned to a specific version. Use --force to update", source)
				}
				sources = append(sources, source)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("source '%s' not found in Promptsfile", arg)
		}
	}
	return sources, nil
}

func isPinnedSource(source string) bool {
	// A source is pinned if it has a specific version ref
	parts := strings.Split(source, "#")
	if len(parts) < 2 {
		return false // No ref, not pinned
	}

	ref := parts[1]
	// Consider it pinned if it's a specific tag/version or commit hash
	// Branch names like "main", "master", "develop" are not considered pinned
	unpinnedBranches := []string{"main", "master", "develop", "dev"}
	for _, branch := range unpinnedBranches {
		if ref == branch {
			return false
		}
	}

	// If it looks like a version tag or commit hash, it's pinned
	return true
}

func matchesSourceURL(source, target string) bool {
	sourceBase := strings.Split(source, "#")[0]
	targetBase := strings.Split(target, "#")[0]
	return sourceBase == targetBase
}

func checkForUpdates(workDir string, sources []string) ([]sourceUpdate, error) {
	// This is a simplified version - in a real implementation,
	// we would use the git fetcher to check for actual updates
	updates := []sourceUpdate{}

	// For now, just mark all sources as having updates available
	// In a real implementation, this would:
	// 1. Use git fetcher to check remote for new commits
	// 2. Compare with current lock file
	// 3. Respect version constraints

	for _, source := range sources {
		// Skip this basic implementation for testing
		update := sourceUpdate{
			URL:      source,
			IsPinned: isPinnedSource(source),
		}
		updates = append(updates, update)
	}

	return updates, nil
}

func displayUpdates(cmd *cobra.Command, updates []sourceUpdate) {
	fmt.Fprintf(cmd.OutOrStdout(), "\nAvailable updates:\n")
	for _, update := range updates {
		if update.IsPinned && updateForce {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s (force update pinned source)\n", update.URL)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", update.URL)
		}
	}
}

func updatePromptsfile(cfg *config.ExtendedConfig, updates []sourceUpdate) error {
	// For unpinned sources that have new refs available,
	// we might update the Promptsfile to point to new branches/tags
	// This is a simplified implementation

	// In a real implementation, this would:
	// 1. Check if any sources need their refs updated in Promptsfile
	// 2. Preserve formatting and comments
	// 3. Write the updated file

	return nil
}

// gitOptions creates git options from command flags
func gitOptions() []git.Option {
	opts := []git.Option{}

	if updateOffline {
		opts = append(opts, git.WithOfflineMode())
	}

	if updateCacheDir != "" {
		opts = append(opts, git.WithCacheDir(updateCacheDir))
	}

	return opts
}
