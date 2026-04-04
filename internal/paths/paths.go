// paths.go — Path manager module
// Centralizes ALL path logic. No other module should build paths directly.

package paths

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"liaotao/internal/config"
)

// RuntimePaths holds validated runtime directories.
type RuntimePaths struct {
	TempDir    string
	LogsDir    string
	ReportsDir string
}

// Build creates, validates, and returns all required runtime directories.
func Build(cfg *config.AppConfig) (*RuntimePaths, error) {
	pm := &cfg.PathManager

	rp := &RuntimePaths{
		TempDir:    pm.TempDir,
		LogsDir:    pm.LogsDir,
		ReportsDir: pm.ReportsDir,
	}

	for _, dir := range []string{rp.TempDir, rp.LogsDir, rp.ReportsDir} {
		if err := ensureSafe(dir, pm.AllowedRoots); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	return rp, nil
}

// ResolveOutput returns a non-colliding output file path using the given strategy.
func ResolveOutput(source, outputDir, ext, strategy string) (string, error) {
	stem := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	candidate := filepath.Join(outputDir, stem+"."+ext)

	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate, nil
	}

	switch strategy {
	case "increment":
		for i := 1; i < 10000; i++ {
			p := filepath.Join(outputDir, fmt.Sprintf("%s_%03d.%s", stem, i, ext))
			if _, err := os.Stat(p); os.IsNotExist(err) {
				return p, nil
			}
		}
		return "", fmt.Errorf("collision increment overflow for %s", source)

	case "suffix":
		ts := time.Now().Unix()
		return filepath.Join(outputDir, fmt.Sprintf("%s_%d.%s", stem, ts, ext)), nil

	case "short_hash":
		h := sha256.Sum256([]byte(source))
		hash := fmt.Sprintf("%08x", h[:4])
		return filepath.Join(outputDir, fmt.Sprintf("%s_%s.%s", stem, hash, ext)), nil

	default:
		return "", fmt.Errorf("unknown collision strategy: %s", strategy)
	}
}

// EnsureWithinAllowed checks that a path is inside at least one allowed root.
func EnsureWithinAllowed(path string, allowedRoots []string) error {
	return ensureSafe(path, allowedRoots)
}

func ensureSafe(path string, allowedRoots []string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("cannot resolve path %s: %w", path, err)
	}

	for _, root := range allowedRoots {
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		if strings.HasPrefix(abs, rootAbs) {
			return nil
		}
	}

	return fmt.Errorf("path '%s' is outside allowed roots", path)
}
