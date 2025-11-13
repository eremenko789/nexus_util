package nexus

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

func encodeRepositoryPath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

func (c *NexusClient) repositoryURL(repository, assetPath string) string {
	encodedPath := encodeRepositoryPath(assetPath)
	return fmt.Sprintf("%s/repository/%s/%s", c.BaseURL, repository, encodedPath)
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

// Logf prints a message if not in quiet mode
func (c *NexusClient) Logf(format string, args ...interface{}) {
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

		c.Logf("REST API request: %s", searchURL)

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

	c.Logf("Found %d files in directory '%s'", len(allFiles), dirPath)
	return allFiles, nil
}

// DeleteFile deletes a file from Nexus repository
func (c *NexusClient) DeleteFile(repository string, filePath string) error {
	fileURL := c.repositoryURL(repository, filePath)

	if c.DryRun {
		c.Logf("File '%s' planned for deletion from %s", filePath, fileURL)
		return nil
	}

	c.Logf("Deleting file '%s' from %s...", filePath, fileURL)

	resp, err := c.makeRequest("DELETE", fileURL, nil)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case httpStatusNotFound:
		c.Logf("File '%s' not found in repository (404)", filePath)
	case httpStatusNoContent:
		c.Logf("File '%s' deleted successfully", filePath)
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

	c.Logf("Deleting directory '%s' from repository...", dirPath)

	files, err := c.GetFilesInDirectory(repository, dirPath)
	if err != nil {
		return fmt.Errorf("failed to get files in directory: %w", err)
	}

	if len(files) == 0 {
		c.Logf("No files found in directory '%s'", dirPath)
		return nil
	}

	deletedCount := 0
	for _, filePath := range files {
		if err := c.DeleteFile(repository, filePath); err != nil {
			return fmt.Errorf("failed to delete file %s: %w", filePath, err)
		}
		deletedCount++
	}

	c.Logf("Directory '%s' deletion completed. %d files processed", dirPath, deletedCount)
	return nil
}

// DownloadFile downloads a file from Nexus repository
func (c *NexusClient) DownloadFile(repository string, filePath string, destPath string) error {
	// Create destination directory if it doesn't exist
	if c.DryRun {
		c.Logf("Directory '%s' planned for creation", filepath.Dir(destPath))
	} else if err := os.MkdirAll(filepath.Dir(destPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Build download URL
	encodedPath := url.QueryEscape(filePath)
	downloadURL := fmt.Sprintf("%s/service/rest/v1/search/assets/download?repository=%s&name=%s",
		c.BaseURL, repository, encodedPath)

	c.Logf("REST API: %s", downloadURL)
	c.Logf("DESTINATION: %s", destPath)

	if c.DryRun {
		c.Logf("File '%s' planned for download to %s", filePath, destPath)
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

	c.Logf("Success file download...")
	return nil
}

// UploadFile uploads a file to Nexus repository
func (c *NexusClient) UploadFile(repository string, filePath string, destPath string) error {
	fileURL := c.repositoryURL(repository, destPath)

	if c.DryRun {
		c.Logf("File '%s' planned for pushing to %s", filePath, fileURL)
		return nil
	}

	c.Logf("File '%s' will be pushed as %s...", filePath, fileURL)

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
		c.Logf("Sending file '%s' completed", filePath)
	} else {
		return fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}

	return nil
}

// UploadDirectory uploads all files in a directory recursively
func (c *NexusClient) UploadDirectory(repository string, dirPath string, relative bool, destination string) error {
	c.Logf("Process directory '%s'", dirPath)
	if destination == "" {
		c.Logf("Destination is empty, using default '/'")
	}
	c.Logf("Destination: %s", destination)

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
		c.Logf("DestPath: %s", destPath)

		return c.UploadFile(repository, path, destPath)
	}

	return filepath.Walk(dirPath, uploadFunc)
}

// DownloadFileWithPath downloads a file from Nexus repository with custom destination path
func (c *NexusClient) DownloadFileWithPath(repository string, filePath string, destination string, root string) error {
	c.Logf("Download file %s ...", filePath)

	// Build full path if root is specified
	var fullPath string
	if root != "" && !strings.HasPrefix(filePath, root) {
		fullPath = root + "/" + filePath
	} else {
		fullPath = filePath
	}

	// Determine destination path
	fileName := filepath.Base(filePath)
	c.Logf("File name: %s", fileName)
	destPath := filepath.Join(destination, fileName)
	c.Logf("Destination path: %s", destPath)

	// Download the file
	return c.DownloadFile(repository, fullPath, destPath)
}

// DownloadDirectoryWithPath downloads a directory from Nexus repository with custom destination path
func (c *NexusClient) DownloadDirectoryWithPath(repository string, dirPath string, destination string, root string, saveStructure bool) error {
	c.Logf("Download dir %s ...", dirPath)

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
		c.Logf("file '%s' searched", file)

		// Calculate relative path
		var relPath string
		if root != "" {
			relPath = strings.TrimPrefix(file, root+"/")
		} else {
			relPath = file
		}

		// Get the filename from the variable 'file', which may contain a relative path
		fileName := filepath.Base(file)
		c.Logf("File name: %s", fileName)

		// Build destination path
		var destPath string
		if saveStructure {
			destPath = filepath.Join(destination, relPath)
		} else {
			destPath = filepath.Join(destination, fileName)
		}
		c.Logf("Destination path: %s", destPath)

		// Download the file
		if err := c.DownloadFile(repository, file, destPath); err != nil {
			return fmt.Errorf("failed to download file %s: %w", file, err)
		}
	}

	c.Logf("Success dir %s ...", dirPath)
	return nil
}

// ListRepositories lists all repositories configured in the Nexus instance
func (c *NexusClient) ListRepositories() ([]Repository, error) {
	// Build repositories API URL
	reposURL := fmt.Sprintf("%s/service/rest/v1/repositories", c.BaseURL)

	c.Logf("REST API request: %s", reposURL)

	if c.DryRun {
		c.Logf("Dry run: Would list repositories from %s", reposURL)
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

	c.Logf("Found %d repositories", len(repositories))
	return repositories, nil
}

// FileExists checks if a file exists in the Nexus repository
func (c *NexusClient) FileExists(repository string, filePath string) (bool, error) {
	fileURL := c.repositoryURL(repository, filePath)

	resp, err := c.makeRequest("HEAD", fileURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == httpStatusOK, nil
}

// GetFileSize gets the size of a file from the Nexus repository
func (c *NexusClient) GetFileSize(repository string, filePath string) (int64, error) {
	fileURL := c.repositoryURL(repository, filePath)

	resp, err := c.makeRequest("HEAD", fileURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get file size: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != httpStatusOK {
		return 0, fmt.Errorf("file not found (status %d)", resp.StatusCode)
	}

	contentLength := resp.ContentLength
	if contentLength < 0 {
		return 0, fmt.Errorf("invalid content length")
	}

	return contentLength, nil
}

// DownloadToBuffer downloads a file into memory
func (c *NexusClient) DownloadToBuffer(repository string, filePath string) ([]byte, error) {
	// Build download URL
	encodedPath := url.QueryEscape(filePath)
	downloadURL := fmt.Sprintf("%s/service/rest/v1/search/assets/download?repository=%s&name=%s",
		c.BaseURL, repository, encodedPath)

	c.Logf("Downloading to buffer: %s", downloadURL)

	if c.DryRun {
		c.Logf("Dry run: Would download file from %s", downloadURL)
		return nil, nil
	}
	resp, err := c.makeRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != httpStatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// UploadFromBuffer uploads file content from memory
func (c *NexusClient) UploadFromBuffer(repository string, destPath string, content []byte) error {
	fileURL := c.repositoryURL(repository, destPath)

	if c.DryRun {
		c.Logf("File planned for pushing to %s", fileURL)
		return nil
	}

	c.Logf("Uploading from buffer to %s...", fileURL)

	resp, err := c.makeRequest("PUT", fileURL, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= httpStatusOK && resp.StatusCode < 300 {
		c.Logf("Upload completed")
	} else {
		return fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}

	return nil
}

// TransferFile transfers a file between two Nexus servers
func (c *NexusClient) TransferFile(target *NexusClient, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
	// Download from source
	c.Logf("Downloading '%s' from %s...", filePath, c.BaseURL)
	content, err := c.DownloadToBuffer(sourceRepo, filePath)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Upload to target
	c.Logf("Uploading '%s' to %s...", filePath, target.BaseURL)
	if err := target.UploadFromBuffer(targetRepo, filePath, content); err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}
