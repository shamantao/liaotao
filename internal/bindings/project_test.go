/*
  project_test.go -- Regression tests for project management and conversation filtering.
  Covers OBJ-01/02 and PROJ-01..04 backend behavior.
*/

package bindings

import (
	"context"
	"testing"
)

func TestProject_CRUDAndArchive(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	projects, err := svc.ListProjects(ctx, ListProjectsPayload{})
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) == 0 || projects[0].Name != "Unsorted" {
		t.Fatalf("expected default project Unsorted, got %+v", projects)
	}

	created, err := svc.CreateProject(ctx, CreateProjectPayload{Name: "Alpha", Description: "alpha project"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if created.Name != "Alpha" {
		t.Fatalf("created name mismatch: %+v", created)
	}

	renamed, err := svc.RenameProject(ctx, RenameProjectPayload{ProjectID: created.ID, Name: "Alpha Renamed"})
	if err != nil {
		t.Fatalf("RenameProject: %v", err)
	}
	if renamed.Name != "Alpha Renamed" {
		t.Fatalf("renamed name mismatch: %+v", renamed)
	}

	archived, err := svc.ArchiveProject(ctx, ArchiveProjectPayload{ProjectID: created.ID, Archived: true})
	if err != nil {
		t.Fatalf("ArchiveProject: %v", err)
	}
	if !archived.Archived {
		t.Fatalf("expected project archived=true, got %+v", archived)
	}

	visible, err := svc.ListProjects(ctx, ListProjectsPayload{IncludeArchived: false})
	if err != nil {
		t.Fatalf("ListProjects visible: %v", err)
	}
	for _, p := range visible {
		if p.ID == created.ID {
			t.Fatalf("archived project should not be visible by default")
		}
	}
}

func TestConversation_ProjectFilter(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)

	projA, err := svc.CreateProject(ctx, CreateProjectPayload{Name: "A"})
	if err != nil {
		t.Fatalf("CreateProject A: %v", err)
	}
	projB, err := svc.CreateProject(ctx, CreateProjectPayload{Name: "B"})
	if err != nil {
		t.Fatalf("CreateProject B: %v", err)
	}

	convA, err := svc.CreateConversation(ctx, CreateConversationPayload{Title: "conv-a", ProjectID: projA.ID, Model: "gpt-4o-mini"})
	if err != nil {
		t.Fatalf("CreateConversation A: %v", err)
	}
	if convA.ProjectID != projA.ID {
		t.Fatalf("expected project_id=%d, got %+v", projA.ID, convA)
	}

	_, err = svc.CreateConversation(ctx, CreateConversationPayload{Title: "conv-b", ProjectID: projB.ID, Model: "gpt-4o-mini"})
	if err != nil {
		t.Fatalf("CreateConversation B: %v", err)
	}

	itemsA, err := svc.ListConversations(ctx, ListConversationsPayload{Limit: 50, ProjectID: projA.ID})
	if err != nil {
		t.Fatalf("ListConversations project A: %v", err)
	}
	if len(itemsA) != 1 || itemsA[0].ProjectID != projA.ID {
		t.Fatalf("project filter A mismatch: %+v", itemsA)
	}

	itemsB, err := svc.SearchConversations(ctx, SearchConversationsPayload{Query: "conv", Limit: 50, ProjectID: projB.ID})
	if err != nil {
		t.Fatalf("SearchConversations project B: %v", err)
	}
	if len(itemsB) != 1 || itemsB[0].ProjectID != projB.ID {
		t.Fatalf("project filter B mismatch: %+v", itemsB)
	}
}

func TestProject_DashboardAndRetrievalBackend(t *testing.T) {
	ctx := context.Background()
	database := newConversationTestDB(t)
	svc := NewService(database)
	t.Setenv("LIAOTAO_ATTACHMENTS_DIR", t.TempDir())

	project, err := svc.CreateProject(ctx, CreateProjectPayload{Name: "Dashboard"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	conv, err := svc.CreateConversation(ctx, CreateConversationPayload{Title: "Dash chat", ProjectID: project.ID, Model: "gpt-4o-mini"})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if err := svc.SaveMessage(ctx, MessagePayload{ConversationID: conv.ID, Role: "user", Content: "token sample for dashboard"}); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	if _, err := svc.SetProjectRetrievalBackend(ctx, SetProjectRetrievalBackendPayload{ProjectID: project.ID, Backend: "external"}); err != nil {
		t.Fatalf("SetProjectRetrievalBackend: %v", err)
	}

	dashboard, err := svc.GetProjectDashboard(ctx, ProjectDashboardPayload{ProjectID: project.ID})
	if err != nil {
		t.Fatalf("GetProjectDashboard: %v", err)
	}
	if dashboard.ProjectID != project.ID {
		t.Fatalf("dashboard project mismatch: %+v", dashboard)
	}
	if dashboard.ConversationCount != 1 {
		t.Fatalf("expected 1 conversation, got %d", dashboard.ConversationCount)
	}
	if dashboard.TotalTokens <= 0 {
		t.Fatalf("expected total_tokens > 0, got %d", dashboard.TotalTokens)
	}
	if dashboard.RetrievalBackend != "external" {
		t.Fatalf("expected retrieval backend external, got %q", dashboard.RetrievalBackend)
	}
}
