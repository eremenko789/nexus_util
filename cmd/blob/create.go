package blob

import (
	"fmt"

	"nexus-util/config"
	"nexus-util/nexus"

	"github.com/spf13/cobra"
)

var CreateCmd = &cobra.Command{
	Use:   "create <blob-store-name>",
	Short: "Create a new blob store",
	Long: `Create a new blob store in Nexus instance.
This command uses the Nexus REST API to create blob stores.

Examples:
  # Create a file blob store
  nexus-util blob create my-blob-store -a http://nexus.example.com -u user -p pass --type file --path /path/to/store

  # Create a file blob store with custom config file
  nexus-util blob create my-blob-store -c /path/to/config.yaml --type file --path /path/to/store`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func runCreate(cmd *cobra.Command, args []string) error {
	blobStoreName := args[0]

	// Get flags
	configPath, _ := cmd.Flags().GetString("config")
	address, _ := cmd.Flags().GetString("address")
	user, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry")
	insecure, _ := cmd.Flags().GetBool("insecure")
	blobType, _ := cmd.Flags().GetString("type")
	path, _ := cmd.Flags().GetString("path")
	softQuota, _ := cmd.Flags().GetUint64("soft-quota")
	softQuotaType, _ := cmd.Flags().GetString("soft-quota-type")

	// Validate required flags
	if blobType == "" {
		return fmt.Errorf("--type is required (e.g., 'file')")
	}
	if blobType == "file" && path == "" {
		return fmt.Errorf("--path is required for file blob stores")
	}

	// Load configuration
	flags := map[string]interface{}{
		"nexusAddress": address,
		"user":         user,
		"password":     password,
	}

	cfg, err := config.LoadConfigWithFlags(configPath, flags)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create Nexus client
	client := nexus.NewNexusClient(cfg.GetNexusAddress(), cfg.GetUser(), cfg.GetPassword(), quiet, dryRun, insecure)

	// Create blob store configuration
	blobStoreConfig := nexus.BlobStoreConfig{
		Name: blobStoreName,
		Type: blobType,
		Path: path,
	}

	if softQuota != 0 {
		blobStoreConfig.SoftQuota = &nexus.SoftQuota{
			Limit: softQuota,
			Type:  softQuotaType,
		}
	}

	// Create blob store
	if err := client.CreateBlobStore(blobStoreConfig); err != nil {
		return fmt.Errorf("failed to create blob store: %w", err)
	}

	if !quiet {
		fmt.Printf("Blob store '%s' created successfully\n", blobStoreName)
	}

	return nil
}

func init() {
	CreateCmd.Flags().StringP("type", "t", "", "Blob store type (required, e.g., 'file')")
	CreateCmd.Flags().String("path", "", "Path for file blob store (required for file type)")
	CreateCmd.Flags().String("soft-quota", "", "Soft quota limit (e.g., '100M', '10G')")
	CreateCmd.Flags().String("soft-quota-type", "spaceRemainingQuota", "Soft quota type (spaceRemainingQuota or spaceUsedQuota)")

	BlobCmd.AddCommand(CreateCmd)
}
