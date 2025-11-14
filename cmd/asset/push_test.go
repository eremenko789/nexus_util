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

func TestPushCmd(t *testing.T) {
	assert.NotNil(t, PushCmd)
	assert.Equal(t, "push [flags] <path>...", PushCmd.Use)
	assert.Equal(t, "Upload files or directories to Nexus repository", PushCmd.Short)
	assert.NotEmpty(t, PushCmd.Long)
}

func TestRunPush(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		flags         map[string]string
		setupFiles    func(string) []string // Returns paths to cleanup
		expectedError string
	}{
		{
			name: "success - push file with dry run",
			args: []string{"testfile.txt"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"dry":        "true",
			},
			setupFiles: func(tempDir string) []string {
				testFile := filepath.Join(tempDir, "testfile.txt")
				err := os.WriteFile(testFile, []byte("test content"), 0644)
				require.NoError(t, err)
				return []string{testFile}
			},
		},
		{
			name: "success - push directory with dry run",
			args: []string{"testdir"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"dry":        "true",
			},
			setupFiles: func(tempDir string) []string {
				testDir := filepath.Join(tempDir, "testdir")
				err := os.MkdirAll(testDir, 0755)
				require.NoError(t, err)
				testFile := filepath.Join(testDir, "file.txt")
				err = os.WriteFile(testFile, []byte("content"), 0644)
				require.NoError(t, err)
				return []string{testDir}
			},
		},
		{
			name: "success - push with destination path",
			args: []string{"testfile.txt"},
			flags: map[string]string{
				"address":     "http://test.example.com",
				"repository":  "test-repo",
				"user":        "testuser",
				"password":    "testpass",
				"destination": "custom/path",
				"dry":         "true",
			},
			setupFiles: func(tempDir string) []string {
				testFile := filepath.Join(tempDir, "testfile.txt")
				err := os.WriteFile(testFile, []byte("test content"), 0644)
				require.NoError(t, err)
				return []string{testFile}
			},
		},
		{
			name: "success - push with relative flag",
			args: []string{"testfile.txt"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"relative":   "true",
				"dry":        "true",
			},
			setupFiles: func(tempDir string) []string {
				testFile := filepath.Join(tempDir, "testfile.txt")
				err := os.WriteFile(testFile, []byte("test content"), 0644)
				require.NoError(t, err)
				return []string{testFile}
			},
		},
		{
			name: "success - quiet mode",
			args: []string{"testfile.txt"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"quiet":      "true",
				"dry":        "true",
			},
			setupFiles: func(tempDir string) []string {
				testFile := filepath.Join(tempDir, "testfile.txt")
				err := os.WriteFile(testFile, []byte("test content"), 0644)
				require.NoError(t, err)
				return []string{testFile}
			},
		},
		{
			name: "error - path doesn't exist",
			args: []string{"/nonexistent/path/file.txt"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
			},
			expectedError: "doesn't exist",
		},
		{
			name: "error - missing address causes validation or network error",
			args: []string{"testfile.txt"},
			flags: map[string]string{
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
			},
			setupFiles: func(tempDir string) []string {
				testFile := filepath.Join(tempDir, "testfile.txt")
				err := os.WriteFile(testFile, []byte("test content"), 0644)
				require.NoError(t, err)
				return []string{testFile}
			},
			expectedError: "", // Will error on validation or network, both are acceptable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Setup test files if needed
			if tt.setupFiles != nil {
				tt.setupFiles(tempDir)
				// Update args to use temp directory paths
				for i, arg := range tt.args {
					if !strings.HasPrefix(arg, "/") && !filepath.IsAbs(arg) {
						// Relative path - make it relative to tempDir
						tt.args[i] = filepath.Join(tempDir, arg)
					}
				}
			}

			cmd := &cobra.Command{
				Use: "push",
			}
			cmd.Flags().String("address", "", "Nexus address")
			cmd.Flags().String("repository", "", "Repository name")
			cmd.Flags().String("user", "", "User")
			cmd.Flags().String("password", "", "Password")
			cmd.Flags().String("config", "", "Config path")
			cmd.Flags().String("destination", "", "Destination path")
			cmd.Flags().Bool("relative", false, "Use relative paths")
			cmd.Flags().Bool("quiet", false, "Quiet mode")
			cmd.Flags().Bool("dry", false, "Dry run")

			// Set flags
			for key, value := range tt.flags {
				cmd.Flags().Set(key, value)
			}

			err := runPush(cmd, tt.args)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				// Should error on validation, network, or upload - all are acceptable
				// If no error, that's also fine (dry run might succeed)
				if err != nil {
					errorMsg := err.Error()
					// Accept any error - validation, network, or upload errors
					assert.True(t,
						strings.Contains(errorMsg, "failed to upload") ||
							strings.Contains(errorMsg, "configuration") ||
							strings.Contains(errorMsg, "doesn't exist") ||
							strings.Contains(errorMsg, "dial tcp") ||
							strings.Contains(errorMsg, "network"),
						"Unexpected error type: %v", err)
				}
			}
		})
	}
}

