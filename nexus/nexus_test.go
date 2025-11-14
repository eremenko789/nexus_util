package nexus

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockHTTPClient is a mock implementation of HTTPClient for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, errors.New("mock HTTP client: DoFunc not set")
}

// MockFileSystem is a mock implementation of FileSystem for testing
type MockFileSystem struct {
	OpenFunc      func(name string) (File, error)
	CreateFunc    func(name string) (File, error)
	StatFunc      func(name string) (os.FileInfo, error)
	MkdirAllFunc  func(path string, perm os.FileMode) error
	ReadFileFunc  func(name string) ([]byte, error)
	WriteFileFunc func(name string, data []byte, perm os.FileMode) error
	WalkFunc      func(root string, walkFn filepath.WalkFunc) error
}

func (m *MockFileSystem) Open(name string) (File, error) {
	if m.OpenFunc != nil {
		return m.OpenFunc(name)
	}
	return nil, errors.New("mock file system: OpenFunc not set")
}

func (m *MockFileSystem) Create(name string) (File, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(name)
	}
	return nil, errors.New("mock file system: CreateFunc not set")
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.StatFunc != nil {
		return m.StatFunc(name)
	}
	return nil, errors.New("mock file system: StatFunc not set")
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if m.MkdirAllFunc != nil {
		return m.MkdirAllFunc(path, perm)
	}
	return errors.New("mock file system: MkdirAllFunc not set")
}

func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(name)
	}
	return nil, errors.New("mock file system: ReadFileFunc not set")
}

func (m *MockFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	if m.WriteFileFunc != nil {
		return m.WriteFileFunc(name, data, perm)
	}
	return errors.New("mock file system: WriteFileFunc not set")
}

func (m *MockFileSystem) Walk(root string, walkFn filepath.WalkFunc) error {
	if m.WalkFunc != nil {
		return m.WalkFunc(root, walkFn)
	}
	return errors.New("mock file system: WalkFunc not set")
}

// MockFile is a mock implementation of File for testing
type MockFile struct {
	io.Reader
	io.Writer
	io.Closer
	StatFunc func() (os.FileInfo, error)
}

func (m *MockFile) Stat() (os.FileInfo, error) {
	if m.StatFunc != nil {
		return m.StatFunc()
	}
	return nil, errors.New("mock file: StatFunc not set")
}

// errorReader is a reader that always returns an error
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

// errorWriter is a writer that always returns an error
type errorWriter struct {
	err error
}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, e.err
}

func TestNewNexusClient(t *testing.T) {
	t.Run("creates client with default dependencies", func(t *testing.T) {
		client := NewNexusClient("http://test.example.com", "user", "pass", false, false)

		assert.NotNil(t, client)
		assert.Equal(t, "http://test.example.com", client.BaseURL)
		assert.Equal(t, "user", client.Username)
		assert.Equal(t, "pass", client.Password)
		assert.False(t, client.Quiet)
		assert.False(t, client.DryRun)
		assert.NotNil(t, client.HTTPClient)
		assert.NotNil(t, client.FileSystem)
	})

	t.Run("removes trailing slash from baseURL", func(t *testing.T) {
		client := NewNexusClient("http://test.example.com/", "user", "pass", false, false)
		assert.Equal(t, "http://test.example.com", client.BaseURL)
	})

	t.Run("creates client with custom dependencies", func(t *testing.T) {
		mockHTTP := &MockHTTPClient{}
		mockFS := &MockFileSystem{}

		client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", false, false, mockHTTP, mockFS)

		assert.NotNil(t, client)
		assert.Equal(t, mockHTTP, client.HTTPClient)
		assert.Equal(t, mockFS, client.FileSystem)
	})

	t.Run("removes backslash from baseURL", func(t *testing.T) {
		client := NewNexusClient("http://test.example.com\\", "user", "pass", false, false)
		assert.Equal(t, "http://test.example.com", client.BaseURL)
	})

	t.Run("creates client with nil httpClient uses default", func(t *testing.T) {
		mockFS := &MockFileSystem{}
		client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", false, false, nil, mockFS)
		assert.NotNil(t, client)
		assert.NotNil(t, client.HTTPClient)
		assert.Equal(t, mockFS, client.FileSystem)
	})

	t.Run("creates client with nil fileSystem uses default", func(t *testing.T) {
		mockHTTP := &MockHTTPClient{}
		client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", false, false, mockHTTP, nil)
		assert.NotNil(t, client)
		assert.Equal(t, mockHTTP, client.HTTPClient)
		assert.NotNil(t, client.FileSystem)
	})
}

