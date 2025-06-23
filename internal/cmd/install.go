package cmd

import (
	"fmt"
	"os"

	"github.com/kovyrin/prompt-sync/internal/workflow"
	"github.com/spf13/cobra"
)

var (
	installStrict       bool
	installOffline      bool
	installCacheDir     string
	installAllowUnknown bool
	installYes          bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install prompt packs from Promptsfile",
	Long: `Install fetches all prompt packs specified in your Promptsfile,
renders them using the configured adapters, and creates a lock file.`,
	RunE: runInstall,
}

var ciInstallCmd = &cobra.Command{
	Use:   "ci-install",
	Short: "Install prompt packs in CI mode (alias for install --yes --strict)",
	Long: `CI-friendly installation that runs non-interactively with strict error checking.
Equivalent to: prompt-sync install --yes --strict`,
	RunE: runCIInstall,
}

func init() {
	RootCmd.AddCommand(installCmd)
	RootCmd.AddCommand(ciInstallCmd)

	installCmd.Flags().BoolVar(&installStrict, "strict", false, "Treat warnings as errors")
	installCmd.Flags().BoolVar(&installOffline, "offline", false, "Use only cached repositories")
	installCmd.Flags().StringVar(&installCacheDir, "cache-dir", "", "Override cache directory")
	installCmd.Flags().BoolVar(&installAllowUnknown, "allow-unknown", false, "Allow untrusted sources")
	installCmd.Flags().BoolVarP(&installYes, "yes", "y", false, "Assume yes to all prompts")
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Check for CI mode
	if os.Getenv("CI") == "true" || installYes {
		installStrict = true
	}

	// Get workspace directory
	workspaceDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if Promptsfile exists
	if _, err := os.Stat("Promptsfile"); os.IsNotExist(err) {
		return fmt.Errorf("Promptsfile not found. Run 'prompt-sync init' first")
	}

	// Create installer
	installer, err := workflow.New(workflow.InstallOptions{
		WorkspaceDir: workspaceDir,
		StrictMode:   installStrict,
		VerifyOnly:   false,
		Offline:      installOffline,
		CacheDir:     installCacheDir,
		AllowUnknown: installAllowUnknown,
	})
	if err != nil {
		return fmt.Errorf("failed to create installer: %w", err)
	}

	// Run installation
	if err := installer.Execute(); err != nil {
		return err
	}

	fmt.Println("✓ Installation complete")
	return nil
}

func runCIInstall(cmd *cobra.Command, args []string) error {
	// Set CI mode flags
	installStrict = true
	installYes = true

	// Run regular install
	return runInstall(cmd, args)
}
