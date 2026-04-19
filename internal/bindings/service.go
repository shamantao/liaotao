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

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Service centralizes backend methods exposed to the frontend.
type Service struct {
	db           *sql.DB
	client       *http.Client
	cancelMu     sync.Mutex
	cancels      map[string]context.CancelFunc
	allowedRoots []string // path sandbox for built-in read_file tool
	app          *application.App
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

// SetApp provides the Wails application instance, enabling native OS dialogs.
func (s *Service) SetApp(app *application.App) {
	s.app = app
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
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	ProjectID    int64     `json:"project_id"`
	Project      string    `json:"project"`
	ProviderID   int64     `json:"provider_id"` // numeric FK; 0 = no provider
	Provider     string    `json:"provider"`    // resolved display name
	Model        string    `json:"model"`
	Temperature  float64   `json:"temperature"`
	MaxTokens    int       `json:"max_tokens"`
	SystemPrompt string    `json:"system_prompt"`
	UpdatedAt    string    `json:"updated_at"`
	TokenCount   int64     `json:"token_count"` // CONV2-05: estimated total tokens
	Tags         []TagItem `json:"tags"`        // CONV2-02: user-defined tags
}

// TagItem is a lightweight tag descriptor returned inside ConversationSummary.
type TagItem struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// CreateConversationPayload is the frontend request payload to create a conversation.
type CreateConversationPayload struct {
	Title      string `json:"title"`
	ProjectID  int64  `json:"project_id"`
	ProviderID int64  `json:"provider_id"` // numeric ID from providers table; 0 = no provider
	Model      string `json:"model"`
}

// ListConversationsPayload controls paging for conversation listing.
type ListConversationsPayload struct {
	Limit     int   `json:"limit"`
	ProjectID int64 `json:"project_id"`
}

// SearchConversationsPayload controls title/content search for conversations.
type SearchConversationsPayload struct {
	Query     string `json:"query"`
	Limit     int    `json:"limit"`
	ProjectID int64  `json:"project_id"`
}

// RenameConversationPayload renames one conversation.
type RenameConversationPayload struct {
	ConversationID int64  `json:"conversation_id"`
	Title          string `json:"title"`
}

// AssignConversationGroupPayload assigns one conversation to a project/group.
// project_id <= 0 means fallback to default project (unassigned in Groups UI).
type AssignConversationGroupPayload struct {
	ConversationID int64 `json:"conversation_id"`
	ProjectID      int64 `json:"project_id"`
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

// conversationSelectSQL is the canonical SELECT column list for conversation queries.
// It computes token_count inline so callers always get a populated value.
const conversationSelectSQL = `
	c.id, c.title,
	COALESCE(c.project_id, 1), COALESCE(pr.name, ''),
	COALESCE(c.provider_id, 0), COALESCE(p.name, ''),
	c.model, COALESCE(c.temperature, 0.7), COALESCE(c.max_tokens, 0),
	COALESCE(c.system_prompt, ''), c.updated_at,
	COALESCE((
		SELECT SUM(
			COALESCE(json_extract(m.token_stats, '$.tokens_in'),  0) +
			COALESCE(json_extract(m.token_stats, '$.tokens_out'), 0)
		) FROM messages m WHERE m.conversation_id = c.id
	), 0)`

// scanConversationSummary scans the conversationSelectSQL columns into item.
// Tags are left empty and filled by a separate batch query when needed.
func scanConversationSummary(scanner interface{ Scan(...any) error }, item *ConversationSummary) error {
	return scanner.Scan(
		&item.ID, &item.Title,
		&item.ProjectID, &item.Project,
		&item.ProviderID, &item.Provider,
		&item.Model, &item.Temperature, &item.MaxTokens,
		&item.SystemPrompt, &item.UpdatedAt,
		&item.TokenCount,
	)
}

// loadTagsForConversations enriches a slice of ConversationSummary with tags.
func (s *Service) loadTagsForConversations(ctx context.Context, items []ConversationSummary) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]any, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	rows, err := s.db.QueryContext(ctx,
		`SELECT ct.conversation_id, t.id, t.name, t.color
		 FROM conversation_tags ct
		 JOIN tags t ON t.id = ct.tag_id
		 WHERE ct.conversation_id IN (`+placeholders+`)
		 ORDER BY t.name`,
		ids...,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	tagMap := make(map[int64][]TagItem)
	for rows.Next() {
		var convID int64
		var tag TagItem
		if err := rows.Scan(&convID, &tag.ID, &tag.Name, &tag.Color); err != nil {
			return err
		}
		tagMap[convID] = append(tagMap[convID], tag)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range items {
		if tags, ok := tagMap[items[i].ID]; ok {
			items[i].Tags = tags
		} else {
			items[i].Tags = []TagItem{}
		}
	}
	return nil
}

func (s *Service) CreateConversation(ctx context.Context, payload CreateConversationPayload) (ConversationSummary, error) {
	title := payload.Title
	if title == "" {
		title = "New chat"
	}
	model := payload.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	defaultSystemPrompt := s.getSettingValue(ctx, "default_system_prompt", "")
	projectID := payload.ProjectID
	if projectID <= 0 {
		var err error
		projectID, err = s.getDefaultProjectID(ctx)
		if err != nil {
			return ConversationSummary{}, err
		}
	}

	// NULLIF(provider_id, 0) stores NULL when no provider is selected.
	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO conversations (title, project_id, provider_id, model, temperature, max_tokens, system_prompt)
		 VALUES (?, ?, NULLIF(?, 0), ?, 0.7, 0, ?)`,
		title,
		projectID,
		payload.ProviderID,
		model,
		defaultSystemPrompt,
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
		`SELECT`+conversationSelectSQL+`
		 FROM conversations c
		 LEFT JOIN projects pr ON pr.id = c.project_id
		 LEFT JOIN providers p ON p.id = c.provider_id
		 WHERE c.id = ?`,
		id,
	).Scan(&item.ID, &item.Title, &item.ProjectID, &item.Project, &item.ProviderID, &item.Provider, &item.Model, &item.Temperature, &item.MaxTokens, &item.SystemPrompt, &item.UpdatedAt, &item.TokenCount)
	if err != nil {
		return ConversationSummary{}, err
	}
	item.Tags = []TagItem{}

	return item, nil
}

// ListConversations returns conversations ordered by latest activity.

func (s *Service) ListConversations(ctx context.Context, payload ListConversationsPayload) ([]ConversationSummary, error) {
	return s.listConversationsWithQuery(ctx, "", payload.Limit, payload.ProjectID)
}

// SearchConversations returns conversations matching title or message content.
func (s *Service) SearchConversations(ctx context.Context, payload SearchConversationsPayload) ([]ConversationSummary, error) {
	return s.listConversationsWithQuery(ctx, payload.Query, payload.Limit, payload.ProjectID)
}

func (s *Service) listConversationsWithQuery(ctx context.Context, query string, limit int, projectID int64) ([]ConversationSummary, error) {
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
			`SELECT`+conversationSelectSQL+`
			 FROM conversations c
			 LEFT JOIN projects pr ON pr.id = c.project_id
			 LEFT JOIN providers p ON p.id = c.provider_id
			 WHERE (? <= 0 OR c.project_id = ?)
			 ORDER BY c.updated_at DESC
			 LIMIT ?`,
			projectID,
			projectID,
			limit,
		)
	} else {
		// Use FTS5 for fast full-text search across message content; fall back
		// to a LIKE title match as well so partial title queries still work.
		trimmedQ := trimmed + "*"
		needle := "%" + strings.ToLower(trimmed) + "%"
		rows, err = s.db.QueryContext(
			ctx,
			`SELECT`+conversationSelectSQL+`
			 FROM conversations c
			 LEFT JOIN projects pr ON pr.id = c.project_id
			 LEFT JOIN providers p ON p.id = c.provider_id
			 WHERE (? <= 0 OR c.project_id = ?)
			   AND (
			       LOWER(c.title) LIKE ?
			    OR c.id IN (
			         SELECT DISTINCT m.conversation_id
			         FROM messages_fts
			         JOIN messages m ON m.id = messages_fts.rowid
			         WHERE messages_fts MATCH ?
			       )
			    )
			 ORDER BY c.updated_at DESC
			 LIMIT ?`,
			projectID,
			projectID,
			needle,
			trimmedQ,
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

// RenameConversation updates only the title — updated_at is preserved so ordering
// reflects the last message activity, not metadata edits.
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
		`UPDATE conversations SET title = ? WHERE id = ?`,
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
		`SELECT`+conversationSelectSQL+`
		 FROM conversations c
		 LEFT JOIN projects pr ON pr.id = c.project_id
		 LEFT JOIN providers p ON p.id = c.provider_id
		 WHERE c.id = ?`,
		payload.ConversationID,
	).Scan(&item.ID, &item.Title, &item.ProjectID, &item.Project, &item.ProviderID, &item.Provider, &item.Model, &item.Temperature, &item.MaxTokens, &item.SystemPrompt, &item.UpdatedAt, &item.TokenCount)
	if err != nil {
		return ConversationSummary{}, err
	}
	item.Tags = []TagItem{}

	return item, nil
}

// AssignConversationGroup re-assigns one conversation to a project/group.
// project_id <= 0 maps to default project.
func (s *Service) AssignConversationGroup(ctx context.Context, payload AssignConversationGroupPayload) (ConversationSummary, error) {
	if payload.ConversationID <= 0 {
		return ConversationSummary{}, fmt.Errorf("conversation id must be > 0")
	}

	targetProjectID := payload.ProjectID
	if targetProjectID <= 0 {
		var err error
		targetProjectID, err = s.getDefaultProjectID(ctx)
		if err != nil {
			return ConversationSummary{}, err
		}
	} else {
		if _, err := s.getProjectByID(ctx, targetProjectID); err != nil {
			return ConversationSummary{}, err
		}
	}

	res, err := s.db.ExecContext(
		ctx,
		`UPDATE conversations
		 SET project_id = ?,
		     updated_at = datetime('now')
		 WHERE id = ?`,
		targetProjectID,
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
		`SELECT`+conversationSelectSQL+`
		 FROM conversations c
		 LEFT JOIN projects pr ON pr.id = c.project_id
		 LEFT JOIN providers p ON p.id = c.provider_id
		 WHERE c.id = ?`,
		payload.ConversationID,
	).Scan(&item.ID, &item.Title, &item.ProjectID, &item.Project, &item.ProviderID, &item.Provider, &item.Model, &item.Temperature, &item.MaxTokens, &item.SystemPrompt, &item.UpdatedAt, &item.TokenCount)
	if err != nil {
		return ConversationSummary{}, err
	}
	item.Tags = []TagItem{}

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
		`SELECT`+conversationSelectSQL+`
		 FROM conversations c
		 LEFT JOIN projects pr ON pr.id = c.project_id
		 LEFT JOIN providers p ON p.id = c.provider_id
		 WHERE c.id = ?`,
		payload.ConversationID,
	).Scan(&item.ID, &item.Title, &item.ProjectID, &item.Project, &item.ProviderID, &item.Provider, &item.Model, &item.Temperature, &item.MaxTokens, &item.SystemPrompt, &item.UpdatedAt, &item.TokenCount)
	if err != nil {
		return ConversationSummary{}, err
	}
	item.Tags = []TagItem{}

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

// DeleteMessagePayload identifies one message to remove from a conversation.
type DeleteMessagePayload struct {
	ConversationID int64 `json:"conversation_id"`
	MessageID      int64 `json:"message_id"`
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

	if err := tx.Commit(); err != nil {
		return err
	}

	// ConversationMemory is best-effort and must not block chat persistence.
	_ = s.upsertConversationMemory(ctx, payload.ConversationID)

	return nil
}

// DeleteMessage removes one persisted message and refreshes the parent conversation timestamp.
func (s *Service) DeleteMessage(ctx context.Context, payload DeleteMessagePayload) (map[string]any, error) {
	if payload.ConversationID <= 0 {
		return nil, fmt.Errorf("conversation_id must be > 0")
	}
	if payload.MessageID <= 0 {
		return nil, fmt.Errorf("message_id must be > 0")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(
		ctx,
		`DELETE FROM messages WHERE id = ? AND conversation_id = ?`,
		payload.MessageID,
		payload.ConversationID,
	)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return map[string]any{"ok": false, "reason": "message_not_found"}, nil
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE conversations SET updated_at = datetime('now') WHERE id = ?`,
		payload.ConversationID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	_ = s.upsertConversationMemory(ctx, payload.ConversationID)

	return map[string]any{"ok": true}, nil
}

// DeleteConversation removes a conversation and all messages (FK cascade).
func (s *Service) DeleteConversation(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("conversation id must be > 0")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM conversations WHERE id = ?`, id)
	return err
}