func TestNexusClient_Logf(t *testing.T) {
	t.Run("logs message when not quiet", func(t *testing.T) {
		client := NewNexusClient("http://test.example.com", "user", "pass", false, false)
		// This is hard to test without capturing stdout, so we just verify it doesn't panic
		client.Logf("test message")
	})

	t.Run("does not log when quiet", func(t *testing.T) {
		client := NewNexusClient("http://test.example.com", "user", "pass", true, false)
		// This is hard to test without capturing stdout, so we just verify it doesn't panic
		client.Logf("test message")
	})

	t.Run("logs formatted message", func(t *testing.T) {
		client := NewNexusClient("http://test.example.com", "user", "pass", false, false)
		client.Logf("test message %s %d", "arg", 123)
	})
}

func TestNexusClient_makeRequest(t *testing.T) {
	t.Run("success - makes request without auth", func(t *testing.T) {
		mockHTTP := &MockHTTPClient{}
		mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: httpStatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			}, nil
		}

		client := NewNexusClientWithDeps("http://test.example.com", "", "", false, false, mockHTTP, nil)
		resp, err := client.makeRequest("GET", "http://test.example.com/test", nil)
		require.NoError(t, err)
		assert.Equal(t, httpStatusOK, resp.StatusCode)
	})

	t.Run("success - makes request with auth", func(t *testing.T) {
		mockHTTP := &MockHTTPClient{}
		mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
			// Verify auth header is set
			user, pass, ok := req.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "user", user)
			assert.Equal(t, "pass", pass)
			return &http.Response{
				StatusCode: httpStatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			}, nil
		}

		client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", false, false, mockHTTP, nil)
		resp, err := client.makeRequest("GET", "http://test.example.com/test", nil)
		require.NoError(t, err)
		assert.Equal(t, httpStatusOK, resp.StatusCode)
	})

	t.Run("error - NewRequestWithContext fails", func(t *testing.T) {
		// This is hard to test without mocking http.NewRequestWithContext
		// But we can test with invalid URL
		client := NewNexusClient("http://test.example.com", "user", "pass", false, false)
		// Invalid URL should cause error
		_, err := client.makeRequest("GET", "://invalid-url", nil)
		require.Error(t, err)
	})

	t.Run("error - HTTP client Do fails", func(t *testing.T) {
		mockHTTP := &MockHTTPClient{}
		mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		}

		client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", false, false, mockHTTP, nil)
		_, err := client.makeRequest("GET", "http://test.example.com/test", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "network error")
	})
}

func TestNexusClient_GetFilesInDirectory(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		dirPath       string
		mockResponse  SearchAssetsResponse
		statusCode    int
		continuation  string
		expectedFiles []string
		expectedError string
		setupMock     func(*MockHTTPClient, SearchAssetsResponse, int, string)
	}{
		{
			name:       "success - single page",
			repository: "test-repo",
			dirPath:    "test-dir",
			mockResponse: SearchAssetsResponse{
				Items: []Asset{
					{Path: "test-dir/file1.txt"},
					{Path: "test-dir/file2.txt"},
				},
				ContinuationToken: "",
			},
			statusCode:    httpStatusOK,
			expectedFiles: []string{"test-dir/file1.txt", "test-dir/file2.txt"},
		},
		{
			name:       "success - multiple pages",
			repository: "test-repo",
			dirPath:    "test-dir",
			mockResponse: SearchAssetsResponse{
				Items: []Asset{
					{Path: "test-dir/file1.txt"},
				},
				ContinuationToken: "token123",
			},
			statusCode:   httpStatusOK,
			continuation: "token123",
			setupMock: func(mock *MockHTTPClient, resp SearchAssetsResponse, statusCode int, continuation string) {
				callCount := 0
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					callCount++
					if callCount == 1 {
						body, _ := json.Marshal(resp)
						return &http.Response{
							StatusCode: statusCode,
							Body:       io.NopCloser(bytes.NewReader(body)),
						}, nil
					}
					// Second call with continuation token
					body, _ := json.Marshal(SearchAssetsResponse{
						Items:             []Asset{{Path: "test-dir/file2.txt"}},
						ContinuationToken: "",
					})
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader(body)),
					}, nil
				}
			},
			expectedFiles: []string{"test-dir/file1.txt", "test-dir/file2.txt"},
		},
		{
			name:       "success - root directory",
			repository: "test-repo",
			dirPath:    "",
			mockResponse: SearchAssetsResponse{
				Items: []Asset{
					{Path: "file1.txt"},
					{Path: "file2.txt"},
				},
				ContinuationToken: "",
			},
			statusCode:    httpStatusOK,
			expectedFiles: []string{"file1.txt", "file2.txt"},
		},
		{
			name:          "error - HTTP request fails",
			repository:    "test-repo",
			dirPath:       "test-dir",
			expectedError: "failed to search assets",
			setupMock: func(mock *MockHTTPClient, resp SearchAssetsResponse, statusCode int, continuation string) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				}
			},
		},
		{
			name:          "error - non-200 status code",
			repository:    "test-repo",
			dirPath:       "test-dir",
			statusCode:    500,
			expectedError: "search request failed with status 500",
			setupMock: func(mock *MockHTTPClient, resp SearchAssetsResponse, statusCode int, continuation string) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			},
		},
		{
			name:          "error - invalid JSON response",
			repository:    "test-repo",
			dirPath:       "test-dir",
			statusCode:    httpStatusOK,
			expectedError: "failed to decode search response",
			setupMock: func(mock *MockHTTPClient, resp SearchAssetsResponse, statusCode int, continuation string) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte("invalid json"))),
					}, nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockHTTP, tt.mockResponse, tt.statusCode, tt.continuation)
			} else {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					body, _ := json.Marshal(tt.mockResponse)
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewReader(body)),
					}, nil
				}
			}

			client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, false, mockHTTP, &RealFileSystem{})

			files, err := client.GetFilesInDirectory(tt.repository, tt.dirPath)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, files)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedFiles, files)
			}
		})
	}
}

