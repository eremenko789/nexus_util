package main

import (
	"bytes"
	"context"
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

const (
	// HTTP status codes
	httpStatusOK        = 200
	httpStatusNoContent = 204
	httpStatusNotFound  = 404

	// File permissions
	dirPerm  = 0o755
	filePerm = 0o600

	// HTTP client timeout
	httpTimeout = 30 * time.Second
)

// NexusClient represents a client for Nexus OSS API
type NexusClient struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client
	Quiet      bool
	DryRun     bool
}

// NewNexusClient creates a new Nexus client
func NewNexusClient(baseURL, username, password string, quiet, dryRun bool) *NexusClient {
	// Remove trailing slash from baseURL
	baseURL = strings.TrimSuffix(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "\\")

	return &NexusClient{
		BaseURL:    baseURL,
		Username:   username,
		Password:   password,
		HTTPClient: &http.Client{Timeout: httpTimeout},
		Quiet:      quiet,
		DryRun:     dryRun,
	}
}

// logf prints a message if not in quiet mode
func (c *NexusClient) logf(format string, args ...interface{}) {
	if !c.Quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// makeRequest makes an HTTP request with basic auth
func (c *NexusClient) makeRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), method, url, body)
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

// Repository represents a Nexus repository
type Repository struct {
	Name   string `json:"name"`
	Format string `json:"format"`
	Type   string `json:"type"`
	URL    string `json:"url"`
}

// GetFilesInDirectory gets all files in a directory recursively
func (c *NexusClient) GetFilesInDirectory(repository string, dirPath string) ([]string, error) {
	var allFiles []string
	continuationToken := ""

	for {
		// Build search URL
		searchURL := fmt.Sprintf("%s/service/rest/v1/search/assets?repository=%s", c.BaseURL, repository)

		if dirPath != "" {
			dirPath = strings.TrimSuffix(dirPath, "/")
			// URL encode the directory path
			encodedPath := url.QueryEscape(dirPath + "/")
			searchURL += "&name=" + encodedPath + "*"
		}

		if continuationToken != "" {
			searchURL += "&continuationToken=" + continuationToken
		}

		c.logf("REST API request: %s", searchURL)

		resp, err := c.makeRequest("GET", searchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to search assets: %w", err)
		}

		if resp.StatusCode != httpStatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("search request failed with status %d", resp.StatusCode)
		}

		var searchResp SearchAssetsResponse
		if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode search response: %w", err)
		}
		resp.Body.Close()

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

	c.logf("Found %d files in directory '%s'", len(allFiles), dirPath)
	return allFiles, nil
}

// DeleteFile deletes a file from Nexus repository
func (c *NexusClient) DeleteFile(repository string, filePath string) error {
	fileURL := fmt.Sprintf("%s/repository/%s/%s", c.BaseURL, repository, filePath)

	if c.DryRun {
		c.logf("File '%s' planned for deletion from %s", filePath, fileURL)
		return nil
	}

	c.logf("Deleting file '%s' from %s...", filePath, fileURL)

	resp, err := c.makeRequest("DELETE", fileURL, nil)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case httpStatusNotFound:
		c.logf("File '%s' not found in repository (404)", filePath)
	case httpStatusNoContent:
		c.logf("File '%s' deleted successfully", filePath)
	default:
		return fmt.Errorf("unexpected response code %d for file '%s'", resp.StatusCode, filePath)
	}

	return nil
}

// DeleteDirectory deletes all files in a directory
func (c *NexusClient) DeleteDirectory(repository string, dirPath string) error {
	// Remove trailing slash
	dirPath = strings.TrimSuffix(dirPath, "/")
	dirPath = strings.TrimSuffix(dirPath, "\\")

	c.logf("Deleting directory '%s' from repository...", dirPath)

	files, err := c.GetFilesInDirectory(repository, dirPath)
	if err != nil {
		return fmt.Errorf("failed to get files in directory: %w", err)
	}

	if len(files) == 0 {
		c.logf("No files found in directory '%s'", dirPath)
		return nil
	}

	deletedCount := 0
	for _, filePath := range files {
		if err := c.DeleteFile(repository, filePath); err != nil {
			return fmt.Errorf("failed to delete file %s: %w", filePath, err)
		}
		deletedCount++
	}

	c.logf("Directory '%s' deletion completed. %d files processed", dirPath, deletedCount)
	return nil
}

