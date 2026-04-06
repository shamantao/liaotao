/*
  mcp_servers.go -- Wails bindings and DB helpers for MCP server management (MCP-07).
  Responsibilities: CRUD for mcp_servers table, listActiveMCPServers used by dispatcher.
  Exposes ListMCPServers, SaveMCPServer, DeleteMCPServer, ToggleMCPServer to frontend.
*/

package bindings

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// ── DB helpers ────────────────────────────────────────────────────────────

// listActiveMCPServers returns all active MCP servers ordered by id ASC.
// Called by mcp_dispatch.go when routing tool calls to external servers.
func (s *Service) listActiveMCPServers(ctx context.Context) ([]MCPServerConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, transport, url, command, args, active
		FROM mcp_servers WHERE active=1 ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []MCPServerConfig
	for rows.Next() {
		var srv MCPServerConfig
		if err := rows.Scan(&srv.ID, &srv.Name, &srv.Transport, &srv.URL, &srv.Command, &srv.Args, &srv.Active); err != nil {
			return nil, err
		}
		servers = append(servers, srv)
	}
	return servers, rows.Err()
}

// ── Public Wails bindings ─────────────────────────────────────────────────

// ListMCPServers returns all configured MCP servers (active and inactive).
func (s *Service) ListMCPServers(ctx context.Context) ([]MCPServerConfig, error) {
	slog.Debug("ListMCPServers called")
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, transport, url, command, args, active
		FROM mcp_servers ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []MCPServerConfig
	for rows.Next() {
		var srv MCPServerConfig
		if err := rows.Scan(&srv.ID, &srv.Name, &srv.Transport, &srv.URL, &srv.Command, &srv.Args, &srv.Active); err != nil {
			return nil, err
		}
		servers = append(servers, srv)
	}
	return servers, rows.Err()
}

// SaveMCPServerPayload is the frontend payload to create or update a MCP server.
type SaveMCPServerPayload struct {
	ID        int64  `json:"id"` // 0 = create, >0 = update
	Name      string `json:"name"`
	Transport string `json:"transport"` // "http" | "stdio"
	URL       string `json:"url"`
	Command   string `json:"command"`
	Args      string `json:"args"` // JSON array string e.g. ["mcp", "stdio"]
	Active    bool   `json:"active"`
}

// SaveMCPServer creates or updates a MCP server configuration.
func (s *Service) SaveMCPServer(ctx context.Context, p SaveMCPServerPayload) (map[string]any, error) {
	if p.Name == "" {
		return nil, fmt.Errorf("server name is required")
	}
	if p.Transport != "http" && p.Transport != "stdio" && p.Transport != "sse" {
		return nil, fmt.Errorf("transport must be 'http', 'sse', or 'stdio'")
	}
	if p.Transport == "stdio" && p.Command == "" {
		return nil, fmt.Errorf("command is required for stdio transport")
	}
	if (p.Transport == "http" || p.Transport == "sse") && p.URL == "" {
		return nil, fmt.Errorf("url is required for http/sse transport")
	}
	if p.Args == "" {
		p.Args = "[]"
	}

	active := 0
	if p.Active {
		active = 1
	}

	if p.ID == 0 {
		res, err := s.db.ExecContext(ctx, `
			INSERT INTO mcp_servers (name, transport, url, command, args, active, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, p.Name, p.Transport, p.URL, p.Command, p.Args, active, time.Now().Format(time.RFC3339))
		if err != nil {
			return nil, fmt.Errorf("create mcp server: %w", err)
		}
		id, _ := res.LastInsertId()
		return map[string]any{"ok": true, "id": id}, nil
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE mcp_servers SET name=?, transport=?, url=?, command=?, args=?, active=?, updated_at=?
		WHERE id=?
	`, p.Name, p.Transport, p.URL, p.Command, p.Args, active, time.Now().Format(time.RFC3339), p.ID)
	if err != nil {
		return nil, fmt.Errorf("update mcp server: %w", err)
	}
	return map[string]any{"ok": true, "id": p.ID}, nil
}

// DeleteMCPServer removes a MCP server by id.
func (s *Service) DeleteMCPServer(ctx context.Context, id int64) (map[string]any, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM mcp_servers WHERE id=?`, id)
	if err != nil {
		return nil, fmt.Errorf("delete mcp server: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, sql.ErrNoRows
	}
	return map[string]any{"ok": true}, nil
}

// ToggleMCPServerPayload sets a server's active state.
type ToggleMCPServerPayload struct {
	ID     int64 `json:"id"`
	Active bool  `json:"active"`
}

// ToggleMCPServer enables or disables a MCP server.
func (s *Service) ToggleMCPServer(ctx context.Context, p ToggleMCPServerPayload) (map[string]any, error) {
	active := 0
	if p.Active {
		active = 1
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE mcp_servers SET active=?, updated_at=? WHERE id=?`,
		active, time.Now().Format(time.RFC3339), p.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("toggle mcp server: %w", err)
	}
	return map[string]any{"ok": true}, nil
}

// PingMCPServer opens a transport to the given MCP server, lists its tools and returns them.
// Used by the settings UI "Test connection" button to verify connectivity and discover tools.
func (s *Service) PingMCPServer(ctx context.Context, id int64) (map[string]any, error) {
	var srv MCPServerConfig
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, transport, url, command, args, active FROM mcp_servers WHERE id=?`, id,
	).Scan(&srv.ID, &srv.Name, &srv.Transport, &srv.URL, &srv.Command, &srv.Args, &srv.Active)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	t, openErr := s.openMCPTransport(srv)
	if openErr != nil {
		return map[string]any{"ok": false, "error": openErr.Error(), "tools": []string{}}, nil
	}
	defer t.Close() //nolint:errcheck

	tCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tools, listErr := t.ListTools(tCtx)
	if listErr != nil {
		return map[string]any{"ok": false, "error": listErr.Error(), "tools": []string{}}, nil
	}

	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	slog.Info("mcp: ping successful", "server", srv.Name, "tools", len(names))
	return map[string]any{"ok": true, "tools": names}, nil
}
