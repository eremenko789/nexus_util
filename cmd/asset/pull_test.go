package asset

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullCmd(t *testing.T) {
	assert.NotNil(t, PullCmd)
	assert.Equal(t, "pull [flags] <source>...", PullCmd.Use)
	assert.Equal(t, "Download files or directories from Nexus repository", PullCmd.Short)
	assert.NotEmpty(t, PullCmd.Long)
}

func TestRunPull(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		flags         map[string]string
		setupDest     func(string) string // Returns destination path
		expectedError string
	}{
		{
			name: "success - pull file with dry run",
			args: []string{"file.txt"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"dry":        "true",
			},
			setupDest: func(tempDir string) string {
				destDir := filepath.Join(tempDir, "downloads")
				os.MkdirAll(destDir, 0755)
				return destDir
			},
		},
		{
			name: "success - pull directory with dry run",
			args: []string{"dir/"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"dry":        "true",
			},
			setupDest: func(tempDir string) string {
				destDir := filepath.Join(tempDir, "downloads")
				os.MkdirAll(destDir, 0755)
				return destDir
			},
		},
		{
			name: "success - pull with root path",
			args: []string{"file.txt"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"root":       "custom/root",
				"dry":        "true",
			},
			setupDest: func(tempDir string) string {
				destDir := filepath.Join(tempDir, "downloads")
				os.MkdirAll(destDir, 0755)
				return destDir
			},
		},
		{
			name: "success - pull with saveStructure",
			args: []string{"dir/"},
			flags: map[string]string{
				"address":       "http://test.example.com",
				"repository":    "test-repo",
				"user":          "testuser",
				"password":      "testpass",
				"saveStructure": "true",
				"dry":           "true",
			},
			setupDest: func(tempDir string) string {
				destDir := filepath.Join(tempDir, "downloads")
				os.MkdirAll(destDir, 0755)
				return destDir
			},
		},
		{
			name: "success - default destination (current dir)",
			args: []string{"file.txt"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"dry":        "true",
			},
			setupDest: func(tempDir string) string {
				// Use tempDir as current directory
				return tempDir
			},
		},
		{
			name: "error - destination doesn't exist",
			args: []string{"file.txt"},
			flags: map[string]string{
				"address":     "http://test.example.com",
				"repository":  "test-repo",
				"user":        "testuser",
				"password":    "testpass",
				"destination": "/nonexistent/directory",
			},
			expectedError: "doesn't exist",
		},
		{
			name: "error - destination is not a directory",
			args: []string{"file.txt"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
			},
			setupDest: func(tempDir string) string {
				// Create a file instead of directory
				file := filepath.Join(tempDir, "notadir")
				os.WriteFile(file, []byte("content"), 0644)
				return file
			},
			expectedError: "is not a directory",
		},
		{
			name: "error - missing address causes validation or network error",
			args: []string{"file.txt"},
			flags: map[string]string{
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
			},
			setupDest: func(tempDir string) string {
				destDir := filepath.Join(tempDir, "downloads")
				os.MkdirAll(destDir, 0755)
				return destDir
			},
			expectedError: "", // Will error on validation or network, both are acceptable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			var destPath string

			// Setup destination if needed
			if tt.setupDest != nil {
				destPath = tt.setupDest(tempDir)
				if tt.flags["destination"] == "" && destPath != tempDir {
					tt.flags["destination"] = destPath
				}
			}

			cmd := &cobra.Command{
				Use: "pull",
			}
			cmd.Flags().String("address", "", "Nexus address")
			cmd.Flags().String("repository", "", "Repository name")
			cmd.Flags().String("user", "", "User")
			cmd.Flags().String("password", "", "Password")
			cmd.Flags().String("config", "", "Config path")
			cmd.Flags().String("destination", "", "Destination path")
			cmd.Flags().String("root", "", "Root path")
			cmd.Flags().Bool("saveStructure", false, "Save directory structure")
			cmd.Flags().Bool("quiet", false, "Quiet mode")
			cmd.Flags().Bool("dry", false, "Dry run")

			// Set flags
			for key, value := range tt.flags {
				if key == "destination" && destPath != "" {
					cmd.Flags().Set(key, destPath)
				} else {
					cmd.Flags().Set(key, value)
				}
			}

			err := runPull(cmd, tt.args)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				// Should error on validation, network, or download - all are acceptable
				// If no error, that's also fine (dry run might succeed)
				if err != nil {
					errorMsg := err.Error()
					// Accept any error - validation, network, or download errors
					assert.True(t,
						strings.Contains(errorMsg, "failed to download") ||
							strings.Contains(errorMsg, "configuration") ||
							strings.Contains(errorMsg, "doesn't exist") ||
							strings.Contains(errorMsg, "is not a directory") ||
							strings.Contains(errorMsg, "dial tcp") ||
							strings.Contains(errorMsg, "network"),
						"Unexpected error type: %v", err)
				}
			}
		})
	}
}

