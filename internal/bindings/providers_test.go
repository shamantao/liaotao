/*
  providers_test.go -- Unit tests for provider CRUD persistence and filters.
  Uses in-memory SQLite schema dedicated to providers binding tests.
*/

package bindings

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func newProvidersTestService(t *testing.T) *Service {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE providers (
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
	);`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return NewService(db)
}

func TestProviderCRUD(t *testing.T) {
	svc := newProvidersTestService(t)
	ctx := context.Background()

	created, err := svc.CreateProvider(ctx, CreateProviderPayload{
		Name:        "Ollama Local",
		Type:        "ollama",
		URL:         "http://localhost:11434/v1",
		APIKey:      "",
		Description: "local runtime",
		UseInRAG:    true,
		Active:      true,
		Temperature: 0.8,
		NumCtx:      4096,
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	if created.ID <= 0 {
		t.Fatalf("expected positive id, got %d", created.ID)
	}
	if created.Name != "Ollama Local" || created.Type != "ollama" {
		t.Fatalf("unexpected created provider: %+v", created)
	}

	list, err := svc.ListProviders(ctx, ListProvidersPayload{})
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(list))
	}

	updated, err := svc.UpdateProvider(ctx, UpdateProviderPayload{
		ID:          created.ID,
		Name:        "Ollama Local",
		Type:        "ollama",
		URL:         "http://localhost:11434/v1",
		Description: "updated",
		UseInRAG:    false,
		Active:      false,
		Temperature: 1.0,
		NumCtx:      8192,
	})
	if err != nil {
		t.Fatalf("UpdateProvider: %v", err)
	}
	if updated.Active {
		t.Fatalf("expected inactive provider after update")
	}
	if updated.NumCtx != 8192 {
		t.Fatalf("expected num_ctx 8192, got %d", updated.NumCtx)
	}

	activeOnly, err := svc.ListProviders(ctx, ListProvidersPayload{ActiveOnly: true})
	if err != nil {
		t.Fatalf("ListProviders(active_only): %v", err)
	}
	if len(activeOnly) != 0 {
		t.Fatalf("expected 0 active providers, got %d", len(activeOnly))
	}

	deleted, err := svc.DeleteProvider(ctx, DeleteProviderPayload{ID: created.ID})
	if err != nil {
		t.Fatalf("DeleteProvider: %v", err)
	}
	if ok, _ := deleted["ok"].(bool); !ok {
		t.Fatalf("expected delete ok=true, got %+v", deleted)
	}

	afterDelete, err := svc.ListProviders(ctx, ListProvidersPayload{})
	if err != nil {
		t.Fatalf("ListProviders after delete: %v", err)
	}
	if len(afterDelete) != 0 {
		t.Fatalf("expected empty list after delete, got %d", len(afterDelete))
	}
}

func TestProviderNameMustBeUnique(t *testing.T) {
	svc := newProvidersTestService(t)
	ctx := context.Background()

	_, err := svc.CreateProvider(ctx, CreateProviderPayload{Name: "Main", Type: "openai-compatible"})
	if err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}
	_, err = svc.CreateProvider(ctx, CreateProviderPayload{Name: "Main", Type: "openai-compatible"})
	if err == nil {
		t.Fatalf("expected duplicate name error, got nil")
	}
}

func TestProviderPersistenceAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "providers-test.db")

	open := func() (*Service, func()) {
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			t.Fatalf("open sqlite file: %v", err)
		}
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS providers (
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
		);`)
		if err != nil {
			t.Fatalf("create schema file db: %v", err)
		}
		return NewService(db), func() { _ = db.Close() }
	}

	ctx := context.Background()

	firstSvc, closeFirst := open()
	created, err := firstSvc.CreateProvider(ctx, CreateProviderPayload{Name: "Persistent", Type: "ollama", Active: true})
	if err != nil {
		closeFirst()
		t.Fatalf("CreateProvider: %v", err)
	}
	closeFirst()

	secondSvc, closeSecond := open()
	defer closeSecond()

	list, err := secondSvc.ListProviders(ctx, ListProvidersPayload{})
	if err != nil {
		t.Fatalf("ListProviders after reopen: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 provider after reopen, got %d", len(list))
	}
	if list[0].ID != created.ID || list[0].Name != "Persistent" {
		t.Fatalf("unexpected provider after reopen: %+v", list[0])
	}
}

