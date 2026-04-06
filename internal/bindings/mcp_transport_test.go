/*
  mcp_transport_test.go -- Regression tests for MCP transports (stdio, HTTP, SSE).
  Each test uses in-memory mock servers to avoid external dependencies.
  Run with: go test ./internal/bindings -run TestMCP -v
*/

package bindings

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ── Mock stdio MCP server helpers ─────────────────────────────────────────

// mockMCPServer simulates a minimal MCP server over in-memory pipes.
// It handles initialize, notifications/initialized, tools/list, and tools/call.
type mockMCPServer struct {
	tools    []MCPTool
	callResp string // fixed response for all tool calls
	callErr  bool   // if true, return isError:true
}

// handleRequests reads JSON-RPC requests from r and writes responses to w.
// Runs in a goroutine — call go s.handleRequests(r, w).
func (s *mockMCPServer) handleRequests(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req map[string]any
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		method, _ := req["method"].(string)
		id := req["id"] // may be nil for notifications

		// Notifications have no ID and expect no response.
		if id == nil {
			continue
		}

		var result any
		switch method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{}}
		case "tools/list":
			result = map[string]any{"tools": s.tools}
		case "tools/call":
			params, _ := req["params"].(map[string]any)
			toolName, _ := params["name"].(string)
			if s.callErr {
				result = map[string]any{
					"content": []map[string]any{{"type": "text", "text": "tool error"}},
					"isError": true,
				}
			} else {
				text := s.callResp
				if text == "" {
					text = fmt.Sprintf("result from %s", toolName)
				}
				result = map[string]any{
					"content": []map[string]any{{"type": "text", "text": text}},
				}
			}
		default:
			resp, _ := json.Marshal(map[string]any{
				"jsonrpc": "2.0", "id": id,
				"error": map[string]any{"code": -32601, "message": "method not found"},
			})
			fmt.Fprintf(w, "%s\n", resp) //nolint:errcheck
			continue
		}

		resp, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
		fmt.Fprintf(w, "%s\n", resp) //nolint:errcheck
	}
}

// newPipedStdioTransport creates a stdioTransport connected to a mock server via pipes.
func newPipedStdioTransport(srv *mockMCPServer, initTimeout time.Duration) (*stdioTransport, func()) {
	// serverR ← clientW : transport writes requests here
	serverR, clientW := io.Pipe()
	// clientR ← serverW : transport reads responses from here
	clientR, serverW := io.Pipe()

	go srv.handleRequests(serverR, serverW)

	tr := newStdioTransportFromPipes(clientR, clientW, initTimeout)
	cleanup := func() {
		tr.Close()
		serverW.Close()
		serverR.Close()
	}
	return tr, cleanup
}

// ── Stdio transport tests ────────────────────────────────────────────────

func TestStdioTransport_ListTools(t *testing.T) {
	srv := &mockMCPServer{
		tools: []MCPTool{
			{Name: "aitao_search", Description: "Search documents"},
			{Name: "calculator", Description: "Compute math"},
		},
	}
	tr, cleanup := newPipedStdioTransport(srv, 2*time.Second)
	defer cleanup()

	tools, err := tr.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "aitao_search" || tools[1].Name != "calculator" {
		t.Errorf("unexpected tool names: %v", tools)
	}
}

func TestStdioTransport_CallTool(t *testing.T) {
	srv := &mockMCPServer{
		tools:    []MCPTool{{Name: "echo"}},
		callResp: "hello world",
	}
	tr, cleanup := newPipedStdioTransport(srv, 2*time.Second)
	defer cleanup()

	result, err := tr.CallTool(context.Background(), "echo", `{"msg":"hello"}`)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}
}

