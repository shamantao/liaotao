// plugins.go -- Backend plugin script loader for frontend plugin system.
// Exposes ListPluginScripts for PLUG-04 (load JS plugins from plugins/ directory).

package bindings

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PluginScriptRecord is returned to the frontend plugin loader.
type PluginScriptRecord struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// ListPluginScripts returns all plugin JS scripts found in plugins/.
// Files are loaded in lexical order for deterministic startup behavior.
func (s *Service) ListPluginScripts(_ context.Context) ([]PluginScriptRecord, error) {
	pluginsDir := resolvePluginsDir()
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []PluginScriptRecord{}, nil
		}
		return nil, fmt.Errorf("read plugins dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".plugin.js") || strings.HasSuffix(name, ".js") {
			files = append(files, name)
		}
	}
	sort.Strings(files)

	out := make([]PluginScriptRecord, 0, len(files))
	for _, name := range files {
		fullPath := filepath.Join(pluginsDir, name)
		body, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("read plugin file %s: %w", name, err)
		}
		out = append(out, PluginScriptRecord{
			Name:    name,
			Path:    fullPath,
			Content: string(body),
		})
	}

	return out, nil
}

func resolvePluginsDir() string {
	if envDir := strings.TrimSpace(os.Getenv("LIAOTAO_PLUGINS_DIR")); envDir != "" {
		return envDir
	}
	return "plugins"
}
