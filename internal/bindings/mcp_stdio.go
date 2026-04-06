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
)

// stdioTransport implements mcpTransport over a child process stdin/stdout.
type stdioTransport struct {
	command string
	args    []string

	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	nextID  atomic.Int64

	// pending maps request ID → response channel.
	pendingMu sync.Mutex
	pending   map[int64]chan jsonRPCResponse
}

// newStdioTransport creates a stdio transport that will spawn the given command.
// The process is not started until the first call to ListTools or CallTool.
func newStdioTransport(command string, args []string) *stdioTransport {
	return &stdioTransport{
		command: command,
		args:    args,
		pending: make(map[int64]chan jsonRPCResponse),
	}
}

// ensureStarted lazily spawns the child process on first use.
func (t *stdioTransport) ensureStarted() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cmd != nil {
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

// call sends a JSON-RPC request to the child process and waits for the response.
func (t *stdioTransport) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
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
	if t.cmd == nil {
		return nil
	}
	_ = t.stdin.Close()
	err := t.cmd.Process.Kill()
	t.cmd = nil
	return err
}
