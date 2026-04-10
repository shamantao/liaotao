/*
  conversation_test.go -- Regression tests for conversation CRUD (DEBT-01).
  Covers DEBT-01: provider_id must be stored and retrieved as INTEGER FK,
  never as a plain text string.
*/

package bindings

import (
	"context"
	"database/sql"
	"testing"

	"liaotao/internal/db"

	_ "modernc.org/sqlite"
)

// newConversationTestDB creates an in-memory DB with the full v1 schema applied.
func newConversationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.ApplySchemaForTest(database); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return database
}

// TestConversation_ProviderIDStoredAsInteger verifies that CreateConversation
// stores a numeric provider_id FK and that ListConversations returns the
// resolved provider name — not a raw string.
// Regression for DEBT-01: previously conversations.js sent prov.name (string).
func TestConversation_ProviderIDStoredAsInteger(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	// Create a provider to reference.
	prov, err := svc.CreateProvider(ctx, CreateProviderPayload{
		Name:   "TestGroq",
		Type:   "openai-compatible",
		URL:    "https://api.groq.com",
		Active: true,
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	// Create a conversation using the numeric provider ID (as the JS now sends).
	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "DEBT-01 test",
		ProviderID: prov.ID,
		Model:      "llama-3.3-70b",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	if conv.ProviderID != prov.ID {
		t.Errorf("ProviderID = %d, want %d", conv.ProviderID, prov.ID)
	}
	if conv.Provider != "TestGroq" {
		t.Errorf("Provider name = %q, want \"TestGroq\"", conv.Provider)
	}

	// Verify via ListConversations that the JOIN resolves the name correctly.
	list, err := svc.ListConversations(ctx, ListConversationsPayload{Limit: 10})
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(list))
	}
	got := list[0]
	if got.ProviderID != prov.ID {
		t.Errorf("listed ProviderID = %d, want %d", got.ProviderID, prov.ID)
	}
	if got.Provider != "TestGroq" {
		t.Errorf("listed Provider = %q, want \"TestGroq\"", got.Provider)
	}
}

// TestConversation_NoProvider verifies that a conversation created without a
// provider stores NULL (provider_id = 0) and does not error.
func TestConversation_NoProvider(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "anonymous chat",
		ProviderID: 0, // no provider
		Model:      "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("CreateConversation with no provider: %v", err)
	}
	if conv.ProviderID != 0 {
		t.Errorf("expected ProviderID 0 (no provider), got %d", conv.ProviderID)
	}
	if conv.Provider != "" {
		t.Errorf("expected empty Provider name, got %q", conv.Provider)
	}
}

// TestConversation_ProviderDeletedSetNullFK verifies ON DELETE SET NULL:
// deleting a provider must not delete the conversation, only clear provider_id.
func TestConversation_ProviderDeletedSetNullFK(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	prov, err := svc.CreateProvider(ctx, CreateProviderPayload{
		Name:   "TempProvider",
		Type:   "openai-compatible",
		URL:    "http://localhost",
		Active: true,
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "orphan test",
		ProviderID: prov.ID,
		Model:      "test-model",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// Delete the provider.
	if _, err := svc.DeleteProvider(ctx, DeleteProviderPayload{ID: prov.ID}); err != nil {
		t.Fatalf("DeleteProvider: %v", err)
	}

	// The conversation must still exist with provider_id = 0 (NULL).
	list, err := svc.ListConversations(ctx, ListConversationsPayload{Limit: 10})
	if err != nil {
		t.Fatalf("ListConversations after delete: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected conversation to survive provider deletion, got %d conversations", len(list))
	}
	if list[0].ID != conv.ID {
		t.Errorf("wrong conversation returned: %d", list[0].ID)
	}
	if list[0].ProviderID != 0 {
		t.Errorf("expected ProviderID 0 after provider deletion, got %d", list[0].ProviderID)
	}
}
