/*
  mcp_stdio.go -- stdio MCP transport (MCP-02).
  Responsibilities: spawn an external MCP server process (e.g. "aitao mcp stdio"),
  communicate via stdin/stdout using JSON-RPC 2.0 newline-delimited messages.
  Implements the mcpTransport interface.
*/

package bindings

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// stdioTransport implements mcpTransport over a child process stdin/stdout.
type stdioTransport struct {
	command     string
	args        []string
	initTimeout time.Duration // timeout for the MCP initialize handshake (default 30s)

	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	started bool // true when process (or test pipes) are ready
	nextID  atomic.Int64

	// pending maps request ID → response channel.
	pendingMu sync.Mutex
	pending   map[int64]chan jsonRPCResponse

	// MCP protocol requires an initialize handshake before any tool call.
	initOnce sync.Once
	initErr  error
}

// newStdioTransport creates a stdio transport that will spawn the given command.
// The process is not started until the first call to ListTools or CallTool.
func newStdioTransport(command string, args []string) *stdioTransport {
	return &stdioTransport{
		command:     command,
		args:        args,
		pending:     make(map[int64]chan jsonRPCResponse),
		initTimeout: 30 * time.Second, // subprocess may need time to load ML models
	}
}

// newStdioTransportFromPipes creates a stdioTransport backed by pre-opened pipes.
// Used in tests to avoid spawning a subprocess. The readLoop is started immediately.
func newStdioTransportFromPipes(r io.Reader, w io.WriteCloser, initTimeout time.Duration) *stdioTransport {
	t := &stdioTransport{
		pending:     make(map[int64]chan jsonRPCResponse),
		initTimeout: initTimeout,
		stdin:       w,
		scanner:     bufio.NewScanner(r),
		started:     true,
	}
	go t.readLoop()
	return t
}

// ensureStarted lazily spawns the child process on first use.
func (t *stdioTransport) ensureStarted() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.started {
		return nil
	}
	cmd := exec.Command(t.command, t.args...) //nolint:gosec — command comes from user config, validated on save
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdio transport: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdio transport: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("stdio transport: start %q: %w", t.command, err)
	}

	t.cmd = cmd
	t.stdin = stdin
	t.scanner = bufio.NewScanner(stdout)
	t.started = true

	// Start reader goroutine: routes responses to pending channels.
	go t.readLoop()

	slog.Debug("mcp stdio: process started", "command", t.command)
	return nil
}

// readLoop reads newline-delimited JSON responses from the child process stdout
// and dispatches each response to the matching pending channel.
func (t *stdioTransport) readLoop() {
	for t.scanner.Scan() {
		line := strings.TrimSpace(t.scanner.Text())
		if line == "" {
			continue
		}
		var resp jsonRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			slog.Warn("mcp stdio: parse error", "line", line, "err", err)
			continue
		}
		// Route to waiting caller.
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
	}
	// Process exited: close all pending channels with an error response.
	t.pendingMu.Lock()
	for id, ch := range t.pending {
		ch <- jsonRPCResponse{Error: &jsonRPCError{Code: -32000, Message: "process exited"}}
		delete(t.pending, id)
	}
	t.pendingMu.Unlock()
}

// sendRaw sends a JSON-RPC request to the child process and waits for the response.
// Does NOT enforce the MCP initialize handshake — used by ensureInitialized itself.
func (t *stdioTransport) sendRaw(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if err := t.ensureStarted(); err != nil {
		return nil, err
	}

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

	line, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan jsonRPCResponse, 1)
	t.pendingMu.Lock()
	t.pending[id] = ch
	t.pendingMu.Unlock()

	t.mu.Lock()
	_, writeErr := fmt.Fprintf(t.stdin, "%s\n", line)
	t.mu.Unlock()
	if writeErr != nil {
		t.pendingMu.Lock()
		delete(t.pending, id)
		t.pendingMu.Unlock()
		return nil, fmt.Errorf("stdio write: %w", writeErr)
	}

	select {
	case <-ctx.Done():
		t.pendingMu.Lock()
		delete(t.pending, id)
		t.pendingMu.Unlock()
		return nil, ctx.Err()
	case resp := <-ch:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	}
}

// ensureInitialized performs the MCP initialize handshake exactly once.
// The MCP protocol requires this before any tools/list or tools/call.
// Uses t.initTimeout (default 30s) — subprocess may need time to load ML models.
func (t *stdioTransport) ensureInitialized() {
	t.initOnce.Do(func() {
		slog.Debug("mcp stdio: starting initialize handshake", "command", t.command, "timeout", t.initTimeout)
		ctx, cancel := context.WithTimeout(context.Background(), t.initTimeout)
		defer cancel()

		params := map[string]any{
			"protocolVersion": "2024-11-05",
			"clientInfo":      map[string]any{"name": "liaotao", "version": "0.1"},
			"capabilities":    map[string]any{},
		}
		if _, err := t.sendRaw(ctx, "initialize", params); err != nil {
			t.initErr = fmt.Errorf("mcp initialize: %w", err)
			return
		}

		// notifications/initialized has no ID and expects no response.
		notif, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
		t.mu.Lock()
		fmt.Fprintf(t.stdin, "%s\n", notif) //nolint:errcheck
		t.mu.Unlock()

		slog.Debug("mcp stdio: initialized", "command", t.command)
	})
}

// call ensures the MCP handshake has been done, then sends the request.
func (t *stdioTransport) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	t.ensureInitialized()
	if t.initErr != nil {
		return nil, t.initErr
	}
	return t.sendRaw(ctx, method, params)
}

// ListTools calls tools/list on the child process.
func (t *stdioTransport) ListTools(ctx context.Context) ([]MCPTool, error) {
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

// CallTool invokes a tool on the child process and returns its text content.
func (t *stdioTransport) CallTool(ctx context.Context, toolName string, argsJSON string) (string, error) {
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

// Close terminates the child process.
func (t *stdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stdin != nil {
		_ = t.stdin.Close()
		t.stdin = nil
	}
	if t.cmd != nil {
		err := t.cmd.Process.Kill()
		t.cmd = nil
		return err
	}
	return nil
}