func TestStdioTransport_InitializeOnce(t *testing.T) {
	// Count how many "initialize" calls the server receives.
	initCount := 0
	pr, pw := io.Pipe()
	cr, cw := io.Pipe()

	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			line := scanner.Text()
			var req map[string]any
			json.Unmarshal([]byte(line), &req) //nolint:errcheck
			method, _ := req["method"].(string)
			id := req["id"]
			if id == nil {
				continue
			}
			if method == "initialize" {
				initCount++
			}
			var result any
			switch method {
			case "initialize":
				result = map[string]any{"protocolVersion": "2024-11-05"}
			case "tools/list":
				result = map[string]any{"tools": []MCPTool{{Name: "t1"}}}
			}
			resp, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
			fmt.Fprintf(cw, "%s\n", resp) //nolint:errcheck
		}
	}()

	tr := newStdioTransportFromPipes(cr, pw, 2*time.Second)
	defer func() { tr.Close(); cw.Close() }()

	// Call ListTools twice — initialize must be sent only once.
	for i := 0; i < 2; i++ {
		if _, err := tr.ListTools(context.Background()); err != nil {
			t.Fatalf("ListTools %d failed: %v", i+1, err)
		}
	}
	if initCount != 1 {
		t.Errorf("initialize sent %d times, expected exactly 1", initCount)
	}
}

func TestStdioTransport_ToolCallError(t *testing.T) {
	srv := &mockMCPServer{
		tools:   []MCPTool{{Name: "failing_tool"}},
		callErr: true,
	}
	tr, cleanup := newPipedStdioTransport(srv, 2*time.Second)
	defer cleanup()

	result, err := tr.CallTool(context.Background(), "failing_tool", "{}")
	if err != nil {
		t.Fatalf("unexpected transport error (tool errors should be returned as text): %v", err)
	}
	// isError:true tools return "error: <text>"
	if !strings.HasPrefix(result, "error:") {
		t.Errorf("expected result starting with 'error:', got %q", result)
	}
}

func TestStdioTransport_ContextTimeout(t *testing.T) {
	// Server that never responds — verifies context cancellation propagates.
	// serverR ← clientW : transport writes here, server drains but ignores.
	// clientR ← serverW : server never writes here, so transport gets no response.
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	// Drain writes from transport to avoid blocking the transport write path.
	go func() { io.Copy(io.Discard, serverR) }() //nolint:errcheck

	tr := newStdioTransportFromPipes(clientR, clientW, 100*time.Millisecond)
	defer func() {
		tr.Close()    // closes clientW → serverR.Read() returns error → drain goroutine exits
		serverW.Close() // triggers clientR EOF → readLoop exits
		serverR.Close()
	}()

	_, err := tr.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// ── HTTP (Streamable HTTP) transport tests ────────────────────────────────

// mockHTTPMCPServer creates an httptest.Server that handles MCP Streamable HTTP.
func mockHTTPMCPServer(t *testing.T, tools []MCPTool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		method, _ := req["method"].(string)
		id := req["id"]

		var result any
		switch method {
		case "tools/list":
			result = map[string]any{"tools": tools}
		case "tools/call":
			params, _ := req["params"].(map[string]any)
			name, _ := params["name"].(string)
			result = map[string]any{
				"content": []map[string]any{{"type": "text", "text": "result:" + name}},
			}
		default:
			result = map[string]any{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result}) //nolint:errcheck
	}))
}

func TestHTTPTransport_ListTools(t *testing.T) {
	wantTools := []MCPTool{{Name: "http_tool", Description: "via HTTP"}}
	ts := mockHTTPMCPServer(t, wantTools)
	defer ts.Close()

	tr := newHTTPTransport(ts.URL, ts.Client())
	defer tr.Close()

	tools, err := tr.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "http_tool" {
		t.Errorf("unexpected tools: %v", tools)
	}
}

func TestHTTPTransport_CallTool(t *testing.T) {
	ts := mockHTTPMCPServer(t, []MCPTool{{Name: "greet"}})
	defer ts.Close()

	tr := newHTTPTransport(ts.URL, ts.Client())
	result, err := tr.CallTool(context.Background(), "greet", `{}`)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result != "result:greet" {
		t.Errorf("expected 'result:greet', got %q", result)
	}
}

func TestHTTPTransport_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	tr := newHTTPTransport(ts.URL, ts.Client())
	_, err := tr.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
}

func TestHTTPTransport_NotFoundReturnsHelpfulError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	tr := newHTTPTransport(ts.URL, ts.Client())
	_, err := tr.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
	if !strings.Contains(err.Error(), "SSE transport instead") {
		t.Errorf("expected hint about SSE transport in error, got: %v", err)
	}
}

// ── SSE transport tests ────────────────────────────────────────────────────

