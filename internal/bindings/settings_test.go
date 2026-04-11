/*
  settings_test.go -- Unit tests for settings bindings (SET-02/05/06).
  Covers general settings persistence and TOML import/export roundtrip.
*/

package bindings

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func newSettingsTestService(t *testing.T) *Service {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL DEFAULT 'openai-compatible',
			url TEXT NOT NULL DEFAULT '',
			api_key TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			use_in_rag INTEGER NOT NULL DEFAULT 0,
			active INTEGER NOT NULL DEFAULT 1,
			temperature REAL NOT NULL DEFAULT 0.7,
			num_ctx INTEGER NOT NULL DEFAULT 1024,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE mcp_servers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			transport TEXT NOT NULL DEFAULT 'http',
			url TEXT NOT NULL DEFAULT '',
			command TEXT NOT NULL DEFAULT '',
			args TEXT NOT NULL DEFAULT '[]',
			active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return NewService(db)
}

func TestSettings_GeneralRoundtrip(t *testing.T) {
	svc := newSettingsTestService(t)
	ctx := context.Background()

	updated, err := svc.UpdateGeneralSettings(ctx, GeneralSettings{
		Language:            "en",
		Theme:               "dark",
		DefaultSystemPrompt: "Be concise",
	})
	if err != nil {
		t.Fatalf("UpdateGeneralSettings: %v", err)
	}
	if updated.Language != "en" || updated.DefaultSystemPrompt != "Be concise" {
		t.Fatalf("unexpected update result: %+v", updated)
	}

	loaded, err := svc.GetGeneralSettings(ctx)
	if err != nil {
		t.Fatalf("GetGeneralSettings: %v", err)
	}
	if loaded.Language != "en" || loaded.Theme != "dark" || loaded.DefaultSystemPrompt != "Be concise" {
		t.Fatalf("unexpected loaded settings: %+v", loaded)
	}
}

func TestSettings_LanguageSupportsZhTW(t *testing.T) {
	svc := newSettingsTestService(t)
	ctx := context.Background()

	updated, err := svc.UpdateGeneralSettings(ctx, GeneralSettings{
		Language:            "zh-TW",
		Theme:               "dark",
		DefaultSystemPrompt: "",
	})
	if err != nil {
		t.Fatalf("UpdateGeneralSettings: %v", err)
	}
	if updated.Language != "zh-TW" {
		t.Fatalf("unexpected language after update: %+v", updated)
	}

	loaded, err := svc.GetGeneralSettings(ctx)
	if err != nil {
		t.Fatalf("GetGeneralSettings: %v", err)
	}
	if loaded.Language != "zh-TW" {
		t.Fatalf("unexpected loaded language: %+v", loaded)
	}
}

func TestSettings_ExportImportConfiguration(t *testing.T) {
	svc := newSettingsTestService(t)
	ctx := context.Background()

	_, _ = svc.UpdateGeneralSettings(ctx, GeneralSettings{Language: "fr", Theme: "dark", DefaultSystemPrompt: "Global prompt"})
	_, err := svc.CreateProvider(ctx, CreateProviderPayload{
		Name:        "Provider One",
		Type:        "openai-compatible",
		URL:         "https://api.example.com/v1",
		APIKey:      "sk-export-me",
		Description: "demo",
		UseInRAG:    true,
		Active:      true,
		Temperature: 0.9,
		NumCtx:      2048,
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	_, err = svc.SaveMCPServer(ctx, SaveMCPServerPayload{
		Name:      "mcp-one",
		Transport: "http",
		URL:       "http://localhost:8201/mcp",
		Args:      "[]",
		Active:    true,
	})
	if err != nil {
		t.Fatalf("SaveMCPServer: %v", err)
	}

	exported, err := svc.ExportConfiguration(ctx)
	if err != nil {
		t.Fatalf("ExportConfiguration: %v", err)
	}
	if !strings.Contains(exported, "[general]") || !strings.Contains(exported, "Provider One") {
		t.Fatalf("unexpected export content:\n%s", exported)
	}

	// Import into a fresh service.
	svc2 := newSettingsTestService(t)
	if _, err := svc2.ImportConfiguration(ctx, settingsImportPayload{TOML: exported}); err != nil {
		t.Fatalf("ImportConfiguration: %v", err)
	}

	settings, err := svc2.GetGeneralSettings(ctx)
	if err != nil {
		t.Fatalf("GetGeneralSettings imported: %v", err)
	}
	if settings.DefaultSystemPrompt != "Global prompt" {
		t.Fatalf("general settings not imported: %+v", settings)
	}

	providers, err := svc2.ListProviders(ctx, ListProvidersPayload{})
	if err != nil {
		t.Fatalf("ListProviders imported: %v", err)
	}
	if len(providers) != 1 || providers[0].Name != "Provider One" {
		t.Fatalf("providers not imported: %+v", providers)
	}

	mcpServers, err := svc2.ListMCPServers(ctx)
	if err != nil {
		t.Fatalf("ListMCPServers imported: %v", err)
	}
	if len(mcpServers) != 1 || mcpServers[0].Name != "mcp-one" {
		t.Fatalf("mcp servers not imported: %+v", mcpServers)
	}
}

func TestSettings_ExportConfigurationToFile(t *testing.T) {
	svc := newSettingsTestService(t)
	ctx := context.Background()

	tmpHome := t.TempDir()
	downloadsDir := filepath.Join(tmpHome, "Downloads")
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		t.Fatalf("mkdir downloads: %v", err)
	}
	t.Setenv("HOME", tmpHome)

	result, err := svc.ExportConfigurationToFile(ctx)
	if err != nil {
		t.Fatalf("ExportConfigurationToFile: %v", err)
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got %+v", result)
	}

	pathValue, _ := result["path"].(string)
	if strings.TrimSpace(pathValue) == "" {
		t.Fatalf("missing exported file path: %+v", result)
	}
	raw, readErr := os.ReadFile(pathValue)
	if readErr != nil {
		t.Fatalf("read exported file: %v", readErr)
	}
	content := string(raw)
	if !strings.Contains(content, "[general]") {
		t.Fatalf("unexpected export file content:\n%s", content)
	}
}
