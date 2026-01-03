package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version information set at build time
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Display the version, git commit, and build date of the Carbon CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Carbon CLI\n")
		fmt.Printf("  Version:    %s\n", Version)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
		fmt.Printf("  Build Date: %s\n", BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