// mockSSEServer creates an httptest.Server implementing the FastMCP two-channel SSE protocol:
//   GET  /sse          → persistent SSE stream, sends "endpoint" event then streams responses
//   POST /messages/?session_id=XXX → receives JSON-RPC requests, routes response to SSE stream
//
// The SSE handler exits cleanly when the client disconnects (r.Context().Done()).
func mockSSEServer(t *testing.T, tools []MCPTool) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	sessions := make(map[string]chan map[string]any)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/sse" && r.Method == http.MethodGet:
			const sessionID = "test-session"
			ch := make(chan map[string]any, 8)
			mu.Lock()
			sessions[sessionID] = ch
			mu.Unlock()

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			flusher := w.(http.Flusher)

			// Send endpoint event so the client knows where to POST requests.
			fmt.Fprintf(w, "event: endpoint\ndata: /messages/?session_id=%s\n\n", sessionID) //nolint:errcheck
			flusher.Flush()

			// Forward responses from the POST handler until the client disconnects.
			for {
				select {
				case <-r.Context().Done(): // client closed the connection
					mu.Lock()
					delete(sessions, sessionID)
					mu.Unlock()
					return
				case resp, ok := <-ch:
					if !ok {
						return
					}
					data, _ := json.Marshal(resp)
					fmt.Fprintf(w, "event: message\ndata: %s\n\n", data) //nolint:errcheck
					flusher.Flush()
				}
			}

		case strings.HasPrefix(r.URL.Path, "/messages/") && r.Method == http.MethodPost:
			sessionID := r.URL.Query().Get("session_id")
			mu.Lock()
			ch := sessions[sessionID]
			mu.Unlock()
			if ch == nil {
				http.Error(w, "unknown session", http.StatusBadRequest)
				return
			}

			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			id := req["id"]
			if id == nil { // notification, no response needed
				w.WriteHeader(http.StatusAccepted)
				return
			}

			method, _ := req["method"].(string)
			var result any
			switch method {
			case "initialize":
				result = map[string]any{"protocolVersion": "2024-11-05"}
			case "tools/list":
				result = map[string]any{"tools": tools}
			case "tools/call":
				params, _ := req["params"].(map[string]any)
				name, _ := params["name"].(string)
				result = map[string]any{
					"content": []map[string]any{{"type": "text", "text": "sse:" + name}},
				}
			default:
				result = map[string]any{}
			}
			ch <- map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
			w.WriteHeader(http.StatusAccepted)

		default:
			http.NotFound(w, r)
		}
	}))
	return ts
}

func TestSSETransport_ListTools(t *testing.T) {
	wantTools := []MCPTool{{Name: "aitao_stats", Description: "System stats"}}
	ts := mockSSEServer(t, wantTools)
	defer ts.Close()

	tr := newSSETransport(ts.URL, ts.Client())
	defer tr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tools, err := tr.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "aitao_stats" {
		t.Errorf("unexpected tools: %v", tools)
	}
}

func TestSSETransport_CallTool(t *testing.T) {
	ts := mockSSEServer(t, []MCPTool{{Name: "aitao_search"}})
	defer ts.Close()

	tr := newSSETransport(ts.URL, ts.Client())
	defer tr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tr.CallTool(ctx, "aitao_search", `{"query":"test"}`)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result != "sse:aitao_search" {
		t.Errorf("expected 'sse:aitao_search', got %q", result)
	}
}

func TestSSETransport_UnreachableServer(t *testing.T) {
	tr := newSSETransport("http://127.0.0.1:19999", http.DefaultClient)
	defer tr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := tr.ListTools(ctx)
	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
}

// ── extractToolContent regression tests ──────────────────────────────────

func TestExtractToolContent_ArrayFormat(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": "first"},
			{"type": "text", "text": "second"},
		},
	})
	got, err := extractToolContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "first\nsecond" {
		t.Errorf("expected 'first\\nsecond', got %q", got)
	}
}

func TestExtractToolContent_IsError(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"content": []map[string]any{{"type": "text", "text": "something went wrong"}},
		"isError": true,
	})
	got, err := extractToolContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, "error:") {
		t.Errorf("expected 'error:' prefix, got %q", got)
	}
}

func TestExtractToolContent_StringFallback(t *testing.T) {
	raw, _ := json.Marshal("plain string result")
	got, err := extractToolContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "plain string result" {
		t.Errorf("expected 'plain string result', got %q", got)
	}
}