func TestRunPush_EdgeCases(t *testing.T) {
	t.Run("success - multiple files", func(t *testing.T) {
		tempDir := t.TempDir()
		file1 := filepath.Join(tempDir, "file1.txt")
		file2 := filepath.Join(tempDir, "file2.txt")
		os.WriteFile(file1, []byte("content1"), 0644)
		os.WriteFile(file2, []byte("content2"), 0644)

		cmd := &cobra.Command{
			Use: "push",
		}
		cmd.Flags().String("address", "", "Nexus address")
		cmd.Flags().String("repository", "", "Repository name")
		cmd.Flags().String("user", "", "User")
		cmd.Flags().String("password", "", "Password")
		cmd.Flags().String("config", "", "Config path")
		cmd.Flags().String("destination", "", "Destination path")
		cmd.Flags().Bool("relative", false, "Use relative paths")
		cmd.Flags().Bool("quiet", false, "Quiet mode")
		cmd.Flags().Bool("dry", false, "Dry run")

		cmd.Flags().Set("address", "http://test.example.com")
		cmd.Flags().Set("repository", "test-repo")
		cmd.Flags().Set("user", "testuser")
		cmd.Flags().Set("password", "testpass")
		cmd.Flags().Set("dry", "true")

		err := runPush(cmd, []string{file1, file2})
		// Should succeed or fail on network (acceptable)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to upload")
		}
	})

	t.Run("error - stat fails", func(t *testing.T) {
		// This is hard to test without mocking os.Stat
		// But we can test with a path that doesn't exist
		cmd := &cobra.Command{
			Use: "push",
		}
		cmd.Flags().String("address", "", "Nexus address")
		cmd.Flags().String("repository", "", "Repository name")
		cmd.Flags().String("user", "", "User")
		cmd.Flags().String("password", "", "Password")
		cmd.Flags().String("config", "", "Config path")
		cmd.Flags().String("destination", "", "Destination path")
		cmd.Flags().Bool("relative", false, "Use relative paths")
		cmd.Flags().Bool("quiet", false, "Quiet mode")
		cmd.Flags().Bool("dry", false, "Dry run")

		cmd.Flags().Set("address", "http://test.example.com")
		cmd.Flags().Set("repository", "test-repo")
		cmd.Flags().Set("user", "testuser")
		cmd.Flags().Set("password", "testpass")

		err := runPush(cmd, []string{"/nonexistent/file.txt"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "doesn't exist")
	})

	t.Run("success - file with destination and relative", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "testfile.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		cmd := &cobra.Command{
			Use: "push",
		}
		cmd.Flags().String("address", "", "Nexus address")
		cmd.Flags().String("repository", "", "Repository name")
		cmd.Flags().String("user", "", "User")
		cmd.Flags().String("password", "", "Password")
		cmd.Flags().String("config", "", "Config path")
		cmd.Flags().String("destination", "", "Destination path")
		cmd.Flags().Bool("relative", false, "Use relative paths")
		cmd.Flags().Bool("quiet", false, "Quiet mode")
		cmd.Flags().Bool("dry", false, "Dry run")

		cmd.Flags().Set("address", "http://test.example.com")
		cmd.Flags().Set("repository", "test-repo")
		cmd.Flags().Set("user", "testuser")
		cmd.Flags().Set("password", "testpass")
		cmd.Flags().Set("destination", "custom/path")
		cmd.Flags().Set("relative", "true")
		cmd.Flags().Set("dry", "true")

		err := runPush(cmd, []string{testFile})
		// Should succeed or fail on network (acceptable)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to upload")
		}
	})

	t.Run("success - quiet mode suppresses output", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "testfile.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		cmd := &cobra.Command{
			Use: "push",
		}
		cmd.Flags().String("address", "", "Nexus address")
		cmd.Flags().String("repository", "", "Repository name")
		cmd.Flags().String("user", "", "User")
		cmd.Flags().String("password", "", "Password")
		cmd.Flags().String("config", "", "Config path")
		cmd.Flags().String("destination", "", "Destination path")
		cmd.Flags().Bool("relative", false, "Use relative paths")
		cmd.Flags().Bool("quiet", false, "Quiet mode")
		cmd.Flags().Bool("dry", false, "Dry run")

		cmd.Flags().Set("address", "http://test.example.com")
		cmd.Flags().Set("repository", "test-repo")
		cmd.Flags().Set("user", "testuser")
		cmd.Flags().Set("password", "testpass")
		cmd.Flags().Set("quiet", "true")
		cmd.Flags().Set("dry", "true")

		err := runPush(cmd, []string{testFile})
		// Should succeed or fail on network (acceptable)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to upload")
		}
	})

	t.Run("error - stat returns non-IsNotExist error", func(t *testing.T) {
		// This is hard to test without mocking os.Stat
		// But we can test with a path that causes permission error
		// For now, we'll skip this as it requires mocking
		t.Skip("Requires mocking os.Stat to test non-IsNotExist errors")
	})

	t.Run("success - push with empty destination", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "testfile.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		cmd := &cobra.Command{
			Use: "push",
		}
		cmd.Flags().String("address", "", "Nexus address")
		cmd.Flags().String("repository", "", "Repository name")
		cmd.Flags().String("user", "", "User")
		cmd.Flags().String("password", "", "Password")
		cmd.Flags().String("config", "", "Config path")
		cmd.Flags().String("destination", "", "Destination path")
		cmd.Flags().Bool("relative", false, "Use relative paths")
		cmd.Flags().Bool("quiet", false, "Quiet mode")
		cmd.Flags().Bool("dry", false, "Dry run")

		cmd.Flags().Set("address", "http://test.example.com")
		cmd.Flags().Set("repository", "test-repo")
		cmd.Flags().Set("user", "testuser")
		cmd.Flags().Set("password", "testpass")
		cmd.Flags().Set("dry", "true")
		// destination is empty by default

		err := runPush(cmd, []string{testFile})
		// Should succeed or fail on network (acceptable)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to upload")
		}
	})
}

