/*
  mcp_sse.go -- FastMCP legacy SSE transport (MCP-02).
  Responsibilities: connect to a FastMCP 1.x SSE server using the two-channel protocol:
    - GET  {base}/sse  → persistent SSE stream (receive responses + endpoint event)
    - POST {msgURL}    → send JSON-RPC requests (session URL from endpoint event)
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
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// sseTransport implements the FastMCP two-channel SSE protocol.
// It maintains a persistent GET /sse connection and POSTs requests
// to the session messages endpoint received in the SSE "endpoint" event.
type sseTransport struct {
	baseURL string
	client  *http.Client
	nextID  atomic.Int64

	mu      sync.Mutex
	msgURL  string        // absolute POST URL from the SSE "endpoint" event
	sseConn io.ReadCloser // keep-alive SSE response body

	pendingMu sync.Mutex
	pending   map[int64]chan jsonRPCResponse

	connectOnce sync.Once
	connectErr  error

	initOnce sync.Once
	initErr  error
}

func newSSETransport(baseURL string, client *http.Client) *sseTransport {
	return &sseTransport{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
		pending: make(map[int64]chan jsonRPCResponse),
	}
}

// connect opens the GET /sse stream and waits for the "endpoint" event.
// Uses sync.Once so the connection is established at most once per transport instance.
func (t *sseTransport) connect() error {
	t.connectOnce.Do(func() {
		// Use context.Background() so the SSE stream stays alive beyond any single call timeout.
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, t.baseURL+"/sse", nil)
		if err != nil {
			t.connectErr = fmt.Errorf("SSE: build request: %w", err)
			return
		}
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")

		resp, err := t.client.Do(req)
		if err != nil {
			t.connectErr = fmt.Errorf("SSE: connect to %s/sse: %w", t.baseURL, err)
			return
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			t.connectErr = fmt.Errorf("SSE: connect HTTP %d", resp.StatusCode)
			return
		}

		// Read SSE lines until we receive the "endpoint" event with the session URL.
		scanner := bufio.NewScanner(resp.Body)
		eventType := ""
		msgPath := ""
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event:") {
				eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			} else if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if eventType == "endpoint" {
					msgPath = data
					break
				}
			} else if line == "" {
				eventType = ""
			}
		}

		if msgPath == "" {
			resp.Body.Close()
			t.connectErr = fmt.Errorf("SSE: no endpoint event received from %s/sse", t.baseURL)
			return
		}

		// Build absolute messages URL. FastMCP usually returns a relative path like
		// "/messages/?session_id=XXXX", so we prepend the base URL.
		if strings.HasPrefix(msgPath, "http") {
			t.msgURL = msgPath
		} else {
			t.msgURL = t.baseURL + msgPath
		}
		t.sseConn = resp.Body

		go t.readLoop(scanner, resp.Body)
		slog.Debug("mcp sse: connected", "msgURL", t.msgURL)
	})
	return t.connectErr
}

// readLoop reads SSE "message" events and dispatches JSON-RPC responses to waiting callers.
func (t *sseTransport) readLoop(scanner *bufio.Scanner, body io.ReadCloser) {
	eventType := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") && eventType == "message" {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			var resp jsonRPCResponse
			if err := json.Unmarshal([]byte(data), &resp); err != nil {
				slog.Warn("mcp sse: parse error", "data", data, "err", err)
				eventType = ""
				continue
			}
			id := int64(0)
			switch v := resp.ID.(type) {
			case float64:
				id = int64(v)
			case json.Number:
				id, _ = v.Int64()
			}
			t.pendingMu.Lock()
			ch, ok := t.pending[id]
			if ok {
				delete(t.pending, id)
			}
			t.pendingMu.Unlock()
			if ok {
				ch <- resp
			}
			eventType = ""
		} else if line == "" {
			eventType = ""
		}
	}
	// Stream closed — unblock all pending callers.
	t.pendingMu.Lock()
	for id, ch := range t.pending {
		ch <- jsonRPCResponse{Error: &jsonRPCError{Code: -32000, Message: "SSE stream closed"}}
		delete(t.pending, id)
	}
	t.pendingMu.Unlock()
}

// ensureInitialized sends the MCP initialize handshake exactly once.
// Required by MCP spec before tools/list or tools/call.
func (t *sseTransport) ensureInitialized() {
	t.initOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		params := map[string]any{
			"protocolVersion": "2024-11-05",
			"clientInfo":      map[string]any{"name": "liaotao", "version": "0.1"},
			"capabilities":    map[string]any{},
		}
		if _, err := t.sendRaw(ctx, "initialize", params); err != nil {
			t.initErr = fmt.Errorf("mcp sse initialize: %w", err)
			return
		}

		// notifications/initialized has no ID and expects no response — fire and forget.
		t.mu.Lock()
		msgURL := t.msgURL
		t.mu.Unlock()
		notif, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
		req, _ := http.NewRequest(http.MethodPost, msgURL, bytes.NewReader(notif))
		req.Header.Set("Content-Type", "application/json")
		if resp, err := t.client.Do(req); err == nil {
			resp.Body.Close()
		}
		slog.Debug("mcp sse: initialized")
	})
}

// sendRaw POSTs a JSON-RPC request to the session messages URL and waits for
// the response to arrive on the persistent SSE stream.
func (t *sseTransport) sendRaw(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if err := t.connect(); err != nil {
		return nil, err
	}

	id := t.nextID.Add(1)
	rpcReq := jsonRPCRequest{JSONRPC: "2.0", ID: id, Method: method}
	if params != nil {
		encoded, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		rpcReq.Params = json.RawMessage(encoded)
	}
	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	t.mu.Lock()
	msgURL := t.msgURL
	t.mu.Unlock()

	ch := make(chan jsonRPCResponse, 1)
	t.pendingMu.Lock()
	t.pending[id] = ch
	t.pendingMu.Unlock()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, msgURL, bytes.NewReader(body))
	if err != nil {
		t.pendingMu.Lock()
		delete(t.pending, id)
		t.pendingMu.Unlock()
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	postResp, err := t.client.Do(httpReq)
	if err != nil {
		t.pendingMu.Lock()
		delete(t.pending, id)
		t.pendingMu.Unlock()
		return nil, fmt.Errorf("SSE post: %w", err)
	}
	postResp.Body.Close()
	if postResp.StatusCode < 200 || postResp.StatusCode >= 300 {
		t.pendingMu.Lock()
		delete(t.pending, id)
		t.pendingMu.Unlock()
		return nil, fmt.Errorf("SSE post HTTP %d", postResp.StatusCode)
	}

	select {
	case <-ctx.Done():
		t.pendingMu.Lock()
		delete(t.pending, id)
		t.pendingMu.Unlock()
		return nil, ctx.Err()
	case rpcResp := <-ch:
		if rpcResp.Error != nil {
			return nil, rpcResp.Error
		}
		return rpcResp.Result, nil
	}
}

// call ensures the MCP handshake is done, then sends the request.
func (t *sseTransport) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	t.ensureInitialized()
	if t.initErr != nil {
		return nil, t.initErr
	}
	return t.sendRaw(ctx, method, params)
}

// ListTools returns the tools exposed by the FastMCP SSE server.
func (t *sseTransport) ListTools(ctx context.Context) ([]MCPTool, error) {
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

// CallTool invokes a tool on the FastMCP SSE server.
func (t *sseTransport) CallTool(ctx context.Context, toolName string, argsJSON string) (string, error) {
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

// Close terminates the persistent SSE connection.
func (t *sseTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.sseConn != nil {
		err := t.sseConn.Close()
		t.sseConn = nil
		return err
	}
	return nil
}
