/*
  attachments.go -- Conversation attachment bindings exposed to frontend.
  Responsibilities: upload files to local storage and list attachments per conversation.
*/

package bindings

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var attachmentUnsafeChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// AttachmentSummary is returned to the frontend attachment panel.
type AttachmentSummary struct {
	ID              int64  `json:"id"`
	ConversationID  int64  `json:"conversation_id"`
	ProjectID       int64  `json:"project_id"`
	FileName        string `json:"file_name"`
	MimeType        string `json:"mime_type"`
	SizeBytes       int64  `json:"size_bytes"`
	SharedInProject bool   `json:"shared_in_project"`
	StoragePath     string `json:"storage_path"`
	CreatedAt       string `json:"created_at"`
}

// UploadAttachmentPayload contains one dropped file encoded in base64.
type UploadAttachmentPayload struct {
	ConversationID int64  `json:"conversation_id"`
	FileName       string `json:"file_name"`
	MimeType       string `json:"mime_type"`
	ContentBase64  string `json:"content_base64"`
}

// ListAttachmentsPayload scopes attachment listing by conversation.
type ListAttachmentsPayload struct {
	ConversationID int64 `json:"conversation_id"`
	Limit          int   `json:"limit"`
}

// SetAttachmentProjectScopePayload toggles whether one attachment is shared in project knowledge base.
type SetAttachmentProjectScopePayload struct {
	AttachmentID int64 `json:"attachment_id"`
	Shared       bool  `json:"shared"`
}

// UploadAttachment persists one attachment in DB and writes file bytes under data/attachments/<conversation_id>/.
func (s *Service) UploadAttachment(ctx context.Context, payload UploadAttachmentPayload) (AttachmentSummary, error) {
	if payload.ConversationID <= 0 {
		return AttachmentSummary{}, fmt.Errorf("conversation_id must be > 0")
	}
	name := sanitizeAttachmentName(payload.FileName)
	if name == "" {
		return AttachmentSummary{}, fmt.Errorf("file_name is required")
	}
	if strings.TrimSpace(payload.ContentBase64) == "" {
		return AttachmentSummary{}, fmt.Errorf("content_base64 is required")
	}

	var projectID int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(project_id, 1) FROM conversations WHERE id = ?`, payload.ConversationID).Scan(&projectID); err != nil {
		return AttachmentSummary{}, fmt.Errorf("conversation not found: %w", err)
	}

	raw, err := base64.StdEncoding.DecodeString(payload.ContentBase64)
	if err != nil {
		return AttachmentSummary{}, fmt.Errorf("decode content_base64: %w", err)
	}

	convDir := filepath.Join(attachmentsRootDir(), strconv.FormatInt(payload.ConversationID, 10))
	if err := os.MkdirAll(convDir, 0o755); err != nil {
		return AttachmentSummary{}, fmt.Errorf("create attachment dir: %w", err)
	}
	fullPath, storedName, err := reserveAttachmentPath(convDir, name)
	if err != nil {
		return AttachmentSummary{}, err
	}
	if err := os.WriteFile(fullPath, raw, 0o644); err != nil {
		return AttachmentSummary{}, fmt.Errorf("write attachment: %w", err)
	}

	storagePath := filepath.ToSlash(filepath.Join("data", "attachments", strconv.FormatInt(payload.ConversationID, 10), storedName))
	if projectID > 0 {
		_, _ = s.db.ExecContext(ctx, `UPDATE projects SET retrieval_indexing = 1 WHERE id = ?`, projectID)
	}

	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO attachments (conversation_id, file_name, storage_path, mime_type, size_bytes, shared_in_project)
		 VALUES (?, ?, ?, ?, ?, 0)`,
		payload.ConversationID,
		payload.FileName,
		storagePath,
		strings.TrimSpace(payload.MimeType),
		len(raw),
	)
	if err != nil {
		return AttachmentSummary{}, err
	}
	attachmentID, err := res.LastInsertId()
	if err != nil {
		return AttachmentSummary{}, err
	}

	_, _ = s.db.ExecContext(ctx, `UPDATE conversations SET updated_at = datetime('now') WHERE id = ?`, payload.ConversationID)

	item := AttachmentSummary{}
	var sharedInt int
	err = s.db.QueryRowContext(
		ctx,
		`SELECT a.id, a.conversation_id, COALESCE(c.project_id, 1), a.file_name, a.mime_type, a.size_bytes, COALESCE(a.shared_in_project, 0), a.storage_path, a.created_at
		 FROM attachments a
		 JOIN conversations c ON c.id = a.conversation_id
		 WHERE a.id = ?`,
		attachmentID,
	).Scan(&item.ID, &item.ConversationID, &item.ProjectID, &item.FileName, &item.MimeType, &item.SizeBytes, &sharedInt, &item.StoragePath, &item.CreatedAt)
	if err != nil {
		if projectID > 0 {
			_, _ = s.db.ExecContext(ctx, `UPDATE projects SET retrieval_indexing = 0 WHERE id = ?`, projectID)
		}
		return AttachmentSummary{}, err
	}
	item.SharedInProject = sharedInt != 0
	s.indexAttachmentKnowledge(ctx, KnowledgeIndexInput{
		ConversationID: item.ConversationID,
		ProjectID:      item.ProjectID,
		AttachmentID:   item.ID,
		SourceName:     item.FileName,
		SourcePath:     item.StoragePath,
		MimeType:       item.MimeType,
		Content:        string(raw),
	})
	if item.ProjectID > 0 {
		_, _ = s.db.ExecContext(ctx, `UPDATE projects SET retrieval_indexing = 0 WHERE id = ?`, item.ProjectID)
	}
	return item, nil
}

