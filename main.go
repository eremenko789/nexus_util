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
This tool combines the functionality of nexus_push, nexus_pull, and nexus_delete scripts.

Configuration:
  The tool supports configuration via a YAML file. By default, it looks for
  ~/.nexus-util.yaml, but you can specify a custom path with --config.

  Example configuration file:
    nexus:
      address: http://nexus.example.com
    repository: myrepo
    user: myuser
    password: mypassword

  Command line flags override configuration file values.`,
		Version: fmt.Sprintf("%s (build: %s)", version, build),
	}

	// Add global flags
	rootCmd.PersistentFlags().StringP("address", "a", "", "Nexus OSS host address (overrides config file)")
	rootCmd.PersistentFlags().StringP("repository", "r", "", "Nexus OSS raw repository name (overrides config file)")
	rootCmd.PersistentFlags().StringP("user", "u", "", "User authentication login (overrides config file)")
	rootCmd.PersistentFlags().StringP("password", "p", "", "User authentication password (overrides config file)")
	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to configuration file (default: ~/.nexus-util.yaml)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Quiet mode - minimal output")
	rootCmd.PersistentFlags().Bool("dry", false, "Dry run - show what would be done without actually doing it")

	// Add commands
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(initCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}