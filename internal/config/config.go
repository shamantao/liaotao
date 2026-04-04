// config.go — Configuration module
// Loads and merges TOML layers: default → user → project → env vars.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
)

// ---------------------------------------------------------------------------
// Schema

// AppConfig is the root configuration struct.
type AppConfig struct {
	App         AppSection         `toml:"app"`
	Config      ConfigSection      `toml:"config"`
	PathManager PathManagerSection `toml:"path_manager"`
	Database    DatabaseSection    `toml:"database"`
	Logger      LoggerSection      `toml:"logger"`
	Reporting   ReportingSection   `toml:"reporting"`
}

// AppSection holds general application metadata.
type AppSection struct {
	Name     string `toml:"name"`
	Version  string `toml:"version"`
	Mode     string `toml:"mode"`
	Language string `toml:"language"`
}

// ConfigSection holds config-system settings.
type ConfigSection struct {
	SchemaVersion      int  `toml:"schema_version"`
	EnableLayeredMerge bool `toml:"enable_layered_merge"`
	StrictMode         bool `toml:"strict_mode"`
}

// PathManagerSection holds path management rules.
type PathManagerSection struct {
	AllowedRoots      []string `toml:"allowed_roots"`
	TempDir           string   `toml:"temp_dir"`
	LogsDir           string   `toml:"logs_dir"`
	ReportsDir        string   `toml:"reports_dir"`
	CollisionStrategy string   `toml:"collision_strategy"`
	NormalizeUnicode  bool     `toml:"normalize_unicode"`
	TrimWhitespace    bool     `toml:"trim_whitespace"`
}

// DatabaseSection holds SQLite runtime parameters.
type DatabaseSection struct {
	Path        string `toml:"path"`
	BusyTimeout int    `toml:"busy_timeout_ms"`
	JournalMode string `toml:"journal_mode"`
	ForeignKeys bool   `toml:"foreign_keys"`
}

// LoggerSection holds logging parameters.
type LoggerSection struct {
	Level             string `toml:"level"`
	ConsolePretty     bool   `toml:"console_pretty"`
	FileJSON          bool   `toml:"file_json"`
	RotationEnabled   bool   `toml:"rotation_enabled"`
	MaxFileMB         int    `toml:"max_file_mb"`
	MaxFiles          int    `toml:"max_files"`
	IncludeContextIDs bool   `toml:"include_context_ids"`
}

// ReportingSection holds report output settings.
type ReportingSection struct {
	Enabled       bool `toml:"enabled"`
	JSONReport    bool `toml:"json_report"`
	CSVReport     bool `toml:"csv_report"`
	IncludeFailed bool `toml:"include_failed"`
}

// ---------------------------------------------------------------------------
// Loader

// Load reads and merges config from all layers (default → user → project → env).
func Load() (*AppConfig, error) {
	cfg := &AppConfig{}

	// Layer 1: bundled default (required)
	defaultPath := bundledDefaultPath()
	if err := loadTOML(defaultPath, cfg); err != nil {
		return nil, fmt.Errorf("default config: %w", err)
	}

	// Layer 2: user override (optional)
	if userPath, ok := userConfigPath(); ok {
		_ = loadTOML(userPath, cfg) // silently skip if missing
	}

	// Layer 3: project-local override (optional)
	_ = loadTOML("config/project.toml", cfg)

	// Layer 4: environment variables (APP__ prefix)
	applyEnvOverrides(cfg)

	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadTOML(path string, cfg *AppConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = toml.Decode(string(data), cfg)
	return err
}

func bundledDefaultPath() string {
	// Try next to executable first (production)
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "config", "default.toml")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Fallback: relative to CWD (development)
	return filepath.Join("config", "default.toml")
}

func userConfigPath() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	p := filepath.Join(home, ".config", "liaotao", "user.toml")
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return p, false
}

// applyEnvOverrides maps APP__SECTION__KEY env vars onto the config struct.
func applyEnvOverrides(cfg *AppConfig) {
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "APP__") {
			continue
		}
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimPrefix(parts[0], "APP__")
		val := parts[1]
		applyEnvValue(cfg, key, val)
	}
}

func applyEnvValue(cfg *AppConfig, key, val string) {
	segments := strings.SplitN(strings.ToLower(key), "__", 2)
	if len(segments) != 2 {
		return
	}
	section, field := segments[0], segments[1]

	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("toml")
		if tag == section {
			sv := v.Field(i)
			st := sv.Type()
			for j := 0; j < st.NumField(); j++ {
				ftag := st.Field(j).Tag.Get("toml")
				if ftag == field {
					fv := sv.Field(j)
					if fv.CanSet() && fv.Kind() == reflect.String {
						fv.SetString(val)
					}
				}
			}
		}
	}
}

func validate(cfg *AppConfig) error {
	if cfg.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}
	if cfg.App.Mode != "debug" && cfg.App.Mode != "normal" {
		return fmt.Errorf("app.mode must be 'debug' or 'normal', got '%s'", cfg.App.Mode)
	}
	if cfg.Database.Path == "" {
		return fmt.Errorf("database.path is required")
	}
	if cfg.Database.BusyTimeout < 0 {
		return fmt.Errorf("database.busy_timeout_ms must be >= 0")
	}
	return nil
}
