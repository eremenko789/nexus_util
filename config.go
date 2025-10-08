package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	// File permissions
	configDirPerm  = 0o755
	configFilePerm = 0o600
)

// Config represents the application configuration
type Config struct {
	NexusAddress string `yaml:"nexusAddress" mapstructure:"nexusAddress"`
	User         string `yaml:"user" mapstructure:"user"`
	Password     string `yaml:"password" mapstructure:"password"`
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
	viper.SetDefault("nexusAddress", "")
	viper.SetDefault("user", "")
	viper.SetDefault("password", "")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		// Check if it's a file not found error or file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && !os.IsNotExist(err) {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found or doesn't exist, continue with defaults
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

// LoadConfigWithFlags loads configuration with command line flag overrides
func LoadConfigWithFlags(configPath string, flags map[string]interface{}) (*Config, error) {
	return LoadConfig(configPath, flags)
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, configPath string) error {
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, configDirPerm); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, configFilePerm); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// ValidateConfig validates the configuration
func (c *Config) Validate() error {
	if c.NexusAddress == "" {
		return fmt.Errorf("nexus address is required")
	}
	return nil
}

// GetNexusAddress returns the Nexus address
func (c *Config) GetNexusAddress() string {
	return c.NexusAddress
}

// GetUser returns the username
func (c *Config) GetUser() string {
	return c.User
}

// GetPassword returns the password
func (c *Config) GetPassword() string {
	return c.Password
}
