/*
  updates.go -- Auto-update feature bindings.
  Responsibilities: check for new versions, compare versions, prepare download metadata.
*/

package bindings

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// GitHubRelease represents a GitHub API release object (subset of fields).
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
		Size        int64  `json:"size"`
	} `json:"assets"`
	PublishedAt string `json:"published_at"`
	Prerelease  bool   `json:"prerelease"`
	Draft       bool   `json:"draft"`
}

// UpdateCheckResult is returned by CheckForUpdate binding.
type UpdateCheckResult struct {
	HasUpdate       bool        `json:"has_update"`         // true if newer version exists
	CurrentVersion  string      `json:"current_version"`    // e.g., "0.2.3"
	LatestVersion   string      `json:"latest_version"`     // e.g., "0.2.4"
	LatestRelease   *GitHubRelease `json:"latest_release"`  // full release metadata
	Error           string      `json:"error,omitempty"`   // error message if check failed
	CheckedAt       time.Time   `json:"checked_at"`        // timestamp of check
}

const (
	githubRepoURL    = "https://api.github.com/repos/shamantao/liaotao/releases/latest"
	updateCheckTimeout = 5 * time.Second
)

// CheckForUpdate fetches the latest release from GitHub and compares with current version.
func (s *Service) CheckForUpdate(ctx context.Context) (UpdateCheckResult, error) {
	currentVer := getCurrentVersion()

	// Add timeout to context if not already present
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, updateCheckTimeout)
		defer cancel()
	}

	release, err := fetchLatestReleaseWithClient(ctx, &http.Client{Timeout: updateCheckTimeout})
	if err != nil {
		return UpdateCheckResult{
			CurrentVersion: currentVer,
			Error:          err.Error(),
			CheckedAt:      time.Now(),
		}, nil
	}

	// Parse versions for comparison
	latestVer := parseVersion(release.TagName) // e.g., "v0.2.4" -> "0.2.4"
	hasUpdate := compareVersions(currentVer, latestVer) < 0

	return UpdateCheckResult{
		HasUpdate:      hasUpdate,
		CurrentVersion: currentVer,
		LatestVersion:  latestVer,
		LatestRelease:  release,
		CheckedAt:      time.Now(),
	}, nil
}

// getCurrentVersion reads version from VERSION file or returns "dev".
func getCurrentVersion() string {
	version := "dev"
	if data, err := os.ReadFile("VERSION"); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			version = v
		}
	}
	return version
}

// fetchLatestReleaseWithClient calls GitHub API with provided HTTP client.
// Allows tests to inject mock clients pointing to test servers.
func fetchLatestReleaseWithClient(ctx context.Context, client *http.Client) (*GitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", githubRepoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add User-Agent (GitHub API recommends it)
	req.Header.Set("User-Agent", "liaotao-autoupdate/0.1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse release: %w", err)
	}

	return &release, nil
}

// parseVersion strips "v" prefix from semantic version (e.g., "v0.2.4" -> "0.2.4").
func parseVersion(tagName string) string {
	ver := strings.TrimSpace(tagName)
	if strings.HasPrefix(ver, "v") {
		ver = ver[1:]
	}
	return ver
}

// compareVersions compares two semantic versions (e.g., "0.2.3" vs "0.2.4" or "v0.2.3" vs "v0.2.4").
// Returns: -1 if a < b, 0 if a == b, 1 if a > b.
// Handles "v" prefix automatically.
func compareVersions(a, b string) int {
	// Normalize versions by removing "v" prefix if present
	a = parseVersion(a)
	b = parseVersion(b)

	aParts := parseSemVer(a)
	bParts := parseSemVer(b)

	// Compare major, minor, patch
	for i := 0; i < 3 && i < len(aParts) && i < len(bParts); i++ {
		aNum := aParts[i]
		bNum := bParts[i]
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
	}

	// If one version has more parts and they're all equal up to this point
	if len(aParts) < len(bParts) {
		return -1
	}
	if len(aParts) > len(bParts) {
		return 1
	}

	return 0
}

