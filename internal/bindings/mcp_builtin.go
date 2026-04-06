/*
  mcp_builtin.go -- Built-in MCP tools that need no external server (MCP-06).
  Responsibilities: current_datetime, calculator.
  These tools behave identically to external MCP tools from the tool-call loop's
  perspective — they are registered in the dispatcher and called by name.
*/

package bindings

import (
	"encoding/json"
	"fmt"
	"math"
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
	}
}

// dispatchBuiltin executes a built-in tool by name and returns its text result.
// Returns ("", false) when the tool name is not a built-in.
func dispatchBuiltin(toolName string, argsJSON string) (result string, handled bool) {
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
