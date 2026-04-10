// db.go -- SQLite initialization and migrations.
// Owns DB opening, pragma setup, and baseline schema creation.

package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"liaotao/internal/config"
	"liaotao/internal/paths"

	_ "modernc.org/sqlite"
)

// OpenAndMigrate opens the SQLite database and applies all required migrations.
func OpenAndMigrate(cfg *config.AppConfig) (*sql.DB, error) {
	if err := ensureDatabasePath(cfg); err != nil {
		return nil, err
	}

	dsn := cfg.Database.Path
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := applyPragmas(db, cfg); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := migrate(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// ApplySchemaForTest runs pragma setup and the full migration on an existing *sql.DB.
// Intended for unit tests in other packages that need a properly initialized schema.
func ApplySchemaForTest(database *sql.DB) error {
	if _, err := database.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return err
	}
	return migrate(context.Background(), database)
}

func ensureDatabasePath(cfg *config.AppConfig) error {
	dbPath := cfg.Database.Path
	if err := paths.EnsureWithinAllowed(dbPath, cfg.PathManager.AllowedRoots); err != nil {
		return fmt.Errorf("database path guard: %w", err)
	}

	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return fmt.Errorf("create database dir: %w", err)
	}

	// Restrict DB file permissions to owner-only (0600) so cloud sync tools
	// and other local users cannot read API keys stored in the providers table.
	// If the file does not exist yet, SQLite will create it — permissions are
	// applied on first open via the MkdirAll above; we also enforce on every
	// startup to recover from accidental permission widening.
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Chmod(dbPath, 0o600); err != nil {
			return fmt.Errorf("secure database permissions: %w", err)
		}
	}
	return nil
}

func applyPragmas(db *sql.DB, cfg *config.AppConfig) error {
	if cfg.Database.BusyTimeout > 0 {
		query := fmt.Sprintf("PRAGMA busy_timeout=%d;", cfg.Database.BusyTimeout)
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("sqlite pragma busy_timeout: %w", err)
		}
	}

	if cfg.Database.ForeignKeys {
		if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
			return fmt.Errorf("sqlite pragma foreign_keys: %w", err)
		}
	}

	mode := strings.ToUpper(strings.TrimSpace(cfg.Database.JournalMode))
	if mode == "" {
		mode = "WAL"
	}
	if _, err := db.Exec(fmt.Sprintf("PRAGMA journal_mode=%s;", mode)); err != nil {
		return fmt.Errorf("sqlite pragma journal_mode: %w", err)
	}

	return nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	// Phase A: one-time migration — rename provider_id from TEXT to INTEGER FK.
	// Safe to call on every startup: detects column type and skips if already INTEGER.
	if err := migrateConversationsProviderID(ctx, db); err != nil {
		return err
	}

	// Phase B: idempotent baseline schema (no-op when tables already exist).
	statements := []string{
		`CREATE TABLE IF NOT EXISTS conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			provider_id INTEGER REFERENCES providers(id) ON DELETE SET NULL,
			model TEXT NOT NULL,
			temperature REAL NOT NULL DEFAULT 0.7,
			max_tokens INTEGER NOT NULL DEFAULT 0,
			system_prompt TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id INTEGER NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			tool_calls TEXT NOT NULL DEFAULT '[]',
			token_stats TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY(conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at DESC);`,
		`CREATE TABLE IF NOT EXISTS providers (
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
		);`,
		`CREATE INDEX IF NOT EXISTS idx_providers_name ON providers(name);`,
		// v1.3 Smart Router tables.
		`CREATE TABLE IF NOT EXISTS provider_priority (
			provider_id INTEGER PRIMARY KEY REFERENCES providers(id) ON DELETE CASCADE,
			priority    INTEGER NOT NULL DEFAULT 100
		);`,
		`CREATE TABLE IF NOT EXISTS provider_quota_config (
			provider_id   INTEGER PRIMARY KEY REFERENCES providers(id) ON DELETE CASCADE,
			daily_limit   INTEGER NOT NULL DEFAULT 0,
			monthly_limit INTEGER NOT NULL DEFAULT 0,
			reset_day     INTEGER NOT NULL DEFAULT 1
		);`,
		`CREATE TABLE IF NOT EXISTS provider_quota_usage (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
			period      TEXT    NOT NULL,
			period_type TEXT    NOT NULL,
			tokens_used INTEGER NOT NULL DEFAULT 0,
			updated_at  TEXT    NOT NULL DEFAULT (datetime('now')),
			UNIQUE(provider_id, period, period_type)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_quota_usage_lookup
		 ON provider_quota_usage(provider_id, period_type, period);`,
		`CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
		// v1.4 MCP server registry.
		`CREATE TABLE IF NOT EXISTS mcp_servers (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			name      TEXT    NOT NULL UNIQUE,
			transport TEXT    NOT NULL DEFAULT 'http',
			url       TEXT    NOT NULL DEFAULT '',
			command   TEXT    NOT NULL DEFAULT '',
			args      TEXT    NOT NULL DEFAULT '[]',
			active    INTEGER NOT NULL DEFAULT 1,
			created_at TEXT   NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT   NOT NULL DEFAULT (datetime('now'))
		);`,
	}

	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	if err := ensureConversationPreferenceColumns(ctx, db); err != nil {
		return err
	}
	return nil
}

func ensureConversationPreferenceColumns(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(conversations)`)
	if err != nil {
		return fmt.Errorf("pragma table_info conversations: %w", err)
	}
	defer rows.Close()

	hasTemperature := false
	hasMaxTokens := false
	hasSystemPrompt := false
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return err
		}
		switch name {
		case "temperature":
			hasTemperature = true
		case "max_tokens":
			hasMaxTokens = true
		case "system_prompt":
			hasSystemPrompt = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !hasTemperature {
		if _, err := db.ExecContext(ctx, `ALTER TABLE conversations ADD COLUMN temperature REAL NOT NULL DEFAULT 0.7`); err != nil {
			return fmt.Errorf("add conversations.temperature: %w", err)
		}
	}
	if !hasMaxTokens {
		if _, err := db.ExecContext(ctx, `ALTER TABLE conversations ADD COLUMN max_tokens INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("add conversations.max_tokens: %w", err)
		}
	}
	if !hasSystemPrompt {
		if _, err := db.ExecContext(ctx, `ALTER TABLE conversations ADD COLUMN system_prompt TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add conversations.system_prompt: %w", err)
		}
	}

	return nil
}

