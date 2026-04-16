/*
  tags_test.go -- Tests for conversation tag CRUD and filtering (CONV2-02).
*/

package bindings

import (
	"context"
	"testing"
)

// TestTags_CreateAndList verifies that CreateTag stores and ListTags retrieves tags.
func TestTags_CreateAndList(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	tag, err := svc.CreateTag(ctx, CreateTagPayload{Name: "important", Color: "#ff0000"})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	if tag.ID <= 0 {
		t.Fatal("expected positive tag ID")
	}
	if tag.Name != "important" || tag.Color != "#ff0000" {
		t.Errorf("unexpected tag: %+v", tag)
	}

	list, err := svc.ListTags(ctx)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(list) != 1 || list[0].ID != tag.ID {
		t.Errorf("expected 1 tag, got %d", len(list))
	}
}

// TestTags_DefaultColor ensures missing color falls back to the default value.
func TestTags_DefaultColor(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	tag, err := svc.CreateTag(ctx, CreateTagPayload{Name: "generic"})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	if tag.Color != "#6c757d" {
		t.Errorf("want default color, got %q", tag.Color)
	}
}

// TestTags_AddRemove verifies that a tag can be attached and detached from a conversation.
func TestTags_AddRemove(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	conv, _ := svc.CreateConversation(ctx, CreateConversationPayload{Title: "tagged", Model: "gpt-4o"})
	tag, _ := svc.CreateTag(ctx, CreateTagPayload{Name: "urgent"})

	if err := svc.AddTagToConversation(ctx, AddTagToConversationPayload{
		ConversationID: conv.ID, TagID: tag.ID,
	}); err != nil {
		t.Fatalf("AddTagToConversation: %v", err)
	}

	// List should now show the tag on the conversation.
	convs, err := svc.ListConversationsByTag(ctx, ListTagsByConversationPayload{TagID: tag.ID})
	if err != nil {
		t.Fatalf("ListConversationsByTag: %v", err)
	}
	if len(convs) != 1 || convs[0].ID != conv.ID {
		t.Errorf("expected 1 tagged conversation, got %d", len(convs))
	}

	// Remove; list must be empty again.
	if err := svc.RemoveTagFromConversation(ctx, RemoveTagFromConversationPayload{
		ConversationID: conv.ID, TagID: tag.ID,
	}); err != nil {
		t.Fatalf("RemoveTagFromConversation: %v", err)
	}

	convs, err = svc.ListConversationsByTag(ctx, ListTagsByConversationPayload{TagID: tag.ID})
	if err != nil {
		t.Fatalf("ListConversationsByTag after remove: %v", err)
	}
	if len(convs) != 0 {
		t.Errorf("expected 0 conversations, got %d", len(convs))
	}
}

// TestTags_DeleteTag verifies cascade removal of conversation_tags entries.
func TestTags_DeleteTag(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	conv, _ := svc.CreateConversation(ctx, CreateConversationPayload{Title: "c1", Model: "gpt-4o"})
	tag, _ := svc.CreateTag(ctx, CreateTagPayload{Name: "temp"})
	_ = svc.AddTagToConversation(ctx, AddTagToConversationPayload{ConversationID: conv.ID, TagID: tag.ID})

	if err := svc.DeleteTag(ctx, tag.ID); err != nil {
		t.Fatalf("DeleteTag: %v", err)
	}

	list, _ := svc.ListTags(ctx)
	if len(list) != 0 {
		t.Errorf("expected no tags after delete, got %d", len(list))
	}

	// Conversation should still exist without tags.
	convs, _ := svc.ListConversations(ctx, ListConversationsPayload{Limit: 10})
	if len(convs) == 0 {
		t.Fatal("conversation was deleted along with tag")
	}
	if len(convs[0].Tags) != 0 {
		t.Errorf("expected empty tags on conversation, got %v", convs[0].Tags)
	}
}

// TestTags_TokenCountInSummary verifies that CONV2-05 token_count is populated.
func TestTags_TokenCountInSummary(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	conv, _ := svc.CreateConversation(ctx, CreateConversationPayload{Title: "t", Model: "m"})

	// Insert a message with known token stats.
	_, err := database.ExecContext(ctx,
		`INSERT INTO messages (conversation_id, role, content, token_stats)
		 VALUES (?, 'user', 'hello', '{"tokens_in":10,"tokens_out":20}')`,
		conv.ID,
	)
	if err != nil {
		t.Fatalf("insert message: %v", err)
	}

	convs, err := svc.ListConversations(ctx, ListConversationsPayload{Limit: 10})
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(convs) == 0 {
		t.Fatal("no conversations")
	}
	if convs[0].TokenCount != 30 {
		t.Errorf("expected TokenCount=30, got %d", convs[0].TokenCount)
	}
}

// TestTags_FTS5Search verifies that SearchConversations matches message body via FTS5.
func TestTags_FTS5Search(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	conv, _ := svc.CreateConversation(ctx, CreateConversationPayload{Title: "research", Model: "m"})
	_, err := database.ExecContext(ctx,
		`INSERT INTO messages (conversation_id, role, content) VALUES (?, 'user', 'quantum entanglement experiment')`,
		conv.ID,
	)
	if err != nil {
		t.Fatalf("insert message: %v", err)
	}

	// FTS5 MATCH should find the conversation.
	results, err := svc.SearchConversations(ctx, SearchConversationsPayload{
		Query: "quantum",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("SearchConversations (FTS5): %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected FTS5 match, got 0 results")
	}
	if results[0].ID != conv.ID {
		t.Errorf("expected conversation %d, got %d", conv.ID, results[0].ID)
	}
}
