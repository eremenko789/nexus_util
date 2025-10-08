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

var pullCmd = &cobra.Command{
	Use:   "pull [flags] <source>...",
	Short: "Download files or directories from Nexus repository",
	Long: `Download files or directories from Nexus OSS Raw Repository.
This command combines the functionality of the original nexus_pull.py script.

Examples:
  # Download a single file
  nexus-util pull -a http://nexus.example.com -r myrepo -u user -p pass -d ./downloads file.txt

  # Download a directory
  nexus-util pull -a http://nexus.example.com -r myrepo -u user -p pass -d ./downloads dir/

  # Download with custom root path
  nexus-util pull -a http://nexus.example.com -r myrepo -u user -p pass -d ./downloads --root custom/path file.txt`,
	Args: cobra.MinimumNArgs(1),
	RunE: runPull,
}

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

func init() {
	// Push command flags
	pushCmd.Flags().StringP("destination", "d", "", "Destination path in Nexus repository")
	pushCmd.Flags().Bool("relative", false, "Use relative paths when uploading directories")

	// Pull command flags
	pullCmd.Flags().StringP("destination", "d", "", "Local destination path (required)")
	pullCmd.Flags().String("root", "", "Root path in Nexus repository")
	pullCmd.MarkFlagRequired("destination")
}

func runPush(cmd *cobra.Command, args []string) error {
	// Get common flags
	address, _ := cmd.Flags().GetString("address")
	repository, _ := cmd.Flags().GetString("repository")
	username, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry")

	// Get push-specific flags
	destination, _ := cmd.Flags().GetString("destination")
	relative, _ := cmd.Flags().GetBool("relative")

	// Validate required flags
	if address == "" || repository == "" {
		return fmt.Errorf("address and repository are required")
	}

	// Create Nexus client
	client := NewNexusClient(address, repository, username, password, quiet, dryRun)

	// Process each path
	for _, path := range args {
		client.log("Process path '%s'", path)

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
			client.log("path '%s' is directory", path)
			if err := client.UploadDirectory(path, relative); err != nil {
				return fmt.Errorf("failed to upload directory: %w", err)
			}
		} else {
			// Upload file
			client.log("path '%s' is file", path)
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

			if err := client.UploadFile(path, destPath); err != nil {
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
	linkURL := fmt.Sprintf("%s/#browse/browse:%s:%s", address, repository, linkDest)
	fmt.Println(linkURL)

	if !quiet {
		fmt.Println("Success!")
	}

	return nil
}

func runPull(cmd *cobra.Command, args []string) error {
	// Get common flags
	address, _ := cmd.Flags().GetString("address")
	repository, _ := cmd.Flags().GetString("repository")
	username, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry")

	// Get pull-specific flags
	destination, _ := cmd.Flags().GetString("destination")
	root, _ := cmd.Flags().GetString("root")

	// Validate required flags
	if address == "" || repository == "" {
		return fmt.Errorf("address and repository are required")
	}

	// Validate destination directory
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
	client := NewNexusClient(address, repository, username, password, quiet, dryRun)

	// Process each source
	for _, source := range args {
		client.log("Process source '%s'", source)

		// Determine if it's a directory (ends with /)
		isDir := strings.HasSuffix(source, "/") || strings.HasSuffix(source, "\\")

		if isDir {
			// Download directory
			client.log("source '%s' is directory", source)
			if err := downloadDirectory(client, source, destination, root); err != nil {
				return fmt.Errorf("failed to download directory: %w", err)
			}
		} else {
			// Download file
			client.log("source '%s' is file", source)
			if err := downloadFile(client, source, destination, root); err != nil {
				return fmt.Errorf("failed to download file: %w", err)
			}
		}
	}

	if !quiet {
		fmt.Println("Success!")
	}

	return nil
}

func runDelete(cmd *cobra.Command, args []string) error {
	// Get common flags
	address, _ := cmd.Flags().GetString("address")
	repository, _ := cmd.Flags().GetString("repository")
	username, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry")

	// Validate required flags
	if address == "" || repository == "" {
		return fmt.Errorf("address and repository are required")
	}

	// Create Nexus client
	client := NewNexusClient(address, repository, username, password, quiet, dryRun)

	// Process each path
	for _, path := range args {
		client.log("Process path '%s'", path)

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
	linkURL := fmt.Sprintf("%s/#browse/browse:%s", address, repository)
	fmt.Println(linkURL)

	if !quiet {
		fmt.Println("Success!")
	}

	return nil
}

func downloadFile(client *NexusClient, filePath, destination, root string) error {
	client.log("Download file %s ...", filePath)

	// Build full path if root is specified
	var fullPath string
	if root != "" && !strings.HasPrefix(filePath, root) {
		fullPath = root + "/" + filePath
	} else {
		fullPath = filePath
	}

	// Determine destination path
	fileName := filepath.Base(filePath)
	destPath := filepath.Join(destination, fileName)

	// Download the file
	return client.DownloadFile(fullPath, destPath)
}

func downloadDirectory(client *NexusClient, dirPath, destination, root string) error {
	client.log("Download dir %s ...", dirPath)

	// Build full path if root is specified
	var fullPath string
	if root != "" && !strings.HasPrefix(dirPath, root) {
		fullPath = root + "/" + dirPath
	} else {
		fullPath = dirPath
	}

	// Get all files in directory
	files, err := client.GetFilesInDirectory(fullPath)
	if err != nil {
		return fmt.Errorf("failed to get files in directory: %w", err)
	}

	// Download each file
	for _, file := range files {
		client.log("file '%s' searched", file)

		// Calculate relative path
		var relPath string
		if root != "" {
			relPath = strings.TrimPrefix(file, root+"/")
		} else {
			relPath = file
		}

		// Build destination path
		destPath := filepath.Join(destination, relPath)

		// Download the file
		if err := client.DownloadFile(file, destPath); err != nil {
			return fmt.Errorf("failed to download file %s: %w", file, err)
		}
	}

	client.log("Success dir %s ...", dirPath)
	return nil
}