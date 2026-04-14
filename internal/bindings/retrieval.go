/*
  retrieval.go -- Request context and retrieval orchestration.
  Responsibilities: define retrieval domain objects/interfaces, build RequestContext,
  and inject knowledge snippets into user prompts before provider calls.
*/

package bindings

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
)

// RequestContextMessage is a normalized conversation message for provider context.
type RequestContextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RetrievedSnippet is one retrieval hit selected for prompt enrichment.
type RetrievedSnippet struct {
	KnowledgeItemID int64   `json:"knowledge_item_id"`
	SourceName      string  `json:"source_name"`
	SourcePath      string  `json:"source_path"`
	ChunkIndex      int     `json:"chunk_index"`
	Score           float64 `json:"score"`
	Content         string  `json:"content"`
}

// RequestContext is the computed provider payload context (OBJ-05).
type RequestContext struct {
	ConversationID int64                   `json:"conversation_id"`
	ProjectID      int64                   `json:"project_id"`
	Query          string                  `json:"query"`
	RecentMessages []RequestContextMessage `json:"recent_messages"`
	MemorySummary  string                  `json:"memory_summary"`
	Snippets       []RetrievedSnippet      `json:"snippets"`
}

// KnowledgeIndexInput contains one source document to split/index.
type KnowledgeIndexInput struct {
	ConversationID int64
	ProjectID      int64
	AttachmentID   int64
	SourceName     string
	SourcePath     string
	MimeType       string
	Content        string
}

// RetrievalQuery describes a retrieval request for the active chat turn.
type RetrievalQuery struct {
	ConversationID int64
	ProjectID      int64
	Query          string
	TopK           int
}

// RetrievalBackend is the pluggable retrieval interface (OBJ-07).
type RetrievalBackend interface {
	IndexAttachment(ctx context.Context, input KnowledgeIndexInput) error
	Retrieve(ctx context.Context, query RetrievalQuery) ([]RetrievedSnippet, error)
}

var splitWordsRegex = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func (s *Service) retrievalBackend() RetrievalBackend {
	return &localRetrievalBackend{db: s.db}
}

type noopExternalRetrievalBackend struct{}

func (b *noopExternalRetrievalBackend) IndexAttachment(_ context.Context, _ KnowledgeIndexInput) error {
	return nil
}

func (b *noopExternalRetrievalBackend) Retrieve(_ context.Context, _ RetrievalQuery) ([]RetrievedSnippet, error) {
	return []RetrievedSnippet{}, nil
}

func (s *Service) retrievalBackendForProject(ctx context.Context, projectID int64) RetrievalBackend {
	backendKey := "local"
	if projectID > 0 {
		_ = s.db.QueryRowContext(
			ctx,
			`SELECT COALESCE(retrieval_backend, 'local') FROM projects WHERE id = ?`,
			projectID,
		).Scan(&backendKey)
	}
	if strings.EqualFold(strings.TrimSpace(backendKey), "external") {
		return &noopExternalRetrievalBackend{}
	}
	return s.retrievalBackend()
}

