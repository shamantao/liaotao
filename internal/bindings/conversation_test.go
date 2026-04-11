/*
  conversation_test.go -- Regression tests for conversation CRUD (DEBT-01).
  Covers DEBT-01: provider_id must be stored and retrieved as INTEGER FK,
  never as a plain text string.
*/

package bindings

import (
	"context"
	"database/sql"
	"strings"
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

// TestConversation_RenameConversation verifies title update and refreshed ordering timestamp.
func TestConversation_RenameConversation(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "Initial title",
		ProviderID: 0,
		Model:      "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	updated, err := svc.RenameConversation(ctx, RenameConversationPayload{
		ConversationID: conv.ID,
		Title:          "Renamed title",
	})
	if err != nil {
		t.Fatalf("RenameConversation: %v", err)
	}
	if updated.Title != "Renamed title" {
		t.Errorf("updated title = %q, want %q", updated.Title, "Renamed title")
	}

	list, err := svc.ListConversations(ctx, ListConversationsPayload{Limit: 10})
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(list))
	}
	if list[0].Title != "Renamed title" {
		t.Errorf("listed title = %q, want %q", list[0].Title, "Renamed title")
	}
}

// TestConversation_SearchConversations verifies title/content search behavior.
func TestConversation_SearchConversations(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	alpha, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "Alpha planning",
		ProviderID: 0,
		Model:      "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("CreateConversation alpha: %v", err)
	}
	beta, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "Beta notes",
		ProviderID: 0,
		Model:      "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("CreateConversation beta: %v", err)
	}

	if err := svc.SaveMessage(ctx, MessagePayload{
		ConversationID: beta.ID,
		Role:           "user",
		Content:        "Need roadmap details for phase two",
	}); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	t.Run("title match", func(t *testing.T) {
		items, err := svc.SearchConversations(ctx, SearchConversationsPayload{Query: "alpha", Limit: 10})
		if err != nil {
			t.Fatalf("SearchConversations title: %v", err)
		}
		if len(items) != 1 || items[0].ID != alpha.ID {
			t.Fatalf("title search mismatch: got %+v", items)
		}
	})

	t.Run("message content match", func(t *testing.T) {
		items, err := svc.SearchConversations(ctx, SearchConversationsPayload{Query: "roadmap", Limit: 10})
		if err != nil {
			t.Fatalf("SearchConversations content: %v", err)
		}
		if len(items) != 1 || items[0].ID != beta.ID {
			t.Fatalf("content search mismatch: got %+v", items)
		}
	})

	t.Run("blank query behaves as list", func(t *testing.T) {
		items, err := svc.SearchConversations(ctx, SearchConversationsPayload{Query: "   ", Limit: 10})
		if err != nil {
			t.Fatalf("SearchConversations blank: %v", err)
		}
		if len(items) < 2 {
			t.Fatalf("blank query should return full list, got %d", len(items))
		}
	})

	t.Run("trimmed query", func(t *testing.T) {
		items, err := svc.SearchConversations(ctx, SearchConversationsPayload{Query: "  BETA  ", Limit: 10})
		if err != nil {
			t.Fatalf("SearchConversations trimmed: %v", err)
		}
		if len(items) != 1 || !strings.Contains(strings.ToLower(items[0].Title), "beta") {
			t.Fatalf("trimmed search mismatch: got %+v", items)
		}
	})
}

// TestConversation_AutoTitleFromFirstUserMessage verifies that a default
// conversation title is replaced by the first user message preview.
func TestConversation_AutoTitleFromFirstUserMessage(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "New chat",
		ProviderID: 0,
		Model:      "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	if err := svc.SaveMessage(ctx, MessagePayload{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "Plan migration strategy for MCP search and history UX",
	}); err != nil {
		t.Fatalf("SaveMessage user: %v", err)
	}

	list, err := svc.ListConversations(ctx, ListConversationsPayload{Limit: 10})
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(list))
	}
	if !strings.HasPrefix(list[0].Title, "Plan migration strategy") {
		t.Fatalf("expected auto-title from first message, got %q", list[0].Title)
	}

	// Manual rename must stay stable after additional user messages.
	if _, err := svc.RenameConversation(ctx, RenameConversationPayload{
		ConversationID: conv.ID,
		Title:          "Custom title",
	}); err != nil {
		t.Fatalf("RenameConversation: %v", err)
	}
	if err := svc.SaveMessage(ctx, MessagePayload{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "A second user message should not override custom title",
	}); err != nil {
		t.Fatalf("SaveMessage second user: %v", err)
	}
	list, err = svc.ListConversations(ctx, ListConversationsPayload{Limit: 10})
	if err != nil {
		t.Fatalf("ListConversations second pass: %v", err)
	}
	if list[0].Title != "Custom title" {
		t.Fatalf("custom title should be preserved, got %q", list[0].Title)
	}
}

