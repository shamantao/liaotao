/*
  mcp_builtin.go -- Built-in MCP tools that need no external server (MCP-06).
  Responsibilities: current_datetime, calculator, read_file, web_fetch.
  These tools behave identically to external MCP tools from the tool-call loop's
  perspective — they are registered in the dispatcher and called by name.
*/

package bindings

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// builtinToolDefs returns the MCPTool descriptors for all built-in tools.
// These are merged with discovered server tools and sent to the model as
// available functions during tool-aware generation.
func builtinToolDefs() []MCPTool {
	return []MCPTool{
		{
			Name:        "current_datetime",
			Description: "Returns the current local date and time in ISO 8601 format.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
		},
		{
			Name:        "calculator",
			Description: "Evaluates a simple arithmetic expression and returns the result. Supports +, -, *, /, ^ (power), sqrt(x), and parentheses.",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{
					"expression":{"type":"string","description":"The arithmetic expression to evaluate, e.g. '2 + 3 * 4' or 'sqrt(16)'"}
				},
				"required":["expression"]
			}`),
		},
		{
			Name:        "read_file",
			Description: "Reads a text file from disk and returns its content. Only paths within the configured allowed_roots are accessible.",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{
					"path":{"type":"string","description":"Absolute path to the file to read."}
				},
				"required":["path"]
			}`),
		},
		{
			Name:        "web_fetch",
			Description: "Fetches the body of a public HTTP or HTTPS URL and returns its text content (max 64 KB). Localhost and private IP ranges are blocked.",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{
					"url":{"type":"string","description":"The public HTTP or HTTPS URL to fetch."}
				},
				"required":["url"]
			}`),
		},
	}
}

// dispatchBuiltin executes a built-in tool by name and returns its text result.
// allowedRoots restricts which paths the read_file tool may access.
// Returns ("", false) when the tool name is not a built-in.
func dispatchBuiltin(toolName string, argsJSON string, allowedRoots []string) (result string, handled bool) {
	switch toolName {
	case "current_datetime":
		return time.Now().Format(time.RFC3339), true
	case "calculator":
		var args struct {
			Expression string `json:"expression"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments — %s", err.Error()), true
		}
		val, err := evalExpression(strings.TrimSpace(args.Expression))
		if err != nil {
			return fmt.Sprintf("error: %s", err.Error()), true
		}
		// Format: integer when no fractional part, float otherwise.
		if val == math.Trunc(val) && !math.IsInf(val, 0) {
			return strconv.FormatInt(int64(val), 10), true
		}
		return strconv.FormatFloat(val, 'f', -1, 64), true
	case "read_file":
		return builtinReadFile(argsJSON, allowedRoots)
	case "web_fetch":
		return builtinWebFetch(argsJSON)
	default:
		return "", false
	}
}

// evalExpression is a minimal recursive descent parser for arithmetic expressions.
// Supports: +  -  *  /  ^  unary minus  sqrt(x)  parentheses.
func evalExpression(expr string) (float64, error) {
	p := &exprParser{input: expr, pos: 0}
	val, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	p.skipSpaces()
	if p.pos < len(p.input) {
		return 0, fmt.Errorf("unexpected character '%c' at position %d", p.input[p.pos], p.pos)
	}
	return val, nil
}

type exprParser struct {
	input string
	pos   int
}

func (p *exprParser) skipSpaces() {
	for p.pos < len(p.input) && p.input[p.pos] == ' ' {
		p.pos++
	}
}

// parseExpr handles addition and subtraction (lowest precedence).
func (p *exprParser) parseExpr() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpaces()
		if p.pos >= len(p.input) {
			break
		}
		op := p.input[p.pos]
		if op != '+' && op != '-' {
			break
		}
		p.pos++
		right, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

// parseTerm handles multiplication and division.
func (p *exprParser) parseTerm() (float64, error) {
	left, err := p.parsePower()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpaces()
		if p.pos >= len(p.input) {
			break
		}
		op := p.input[p.pos]
		if op != '*' && op != '/' {
			break
		}
		p.pos++
		right, err := p.parsePower()
		if err != nil {
			return 0, err
		}
		if op == '*' {
			left *= right
		} else {
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left /= right
		}
	}
	return left, nil
}

// parsePower handles ^ (right-associative exponentiation).
func (p *exprParser) parsePower() (float64, error) {
	base, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	p.skipSpaces()
	if p.pos < len(p.input) && p.input[p.pos] == '^' {
		p.pos++
		exp, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		return math.Pow(base, exp), nil
	}
	return base, nil
}

// parseUnary handles unary minus and delegates to parseAtom.
func (p *exprParser) parseUnary() (float64, error) {
	p.skipSpaces()
	if p.pos < len(p.input) && p.input[p.pos] == '-' {
		p.pos++
		val, err := p.parseAtom()
		if err != nil {
			return 0, err
		}
		return -val, nil
	}
	return p.parseAtom()
}