func TestNexusClient_DeleteFile(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		filePath      string
		dryRun        bool
		statusCode    int
		expectedError string
		setupMock     func(*MockHTTPClient, int)
	}{
		{
			name:       "success - file deleted",
			repository: "test-repo",
			filePath:   "test-file.txt",
			dryRun:     false,
			statusCode: httpStatusNoContent,
		},
		{
			name:       "success - dry run",
			repository: "test-repo",
			filePath:   "test-file.txt",
			dryRun:     true,
		},
		{
			name:       "success - file not found (404)",
			repository: "test-repo",
			filePath:   "test-file.txt",
			dryRun:     false,
			statusCode: httpStatusNotFound,
		},
		{
			name:          "error - HTTP request fails",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			dryRun:        false,
			expectedError: "failed to delete file",
			setupMock: func(mock *MockHTTPClient, statusCode int) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				}
			},
		},
		{
			name:          "error - unexpected status code",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			dryRun:        false,
			statusCode:    500,
			expectedError: "unexpected response code 500",
			setupMock: func(mock *MockHTTPClient, statusCode int) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockHTTP, tt.statusCode)
			} else {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			}

			client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, tt.dryRun, mockHTTP, &RealFileSystem{})

			err := client.DeleteFile(tt.repository, tt.filePath)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNexusClient_DeleteDirectory(t *testing.T) {
	t.Run("success - deletes all files in directory", func(t *testing.T) {
		mockHTTP := &MockHTTPClient{}
		mockFS := &MockFileSystem{}

		// Mock GetFilesInDirectory
		mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "search/assets") {
				body, _ := json.Marshal(SearchAssetsResponse{
					Items: []Asset{
						{Path: "test-dir/file1.txt"},
						{Path: "test-dir/file2.txt"},
					},
					ContinuationToken: "",
				})
				return &http.Response{
					StatusCode: httpStatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			}
			// Mock DeleteFile calls
			return &http.Response{
				StatusCode: httpStatusNoContent,
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			}, nil
		}

		client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, false, mockHTTP, mockFS)

		err := client.DeleteDirectory("test-repo", "test-dir")
		require.NoError(t, err)
	})

	t.Run("success - empty directory", func(t *testing.T) {
		mockHTTP := &MockHTTPClient{}
		mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
			body, _ := json.Marshal(SearchAssetsResponse{
				Items:             []Asset{},
				ContinuationToken: "",
			})
			return &http.Response{
				StatusCode: httpStatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
		}

		client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, false, mockHTTP, &RealFileSystem{})

		err := client.DeleteDirectory("test-repo", "test-dir")
		require.NoError(t, err)
	})

	t.Run("error - GetFilesInDirectory fails", func(t *testing.T) {
		mockHTTP := &MockHTTPClient{}
		mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		}

		client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, false, mockHTTP, &RealFileSystem{})

		err := client.DeleteDirectory("test-repo", "test-dir")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get files in directory")
	})
}