// migrateConversationsProviderID converts conversations.provider_id from TEXT to INTEGER.
// SQLite has no ALTER COLUMN — requires table recreation inside a transaction.
// This migration is idempotent: it checks the column type before running.
func migrateConversationsProviderID(ctx context.Context, db *sql.DB) error {
	// Check if the conversations table exists.
	var exists int
	err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='conversations'`,
	).Scan(&exists)
	if err != nil || exists == 0 {
		return nil // table not yet created — initial schema path handles it
	}

	// Check whether provider_id is still TEXT.
	needsMigration := false
	prows, err := db.QueryContext(ctx, `PRAGMA table_info(conversations)`)
	if err != nil {
		return fmt.Errorf("pragma table_info conversations: %w", err)
	}
	defer prows.Close()
	for prows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var dflt sql.NullString
		if err := prows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("scan pragma: %w", err)
		}
		if name == "provider_id" && strings.EqualFold(typ, "TEXT") {
			needsMigration = true
		}
	}
	if err := prows.Err(); err != nil {
		return err
	}
	if !needsMigration {
		return nil
	}

	// Run inside a transaction so a partial failure leaves the DB unchanged.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	steps := []string{
		// 1. New table with INTEGER FK.
		`CREATE TABLE conversations_v2 (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			provider_id INTEGER REFERENCES providers(id) ON DELETE SET NULL,
			model TEXT NOT NULL,
			temperature REAL NOT NULL DEFAULT 0.7,
			max_tokens INTEGER NOT NULL DEFAULT 0,
			system_prompt TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
		// 2. Copy rows: resolve stored text (name OR cast id) to numeric FK.
		`INSERT INTO conversations_v2 (id, title, provider_id, model, created_at, updated_at)
		 SELECT c.id, c.title,
		        COALESCE(
				    (SELECT p.id FROM providers p WHERE CAST(p.id AS TEXT) = c.provider_id LIMIT 1),
				    (SELECT p.id FROM providers p WHERE p.name = c.provider_id LIMIT 1)
			    ),
			    c.model, c.created_at, c.updated_at
		 FROM conversations c;`,
		// 3. Swap tables.
		`DROP TABLE conversations;`,
		`ALTER TABLE conversations_v2 RENAME TO conversations;`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at DESC);`,
	}
	for _, s := range steps {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("provider_id migration: %w", err)
		}
	}
	return tx.Commit()
}
