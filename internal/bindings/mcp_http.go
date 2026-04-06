/*
  mcp_http.go -- Streamable HTTP MCP transport (MCP-01, MCP-03, MCP-04).
  Responsibilities: JSON-RPC 2.0 over MCP Streamable HTTP (MCP 1.0 standard).
  Legacy FastMCP SSE servers are handled by mcp_sse.go instead.
  Implements the mcpTransport interface.
*/

package bindings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
)

// httpTransport implements mcpTransport over MCP Streamable HTTP (MCP 1.0 standard).
// The URL must point to the MCP endpoint, e.g. "http://host:8201/mcp".
type httpTransport struct {
	url    string
	client *http.Client
	nextID atomic.Int64
}

// newHTTPTransport creates a Streamable HTTP MCP transport.
func newHTTPTransport(url string, client *http.Client) *httpTransport {
	return &httpTransport{url: strings.TrimRight(url, "/"), client: client}
}

// Close is a no-op for HTTP (stateless).
func (t *httpTransport) Close() error { return nil }

// ListTools calls tools/list and returns the server's tool definitions (MCP-04).
func (t *httpTransport) ListTools(ctx context.Context) ([]MCPTool, error) {
	result, err := t.call(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Tools []MCPTool `json:"tools"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("tools/list parse error: %w", err)
	}
	return resp.Tools, nil
}

// CallTool invokes a tool on the MCP server and returns its text content.
func (t *httpTransport) CallTool(ctx context.Context, toolName string, argsJSON string) (string, error) {
	var args any
	if argsJSON != "" && argsJSON != "null" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid tool arguments: %w", err)
		}
	} else {
		args = map[string]any{}
	}
	params := map[string]any{"name": toolName, "arguments": args}
	result, err := t.call(ctx, "tools/call", params)
	if err != nil {
		return "", err
	}
	return extractToolContent(result)
}

// call sends a JSON-RPC 2.0 request using MCP Streamable HTTP and returns the result.
func (t *httpTransport) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := t.nextID.Add(1)
	req := jsonRPCRequest{JSONRPC: "2.0", ID: id, Method: method}
	if params != nil {
		encoded, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		req.Params = json.RawMessage(encoded)
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		return nil, fmt.Errorf("streamable HTTP not supported (status %d) — try SSE transport instead", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	return extractJSONRPCResult(raw)
}

// extractJSONRPCResult unmarshals a JSON-RPC response and returns the result field.
func extractJSONRPCResult(raw []byte) (json.RawMessage, error) {
	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(raw, &rpcResp); err != nil {
		return nil, fmt.Errorf("JSON-RPC parse error: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}
	return rpcResp.Result, nil
}

// extractToolContent parses the MCP tools/call result and returns printable text.
// MCP spec: result = {"content": [{"type":"text","text":"..."}]} or {"content":"..."}
func extractToolContent(raw json.RawMessage) (string, error) {
	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(raw, &resp); err == nil && len(resp.Content) > 0 {
		var parts []string
		for _, block := range resp.Content {
			if block.Type == "text" || block.Type == "" {
				parts = append(parts, block.Text)
			}
		}
		result := strings.Join(parts, "\n")
		if resp.IsError {
			return fmt.Sprintf("error: %s", result), nil
		}
		return result, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	return string(raw), nil
}