// TestAPIKeyNotSerialisedToFrontend verifies PROV-04: the API key must never appear
// in the JSON representation returned to the frontend (api_key masked, api_key_set visible).
func TestAPIKeyNotSerialisedToFrontend(t *testing.T) {
	svc := newProvidersTestService(t)
	ctx := context.Background()

	_, err := svc.CreateProvider(ctx, CreateProviderPayload{
		Name:   "Secure Provider",
		Type:   "openai-compatible",
		APIKey: "sk-supersecret",
		Active: true,
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	list, err := svc.ListProviders(ctx, ListProvidersPayload{})
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("expected at least one provider")
	}

	rec := list[0]

	// In-memory: APIKeySet must reflect that a key is stored.
	if !rec.APIKeySet {
		t.Error("expected APIKeySet=true, got false")
	}

	// JSON: "api_key" must not appear in the encoded output.
	encoded, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if _, exists := raw["api_key"]; exists {
		t.Error("api_key must not appear in JSON output (should be masked with json:\"-\")")
	}
	if v, ok := raw["api_key_set"].(bool); !ok || !v {
		t.Errorf("api_key_set must be true in JSON, got %v", raw["api_key_set"])
	}
}

// TestUpdateProviderPreservesKeyWhenEmpty verifies PROV-04: updating a provider
// with an empty APIKey in the payload must not clear the existing stored key.
func TestUpdateProviderPreservesKeyWhenEmpty(t *testing.T) {
	svc := newProvidersTestService(t)
	ctx := context.Background()

	created, err := svc.CreateProvider(ctx, CreateProviderPayload{
		Name:   "Key Preservation Test",
		Type:   "openai-compatible",
		APIKey: "original-key",
		URL:    "http://example.com/v1",
		Active: true,
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	// Update with blank APIKey — must NOT wipe the original key.
	_, err = svc.UpdateProvider(ctx, UpdateProviderPayload{
		ID:     created.ID,
		Name:   "Key Preservation Test",
		Type:   "openai-compatible",
		URL:    "http://example.com/v1",
		APIKey: "",
		Active: true,
	})
	if err != nil {
		t.Fatalf("UpdateProvider: %v", err)
	}

	// Retrieve credentials directly — the raw key must still be "original-key".
	cfg, err := svc.getProviderCredentials(ctx, created.ID)
	if err != nil {
		t.Fatalf("getProviderCredentials: %v", err)
	}
	if cfg.APIKey != "original-key" {
		t.Errorf("expected key to be preserved as %q, got %q", "original-key", cfg.APIKey)
	}
}

// TestProviderAPIKeyStoredEncrypted verifies SET-07 behavior:
// provider API keys are encrypted before being persisted in SQLite.
func TestProviderAPIKeyStoredEncrypted(t *testing.T) {
	_ = os.Setenv("LIAOTAO_MASTER_KEY", "test-master-key")
	t.Cleanup(func() {
		_ = os.Unsetenv("LIAOTAO_MASTER_KEY")
	})

	svc := newProvidersTestService(t)
	ctx := context.Background()

	created, err := svc.CreateProvider(ctx, CreateProviderPayload{
		Name:   "Encrypted Provider",
		Type:   "openai-compatible",
		APIKey: "sk-plaintext-secret",
		Active: true,
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	var raw string
	err = svc.db.QueryRowContext(ctx, `SELECT api_key FROM providers WHERE id=?`, created.ID).Scan(&raw)
	if err != nil {
		t.Fatalf("query api_key: %v", err)
	}
	if raw == "sk-plaintext-secret" {
		t.Fatalf("api key must not be stored in clear text")
	}
	if !strings.HasPrefix(raw, encryptedAPIKeyPrefix) {
		t.Fatalf("api key must be encrypted, got %q", raw)
	}

	cfg, err := svc.getProviderCredentials(ctx, created.ID)
	if err != nil {
		t.Fatalf("getProviderCredentials: %v", err)
	}
	if cfg.APIKey != "sk-plaintext-secret" {
		t.Fatalf("decrypted key mismatch: got %q", cfg.APIKey)
	}
}
