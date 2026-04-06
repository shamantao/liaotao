/*
  mcp_http.go -- HTTP/SSE MCP transport (MCP-01, MCP-03, MCP-04).
  Responsibilities: JSON-RPC 2.0 over HTTP (MCP Streamable HTTP, MCP 1.0 standard)
  and SSE legacy transport. Connects to aitao on :8201 or any configured URL.
  Implements the mcpTransport interface.
*/

package bindings

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
)

// httpTransport implements mcpTransport over HTTP (Streamable HTTP + SSE fallback).
type httpTransport struct {
	url    string
	client *http.Client
	nextID atomic.Int64
}

// newHTTPTransport creates an HTTP MCP transport pointed at the given base URL.
// The URL should be the MCP endpoint, e.g. "http://localhost:8201/mcp".
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

	// MCP spec: result is {"tools": [...]}
	var resp struct {
		Tools []MCPTool `json:"tools"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("tools/list parse error: %w", err)
	}
	return resp.Tools, nil
}

// CallTool invokes a tool on the MCP server and returns its text content.
// Handles both single-content and multi-content (array) responses.
func (t *httpTransport) CallTool(ctx context.Context, toolName string, argsJSON string) (string, error) {
	var args any
	if argsJSON != "" && argsJSON != "null" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid tool arguments: %w", err)
		}
	} else {
		args = map[string]any{}
	}

	params := map[string]any{
		"name":      toolName,
		"arguments": args,
	}
	result, err := t.call(ctx, "tools/call", params)
	if err != nil {
		return "", err
	}

	return extractToolContent(result)
}

// call sends a JSON-RPC 2.0 request and returns the raw result bytes.
// Tries Streamable HTTP first; falls back to SSE transport on 4xx.
func (t *httpTransport) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := t.nextID.Add(1)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
	}
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

	// Try Streamable HTTP (MCP 1.0 standard) first.
	result, err := t.callStreamableHTTP(ctx, body)
	if err == nil {
		return result, nil
	}

	// Fall back to SSE transport (legacy FastMCP / older servers).
	return t.callSSE(ctx, body)
}

// callStreamableHTTP sends a JSON-RPC request to the MCP HTTP endpoint
// and reads the response as a single JSON object (non-streaming response).
func (t *httpTransport) callStreamableHTTP(ctx context.Context, body []byte) (json.RawMessage, error) {
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
		return nil, fmt.Errorf("streamable HTTP not supported (status %d)", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	// If server responds with SSE content-type, delegate to SSE parser.
	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		return parseSSEResponse(resp.Body)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	return extractJSONRPCResult(raw)
}

// callSSE sends a JSON-RPC request to the SSE endpoint and reads the streamed response.
// Used for legacy FastMCP servers (aitao ≤ MCP 0.9).
func (t *httpTransport) callSSE(ctx context.Context, body []byte) (json.RawMessage, error) {
	sseURL := t.url + "/sse"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, sseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("SSE transport: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("SSE HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return parseSSEResponse(resp.Body)
}

// parseSSEResponse reads an SSE stream and returns the first JSON-RPC result found.
func parseSSEResponse(body io.Reader) (json.RawMessage, error) {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		return extractJSONRPCResult([]byte(data))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("SSE read error: %w", err)
	}
	return nil, fmt.Errorf("SSE stream ended without a result")
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
	// Try array-of-content-blocks format (standard MCP).
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

	// Fallback: treat entire result as a string.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	return string(raw), nil
}
