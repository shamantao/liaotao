// service.go -- Go binding contracts for frontend communication.
// Defines chat/providers/settings/conversations methods for Wails bindings.

package bindings

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
)

// Service centralizes backend methods exposed to the frontend.
type Service struct {
	db       *sql.DB
	client   *http.Client
	cancelMu sync.Mutex
	cancels  map[string]context.CancelFunc
}

// NewService creates the binding service shared by all MVP domains.
func NewService(db *sql.DB) *Service {
	return &Service{
		db:      db,
		client:  &http.Client{},
		cancels: make(map[string]context.CancelFunc),
	}
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

// CreateConversationPayload is the frontend request payload to create a conversation.
type CreateConversationPayload struct {
	Title      string `json:"title"`
	ProviderID string `json:"provider_id"`
	Model      string `json:"model"`
}

// ListConversationsPayload controls paging for conversation listing.
type ListConversationsPayload struct {
	Limit int `json:"limit"`
}

// ListMessagesPayload controls message listing for one conversation.
type ListMessagesPayload struct {
	ConversationID int64 `json:"conversation_id"`
	Limit          int   `json:"limit"`
}

// MessageSummary is a persisted message row returned to frontend.
type MessageSummary struct {
	ID        int64  `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// CreateConversation inserts and returns a new conversation id.
func (s *Service) CreateConversation(ctx context.Context, payload CreateConversationPayload) (ConversationSummary, error) {
	title := payload.Title
	if title == "" {
		title = "New chat"
	}
	providerID := payload.ProviderID
	if providerID == "" {
		providerID = "openai-compatible"
	}
	model := payload.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO conversations (title, provider_id, model) VALUES (?, ?, ?)`,
		title,
		providerID,
		model,
	)
	if err != nil {
		return ConversationSummary{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ConversationSummary{}, err
	}

	item := ConversationSummary{}
	err = s.db.QueryRowContext(
		ctx,
		`SELECT id, title, provider_id, model, updated_at FROM conversations WHERE id = ?`,
		id,
	).Scan(&item.ID, &item.Title, &item.Provider, &item.Model, &item.UpdatedAt)
	if err != nil {
		return ConversationSummary{}, err
	}

	return item, nil
}

// ListConversations returns conversations ordered by latest activity.

func (s *Service) ListConversations(ctx context.Context, payload ListConversationsPayload) ([]ConversationSummary, error) {
	limit := payload.Limit
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

// ListMessages returns messages for a conversation ordered by creation.
func (s *Service) ListMessages(ctx context.Context, payload ListMessagesPayload) ([]MessageSummary, error) {
	if payload.ConversationID <= 0 {
		return []MessageSummary{}, nil
	}

	limit := payload.Limit
	if limit <= 0 {
		limit = 1000
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, role, content, created_at
		 FROM messages
		 WHERE conversation_id = ?
		 ORDER BY id ASC
		 LIMIT ?`,
		payload.ConversationID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MessageSummary, 0, limit)
	for rows.Next() {
		var item MessageSummary
		if err := rows.Scan(&item.ID, &item.Role, &item.Content, &item.CreatedAt); err != nil {
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
