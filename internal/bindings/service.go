// service.go -- Go binding contracts for frontend communication.
// Defines chat/providers/settings/conversations methods for Wails bindings.

package bindings

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
)

// Service centralizes backend methods exposed to the frontend.
type Service struct {
	db           *sql.DB
	client       *http.Client
	cancelMu     sync.Mutex
	cancels      map[string]context.CancelFunc
	allowedRoots []string // path sandbox for built-in read_file tool
}

// NewService creates the binding service shared by all MVP domains.
// allowedRoots is optional: pass cfg.PathManager.AllowedRoots to enable read_file sandboxing.
func NewService(db *sql.DB, allowedRoots ...string) *Service {
	return &Service{
		db:           db,
		client:       &http.Client{},
		cancels:      make(map[string]context.CancelFunc),
		allowedRoots: allowedRoots,
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
	ID           int64   `json:"id"`
	Title        string  `json:"title"`
	ProviderID   int64   `json:"provider_id"` // numeric FK; 0 = no provider
	Provider     string  `json:"provider"`    // resolved display name
	Model        string  `json:"model"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
	SystemPrompt string  `json:"system_prompt"`
	UpdatedAt    string  `json:"updated_at"`
}

// CreateConversationPayload is the frontend request payload to create a conversation.
type CreateConversationPayload struct {
	Title      string `json:"title"`
	ProviderID int64  `json:"provider_id"` // numeric ID from providers table; 0 = no provider
	Model      string `json:"model"`
}

// ListConversationsPayload controls paging for conversation listing.
type ListConversationsPayload struct {
	Limit int `json:"limit"`
}

// SearchConversationsPayload controls title/content search for conversations.
type SearchConversationsPayload struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// RenameConversationPayload renames one conversation.
type RenameConversationPayload struct {
	ConversationID int64  `json:"conversation_id"`
	Title          string `json:"title"`
}

// UpdateConversationSettingsPayload updates provider/model/runtime generation settings.
type UpdateConversationSettingsPayload struct {
	ConversationID int64   `json:"conversation_id"`
	ProviderID     int64   `json:"provider_id"`
	Model          string  `json:"model"`
	Temperature    float64 `json:"temperature"`
	MaxTokens      int     `json:"max_tokens"`
	SystemPrompt   string  `json:"system_prompt"`
}

// ListMessagesPayload controls message listing for one conversation.
type ListMessagesPayload struct {
	ConversationID int64 `json:"conversation_id"`
	Limit          int   `json:"limit"`
}

// MessageSummary is a persisted message row returned to frontend.
type MessageTokenStats struct {
	TokensIn        int     `json:"tokens_in,omitempty"`
	TokensOut       int     `json:"tokens_out,omitempty"`
	DurationMS      int64   `json:"duration_ms,omitempty"`
	TokensPerSecond float64 `json:"tokens_per_second,omitempty"`
	Estimated       bool    `json:"estimated,omitempty"`
}

type MessageSummary struct {
	ID         int64             `json:"id"`
	Role       string            `json:"role"`
	Content    string            `json:"content"`
	TokenStats MessageTokenStats `json:"token_stats"`
	CreatedAt  string            `json:"created_at"`
}

// CreateConversation inserts and returns a new conversation id.
func (s *Service) CreateConversation(ctx context.Context, payload CreateConversationPayload) (ConversationSummary, error) {
	title := payload.Title
	if title == "" {
		title = "New chat"
	}
	model := payload.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	// NULLIF(provider_id, 0) stores NULL when no provider is selected.
	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO conversations (title, provider_id, model, temperature, max_tokens, system_prompt)
		 VALUES (?, NULLIF(?, 0), ?, 0.7, 0, '')`,
		title,
		payload.ProviderID,
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
		`SELECT c.id, c.title, COALESCE(c.provider_id, 0), COALESCE(p.name, ''), c.model,
		        COALESCE(c.temperature, 0.7), COALESCE(c.max_tokens, 0), COALESCE(c.system_prompt, ''), c.updated_at
		 FROM conversations c
		 LEFT JOIN providers p ON p.id = c.provider_id
		 WHERE c.id = ?`,
		id,
	).Scan(&item.ID, &item.Title, &item.ProviderID, &item.Provider, &item.Model, &item.Temperature, &item.MaxTokens, &item.SystemPrompt, &item.UpdatedAt)
	if err != nil {
		return ConversationSummary{}, err
	}

	return item, nil
}

