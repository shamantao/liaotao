/*
  retrieval_local.go -- SQLite-backed local retrieval backend.
  Responsibilities: chunk attachment text into knowledge_items and rank snippets
  by lexical overlap for request-time retrieval.
*/

package bindings

import (
	"context"
	"database/sql"
	"strings"
)

type localRetrievalBackend struct {
	db *sql.DB
}

func (b *localRetrievalBackend) IndexAttachment(ctx context.Context, input KnowledgeIndexInput) error {
	content := strings.TrimSpace(input.Content)
	if input.AttachmentID <= 0 || input.ConversationID <= 0 || input.ProjectID <= 0 || content == "" {
		return nil
	}

	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM knowledge_items WHERE attachment_id = ?`, input.AttachmentID); err != nil {
		return err
	}

	chunks := chunkText(content, 900, 120)
	if len(chunks) == 0 {
		return tx.Commit()
	}

	for idx, chunk := range chunks {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO knowledge_items
			 (conversation_id, project_id, attachment_id, scope, source_name, source_path, mime_type, chunk_index, chunk_text)
			 VALUES (?, ?, ?, 'conversation', ?, ?, ?, ?, ?)`,
			input.ConversationID,
			input.ProjectID,
			input.AttachmentID,
			input.SourceName,
			input.SourcePath,
			input.MimeType,
			idx,
			chunk,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (b *localRetrievalBackend) Retrieve(ctx context.Context, query RetrievalQuery) ([]RetrievedSnippet, error) {
	if query.ConversationID <= 0 || strings.TrimSpace(query.Query) == "" {
		return []RetrievedSnippet{}, nil
	}
	k := query.TopK
	if k <= 0 {
		k = 4
	}

	rows, err := b.db.QueryContext(
		ctx,
		`SELECT id, source_name, source_path, chunk_index, chunk_text
		 FROM knowledge_items
		 WHERE (scope = 'conversation' AND conversation_id = ?)
		    OR (scope = 'project' AND ? > 0 AND project_id = ?)
		 ORDER BY id DESC
		 LIMIT 600`,
		query.ConversationID,
		query.ProjectID,
		query.ProjectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	terms := tokenizeQuery(query.Query)
	if len(terms) == 0 {
		return []RetrievedSnippet{}, nil
	}

	out := make([]RetrievedSnippet, 0, k*3)
	for rows.Next() {
		var item RetrievedSnippet
		if err := rows.Scan(&item.KnowledgeItemID, &item.SourceName, &item.SourcePath, &item.ChunkIndex, &item.Content); err != nil {
			return nil, err
		}
		score := lexicalScore(item.Content, terms)
		if score <= 0 {
			continue
		}
		item.Score = score
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return []RetrievedSnippet{}, nil
	}
	stableSortSnippets(out)
	if len(out) > k {
		out = out[:k]
	}
	return out, nil
}

func chunkText(content string, chunkSize, overlap int) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || chunkSize <= 0 {
		return nil
	}
	runes := []rune(trimmed)
	if len(runes) <= chunkSize {
		return []string{trimmed}
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 4
	}
	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize
	}

	chunks := make([]string, 0, len(runes)/step+1)
	for start := 0; start < len(runes); start += step {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func lexicalScore(content string, terms []string) float64 {
	text := strings.ToLower(content)
	score := 0.0
	for _, term := range terms {
		if term == "" {
			continue
		}
		hits := strings.Count(text, term)
		if hits > 0 {
			score += float64(hits)
		}
	}
	if score == 0 {
		return 0
	}
	lengthPenalty := 1.0 + float64(len(content))/2000.0
	return score / lengthPenalty
}

func stableSortSnippets(items []RetrievedSnippet) {
	for i := 0; i < len(items)-1; i++ {
		best := i
		for j := i + 1; j < len(items); j++ {
			if items[j].Score > items[best].Score {
				best = j
			}
		}
		if best != i {
			items[i], items[best] = items[best], items[i]
		}
	}
}
