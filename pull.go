package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull [flags] <source>...",
	Short: "Download files or directories from Nexus repository",
	Long: `Download files or directories from Nexus OSS Raw Repository.
This command combines the functionality of the original nexus_pull.py script.

Examples:
  # Download a single file
  nexus-util asset pull -a http://nexus.example.com -r myrepo -u user -p pass -d ./downloads file.txt

  # Download a directory
  nexus-util asset pull -a http://nexus.example.com -r myrepo -u user -p pass -d ./downloads dir/

  # Download with custom root path
  nexus-util asset pull -a http://nexus.example.com -r myrepo -u user -p pass -d ./downloads --root custom/path file.txt
  
  # Save directory structure
  nexus-util asset pull -a http://nexus.example.com -r myrepo -u user -p pass -d ./downloads --saveStructure dir/subdir1/subdir2/`,
	Args: cobra.MinimumNArgs(1),
	RunE: runPull,
}

func runPull(cmd *cobra.Command, args []string) error {
	// Get common flags
	address, _ := cmd.Flags().GetString("address")
	repository, _ := cmd.Flags().GetString("repository")
	username, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	configPath, _ := cmd.Flags().GetString("config")
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry")

	// Get pull-specific flags
	destination, _ := cmd.Flags().GetString("destination")
	root, _ := cmd.Flags().GetString("root")
	saveStructure, _ := cmd.Flags().GetBool("saveStructure")

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

	// Validate destination directory
	if destination == "" {
		destination = "."
	}

	if _, err := os.Stat(destination); os.IsNotExist(err) {
		return fmt.Errorf("destination path '%s' doesn't exist", destination)
	}
	if info, err := os.Stat(destination); err != nil || !info.IsDir() {
		return fmt.Errorf("destination path '%s' is not a directory", destination)
	}

	// Remove trailing slash from destination
	destination = strings.TrimSuffix(destination, "/")
	destination = strings.TrimSuffix(destination, "\\")

	// Create Nexus client
	client := NewNexusClient(config.GetNexusAddress(), config.GetUser(), config.GetPassword(), quiet, dryRun)

	// Process each source
	for _, source := range args {
		client.logf("Process source '%s'", source)

		// Determine if it's a directory (ends with /)
		isDir := strings.HasSuffix(source, "/") || strings.HasSuffix(source, "\\")

		if isDir {
			// Download directory
			client.logf("source '%s' is directory", source)
			if err := client.DownloadDirectoryWithPath(repository, source, destination, root, saveStructure); err != nil {
				return fmt.Errorf("failed to download directory: %w", err)
			}
		} else {
			// Download file
			client.logf("source '%s' is file", source)
			if err := client.DownloadFileWithPath(repository, source, destination, root); err != nil {
				return fmt.Errorf("failed to download file: %w", err)
			}
		}
	}

	if !quiet {
		fmt.Println("Success!")
	}

	return nil
}