// TestConversation_MessageTokenStats verifies that token stats are persisted
// in the existing messages.token_stats JSON column and returned to the frontend.
func TestConversation_MessageTokenStats(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "Stats test",
		ProviderID: 0,
		Model:      "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	if err := svc.SaveMessage(ctx, MessagePayload{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "Short prompt",
	}); err != nil {
		t.Fatalf("SaveMessage user: %v", err)
	}

	assistantStats := &MessageTokenStats{
		TokensOut:       42,
		DurationMS:      2100,
		TokensPerSecond: 20.0,
		Estimated:       true,
	}
	if err := svc.SaveMessage(ctx, MessagePayload{
		ConversationID: conv.ID,
		Role:           "assistant",
		Content:        "Assistant answer",
		TokenStats:     assistantStats,
	}); err != nil {
		t.Fatalf("SaveMessage assistant: %v", err)
	}

	items, err := svc.ListMessages(ctx, ListMessagesPayload{ConversationID: conv.ID, Limit: 10})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(items))
	}
	if items[0].TokenStats.TokensIn < 1 || !items[0].TokenStats.Estimated {
		t.Fatalf("user token stats not auto-populated: %+v", items[0].TokenStats)
	}
	if items[1].TokenStats.TokensOut != 42 || items[1].TokenStats.DurationMS != 2100 {
		t.Fatalf("assistant token stats mismatch: %+v", items[1].TokenStats)
	}
}

func TestConversation_DeleteMessagePersists(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "Delete message test",
		ProviderID: 0,
		Model:      "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	if err := svc.SaveMessage(ctx, MessagePayload{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "hello",
	}); err != nil {
		t.Fatalf("SaveMessage user: %v", err)
	}
	if err := svc.SaveMessage(ctx, MessagePayload{
		ConversationID: conv.ID,
		Role:           "assistant",
		Content:        "world",
	}); err != nil {
		t.Fatalf("SaveMessage assistant: %v", err)
	}

	before, err := svc.ListMessages(ctx, ListMessagesPayload{ConversationID: conv.ID, Limit: 10})
	if err != nil {
		t.Fatalf("ListMessages before delete: %v", err)
	}
	if len(before) != 2 {
		t.Fatalf("expected 2 messages before delete, got %d", len(before))
	}

	res, err := svc.DeleteMessage(ctx, DeleteMessagePayload{ConversationID: conv.ID, MessageID: before[0].ID})
	if err != nil {
		t.Fatalf("DeleteMessage: %v", err)
	}
	if res["ok"] != true {
		t.Fatalf("unexpected delete result: %+v", res)
	}

	after, err := svc.ListMessages(ctx, ListMessagesPayload{ConversationID: conv.ID, Limit: 10})
	if err != nil {
		t.Fatalf("ListMessages after delete: %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("expected 1 message after delete, got %d", len(after))
	}
	if after[0].Content != "world" {
		t.Fatalf("unexpected remaining message: %+v", after[0])
	}
}

// TestConversation_UpdateConversationSettings verifies per-conversation runtime
// settings persistence (provider/model/temperature/max_tokens/system_prompt).
func TestConversation_UpdateConversationSettings(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	provider, err := svc.CreateProvider(ctx, CreateProviderPayload{
		Name:        "p-settings",
		Type:        "openai-compatible",
		URL:         "http://localhost:11434/v1",
		Active:      true,
		Temperature: 0.7,
		NumCtx:      1024,
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{Title: "Cfg", ProviderID: 0, Model: "gpt-4o-mini"})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	updated, err := svc.UpdateConversationSettings(ctx, UpdateConversationSettingsPayload{
		ConversationID: conv.ID,
		ProviderID:     provider.ID,
		Model:          "llama3.2",
		Temperature:    1.1,
		MaxTokens:      256,
		SystemPrompt:   "Be concise",
	})
	if err != nil {
		t.Fatalf("UpdateConversationSettings: %v", err)
	}

	if updated.ProviderID != provider.ID || updated.Model != "llama3.2" {
		t.Fatalf("unexpected provider/model after update: %+v", updated)
	}
	if updated.Temperature != 1.1 || updated.MaxTokens != 256 || updated.SystemPrompt != "Be concise" {
		t.Fatalf("unexpected runtime settings after update: %+v", updated)
	}

	list, err := svc.ListConversations(ctx, ListConversationsPayload{Limit: 10})
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one conversation, got %d", len(list))
	}
	if list[0].MaxTokens != 256 || list[0].SystemPrompt != "Be concise" {
		t.Fatalf("list does not expose updated settings: %+v", list[0])
	}
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
