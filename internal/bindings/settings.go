/*
  settings.go -- General settings, import/export and about bindings for the Settings tab (v1.7).
  Responsibilities: persist general preferences in SQLite, expose TOML import/export,
  and provide app metadata for the About section.
*/

package bindings

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// GeneralSettings stores user-facing settings persisted in SQLite.
type GeneralSettings struct {
	Language            string `json:"language" toml:"language"`
	Theme               string `json:"theme" toml:"theme"`
	DefaultSystemPrompt string `json:"default_system_prompt" toml:"default_system_prompt"`
}

type settingsExportProvider struct {
	Name        string  `toml:"name"`
	Type        string  `toml:"type"`
	URL         string  `toml:"url"`
	APIKey      string  `toml:"api_key"`
	Description string  `toml:"description"`
	UseInRAG    bool    `toml:"use_in_rag"`
	Active      bool    `toml:"active"`
	Temperature float64 `toml:"temperature"`
	NumCtx      int     `toml:"num_ctx"`
}

type settingsExportMCP struct {
	Name      string `toml:"name"`
	Transport string `toml:"transport"`
	URL       string `toml:"url"`
	Command   string `toml:"command"`
	Args      string `toml:"args"`
	Active    bool   `toml:"active"`
}

type settingsExportPayload struct {
	General   GeneralSettings          `toml:"general"`
	Providers []settingsExportProvider `toml:"providers"`
	MCP       []settingsExportMCP      `toml:"mcp_servers"`
}

type settingsImportPayload struct {
	TOML string `json:"toml"`
}

func sanitizeLanguage(v string) string {
	lang := strings.TrimSpace(v)
	switch strings.ToLower(lang) {
	case "en":
		return "en"
	case "fr":
		return "fr"
	case "zh-tw":
		return "zh-TW"
	default:
		return "en"
	}
}

func sanitizeTheme(v string) string {
	theme := strings.ToLower(strings.TrimSpace(v))
	if theme == "" {
		return "dark"
	}
	if theme != "dark" {
		return "dark"
	}
	return theme
}

func normalizeGeneralSettings(in GeneralSettings) GeneralSettings {
	return GeneralSettings{
		Language:            sanitizeLanguage(in.Language),
		Theme:               sanitizeTheme(in.Theme),
		DefaultSystemPrompt: strings.TrimSpace(in.DefaultSystemPrompt),
	}
}

