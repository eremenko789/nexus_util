package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all repositories in Nexus instance",
	Long: `List all repositories configured in the Nexus instance with their types and formats.
This command uses the Nexus REST API to retrieve repository information.

Examples:
  # List all repositories
  nexus-util list -a http://nexus.example.com -u user -p pass

  # List repositories in quiet mode
  nexus-util list -q -a http://nexus.example.com -u user -p pass

  # List repositories with custom config file
  nexus-util list -c /path/to/config.yaml`,
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

	// Load configuration
	flags := map[string]interface{}{
		"nexusAddress": address,
		"user":         user,
		"password":     password,
	}

	config, err := LoadConfigWithFlags(configPath, flags)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create Nexus client (repository not needed for listing)
	client := NewNexusClient(config.GetNexusAddress(), config.GetUser(), config.GetPassword(), quiet, dryRun)

	// Debug: output args
	client.logf("List command args: %v", args)

	// List repositories
	repositories, err := client.ListRepositories()
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	// Display results
	if len(repositories) == 0 {
		fmt.Println("No repositories found.")
		return nil
	}

	// Create tabwriter for formatted output
	padding := 2
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', 0)
	defer w.Flush()

	// Print header
	fmt.Fprintln(w, "NAME\tFORMAT\tTYPE\tBROWSER")
	fmt.Fprintln(w, "----\t------\t----\t---")

	// Print repositories
	for _, repo := range repositories {
		browseUrl := client.BaseURL + "/#browse/browse:" + repo.Name
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			repo.Name,
			repo.Format,
			repo.Type,
			browseUrl)
	}

	return nil
}
