package asset

import (
	"fmt"
	"strings"

	"nexus-util/config"
	"nexus-util/nexus"

	"github.com/spf13/cobra"
)

var DeleteCmd = &cobra.Command{
	Use:   "delete [flags] <path>...",
	Short: "Delete files or directories from Nexus repository",
	Long: `Delete files or directories from Nexus OSS Raw Repository.
This command combines the functionality of the original nexus_delete.py script.

Examples:
  # Delete a single file
  nexus-util asset delete -a http://nexus.example.com -r myrepo -u user -p pass file.txt

  # Delete a directory
  nexus-util asset delete -a http://nexus.example.com -r myrepo -u user -p pass dir/

  # Dry run to see what would be deleted
  nexus-util asset delete --dry -a http://nexus.example.com -r myrepo -u user -p pass file.txt`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
	// Get common flags
	address, _ := cmd.Flags().GetString("address")
	repository, _ := cmd.Flags().GetString("repository")
	username, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	configPath, _ := cmd.Flags().GetString("config")
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry")
	insecure, _ := cmd.Flags().GetBool("insecure")

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
	client := nexus.NewNexusClient(cfg.GetNexusAddress(), cfg.GetUser(), cfg.GetPassword(), quiet, dryRun, insecure)

	// Process each path
	for _, path := range args {
		client.Logf("Process path '%s'", path)

		// Determine if it's a directory (ends with /)
		isDir := strings.HasSuffix(path, "/") || strings.HasSuffix(path, "\\")

		if isDir {
			// Delete directory
			if err := client.DeleteDirectory(repository, path); err != nil {
				return fmt.Errorf("failed to delete directory: %w", err)
			}
		} else {
			// Delete file
			if err := client.DeleteFile(repository, path); err != nil {
				return fmt.Errorf("failed to delete file: %w", err)
			}
		}
	}

	// Print browse URL
	linkURL := fmt.Sprintf("%s/#browse/browse:%s", cfg.GetNexusAddress(), repository)
	fmt.Println(linkURL)

	if !quiet {
		fmt.Println("Success!")
	}

	return nil
}
