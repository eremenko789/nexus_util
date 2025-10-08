package main

import (
	"testing"
)

func TestNexusClientCreation(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "test-repo", "testuser", "testpass", false, false)

	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}

	if client.BaseURL != "http://test-nexus.example.com" {
		t.Errorf("Expected BaseURL 'http://test-nexus.example.com', got '%s'", client.BaseURL)
	}

	if client.Repository != "test-repo" {
		t.Errorf("Expected Repository 'test-repo', got '%s'", client.Repository)
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
	client := NewNexusClient("http://test-nexus.example.com/", "test-repo", "testuser", "testpass", false, false)

	if client.BaseURL != "http://test-nexus.example.com" {
		t.Errorf("Expected BaseURL without trailing slash 'http://test-nexus.example.com', got '%s'", client.BaseURL)
	}
}

func TestNexusClientQuietMode(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "test-repo", "testuser", "testpass", true, false)

	if !client.Quiet {
		t.Error("Expected Quiet mode to be enabled")
	}
}

func TestNexusClientDryRunMode(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "test-repo", "testuser", "testpass", false, true)

	if !client.DryRun {
		t.Error("Expected DryRun mode to be enabled")
	}
}

func TestNexusClientHTTPClient(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "test-repo", "testuser", "testpass", false, false)

	if client.HTTPClient == nil {
		t.Error("Expected HTTPClient to be initialized")
	}
}
