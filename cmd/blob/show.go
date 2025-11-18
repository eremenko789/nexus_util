package blob

import (
	"encoding/json"
	"fmt"

	"nexus-util/config"
	"nexus-util/nexus"

	"github.com/spf13/cobra"
)

var ShowCmd = &cobra.Command{
	Use:   "show <blob-store-name>",
	Short: "Show detailed information about a blob store",
	Long: `Show detailed information about a specific blob store.
This command uses the Nexus REST API to retrieve blob store information.

Examples:
  # Show information about a blob store
  nexus-util blob show my-blob-store -a http://nexus.example.com -u user -p pass

  # Show information with custom config file
  nexus-util blob show my-blob-store -c /path/to/config.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

func runShow(cmd *cobra.Command, args []string) error {
	blobStoreName := args[0]

	// Get flags
	configPath, _ := cmd.Flags().GetString("config")
	address, _ := cmd.Flags().GetString("address")
	user, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry")
	insecure, _ := cmd.Flags().GetBool("insecure")

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

	// Get blob store information
	blobStore, err := client.GetBlobStore(blobStoreName)
	if err != nil {
		return fmt.Errorf("failed to get blob store information: %w", err)
	}

	// Display results in JSON format for detailed information
	jsonData, err := json.MarshalIndent(blobStore, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal blob store information: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func init() {
	BlobCmd.AddCommand(ShowCmd)
}