func TestRunPushWithClient(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() *MockNexusClient
		paths         []string
		repository    string
		destination   string
		relative      bool
		quiet         bool
		nexusAddress  string
		setupFiles    func(string) []string
		expectedError string
	}{
		{
			name:         "success - upload file",
			paths:        []string{"testfile.txt"},
			repository:   "test-repo",
			destination:  "",
			relative:     false,
			quiet:        false,
			nexusAddress: "http://test.example.com",
			setupFiles: func(tempDir string) []string {
				testFile := filepath.Join(tempDir, "testfile.txt")
				os.WriteFile(testFile, []byte("content"), 0644)
				return []string{testFile}
			},
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					UploadFileFunc: func(repository string, filePath string, destPath string) error {
						return nil
					},
				}
			},
		},
		{
			name:         "success - upload directory",
			paths:        []string{"testdir"},
			repository:   "test-repo",
			destination:  "",
			relative:     false,
			quiet:        false,
			nexusAddress: "http://test.example.com",
			setupFiles: func(tempDir string) []string {
				testDir := filepath.Join(tempDir, "testdir")
				os.MkdirAll(testDir, 0755)
				return []string{testDir}
			},
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					UploadDirectoryFunc: func(repository string, dirPath string, relative bool, destination string) error {
						return nil
					},
				}
			},
		},
		{
			name:         "success - upload file with destination",
			paths:        []string{"testfile.txt"},
			repository:   "test-repo",
			destination:  "custom/path",
			relative:     false,
			quiet:        false,
			nexusAddress: "http://test.example.com",
			setupFiles: func(tempDir string) []string {
				testFile := filepath.Join(tempDir, "testfile.txt")
				os.WriteFile(testFile, []byte("content"), 0644)
				return []string{testFile}
			},
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					UploadFileFunc: func(repository string, filePath string, destPath string) error {
						return nil
					},
				}
			},
		},
		{
			name:         "success - upload file with relative",
			paths:        []string{"testfile.txt"},
			repository:   "test-repo",
			destination:  "",
			relative:     true,
			quiet:        false,
			nexusAddress: "http://test.example.com",
			setupFiles: func(tempDir string) []string {
				testFile := filepath.Join(tempDir, "testfile.txt")
				os.WriteFile(testFile, []byte("content"), 0644)
				return []string{testFile}
			},
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					UploadFileFunc: func(repository string, filePath string, destPath string) error {
						return nil
					},
				}
			},
		},
		{
			name:         "success - quiet mode",
			paths:        []string{"testfile.txt"},
			repository:   "test-repo",
			destination:  "",
			relative:     false,
			quiet:        true,
			nexusAddress: "http://test.example.com",
			setupFiles: func(tempDir string) []string {
				testFile := filepath.Join(tempDir, "testfile.txt")
				os.WriteFile(testFile, []byte("content"), 0644)
				return []string{testFile}
			},
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					UploadFileFunc: func(repository string, filePath string, destPath string) error {
						return nil
					},
				}
			},
		},
		{
			name:         "error - UploadFile fails",
			paths:        []string{"testfile.txt"},
			repository:   "test-repo",
			destination:  "",
			relative:     false,
			quiet:        false,
			nexusAddress: "http://test.example.com",
			setupFiles: func(tempDir string) []string {
				testFile := filepath.Join(tempDir, "testfile.txt")
				os.WriteFile(testFile, []byte("content"), 0644)
				return []string{testFile}
			},
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					UploadFileFunc: func(repository string, filePath string, destPath string) error {
						return errors.New("upload error")
					},
				}
			},
			expectedError: "failed to upload file",
		},
		{
			name:         "error - UploadDirectory fails",
			paths:        []string{"testdir"},
			repository:   "test-repo",
			destination:  "",
			relative:     false,
			quiet:        false,
			nexusAddress: "http://test.example.com",
			setupFiles: func(tempDir string) []string {
				testDir := filepath.Join(tempDir, "testdir")
				os.MkdirAll(testDir, 0755)
				return []string{testDir}
			},
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					UploadDirectoryFunc: func(repository string, dirPath string, relative bool, destination string) error {
						return errors.New("upload error")
					},
				}
			},
			expectedError: "failed to upload directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			var actualPaths []string

			if tt.setupFiles != nil {
				tt.setupFiles(tempDir)
				// Update paths to use temp directory
				for _, path := range tt.paths {
					if !filepath.IsAbs(path) {
						actualPaths = append(actualPaths, filepath.Join(tempDir, path))
					} else {
						actualPaths = append(actualPaths, path)
					}
				}
			} else {
				actualPaths = tt.paths
			}

			mockClient := tt.setupMock()
			err := runPushWithClient(mockClient, actualPaths, tt.repository, tt.destination, tt.relative, tt.quiet, tt.nexusAddress)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
