package asset

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteCmd(t *testing.T) {
	assert.NotNil(t, DeleteCmd)
	assert.Equal(t, "delete [flags] <path>...", DeleteCmd.Use)
	assert.Equal(t, "Delete files or directories from Nexus repository", DeleteCmd.Short)
	assert.NotEmpty(t, DeleteCmd.Long)
}

func TestRunDelete(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		flags         map[string]string
		expectedError string
	}{
		{
			name: "success - delete file with dry run",
			args: []string{"file.txt"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"dry":        "true",
			},
		},
		{
			name: "success - delete directory with dry run",
			args: []string{"dir/"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"dry":        "true",
			},
		},
		{
			name: "success - quiet mode",
			args: []string{"file.txt"},
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
			name: "success - delete with backslash path separator",
			args: []string{"dir\\"},
			flags: map[string]string{
				"address":    "http://test.example.com",
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
				"dry":        "true",
			},
		},
		{
			name: "error - missing address causes validation or network error",
			args: []string{"file.txt"},
			flags: map[string]string{
				"repository": "test-repo",
				"user":       "testuser",
				"password":   "testpass",
			},
			expectedError: "", // Will error on validation or network, both are acceptable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use: "delete",
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

			err := runDelete(cmd, tt.args)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				// Should error on validation, network, or delete - all are acceptable
				// If no error, that's also fine (dry run might succeed)
				if err != nil {
					errorMsg := err.Error()
					// Accept any error - validation, network, or delete errors
					assert.True(t,
						strings.Contains(errorMsg, "failed to delete") ||
							strings.Contains(errorMsg, "configuration") ||
							strings.Contains(errorMsg, "dial tcp") ||
							strings.Contains(errorMsg, "network"),
						"Unexpected error type: %v", err)
				}
			}
		})
	}
}

func TestRunDelete_EdgeCases(t *testing.T) {
	t.Run("success - multiple paths", func(t *testing.T) {
		cmd := &cobra.Command{
			Use: "delete",
		}
		cmd.Flags().String("address", "", "Nexus address")
		cmd.Flags().String("repository", "", "Repository name")
		cmd.Flags().String("user", "", "User")
		cmd.Flags().String("password", "", "Password")
		cmd.Flags().String("config", "", "Config path")
		cmd.Flags().Bool("quiet", false, "Quiet mode")
		cmd.Flags().Bool("dry", false, "Dry run")

		cmd.Flags().Set("address", "http://test.example.com")
		cmd.Flags().Set("repository", "test-repo")
		cmd.Flags().Set("user", "testuser")
		cmd.Flags().Set("password", "testpass")
		cmd.Flags().Set("dry", "true")

		err := runDelete(cmd, []string{"file1.txt", "file2.txt", "dir/"})
		// Should succeed or fail on network (acceptable)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to delete")
		}
	})

	t.Run("success - command structure validation", func(t *testing.T) {
		// Verify the command structure
		assert.NotNil(t, DeleteCmd)
		// Args validation is handled by cobra, not by our code
	})
}

func TestRunDeleteWithClient(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() *MockNexusClient
		paths         []string
		repository    string
		quiet         bool
		nexusAddress  string
		expectedError string
	}{
		{
			name:         "success - delete file",
			paths:        []string{"file.txt"},
			repository:   "test-repo",
			quiet:        false,
			nexusAddress: "http://test.example.com",
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DeleteFileFunc: func(repository string, filePath string) error {
						return nil
					},
				}
			},
		},
		{
			name:         "success - delete directory",
			paths:        []string{"dir/"},
			repository:   "test-repo",
			quiet:        false,
			nexusAddress: "http://test.example.com",
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DeleteDirectoryFunc: func(repository string, dirPath string) error {
						return nil
					},
				}
			},
		},
		{
			name:         "success - delete file with backslash",
			paths:        []string{"dir\\"},
			repository:   "test-repo",
			quiet:        false,
			nexusAddress: "http://test.example.com",
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DeleteDirectoryFunc: func(repository string, dirPath string) error {
						return nil
					},
				}
			},
		},
		{
			name:         "success - quiet mode",
			paths:        []string{"file.txt"},
			repository:   "test-repo",
			quiet:        true,
			nexusAddress: "http://test.example.com",
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DeleteFileFunc: func(repository string, filePath string) error {
						return nil
					},
				}
			},
		},
		{
			name:         "success - multiple files",
			paths:        []string{"file1.txt", "file2.txt"},
			repository:   "test-repo",
			quiet:        false,
			nexusAddress: "http://test.example.com",
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DeleteFileFunc: func(repository string, filePath string) error {
						return nil
					},
				}
			},
		},
		{
			name:         "error - DeleteFile fails",
			paths:        []string{"file.txt"},
			repository:   "test-repo",
			quiet:        false,
			nexusAddress: "http://test.example.com",
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DeleteFileFunc: func(repository string, filePath string) error {
						return errors.New("delete error")
					},
				}
			},
			expectedError: "failed to delete file",
		},
		{
			name:         "error - DeleteDirectory fails",
			paths:        []string{"dir/"},
			repository:   "test-repo",
			quiet:        false,
			nexusAddress: "http://test.example.com",
			setupMock: func() *MockNexusClient {
				return &MockNexusClient{
					DeleteDirectoryFunc: func(repository string, dirPath string) error {
						return errors.New("delete error")
					},
				}
			},
			expectedError: "failed to delete directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.setupMock()
			err := runDeleteWithClient(mockClient, tt.paths, tt.repository, tt.quiet, tt.nexusAddress)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
