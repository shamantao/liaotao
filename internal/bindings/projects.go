// projects.go -- Project management bindings for MVP v2.4.
// Exposes project list/create/rename/archive and default-project helpers.

package bindings

import (
	"context"
	"fmt"
	"strings"
)

const defaultProjectName = "Unsorted"

// ProjectRecord is returned to the frontend for project selectors and management.
type ProjectRecord struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Archived    bool   `json:"archived"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
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
	query := `SELECT id, name, description, archived, created_at, updated_at
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
		var archivedInt int
		if err := rows.Scan(&rec.ID, &rec.Name, &rec.Description, &archivedInt, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		rec.Archived = archivedInt != 0
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
		`INSERT INTO projects (name, description, archived, updated_at) VALUES (?, ?, 0, datetime('now'))`,
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
	var archivedInt int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, archived, created_at, updated_at FROM projects WHERE id = ?`,
		id,
	).Scan(&rec.ID, &rec.Name, &rec.Description, &archivedInt, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		return ProjectRecord{}, err
	}
	rec.Archived = archivedInt != 0
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
		`INSERT OR IGNORE INTO projects (id, name, description, archived) VALUES (1, ?, 'Default project', 0)`,
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
