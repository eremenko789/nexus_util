package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Nexus      NexusConfig `yaml:"nexus" mapstructure:"nexus"`
	Repository string      `yaml:"repository" mapstructure:"repository"`
	User       string      `yaml:"user" mapstructure:"user"`
	Password   string      `yaml:"password" mapstructure:"password"`
}

// NexusConfig represents Nexus server configuration
type NexusConfig struct {
	Address string `yaml:"address" mapstructure:"address"`
}

// DefaultConfigPath returns the default configuration file path
func DefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return "nexus-util.yaml"
	}
	return filepath.Join(homeDir, ".nexus-util.yaml")
}

// LoadConfig loads configuration from file and command line flags
func LoadConfig(configPath string, cmdFlags map[string]interface{}) (*Config, error) {
	// Set default config file path if not provided
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	// Initialize viper
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set default values
	viper.SetDefault("nexus.address", "")
	viper.SetDefault("repository", "")
	viper.SetDefault("user", "")
	viper.SetDefault("password", "")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, continue with defaults
	}

	// Override with command line flags if provided
	for key, value := range cmdFlags {
		if value != nil && value != "" {
			viper.Set(key, value)
		}
	}

	// Unmarshal into Config struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, configPath string) error {
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// ValidateConfig validates the configuration
func (c *Config) Validate() error {
	if c.Nexus.Address == "" {
		return fmt.Errorf("nexus address is required")
	}
	if c.Repository == "" {
		return fmt.Errorf("repository name is required")
	}
	return nil
}

// GetNexusAddress returns the Nexus address
func (c *Config) GetNexusAddress() string {
	return c.Nexus.Address
}

// GetRepository returns the repository name
func (c *Config) GetRepository() string {
	return c.Repository
}

// GetUser returns the username
func (c *Config) GetUser() string {
	return c.User
}

// GetPassword returns the password
func (c *Config) GetPassword() string {
	return c.Password
}