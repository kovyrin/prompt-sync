package cmd

import (
	"fmt"
	"os"

	"github.com/kovyrin/prompt-sync/internal/workflow"
	"github.com/spf13/cobra"
)

var (
	verifyAllowUnknown bool
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify installed prompts match lock file",
	Long: `Verify checks that all rendered prompt files match the hashes
recorded in Promptsfile.lock, detecting any drift or tampering.`,
	RunE: runVerify,
}

func init() {
	RootCmd.AddCommand(verifyCmd)
	verifyCmd.Flags().BoolVar(&verifyAllowUnknown, "allow-unknown", false, "Allow untrusted sources")
}

func runVerify(cmd *cobra.Command, args []string) error {
	// Get workspace directory
	workspaceDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create installer in verify mode
	installer, err := workflow.New(workflow.InstallOptions{
		WorkspaceDir: workspaceDir,
		StrictMode:   true, // Always strict in verify mode
		VerifyOnly:   true,
		Offline:      true, // Don't fetch in verify mode
		AllowUnknown: verifyAllowUnknown,
	})
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	// Run verification
	if err := installer.Execute(); err != nil {
		return err
	}

	fmt.Println("✓ All files verified successfully")
	return nil
}
