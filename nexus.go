package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NexusClient represents a client for Nexus OSS API
type NexusClient struct {
	BaseURL    string
	Repository string
	Username   string
	Password   string
	HTTPClient *http.Client
	Quiet      bool
	DryRun     bool
}

// NewNexusClient creates a new Nexus client
func NewNexusClient(baseURL, repository, username, password string, quiet, dryRun bool) *NexusClient {
	// Remove trailing slash from baseURL
	baseURL = strings.TrimSuffix(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "\\")

	return &NexusClient{
		BaseURL:    baseURL,
		Repository: repository,
		Username:   username,
		Password:   password,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Quiet:      quiet,
		DryRun:     dryRun,
	}
}

// log prints a message if not in quiet mode
func (c *NexusClient) log(format string, args ...interface{}) {
	if !c.Quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// makeRequest makes an HTTP request with basic auth
func (c *NexusClient) makeRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	return c.HTTPClient.Do(req)
}

// SearchAssetsResponse represents the response from Nexus search API
type SearchAssetsResponse struct {
	Items             []Asset `json:"items"`
	ContinuationToken string  `json:"continuationToken"`
}

// Asset represents a file in Nexus repository
type Asset struct {
	Path string `json:"path"`
}

// GetFilesInDirectory gets all files in a directory recursively
func (c *NexusClient) GetFilesInDirectory(dirPath string) ([]string, error) {
	var allFiles []string
	continuationToken := ""

	for {
		// Build search URL
		searchURL := fmt.Sprintf("%s/service/rest/v1/search/assets?repository=%s", c.BaseURL, c.Repository)

		if dirPath != "" {
			dirPath = strings.TrimSuffix(dirPath, "/")
			// URL encode the directory path
			encodedPath := url.QueryEscape(dirPath + "/")
			searchURL += "&name=" + encodedPath + "*"
		}

		if continuationToken != "" {
			searchURL += "&continuationToken=" + continuationToken
		}

		c.log("REST API request: %s", searchURL)

		resp, err := c.makeRequest("GET", searchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to search assets: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("search request failed with status %d", resp.StatusCode)
		}

		var searchResp SearchAssetsResponse
		if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
			return nil, fmt.Errorf("failed to decode search response: %w", err)
		}

		// Filter files that start with the directory path
		for _, item := range searchResp.Items {
			if dirPath == "" || strings.HasPrefix(item.Path, dirPath) {
				allFiles = append(allFiles, item.Path)
			}
		}

		// Check if there are more results
		if searchResp.ContinuationToken == "" {
			break
		}
		continuationToken = searchResp.ContinuationToken
	}

	c.log("Found %d files in directory '%s'", len(allFiles), dirPath)
	return allFiles, nil
}

// DeleteFile deletes a file from Nexus repository
func (c *NexusClient) DeleteFile(filePath string) error {
	fileURL := fmt.Sprintf("%s/repository/%s/%s", c.BaseURL, c.Repository, filePath)

	if c.DryRun {
		c.log("File '%s' planned for deletion from %s", filePath, fileURL)
		return nil
	}

	c.log("Deleting file '%s' from %s...", filePath, fileURL)

	resp, err := c.makeRequest("DELETE", fileURL, nil)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 404:
		c.log("File '%s' not found in repository (404)", filePath)
	case 204:
		c.log("File '%s' deleted successfully", filePath)
	default:
		return fmt.Errorf("unexpected response code %d for file '%s'", resp.StatusCode, filePath)
	}

	return nil
}

// DeleteDirectory deletes all files in a directory
func (c *NexusClient) DeleteDirectory(dirPath string) error {
	// Remove trailing slash
	dirPath = strings.TrimSuffix(dirPath, "/")
	dirPath = strings.TrimSuffix(dirPath, "\\")

	c.log("Deleting directory '%s' from repository...", dirPath)

	files, err := c.GetFilesInDirectory(dirPath)
	if err != nil {
		return fmt.Errorf("failed to get files in directory: %w", err)
	}

	if len(files) == 0 {
		c.log("No files found in directory '%s'", dirPath)
		return nil
	}

	deletedCount := 0
	for _, filePath := range files {
		if err := c.DeleteFile(filePath); err != nil {
			return fmt.Errorf("failed to delete file %s: %w", filePath, err)
		}
		deletedCount++
	}

	c.log("Directory '%s' deletion completed. %d files processed", dirPath, deletedCount)
	return nil
}

// DownloadFile downloads a file from Nexus repository
func (c *NexusClient) DownloadFile(filePath, destPath string) error {
	// Create destination directory if it doesn't exist
	if c.DryRun {
		c.log("Directory '%s' planned for creation", filepath.Dir(destPath))
	} else if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Build download URL
	encodedPath := url.QueryEscape(filePath)
	downloadURL := fmt.Sprintf("%s/service/rest/v1/search/assets/download?repository=%s&name=%s",
		c.BaseURL, c.Repository, encodedPath)

	c.log("REST API: %s", downloadURL)
	c.log("DESTINATION: %s", destPath)

	if c.DryRun {
		c.log("File '%s' planned for download to %s", filePath, destPath)
		return nil
	}

	resp, err := c.makeRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create destination file
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer file.Close()

	// Copy content
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	c.log("Success file download...")
	return nil
}

// UploadFile uploads a file to Nexus repository
func (c *NexusClient) UploadFile(filePath, destPath string) error {
	fileURL := fmt.Sprintf("%s/repository/%s/%s", c.BaseURL, c.Repository, destPath)

	if c.DryRun {
		c.log("File '%s' planned for pushing to %s", filePath, fileURL)
		return nil
	}

	c.log("File '%s' will be pushed as %s...", filePath, fileURL)

	// Read file content
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	resp, err := c.makeRequest("PUT", fileURL, bytes.NewReader(fileContent))
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.log("Sending file '%s' completed", filePath)
	} else {
		return fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}

	return nil
}

// UploadDirectory uploads all files in a directory recursively
func (c *NexusClient) UploadDirectory(dirPath string, relative bool) error {
	c.log("Process directory '%s'", dirPath)

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		var destPath string
		if relative {
			relPath, err := filepath.Rel(dirPath, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}
			destPath = relPath
		} else {
			destPath = path
		}

		// Convert to forward slashes for URL
		destPath = strings.ReplaceAll(destPath, "\\", "/")

		return c.UploadFile(path, destPath)
	})
}
