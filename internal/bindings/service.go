// service.go -- Go binding contracts for frontend communication.
// Defines chat/providers/settings/conversations methods for Wails bindings.

package bindings

import (
	"context"
	"database/sql"
	"fmt"
)

// Service centralizes backend methods exposed to the frontend.
type Service struct {
	db *sql.DB
}

// NewService creates the binding service shared by all MVP domains.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// Health returns basic runtime status used by UI startup checks.
func (s *Service) Health(ctx context.Context) (map[string]any, error) {
	if err := s.db.PingContext(ctx); err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":       true,
		"database": "connected",
	}, nil
}

// ConversationSummary is a thin list item payload for the sidebar.
type ConversationSummary struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	UpdatedAt string `json:"updated_at"`
}

// CreateConversation inserts and returns a new conversation id.
func (s *Service) CreateConversation(ctx context.Context, title, providerID, model string) (int64, error) {
	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO conversations (title, provider_id, model) VALUES (?, ?, ?)`,
		title,
		providerID,
		model,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// ListConversations returns conversations ordered by latest activity.
func (s *Service) ListConversations(ctx context.Context, limit int) ([]ConversationSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, title, provider_id, model, updated_at
		 FROM conversations
		 ORDER BY updated_at DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ConversationSummary, 0, limit)
	for rows.Next() {
		var item ConversationSummary
		if err := rows.Scan(&item.ID, &item.Title, &item.Provider, &item.Model, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// MessagePayload is the persistent format for chat messages.
type MessagePayload struct {
	ConversationID int64  `json:"conversation_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
}

// SaveMessage persists one message and refreshes parent updated_at.
func (s *Service) SaveMessage(ctx context.Context, payload MessagePayload) error {
	if payload.ConversationID <= 0 {
		return fmt.Errorf("conversation_id must be > 0")
	}
	if payload.Role == "" || payload.Content == "" {
		return fmt.Errorf("role and content are required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO messages (conversation_id, role, content) VALUES (?, ?, ?)`,
		payload.ConversationID,
		payload.Role,
		payload.Content,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE conversations SET updated_at = datetime('now') WHERE id = ?`,
		payload.ConversationID,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteConversation removes a conversation and all messages (FK cascade).
func (s *Service) DeleteConversation(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("conversation id must be > 0")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM conversations WHERE id = ?`, id)
	return err
}
