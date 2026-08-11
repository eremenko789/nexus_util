package nexus

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNexusClientCreation(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "testuser", "testpass", false, false, false)

	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}

	if client.BaseURL != "http://test-nexus.example.com" {
		t.Errorf("Expected BaseURL 'http://test-nexus.example.com', got '%s'", client.BaseURL)
	}

	if client.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got '%s'", client.Username)
	}

	if client.Password != "testpass" {
		t.Errorf("Expected Password 'testpass', got '%s'", client.Password)
	}
}

func TestNexusClientWithTrailingSlash(t *testing.T) {
	// Test that trailing slashes are removed
	client := NewNexusClient("http://test-nexus.example.com/", "testuser", "testpass", false, false, false)

	if client.BaseURL != "http://test-nexus.example.com" {
		t.Errorf("Expected BaseURL without trailing slash 'http://test-nexus.example.com', got '%s'", client.BaseURL)
	}
}

func TestNexusClientQuietMode(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "testuser", "testpass", true, false, false)

	if !client.Quiet {
		t.Error("Expected Quiet mode to be enabled")
	}
}

func TestNexusClientDryRunMode(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "testuser", "testpass", false, true, false)

	if !client.DryRun {
		t.Error("Expected DryRun mode to be enabled")
	}
}

func TestNexusClientHTTPClient(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "testuser", "testpass", false, false, false)

	if client.HTTPClient == nil {
		t.Error("Expected HTTPClient to be initialized")
	}
}

func TestNexusClientInsecureMode(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "testuser", "testpass", false, false, true)

	if !client.Insecure {
		t.Error("Expected Insecure mode to be enabled")
	}

	// Check that HTTP client has insecure TLS config
	if client.HTTPClient.Transport == nil {
		t.Error("Expected Transport to be set when insecure mode is enabled")
	}

	transport, ok := client.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Error("Expected Transport to be *http.Transport")
	}

	if transport.TLSClientConfig == nil {
		t.Error("Expected TLSClientConfig to be set")
	}

	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true")
	}
}

func TestEncodeRepositoryPathWithSpaces(t *testing.T) {
	encoded := encodeRepositoryPath("dir/with space/file name.txt")

	expected := "dir/with%20space/file%20name.txt"
	if encoded != expected {
		t.Fatalf("Expected encoded path '%s', got '%s'", expected, encoded)
	}
}

func TestRepositoryURLWithSpaces(t *testing.T) {
	client := NewNexusClient("http://nexus.redkit-lab.ru:8081", "testuser", "testpass", false, false, false)

	// got := client.repositoryURL("myrepo", "dir/with space/file name.txt")
	// expected := "http://test-nexus.example.com/repository/myrepo/dir/with%20space/file%20name.txt"

	// if got != expected {
	// 	t.Fatalf("Expected repository URL '%s', got '%s'", expected, got)
	// }

	got := client.repositoryURL("scada", "2507/rc/192/linux/install_logs/info_Alt Linux (Workstation)_2861.json")
	expected := "http://nexus.redkit-lab.ru:8081/repository/scada/2507/rc/192/linux/install_logs/info_Alt%20Linux%20(Workstation)_2861.json"

	if got != expected {
		t.Fatalf("Expected repository URL '%s', got '%s'", expected, got)
	}

}

func TestNexusClientHasNoOverallHTTPTimeout(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "testuser", "testpass", true, false, false)
	if client.HTTPClient.Timeout != 0 {
		t.Fatalf("Expected HTTPClient.Timeout=0 for large downloads, got %v", client.HTTPClient.Timeout)
	}
	transport, ok := client.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Expected *http.Transport, got %T", client.HTTPClient.Transport)
	}
	if transport.ResponseHeaderTimeout != responseHeaderTimeout {
		t.Fatalf("Expected ResponseHeaderTimeout=%v for uploads, got %v", responseHeaderTimeout, transport.ResponseHeaderTimeout)
	}
	if responseHeaderTimeout < 30*time.Minute {
		t.Fatalf("ResponseHeaderTimeout too short for asset push: %v", responseHeaderTimeout)
	}
	if uploadTimeout < 30*time.Minute {
		t.Fatalf("uploadTimeout too short for asset push: %v", uploadTimeout)
	}
}

func TestDownloadFileByUrlStreamsToDisk(t *testing.T) {
	const payload = "streamed-asset-payload"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			t.Errorf("unexpected auth: ok=%v user=%q pass=%q", ok, user, pass)
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		_, _ = io.WriteString(w, payload)
	}))
	defer server.Close()

	client := NewNexusClient(server.URL, "testuser", "testpass", true, false, false)
	destDir := t.TempDir()
	destPath := filepath.Join(destDir, "subdir", "asset.bin")

	if err := client.DownloadFileByUrl(server.URL+"/repository/raw/big.bin", destPath); err != nil {
		t.Fatalf("DownloadFileByUrl failed: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(got) != payload {
		t.Fatalf("downloaded content mismatch: got %q want %q", got, payload)
	}
}

func TestDownloadFileByUrlIncompleteBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Advertise more bytes than we actually send — mimics truncated multi-GB transfers.
		w.Header().Set("Content-Length", "1024")
		_, _ = io.WriteString(w, "short")
	}))
	defer server.Close()

	client := NewNexusClient(server.URL, "testuser", "testpass", true, false, false)
	destPath := filepath.Join(t.TempDir(), "truncated.bin")

	err := client.DownloadFileByUrl(server.URL+"/repository/raw/big.bin", destPath)
	if err == nil {
		t.Fatal("expected incomplete download error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to write file content") &&
		!strings.Contains(err.Error(), "incomplete download") &&
		!strings.Contains(err.Error(), "EOF") {
		t.Fatalf("expected EOF/incomplete download error, got: %v", err)
	}
}

func TestDownloadFileByUrlDryRun(t *testing.T) {
	client := NewNexusClient("http://test-nexus.example.com", "testuser", "testpass", true, true, false)
	destPath := filepath.Join(t.TempDir(), "should-not-exist.bin")

	if err := client.DownloadFileByUrl("http://test-nexus.example.com/repository/raw/big.bin", destPath); err != nil {
		t.Fatalf("dry-run DownloadFileByUrl failed: %v", err)
	}
	if _, err := os.Stat(destPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not create destination file, stat err=%v", err)
	}
}

func TestUploadFileStreamsFromDisk(t *testing.T) {
	const payload = "upload-stream-payload"
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read upload body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	srcPath := filepath.Join(t.TempDir(), "source.bin")
	if err := os.WriteFile(srcPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	client := NewNexusClient(server.URL, "testuser", "testpass", true, false, false)
	if err := client.UploadFile("raw", srcPath, "path/to/asset.bin"); err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	if string(gotBody) != payload {
		t.Fatalf("uploaded content mismatch: got %q want %q", gotBody, payload)
	}
}

func TestUploadFileAllowsSlowResponseHeaders(t *testing.T) {
	// Simulates Nexus taking a while after receiving the body before sending
	// response headers. A short ResponseHeaderTimeout aborts otherwise-successful PUTs.
	const payload = "slow-header-upload"
	const headerDelay = 1500 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read upload body: %v", err)
			return
		}
		if string(body) != payload {
			t.Errorf("unexpected body: %q", body)
		}
		time.Sleep(headerDelay)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	srcPath := filepath.Join(t.TempDir(), "slow.bin")
	if err := os.WriteFile(srcPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	if responseHeaderTimeout < 30*time.Minute {
		t.Fatalf("production ResponseHeaderTimeout too short for asset push: %v", responseHeaderTimeout)
	}
	if uploadTimeout < 30*time.Minute {
		t.Fatalf("production uploadTimeout too short for asset push: %v", uploadTimeout)
	}

	shortClient := NewNexusClient(server.URL, "testuser", "testpass", true, false, false)
	shortTransport := shortClient.HTTPClient.Transport.(*http.Transport).Clone()
	shortTransport.ResponseHeaderTimeout = 200 * time.Millisecond
	shortClient.HTTPClient.Transport = shortTransport
	if err := shortClient.UploadFile("raw", srcPath, "path/to/slow.bin"); err == nil {
		t.Fatal("expected upload to fail with short ResponseHeaderTimeout")
	}

	okClient := NewNexusClient(server.URL, "testuser", "testpass", true, false, false)
	okTransport := okClient.HTTPClient.Transport.(*http.Transport).Clone()
	okTransport.ResponseHeaderTimeout = 5 * time.Second
	okClient.HTTPClient.Transport = okTransport
	if err := okClient.UploadFile("raw", srcPath, "path/to/slow.bin"); err != nil {
		t.Fatalf("UploadFile should wait for slow response headers: %v", err)
	}
}

func TestUploadFromBufferUsesUploadTimeout(t *testing.T) {
	const payload = "buffer-upload-payload"
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read upload body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewNexusClient(server.URL, "testuser", "testpass", true, false, false)
	if err := client.UploadFromBuffer("raw", "path/to/buffer.bin", []byte(payload)); err != nil {
		t.Fatalf("UploadFromBuffer failed: %v", err)
	}
	if string(gotBody) != payload {
		t.Fatalf("uploaded content mismatch: got %q want %q", gotBody, payload)
	}
}
