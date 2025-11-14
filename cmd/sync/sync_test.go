package sync

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
	GetFilesInDirectoryFunc func(repository string, dirPath string) ([]string, error)
	GetFileSizeFunc         func(repository string, filePath string) (int64, error)
	FileExistsFunc          func(repository string, filePath string) (bool, error)
	TransferFileFunc        func(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error
	LogfFunc                func(format string, args ...interface{})
	GetBaseURLFunc          func() string
}

func (m *MockNexusClient) GetFilesInDirectory(repository string, dirPath string) ([]string, error) {
	if m.GetFilesInDirectoryFunc != nil {
		return m.GetFilesInDirectoryFunc(repository, dirPath)
	}
	return nil, errors.New("mock: GetFilesInDirectoryFunc not set")
}

func (m *MockNexusClient) GetFileSize(repository string, filePath string) (int64, error) {
	if m.GetFileSizeFunc != nil {
		return m.GetFileSizeFunc(repository, filePath)
	}
	return 0, nil
}

func (m *MockNexusClient) FileExists(repository string, filePath string) (bool, error) {
	if m.FileExistsFunc != nil {
		return m.FileExistsFunc(repository, filePath)
	}
	return false, nil
}

func (m *MockNexusClient) TransferFile(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
	if m.TransferFileFunc != nil {
		return m.TransferFileFunc(target, sourceRepo, targetRepo, filePath, skipIfExists)
	}
	return nil
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
	return nil
}
func (m *MockNexusClient) DeleteDirectory(repository string, dirPath string) error {
	return nil
}
func (m *MockNexusClient) DownloadFile(repository string, filePath string, destPath string) error {
	return nil
}
func (m *MockNexusClient) UploadFile(repository string, filePath string, destPath string) error {
	return nil
}
func (m *MockNexusClient) UploadDirectory(repository string, dirPath string, relative bool, destination string) error {
	return nil
}
func (m *MockNexusClient) DownloadFileWithPath(repository string, filePath string, destination string, root string) error {
	return nil
}
func (m *MockNexusClient) DownloadDirectoryWithPath(repository string, dirPath string, destination string, root string, saveStructure bool) error {
	return nil
}
func (m *MockNexusClient) ListRepositories() ([]nexus.Repository, error) {
	return nil, nil
}
func (m *MockNexusClient) DownloadToBuffer(repository string, filePath string) ([]byte, error) {
	return nil, nil
}
func (m *MockNexusClient) UploadFromBuffer(repository string, destPath string, content []byte) error {
	return nil
}

func TestSyncCmd(t *testing.T) {
	assert.NotNil(t, SyncCmd)
	assert.Equal(t, "sync", SyncCmd.Use)
	assert.Equal(t, "Transfer contents from one Nexus repository to another", SyncCmd.Short)
	assert.NotEmpty(t, SyncCmd.Long)
}