func (s *Service) buildRequestContext(ctx context.Context, conversationID int64, prompt string) (RequestContext, error) {
	if conversationID <= 0 {
		return RequestContext{}, fmt.Errorf("conversation_id must be > 0")
	}

	var projectID int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(project_id, 1) FROM conversations WHERE id = ?`, conversationID).Scan(&projectID); err != nil {
		return RequestContext{}, err
	}

	recent, err := s.loadRecentMessagesForContext(ctx, conversationID, 8)
	if err != nil {
		return RequestContext{}, err
	}
	memorySummary, err := s.loadConversationMemorySummary(ctx, conversationID)
	if err != nil {
		return RequestContext{}, err
	}
	if memorySummary == "" {
		memorySummary, err = s.buildConversationMemorySummary(ctx, conversationID, 8, 16)
		if err != nil {
			return RequestContext{}, err
		}
		if upsertErr := s.upsertConversationMemory(ctx, conversationID); upsertErr != nil {
			slog.Warn("upsert conversation memory failed", "conversation_id", conversationID, "err", upsertErr)
		}
	}

	snippets, err := s.retrievalBackendForProject(ctx, projectID).Retrieve(ctx, RetrievalQuery{
		ConversationID: conversationID,
		ProjectID:      projectID,
		Query:          prompt,
		TopK:           4,
	})
	if err != nil {
		return RequestContext{}, err
	}

	return RequestContext{
		ConversationID: conversationID,
		ProjectID:      projectID,
		Query:          prompt,
		RecentMessages: recent,
		MemorySummary:  memorySummary,
		Snippets:       snippets,
	}, nil
}

func (s *Service) loadConversationMemorySummary(ctx context.Context, conversationID int64) (string, error) {
	var summary string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COALESCE(summary_text, '') FROM conversation_memory WHERE conversation_id = ?`,
		conversationID,
	).Scan(&summary)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(summary), nil
}

