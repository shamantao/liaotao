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

func ensureDatabasePath(cfg *config.AppConfig) error {
	dbPath := cfg.Database.Path
	if err := paths.EnsureWithinAllowed(dbPath, cfg.PathManager.AllowedRoots); err != nil {
		return fmt.Errorf("database path guard: %w", err)
	}

	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return fmt.Errorf("create database dir: %w", err)
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
	statements := []string{
		`CREATE TABLE IF NOT EXISTS conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			model TEXT NOT NULL,
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
	}

	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}
