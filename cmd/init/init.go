package initcmd

import (
	"fmt"

	"nexus-util/config"
	"github.com/spf13/cobra"
)

var InitCmd = &cobra.Command{
	Use:   "init [flags]",
	Short: "Initialize configuration file with default values",
	Long: `Initialize a configuration file with default values for Nexus connection.

This command creates a configuration file at the specified location (or default)
with the provided Nexus server details. The configuration file can then be used
to avoid specifying connection details on every command.

Examples:
  # Initialize with default config file location (~/.nexus-util.yaml)
  nexus-util init --address http://nexus.example.com --user myuser --password mypass

  # Initialize with custom config file location
  nexus-util init --config ./my-config.yaml --address http://nexus.example.com --user myuser --password mypass

  # Initialize without password (will be prompted)
  nexus-util init --address http://nexus.example.com --user myuser`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, _ []string) error {
	// Get flags
	address, _ := cmd.Flags().GetString("address")
	user, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	configPath, _ := cmd.Flags().GetString("config")

	// Prompt for password if not provided
	if password == "" {
		fmt.Print("Enter password: ")
		var err error
		password, err = readPassword()
		if err != nil {
			return fmt.Errorf("error reading password: %w", err)
		}
		fmt.Println()
	}

	// Create config
	cfg := &config.Config{
		NexusAddress: address,
		User:         user,
		Password:     password,
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save config
	if err := config.SaveConfig(cfg, configPath); err != nil {
		return fmt.Errorf("error saving configuration: %w", err)
	}

	// Show success message
	actualPath := configPath
	if actualPath == "" {
		actualPath = config.DefaultConfigPath()
	}

	fmt.Printf("Configuration saved to: %s\n", actualPath)
	fmt.Println("You can now use nexus-util commands without specifying connection details.")
	fmt.Println("Example: nexus-util asset push -r myrepo file.txt")

	return nil
}

// readPassword reads a password from stdin without echoing
func readPassword() (string, error) {
	// For simplicity, we'll use a basic approach
	// In a production environment, you might want to use golang.org/x/term
	var password string
	_, err := fmt.Scanln(&password)
	return password, err
}