// parseSemVer splits "0.2.3" into [0, 2, 3], handling non-numeric suffixes gracefully.
func parseSemVer(ver string) []int {
	// Remove any non-numeric suffix (e.g., "0.2.3-beta" -> "0.2.3")
	baseParts := strings.Split(strings.Split(ver, "-")[0], ".")

	var parts []int
	for _, part := range baseParts {
		if num, err := strconv.Atoi(part); err == nil {
			parts = append(parts, num)
		}
	}

	// Ensure at least 3 parts (major, minor, patch)
	for len(parts) < 3 {
		parts = append(parts, 0)
	}

	return parts
}

// DownloadInstallResult is returned by DownloadAndInstallUpdate binding (UPD-03).
type DownloadInstallResult struct {
	Success       bool   `json:"success"`               // true if download+install succeeded
	Version       string `json:"version"`               // version that was installed (e.g., "0.2.4")
	BinaryPath    string `json:"binary_path,omitempty"` // path to installed binary
	Error         string `json:"error,omitempty"`       // error message if failed
	Message       string `json:"message,omitempty"`     // human-friendly status message
	InstalledAt   time.Time `json:"installed_at"`       // timestamp of installation
}

// ReleaseAsset represents a GitHub Release asset.
type ReleaseAsset struct {
	Name        string
	DownloadURL string
	Size        int64
}

// DownloadAndInstallUpdate downloads the latest release binary for the current platform
// and installs it to an appropriate system location.
// Implements UPD-03: one-click install of new binary.
func (s *Service) DownloadAndInstallUpdate(ctx context.Context) (DownloadInstallResult, error) {
	// Check for updates first to get the latest release
	updateResult, err := s.CheckForUpdate(ctx)
	if err != nil {
		return DownloadInstallResult{
			Error:       fmt.Sprintf("version check failed: %v", err),
			InstalledAt: time.Now(),
		}, nil
	}

	if updateResult.Error != "" {
		return DownloadInstallResult{
			Error:       updateResult.Error,
			InstalledAt: time.Now(),
		}, nil
	}

	if !updateResult.HasUpdate {
		return DownloadInstallResult{
			Success:   false,
			Error:     "already running latest version",
			Message:   fmt.Sprintf("Current: %s, Latest: %s", updateResult.CurrentVersion, updateResult.LatestVersion),
			InstalledAt: time.Now(),
		}, nil
	}

	// Find the appropriate binary asset for this platform
	asset := findBinaryAssetForPlatform(updateResult.LatestRelease)
	if asset == nil {
		return DownloadInstallResult{
			Error:       "no binary available for this platform",
			Message:     fmt.Sprintf("platform: %s/%s", runtime.GOOS, runtime.GOARCH),
			InstalledAt: time.Now(),
		}, nil
	}

	// Download the binary
	binPath, err := downloadAndExtractBinary(ctx, asset, updateResult.LatestVersion)
	if err != nil {
		return DownloadInstallResult{
			Error:       fmt.Sprintf("download/extract failed: %v", err),
			InstalledAt: time.Now(),
		}, nil
	}

	return DownloadInstallResult{
		Success:     true,
		Version:     updateResult.LatestVersion,
		BinaryPath:  binPath,
		Message:     fmt.Sprintf("Successfully installed liaotao %s to %s", updateResult.LatestVersion, binPath),
		InstalledAt: time.Now(),
	}, nil
}

// findBinaryAssetForPlatform returns the appropriate GitHub asset for the current OS/arch.
func findBinaryAssetForPlatform(release *GitHubRelease) *ReleaseAsset {
	if release == nil || len(release.Assets) == 0 {
		return nil
	}

	expected := getBinaryFilenamForPlatform(release.TagName)
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, expected) {
			return &ReleaseAsset{
				Name:        asset.Name,
				DownloadURL: asset.DownloadURL,
				Size:        asset.Size,
			}
		}
	}

	return nil
}

