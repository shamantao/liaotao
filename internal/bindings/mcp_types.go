/*
  mcp_types.go -- MCP (Model Context Protocol) shared types.
  Responsibilities: JSON-RPC 2.0 wire types, tool call/result structures,
  MCP server configuration record stored in SQLite.
  Used by built-in dispatcher, HTTP/SSE transport, and stdio transport.
*/

package bindings

import "encoding/json"

// ─── JSON-RPC 2.0 ──────────────────────────────────────────────────────────

// jsonRPCRequest is a JSON-RPC 2.0 request (or notification when ID is nil).
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError is the error object inside a JSON-RPC 2.0 error response.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *jsonRPCError) Error() string { return e.Message }

// ─── MCP tool definition (returned by tools/list) ─────────────────────────

// MCPTool describes a tool exposed by an MCP server.
type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// ─── OpenAI function/tool call format (in model responses) ────────────────

// ToolCallFunction holds the function name and JSON-encoded arguments.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string, parsed on dispatch
}

// ToolCall is one tool call emitted by the model (OpenAI format).
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // always "function"
	Function ToolCallFunction `json:"function"`
}

// ToolResult is the result to re-inject into the conversation after execution.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Role       string `json:"role"` // always "tool"
	Content    string `json:"content"`
}

// ─── MCP server configuration (stored in SQLite mcp_servers table) ────────

// MCPServerConfig is one configured MCP server.
type MCPServerConfig struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Transport string `json:"transport"` // "http" | "stdio"
	URL       string `json:"url"`       // for http transport
	Command   string `json:"command"`   // for stdio transport
	Args      string `json:"args"`      // JSON array string, for stdio transport
	Active    bool   `json:"active"`
}

// ─── MCP dispatch result ───────────────────────────────────────────────────

// mcpCallResult is the internal result of dispatching one tool call.
type mcpCallResult struct {
	ToolCallID string
	Content    string
	Err        error
}
