package blob

import (
	"fmt"
	"os"
	"text/tabwriter"

	"nexus-util/config"
	"nexus-util/nexus"

	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all blob stores in Nexus instance",
	Long: `List all blob stores configured in the Nexus instance.
This command uses the Nexus REST API to retrieve blob store information.

Examples:
  # List all blob stores
  nexus-util blob list -a http://nexus.example.com -u user -p pass

  # List blob stores in quiet mode
  nexus-util blob list -q -a http://nexus.example.com -u user -p pass

  # List blob stores with custom config file
  nexus-util blob list -c /path/to/config.yaml`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
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

	// List blob stores
	blobStores, err := client.ListBlobStores()
	if err != nil {
		return fmt.Errorf("failed to list blob stores: %w", err)
	}

	// Display results
	if len(blobStores) == 0 {
		fmt.Println("No blob stores found.")
		return nil
	}

	// Create tabwriter for formatted output
	padding := 2
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', 0)
	defer w.Flush()

	// Print header
	fmt.Fprintln(w, "NAME\tTYPE\tAVAILABLE SPACE\tTOTAL SPACE\tBLOB COUNT")
	fmt.Fprintln(w, "----\t----\t---------------\t-----------\t----------")

	// Print blob stores
	for _, blobStore := range blobStores {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
			blobStore.Name,
			blobStore.Type,
			formatBytes(blobStore.AvailableSpaceInBytes),
			formatBytes(blobStore.TotalSizeInBytes),
			blobStore.BlobCount)
	}

	return nil
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func init() {
	BlobCmd.AddCommand(ListCmd)
}