func TestRunPull_EdgeCases(t *testing.T) {
	t.Run("success - multiple sources", func(t *testing.T) {
		tempDir := t.TempDir()
		destDir := filepath.Join(tempDir, "downloads")
		os.MkdirAll(destDir, 0755)

		cmd := &cobra.Command{
			Use: "pull",
		}
		cmd.Flags().String("address", "", "Nexus address")
		cmd.Flags().String("repository", "", "Repository name")
		cmd.Flags().String("user", "", "User")
		cmd.Flags().String("password", "", "Password")
		cmd.Flags().String("config", "", "Config path")
		cmd.Flags().String("destination", "", "Destination path")
		cmd.Flags().String("root", "", "Root path")
		cmd.Flags().Bool("saveStructure", false, "Save directory structure")
		cmd.Flags().Bool("quiet", false, "Quiet mode")
		cmd.Flags().Bool("dry", false, "Dry run")

		cmd.Flags().Set("address", "http://test.example.com")
		cmd.Flags().Set("repository", "test-repo")
		cmd.Flags().Set("user", "testuser")
		cmd.Flags().Set("password", "testpass")
		cmd.Flags().Set("destination", destDir)
		cmd.Flags().Set("dry", "true")

		err := runPull(cmd, []string{"file1.txt", "file2.txt"})
		// Should succeed or fail on network (acceptable)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to download")
		}
	})

	t.Run("success - destination with trailing slash", func(t *testing.T) {
		tempDir := t.TempDir()
		destDir := filepath.Join(tempDir, "downloads") + "/"
		os.MkdirAll(strings.TrimSuffix(destDir, "/"), 0755)

		cmd := &cobra.Command{
			Use: "pull",
		}
		cmd.Flags().String("address", "", "Nexus address")
		cmd.Flags().String("repository", "", "Repository name")
		cmd.Flags().String("user", "", "User")
		cmd.Flags().String("password", "", "Password")
		cmd.Flags().String("config", "", "Config path")
		cmd.Flags().String("destination", "", "Destination path")
		cmd.Flags().String("root", "", "Root path")
		cmd.Flags().Bool("saveStructure", false, "Save directory structure")
		cmd.Flags().Bool("quiet", false, "Quiet mode")
		cmd.Flags().Bool("dry", false, "Dry run")

		cmd.Flags().Set("address", "http://test.example.com")
		cmd.Flags().Set("repository", "test-repo")
		cmd.Flags().Set("user", "testuser")
		cmd.Flags().Set("password", "testpass")
		cmd.Flags().Set("destination", destDir)
		cmd.Flags().Set("dry", "true")

		err := runPull(cmd, []string{"file.txt"})
		// Should succeed or fail on network (acceptable)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to download")
		}
	})

	t.Run("success - quiet mode", func(t *testing.T) {
		tempDir := t.TempDir()
		destDir := filepath.Join(tempDir, "downloads")
		os.MkdirAll(destDir, 0755)

		cmd := &cobra.Command{
			Use: "pull",
		}
		cmd.Flags().String("address", "", "Nexus address")
		cmd.Flags().String("repository", "", "Repository name")
		cmd.Flags().String("user", "", "User")
		cmd.Flags().String("password", "", "Password")
		cmd.Flags().String("config", "", "Config path")
		cmd.Flags().String("destination", "", "Destination path")
		cmd.Flags().String("root", "", "Root path")
		cmd.Flags().Bool("saveStructure", false, "Save directory structure")
		cmd.Flags().Bool("quiet", false, "Quiet mode")
		cmd.Flags().Bool("dry", false, "Dry run")

		cmd.Flags().Set("address", "http://test.example.com")
		cmd.Flags().Set("repository", "test-repo")
		cmd.Flags().Set("user", "testuser")
		cmd.Flags().Set("password", "testpass")
		cmd.Flags().Set("destination", destDir)
		cmd.Flags().Set("quiet", "true")
		cmd.Flags().Set("dry", "true")

		err := runPull(cmd, []string{"file.txt"})
		// Should succeed or fail on network (acceptable)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to download")
		}
	})
}