func (s *Service) getSettingValue(ctx context.Context, key string, fallback string) string {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM app_settings WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return fallback
	}
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (s *Service) setSettingValue(ctx context.Context, key string, value string) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO app_settings (key, value, updated_at)
		 VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = datetime('now')`,
		key,
		value,
	)
	return err
}

// GetGeneralSettings returns persisted general settings with safe defaults.
func (s *Service) GetGeneralSettings(ctx context.Context) (GeneralSettings, error) {
	settings := GeneralSettings{
		Language:            s.getSettingValue(ctx, "language", "en"),
		Theme:               s.getSettingValue(ctx, "theme", "dark"),
		DefaultSystemPrompt: s.getSettingValue(ctx, "default_system_prompt", ""),
	}
	return normalizeGeneralSettings(settings), nil
}

// UpdateGeneralSettings persists language/theme/default system prompt.
func (s *Service) UpdateGeneralSettings(ctx context.Context, payload GeneralSettings) (GeneralSettings, error) {
	normalized := normalizeGeneralSettings(payload)
	if err := s.setSettingValue(ctx, "language", normalized.Language); err != nil {
		return GeneralSettings{}, err
	}
	if err := s.setSettingValue(ctx, "theme", normalized.Theme); err != nil {
		return GeneralSettings{}, err
	}
	if err := s.setSettingValue(ctx, "default_system_prompt", normalized.DefaultSystemPrompt); err != nil {
		return GeneralSettings{}, err
	}
	return normalized, nil
}

// ExportConfiguration exports general settings + providers + MCP servers as TOML.
func (s *Service) ExportConfiguration(ctx context.Context) (string, error) {
	general, err := s.GetGeneralSettings(ctx)
	if err != nil {
		return "", err
	}

	providers := make([]settingsExportProvider, 0)
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, type, url, api_key, description, use_in_rag, active, temperature, num_ctx
		FROM providers ORDER BY id ASC
	`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var p settingsExportProvider
		var apiKey string
		var useInRAG, active int
		if err := rows.Scan(&p.Name, &p.Type, &p.URL, &apiKey, &p.Description, &useInRAG, &active, &p.Temperature, &p.NumCtx); err != nil {
			return "", err
		}
		if plain, _, decErr := decryptAPIKeyValue(apiKey); decErr == nil {
			p.APIKey = plain
		} else {
			p.APIKey = strings.TrimSpace(apiKey)
		}
		p.UseInRAG = useInRAG == 1
		p.Active = active == 1
		providers = append(providers, p)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	mcpServers := make([]settingsExportMCP, 0)
	mcpRows, err := s.db.QueryContext(ctx, `
		SELECT name, transport, url, command, args, active
		FROM mcp_servers ORDER BY id ASC
	`)
	if err != nil {
		return "", err
	}
	defer mcpRows.Close()
	for mcpRows.Next() {
		var item settingsExportMCP
		var active int
		if err := mcpRows.Scan(&item.Name, &item.Transport, &item.URL, &item.Command, &item.Args, &active); err != nil {
			return "", err
		}
		item.Active = active == 1
		mcpServers = append(mcpServers, item)
	}
	if err := mcpRows.Err(); err != nil {
		return "", err
	}

	payload := settingsExportPayload{
		General:   general,
		Providers: providers,
		MCP:       mcpServers,
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(payload); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ExportConfigurationToFile writes the TOML export to a local file and returns its path.
// Primary target is ~/Downloads; falls back to ~ if Downloads is unavailable.
func (s *Service) ExportConfigurationToFile(ctx context.Context) (map[string]any, error) {
	tomlText, err := s.ExportConfiguration(ctx)
	if err != nil {
		return nil, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	targetDir := filepath.Join(homeDir, "Downloads")
	if stat, statErr := os.Stat(targetDir); statErr != nil || !stat.IsDir() {
		targetDir = homeDir
	}

	fileName := fmt.Sprintf("liaotao-config-%s.toml", time.Now().Format("20060102-150405"))
	fullPath := filepath.Join(targetDir, fileName)
	if err := os.WriteFile(fullPath, []byte(tomlText), 0o600); err != nil {
		return nil, err
	}

	return map[string]any{
		"ok":       true,
		"path":     fullPath,
		"filename": fileName,
	}, nil
}

// ImportConfiguration imports TOML settings and upserts providers + MCP servers by name.
func (s *Service) ImportConfiguration(ctx context.Context, payload settingsImportPayload) (map[string]any, error) {
	raw := strings.TrimSpace(payload.TOML)
	if raw == "" {
		return nil, fmt.Errorf("toml content is required")
	}

	var parsed settingsExportPayload
	if _, err := toml.Decode(raw, &parsed); err != nil {
		return nil, fmt.Errorf("invalid toml: %w", err)
	}

	if _, err := s.UpdateGeneralSettings(ctx, parsed.General); err != nil {
		return nil, err
	}

	for _, p := range parsed.Providers {
		normalizedName := strings.TrimSpace(p.Name)
		if normalizedName == "" {
			continue
		}
		var existingID int64
		err := s.db.QueryRowContext(ctx, `SELECT id FROM providers WHERE name = ?`, normalizedName).Scan(&existingID)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if err == sql.ErrNoRows {
			if _, createErr := s.CreateProvider(ctx, CreateProviderPayload{
				Name:        normalizedName,
				Type:        strings.TrimSpace(p.Type),
				URL:         strings.TrimSpace(p.URL),
				APIKey:      strings.TrimSpace(p.APIKey),
				Description: strings.TrimSpace(p.Description),
				UseInRAG:    p.UseInRAG,
				Active:      p.Active,
				Temperature: p.Temperature,
				NumCtx:      p.NumCtx,
			}); createErr != nil {
				return nil, createErr
			}
			continue
		}
		if _, updateErr := s.UpdateProvider(ctx, UpdateProviderPayload{
			ID:          existingID,
			Name:        normalizedName,
			Type:        strings.TrimSpace(p.Type),
			URL:         strings.TrimSpace(p.URL),
			APIKey:      strings.TrimSpace(p.APIKey),
			Description: strings.TrimSpace(p.Description),
			UseInRAG:    p.UseInRAG,
			Active:      p.Active,
			Temperature: p.Temperature,
			NumCtx:      p.NumCtx,
		}); updateErr != nil {
			return nil, updateErr
		}
	}

	for _, m := range parsed.MCP {
		name := strings.TrimSpace(m.Name)
		if name == "" {
			continue
		}
		var existingID int64
		err := s.db.QueryRowContext(ctx, `SELECT id FROM mcp_servers WHERE name = ?`, name).Scan(&existingID)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		payload := SaveMCPServerPayload{
			ID:        existingID,
			Name:      name,
			Transport: strings.TrimSpace(m.Transport),
			URL:       strings.TrimSpace(m.URL),
			Command:   strings.TrimSpace(m.Command),
			Args:      strings.TrimSpace(m.Args),
			Active:    m.Active,
		}
		if _, saveErr := s.SaveMCPServer(ctx, payload); saveErr != nil {
			return nil, saveErr
		}
	}

	return map[string]any{"ok": true}, nil
}

// GetAboutInfo returns static metadata shown in the About section.
func (s *Service) GetAboutInfo(_ context.Context) (map[string]any, error) {
	version := "dev"
	if data, err := os.ReadFile("VERSION"); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			version = v
		}
	}
	return map[string]any{
		"name":        "liaotao",
		"version":     version,
		"description": "Desktop AI chat orchestrator with provider routing and MCP tools.",
		"links": map[string]string{
			"docs": "https://github.com/",
		},
		"credits": []string{
			"Wails v3",
			"SQLite (modernc)",
			"KaTeX",
			"Prism.js",
		},
	}, nil
}
