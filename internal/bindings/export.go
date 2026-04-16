/*
  export.go -- Conversation and project export bindings (EXP-01 to EXP-05 / CONV2-03).
  Supports Markdown and JSON formats for single conversation or all conversations
  in a project. Files are written to the user's home directory ~/liaotao-exports/.
*/

package bindings

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExportFormat identifies the output format for an export operation.
type ExportFormat string

const (
	ExportFormatMarkdown ExportFormat = "markdown"
	ExportFormatJSON     ExportFormat = "json"
)

// ExportConversationPayload is the frontend request to export one conversation.
type ExportConversationPayload struct {
	ConversationID int64        `json:"conversation_id"`
	Format         ExportFormat `json:"format"`
}

// ExportProjectPayload is the frontend request to export all conversations in a project.
type ExportProjectPayload struct {
	ProjectID int64        `json:"project_id"`
	Format    ExportFormat `json:"format"`
}

// ExportResult is returned to the frontend after a successful export.
type ExportResult struct {
	FilePath  string `json:"file_path"`
	Format    string `json:"format"`
	ItemCount int    `json:"item_count"`
}

// ExportConversation exports a single conversation to a file and returns its path.
func (s *Service) ExportConversation(ctx context.Context, payload ExportConversationPayload) (ExportResult, error) {
	if payload.ConversationID <= 0 {
		return ExportResult{}, fmt.Errorf("conversation_id must be > 0")
	}

	msgs, err := s.ListMessages(ctx, ListMessagesPayload{
		ConversationID: payload.ConversationID,
		Limit:          10000,
	})
	if err != nil {
		return ExportResult{}, fmt.Errorf("list messages: %w", err)
	}

	// Fetch conversation metadata.
	var convTitle, convModel string
	_ = s.db.QueryRowContext(ctx,
		`SELECT title, model FROM conversations WHERE id = ?`, payload.ConversationID,
	).Scan(&convTitle, &convModel)
	if convTitle == "" {
		convTitle = fmt.Sprintf("conversation-%d", payload.ConversationID)
	}

	outDir, err := ensureExportDir()
	if err != nil {
		return ExportResult{}, err
	}

	ts := time.Now().Format("20060102-150405")
	slug := sanitizeFilename(convTitle)
	format := normalizeFormat(payload.Format)

	var (
		content []byte
		ext     string
	)
	switch format {
	case ExportFormatJSON:
		ext = "json"
		content, err = marshalConversationJSON(payload.ConversationID, convTitle, convModel, msgs)
	default:
		ext = "md"
		content = marshalConversationMarkdown(convTitle, convModel, msgs)
	}
	if err != nil {
		return ExportResult{}, err
	}

	outPath := filepath.Join(outDir, fmt.Sprintf("%s-%s.%s", slug, ts, ext))
	if err := os.WriteFile(outPath, content, 0o644); err != nil {
		return ExportResult{}, fmt.Errorf("write export: %w", err)
	}

	return ExportResult{
		FilePath:  outPath,
		Format:    string(format),
		ItemCount: len(msgs),
	}, nil
}

// ExportProject exports all conversations in a project into a single JSON or
// Markdown file (one section per conversation).
func (s *Service) ExportProject(ctx context.Context, payload ExportProjectPayload) (ExportResult, error) {
	if payload.ProjectID <= 0 {
		return ExportResult{}, fmt.Errorf("project_id must be > 0")
	}

	convs, err := s.listConversationsWithQuery(ctx, "", 1000, payload.ProjectID)
	if err != nil {
		return ExportResult{}, fmt.Errorf("list conversations: %w", err)
	}

	outDir, err := ensureExportDir()
	if err != nil {
		return ExportResult{}, err
	}

	format := normalizeFormat(payload.Format)
	ts := time.Now().Format("20060102-150405")

	// Fetch project name for the filename.
	var projName string
	_ = s.db.QueryRowContext(ctx,
		`SELECT name FROM projects WHERE id = ?`, payload.ProjectID,
	).Scan(&projName)
	if projName == "" {
		projName = fmt.Sprintf("project-%d", payload.ProjectID)
	}
	slug := sanitizeFilename(projName)

	type convExport struct {
		ID       int64            `json:"id"`
		Title    string           `json:"title"`
		Model    string           `json:"model"`
		Messages []MessageSummary `json:"messages"`
	}

	totalMessages := 0
	var sb strings.Builder
	var jsonExports []convExport

	for _, conv := range convs {
		msgs, err := s.ListMessages(ctx, ListMessagesPayload{
			ConversationID: conv.ID,
			Limit:          10000,
		})
		if err != nil {
			return ExportResult{}, fmt.Errorf("list messages for conv %d: %w", conv.ID, err)
		}
		totalMessages += len(msgs)

		if format == ExportFormatJSON {
			jsonExports = append(jsonExports, convExport{
				ID:       conv.ID,
				Title:    conv.Title,
				Model:    conv.Model,
				Messages: msgs,
			})
		} else {
			sb.WriteString(renderMarkdownConversation(conv.Title, conv.Model, msgs))
			sb.WriteString("\n\n---\n\n")
		}
	}

	var (
		content []byte
		ext     string
	)
	switch format {
	case ExportFormatJSON:
		ext = "json"
		content, err = json.MarshalIndent(jsonExports, "", "  ")
		if err != nil {
			return ExportResult{}, err
		}
	default:
		ext = "md"
		content = []byte(sb.String())
	}

	outPath := filepath.Join(outDir, fmt.Sprintf("%s-%s.%s", slug, ts, ext))
	if err := os.WriteFile(outPath, content, 0o644); err != nil {
		return ExportResult{}, fmt.Errorf("write export: %w", err)
	}

	return ExportResult{
		FilePath:  outPath,
		Format:    string(format),
		ItemCount: totalMessages,
	}, nil
}

// ── helpers ────────────────────────────────────────────────────────────────

func ensureExportDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, "liaotao-exports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create export dir: %w", err)
	}
	return dir, nil
}

func normalizeFormat(f ExportFormat) ExportFormat {
	if strings.ToLower(string(f)) == "json" {
		return ExportFormatJSON
	}
	return ExportFormatMarkdown
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	result := b.String()
	if len(result) > 60 {
		result = result[:60]
	}
	if result == "" {
		result = "export"
	}
	return result
}

type jsonConversationExport struct {
	ID       int64            `json:"id"`
	Title    string           `json:"title"`
	Model    string           `json:"model"`
	Messages []MessageSummary `json:"messages"`
}

func marshalConversationJSON(id int64, title, model string, msgs []MessageSummary) ([]byte, error) {
	return json.MarshalIndent(jsonConversationExport{
		ID:       id,
		Title:    title,
		Model:    model,
		Messages: msgs,
	}, "", "  ")
}

func marshalConversationMarkdown(title, model string, msgs []MessageSummary) []byte {
	return []byte(renderMarkdownConversation(title, model, msgs))
}

func renderMarkdownConversation(title, model string, msgs []MessageSummary) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	if model != "" {
		sb.WriteString(fmt.Sprintf("*Model: %s*\n\n", model))
	}
	for _, m := range msgs {
		role := strings.ToUpper(m.Role[:1]) + m.Role[1:]
		sb.WriteString(fmt.Sprintf("**%s**\n\n%s\n\n", role, m.Content))
	}
	return sb.String()
}
