package nexus

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// HTTPClient interface for making HTTP requests
// This allows us to mock HTTP calls in tests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// FileSystem interface for file operations
// This allows us to mock file system operations in tests
type FileSystem interface {
	Open(name string) (File, error)
	Create(name string) (File, error)
	Stat(name string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Walk(root string, walkFn filepath.WalkFunc) error
}

// File interface for file operations
type File interface {
	io.Reader
	io.Writer
	io.Closer
	Stat() (os.FileInfo, error)
}

// RealFileSystem implements FileSystem using real os operations
type RealFileSystem struct{}

func (fs *RealFileSystem) Open(name string) (File, error) {
	return os.Open(name)
}

func (fs *RealFileSystem) Create(name string) (File, error) {
	return os.Create(name)
}

func (fs *RealFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (fs *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *RealFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (fs *RealFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (fs *RealFileSystem) Walk(root string, walkFn filepath.WalkFunc) error {
	return filepath.Walk(root, walkFn)
}

// Client interface for Nexus operations
// This allows commands to use mocks in tests
type Client interface {
	GetFilesInDirectory(repository string, dirPath string) ([]string, error)
	DeleteFile(repository string, filePath string) error
	DeleteDirectory(repository string, dirPath string) error
	DownloadFile(repository string, filePath string, destPath string) error
	UploadFile(repository string, filePath string, destPath string) error
	UploadDirectory(repository string, dirPath string, relative bool, destination string) error
	DownloadFileWithPath(repository string, filePath string, destination string, root string) error
	DownloadDirectoryWithPath(repository string, dirPath string, destination string, root string, saveStructure bool) error
	ListRepositories() ([]Repository, error)
	FileExists(repository string, filePath string) (bool, error)
	GetFileSize(repository string, filePath string) (int64, error)
	DownloadToBuffer(repository string, filePath string) ([]byte, error)
	UploadFromBuffer(repository string, destPath string, content []byte) error
	TransferFile(target Client, sourceRepo string, targetRepo string, filePath string, skipIfExists bool) error
	Logf(format string, args ...interface{})
	GetBaseURL() string // Returns the base URL of the Nexus server
}

