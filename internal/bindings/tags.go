/*
  tags.go -- CRUD bindings for conversation tags (CONV2-02).
  Provides tag creation, listing, deletion and association with conversations.
*/

package bindings

import (
	"context"
	"fmt"
	"strings"
)

// CreateTagPayload is the request payload to create a new tag.
type CreateTagPayload struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// TagRecord is the full tag descriptor returned by CRUD operations.
type TagRecord struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	CreatedAt string `json:"created_at"`
}

// AddTagToConversationPayload associates a tag with a conversation.
type AddTagToConversationPayload struct {
	ConversationID int64 `json:"conversation_id"`
	TagID          int64 `json:"tag_id"`
}

// RemoveTagFromConversationPayload removes a tag from a conversation.
type RemoveTagFromConversationPayload struct {
	ConversationID int64 `json:"conversation_id"`
	TagID          int64 `json:"tag_id"`
}

// ListTagsByConversationPayload filters conversations by tag.
type ListTagsByConversationPayload struct {
	TagID     int64 `json:"tag_id"`
	Limit     int   `json:"limit"`
	ProjectID int64 `json:"project_id"`
}

// CreateTag inserts a new user-defined tag.
func (s *Service) CreateTag(ctx context.Context, payload CreateTagPayload) (TagRecord, error) {
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return TagRecord{}, fmt.Errorf("tag name is required")
	}
	color := strings.TrimSpace(payload.Color)
	if color == "" {
		color = "#6c757d"
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tags (name, color) VALUES (?, ?)`,
		name, color,
	)
	if err != nil {
		return TagRecord{}, fmt.Errorf("create tag: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return TagRecord{}, err
	}
	return s.getTagByID(ctx, id)
}

// ListTags returns all tags ordered by name.
func (s *Service) ListTags(ctx context.Context) ([]TagRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, color, created_at FROM tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TagRecord
	for rows.Next() {
		var t TagRecord
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []TagRecord{}
	}
	return out, nil
}

// DeleteTag removes a tag and all its conversation associations.
func (s *Service) DeleteTag(ctx context.Context, tagID int64) error {
	if tagID <= 0 {
		return fmt.Errorf("tag id must be > 0")
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, tagID)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("tag %d not found", tagID)
	}
	return nil
}

// UpdateTag renames a tag or changes its color.
func (s *Service) UpdateTag(ctx context.Context, payload TagRecord) (TagRecord, error) {
	if payload.ID <= 0 {
		return TagRecord{}, fmt.Errorf("tag id must be > 0")
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return TagRecord{}, fmt.Errorf("tag name is required")
	}
	color := strings.TrimSpace(payload.Color)
	if color == "" {
		color = "#6c757d"
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE tags SET name = ?, color = ? WHERE id = ?`,
		name, color, payload.ID,
	)
	if err != nil {
		return TagRecord{}, fmt.Errorf("update tag: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return TagRecord{}, fmt.Errorf("tag %d not found", payload.ID)
	}
	return s.getTagByID(ctx, payload.ID)
}

// AddTagToConversation links a tag to a conversation (idempotent).
func (s *Service) AddTagToConversation(ctx context.Context, payload AddTagToConversationPayload) error {
	if payload.ConversationID <= 0 || payload.TagID <= 0 {
		return fmt.Errorf("conversation_id and tag_id must be > 0")
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO conversation_tags (conversation_id, tag_id) VALUES (?, ?)`,
		payload.ConversationID, payload.TagID,
	)
	return err
}

// RemoveTagFromConversation removes the association between tag and conversation.
func (s *Service) RemoveTagFromConversation(ctx context.Context, payload RemoveTagFromConversationPayload) error {
	if payload.ConversationID <= 0 || payload.TagID <= 0 {
		return fmt.Errorf("conversation_id and tag_id must be > 0")
	}
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM conversation_tags WHERE conversation_id = ? AND tag_id = ?`,
		payload.ConversationID, payload.TagID,
	)
	return err
}

// ListConversationsByTag returns conversations that have a given tag applied.
func (s *Service) ListConversationsByTag(ctx context.Context, payload ListTagsByConversationPayload) ([]ConversationSummary, error) {
	if payload.TagID <= 0 {
		return s.listConversationsWithQuery(ctx, "", payload.Limit, payload.ProjectID)
	}
	limit := payload.Limit
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT`+conversationSelectSQL+`
		 FROM conversations c
		 LEFT JOIN projects pr ON pr.id = c.project_id
		 LEFT JOIN providers p  ON p.id  = c.provider_id
		 WHERE c.id IN (
		     SELECT conversation_id FROM conversation_tags WHERE tag_id = ?
		 )
		 AND (? <= 0 OR c.project_id = ?)
		 ORDER BY c.updated_at DESC
		 LIMIT ?`,
		payload.TagID,
		payload.ProjectID,
		payload.ProjectID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ConversationSummary, 0, limit)
	for rows.Next() {
		var item ConversationSummary
		if err := rows.Scan(&item.ID, &item.Title, &item.ProjectID, &item.Project, &item.ProviderID, &item.Provider, &item.Model, &item.Temperature, &item.MaxTokens, &item.SystemPrompt, &item.UpdatedAt, &item.TokenCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.loadTagsForConversations(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) getTagByID(ctx context.Context, id int64) (TagRecord, error) {
	var t TagRecord
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, color, created_at FROM tags WHERE id = ?`, id,
	).Scan(&t.ID, &t.Name, &t.Color, &t.CreatedAt)
	if err != nil {
		return TagRecord{}, err
	}
	return t, nil
}
