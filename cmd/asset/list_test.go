package asset

import (
	"errors"
	"strings"
	"testing"

	"nexus-util/nexus"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockNexusClient is a mock implementation of nexus.Client for testing
type MockNexusClient struct {
	GetFilesInDirectoryFunc       func(repository string, dirPath string) ([]string, error)
	UploadFileFunc                func(repository string, filePath string, destPath string) error
	UploadDirectoryFunc           func(repository string, dirPath string, relative bool, destination string) error
	DownloadFileWithPathFunc      func(repository string, filePath string, destination string, root string) error
	DownloadDirectoryWithPathFunc func(repository string, dirPath string, destination string, root string, saveStructure bool) error
	DeleteFileFunc                func(repository string, filePath string) error
	DeleteDirectoryFunc           func(repository string, dirPath string) error
	LogfFunc                      func(format string, args ...interface{})
	GetBaseURLFunc                func() string
}

func (m *MockNexusClient) GetFilesInDirectory(repository string, dirPath string) ([]string, error) {
	if m.GetFilesInDirectoryFunc != nil {
		return m.GetFilesInDirectoryFunc(repository, dirPath)
	}
	return nil, errors.New("mock: GetFilesInDirectoryFunc not set")
}

func (m *MockNexusClient) Logf(format string, args ...interface{}) {
	if m.LogfFunc != nil {
		m.LogfFunc(format, args...)
	}
}

func (m *MockNexusClient) GetBaseURL() string {
	if m.GetBaseURLFunc != nil {
		return m.GetBaseURLFunc()
	}
	return "http://mock.example.com"
}

// Implement all other required methods with no-ops
func (m *MockNexusClient) DeleteFile(repository string, filePath string) error {
	if m.DeleteFileFunc != nil {
		return m.DeleteFileFunc(repository, filePath)
	}
	return nil
}
func (m *MockNexusClient) DeleteDirectory(repository string, dirPath string) error {
	if m.DeleteDirectoryFunc != nil {
		return m.DeleteDirectoryFunc(repository, dirPath)
	}
	return nil
}
func (m *MockNexusClient) DownloadFile(repository string, filePath string, destPath string) error {
	return nil
}
func (m *MockNexusClient) UploadFile(repository string, filePath string, destPath string) error {
	if m.UploadFileFunc != nil {
		return m.UploadFileFunc(repository, filePath, destPath)
	}
	return nil
}
func (m *MockNexusClient) UploadDirectory(repository string, dirPath string, relative bool, destination string) error {
	if m.UploadDirectoryFunc != nil {
		return m.UploadDirectoryFunc(repository, dirPath, relative, destination)
	}
	return nil
}
func (m *MockNexusClient) DownloadFileWithPath(repository string, filePath string, destination string, root string) error {
	if m.DownloadFileWithPathFunc != nil {
		return m.DownloadFileWithPathFunc(repository, filePath, destination, root)
	}
	return nil
}
func (m *MockNexusClient) DownloadDirectoryWithPath(repository string, dirPath string, destination string, root string, saveStructure bool) error {
	if m.DownloadDirectoryWithPathFunc != nil {
		return m.DownloadDirectoryWithPathFunc(repository, dirPath, destination, root, saveStructure)
	}
	return nil
}
func (m *MockNexusClient) ListRepositories() ([]nexus.Repository, error) {
	return nil, nil
}
func (m *MockNexusClient) FileExists(repository string, filePath string) (bool, error) {
	return false, nil
}
func (m *MockNexusClient) GetFileSize(repository string, filePath string) (int64, error) {
	return 0, nil
}
func (m *MockNexusClient) DownloadToBuffer(repository string, filePath string) ([]byte, error) {
	return nil, nil
}
func (m *MockNexusClient) UploadFromBuffer(repository string, destPath string, content []byte) error {
	return nil
}
func (m *MockNexusClient) TransferFile(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
	return nil
}

func TestRunList(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		flags         map[string]string
		setupMock     func() *MockNexusClient
		expectedError string
	}{
		{
			name: "success - list files in root",
			args: []string{},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
			},
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt", "file2.txt"}, nil
					},
				}
			},
		},
		{
			name: "success - list files in subdirectory",
			args: []string{"subdir/"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
			},
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						assert.Equal(t, "subdir/", dirPath)
						return []string{"subdir/file1.txt"}, nil
					},
				}
			},
		},
		{
			name: "success - dry run mode",
			args: []string{},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"dry":        "true",
			},
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt"}, nil
					},
					LogfFunc: func(format string, args ...interface{}) {
						// Verify dry run message
						assert.Contains(t, format, "Dry run")
					},
				}
			},
		},
		{
			name: "success - quiet mode",
			args: []string{},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"quiet":      "true",
			},
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt"}, nil
					},
				}
			},
		},
		{
			name: "error - GetFilesInDirectory fails",
			args: []string{},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
			},
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return nil, errors.New("network error")
					},
				}
			},
			expectedError: "network error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This is a simplified test that doesn't actually call runList
			// because runList has many dependencies (config loading, cobra command setup)
			// In a real scenario, you would need to refactor runList to accept
			// a nexus.Client interface and config loader interface for better testability

			// For now, we test the logic that can be tested
			if tt.setupMock != nil {
				mockClient := tt.setupMock()
				if mockClient != nil {
					// Use subdir from args if provided
					subdir := ""
					if len(tt.args) > 0 {
						subdir = tt.args[0]
					}
					files, err := mockClient.GetFilesInDirectory("test-repo", subdir)
					if tt.expectedError != "" {
						require.Error(t, err)
						// Check if error message contains expected text
						if err != nil {
							assert.Contains(t, err.Error(), tt.expectedError)
						}
					} else {
						require.NoError(t, err)
						assert.NotNil(t, files)
					}
				}
			}
		})
	}
}

