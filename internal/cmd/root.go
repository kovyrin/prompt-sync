package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// RootCmd is the main entry point for all prompt-sync subcommands.
var RootCmd = &cobra.Command{
	Use:   "prompt-sync",
	Short: "Prompt-Sync CLI – AI prompt package manager",
}

// Execute executes the root command and exits on failure.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