func TestNexusClient_DownloadFile(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		filePath      string
		destPath      string
		dryRun        bool
		statusCode    int
		fileContent   string
		expectedError string
		setupMocks    func(*MockHTTPClient, *MockFileSystem, int, string)
	}{
		{
			name:        "success - downloads file",
			repository:  "test-repo",
			filePath:    "test-file.txt",
			destPath:    "/tmp/test-file.txt",
			dryRun:      false,
			statusCode:  httpStatusOK,
			fileContent: "test content",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem, statusCode int, content string) {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte(content))),
					}, nil
				}
				mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				mockFS.CreateFunc = func(name string) (File, error) {
					return &MockFile{
						Writer: &bytes.Buffer{},
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
			},
		},
		{
			name:       "success - dry run",
			repository: "test-repo",
			filePath:   "test-file.txt",
			destPath:   "/tmp/test-file.txt",
			dryRun:     true,
		},
		{
			name:          "error - HTTP request fails",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			destPath:      "/tmp/test-file.txt",
			dryRun:        false,
			expectedError: "failed to download file",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem, statusCode int, content string) {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				}
				mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
			},
		},
		{
			name:          "error - non-200 status code",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			destPath:      "/tmp/test-file.txt",
			dryRun:        false,
			statusCode:    404,
			expectedError: "download failed with status 404",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem, statusCode int, content string) {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
				mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
			},
		},
		{
			name:          "error - cannot create directory",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			destPath:      "/tmp/test-file.txt",
			dryRun:        false,
			expectedError: "failed to create destination directory",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem, statusCode int, content string) {
				mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
					return errors.New("permission denied")
				}
			},
		},
		{
			name:          "error - cannot create file",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			destPath:      "/tmp/test-file.txt",
			dryRun:        false,
			statusCode:    httpStatusOK,
			expectedError: "failed to create destination file",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem, statusCode int, content string) {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte(content))),
					}, nil
				}
				mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				mockFS.CreateFunc = func(name string) (File, error) {
					return nil, errors.New("permission denied")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{}
			mockFS := &MockFileSystem{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockHTTP, mockFS, tt.statusCode, tt.fileContent)
			}

			client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, tt.dryRun, mockHTTP, mockFS)

			err := client.DownloadFile(tt.repository, tt.filePath, tt.destPath)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNexusClient_UploadFile(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		filePath      string
		destPath      string
		dryRun        bool
		fileContent   string
		statusCode    int
		expectedError string
		setupMocks    func(*MockHTTPClient, *MockFileSystem, string, int)
	}{
		{
			name:        "success - uploads file",
			repository:  "test-repo",
			filePath:    "/tmp/test-file.txt",
			destPath:    "test-file.txt",
			dryRun:      false,
			fileContent: "test content",
			statusCode:  httpStatusOK,
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem, content string, statusCode int) {
				mockFS.OpenFunc = func(name string) (File, error) {
					return &MockFile{
						Reader: bytes.NewReader([]byte(content)),
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			},
		},
		{
			name:       "success - dry run",
			repository: "test-repo",
			filePath:   "/tmp/test-file.txt",
			destPath:   "test-file.txt",
			dryRun:     true,
		},
		{
			name:          "error - cannot open file",
			repository:    "test-repo",
			filePath:      "/tmp/test-file.txt",
			destPath:      "test-file.txt",
			dryRun:        false,
			expectedError: "failed to open file",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem, content string, statusCode int) {
				mockFS.OpenFunc = func(name string) (File, error) {
					return nil, errors.New("file not found")
				}
			},
		},
		{
			name:          "error - HTTP request fails",
			repository:    "test-repo",
			filePath:      "/tmp/test-file.txt",
			destPath:      "test-file.txt",
			dryRun:        false,
			fileContent:   "test content",
			expectedError: "failed to upload file",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem, content string, statusCode int) {
				mockFS.OpenFunc = func(name string) (File, error) {
					return &MockFile{
						Reader: bytes.NewReader([]byte(content)),
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				}
			},
		},
		{
			name:          "error - non-2xx status code",
			repository:    "test-repo",
			filePath:      "/tmp/test-file.txt",
			destPath:      "test-file.txt",
			dryRun:        false,
			fileContent:   "test content",
			statusCode:    500,
			expectedError: "upload failed with status 500",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem, content string, statusCode int) {
				mockFS.OpenFunc = func(name string) (File, error) {
					return &MockFile{
						Reader: bytes.NewReader([]byte(content)),
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			},
		},
		{
			name:          "error - ReadAll fails",
			repository:    "test-repo",
			filePath:      "/tmp/test-file.txt",
			destPath:      "test-file.txt",
			dryRun:        false,
			expectedError: "failed to read file",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem, content string, statusCode int) {
				mockFS.OpenFunc = func(name string) (File, error) {
					// Return a file that will fail on ReadAll
					return &MockFile{
						Reader: &errorReader{err: errors.New("read error")},
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
			},
		},
		{
			name:        "success - status code 201 (created)",
			repository:  "test-repo",
			filePath:    "/tmp/test-file.txt",
			destPath:    "test-file.txt",
			dryRun:      false,
			fileContent: "test content",
			statusCode:  201,
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem, content string, statusCode int) {
				mockFS.OpenFunc = func(name string) (File, error) {
					return &MockFile{
						Reader: bytes.NewReader([]byte(content)),
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			},
		},
		{
			name:        "success - status code 299",
			repository:  "test-repo",
			filePath:    "/tmp/test-file.txt",
			destPath:    "test-file.txt",
			dryRun:      false,
			fileContent: "test content",
			statusCode:  299,
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem, content string, statusCode int) {
				mockFS.OpenFunc = func(name string) (File, error) {
					return &MockFile{
						Reader: bytes.NewReader([]byte(content)),
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{}
			mockFS := &MockFileSystem{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockHTTP, mockFS, tt.fileContent, tt.statusCode)
			}

			client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, tt.dryRun, mockHTTP, mockFS)

			err := client.UploadFile(tt.repository, tt.filePath, tt.destPath)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNexusClient_ListRepositories(t *testing.T) {
	tests := []struct {
		name          string
		dryRun        bool
		statusCode    int
		repositories  []Repository
		expectedError string
		setupMock     func(*MockHTTPClient, []Repository, int)
	}{
		{
			name:       "success - lists repositories",
			dryRun:     false,
			statusCode: httpStatusOK,
			repositories: []Repository{
				{Name: "repo1", Format: "raw", Type: "hosted", URL: "http://test.example.com/repository/repo1"},
				{Name: "repo2", Format: "raw", Type: "hosted", URL: "http://test.example.com/repository/repo2"},
			},
		},
		{
			name:         "success - dry run",
			dryRun:       true,
			repositories: []Repository{},
		},
		{
			name:          "error - HTTP request fails",
			dryRun:        false,
			expectedError: "failed to list repositories",
			setupMock: func(mock *MockHTTPClient, repos []Repository, statusCode int) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				}
			},
		},
		{
			name:          "error - non-200 status code",
			dryRun:        false,
			statusCode:    500,
			expectedError: "repositories request failed with status 500",
			setupMock: func(mock *MockHTTPClient, repos []Repository, statusCode int) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			},
		},
		{
			name:          "error - invalid JSON",
			dryRun:        false,
			statusCode:    httpStatusOK,
			expectedError: "failed to decode repositories response",
			setupMock: func(mock *MockHTTPClient, repos []Repository, statusCode int) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte("invalid json"))),
					}, nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockHTTP, tt.repositories, tt.statusCode)
			} else {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					body, _ := json.Marshal(tt.repositories)
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewReader(body)),
					}, nil
				}
			}

			client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, tt.dryRun, mockHTTP, &RealFileSystem{})

			repos, err := client.ListRepositories()

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, repos)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.repositories, repos)
			}
		})
	}
}

