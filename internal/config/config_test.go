package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultConfig(t *testing.T) {
	// Create a temporary default.toml for testing
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[app]
name = "TestApp"
version = "0.1.0"
mode = "debug"
language = "en"

[config]
schema_version = 1
enable_layered_merge = true
strict_mode = false

[path_manager]
allowed_roots = ["/tmp"]
temp_dir = "/tmp/test/.tmp"
logs_dir = "/tmp/test/logs"
reports_dir = "/tmp/test/reports"
collision_strategy = "increment"
normalize_unicode = false
trim_whitespace = true

[database]
path = "/tmp/test/data/liaotao.db"
busy_timeout_ms = 5000
journal_mode = "WAL"
foreign_keys = true

[logger]
level = "info"
console_pretty = true
file_json = true
rotation_enabled = true
max_file_mb = 20
max_files = 5
include_context_ids = true

[reporting]
enabled = true
json_report = true
csv_report = false
include_failed = true
`
	if err := os.WriteFile(filepath.Join(configDir, "default.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Change to tmpDir so bundledDefaultPath() finds config/default.toml
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.App.Name != "TestApp" {
		t.Errorf("expected app.name=TestApp, got %s", cfg.App.Name)
	}
	if cfg.App.Mode != "debug" {
		t.Errorf("expected app.mode=debug, got %s", cfg.App.Mode)
	}
}

func TestValidateRejectsInvalidMode(t *testing.T) {
	cfg := &AppConfig{
		App: AppSection{Name: "Test", Mode: "invalid"},
	}
	if err := validate(cfg); err == nil {
		t.Error("expected validation error for invalid mode")
	}
}

func TestEnvOverride(t *testing.T) {
	cfg := &AppConfig{
		App: AppSection{Name: "Test", Mode: "debug"},
	}
	t.Setenv("APP__APP__MODE", "normal")
	applyEnvOverrides(cfg)

	if cfg.App.Mode != "normal" {
		t.Errorf("expected mode=normal after env override, got %s", cfg.App.Mode)
	}
}

func TestLoadBuiltInDefaultWhenConfigFileMissing(t *testing.T) {
	tmpDir := t.TempDir()

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir tmp dir: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with missing default.toml should fallback, got error: %v", err)
	}

	if cfg.App.Name != "liaotao" {
		t.Fatalf("expected built-in app.name=liaotao, got %q", cfg.App.Name)
	}
	if cfg.Database.Path == "" {
		t.Fatal("expected built-in database.path to be set")
	}
}
