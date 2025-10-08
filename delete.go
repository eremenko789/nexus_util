package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [flags] <path>...",
	Short: "Delete files or directories from Nexus repository",
	Long: `Delete files or directories from Nexus OSS Raw Repository.
This command combines the functionality of the original nexus_delete.py script.

Examples:
  # Delete a single file
  nexus-util delete -a http://nexus.example.com -r myrepo -u user -p pass file.txt

  # Delete a directory
  nexus-util delete -a http://nexus.example.com -r myrepo -u user -p pass dir/

  # Dry run to see what would be deleted
  nexus-util delete --dry -a http://nexus.example.com -r myrepo -u user -p pass file.txt`,
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

	// Load configuration
	config, err := LoadConfigWithFlags(configPath, map[string]interface{}{
		"nexusAddress": address,
		"user":         username,
		"password":     password,
	})
	if err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create Nexus client
	client := NewNexusClient(config.GetNexusAddress(), repository, config.GetUser(), config.GetPassword(), quiet, dryRun)

	// Process each path
	for _, path := range args {
		client.logf("Process path '%s'", path)

		// Determine if it's a directory (ends with /)
		isDir := strings.HasSuffix(path, "/") || strings.HasSuffix(path, "\\")

		if isDir {
			// Delete directory
			if err := client.DeleteDirectory(path); err != nil {
				return fmt.Errorf("failed to delete directory: %w", err)
			}
		} else {
			// Delete file
			if err := client.DeleteFile(path); err != nil {
				return fmt.Errorf("failed to delete file: %w", err)
			}
		}
	}

	// Print browse URL
	linkURL := fmt.Sprintf("%s/#browse/browse:%s", config.GetNexusAddress(), repository)
	fmt.Println(linkURL)

	if !quiet {
		fmt.Println("Success!")
	}

	return nil
}
