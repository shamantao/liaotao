/*
 * SqliteSchema.kt - schema definitions and migrations for Liaotao persistence.
 * Responsibilities: declare versioned SQL migrations and expose migration
 * metadata consumed by the runtime migration engine.
 */

package io.liaotao.persistence.sqlite

internal data class SqlMigration(
    val version: Int,
    val name: String,
    val statements: List<String>,
)

internal object SqliteSchema {
    val migrations: List<SqlMigration> = listOf(
        SqlMigration(
            version = 1,
            name = "init_core_schema",
            statements = listOf(
                """
                CREATE TABLE IF NOT EXISTS projects (
                    id TEXT PRIMARY KEY,
                    name TEXT NOT NULL,
                    description TEXT NOT NULL DEFAULT '',
                    created_at TEXT NOT NULL,
                    updated_at TEXT NOT NULL
                )
                """.trimIndent(),
                """
                CREATE TABLE IF NOT EXISTS conversations (
                    id TEXT PRIMARY KEY,
                    project_id TEXT NOT NULL,
                    title TEXT NOT NULL,
                    source TEXT NOT NULL,
                    model TEXT NOT NULL,
                    created_at TEXT NOT NULL,
                    updated_at TEXT NOT NULL,
                    last_activity_at TEXT NOT NULL,
                    archived_at TEXT,
                    FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
                )
                """.trimIndent(),
                """
                CREATE TABLE IF NOT EXISTS messages (
                    id TEXT PRIMARY KEY,
                    conversation_id TEXT NOT NULL,
                    role TEXT NOT NULL,
                    content TEXT NOT NULL,
                    created_at TEXT NOT NULL,
                    FOREIGN KEY(conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
                )
                """.trimIndent(),
                """
                CREATE TABLE IF NOT EXISTS connector_instances (
                    id TEXT PRIMARY KEY,
                    connector_type TEXT NOT NULL,
                    display_name TEXT NOT NULL,
                    base_url TEXT,
                    is_enabled INTEGER NOT NULL DEFAULT 1,
                    created_at TEXT NOT NULL,
                    updated_at TEXT NOT NULL
                )
                """.trimIndent(),
                """
                CREATE TABLE IF NOT EXISTS mcp_servers (
                    id TEXT PRIMARY KEY,
                    name TEXT NOT NULL,
                    url TEXT NOT NULL,
                    auth_ref TEXT,
                    is_enabled INTEGER NOT NULL DEFAULT 1,
                    created_at TEXT NOT NULL,
                    updated_at TEXT NOT NULL
                )
                """.trimIndent(),
                """
                CREATE TABLE IF NOT EXISTS execution_runs (
                    id TEXT PRIMARY KEY,
                    conversation_id TEXT,
                    connector_instance_id TEXT,
                    status TEXT NOT NULL,
                    started_at TEXT NOT NULL,
                    finished_at TEXT,
                    error_message TEXT,
                    FOREIGN KEY(conversation_id) REFERENCES conversations(id) ON DELETE SET NULL,
                    FOREIGN KEY(connector_instance_id) REFERENCES connector_instances(id) ON DELETE SET NULL
                )
                """.trimIndent(),
                "CREATE INDEX IF NOT EXISTS idx_projects_updated_at ON projects(updated_at DESC)",
                "CREATE INDEX IF NOT EXISTS idx_conversations_project_archived ON conversations(project_id, archived_at)",
                "CREATE INDEX IF NOT EXISTS idx_conversations_project_last_activity ON conversations(project_id, last_activity_at DESC)",
                "CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id)",
                "CREATE INDEX IF NOT EXISTS idx_execution_runs_started_at ON execution_runs(started_at DESC)",
                "CREATE VIRTUAL TABLE IF NOT EXISTS message_fts USING fts5(message_id UNINDEXED, conversation_id UNINDEXED, content)",
                """
                CREATE TRIGGER IF NOT EXISTS trg_messages_ai AFTER INSERT ON messages BEGIN
                  INSERT INTO message_fts(message_id, conversation_id, content)
                  VALUES (new.id, new.conversation_id, new.content);
                END
                """.trimIndent(),
                """
                CREATE TRIGGER IF NOT EXISTS trg_messages_ad AFTER DELETE ON messages BEGIN
                  DELETE FROM message_fts WHERE message_id = old.id;
                END
                """.trimIndent(),
                """
                CREATE TRIGGER IF NOT EXISTS trg_messages_au AFTER UPDATE ON messages BEGIN
                  UPDATE message_fts
                  SET conversation_id = new.conversation_id,
                      content = new.content
                  WHERE message_id = old.id;
                END
                """.trimIndent(),
            ),
        ),
        SqlMigration(
            version = 2,
            name = "connector_settings_columns_v2",
            statements = listOf(
                "ALTER TABLE connector_instances ADD COLUMN default_model TEXT NOT NULL DEFAULT ''",
                "ALTER TABLE connector_instances ADD COLUMN secret_ref TEXT",
                "ALTER TABLE connector_instances ADD COLUMN connection_health TEXT NOT NULL DEFAULT 'UNKNOWN'",
                "ALTER TABLE connector_instances ADD COLUMN connection_message TEXT NOT NULL DEFAULT 'Not checked'",
            ),
        ),
    )
}