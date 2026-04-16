/*
  export_test.go -- Tests for ExportConversation and ExportProject (EXP-01..05 / CONV2-03).
*/

package bindings

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestExport_ConversationMarkdown exports a conversation as Markdown and checks the file.
func TestExport_ConversationMarkdown(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	conv, _ := svc.CreateConversation(ctx, CreateConversationPayload{Title: "My Chat", Model: "gpt-4o"})
	_, _ = database.ExecContext(ctx,
		`INSERT INTO messages (conversation_id, role, content) VALUES (?, 'user', 'Hello world')`, conv.ID)
	_, _ = database.ExecContext(ctx,
		`INSERT INTO messages (conversation_id, role, content) VALUES (?, 'assistant', 'Hi there')`, conv.ID)

	result, err := svc.ExportConversation(ctx, ExportConversationPayload{
		ConversationID: conv.ID,
		Format:         ExportFormatMarkdown,
	})
	if err != nil {
		t.Fatalf("ExportConversation: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(result.FilePath) })

	if !strings.HasSuffix(result.FilePath, ".md") {
		t.Errorf("expected .md file, got %q", result.FilePath)
	}
	if result.ItemCount != 2 {
		t.Errorf("expected 2 messages, got %d", result.ItemCount)
	}

	data, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("read export file: %v", err)
	}
	if !strings.Contains(string(data), "Hello world") {
		t.Error("markdown export does not contain message content")
	}
	if !strings.Contains(string(data), "My Chat") {
		t.Error("markdown export does not contain conversation title")
	}
}

// TestExport_ConversationJSON exports a conversation as JSON and validates the structure.
func TestExport_ConversationJSON(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	conv, _ := svc.CreateConversation(ctx, CreateConversationPayload{Title: "JSON Chat", Model: "gpt-4o-mini"})
	_, _ = database.ExecContext(ctx,
		`INSERT INTO messages (conversation_id, role, content) VALUES (?, 'user', 'test msg')`, conv.ID)

	result, err := svc.ExportConversation(ctx, ExportConversationPayload{
		ConversationID: conv.ID,
		Format:         ExportFormatJSON,
	})
	if err != nil {
		t.Fatalf("ExportConversation JSON: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(result.FilePath) })

	data, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("read json export: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON export: %v", err)
	}
	if parsed["title"] != "JSON Chat" {
		t.Errorf("unexpected title in JSON: %v", parsed["title"])
	}
}

// TestExport_ProjectMarkdown exports all conversations in project 1.
func TestExport_ProjectMarkdown(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	c1, _ := svc.CreateConversation(ctx, CreateConversationPayload{Title: "conv-a", Model: "m"})
	_, _ = database.ExecContext(ctx,
		`INSERT INTO messages (conversation_id, role, content) VALUES (?, 'user', 'alpha')`, c1.ID)
	c2, _ := svc.CreateConversation(ctx, CreateConversationPayload{Title: "conv-b", Model: "m"})
	_, _ = database.ExecContext(ctx,
		`INSERT INTO messages (conversation_id, role, content) VALUES (?, 'user', 'beta')`, c2.ID)

	result, err := svc.ExportProject(ctx, ExportProjectPayload{
		ProjectID: 1,
		Format:    ExportFormatMarkdown,
	})
	if err != nil {
		t.Fatalf("ExportProject: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(result.FilePath) })

	if result.ItemCount != 2 {
		t.Errorf("expected 2 total messages, got %d", result.ItemCount)
	}

	data, _ := os.ReadFile(result.FilePath)
	if !strings.Contains(string(data), "alpha") || !strings.Contains(string(data), "beta") {
		t.Error("project export is missing message content")
	}
}
