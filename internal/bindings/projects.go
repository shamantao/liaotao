// projects.go -- Project management bindings for MVP v2.4.
// Exposes project list/create/rename/archive and default-project helpers.

package bindings

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const defaultProjectName = "Unsorted"

// ProjectRecord is returned to the frontend for project selectors and management.
type ProjectRecord struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	Archived          bool   `json:"archived"`
	RetrievalBackend  string `json:"retrieval_backend"`
	RetrievalIndexing bool   `json:"retrieval_indexing"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// SetProjectRetrievalBackendPayload updates retrieval backend strategy for one project.
type SetProjectRetrievalBackendPayload struct {
	ProjectID int64  `json:"project_id"`
	Backend   string `json:"backend"`
}

// ProjectDashboardPayload scopes project dashboard retrieval.
type ProjectDashboardPayload struct {
	ProjectID int64 `json:"project_id"`
}

// ProjectDashboard summarizes project activity and retrieval status (PROJ-06/KR-06).
type ProjectDashboard struct {
	ProjectID             int64  `json:"project_id"`
	ConversationCount     int64  `json:"conversation_count"`
	TotalTokens           int64  `json:"total_tokens"`
	FileCount             int64  `json:"file_count"`
	ProjectKnowledgeCount int64  `json:"project_knowledge_count"`
	RetrievalBackend      string `json:"retrieval_backend"`
	RetrievalStatus       string `json:"retrieval_status"`
}

// ListProjectsPayload controls project listing behavior.
type ListProjectsPayload struct {
	IncludeArchived bool `json:"include_archived"`
}

// CreateProjectPayload creates one project.
type CreateProjectPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RenameProjectPayload renames one project.
type RenameProjectPayload struct {
	ProjectID int64  `json:"project_id"`
	Name      string `json:"name"`
}

// ArchiveProjectPayload archives or unarchives one project.
type ArchiveProjectPayload struct {
	ProjectID int64 `json:"project_id"`
	Archived  bool  `json:"archived"`
}

// ListProjects returns projects ordered by default project first, then by name.
func (s *Service) ListProjects(ctx context.Context, payload ListProjectsPayload) ([]ProjectRecord, error) {
	query := `SELECT id, name, description, archived, COALESCE(retrieval_backend, 'local'), COALESCE(retrieval_indexing, 0), created_at, updated_at
		FROM projects`
	args := make([]any, 0, 1)
	if !payload.IncludeArchived {
		query += ` WHERE archived = 0`
	}
	query += ` ORDER BY CASE WHEN name = ? THEN 0 ELSE 1 END, name ASC`
	args = append(args, defaultProjectName)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ProjectRecord, 0, 16)
	for rows.Next() {
		var rec ProjectRecord
		var archivedInt, indexingInt int
		if err := rows.Scan(&rec.ID, &rec.Name, &rec.Description, &archivedInt, &rec.RetrievalBackend, &indexingInt, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		rec.Archived = archivedInt != 0
		rec.RetrievalIndexing = indexingInt != 0
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateProject creates a new project and returns it.
func (s *Service) CreateProject(ctx context.Context, payload CreateProjectPayload) (ProjectRecord, error) {
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return ProjectRecord{}, fmt.Errorf("project name is required")
	}
	desc := strings.TrimSpace(payload.Description)

	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO projects (name, description, archived, retrieval_backend, retrieval_indexing, updated_at)
		 VALUES (?, ?, 0, 'local', 0, datetime('now'))`,
		name,
		desc,
	)
	if err != nil {
		return ProjectRecord{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ProjectRecord{}, err
	}
	return s.getProjectByID(ctx, id)
}

// RenameProject renames one project.
func (s *Service) RenameProject(ctx context.Context, payload RenameProjectPayload) (ProjectRecord, error) {
	if payload.ProjectID <= 0 {
		return ProjectRecord{}, fmt.Errorf("project_id must be > 0")
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return ProjectRecord{}, fmt.Errorf("project name is required")
	}

	res, err := s.db.ExecContext(
		ctx,
		`UPDATE projects SET name = ?, updated_at = datetime('now') WHERE id = ?`,
		name,
		payload.ProjectID,
	)
	if err != nil {
		return ProjectRecord{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return ProjectRecord{}, err
	}
	if affected == 0 {
		return ProjectRecord{}, fmt.Errorf("project %d not found", payload.ProjectID)
	}
	return s.getProjectByID(ctx, payload.ProjectID)
}

// ArchiveProject archives or unarchives one project.
// When archiving a project, its conversations are moved to the default project.
func (s *Service) ArchiveProject(ctx context.Context, payload ArchiveProjectPayload) (ProjectRecord, error) {
	if payload.ProjectID <= 0 {
		return ProjectRecord{}, fmt.Errorf("project_id must be > 0")
	}

	target, err := s.getProjectByID(ctx, payload.ProjectID)
	if err != nil {
		return ProjectRecord{}, err
	}
	if strings.EqualFold(target.Name, defaultProjectName) && payload.Archived {
		return ProjectRecord{}, fmt.Errorf("default project cannot be archived")
	}

	if payload.Archived {
		defaultID, err := s.getDefaultProjectID(ctx)
		if err != nil {
			return ProjectRecord{}, err
		}

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return ProjectRecord{}, err
		}
		defer tx.Rollback()

		if _, err := tx.ExecContext(ctx,
			`UPDATE conversations SET project_id = ? WHERE project_id = ?`,
			defaultID,
			payload.ProjectID,
		); err != nil {
			return ProjectRecord{}, err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE projects SET archived = 1, updated_at = datetime('now') WHERE id = ?`,
			payload.ProjectID,
		); err != nil {
			return ProjectRecord{}, err
		}
		if err := tx.Commit(); err != nil {
			return ProjectRecord{}, err
		}
	} else {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE projects SET archived = 0, updated_at = datetime('now') WHERE id = ?`,
			payload.ProjectID,
		); err != nil {
			return ProjectRecord{}, err
		}
	}

	return s.getProjectByID(ctx, payload.ProjectID)
}

