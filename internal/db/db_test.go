/*
  db_test.go -- Regression tests for SQLite migrations.
  Ensures legacy schemas can be upgraded without startup failures.
*/

package db

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestDB_MigrateLegacyConversationsWithoutProjectID(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	legacy := []string{
		`CREATE TABLE IF NOT EXISTS providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			provider_id INTEGER,
			model TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
	}
	for _, stmt := range legacy {
		if _, err := database.Exec(stmt); err != nil {
			t.Fatalf("seed legacy schema: %v", err)
		}
	}

	if err := ApplySchemaForTest(database); err != nil {
		t.Fatalf("ApplySchemaForTest failed on legacy schema: %v", err)
	}

	var hasProjectID int
	if err := database.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('conversations') WHERE name='project_id'`).Scan(&hasProjectID); err != nil {
		t.Fatalf("check project_id column: %v", err)
	}
	if hasProjectID != 1 {
		t.Fatalf("expected conversations.project_id to exist after migration")
	}

	var hasIndex int
	if err := database.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_conversations_project_id'`).Scan(&hasIndex); err != nil {
		t.Fatalf("check idx_conversations_project_id: %v", err)
	}
	if hasIndex != 1 {
		t.Fatalf("expected idx_conversations_project_id to exist after migration")
	}
}