func TestNexusClient_FileExists(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		filePath      string
		statusCode    int
		expected      bool
		expectedError string
		setupMock     func(*MockHTTPClient, int)
	}{
		{
			name:       "success - file exists",
			repository: "test-repo",
			filePath:   "test-file.txt",
			statusCode: httpStatusOK,
			expected:   true,
		},
		{
			name:       "success - file does not exist",
			repository: "test-repo",
			filePath:   "test-file.txt",
			statusCode: httpStatusNotFound,
			expected:   false,
		},
		{
			name:          "error - HTTP request fails",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			expectedError: "failed to check file existence",
			setupMock: func(mock *MockHTTPClient, statusCode int) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockHTTP, tt.statusCode)
			} else {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			}

			client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, false, mockHTTP, &RealFileSystem{})

			exists, err := client.FileExists(tt.repository, tt.filePath)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, exists)
			}
		})
	}
}

func TestNexusClient_GetFileSize(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		filePath      string
		statusCode    int
		contentLength int64
		expected      int64
		expectedError string
		setupMock     func(*MockHTTPClient, int, int64)
	}{
		{
			name:          "success - gets file size",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			statusCode:    httpStatusOK,
			contentLength: 1024,
			expected:      1024,
		},
		{
			name:          "error - file not found",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			statusCode:    httpStatusNotFound,
			expectedError: "file not found",
			setupMock: func(mock *MockHTTPClient, statusCode int, contentLength int64) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			},
		},
		{
			name:          "error - invalid content length",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			statusCode:    httpStatusOK,
			contentLength: -1,
			expectedError: "invalid content length",
			setupMock: func(mock *MockHTTPClient, statusCode int, contentLength int64) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}
					resp.ContentLength = contentLength
					return resp, nil
				}
			},
		},
		{
			name:          "error - HTTP request fails",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			expectedError: "failed to get file size",
			setupMock: func(mock *MockHTTPClient, statusCode int, contentLength int64) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockHTTP, tt.statusCode, tt.contentLength)
			} else {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}
					resp.ContentLength = tt.contentLength
					return resp, nil
				}
			}

			client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, false, mockHTTP, &RealFileSystem{})

			size, err := client.GetFileSize(tt.repository, tt.filePath)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, size)
			}
		})
	}
}

