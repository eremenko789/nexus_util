package main

import (
	"fmt"
	"os"

	"nexus-util/cmd/asset"
	"nexus-util/cmd/blob"
	initcmd "nexus-util/cmd/init"
	"nexus-util/cmd/repo"
	"nexus-util/cmd/sync"

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
	rootCmd.PersistentFlags().StringP("user", "u", "", "User authentication login (overrides config file)")
	rootCmd.PersistentFlags().StringP("password", "p", "", "User authentication password (overrides config file)")
	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to configuration file (default: ~/.nexus-util.yaml)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Quiet mode - minimal output")
	rootCmd.PersistentFlags().Bool("dry", false, "Dry run - show what would be done without actually doing it")
	rootCmd.PersistentFlags().Bool("insecure", false, "Skip TLS/SSL certificate verification")

	// Initialize commands
	setupCommands()

	// Add commands
	rootCmd.AddCommand(asset.AssetCmd)
	rootCmd.AddCommand(blob.BlobCmd)
	rootCmd.AddCommand(initcmd.InitCmd)
	rootCmd.AddCommand(repo.RepoCmd)
	rootCmd.AddCommand(sync.SyncCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func setupCommands() {
	// Asset command - add repository flag as persistent flag
	asset.AssetCmd.PersistentFlags().StringP("repository", "r", "", "Nexus OSS raw repository name (required)")
	if err := asset.AssetCmd.MarkPersistentFlagRequired("repository"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking repository flag as required: %v\n", err)
	}

	// Add subcommands to asset command
	asset.AssetCmd.AddCommand(asset.PushCmd)
	asset.AssetCmd.AddCommand(asset.PullCmd)
	asset.AssetCmd.AddCommand(asset.DeleteCmd)
	asset.AssetCmd.AddCommand(asset.ListCmd)
	asset.AssetCmd.AddCommand(asset.DiffCmd)

	// Push command flags
	asset.PushCmd.Flags().StringP("destination", "d", "", "Destination path in Nexus repository")
	asset.PushCmd.Flags().Bool("relative", false, "Use relative paths when uploading directories")

	// Pull command flags
	asset.PullCmd.Flags().StringP("destination", "d", "", "Local destination path (required)")
	asset.PullCmd.Flags().String("root", "", "Root path in Nexus repository")
	asset.PullCmd.Flags().BoolP("saveStructure", "s", false, "Save directory structure in destination path")

	// Diff command flags
	asset.DiffCmd.Flags().String("target-address", "", "Target Nexus OSS host address (default: source address)")
	asset.DiffCmd.Flags().String("target-repo", "", "Target Nexus repository name")
	asset.DiffCmd.Flags().String("target-user", "", "Target user authentication login")
	asset.DiffCmd.Flags().String("target-pass", "", "Target user authentication password")
	asset.DiffCmd.Flags().String("local", "", "Local directory to compare against source repository")
	asset.DiffCmd.Flags().String("path", "", "Repository path to compare (applies to both sources)")

	// Init command flags
	initcmd.InitCmd.Flags().StringP("address", "a", "", "Nexus OSS host address (required)")
	initcmd.InitCmd.Flags().StringP("user", "u", "", "User authentication login (required)")
	initcmd.InitCmd.Flags().StringP("password", "p", "", "User authentication password")
	initcmd.InitCmd.Flags().StringP("config", "c", "", "Path to configuration file (default: ~/.nexus-util.yaml)")
	if err := initcmd.InitCmd.MarkFlagRequired("address"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking address flag as required: %v\n", err)
	}
	if err := initcmd.InitCmd.MarkFlagRequired("user"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking user flag as required: %v\n", err)
	}

	// Sync command flags
	sync.SyncCmd.Flags().String("source-address", "", "Source Nexus OSS host address")
	sync.SyncCmd.Flags().String("source-repo", "", "Source Nexus repository name (required)")
	sync.SyncCmd.Flags().String("source-user", "", "Source user authentication login")
	sync.SyncCmd.Flags().String("source-pass", "", "Source user authentication password")
	sync.SyncCmd.Flags().String("target-address", "", "Target Nexus OSS host address")
	sync.SyncCmd.Flags().String("target-repo", "", "Target Nexus repository name (required)")
	sync.SyncCmd.Flags().String("target-user", "", "Target user authentication login")
	sync.SyncCmd.Flags().String("target-pass", "", "Target user authentication password")
	sync.SyncCmd.Flags().Bool("skip-existing", true, "Skip files that already exist in target repository")
	sync.SyncCmd.Flags().Bool("show-progress", true, "Show detailed progress for each file")

	if err := sync.SyncCmd.MarkFlagRequired("source-repo"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking source-repo flag as required: %v\n", err)
	}
	if err := sync.SyncCmd.MarkFlagRequired("target-repo"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking target-repo flag as required: %v\n", err)
	}
}
