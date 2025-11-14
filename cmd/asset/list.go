package asset

import (
	"fmt"

	"nexus-util/config"
	"nexus-util/nexus"

	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list [subdir]",
	Short: "List files in a directory in Nexus repository",
	Long: `List all files in a directory (or root) in Nexus OSS Raw Repository.
If subdir is not provided, lists files in the root of the repository.

Examples:
  # List all files in repository root
  nexus-util asset list -a http://nexus.example.com -r myrepo -u user -p pass

  # List files in a specific subdirectory
  nexus-util asset list -a http://nexus.example.com -r myrepo -u user -p pass subdir/

  # List files with quiet mode (only file paths)
  nexus-util asset list -q -a http://nexus.example.com -r myrepo -u user -p pass subdir/`,
	Args: cobra.MaximumNArgs(1),
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	// Get common flags
	address, _ := cmd.Flags().GetString("address")
	repository, _ := cmd.Flags().GetString("repository")
	username, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	configPath, _ := cmd.Flags().GetString("config")
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry")

	// Get subdir argument (optional)
	var subdir string
	if len(args) > 0 {
		subdir = args[0]
	}

	// Load configuration
	cfg, err := config.LoadConfigWithFlags(configPath, map[string]interface{}{
		"nexusAddress": address,
		"user":         username,
		"password":     password,
	})
	if err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create Nexus client
	client := nexus.NewNexusClient(cfg.GetNexusAddress(), cfg.GetUser(), cfg.GetPassword(), quiet, dryRun)

	return runListWithClient(client, repository, subdir, dryRun, quiet)
}

// runListWithClient performs the actual list operation with provided client
// This function is extracted for testability - it can be called with mock clients in tests
func runListWithClient(client nexus.Client, repository, subdir string, dryRun, quiet bool) error {
	// Get files in directory
	files, err := client.GetFilesInDirectory(repository, subdir)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	// Print files
	if dryRun {
		client.Logf("Dry run: Would list %d files", len(files))
	} else {
		if !quiet {
			if subdir == "" {
				fmt.Printf("Files in repository root (%d files):\n", len(files))
			} else {
				fmt.Printf("Files in '%s' (%d files):\n", subdir, len(files))
			}
		}
		for _, file := range files {
			fmt.Println(file)
		}
	}

	return nil
}