// ListAttachments returns attachments for one conversation.
func (s *Service) ListAttachments(ctx context.Context, payload ListAttachmentsPayload) ([]AttachmentSummary, error) {
	if payload.ConversationID <= 0 {
		return []AttachmentSummary{}, nil
	}
	limit := payload.Limit
	if limit <= 0 {
		limit = 200
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT a.id, a.conversation_id, COALESCE(c.project_id, 1), a.file_name, a.mime_type, a.size_bytes, COALESCE(a.shared_in_project, 0), a.storage_path, a.created_at
		 FROM attachments a
		 JOIN conversations c ON c.id = a.conversation_id
		 WHERE a.conversation_id = ?
		 ORDER BY a.id DESC
		 LIMIT ?`,
		payload.ConversationID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AttachmentSummary, 0, limit)
	for rows.Next() {
		var item AttachmentSummary
		var sharedInt int
		if err := rows.Scan(&item.ID, &item.ConversationID, &item.ProjectID, &item.FileName, &item.MimeType, &item.SizeBytes, &sharedInt, &item.StoragePath, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.SharedInProject = sharedInt != 0
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// SetAttachmentProjectScope toggles project-level sharing and reindexes retrieval scope rows.
func (s *Service) SetAttachmentProjectScope(ctx context.Context, payload SetAttachmentProjectScopePayload) (AttachmentSummary, error) {
	if payload.AttachmentID <= 0 {
		return AttachmentSummary{}, fmt.Errorf("attachment_id must be > 0")
	}

	item, content, err := s.loadAttachmentForScopeUpdate(ctx, payload.AttachmentID)
	if err != nil {
		return AttachmentSummary{}, err
	}
	if item.ProjectID > 0 {
		_, _ = s.db.ExecContext(ctx, `UPDATE projects SET retrieval_indexing = 1 WHERE id = ?`, item.ProjectID)
	}

	sharedInt := 0
	if payload.Shared {
		sharedInt = 1
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE attachments SET shared_in_project = ? WHERE id = ?`,
		sharedInt,
		payload.AttachmentID,
	); err != nil {
		if item.ProjectID > 0 {
			_, _ = s.db.ExecContext(ctx, `UPDATE projects SET retrieval_indexing = 0 WHERE id = ?`, item.ProjectID)
		}
		return AttachmentSummary{}, err
	}

	s.reindexAttachmentScopes(ctx, item, content, payload.Shared)
	if item.ProjectID > 0 {
		_, _ = s.db.ExecContext(ctx, `UPDATE projects SET retrieval_indexing = 0 WHERE id = ?`, item.ProjectID)
	}

	item.SharedInProject = payload.Shared
	return item, nil
}

