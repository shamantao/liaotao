/*
  retrieval_test.go -- Unit tests for request context and local retrieval backend.
  Covers OBJ-05/06/07 and KR-01/02/03.
*/

package bindings

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetrieval_IndexAndRetrieveOnAttachmentUpload(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)
	t.Setenv("LIAOTAO_ATTACHMENTS_DIR", filepath.Join(t.TempDir(), "attachments"))

	project, err := svc.CreateProject(ctx, CreateProjectPayload{Name: "KR Project"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "KR chat",
		ProjectID:  project.ID,
		ProviderID: 0,
		Model:      "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	doc := strings.Repeat("This document describes retrieval chunking for project memory and ranking. ", 40)
	_, err = svc.UploadAttachment(ctx, UploadAttachmentPayload{
		ConversationID: conv.ID,
		FileName:       "knowledge.md",
		MimeType:       "text/markdown",
		ContentBase64:  base64.StdEncoding.EncodeToString([]byte(doc)),
	})
	if err != nil {
		t.Fatalf("UploadAttachment: %v", err)
	}

	var count int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM knowledge_items WHERE conversation_id = ?`, conv.ID).Scan(&count); err != nil {
		t.Fatalf("query knowledge_items count: %v", err)
	}
	if count < 2 {
		t.Fatalf("expected chunked knowledge items, got %d", count)
	}

	hits, err := svc.retrievalBackend().Retrieve(ctx, RetrievalQuery{
		ConversationID: conv.ID,
		ProjectID:      project.ID,
		Query:          "project memory retrieval",
		TopK:           3,
	})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected retrieval hits")
	}
	if hits[0].Score <= 0 {
		t.Fatalf("expected positive score, got %.2f", hits[0].Score)
	}
}

func TestRequestContext_ComputesRecentMessagesSummaryAndSnippets(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)
	t.Setenv("LIAOTAO_ATTACHMENTS_DIR", filepath.Join(t.TempDir(), "attachments"))

	project, err := svc.CreateProject(ctx, CreateProjectPayload{Name: "RC Project"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "RC chat",
		ProjectID:  project.ID,
		ProviderID: 0,
		Model:      "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	seedMessages := []MessagePayload{
		{ConversationID: conv.ID, Role: "user", Content: "We need a migration plan for retrieval."},
		{ConversationID: conv.ID, Role: "assistant", Content: "Sure, let's split into milestones."},
		{ConversationID: conv.ID, Role: "user", Content: "Include chunking and ranking details."},
		{ConversationID: conv.ID, Role: "assistant", Content: "We'll add local sqlite knowledge items."},
		{ConversationID: conv.ID, Role: "user", Content: "Also add tests and acceptance criteria."},
	}
	for _, item := range seedMessages {
		if err := svc.SaveMessage(ctx, item); err != nil {
			t.Fatalf("SaveMessage(%s): %v", item.Role, err)
		}
	}

	_, err = svc.UploadAttachment(ctx, UploadAttachmentPayload{
		ConversationID: conv.ID,
		FileName:       "plan.txt",
		MimeType:       "text/plain",
		ContentBase64:  base64.StdEncoding.EncodeToString([]byte("Retrieval ranking and chunking strategy for sqlite context injection.")),
	})
	if err != nil {
		t.Fatalf("UploadAttachment: %v", err)
	}

	rc, err := svc.buildRequestContext(ctx, conv.ID, "How should retrieval ranking work?")
	if err != nil {
		t.Fatalf("buildRequestContext: %v", err)
	}
	if rc.ConversationID != conv.ID {
		t.Fatalf("unexpected conversation id: %+v", rc)
	}
	if len(rc.RecentMessages) == 0 {
		t.Fatal("expected recent messages in request context")
	}
	if len(rc.Snippets) == 0 {
		t.Fatal("expected retrieved snippets in request context")
	}
	if strings.TrimSpace(promptWithRequestContext("Question", rc)) == "Question" {
		t.Fatal("expected prompt enrichment with context")
	}
}

func TestRetrieval_IgnoresBinaryLikeAttachment(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)
	t.Setenv("LIAOTAO_ATTACHMENTS_DIR", filepath.Join(t.TempDir(), "attachments"))

	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "binary test",
		ProviderID: 0,
		Model:      "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	_, err = svc.UploadAttachment(ctx, UploadAttachmentPayload{
		ConversationID: conv.ID,
		FileName:       "archive.bin",
		MimeType:       "application/octet-stream",
		ContentBase64:  base64.StdEncoding.EncodeToString(binaryData),
	})
	if err != nil {
		t.Fatalf("UploadAttachment: %v", err)
	}

	var count int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM knowledge_items WHERE conversation_id = ?`, conv.ID).Scan(&count); err != nil {
		t.Fatalf("count knowledge_items: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no knowledge items for binary source, got %d", count)
	}
}

