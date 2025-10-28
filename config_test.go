package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test with non-existent config file
	config, err := LoadConfig("/non/existent/path.yaml", nil)
	if err != nil {
		t.Logf("Expected error when config file doesn't exist: %v", err)
	}
	if config == nil {
		t.Error("Expected config to be created even with no file")
	}

	// Test with empty config path (should use default)
	config, err = LoadConfig("", nil)
	if err != nil {
		t.Logf("Expected error when no config file exists: %v", err)
	}
	if config == nil {
		t.Error("Expected config to be created even with no file")
	}
}

func TestConfigWithFile(t *testing.T) {
	const testNexusAddress = "http://test-nexus.example.com"

	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
nexusAddress: "` + testNexusAddress + `"
user: "testuser"
password: "testpass"
`

	err := os.WriteFile(configFile, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := LoadConfig(configFile, nil)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.NexusAddress != testNexusAddress {
		t.Errorf("Expected address '%s', got '%s'", testNexusAddress, config.NexusAddress)
	}

	if config.User != "testuser" {
		t.Errorf("Expected user 'testuser', got '%s'", config.User)
	}

	if config.Password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", config.Password)
	}
}

func TestConfigOverride(t *testing.T) {
	const testNexusAddress = "http://test-nexus.example.com"

	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
nexusAddress: "` + testNexusAddress + `"
user: "testuser"
password: "testpass"
`

	err := os.WriteFile(configFile, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test override with flags
	flags := map[string]interface{}{
		"nexusAddress": "http://override.example.com",
		"user":         "overrideuser",
	}

	config, err := LoadConfig(configFile, flags)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.NexusAddress != "http://override.example.com" {
		t.Errorf("Expected address 'http://override.example.com', got '%s'", config.NexusAddress)
	}

	if config.User != "overrideuser" {
		t.Errorf("Expected user 'overrideuser', got '%s'", config.User)
	}
}

func TestConfigValidation(t *testing.T) {
	// Test valid config
	config := &Config{
		NexusAddress: "http://test-nexus.example.com",
		User:         "testuser",
		Password:     "testpass",
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Expected no validation error, got: %v", err)
	}

	// Test invalid config (missing address)
	config.NexusAddress = ""
	err = config.Validate()
	if err == nil {
		t.Error("Expected validation error for missing address")
	}
}

func TestSaveConfig(t *testing.T) {
	const testNexusAddress = "http://test-nexus.example.com"

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-save-config.yaml")

	config := &Config{
		NexusAddress: testNexusAddress,
		User:         "testuser",
		Password:     "testpass",
	}

	err := SaveConfig(config, configFile)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify the file was created
	if _, statErr := os.Stat(configFile); os.IsNotExist(statErr) {
		t.Error("Config file was not created")
	}

	// Load and verify the saved config
	loadedConfig, err := LoadConfig(configFile, nil)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.NexusAddress != testNexusAddress {
		t.Errorf("Expected address '%s', got '%s'", testNexusAddress, loadedConfig.NexusAddress)
	}
}
