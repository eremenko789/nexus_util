package nexus

import (
	"testing"
)

func TestNexusClientCreation(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "testuser", "testpass", false, false)

	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}

	if client.BaseURL != "http://test-nexus.example.com" {
		t.Errorf("Expected BaseURL 'http://test-nexus.example.com', got '%s'", client.BaseURL)
	}

	if client.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got '%s'", client.Username)
	}

	if client.Password != "testpass" {
		t.Errorf("Expected Password 'testpass', got '%s'", client.Password)
	}
}

func TestNexusClientWithTrailingSlash(t *testing.T) {
	// Test that trailing slashes are removed
	client := NewNexusClient("http://test-nexus.example.com/", "testuser", "testpass", false, false)

	if client.BaseURL != "http://test-nexus.example.com" {
		t.Errorf("Expected BaseURL without trailing slash 'http://test-nexus.example.com', got '%s'", client.BaseURL)
	}
}

func TestNexusClientQuietMode(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "testuser", "testpass", true, false)

	if !client.Quiet {
		t.Error("Expected Quiet mode to be enabled")
	}
}

func TestNexusClientDryRunMode(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "testuser", "testpass", false, true)

	if !client.DryRun {
		t.Error("Expected DryRun mode to be enabled")
	}
}

func TestNexusClientHTTPClient(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "testuser", "testpass", false, false)

	if client.HTTPClient == nil {
		t.Error("Expected HTTPClient to be initialized")
	}
}

func TestEncodeRepositoryPathWithSpaces(t *testing.T) {
	encoded := encodeRepositoryPath("dir/with space/file name.txt")

	expected := "dir/with%20space/file%20name.txt"
	if encoded != expected {
		t.Fatalf("Expected encoded path '%s', got '%s'", expected, encoded)
	}
}

func TestRepositoryURLWithSpaces(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "testuser", "testpass", false, false)

	got := client.repositoryURL("myrepo", "dir/with space/file name.txt")
	expected := "http://test-nexus.example.com/repository/myrepo/dir/with%20space/file%20name.txt"

	if got != expected {
		t.Fatalf("Expected repository URL '%s', got '%s'", expected, got)
	}
}
