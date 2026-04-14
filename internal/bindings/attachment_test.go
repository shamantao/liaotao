/*
  attachment_test.go -- Unit tests for attachment upload/list bindings.
  Covers OBJ-03 and FILE-02 storage contract.
*/

package bindings

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttachment_UploadAndList(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	attachmentsDir := filepath.Join(t.TempDir(), "attachments")
	t.Setenv("LIAOTAO_ATTACHMENTS_DIR", attachmentsDir)

	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{
		Title:      "Attachment test",
		ProviderID: 0,
		Model:      "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	body := []byte("hello attachment")
	uploaded, err := svc.UploadAttachment(ctx, UploadAttachmentPayload{
		ConversationID: conv.ID,
		FileName:       "notes.txt",
		MimeType:       "text/plain",
		ContentBase64:  base64.StdEncoding.EncodeToString(body),
	})
	if err != nil {
		t.Fatalf("UploadAttachment: %v", err)
	}
	if uploaded.ConversationID != conv.ID {
		t.Fatalf("unexpected conversation id: %+v", uploaded)
	}
	if uploaded.ProjectID <= 0 {
		t.Fatalf("expected project id in attachment summary: %+v", uploaded)
	}
	if uploaded.SizeBytes != int64(len(body)) {
		t.Fatalf("size mismatch: got %d want %d", uploaded.SizeBytes, len(body))
	}
	if !strings.Contains(uploaded.StoragePath, "data/attachments/") {
		t.Fatalf("unexpected storage path: %s", uploaded.StoragePath)
	}

	storedFile := filepath.Join(attachmentsDir, filepath.Base(filepath.Dir(uploaded.StoragePath)), filepath.Base(uploaded.StoragePath))
	raw, err := os.ReadFile(storedFile)
	if err != nil {
		t.Fatalf("read stored attachment: %v", err)
	}
	if string(raw) != string(body) {
		t.Fatalf("stored content mismatch: got %q want %q", string(raw), string(body))
	}

	items, err := svc.ListAttachments(ctx, ListAttachmentsPayload{ConversationID: conv.ID, Limit: 20})
	if err != nil {
		t.Fatalf("ListAttachments: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(items))
	}
	if items[0].FileName != "notes.txt" {
		t.Fatalf("unexpected file name: %q", items[0].FileName)
	}
}

func TestAttachment_UploadRejectsInvalidConversation(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)
	t.Setenv("LIAOTAO_ATTACHMENTS_DIR", filepath.Join(t.TempDir(), "attachments"))

	_, err := svc.UploadAttachment(ctx, UploadAttachmentPayload{
		ConversationID: 999,
		FileName:       "ghost.txt",
		MimeType:       "text/plain",
		ContentBase64:  base64.StdEncoding.EncodeToString([]byte("no conversation")),
	})
	if err == nil {
		t.Fatal("expected error for missing conversation")
	}
}