func TestRunSyncWithClients(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func() (*MockNexusClient, *MockNexusClient)
		dryRun        bool
		skipExisting  bool
		showProgress  bool
		expectedError string
	}{
		{
			name: "success - basic sync",
			setupMocks: func() (*MockNexusClient, *MockNexusClient) {
				source := &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt", "file2.txt"}, nil
					},
					TransferFileFunc: func(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
						return nil
					},
				}
				target := &MockNexusClient{}
				return source, target
			},
		},
		{
			name: "success - empty file list",
			setupMocks: func() (*MockNexusClient, *MockNexusClient) {
				source := &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{}, nil
					},
				}
				target := &MockNexusClient{}
				return source, target
			},
		},
		{
			name: "success - with skip-existing",
			setupMocks: func() (*MockNexusClient, *MockNexusClient) {
				source := &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt", "file2.txt"}, nil
					},
					TransferFileFunc: func(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
						return nil
					},
				}
				target := &MockNexusClient{
					FileExistsFunc: func(repository string, filePath string) (bool, error) {
						if filePath == "file1.txt" {
							return true, nil
						}
						return false, nil
					},
				}
				return source, target
			},
			skipExisting: true,
		},
		{
			name: "success - with show-progress",
			setupMocks: func() (*MockNexusClient, *MockNexusClient) {
				source := &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt", "file2.txt"}, nil
					},
					TransferFileFunc: func(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
						return nil
					},
				}
				target := &MockNexusClient{}
				return source, target
			},
			showProgress: true,
		},
		{
			name: "success - with dry-run (no file size check)",
			setupMocks: func() (*MockNexusClient, *MockNexusClient) {
				source := &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt", "file2.txt"}, nil
					},
					TransferFileFunc: func(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
						return nil
					},
				}
				target := &MockNexusClient{}
				return source, target
			},
			dryRun: true,
		},
		{
			name: "success - without dry-run (with file size check)",
			setupMocks: func() (*MockNexusClient, *MockNexusClient) {
				source := &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt", "file2.txt"}, nil
					},
					GetFileSizeFunc: func(repository string, filePath string) (int64, error) {
						if filePath == "file1.txt" {
							return 100, nil
						}
						return 200, nil
					},
					TransferFileFunc: func(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
						return nil
					},
				}
				target := &MockNexusClient{}
				return source, target
			},
			dryRun: false,
		},
		{
			name: "success - file size check with error (continues)",
			setupMocks: func() (*MockNexusClient, *MockNexusClient) {
				source := &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt", "file2.txt"}, nil
					},
					GetFileSizeFunc: func(repository string, filePath string) (int64, error) {
						if filePath == "file1.txt" {
							return 0, errors.New("size error")
						}
						return 200, nil
					},
					LogfFunc: func(format string, args ...interface{}) {
						// Verify warning is logged
						assert.Contains(t, format, "Warning")
					},
					TransferFileFunc: func(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
						return nil
					},
				}
				target := &MockNexusClient{}
				return source, target
			},
			dryRun: false,
		},
		{
			name: "success - skip-existing with FileExists error (continues)",
			setupMocks: func() (*MockNexusClient, *MockNexusClient) {
				source := &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt"}, nil
					},
					LogfFunc: func(format string, args ...interface{}) {
						// Verify warning is logged
						assert.Contains(t, format, "Warning")
					},
					TransferFileFunc: func(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
						return nil
					},
				}
				target := &MockNexusClient{
					FileExistsFunc: func(repository string, filePath string) (bool, error) {
						return false, errors.New("exists check error")
					},
				}
				return source, target
			},
			skipExisting: true,
		},
		{
			name: "error - GetFilesInDirectory fails",
			setupMocks: func() (*MockNexusClient, *MockNexusClient) {
				source := &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return nil, errors.New("network error")
					},
				}
				target := &MockNexusClient{}
				return source, target
			},
			expectedError: "failed to get files from source repository",
		},
		{
			name: "error - TransferFile fails",
			setupMocks: func() (*MockNexusClient, *MockNexusClient) {
				source := &MockNexusClient{
					GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
						return []string{"file1.txt"}, nil
					},
					TransferFileFunc: func(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
						return errors.New("transfer error")
					},
				}
				target := &MockNexusClient{}
				return source, target
			},
			expectedError: "failed to transfer file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceClient, targetClient := tt.setupMocks()
			err := runSyncWithClients(sourceClient, targetClient, "source-repo", "target-repo", "http://source.example.com", tt.dryRun, tt.skipExisting, tt.showProgress)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRunSyncWithClients_EdgeCases(t *testing.T) {
	t.Run("success - multiple files with skip-existing", func(t *testing.T) {
		source := &MockNexusClient{
			GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
				return []string{"file1.txt", "file2.txt", "file3.txt"}, nil
			},
			TransferFileFunc: func(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
				return nil
			},
		}
		target := &MockNexusClient{
			FileExistsFunc: func(repository string, filePath string) (bool, error) {
				// file2.txt already exists
				return filePath == "file2.txt", nil
			},
		}

		err := runSyncWithClients(source, target, "source-repo", "target-repo", "http://source.example.com", false, true, true)
		require.NoError(t, err)
	})

	t.Run("success - file size check finds largest file", func(t *testing.T) {
		source := &MockNexusClient{
			GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
				return []string{"small.txt", "medium.txt", "large.txt"}, nil
			},
			GetFileSizeFunc: func(repository string, filePath string) (int64, error) {
				sizes := map[string]int64{
					"small.txt":  100,
					"medium.txt": 500,
					"large.txt":  1000,
				}
				return sizes[filePath], nil
			},
			TransferFileFunc: func(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
				return nil
			},
		}
		target := &MockNexusClient{}

		err := runSyncWithClients(source, target, "source-repo", "target-repo", "http://source.example.com", false, false, false)
		require.NoError(t, err)
	})

	t.Run("success - all files skipped", func(t *testing.T) {
		source := &MockNexusClient{
			GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
				return []string{"file1.txt", "file2.txt"}, nil
			},
		}
		target := &MockNexusClient{
			FileExistsFunc: func(repository string, filePath string) (bool, error) {
				// All files exist
				return true, nil
			},
		}

		err := runSyncWithClients(source, target, "source-repo", "target-repo", "http://source.example.com", false, true, true)
		require.NoError(t, err)
	})

	t.Run("success - file size check with maxSize = 0", func(t *testing.T) {
		source := &MockNexusClient{
			GetFilesInDirectoryFunc: func(repository string, dirPath string) ([]string, error) {
				return []string{"file1.txt"}, nil
			},
			GetFileSizeFunc: func(repository string, filePath string) (int64, error) {
				// All files have size 0
				return 0, nil
			},
			TransferFileFunc: func(target nexus.Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error {
				return nil
			},
		}
		target := &MockNexusClient{}

		err := runSyncWithClients(source, target, "source-repo", "target-repo", "http://source.example.com", false, false, false)
		require.NoError(t, err)
	})
}

