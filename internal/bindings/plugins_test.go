/*
  plugins_test.go -- Unit tests for frontend plugin script loader bindings.
  Responsibilities: validate plugin directory discovery and deterministic ordering.
*/

package bindings

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPlugins_ListPluginScripts(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LIAOTAO_PLUGINS_DIR", tmp)

	if err := os.WriteFile(filepath.Join(tmp, "z-last.plugin.js"), []byte("window.z = 1;"), 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "a-first.plugin.js"), []byte("window.a = 1;"), 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "ignore.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatalf("write non-plugin: %v", err)
	}

	svc := &Service{}
	items, err := svc.ListPluginScripts(context.Background())
	if err != nil {
		t.Fatalf("ListPluginScripts failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 plugin files, got %d", len(items))
	}
	if items[0].Name != "a-first.plugin.js" {
		t.Fatalf("unexpected order[0]: %s", items[0].Name)
	}
	if items[1].Name != "z-last.plugin.js" {
		t.Fatalf("unexpected order[1]: %s", items[1].Name)
	}
}

func TestPlugins_ListPluginScripts_MissingDir(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "missing")
	t.Setenv("LIAOTAO_PLUGINS_DIR", tmp)

	svc := &Service{}
	items, err := svc.ListPluginScripts(context.Background())
	if err != nil {
		t.Fatalf("expected no error for missing directory, got: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty plugin list, got %d", len(items))
	}
}