func TestNexusClient_DownloadToBuffer(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		filePath      string
		dryRun        bool
		statusCode    int
		content       string
		expected      []byte
		expectedError string
		setupMock     func(*MockHTTPClient, int, string)
	}{
		{
			name:       "success - downloads to buffer",
			repository: "test-repo",
			filePath:   "test-file.txt",
			dryRun:     false,
			statusCode: httpStatusOK,
			content:    "test content",
			expected:   []byte("test content"),
		},
		{
			name:       "success - dry run",
			repository: "test-repo",
			filePath:   "test-file.txt",
			dryRun:     true,
			expected:   nil,
		},
		{
			name:          "error - HTTP request fails",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			dryRun:        false,
			expectedError: "failed to download file",
			setupMock: func(mock *MockHTTPClient, statusCode int, content string) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				}
			},
		},
		{
			name:          "error - non-200 status code",
			repository:    "test-repo",
			filePath:      "test-file.txt",
			dryRun:        false,
			statusCode:    404,
			expectedError: "download failed with status 404",
			setupMock: func(mock *MockHTTPClient, statusCode int, content string) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockHTTP, tt.statusCode, tt.content)
			} else {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte(tt.content))),
					}, nil
				}
			}

			client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, tt.dryRun, mockHTTP, &RealFileSystem{})

			data, err := client.DownloadToBuffer(tt.repository, tt.filePath)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, data)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, data)
			}
		})
	}
}

func TestNexusClient_UploadFromBuffer(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		destPath      string
		content       []byte
		dryRun        bool
		statusCode    int
		expectedError string
		setupMock     func(*MockHTTPClient, int)
	}{
		{
			name:       "success - uploads from buffer",
			repository: "test-repo",
			destPath:   "test-file.txt",
			content:    []byte("test content"),
			dryRun:     false,
			statusCode: httpStatusOK,
		},
		{
			name:       "success - dry run",
			repository: "test-repo",
			destPath:   "test-file.txt",
			content:    []byte("test content"),
			dryRun:     true,
		},
		{
			name:          "error - HTTP request fails",
			repository:    "test-repo",
			destPath:      "test-file.txt",
			content:       []byte("test content"),
			dryRun:        false,
			expectedError: "failed to upload file",
			setupMock: func(mock *MockHTTPClient, statusCode int) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				}
			},
		},
		{
			name:          "error - non-2xx status code",
			repository:    "test-repo",
			destPath:      "test-file.txt",
			content:       []byte("test content"),
			dryRun:        false,
			statusCode:    500,
			expectedError: "upload failed with status 500",
			setupMock: func(mock *MockHTTPClient, statusCode int) {
				mock.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockHTTP, tt.statusCode)
			} else {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			}

			client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, tt.dryRun, mockHTTP, &RealFileSystem{})

			err := client.UploadFromBuffer(tt.repository, tt.destPath, tt.content)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNexusClient_TransferFile(t *testing.T) {
	t.Run("success - transfers file between servers", func(t *testing.T) {
		sourceHTTP := &MockHTTPClient{}
		targetHTTP := &MockHTTPClient{}

		// Mock source download
		sourceHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "download") {
				return &http.Response{
					StatusCode: httpStatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte("test content"))),
				}, nil
			}
			return nil, errors.New("unexpected request")
		}

		// Mock target upload
		targetHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "repository") {
				return &http.Response{
					StatusCode: httpStatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}, nil
			}
			return nil, errors.New("unexpected request")
		}

		sourceClient := NewNexusClientWithDeps("http://source.example.com", "user", "pass", true, false, sourceHTTP, &RealFileSystem{})
		targetClient := NewNexusClientWithDeps("http://target.example.com", "user", "pass", true, false, targetHTTP, &RealFileSystem{})

		err := sourceClient.TransferFile(targetClient, "source-repo", "target-repo", "test-file.txt", false)
		require.NoError(t, err)
	})

	t.Run("error - download fails", func(t *testing.T) {
		sourceHTTP := &MockHTTPClient{}
		targetHTTP := &MockHTTPClient{}

		sourceHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		}

		sourceClient := NewNexusClientWithDeps("http://source.example.com", "user", "pass", true, false, sourceHTTP, &RealFileSystem{})
		targetClient := NewNexusClientWithDeps("http://target.example.com", "user", "pass", true, false, targetHTTP, &RealFileSystem{})

		err := sourceClient.TransferFile(targetClient, "source-repo", "target-repo", "test-file.txt", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to download file")
	})

	t.Run("error - upload fails", func(t *testing.T) {
		sourceHTTP := &MockHTTPClient{}
		targetHTTP := &MockHTTPClient{}

		sourceHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: httpStatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte("test content"))),
			}, nil
		}

		targetHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		}

		sourceClient := NewNexusClientWithDeps("http://source.example.com", "user", "pass", true, false, sourceHTTP, &RealFileSystem{})
		targetClient := NewNexusClientWithDeps("http://target.example.com", "user", "pass", true, false, targetHTTP, &RealFileSystem{})

		err := sourceClient.TransferFile(targetClient, "source-repo", "target-repo", "test-file.txt", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to upload file")
	})
}