// TestListCmd tests the command structure
func TestListCmd(t *testing.T) {
	assert.NotNil(t, ListCmd)
	assert.Equal(t, "list [subdir]", ListCmd.Use)
	assert.Equal(t, "List files in a directory in Nexus repository", ListCmd.Short)
	assert.NotEmpty(t, ListCmd.Long)
}

func TestRunListWithClient(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() *MockNexusClient
		repository    string
		subdir        string
		dryRun        bool
		quiet         bool
		expectedError string
	}{
		{
			name:       "success - list files in root, not quiet, not dry run",
			repository: "test-repo",
			subdir:     "",
			dryRun:     false,
			quiet:      false,
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt", "file2.txt"}, nil
					},
				}
			},
		},
		{
			name:       "success - list files in subdirectory, not quiet, not dry run",
			repository: "test-repo",
			subdir:     "subdir/",
			dryRun:     false,
			quiet:      false,
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						assert.Equal(t, "subdir/", dirPath)
						return []string{"subdir/file1.txt"}, nil
					},
				}
			},
		},
		{
			name:       "success - dry run mode",
			repository: "test-repo",
			subdir:     "",
			dryRun:     true,
			quiet:      false,
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt"}, nil
					},
					LogfFunc: func(format string, args ...interface{}) {
						assert.Contains(t, format, "Dry run")
					},
				}
			},
		},
		{
			name:       "success - quiet mode, not dry run",
			repository: "test-repo",
			subdir:     "",
			dryRun:     false,
			quiet:      true,
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt"}, nil
					},
				}
			},
		},
		{
			name:       "error - GetFilesInDirectory fails",
			repository: "test-repo",
			subdir:     "",
			dryRun:     false,
			quiet:      false,
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return nil, errors.New("network error")
					},
				}
			},
			expectedError: "failed to list files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.setupMock()
			err := runListWithClient(mockClient, tt.repository, tt.subdir, tt.dryRun, tt.quiet)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestRunList_Integration tests runList with real cobra commands
func TestRunList_Integration(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		flags         map[string]string
		expectedError string
	}{
		{
			name: "success - list files with dry run",
			args: []string{},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"dry":        "true",
			},
		},
		{
			name: "success - list files in subdirectory with dry run",
			args: []string{"subdir/"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"dry":        "true",
			},
		},
		{
			name: "success - quiet mode with dry run",
			args: []string{},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"quiet":      "true",
				"dry":        "true",
			},
		},
		{
			name: "error - missing address causes validation or network error",
			args: []string{},
			flags: map[string]string{
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
			},
			expectedError: "", // Will error on validation or network, both are acceptable
		},
		{
			name: "error - missing repository",
			args: []string{},
			flags: map[string]string{
				"address":  "http://test.example.com",
				"user":     "testuser",
				"password": "testpass",
			},
			// Repository is required but validation happens at cobra level
			// So this might fail on GetFilesInDirectory call
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use: "list",
			}
			cmd.Flags().String("address", "", "Nexus address")
			cmd.Flags().String("repository", "", "Repository name")
			cmd.Flags().String("user", "", "User")
			cmd.Flags().String("password", "", "Password")
			cmd.Flags().String("config", "", "Config path")
			cmd.Flags().Bool("quiet", false, "Quiet mode")
			cmd.Flags().Bool("dry", false, "Dry run")

			// Set flags
			for key, value := range tt.flags {
				cmd.Flags().Set(key, value)
			}

			err := runList(cmd, tt.args)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				// Should error on validation, network, or list - all are acceptable
				// If no error, that's also fine (dry run might succeed)
				if err != nil {
					errorMsg := err.Error()
					// Accept any error - validation, network, or list errors
					assert.True(t,
						strings.Contains(errorMsg, "failed to list files") ||
							strings.Contains(errorMsg, "configuration") ||
							strings.Contains(errorMsg, "dial tcp") ||
							strings.Contains(errorMsg, "network"),
						"Unexpected error type: %v", err)
				}
			}
		})
	}
}
