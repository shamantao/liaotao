/*
  chat_stream.go -- Low-level streaming implementations for OpenAI-compat (SSE) and Ollama (NDJSON).
  Responsibilities: HTTP request construction, response parsing, chunk emission.
  Called from chat.go via streamWithFallback.
*/

package bindings

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Service) streamOpenAI(ctx context.Context, convID string, cfg openAIConfig, payload SendMessagePayload, model, prompt string) error {
	startedAt := time.Now()
	reqPayload := openAIChatRequest{
		Model: model,
		Messages: []openAIChatMessage{
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}
	if payload.Temperature > 0 {
		reqPayload.Temperature = &payload.Temperature
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(reqPayload); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL+"/chat/completions", &body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		pe := classifyHTTPError(res.StatusCode, b)
		// Capture Retry-After header for rate-limit backoff.
		if res.StatusCode == 429 {
			if ra := res.Header.Get("Retry-After"); ra != "" {
				if secs, parseErr := strconv.ParseInt(strings.TrimSpace(ra), 10, 64); parseErr == nil && secs > 0 {
					pe.RetryAfterSec = secs
				}
			}
		}
		return pe
	}

	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var assistantBuilder strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if raw == "[DONE]" {
			tokensOut := estimateTokenCount(assistantBuilder.String())
			durationMS := time.Since(startedAt).Milliseconds()
			stats := &MessageTokenStats{
				TokensOut:       tokensOut,
				DurationMS:      durationMS,
				TokensPerSecond: roundToOneDecimal(float64(tokensOut) / maxDurationSeconds(durationMS)),
				Estimated:       true,
			}
			s.emit("chat:chunk", map[string]any{
				"conversation_id": convID,
				"content":         "",
				"done":            true,
			})
			s.emit("chat:done", map[string]any{"conversation_id": convID, "done": true})
			s.persistAssistantMessage(ctx, convID, assistantBuilder.String(), stats)
			return nil
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(raw), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		token := chunk.Choices[0].Delta.Content
		if token == "" {
			continue
		}

		assistantBuilder.WriteString(token)
		s.emit("chat:chunk", map[string]any{
			"conversation_id": convID,
			"content":         token,
			"done":            false,
		})
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	s.emit("chat:chunk", map[string]any{"conversation_id": convID, "content": "", "done": true})
	s.emit("chat:done", map[string]any{"conversation_id": convID, "done": true})
	tokensOut := estimateTokenCount(assistantBuilder.String())
	durationMS := time.Since(startedAt).Milliseconds()
	s.persistAssistantMessage(ctx, convID, assistantBuilder.String(), &MessageTokenStats{
		TokensOut:       tokensOut,
		DurationMS:      durationMS,
		TokensPerSecond: roundToOneDecimal(float64(tokensOut) / maxDurationSeconds(durationMS)),
		Estimated:       true,
	})

	return nil
}

func (s *Service) streamOllama(ctx context.Context, convID string, cfg openAIConfig, payload SendMessagePayload, model, prompt string) error {
	startedAt := time.Now()
	ollamaBase := strings.TrimSuffix(cfg.BaseURL, "/v1")

	reqPayload := map[string]any{
		"model":  model,
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	options := map[string]any{}
	if payload.Temperature > 0 {
		options["temperature"] = payload.Temperature
	}
	if payload.NumCtx > 0 {
		options["num_ctx"] = payload.NumCtx
	}
	if len(options) > 0 {
		reqPayload["options"] = options
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(reqPayload); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ollamaBase+"/api/chat", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return classifyHTTPError(res.StatusCode, b)
	}

	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var assistantBuilder strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}

		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}

		if chunk.Message.Content != "" {
			assistantBuilder.WriteString(chunk.Message.Content)
			s.emit("chat:chunk", map[string]any{
				"conversation_id": convID,
				"content":         chunk.Message.Content,
				"done":            false,
			})
		}

		if chunk.Done {
			tokensOut := estimateTokenCount(assistantBuilder.String())
			durationMS := time.Since(startedAt).Milliseconds()
			s.emit("chat:chunk", map[string]any{"conversation_id": convID, "content": "", "done": true})
			s.emit("chat:done", map[string]any{"conversation_id": convID, "done": true})
			s.persistAssistantMessage(ctx, convID, assistantBuilder.String(), &MessageTokenStats{
				TokensOut:       tokensOut,
				DurationMS:      durationMS,
				TokensPerSecond: roundToOneDecimal(float64(tokensOut) / maxDurationSeconds(durationMS)),
				Estimated:       true,
			})
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	s.emit("chat:chunk", map[string]any{"conversation_id": convID, "content": "", "done": true})
	s.emit("chat:done", map[string]any{"conversation_id": convID, "done": true})
	tokensOut := estimateTokenCount(assistantBuilder.String())
	durationMS := time.Since(startedAt).Milliseconds()
	s.persistAssistantMessage(ctx, convID, assistantBuilder.String(), &MessageTokenStats{
		TokensOut:       tokensOut,
		DurationMS:      durationMS,
		TokensPerSecond: roundToOneDecimal(float64(tokensOut) / maxDurationSeconds(durationMS)),
		Estimated:       true,
	})

	return nil
}

func maxDurationSeconds(durationMS int64) float64 {
	if durationMS <= 0 {
		return 0.001
	}
	return float64(durationMS) / 1000
}