func TestRetrieval_ProjectScopeSharedAcrossConversations(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)
	t.Setenv("LIAOTAO_ATTACHMENTS_DIR", filepath.Join(t.TempDir(), "attachments"))

	project, err := svc.CreateProject(ctx, CreateProjectPayload{Name: "Shared KR"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	convA, err := svc.CreateConversation(ctx, CreateConversationPayload{Title: "A", ProjectID: project.ID, Model: "gpt-4o-mini"})
	if err != nil {
		t.Fatalf("CreateConversation A: %v", err)
	}
	convB, err := svc.CreateConversation(ctx, CreateConversationPayload{Title: "B", ProjectID: project.ID, Model: "gpt-4o-mini"})
	if err != nil {
		t.Fatalf("CreateConversation B: %v", err)
	}

	uploaded, err := svc.UploadAttachment(ctx, UploadAttachmentPayload{
		ConversationID: convA.ID,
		FileName:       "shared-notes.md",
		MimeType:       "text/markdown",
		ContentBase64:  base64.StdEncoding.EncodeToString([]byte("A unique phrase: deterministic retrieval contract")),
	})
	if err != nil {
		t.Fatalf("UploadAttachment: %v", err)
	}

	hitsBefore, err := svc.retrievalBackend().Retrieve(ctx, RetrievalQuery{
		ConversationID: convB.ID,
		ProjectID:      project.ID,
		Query:          "deterministic retrieval contract",
		TopK:           4,
	})
	if err != nil {
		t.Fatalf("Retrieve before share: %v", err)
	}
	if len(hitsBefore) != 0 {
		t.Fatalf("expected no cross-conversation hits before share, got %d", len(hitsBefore))
	}

	if _, err := svc.SetAttachmentProjectScope(ctx, SetAttachmentProjectScopePayload{
		AttachmentID: uploaded.ID,
		Shared:       true,
	}); err != nil {
		t.Fatalf("SetAttachmentProjectScope: %v", err)
	}

	hitsAfter, err := svc.retrievalBackend().Retrieve(ctx, RetrievalQuery{
		ConversationID: convB.ID,
		ProjectID:      project.ID,
		Query:          "deterministic retrieval contract",
		TopK:           4,
	})
	if err != nil {
		t.Fatalf("Retrieve after share: %v", err)
	}
	if len(hitsAfter) == 0 {
		t.Fatal("expected project-scoped retrieval hits after share")
	}
}

func TestConversationMemory_PersistedAfterSaveMessage(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{Title: "Memory", Model: "gpt-4o-mini"})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	for _, text := range []string{
		"First user fact about migration.",
		"Assistant confirms migration path.",
		"Second user fact about retrieval quality.",
	} {
		role := "user"
		if strings.HasPrefix(text, "Assistant") {
			role = "assistant"
		}
		if err := svc.SaveMessage(ctx, MessagePayload{ConversationID: conv.ID, Role: role, Content: text}); err != nil {
			t.Fatalf("SaveMessage: %v", err)
		}
	}

	summary, err := svc.loadConversationMemorySummary(ctx, conv.ID)
	if err != nil {
		t.Fatalf("loadConversationMemorySummary: %v", err)
	}
	if strings.TrimSpace(summary) == "" {
		t.Fatal("expected non-empty persisted conversation memory summary")
	}
}