// parseAtom handles numbers, parenthesised expressions, and sqrt().
func (p *exprParser) parseAtom() (float64, error) {
	p.skipSpaces()
	if p.pos >= len(p.input) {
		return 0, fmt.Errorf("unexpected end of expression")
	}

	// Parenthesised sub-expression
	if p.input[p.pos] == '(' {
		p.pos++
		val, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		p.skipSpaces()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		p.pos++
		return val, nil
	}

	// sqrt() function
	if strings.HasPrefix(p.input[p.pos:], "sqrt(") {
		p.pos += 4 // skip "sqrt"
		val, err := p.parseAtom()
		if err != nil {
			return 0, err
		}
		if val < 0 {
			return 0, fmt.Errorf("sqrt of negative number")
		}
		return math.Sqrt(val), nil
	}

	// Number literal
	start := p.pos
	if p.pos < len(p.input) && p.input[p.pos] == '.' {
		p.pos++
	}
	for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9' || p.input[p.pos] == '.') {
		p.pos++
	}
	if p.pos == start {
		return 0, fmt.Errorf("expected number at position %d (got '%c')", p.pos, p.input[p.pos])
	}
	return strconv.ParseFloat(p.input[start:p.pos], 64)
}

// ---------------------------------------------------------------------------
// read_file built-in tool

const readFileMaxBytes = 512 * 1024 // 512 KB hard cap

// builtinReadFile reads a text file and returns its content as a string.
// The path must be within one of the configured allowed_roots.
func builtinReadFile(argsJSON string, allowedRoots []string) (string, bool) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("error: invalid arguments — %s", err), true
	}
	if args.Path == "" {
		return "error: path is required", true
	}

	// Resolve to absolute, clean path to prevent directory traversal.
	abs, err := filepath.Abs(args.Path)
	if err != nil {
		return fmt.Sprintf("error: cannot resolve path — %s", err), true
	}

	// Enforce allowed_roots sandbox.
	if !isWithinAllowedRoots(abs, allowedRoots) {
		return fmt.Sprintf("error: access denied — path is outside allowed directories"), true
	}

	f, err := os.Open(abs) // #nosec G304 — path is validated against allowed_roots above
	if err != nil {
		return fmt.Sprintf("error: cannot open file — %s", err), true
	}
	defer f.Close()

	content, err := io.ReadAll(io.LimitReader(f, readFileMaxBytes))
	if err != nil {
		return fmt.Sprintf("error: cannot read file — %s", err), true
	}
	return string(content), true
}

// isWithinAllowedRoots checks that the given absolute path is under at least one root.
func isWithinAllowedRoots(abs string, roots []string) bool {
	for _, root := range roots {
		cleanRoot := filepath.Clean(root)
		if strings.HasPrefix(abs, cleanRoot+string(filepath.Separator)) || abs == cleanRoot {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// web_fetch built-in tool

const (
	webFetchMaxBytes = 64 * 1024        // 64 KB response cap
	webFetchTimeout  = 15 * time.Second // per-request timeout
)

// blockedPrefixes lists IP prefixes that must not be reachable via web_fetch (SSRF guard).
var blockedPrefixes = []string{
	"127.", "10.", "169.254.", "192.168.",
	"[::1]", "[fc", "[fd", "[fe80",
}

// builtinWebFetch fetches a public HTTP/HTTPS URL and returns its body as text.
// Blocks localhost and RFC-1918 private IP ranges to prevent SSRF.
func builtinWebFetch(argsJSON string) (string, bool) {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("error: invalid arguments — %s", err), true
	}
	rawURL := strings.TrimSpace(args.URL)
	if rawURL == "" {
		return "error: url is required", true
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Sprintf("error: invalid URL — %s", err), true
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "error: only http and https URLs are allowed", true
	}

	// SSRF guard: reject private/loopback hostnames and IPs.
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || host == "::1" {
		return "error: access to localhost is not allowed", true
	}
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(host, prefix) {
			return fmt.Sprintf("error: access to private IP range %s is not allowed", host), true
		}
	}

	// 172.16.0.0/12 requires a numeric check (172.16–172.31).
	if strings.HasPrefix(host, "172.") {
		parts := strings.Split(host, ".")
		if len(parts) >= 2 {
			if second, convErr := strconv.Atoi(parts[1]); convErr == nil && second >= 16 && second <= 31 {
				return fmt.Sprintf("error: access to private IP range %s is not allowed", host), true
			}
		}
	}

	client := &http.Client{Timeout: webFetchTimeout}
	resp, err := client.Get(rawURL) // #nosec G107 — URL validated above
	if err != nil {
		return fmt.Sprintf("error: fetch failed — %s", err), true
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Sprintf("error: HTTP %d — %s", resp.StatusCode, resp.Status), true
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, webFetchMaxBytes))
	if err != nil {
		return fmt.Sprintf("error: cannot read response — %s", err), true
	}
	return string(body), true
}
