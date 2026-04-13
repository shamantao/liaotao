/*
  updates_test.go -- Tests for auto-update feature (UPD-01, UPD-04).
  Regression tests: version comparison logic, GitHub API integration, checksum validation.
*/

package bindings

import (
	"context"
	"os"
	"path/filepath"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// replaceHostTransport wraps an http.RoundTripper and replaces the request URL host.
// Used for testing to redirect API calls to a mock server.
type replaceHostTransport struct {
	baseURL string
	inner   http.RoundTripper
}

func (t *replaceHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Parse the base URL
	baseURLParsed, _ := url.Parse(t.baseURL)
	// Replace the scheme and host, keep the path
	req.URL.Scheme = baseURLParsed.Scheme
	req.URL.Host = baseURLParsed.Host
	return t.inner.RoundTrip(req)
}

// TestUpdates_VersionComparison tests semantic version comparison logic.
func TestUpdates_VersionComparison(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected int // -1: a<b, 0: a==b, 1: a>b
	}{
		{"equal versions", "0.2.3", "0.2.3", 0},
		{"a < b (patch)", "0.2.2", "0.2.3", -1},
		{"a < b (minor)", "0.1.9", "0.2.0", -1},
		{"a < b (major)", "1.0.0", "2.0.0", -1},
		{"a > b (patch)", "0.2.4", "0.2.3", 1},
		{"a > b (minor)", "0.3.0", "0.2.9", 1},
		{"a > b (major)", "2.0.0", "1.9.9", 1},
		{"dev version", "dev", "0.2.3", -1},
		{"with v prefix removed", "0.2.4", "v0.2.3", 1},
		{"multiple digits", "0.10.1", "0.9.9", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestUpdates_ParseVersion tests version string parsing with v prefix removal.
func TestUpdates_ParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v0.2.3", "0.2.3"},
		{"0.2.3", "0.2.3"},
		{"v1.0.0", "1.0.0"},
		{"  v0.2.3  ", "0.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseVersion(tt.input)
			if result != tt.expected {
				t.Errorf("parseVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestUpdates_FetchLatestRelease_HasNewerVersion tests fetching newer release from API.
// Regression test: ensures newer versions are correctly identified and parsed.
func TestUpdates_FetchLatestRelease_HasNewerVersion(t *testing.T) {
	// Mock GitHub API server returning a newer release
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate GitHub API response with newer version
		response := `{
			"tag_name": "v0.2.4",
			"name": "liaotao v0.2.4",
			"body": "Minor improvements and bug fixes.",
			"assets": [
				{"name": "liaotao-v0.2.4-darwin-arm64.zip", "browser_download_url": "https://github.com/shamantao/liaotao/releases/download/v0.2.4/liaotao-v0.2.4-darwin-arm64.zip", "size": 13500000},
				{"name": "liaotao-v0.2.4-linux-x86_64.tar.gz", "browser_download_url": "https://github.com/shamantao/liaotao/releases/download/v0.2.4/liaotao-v0.2.4-linux-x86_64.tar.gz", "size": 13300000}
			],
			"published_at": "2026-04-13T10:00:00Z",
			"prerelease": false,
			"draft": false
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call with mock server URL
	mockClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &replaceHostTransport{
			baseURL: mockServer.URL,
			inner:   http.DefaultTransport,
		},
	}

	release, err := fetchLatestReleaseWithClient(ctx, mockClient)
	if err != nil {
		t.Fatalf("fetchLatestReleaseWithClient failed: %v", err)
	}

	if release == nil {
		t.Fatal("expected release to be populated")
	}

	if release.TagName != "v0.2.4" {
		t.Errorf("expected tag_name=v0.2.4, got %s", release.TagName)
	}

	if len(release.Assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(release.Assets))
	}
}

// TestUpdates_CheckForUpdate_UpToDate tests correct handling when version is current.
// Regression test: ensures no false-positive update notifications.
func TestUpdates_CheckForUpdate_UpToDate(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"tag_name": "v0.2.3",
			"name": "liaotao v0.2.3",
			"body": "Current release.",
			"assets": [],
			"published_at": "2026-04-13T06:00:00Z",
			"prerelease": false,
			"draft": false
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &replaceHostTransport{
			baseURL: mockServer.URL,
			inner:   http.DefaultTransport,
		},
	}

	release, err := fetchLatestReleaseWithClient(ctx, mockClient)
	if err != nil {
		t.Fatalf("fetchLatestReleaseWithClient failed: %v", err)
	}

	if release.TagName != "v0.2.3" {
		t.Errorf("expected tag_name=v0.2.3, got %s", release.TagName)
	}
}

// TestUpdates_FetchLatestRelease_APIError tests graceful handling of GitHub API failures.
// Regression test: ensures update check doesn't crash on HTTP errors.
func TestUpdates_FetchLatestRelease_APIError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &replaceHostTransport{
			baseURL: mockServer.URL,
			inner:   http.DefaultTransport,
		},
	}

	_, err := fetchLatestReleaseWithClient(ctx, mockClient)
	if err == nil {
		t.Error("expected error when API returns error status")
	}
}

// TestUpdates_FetchLatestRelease_Timeout tests handling of network timeouts.
// Regression test: ensures slow/offline checks return errors instead of hanging.
func TestUpdates_FetchLatestRelease_Timeout(t *testing.T) {
	// Mock server that delays response beyond timeout
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mockClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &replaceHostTransport{
			baseURL: mockServer.URL,
			inner:   http.DefaultTransport,
		},
	}

	_, err := fetchLatestReleaseWithClient(ctx, mockClient)
	if err == nil {
		t.Error("expected error when request times out")
	}
}

// TestUpdates_ExtractChecksumsFromReleaseBody validates parsing checksums from release notes.
func TestUpdates_ExtractChecksumsFromReleaseBody(t *testing.T) {
	body := `
liaotao-v0.3.0-darwin-arm64.zip: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
liaotao-v0.3.0-windows-x86_64.zip: BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB
not-a-checksum-line
bad-file: 12345
`

	checksums := extractChecksumsFromReleaseBody(body)
	if len(checksums) != 2 {
		t.Fatalf("expected 2 checksum entries, got %d", len(checksums))
	}

	if got := checksums["liaotao-v0.3.0-darwin-arm64.zip"]; got != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("unexpected checksum for darwin asset: %q", got)
	}

	if got := checksums["liaotao-v0.3.0-windows-x86_64.zip"]; got != "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB" {
		t.Fatalf("unexpected checksum for windows asset: %q", got)
	}
}

// TestUpdates_ValidateBinaryChecksum validates checksum success, mismatch and missing-checksum behavior.
func TestUpdates_ValidateBinaryChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "liaotao")

	content := []byte("liaotao-update-test")
	if err := os.WriteFile(filePath, content, 0o755); err != nil {
		t.Fatalf("write temp binary: %v", err)
	}

	actual, err := computeFileChecksum(filePath)
	if err != nil {
		t.Fatalf("computeFileChecksum failed: %v", err)
	}

	t.Run("valid checksum", func(t *testing.T) {
		err := validateBinaryChecksum(filePath, map[string]string{"liaotao": actual})
		if err != nil {
			t.Fatalf("expected checksum validation to pass, got: %v", err)
		}
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		err := validateBinaryChecksum(filePath, map[string]string{"liaotao": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"})
		if err == nil {
			t.Fatal("expected checksum mismatch error")
		}
	})

	t.Run("missing checksum is skipped", func(t *testing.T) {
		err := validateBinaryChecksum(filePath, map[string]string{"other-binary": actual})
		if err != nil {
			t.Fatalf("expected missing-checksum case to be skipped, got: %v", err)
		}
	})
}
