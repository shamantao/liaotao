/*
  connection_test.go -- Unit tests for TestConnection binding (PROV-05).
  Uses httptest servers to simulate reachable and unreachable providers.
*/

package bindings

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

// newConnectionTestService creates an in-memory Service and inserts a single provider
// pointing to the given base URL (OpenAI-compatible type).
func newConnectionTestService(t *testing.T, baseURL string) (*Service, int64) {
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

	svc := NewService(db)
	ctx := context.Background()

	p, err := svc.CreateProvider(ctx, CreateProviderPayload{
		Name:   "Test Provider",
		Type:   "openai-compatible",
		URL:    baseURL,
		APIKey: "test-key",
		Active: true,
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	return svc, p.ID
}

// TestTestConnection_OK verifies that a reachable provider returns OK=true
// with the correct model count and a non-negative latency.
func TestTestConnection_OK(t *testing.T) {
	// Fake OpenAI-compatible /models endpoint.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "list",
				"data": []map[string]any{
					{"id": "model-a", "object": "model"},
					{"id": "model-b", "object": "model"},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	svc, provID := newConnectionTestService(t, srv.URL)

	result, err := svc.TestConnection(context.Background(), TestConnectionPayload{ProviderID: provID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Errorf("expected OK=true, got Error=%q", result.Error)
	}
	if result.ModelCount != 2 {
		t.Errorf("expected ModelCount=2, got %d", result.ModelCount)
	}
	if result.LatencyMs < 0 {
		t.Errorf("expected non-negative latency, got %d", result.LatencyMs)
	}
	if result.Error != "" {
		t.Errorf("expected no error string, got %q", result.Error)
	}
}

// TestTestConnection_Unreachable verifies that a closed server returns OK=false
// with a non-empty error message.
func TestTestConnection_Unreachable(t *testing.T) {
	// Create server, then immediately close it so no connections can be established.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseURL := srv.URL
	srv.Close()

	svc, provID := newConnectionTestService(t, baseURL)

	result, err := svc.TestConnection(context.Background(), TestConnectionPayload{ProviderID: provID})
	if err != nil {
		t.Fatalf("unexpected binding error: %v", err)
	}
	if result.OK {
		t.Error("expected OK=false for unreachable server")
	}
	if result.Error == "" {
		t.Error("expected a non-empty Error message for unreachable server")
	}
}

// TestTestConnection_UnknownProvider verifies that an invalid provider_id
// returns a safe error message and does not panic.
func TestTestConnection_UnknownProvider(t *testing.T) {
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
	)`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}
	svc := NewService(db)

	result, err := svc.TestConnection(context.Background(), TestConnectionPayload{ProviderID: 9999})
	if err != nil {
		t.Fatalf("unexpected binding error: %v", err)
	}
	if result.OK {
		t.Error("expected OK=false for unknown provider")
	}
	if result.Error == "" {
		t.Error("expected a non-empty Error message for unknown provider")
	}
}