func (s *Service) getProjectByID(ctx context.Context, id int64) (ProjectRecord, error) {
	var rec ProjectRecord
	var archivedInt, indexingInt int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, archived, COALESCE(retrieval_backend, 'local'), COALESCE(retrieval_indexing, 0), created_at, updated_at
		 FROM projects WHERE id = ?`,
		id,
	).Scan(&rec.ID, &rec.Name, &rec.Description, &archivedInt, &rec.RetrievalBackend, &indexingInt, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		return ProjectRecord{}, err
	}
	rec.Archived = archivedInt != 0
	rec.RetrievalIndexing = indexingInt != 0
	return rec, nil
}

func (s *Service) getDefaultProjectID(ctx context.Context) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM projects WHERE name = ? LIMIT 1`,
		defaultProjectName,
	).Scan(&id)
	if err == nil {
		return id, nil
	}

	res, createErr := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO projects (id, name, description, archived, retrieval_backend, retrieval_indexing)
		 VALUES (1, ?, 'Default project', 0, 'local', 0)`,
		defaultProjectName,
	)
	if createErr != nil {
		return 0, createErr
	}
	insertedID, _ := res.LastInsertId()
	if insertedID > 0 {
		return insertedID, nil
	}

	err = s.db.QueryRowContext(ctx,
		`SELECT id FROM projects WHERE name = ? LIMIT 1`,
		defaultProjectName,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// SetProjectRetrievalBackend persists retrieval backend selection (KR-05).
func (s *Service) SetProjectRetrievalBackend(ctx context.Context, payload SetProjectRetrievalBackendPayload) (ProjectRecord, error) {
	if payload.ProjectID <= 0 {
		return ProjectRecord{}, fmt.Errorf("project_id must be > 0")
	}
	backend := strings.ToLower(strings.TrimSpace(payload.Backend))
	if backend == "" {
		backend = "local"
	}
	if backend != "local" && backend != "external" {
		return ProjectRecord{}, fmt.Errorf("unsupported retrieval backend: %s", backend)
	}

	if _, err := s.db.ExecContext(
		ctx,
		`UPDATE projects SET retrieval_backend = ?, updated_at = datetime('now') WHERE id = ?`,
		backend,
		payload.ProjectID,
	); err != nil {
		return ProjectRecord{}, err
	}
	return s.getProjectByID(ctx, payload.ProjectID)
}

// GetProjectDashboard returns project-level counters and retrieval status (PROJ-06/KR-06).
func (s *Service) GetProjectDashboard(ctx context.Context, payload ProjectDashboardPayload) (ProjectDashboard, error) {
	if payload.ProjectID <= 0 {
		return ProjectDashboard{}, fmt.Errorf("project_id must be > 0")
	}

	dashboard := ProjectDashboard{ProjectID: payload.ProjectID}
	var indexingInt int
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT COALESCE(retrieval_backend, 'local'), COALESCE(retrieval_indexing, 0)
		 FROM projects WHERE id = ?`,
		payload.ProjectID,
	).Scan(&dashboard.RetrievalBackend, &indexingInt); err != nil {
		return ProjectDashboard{}, err
	}

	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM conversations WHERE project_id = ?`,
		payload.ProjectID,
	).Scan(&dashboard.ConversationCount)
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*)
		 FROM attachments a
		 JOIN conversations c ON c.id = a.conversation_id
		 WHERE c.project_id = ?`,
		payload.ProjectID,
	).Scan(&dashboard.FileCount)
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM knowledge_items WHERE project_id = ? AND scope = 'project'`,
		payload.ProjectID,
	).Scan(&dashboard.ProjectKnowledgeCount)

	rows, err := s.db.QueryContext(ctx,
		`SELECT m.token_stats, m.content
		 FROM messages m
		 JOIN conversations c ON c.id = m.conversation_id
		 WHERE c.project_id = ?`,
		payload.ProjectID,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rawStats, content string
			if scanErr := rows.Scan(&rawStats, &content); scanErr != nil {
				continue
			}
			stats := MessageTokenStats{}
			if rawStats != "" {
				_ = json.Unmarshal([]byte(rawStats), &stats)
			}
			estimated := stats.TokensIn + stats.TokensOut
			if estimated <= 0 {
				estimated = estimateTokenCount(content)
			}
			dashboard.TotalTokens += int64(estimated)
		}
	}

	if indexingInt != 0 {
		dashboard.RetrievalStatus = "indexing"
	} else {
		dashboard.RetrievalStatus = "ready"
	}

	return dashboard, nil
}