// ListConversations returns conversations ordered by latest activity.

func (s *Service) ListConversations(ctx context.Context, payload ListConversationsPayload) ([]ConversationSummary, error) {
	return s.listConversationsWithQuery(ctx, "", payload.Limit)
}

// SearchConversations returns conversations matching title or message content.
func (s *Service) SearchConversations(ctx context.Context, payload SearchConversationsPayload) ([]ConversationSummary, error) {
	return s.listConversationsWithQuery(ctx, payload.Query, payload.Limit)
}

func (s *Service) listConversationsWithQuery(ctx context.Context, query string, limit int) ([]ConversationSummary, error) {
	if limit <= 0 {
		limit = 50
	}

	trimmed := strings.TrimSpace(query)
	var (
		rows *sql.Rows
		err  error
	)
	if trimmed == "" {
		rows, err = s.db.QueryContext(
			ctx,
			`SELECT c.id, c.title, COALESCE(c.provider_id, 0), COALESCE(p.name, ''), c.model,
			        COALESCE(c.temperature, 0.7), COALESCE(c.max_tokens, 0), COALESCE(c.system_prompt, ''), c.updated_at
			 FROM conversations c
			 LEFT JOIN providers p ON p.id = c.provider_id
			 ORDER BY c.updated_at DESC
			 LIMIT ?`,
			limit,
		)
	} else {
		needle := "%" + strings.ToLower(trimmed) + "%"
		rows, err = s.db.QueryContext(
			ctx,
			`SELECT c.id, c.title, COALESCE(c.provider_id, 0), COALESCE(p.name, ''), c.model,
			        COALESCE(c.temperature, 0.7), COALESCE(c.max_tokens, 0), COALESCE(c.system_prompt, ''), c.updated_at
			 FROM conversations c
			 LEFT JOIN providers p ON p.id = c.provider_id
			 WHERE LOWER(c.title) LIKE ?
			    OR EXISTS (
			      SELECT 1 FROM messages m
			      WHERE m.conversation_id = c.id
			        AND LOWER(m.content) LIKE ?
			    )
			 ORDER BY c.updated_at DESC
			 LIMIT ?`,
			needle,
			needle,
			limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ConversationSummary, 0, limit)
	for rows.Next() {
		var item ConversationSummary
		if err := rows.Scan(&item.ID, &item.Title, &item.ProviderID, &item.Provider, &item.Model, &item.Temperature, &item.MaxTokens, &item.SystemPrompt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// RenameConversation updates title and refreshes updated_at for one conversation.
func (s *Service) RenameConversation(ctx context.Context, payload RenameConversationPayload) (ConversationSummary, error) {
	if payload.ConversationID <= 0 {
		return ConversationSummary{}, fmt.Errorf("conversation id must be > 0")
	}
	title := strings.TrimSpace(payload.Title)
	if title == "" {
		return ConversationSummary{}, fmt.Errorf("title is required")
	}

	res, err := s.db.ExecContext(
		ctx,
		`UPDATE conversations SET title = ?, updated_at = datetime('now') WHERE id = ?`,
		title,
		payload.ConversationID,
	)
	if err != nil {
		return ConversationSummary{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return ConversationSummary{}, err
	}
	if affected == 0 {
		return ConversationSummary{}, fmt.Errorf("conversation %d not found", payload.ConversationID)
	}

	item := ConversationSummary{}
	err = s.db.QueryRowContext(
		ctx,
		`SELECT c.id, c.title, COALESCE(c.provider_id, 0), COALESCE(p.name, ''), c.model,
		        COALESCE(c.temperature, 0.7), COALESCE(c.max_tokens, 0), COALESCE(c.system_prompt, ''), c.updated_at
		 FROM conversations c
		 LEFT JOIN providers p ON p.id = c.provider_id
		 WHERE c.id = ?`,
		payload.ConversationID,
	).Scan(&item.ID, &item.Title, &item.ProviderID, &item.Provider, &item.Model, &item.Temperature, &item.MaxTokens, &item.SystemPrompt, &item.UpdatedAt)
	if err != nil {
		return ConversationSummary{}, err
	}

	return item, nil
}

// UpdateConversationSettings updates provider/model/runtime settings for one conversation.
func (s *Service) UpdateConversationSettings(ctx context.Context, payload UpdateConversationSettingsPayload) (ConversationSummary, error) {
	if payload.ConversationID <= 0 {
		return ConversationSummary{}, fmt.Errorf("conversation id must be > 0")
	}

	model := strings.TrimSpace(payload.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}
	temperature := payload.Temperature
	if temperature <= 0 {
		temperature = 0.7
	}
	if temperature > 2 {
		temperature = 2
	}
	maxTokens := payload.MaxTokens
	if maxTokens < 0 {
		maxTokens = 0
	}
	systemPrompt := strings.TrimSpace(payload.SystemPrompt)

	res, err := s.db.ExecContext(
		ctx,
		`UPDATE conversations
		 SET provider_id = NULLIF(?, 0),
		     model = ?,
		     temperature = ?,
		     max_tokens = ?,
		     system_prompt = ?,
		     updated_at = datetime('now')
		 WHERE id = ?`,
		payload.ProviderID,
		model,
		temperature,
		maxTokens,
		systemPrompt,
		payload.ConversationID,
	)
	if err != nil {
		return ConversationSummary{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return ConversationSummary{}, err
	}
	if affected == 0 {
		return ConversationSummary{}, fmt.Errorf("conversation %d not found", payload.ConversationID)
	}

	item := ConversationSummary{}
	err = s.db.QueryRowContext(
		ctx,
		`SELECT c.id, c.title, COALESCE(c.provider_id, 0), COALESCE(p.name, ''), c.model,
		        COALESCE(c.temperature, 0.7), COALESCE(c.max_tokens, 0), COALESCE(c.system_prompt, ''), c.updated_at
		 FROM conversations c
		 LEFT JOIN providers p ON p.id = c.provider_id
		 WHERE c.id = ?`,
		payload.ConversationID,
	).Scan(&item.ID, &item.Title, &item.ProviderID, &item.Provider, &item.Model, &item.Temperature, &item.MaxTokens, &item.SystemPrompt, &item.UpdatedAt)
	if err != nil {
		return ConversationSummary{}, err
	}

	return item, nil
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
		`SELECT id, role, content, token_stats, created_at
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
		var rawTokenStats string
		if err := rows.Scan(&item.ID, &item.Role, &item.Content, &rawTokenStats, &item.CreatedAt); err != nil {
			return nil, err
		}
		if rawTokenStats != "" {
			_ = json.Unmarshal([]byte(rawTokenStats), &item.TokenStats)
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
	ConversationID int64              `json:"conversation_id"`
	Role           string             `json:"role"`
	Content        string             `json:"content"`
	TokenStats     *MessageTokenStats `json:"token_stats,omitempty"`
}

func estimateTokenCount(text string) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}
	estimate := len(trimmed) / 4
	if estimate < 1 {
		return 1
	}
	return estimate
}

func roundToOneDecimal(value float64) float64 {
	return math.Round(value*10) / 10
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

	stats := payload.TokenStats
	if stats == nil && payload.Role == "user" {
		stats = &MessageTokenStats{
			TokensIn:  estimateTokenCount(payload.Content),
			Estimated: true,
		}
	}
	rawTokenStats := "{}"
	if stats != nil {
		encoded, err := json.Marshal(stats)
		if err != nil {
			return err
		}
		rawTokenStats = string(encoded)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO messages (conversation_id, role, content, token_stats) VALUES (?, ?, ?, ?)`,
		payload.ConversationID,
		payload.Role,
		payload.Content,
		rawTokenStats,
	); err != nil {
		return err
	}

	if payload.Role == "user" {
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE conversations
			 SET title = CASE
			   WHEN title = 'New chat' THEN SUBSTR(TRIM(REPLACE(?, char(10), ' ')), 1, 80)
			   ELSE title
			 END
			 WHERE id = ?`,
			payload.Content,
			payload.ConversationID,
		); err != nil {
			return err
		}
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