func (s *Service) upsertConversationMemory(ctx context.Context, conversationID int64) error {
	if conversationID <= 0 {
		return nil
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT role, content
		 FROM messages
		 WHERE conversation_id = ?
		 ORDER BY id DESC
		 LIMIT 30`,
		conversationID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	tokens := make([]string, 0, 30)
	facts := make([]string, 0, 8)
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return err
		}
		trimmed := strings.TrimSpace(content)
		if trimmed == "" {
			continue
		}
		if len(trimmed) > 160 {
			trimmed = trimmed[:160] + "..."
		}
		tokens = append(tokens, fmt.Sprintf("[%s] %s", normalizeRole(role), trimmed))
		if normalizeRole(role) == "user" && len(facts) < 8 {
			facts = append(facts, trimmed)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i, j := 0, len(tokens)-1; i < j; i, j = i+1, j-1 {
		tokens[i], tokens[j] = tokens[j], tokens[i]
	}
	for i, j := 0, len(facts)-1; i < j; i, j = i+1, j-1 {
		facts[i], facts[j] = facts[j], facts[i]
	}
	if len(tokens) > 8 {
		tokens = tokens[len(tokens)-8:]
	}
	if len(facts) > 6 {
		facts = facts[len(facts)-6:]
	}
	summary := strings.Join(tokens, "\n")

	encodedFacts, err := json.Marshal(facts)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(
		ctx,
		`INSERT INTO conversation_memory (conversation_id, summary_text, facts_json, message_count, updated_at)
		 VALUES (?, ?, ?, ?, datetime('now'))
		 ON CONFLICT(conversation_id) DO UPDATE SET
		   summary_text = excluded.summary_text,
		   facts_json = excluded.facts_json,
		   message_count = excluded.message_count,
		   updated_at = datetime('now')`,
		conversationID,
		summary,
		string(encodedFacts),
		len(tokens),
	); err != nil {
		return err
	}
	return nil
}

func (s *Service) loadRecentMessagesForContext(ctx context.Context, conversationID int64, limit int) ([]RequestContextMessage, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT role, content
		 FROM messages
		 WHERE conversation_id = ?
		 ORDER BY id DESC
		 LIMIT ?`,
		conversationID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tmp := make([]RequestContextMessage, 0, limit)
	for rows.Next() {
		var item RequestContextMessage
		if err := rows.Scan(&item.Role, &item.Content); err != nil {
			return nil, err
		}
		item.Role = normalizeRole(item.Role)
		item.Content = strings.TrimSpace(item.Content)
		if item.Content != "" {
			tmp = append(tmp, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Query is DESC by id; provider payload expects chronological order.
	for i, j := 0, len(tmp)-1; i < j; i, j = i+1, j-1 {
		tmp[i], tmp[j] = tmp[j], tmp[i]
	}
	return tmp, nil
}

func (s *Service) buildConversationMemorySummary(ctx context.Context, conversationID int64, skipRecent, take int) (string, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT role, content
		 FROM messages
		 WHERE conversation_id = ?
		 ORDER BY id DESC
		 LIMIT ? OFFSET ?`,
		conversationID,
		take,
		skipRecent,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	defer rows.Close()

	tokens := make([]string, 0, take)
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return "", err
		}
		trimmed := strings.TrimSpace(content)
		if trimmed == "" {
			continue
		}
		if len(trimmed) > 140 {
			trimmed = trimmed[:140] + "..."
		}
		tokens = append(tokens, fmt.Sprintf("[%s] %s", normalizeRole(role), trimmed))
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(tokens) == 0 {
		return "", nil
	}
	for i, j := 0, len(tokens)-1; i < j; i, j = i+1, j-1 {
		tokens[i], tokens[j] = tokens[j], tokens[i]
	}
	if len(tokens) > 6 {
		tokens = tokens[len(tokens)-6:]
	}
	return strings.Join(tokens, "\n"), nil
}

func normalizeRole(role string) string {
	r := strings.ToLower(strings.TrimSpace(role))
	switch r {
	case "assistant", "system", "tool":
		return r
	default:
		return "user"
	}
}

func requestContextMessagesToOpenAI(items []RequestContextMessage) []openAIChatMessage {
	if len(items) == 0 {
		return nil
	}
	out := make([]openAIChatMessage, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Content) == "" {
			continue
		}
		out = append(out, openAIChatMessage{
			Role:    normalizeRole(item.Role),
			Content: item.Content,
		})
	}
	return out
}

func promptWithRequestContext(prompt string, rc RequestContext) string {
	prompt = strings.TrimSpace(prompt)
	snippets := rc.Snippets
	if len(snippets) > 4 {
		snippets = snippets[:4]
	}
	if rc.MemorySummary == "" && len(snippets) == 0 {
		return prompt
	}

	builder := strings.Builder{}
	builder.WriteString(prompt)
	builder.WriteString("\n\n")
	builder.WriteString("---\n")
	builder.WriteString("Context for this request (use only if relevant):\n")
	if rc.MemorySummary != "" {
		builder.WriteString("Conversation memory:\n")
		builder.WriteString(rc.MemorySummary)
		builder.WriteString("\n")
	}
	if len(snippets) > 0 {
		builder.WriteString("Knowledge snippets:\n")
		for i, snippet := range snippets {
			builder.WriteString(fmt.Sprintf("%d. %s#%d\n", i+1, snippet.SourceName, snippet.ChunkIndex))
			content := strings.TrimSpace(snippet.Content)
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			builder.WriteString(content)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func tokenizeQuery(value string) []string {
	parts := splitWordsRegex.Split(strings.ToLower(strings.TrimSpace(value)), -1)
	filtered := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		if len(p) < 2 {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		filtered = append(filtered, p)
	}
	sort.Strings(filtered)
	return filtered
}

func (s *Service) indexAttachmentKnowledge(ctx context.Context, input KnowledgeIndexInput) {
	if !isIndexableTextSource(input.SourceName, input.MimeType, input.Content) {
		return
	}
	if err := s.retrievalBackendForProject(ctx, input.ProjectID).IndexAttachment(ctx, input); err != nil {
		slog.Warn("retrieval index attachment failed", "source", input.SourceName, "err", err)
	}
}

func isIndexableTextSource(fileName, mimeType, content string) bool {
	name := strings.ToLower(strings.TrimSpace(fileName))
	mime := strings.ToLower(strings.TrimSpace(mimeType))
	if strings.Contains(content, "\x00") {
		return false
	}
	if strings.HasPrefix(mime, "text/") || mime == "application/json" {
		return true
	}
	supported := []string{".txt", ".md", ".json", ".csv", ".py", ".go", ".js", ".ts"}
	for _, ext := range supported {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}