// DownloadFile downloads a file from Nexus repository
func (c *NexusClient) DownloadFile(repository string, filePath string, destPath string) error {
	// Create destination directory if it doesn't exist
	if c.DryRun {
		c.logf("Directory '%s' planned for creation", filepath.Dir(destPath))
	} else if err := os.MkdirAll(filepath.Dir(destPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Build download URL
	encodedPath := url.QueryEscape(filePath)
	downloadURL := fmt.Sprintf("%s/service/rest/v1/search/assets/download?repository=%s&name=%s",
		c.BaseURL, repository, encodedPath)

	c.logf("REST API: %s", downloadURL)
	c.logf("DESTINATION: %s", destPath)

	if c.DryRun {
		c.logf("File '%s' planned for download to %s", filePath, destPath)
		return nil
	}

	resp, err := c.makeRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != httpStatusOK {
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

	c.logf("Success file download...")
	return nil
}

// UploadFile uploads a file to Nexus repository
func (c *NexusClient) UploadFile(repository string, filePath string, destPath string) error {
	fileURL := fmt.Sprintf("%s/repository/%s/%s", c.BaseURL, repository, destPath)

	if c.DryRun {
		c.logf("File '%s' planned for pushing to %s", filePath, fileURL)
		return nil
	}

	c.logf("File '%s' will be pushed as %s...", filePath, fileURL)

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

	if resp.StatusCode >= httpStatusOK && resp.StatusCode < 300 {
		c.logf("Sending file '%s' completed", filePath)
	} else {
		return fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}

	return nil
}

// UploadDirectory uploads all files in a directory recursively
func (c *NexusClient) UploadDirectory(repository string, dirPath string, relative bool, destination string) error {
	c.logf("Process directory '%s'", dirPath)
	if destination == "" {
		c.logf("Destination is empty, using default '/'")
	}
	c.logf("Destination: %s", destination)

	uploadFunc := func(path string, info os.FileInfo, err error) error {
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
			destPath = destination + relPath
		} else {
			destPath = destination + path
		}

		// Convert to forward slashes for URL
		destPath = strings.ReplaceAll(destPath, "\\", "/")
		c.logf("DestPath: %s", destPath)

		return c.UploadFile(repository, path, destPath)
	}

	return filepath.Walk(dirPath, uploadFunc)
}

// DownloadFileWithPath downloads a file from Nexus repository with custom destination path
func (c *NexusClient) DownloadFileWithPath(repository string, filePath string, destination string, root string) error {
	c.logf("Download file %s ...", filePath)

	// Build full path if root is specified
	var fullPath string
	if root != "" && !strings.HasPrefix(filePath, root) {
		fullPath = root + "/" + filePath
	} else {
		fullPath = filePath
	}

	// Determine destination path
	fileName := filepath.Base(filePath)
	c.logf("File name: %s", fileName)
	destPath := filepath.Join(destination, fileName)
	c.logf("Destination path: %s", destPath)

	// Download the file
	return c.DownloadFile(repository, fullPath, destPath)
}

// DownloadDirectoryWithPath downloads a directory from Nexus repository with custom destination path
func (c *NexusClient) DownloadDirectoryWithPath(repository string, dirPath string, destination string, root string, saveStructure bool) error {
	c.logf("Download dir %s ...", dirPath)

	// Build full path if root is specified
	var fullPath string
	if root != "" && !strings.HasPrefix(dirPath, root) {
		fullPath = root + "/" + dirPath
	} else {
		fullPath = dirPath
	}

	// Get all files in directory
	files, err := c.GetFilesInDirectory(repository, fullPath)
	if err != nil {
		return fmt.Errorf("failed to get files in directory: %w", err)
	}

	// Download each file
	for _, file := range files {
		c.logf("file '%s' searched", file)

		// Calculate relative path
		var relPath string
		if root != "" {
			relPath = strings.TrimPrefix(file, root+"/")
		} else {
			relPath = file
		}

		// Get the filename from the variable 'file', which may contain a relative path
		fileName := filepath.Base(file)
		c.logf("File name: %s", fileName)

		// Build destination path
		var destPath string
		if saveStructure {
			destPath = filepath.Join(destination, relPath)
		} else {
			destPath = filepath.Join(destination, fileName)
		}
		c.logf("Destination path: %s", destPath)

		// Download the file
		if err := c.DownloadFile(repository, file, destPath); err != nil {
			return fmt.Errorf("failed to download file %s: %w", file, err)
		}
	}

	c.logf("Success dir %s ...", dirPath)
	return nil
}

// ListRepositories lists all repositories configured in the Nexus instance
func (c *NexusClient) ListRepositories() ([]Repository, error) {
	// Build repositories API URL
	reposURL := fmt.Sprintf("%s/service/rest/v1/repositories", c.BaseURL)

	c.logf("REST API request: %s", reposURL)

	if c.DryRun {
		c.logf("Dry run: Would list repositories from %s", reposURL)
		// Return empty slice for dry run
		return []Repository{}, nil
	}

	resp, err := c.makeRequest("GET", reposURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != httpStatusOK {
		return nil, fmt.Errorf("repositories request failed with status %d", resp.StatusCode)
	}

	var repositories []Repository
	if err := json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
		return nil, fmt.Errorf("failed to decode repositories response: %w", err)
	}

	c.logf("Found %d repositories", len(repositories))
	return repositories, nil
}