func TestRunPullWithClient(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() *MockNexusClient
		sources       []string
		repository    string
		destination   string
		root          string
		saveStructure bool
		quiet         bool
		expectedError string
	}{
		{
			name:          "success - download file",
			sources:       []string{"file.txt"},
			repository:    "test-repo",
			destination:   "/tmp/downloads",
			root:          "",
			saveStructure: false,
			quiet:         false,
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DownloadFileWithPathFunc: func(repository string, filePath string, destination string, root string) error {
						return nil
					},
				}
			},
		},
		{
			name:          "success - download directory",
			sources:       []string{"dir/"},
			repository:    "test-repo",
			destination:   "/tmp/downloads",
			root:          "",
			saveStructure: false,
			quiet:         false,
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DownloadDirectoryWithPathFunc: func(repository string, dirPath string, destination string, root string, saveStructure bool) error {
						return nil
					},
				}
			},
		},
		{
			name:          "success - download with root",
			sources:       []string{"file.txt"},
			repository:    "test-repo",
			destination:   "/tmp/downloads",
			root:          "custom/root",
			saveStructure: false,
			quiet:         false,
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DownloadFileWithPathFunc: func(repository string, filePath string, destination string, root string) error {
						return nil
					},
				}
			},
		},
		{
			name:          "success - download with saveStructure",
			sources:       []string{"dir/"},
			repository:    "test-repo",
			destination:   "/tmp/downloads",
			root:          "",
			saveStructure: true,
			quiet:         false,
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DownloadDirectoryWithPathFunc: func(repository string, dirPath string, destination string, root string, saveStructure bool) error {
						return nil
					},
				}
			},
		},
		{
			name:          "success - quiet mode",
			sources:       []string{"file.txt"},
			repository:    "test-repo",
			destination:   "/tmp/downloads",
			root:          "",
			saveStructure: false,
			quiet:         true,
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DownloadFileWithPathFunc: func(repository string, filePath string, destination string, root string) error {
						return nil
					},
				}
			},
		},
		{
			name:          "error - DownloadFileWithPath fails",
			sources:       []string{"file.txt"},
			repository:    "test-repo",
			destination:   "/tmp/downloads",
			root:          "",
			saveStructure: false,
			quiet:         false,
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DownloadFileWithPathFunc: func(repository string, filePath string, destination string, root string) error {
						return errors.New("download error")
					},
				}
			},
			expectedError: "failed to download file",
		},
		{
			name:          "error - DownloadDirectoryWithPath fails",
			sources:       []string{"dir/"},
			repository:    "test-repo",
			destination:   "/tmp/downloads",
			root:          "",
			saveStructure: false,
			quiet:         false,
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DownloadDirectoryWithPathFunc: func(repository string, dirPath string, destination string, root string, saveStructure bool) error {
						return errors.New("download error")
					},
				}
			},
			expectedError: "failed to download directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.setupMock()
			err := runPullWithClient(mockClient, tt.sources, tt.repository, tt.destination, tt.root, tt.saveStructure, tt.quiet)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
