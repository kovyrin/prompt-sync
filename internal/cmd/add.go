package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/kovyrin/prompt-sync/internal/config"
	"github.com/kovyrin/prompt-sync/internal/security"
	"github.com/kovyrin/prompt-sync/internal/workflow"
)

var (
	addNoInstall    bool
	addAllowUnknown bool
)

// NewAddCommand creates a new add command
func NewAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <source>",
		Short: "Add a prompt source to Promptsfile",
		Long: `Add a new prompt source to your Promptsfile and optionally install it.

The source should be a Git repository URL, optionally with a ref (branch, tag, or commit):
  - github.com/org/prompts
  - github.com/org/prompts#v1.0.0
  - github.com/org/prompts#main

By default, the command will check if the source is trusted and then run installation.
Use --no-install to skip the installation step.`,
		Args: cobra.ExactArgs(1),
		RunE: runAdd,
	}

	cmd.Flags().BoolVar(&addNoInstall, "no-install", false, "Don't run install after adding the source")
	cmd.Flags().BoolVar(&addAllowUnknown, "allow-unknown", false, "Allow untrusted sources")

	return cmd
}

// AddCmd is the exported add command for backward compatibility
var AddCmd = NewAddCommand()

func init() {
	RootCmd.AddCommand(AddCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	source := args[0]

	// Validate source format
	if err := validateSourceURL(source); err != nil {
		return fmt.Errorf("invalid source URL: %w", err)
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Check if Promptsfile exists
	promptsfilePath := "Promptsfile"
	if _, err := os.Stat(promptsfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Promptsfile not found. Run 'prompt-sync init' first")
	}

	// Load current configuration
	loader := config.NewLoader(workDir)
	cfg, err := loader.Load()
	if err != nil {
		return fmt.Errorf("loading Promptsfile: %w", err)
	}

	// Extract base URL (without ref) for trusted source checking
	baseURL := strings.Split(source, "#")[0]

	// Check if source is trusted (unless --allow-unknown is set)
	if !addAllowUnknown {
		trustedSources := security.NewTrustedSources()
		if !trustedSources.IsTrusted(baseURL) {
			return fmt.Errorf("untrusted source: %s. Use --allow-unknown to bypass this check", baseURL)
		}
	}

	// Check for duplicates
	if err := checkDuplicate(cfg, source); err != nil {
		return err
	}

	// Add the source
	cfg.Sources = append(cfg.Sources, source)

	// Write updated configuration
	if err := writeConfig(promptsfilePath, cfg); err != nil {
		return fmt.Errorf("writing Promptsfile: %w", err)
	}

	fmt.Printf("✓ Added source: %s\n", source)

	// Run installation unless --no-install is set
	if !addNoInstall {
		fmt.Println("Running installation...")

		installer, err := workflow.New(workflow.InstallOptions{
			WorkspaceDir: workDir,
			AllowUnknown: addAllowUnknown,
		})
		if err != nil {
			return fmt.Errorf("creating installer: %w", err)
		}

		if err := installer.Execute(); err != nil {
			return fmt.Errorf("installation failed: %w", err)
		}

		fmt.Println("✓ Installation complete")
	}

	return nil
}

func validateSourceURL(source string) error {
	if source == "" {
		return fmt.Errorf("source URL cannot be empty")
	}

	// Extract URL part (before #)
	parts := strings.Split(source, "#")
	url := parts[0]

	// Basic validation
	if !strings.Contains(url, "/") {
		return fmt.Errorf("invalid repository format")
	}

	// Check for common issues
	if strings.HasSuffix(url, "/") {
		return fmt.Errorf("URL should not end with /")
	}

	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return fmt.Errorf("please use repository path format (e.g., github.com/org/repo) instead of full URL")
	}

	return nil
}

func checkDuplicate(cfg *config.ExtendedConfig, source string) error {
	// Check in regular sources
	for _, existing := range cfg.Sources {
		if existing == source {
			return fmt.Errorf("source '%s' already exists in Promptsfile", source)
		}
		// Also check if it's the same URL with different ref
		existingBase := strings.Split(existing, "#")[0]
		sourceBase := strings.Split(source, "#")[0]
		if existingBase == sourceBase {
			return fmt.Errorf("source '%s' already exists (as '%s')", sourceBase, existing)
		}
	}

	// Check in overlays
	for _, overlay := range cfg.Overlays {
		if overlay.Source == source {
			return fmt.Errorf("source '%s' already exists as %s overlay", source, overlay.Scope)
		}
	}

	return nil
}

func writeConfig(path string, cfg *config.ExtendedConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
