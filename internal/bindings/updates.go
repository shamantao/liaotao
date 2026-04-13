/*
  updates.go -- Auto-update feature bindings.
  Responsibilities: check for new versions, compare versions, prepare download metadata.
*/

package bindings

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
