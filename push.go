package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [flags] <path>...",
	Short: "Upload files or directories to Nexus repository",
	Long: `Upload files or directories to Nexus OSS Raw Repository.
This command combines the functionality of the original nexus_push.py script.

Examples:
  # Upload a single file
  nexus-util push -a http://nexus.example.com -r myrepo -u user -p pass file.txt

  # Upload a directory
  nexus-util push -a http://nexus.example.com -r myrepo -u user -p pass ./localdir/

  # Upload with custom destination path
  nexus-util push -a http://nexus.example.com -r myrepo -u user -p pass -d custom/path file.txt

  # Dry run to see what would be uploaded
  nexus-util push --dry -a http://nexus.example.com -r myrepo -u user -p pass file.txt`,
	Args: cobra.MinimumNArgs(1),
	RunE: runPush,
}

func runPush(cmd *cobra.Command, args []string) error {
	// Get common flags
	address, _ := cmd.Flags().GetString("address")
	repository, _ := cmd.Flags().GetString("repository")
	username, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	configPath, _ := cmd.Flags().GetString("config")
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry")

	// Get push-specific flags
	destination, _ := cmd.Flags().GetString("destination")
	relative, _ := cmd.Flags().GetBool("relative")

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
	client := NewNexusClient(config.GetNexusAddress(), config.GetUser(), config.GetPassword(), quiet, dryRun)

	// Process each path
	for _, path := range args {
		client.logf("Process path '%s'", path)

		// Check if path exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("path '%s' doesn't exist", path)
		}

		// Determine if it's a directory
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat path '%s': %w", path, err)
		}

		if info.IsDir() {
			// Upload directory
			client.logf("path '%s' is directory", path)
			if err := client.UploadDirectory(repository, path, relative, destination); err != nil {
				return fmt.Errorf("failed to upload directory: %w", err)
			}
		} else {
			// Upload file
			client.logf("path '%s' is file", path)
			var destPath string
			if relative {
				destPath = filepath.Base(path)
			} else {
				destPath = path
			}
			if destination != "" {
				destPath = filepath.Join(destination, destPath)
			}
			// Convert to forward slashes for URL
			destPath = strings.ReplaceAll(destPath, "\\", "/")

			if err := client.UploadFile(repository, path, destPath); err != nil {
				return fmt.Errorf("failed to upload file: %w", err)
			}
		}
	}

	// Print browse URL
	linkDest := destination
	if linkDest == "" {
		linkDest = "."
	}
	linkDest = strings.ReplaceAll(linkDest, "/", "%2F")
	linkURL := fmt.Sprintf("%s/#browse/browse:%s:%s", config.GetNexusAddress(), repository, linkDest)
	fmt.Println(linkURL)

	if !quiet {
		fmt.Println("Success!")
	}

	return nil
}