func TestNexusClient_GetBaseURL(t *testing.T) {
	client := NewNexusClient("http://test.example.com", "user", "pass", false, false)
	assert.Equal(t, "http://test.example.com", client.GetBaseURL())
}

func TestNexusClient_DownloadFileWithPath(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		filePath      string
		destination   string
		root          string
		expectedError string
		setupMocks    func(*MockHTTPClient, *MockFileSystem)
	}{
		{
			name:        "success - downloads file with root path",
			repository:  "test-repo",
			filePath:    "file.txt",
			destination: "/tmp",
			root:        "root/path",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem) {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: httpStatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte("content"))),
					}, nil
				}
				mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				mockFS.CreateFunc = func(name string) (File, error) {
					return &MockFile{
						Writer: &bytes.Buffer{},
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
			},
		},
		{
			name:        "success - downloads file without root path",
			repository:  "test-repo",
			filePath:    "file.txt",
			destination: "/tmp",
			root:        "",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem) {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: httpStatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte("content"))),
					}, nil
				}
				mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				mockFS.CreateFunc = func(name string) (File, error) {
					return &MockFile{
						Writer: &bytes.Buffer{},
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
			},
		},
		{
			name:          "error - download fails",
			repository:    "test-repo",
			filePath:      "file.txt",
			destination:   "/tmp",
			root:          "",
			expectedError: "failed to download file",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem) {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				}
				mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{}
			mockFS := &MockFileSystem{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockHTTP, mockFS)
			}

			client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, false, mockHTTP, mockFS)

			err := client.DownloadFileWithPath(tt.repository, tt.filePath, tt.destination, tt.root)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNexusClient_DownloadDirectoryWithPath(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		dirPath       string
		destination   string
		root          string
		saveStructure bool
		expectedError string
		setupMocks    func(*MockHTTPClient, *MockFileSystem)
	}{
		{
			name:          "success - downloads directory with structure",
			repository:    "test-repo",
			dirPath:       "test-dir",
			destination:   "/tmp",
			root:          "",
			saveStructure: true,
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem) {
				callCount := 0
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					if strings.Contains(req.URL.String(), "search/assets") {
						body, _ := json.Marshal(SearchAssetsResponse{
							Items: []Asset{
								{Path: "test-dir/file1.txt"},
								{Path: "test-dir/file2.txt"},
							},
							ContinuationToken: "",
						})
						return &http.Response{
							StatusCode: httpStatusOK,
							Body:       io.NopCloser(bytes.NewReader(body)),
						}, nil
					}
					// Download requests
					callCount++
					return &http.Response{
						StatusCode: httpStatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte("content"))),
					}, nil
				}
				mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				mockFS.CreateFunc = func(name string) (File, error) {
					return &MockFile{
						Writer: &bytes.Buffer{},
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
			},
		},
		{
			name:          "success - downloads directory without structure",
			repository:    "test-repo",
			dirPath:       "test-dir",
			destination:   "/tmp",
			root:          "",
			saveStructure: false,
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem) {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					if strings.Contains(req.URL.String(), "search/assets") {
						body, _ := json.Marshal(SearchAssetsResponse{
							Items: []Asset{
								{Path: "test-dir/file1.txt"},
							},
							ContinuationToken: "",
						})
						return &http.Response{
							StatusCode: httpStatusOK,
							Body:       io.NopCloser(bytes.NewReader(body)),
						}, nil
					}
					return &http.Response{
						StatusCode: httpStatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte("content"))),
					}, nil
				}
				mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				mockFS.CreateFunc = func(name string) (File, error) {
					return &MockFile{
						Writer: &bytes.Buffer{},
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
			},
		},
		{
			name:          "error - GetFilesInDirectory fails",
			repository:    "test-repo",
			dirPath:       "test-dir",
			destination:   "/tmp",
			root:          "",
			saveStructure: false,
			expectedError: "failed to get files in directory",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem) {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{}
			mockFS := &MockFileSystem{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockHTTP, mockFS)
			}

			client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, false, mockHTTP, mockFS)

			err := client.DownloadDirectoryWithPath(tt.repository, tt.dirPath, tt.destination, tt.root, tt.saveStructure)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNexusClient_UploadDirectory(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		dirPath       string
		relative      bool
		destination   string
		dryRun        bool
		expectedError string
		setupMocks    func(*MockHTTPClient, *MockFileSystem)
	}{
		{
			name:        "success - uploads directory with relative paths",
			repository:  "test-repo",
			dirPath:     "/tmp/test-dir",
			relative:    true,
			destination: "upload/",
			dryRun:      false,
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem) {
				walkCalled := false
				mockFS.WalkFunc = func(root string, walkFn filepath.WalkFunc) error {
					if !walkCalled {
						walkCalled = true
						// Simulate walking a file
						info := &mockFileInfo{name: "file.txt", isDir: false}
						return walkFn("/tmp/test-dir/file.txt", info, nil)
					}
					return nil
				}
				mockFS.OpenFunc = func(name string) (File, error) {
					return &MockFile{
						Reader: bytes.NewReader([]byte("content")),
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: httpStatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			},
		},
		{
			name:        "success - uploads directory with absolute paths",
			repository:  "test-repo",
			dirPath:     "/tmp/test-dir",
			relative:    false,
			destination: "upload/",
			dryRun:      false,
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem) {
				walkCalled := false
				mockFS.WalkFunc = func(root string, walkFn filepath.WalkFunc) error {
					if !walkCalled {
						walkCalled = true
						info := &mockFileInfo{name: "file.txt", isDir: false}
						return walkFn("/tmp/test-dir/file.txt", info, nil)
					}
					return nil
				}
				mockFS.OpenFunc = func(name string) (File, error) {
					return &MockFile{
						Reader: bytes.NewReader([]byte("content")),
						Closer: io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: httpStatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				}
			},
		},
		{
			name:        "success - dry run",
			repository:  "test-repo",
			dirPath:     "/tmp/test-dir",
			relative:    true,
			destination: "upload/",
			dryRun:      true,
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem) {
				mockFS.WalkFunc = func(root string, walkFn filepath.WalkFunc) error {
					info := &mockFileInfo{name: "file.txt", isDir: false}
					return walkFn("/tmp/test-dir/file.txt", info, nil)
				}
			},
		},
		{
			name:          "error - walk fails",
			repository:    "test-repo",
			dirPath:       "/tmp/test-dir",
			relative:      true,
			destination:   "upload/",
			dryRun:        false,
			expectedError: "walk error",
			setupMocks: func(mockHTTP *MockHTTPClient, mockFS *MockFileSystem) {
				mockFS.WalkFunc = func(root string, walkFn filepath.WalkFunc) error {
					return errors.New("walk error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &MockHTTPClient{}
			mockFS := &MockFileSystem{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockHTTP, mockFS)
			}

			client := NewNexusClientWithDeps("http://test.example.com", "user", "pass", true, tt.dryRun, mockHTTP, mockFS)

			err := client.UploadDirectory(tt.repository, tt.dirPath, tt.relative, tt.destination)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// mockFileInfo is a mock implementation of os.FileInfo for testing
type mockFileInfo struct {
	name  string
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0 }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// Test helper: create a test HTTP server for integration-style tests
func createTestHTTPServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
	})
	return server
}