func (s *Service) loadAttachmentForScopeUpdate(ctx context.Context, attachmentID int64) (AttachmentSummary, string, error) {
	item := AttachmentSummary{}
	var sharedInt int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT a.id, a.conversation_id, COALESCE(c.project_id, 1), a.file_name, a.mime_type, a.size_bytes, COALESCE(a.shared_in_project, 0), a.storage_path, a.created_at
		 FROM attachments a
		 JOIN conversations c ON c.id = a.conversation_id
		 WHERE a.id = ?`,
		attachmentID,
	).Scan(&item.ID, &item.ConversationID, &item.ProjectID, &item.FileName, &item.MimeType, &item.SizeBytes, &sharedInt, &item.StoragePath, &item.CreatedAt)
	if err != nil {
		return AttachmentSummary{}, "", err
	}
	item.SharedInProject = sharedInt != 0

	fullPath := item.StoragePath
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(".", filepath.FromSlash(fullPath))
	}
	raw, err := os.ReadFile(fullPath)
	if err != nil {
		if envDir := strings.TrimSpace(os.Getenv("LIAOTAO_ATTACHMENTS_DIR")); envDir != "" {
			candidate := filepath.Join(envDir, strconv.FormatInt(item.ConversationID, 10), filepath.Base(item.StoragePath))
			if altRaw, altErr := os.ReadFile(candidate); altErr == nil {
				return item, string(altRaw), nil
			}
		}
		return AttachmentSummary{}, "", err
	}
	return item, string(raw), nil
}

func (s *Service) reindexAttachmentScopes(ctx context.Context, item AttachmentSummary, content string, shared bool) {
	s.indexAttachmentKnowledge(ctx, KnowledgeIndexInput{
		ConversationID: item.ConversationID,
		ProjectID:      item.ProjectID,
		AttachmentID:   item.ID,
		SourceName:     item.FileName,
		SourcePath:     item.StoragePath,
		MimeType:       item.MimeType,
		Content:        content,
	})
	if !shared {
		_, _ = s.db.ExecContext(ctx,
			`DELETE FROM knowledge_items WHERE attachment_id = ? AND scope = 'project'`,
			item.ID,
		)
		return
	}

	chunks := chunkText(content, 900, 120)
	for idx, chunk := range chunks {
		_, _ = s.db.ExecContext(ctx,
			`INSERT INTO knowledge_items
			 (conversation_id, project_id, attachment_id, scope, source_name, source_path, mime_type, chunk_index, chunk_text)
			 VALUES (?, ?, ?, 'project', ?, ?, ?, ?, ?)`,
			item.ConversationID,
			item.ProjectID,
			item.ID,
			item.FileName,
			item.StoragePath,
			item.MimeType,
			idx,
			chunk,
		)
	}
}

func attachmentsRootDir() string {
	if envDir := strings.TrimSpace(os.Getenv("LIAOTAO_ATTACHMENTS_DIR")); envDir != "" {
		return envDir
	}
	return filepath.Join("data", "attachments")
}

func sanitizeAttachmentName(raw string) string {
	base := strings.TrimSpace(filepath.Base(raw))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}
	safe := attachmentUnsafeChars.ReplaceAllString(base, "_")
	safe = strings.Trim(safe, "._-")
	if safe == "" {
		return "attachment.bin"
	}
	return safe
}

func reserveAttachmentPath(dirPath, fileName string) (string, string, error) {
	ext := filepath.Ext(fileName)
	stem := strings.TrimSuffix(fileName, ext)
	if stem == "" {
		stem = "attachment"
	}
	candidate := fileName
	for i := 0; i < 1000; i++ {
		full := filepath.Join(dirPath, candidate)
		if _, err := os.Stat(full); os.IsNotExist(err) {
			return full, candidate, nil
		}
		candidate = fmt.Sprintf("%s_%03d%s", stem, i+1, ext)
	}
	return "", "", fmt.Errorf("cannot reserve attachment path for %s", fileName)
}