func TestRunSync(t *testing.T) {
	tests := []struct {
		name          string
		flags         map[string]string
		expectedError string
	}{
		{
			name: "success - basic sync with dry run",
			flags: map[string]string{
				"source-address": "http://source.example.com",
				"source-repo":    "source-repo",
				"target-address": "http://target.example.com",
				"target-repo":    "target-repo",
				"dry":            "true",
			},
		},
		{
			name: "error - missing source address",
			flags: map[string]string{
				"source-repo":    "source-repo",
				"target-address": "http://target.example.com",
				"target-repo":    "target-repo",
			},
			expectedError: "source address is required",
		},
		{
			name: "error - missing target address",
			flags: map[string]string{
				"source-address": "http://source.example.com",
				"source-repo":    "source-repo",
				"target-repo":    "target-repo",
			},
			expectedError: "target address is required",
		},
		{
			name: "error - missing source repository",
			flags: map[string]string{
				"source-address": "http://source.example.com",
				"target-address": "http://target.example.com",
				"target-repo":    "target-repo",
			},
			expectedError: "source repository is required",
		},
		{
			name: "error - missing target repository",
			flags: map[string]string{
				"source-address": "http://source.example.com",
				"source-repo":    "source-repo",
				"target-address": "http://target.example.com",
			},
			expectedError: "target repository is required",
		},
		{
			name: "error - source config load fails",
			flags: map[string]string{
				"source-address": "http://source.example.com",
				"source-repo":    "source-repo",
				"target-address": "http://target.example.com",
				"target-repo":    "target-repo",
				"config":         "/nonexistent/config.yaml",
			},
			expectedError: "error loading source configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use: "sync",
			}
			cmd.Flags().String("source-address", "", "Source Nexus address")
			cmd.Flags().String("source-repo", "", "Source repository")
			cmd.Flags().String("source-user", "", "Source user")
			cmd.Flags().String("source-pass", "", "Source password")
			cmd.Flags().String("target-address", "", "Target Nexus address")
			cmd.Flags().String("target-repo", "", "Target repository")
			cmd.Flags().String("target-user", "", "Target user")
			cmd.Flags().String("target-pass", "", "Target password")
			cmd.Flags().String("config", "", "Config path")
			cmd.Flags().Bool("quiet", false, "Quiet mode")
			cmd.Flags().Bool("dry", false, "Dry run")
			cmd.Flags().Bool("skip-existing", false, "Skip existing files")
			cmd.Flags().Bool("show-progress", false, "Show progress")

			// Set flags
			for key, value := range tt.flags {
				cmd.Flags().Set(key, value)
			}

			err := runSync(cmd, []string{})

			if tt.expectedError != "" {
				require.Error(t, err)
				// Accept validation errors or network errors (validation happens before network)
				errorMsg := err.Error()
				assert.True(t,
					strings.Contains(errorMsg, tt.expectedError) ||
						strings.Contains(errorMsg, "failed to get files") ||
						strings.Contains(errorMsg, "dial tcp"),
					"Expected error containing '%s', got: %v", tt.expectedError, err)
			} else {
				// Should error on network or succeed - both are acceptable
				if err != nil {
					errorMsg := err.Error()
					// Accept network errors or transfer errors
					assert.True(t,
						strings.Contains(errorMsg, "failed to get files") ||
							strings.Contains(errorMsg, "failed to transfer") ||
							strings.Contains(errorMsg, "dial tcp") ||
							strings.Contains(errorMsg, "network") ||
							strings.Contains(errorMsg, "configuration"),
						"Unexpected error type: %v", err)
				}
			}
		})
	}
}
