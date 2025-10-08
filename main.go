package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"
	build   = "dev"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "nexus-util",
		Short: "Nexus OSS Raw Repository utility tool",
		Long: `A unified tool for managing files and directories in Nexus OSS Raw Repository.
This tool combines the functionality of nexus_push, nexus_pull, and nexus_delete scripts.`,
		Version: fmt.Sprintf("%s (build: %s)", version, build),
	}

	// Add global flags
	rootCmd.PersistentFlags().StringP("address", "a", "", "Nexus OSS host address (required)")
	rootCmd.PersistentFlags().StringP("repository", "r", "", "Nexus OSS raw repository name (required)")
	rootCmd.PersistentFlags().StringP("user", "u", "", "User authentication login")
	rootCmd.PersistentFlags().StringP("password", "p", "", "User authentication password")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Quiet mode - minimal output")
	rootCmd.PersistentFlags().Bool("dry", false, "Dry run - show what would be done without actually doing it")

	// Add commands
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(deleteCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}