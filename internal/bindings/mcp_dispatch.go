/*
  mcp_dispatch.go -- MCP tool call dispatcher (MCP-05, MCP-06).
  Responsibilities: receive ToolCall list from model stream, dispatch each call
  to built-in handler or external MCP server (by tool name prefix or server registry),
  return ToolResult list to re-inject into conversation.
  External server support is wired in by mcp_http.go and mcp_stdio.go.
*/

package bindings

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// mcpTransport is the interface satisfied by both HTTP/SSE and stdio transports.
type mcpTransport interface {
	// CallTool invokes a tool on the remote MCP server and returns its text content.
	CallTool(ctx context.Context, toolName string, argsJSON string) (string, error)
	// ListTools returns the tools advertised by the server.
	ListTools(ctx context.Context) ([]MCPTool, error)
	// Close releases resources held by the transport.
	Close() error
}

// DispatchToolCalls executes a slice of ToolCalls and returns a ToolResult for each.
// Built-in tools are resolved first; unknown tools are forwarded to active MCP servers.
// Errors per tool are returned as plain-text results (never fatal) so the model can
// observe the failure and respond gracefully.
func (s *Service) DispatchToolCalls(ctx context.Context, calls []ToolCall) []ToolResult {
	results := make([]ToolResult, 0, len(calls))
	for _, call := range calls {
		content := s.executeSingleTool(ctx, call.Function.Name, call.Function.Arguments)
		results = append(results, ToolResult{
			ToolCallID: call.ID,
			Role:       "tool",
			Content:    content,
		})
	}
	return results
}

// executeSingleTool runs one tool by name and returns its text result.
// Built-ins are tried first, then active MCP servers in DB order.
func (s *Service) executeSingleTool(ctx context.Context, name, argsJSON string) string {
	// 1. Built-in tools — no external dependency.
	if result, handled := dispatchBuiltin(name, argsJSON); handled {
		slog.Debug("mcp: built-in tool dispatched", "tool", name)
		return result
	}

	// 2. External MCP servers — iterate active servers from DB.
	servers, err := s.listActiveMCPServers(ctx)
	if err != nil {
		slog.Error("mcp: failed to list active servers", "err", err)
		return fmt.Sprintf("error: could not list MCP servers — %s", err.Error())
	}

	for _, srv := range servers {
		transport, openErr := s.openMCPTransport(srv)
		if openErr != nil {
			slog.Warn("mcp: could not open transport", "server", srv.Name, "err", openErr)
			continue
		}
		defer transport.Close() //nolint:errcheck

		// Check whether this server exposes the requested tool.
		tools, listErr := transport.ListTools(ctx)
		if listErr != nil {
			slog.Warn("mcp: tools/list failed", "server", srv.Name, "err", listErr)
			continue
		}
		if !toolListContains(tools, name) {
			continue
		}

		result, callErr := transport.CallTool(ctx, name, argsJSON)
		if callErr != nil {
			slog.Warn("mcp: tool call failed", "server", srv.Name, "tool", name, "err", callErr)
			return fmt.Sprintf("error: %s", callErr.Error())
		}
		slog.Debug("mcp: external tool dispatched", "server", srv.Name, "tool", name)
		return result
	}

	return fmt.Sprintf("error: unknown tool '%s' — no server handles it", name)
}

// toolListContains returns true when tools contains a tool with the given name.
func toolListContains(tools []MCPTool, name string) bool {
	for _, t := range tools {
		if t.Name == name {
			return true
		}
	}
	return false
}

// AllAvailableTools returns the merged list of built-in + external tool definitions.
// Used to populate the `tools` field when sending a request to the model.
func (s *Service) AllAvailableTools(ctx context.Context) []MCPTool {
	tools := builtinToolDefs()

	servers, err := s.listActiveMCPServers(ctx)
	if err != nil {
		return tools
	}

	for _, srv := range servers {
		transport, err := s.openMCPTransport(srv)
		if err != nil {
			slog.Warn("mcp: skipping server for tool discovery", "server", srv.Name, "err", err)
			continue
		}
		defer transport.Close() //nolint:errcheck

		serverTools, err := transport.ListTools(ctx)
		if err != nil {
			slog.Warn("mcp: tools/list failed during discovery", "server", srv.Name, "err", err)
			continue
		}
		tools = append(tools, serverTools...)
	}
	return tools
}

// openMCPTransport creates the appropriate transport for a server config.
// Returns an error when the transport type is unsupported or misconfigured.
func (s *Service) openMCPTransport(srv MCPServerConfig) (mcpTransport, error) {
	switch srv.Transport {
	case "http", "sse":
		return newHTTPTransport(srv.URL, s.client), nil
	case "stdio":
		var args []string
		if srv.Args != "" {
			if err := json.Unmarshal([]byte(srv.Args), &args); err != nil {
				return nil, fmt.Errorf("invalid args JSON for stdio server %q: %w", srv.Name, err)
			}
		}
		return newStdioTransport(srv.Command, args), nil
	default:
		return nil, fmt.Errorf("unsupported transport type %q", srv.Transport)
	}
}
