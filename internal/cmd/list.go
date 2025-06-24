package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kovyrin/prompt-sync/internal/config"
	"github.com/kovyrin/prompt-sync/internal/lock"
)

var (
	showFiles    bool
	showOutdated bool
	outputJSON   bool
)

// ListCmd represents the list command
var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed prompt packages",
	Long: `List all configured and installed prompt packages.

This command shows information about prompt packages defined in your Promptsfile,
including their installation status, versions, and optionally their rendered files.`,
	RunE: runList,
}

func init() {
	ListCmd.Flags().BoolVar(&showFiles, "files", false, "Show rendered file paths")
	ListCmd.Flags().BoolVar(&showOutdated, "outdated", false, "Show only outdated packages")
	ListCmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")

	RootCmd.AddCommand(ListCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Locate Promptsfile (supports root, .ai, or $PROMPT_SYNC_DIR)
	promptsPath, err := config.FindPromptsfilePath(workDir)
	if err != nil {
		return err
	}
	promptsDir := filepath.Dir(promptsPath)

	// Load Promptsfile
	loader := config.NewLoader(promptsDir)
	cfg, err := loader.Load()
	if err != nil {
		return fmt.Errorf("loading Promptsfile: %w", err)
	}

	// Load lock file if it exists
	lockWriter := lock.New(promptsDir)
	lockData, err := lockWriter.Read()
	if err != nil {
		return fmt.Errorf("loading lock file: %w", err)
	}

	// Build source information
	sources := buildPromptInfo(cfg, lockData, workDir)

	// Filter if --outdated is specified (not implemented yet)
	if showOutdated {
		sources = filterOutdated(sources)
	}

	// Output results
	if outputJSON {
		return outputJSONFormat(cmd, sources)
	}
	return outputTableFormat(cmd, sources)
}

type sourceInfo struct {
	URL           string   `json:"url"`
	Commit        string   `json:"commit,omitempty"`
	Ref           string   `json:"ref,omitempty"`
	RenderedFiles []string `json:"rendered_files,omitempty"`
	Installed     bool     `json:"installed"`
}

type listJSONOutput struct {
	Sources []sourceInfo `json:"sources"`
}

func buildPromptInfo(cfg *config.ExtendedConfig, lockData *lock.Lock, workDir string) []sourceInfo {
	var sources []sourceInfo

	// Create a map of lock entries for quick lookup
	lockMap := make(map[string]*lock.Source)
	if lockData != nil {
		for i, entry := range lockData.Sources {
			lockMap[entry.URL] = &lockData.Sources[i]
		}
	}

	// Process sources
	allSources := append([]string{}, cfg.Sources...)
	for _, overlay := range cfg.Overlays {
		allSources = append(allSources, overlay.Source)
	}

	for _, source := range allSources {
		// Extract URL and ref
		parts := strings.Split(source, "#")
		url := parts[0]
		ref := ""
		if len(parts) > 1 {
			ref = parts[1]
		}

		info := sourceInfo{
			URL:       url,
			Ref:       ref,
			Installed: false,
		}

		// Check if installed (exists in lock file)
		if lockEntry, exists := lockMap[url]; exists {
			info.Installed = true
			info.Commit = lockEntry.Commit

			// If showing files, get rendered file paths
			if showFiles {
				for _, file := range lockEntry.Files {
					info.RenderedFiles = append(info.RenderedFiles, file.Path)
				}
			}
		}

		sources = append(sources, info)
	}

	return sources
}

func filterOutdated(sources []sourceInfo) []sourceInfo {
	// TODO: Implement checking for updates
	// For now, return empty list
	return []sourceInfo{}
}

func outputTableFormat(cmd *cobra.Command, sources []sourceInfo) error {
	out := cmd.OutOrStdout()

	if len(sources) == 0 {
		fmt.Fprintln(out, "No sources configured")
		return nil
	}

	if showOutdated && len(sources) == 0 {
		fmt.Fprintln(out, "All sources are up to date")
		return nil
	}

	// Print header
	if showFiles {
		fmt.Fprintf(out, "%-50s %-10s %-12s %s\n", "SOURCE", "REF", "COMMIT", "FILES")
		fmt.Fprintln(out, strings.Repeat("-", 100))
	} else {
		fmt.Fprintf(out, "%-50s %-10s %-12s %-10s\n", "SOURCE", "REF", "COMMIT", "STATUS")
		fmt.Fprintln(out, strings.Repeat("-", 85))
	}

	// Print sources
	for _, s := range sources {
		status := "(not installed)"
		commit := ""
		if s.Installed {
			status = "installed"
			if len(s.Commit) > 7 {
				commit = s.Commit[:7]
			} else {
				commit = s.Commit
			}
		}

		ref := s.Ref
		if ref == "" {
			ref = "-"
		}

		if showFiles && len(s.RenderedFiles) > 0 {
			fmt.Fprintf(out, "%-50s %-10s %-12s\n", s.URL, ref, commit)
			for _, file := range s.RenderedFiles {
				fmt.Fprintf(out, "%-73s %s\n", "", file)
			}
		} else {
			fmt.Fprintf(out, "%-50s %-10s %-12s %-10s\n", s.URL, ref, commit, status)
		}
	}

	return nil
}

func outputJSONFormat(cmd *cobra.Command, sources []sourceInfo) error {
	output := listJSONOutput{Sources: sources}
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
