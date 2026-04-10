/*
  mcp_loop.go -- Tool-call loop for MCP-aware generation (MCP-05).
  Responsibilities: detect tool calls in the OpenAI streaming response,
  dispatch them via DispatchToolCalls, re-inject results into the conversation
  and continue generation. Supports OpenAI-compat (tool_calls delta) and
  Ollama (function call in content, parsed as JSON).
  Called from streamWithCandidates when tools are available.
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
)

// maxToolLoopIterations caps the tool-call loop to prevent infinite cycles.
const maxToolLoopIterations = 10

// mcpToolsToOpenAI converts MCPTool definitions to the OpenAI tools array format.
func mcpToolsToOpenAI(tools []MCPTool) []openAIToolDef {
	result := make([]openAIToolDef, 0, len(tools))
	for _, t := range tools {
		result = append(result, openAIToolDef{
			Type: "function",
			Function: openAIFuncDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return result
}

// streamOpenAIWithTools runs the full tool-call loop for OpenAI-compat providers.
// If no tools are available (empty list), falls back to plain streamOpenAI.
// Each iteration: stream → detect tool_calls → dispatch → re-inject → continue.
func (s *Service) streamOpenAIWithTools(ctx context.Context, convID string, cfg openAIConfig, payload SendMessagePayload, model, prompt string) error {
	tools := s.AllAvailableTools(ctx)
	if len(tools) == 0 {
		return s.streamOpenAI(ctx, convID, cfg, payload, model, prompt)
	}

	// Build conversation history: start with the user message.
	messages := []openAIChatMessage{}
	if payload.SystemPrompt != "" {
		messages = append(messages, openAIChatMessage{Role: "system", Content: payload.SystemPrompt})
	}
	messages = append(messages, openAIChatMessage{Role: "user", Content: prompt})

	for iteration := 0; iteration < maxToolLoopIterations; iteration++ {
		toolCalls, assistantText, err := s.streamOpenAICollecting(ctx, convID, cfg, payload, model, messages, tools, iteration == 0)
		if err != nil {
			return err
		}

		// No tool calls — model gave a final text answer.
		if len(toolCalls) == 0 {
			return nil
		}

		// Emit tool-calling indicator to frontend (MCP-08).
		for _, tc := range toolCalls {
			s.emit("chat:tool_call", map[string]any{
				"conversation_id": convID,
				"tool_name":       tc.Function.Name,
				"status":          "calling",
			})
		}

		// Dispatch tool calls and collect results.
		results := s.DispatchToolCalls(ctx, toolCalls)

		// Emit tool results to frontend for display (MCP-09).
		for _, r := range results {
			s.emit("chat:tool_result", map[string]any{
				"conversation_id": convID,
				"tool_call_id":    r.ToolCallID,
				"content":         r.Content,
			})
		}

		// Update message history: append assistant message with tool calls + tool results.
		messages = append(messages, openAIChatMessage{
			Role:      "assistant",
			Content:   assistantText,
			ToolCalls: toolCalls,
		})
		for _, r := range results {
			messages = append(messages, openAIChatMessage{
				Role:       "tool",
				Content:    r.Content,
				ToolCallID: r.ToolCallID,
			})
		}
	}

	return fmt.Errorf("tool-call loop exceeded %d iterations", maxToolLoopIterations)
}

// streamOpenAICollecting streams one generation turn and collects tool calls.
// emitChunks controls whether text tokens are emitted to the frontend (true on first turn).
// Returns (toolCalls, assistantText, error).
func (s *Service) streamOpenAICollecting(
	ctx context.Context,
	convID string,
	cfg openAIConfig,
	payload SendMessagePayload,
	model string,
	messages []openAIChatMessage,
	tools []MCPTool,
	emitChunks bool,
) ([]ToolCall, string, error) {
	reqPayload := openAIChatRequest{
		Model:      model,
		Messages:   messages,
		Stream:     true,
		Tools:      mcpToolsToOpenAI(tools),
		ToolChoice: "auto",
	}
	if payload.Temperature > 0 {
		reqPayload.Temperature = &payload.Temperature
	}
	if payload.MaxTokens > 0 {
		reqPayload.MaxTokens = payload.MaxTokens
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(reqPayload); err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL+"/chat/completions", &body)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	res, err := s.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, "", classifyHTTPError(res.StatusCode, b)
	}

	return s.parseToolCallStream(ctx, convID, res.Body, emitChunks)
}

// parseToolCallStream reads an OpenAI SSE stream and separates text tokens from tool calls.
// Tool call deltas are accumulated per index and merged into complete ToolCall objects.
func (s *Service) parseToolCallStream(
	ctx context.Context,
	convID string,
	body io.Reader,
	emitChunks bool,
) ([]ToolCall, string, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var textBuilder strings.Builder
	// Accumulate tool call deltas keyed by index.
	type partialTC struct {
		id       string
		typ      string
		name     string
		argsAccu strings.Builder
	}
	partials := map[int]*partialTC{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if raw == "[DONE]" {
			break
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(raw), &chunk); err != nil || len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		// Accumulate text content.
		if delta.Content != "" {
			textBuilder.WriteString(delta.Content)
			if emitChunks {
				s.emit("chat:chunk", map[string]any{
					"conversation_id": convID,
					"content":         delta.Content,
					"done":            false,
				})
			}
		}

		// Accumulate tool call deltas.
		for _, tc := range delta.ToolCalls {
			p, ok := partials[tc.Index]
			if !ok {
				p = &partialTC{}
				partials[tc.Index] = p
			}
			if tc.ID != "" {
				p.id = tc.ID
			}
			if tc.Type != "" {
				p.typ = tc.Type
			}
			if tc.Function.Name != "" {
				p.name = tc.Function.Name
			}
			p.argsAccu.WriteString(tc.Function.Arguments)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}

	// If text was emitted, send done signals.
	if emitChunks && textBuilder.Len() > 0 {
		s.emit("chat:chunk", map[string]any{"conversation_id": convID, "content": "", "done": true})
		s.emit("chat:done", map[string]any{"conversation_id": convID, "done": true})
		s.persistAssistantMessage(ctx, convID, textBuilder.String(), nil)
	}

	// Build complete ToolCall list from accumulated partials.
	if len(partials) == 0 {
		// No tool calls: if no text either, emit done.
		if emitChunks && textBuilder.Len() == 0 {
			s.emit("chat:chunk", map[string]any{"conversation_id": convID, "content": "", "done": true})
			s.emit("chat:done", map[string]any{"conversation_id": convID, "done": true})
		}
		return nil, textBuilder.String(), nil
	}

	calls := make([]ToolCall, 0, len(partials))
	for idx := 0; idx < len(partials); idx++ {
		p, ok := partials[idx]
		if !ok {
			continue
		}
		id := p.id
		if id == "" {
			id = fmt.Sprintf("call_%d", idx)
		}
		calls = append(calls, ToolCall{
			ID:   id,
			Type: "function",
			Function: ToolCallFunction{
				Name:      p.name,
				Arguments: p.argsAccu.String(),
			},
		})
	}
	slog.Debug("mcp: tool calls detected", "count", len(calls))
	return calls, textBuilder.String(), nil
}