// getBinaryFilenameForPlatform returns the expected binary filename for the current platform.
func getBinaryFilenamForPlatform(tagName string) string {
	version := parseVersion(tagName) // strip "v" prefix
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map Go-style OS names to release artifact names
	osName := goos
	if goos == "darwin" {
		osName = "darwin"
	} else if goos == "linux" {
		osName = "linux"
	} else if goos == "windows" {
		osName = "mingw64_nt" // Windows artifacts use mingw naming
	}

	archName := goarch
	if goarch == "amd64" {
		archName = "x86_64"
	}

	return fmt.Sprintf("liaotao-v%s-%s-%s", version, osName, archName)
}

// downloadAndExtractBinary downloads a release asset and extracts the binary to an install path.
func downloadAndExtractBinary(ctx context.Context, asset *ReleaseAsset, version string) (string, error) {
	// Download the asset
	req, err := http.NewRequestWithContext(ctx, "GET", asset.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("create download request: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d downloading %s", resp.StatusCode, asset.Name)
	}

	// Determine install path based on platform
	installPath, err := getInstallPath()
	if err != nil {
		return "", fmt.Errorf("determine install path: %w", err)
	}

	// Extract and place binary
	if strings.HasSuffix(asset.Name, ".zip") {
		if err := extractZipBinary(resp.Body, installPath); err != nil {
			return "", fmt.Errorf("extract zip: %w", err)
		}
	} else if strings.HasSuffix(asset.Name, ".tar.gz") {
		if err := extractTarGzBinary(resp.Body, installPath); err != nil {
			return "", fmt.Errorf("extract tar.gz: %w", err)
		}
	} else {
		return "", fmt.Errorf("unsupported archive format: %s", asset.Name)
	}

	return installPath, nil
}

// getInstallPath returns the appropriate installation directory for the current platform.
func getInstallPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Applications/liaotao
		appDir := filepath.Join(homeDir, "Applications", "liaotao")
		if err := os.MkdirAll(appDir, 0755); err != nil {
			return "", fmt.Errorf("create app dir: %w", err)
		}
		return filepath.Join(appDir, "liaotao"), nil

	case "linux":
		// Linux: ~/.local/bin/liaotao
		binDir := filepath.Join(homeDir, ".local", "bin")
		if err := os.MkdirAll(binDir, 0755); err != nil {
			return "", fmt.Errorf("create bin dir: %w", err)
		}
		return filepath.Join(binDir, "liaotao"), nil

	case "windows":
		// Windows: AppData\Local\liaotao\liaotao.exe
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return "", fmt.Errorf("LOCALAPPDATA env var not set")
		}
		appDir := filepath.Join(localAppData, "liaotao")
		if err := os.MkdirAll(appDir, 0755); err != nil {
			return "", fmt.Errorf("create app dir: %w", err)
		}
		return filepath.Join(appDir, "liaotao.exe"), nil

	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// extractZipBinary extracts the liaotao binary from a zip archive.
func extractZipBinary(src io.Reader, destPath string) error {
	// Read entire zip into memory (GitHub artifacts are small)
	data, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("read zip data: %w", err)
	}

	// Open zip reader
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}

	// Find and extract the liaotao binary
	for _, file := range zr.File {
		if strings.HasSuffix(file.Name, "liaotao") || strings.HasSuffix(file.Name, "liaotao.exe") {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("open file in zip: %w", err)
			}
			defer rc.Close()

			// Write to destination with executable permissions
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("create dest file: %w", err)
			}
			defer out.Close()

			if _, err := io.Copy(out, rc); err != nil {
				return fmt.Errorf("extract binary: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("liaotao binary not found in zip")
}

// extractTarGzBinary extracts the liaotao binary from a tar.gz archive.
func extractTarGzBinary(src io.Reader, destPath string) error {
	// Decompress gzip stream
	gr, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gr.Close()

	// Open tar reader
	tr := tar.NewReader(gr)

	// Find and extract the liaotao binary
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("liaotao binary not found in tar.gz")
		}
		if err != nil {
			return fmt.Errorf("read tar header: %w", err)
		}

		if strings.HasSuffix(hdr.Name, "liaotao") || strings.HasSuffix(hdr.Name, "liaotao.exe") {
			// Write to destination with executable permissions
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("create dest file: %w", err)
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil {
				return fmt.Errorf("extract binary: %w", err)
			}

			return nil
		}
	}
}

