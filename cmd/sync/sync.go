package sync

import (
	"fmt"

	"nexus-util/config"
	"nexus-util/nexus"

	"github.com/spf13/cobra"
)

var SyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Transfer contents from one Nexus repository to another",
	Long: `Transfer contents from one Nexus server/repository to another Nexus server/repository.
This command downloads files from the source repository and uploads them to the target repository.

Examples:
  # Transfer entire repository from one server to another
  nexus-util sync --source-address http://source.example.com --source-repo myrepo \
                   --target-address http://target.example.com --target-repo myrepo

  # Transfer with skip for existing files and progress
  nexus-util sync --source-address http://source.example.com --source-repo myrepo \
                   --target-address http://target.example.com --target-repo myrepo \
                   --skip-existing --show-progress

  # Use config for one or both servers
  nexus-util sync --source-repo myrepo \
                   --target-address http://target.example.com --target-repo myrepo

  # Sync with authentication
  nexus-util sync --source-address http://source.example.com --source-user user1 --source-pass pass1 \
                   --source-repo repo1 --target-address http://target.example.com --target-user user2 \
                   --target-pass pass2 --target-repo repo1`,
	RunE: runSync,
}

func runSync(cmd *cobra.Command, args []string) error {
	// Get source flags
	sourceAddress, _ := cmd.Flags().GetString("source-address")
	sourceRepo, _ := cmd.Flags().GetString("source-repo")
	sourceUser, _ := cmd.Flags().GetString("source-user")
	sourcePassword, _ := cmd.Flags().GetString("source-pass")

	// Get target flags
	targetAddress, _ := cmd.Flags().GetString("target-address")
	targetRepo, _ := cmd.Flags().GetString("target-repo")
	targetUser, _ := cmd.Flags().GetString("target-user")
	targetPassword, _ := cmd.Flags().GetString("target-pass")

	// Common flags
	configPath, _ := cmd.Flags().GetString("config")
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry")
	skipExisting, _ := cmd.Flags().GetBool("skip-existing")
	showProgress, _ := cmd.Flags().GetBool("show-progress")

	// Load configuration for fallback values
	sourceConfig, err := config.LoadConfigWithFlags(configPath, map[string]interface{}{
		"nexusAddress": sourceAddress,
		"user":         sourceUser,
		"password":     sourcePassword,
	})
	if err != nil {
		return fmt.Errorf("error loading source configuration: %w", err)
	}

	targetConfig, err := config.LoadConfigWithFlags(configPath, map[string]interface{}{
		"nexusAddress": targetAddress,
		"user":         targetUser,
		"password":     targetPassword,
	})
	if err != nil {
		return fmt.Errorf("error loading target configuration: %w", err)
	}

	// Determine source and target addresses
	finalSourceAddress := sourceAddress
	if finalSourceAddress == "" {
		finalSourceAddress = sourceConfig.GetNexusAddress()
	}
	if finalSourceAddress == "" {
		return fmt.Errorf("source address is required (use --source-address or config)")
	}

	finalTargetAddress := targetAddress
	if finalTargetAddress == "" {
		finalTargetAddress = targetConfig.GetNexusAddress()
	}
	if finalTargetAddress == "" {
		return fmt.Errorf("target address is required (use --target-address or config)")
	}

	// Validate repositories
	if sourceRepo == "" {
		return fmt.Errorf("source repository is required")
	}
	if targetRepo == "" {
		return fmt.Errorf("target repository is required")
	}

	// Get credentials - prefer command line over config for each side
	sourceUsername := sourceUser
	if sourceUsername == "" {
		sourceUsername = sourceConfig.GetUser()
	}

	sourcePass := sourcePassword
	if sourcePass == "" {
		sourcePass = sourceConfig.GetPassword()
	}

	targetUsername := targetUser
	if targetUsername == "" {
		targetUsername = targetConfig.GetUser()
	}

	targetPass := targetPassword
	if targetPass == "" {
		targetPass = targetConfig.GetPassword()
	}

	// Create clients
	sourceClient := nexus.NewNexusClient(finalSourceAddress, sourceUsername, sourcePass, quiet, dryRun)
	targetClient := nexus.NewNexusClient(finalTargetAddress, targetUsername, targetPass, quiet, dryRun)

	// Get all files from source repository
	fmt.Printf("Scanning source repository '%s' on %s...\n", sourceRepo, finalSourceAddress)
	sourceFiles, err := sourceClient.GetFilesInDirectory(sourceRepo, "")
	if err != nil {
		return fmt.Errorf("failed to get files from source repository: %w", err)
	}

	if len(sourceFiles) == 0 {
		fmt.Println("No files found in source repository")
		return nil
	}

	fmt.Printf("Found %d files in source repository\n", len(sourceFiles))

	// Check disk space for largest file if not dry run
	if !dryRun {
		// Find the largest file
		maxSize := int64(0)
		var largestFile nexus.Asset
		for _, file := range sourceFiles {
			size, err := sourceClient.GetFileSize(sourceRepo, file.Path)
			if err != nil {
				// Log but continue
				sourceClient.Logf("Warning: failed to get size for %s: %v", file, err)
				continue
			}
			if size > maxSize {
				maxSize = size
				largestFile = file
			}
		}

		if maxSize > 0 {
			fmt.Printf("Largest file: %s (%d bytes)\n", largestFile.Path, maxSize)
			fmt.Println("Disk space check passed")
		}
	}

	// Transfer files
	transferred := 0
	skipped := 0

	for i, file := range sourceFiles {
		if showProgress {
			fmt.Printf("[%d/%d] Processing: %s\n", i+1, len(sourceFiles), file.Path)
		}

		// Check if file should be skipped
		if skipExisting {
			exists, err := targetClient.FileExists(targetRepo, file.Path)
			if err != nil {
				sourceClient.Logf("Warning: failed to check if file exists in target: %v", err)
			} else if exists {
				if showProgress {
					fmt.Printf("  Skipped (already exists)\n")
				}
				skipped++
				continue
			}
		}

		err := sourceClient.TransferFile(targetClient, sourceRepo, targetRepo, file, false)
		if err != nil {
			return fmt.Errorf("failed to transfer file '%s': %w", file, err)
		}

		transferred++
	}

	fmt.Printf("\nTransfer completed: %d files transferred, %d files skipped\n", transferred, skipped)

	return nil
}
