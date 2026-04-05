/*
  chat.go -- Chat bindings for OpenAI-compatible and Ollama streaming.
  Responsibilities: expose ListModels/SendMessage/CancelGeneration to the frontend,
  stream SSE/NDJSON chunks via Wails events, handle cancellation, and backoff on 429.
  API keys are resolved Go-side from the provider DB — never accepted from the frontend.
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
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// SendMessagePayload is the frontend request payload for model generation.
// APIKey, Provider and BaseURL have been intentionally removed — resolved Go-side from DB.
type SendMessagePayload struct {
	ConversationID string  `json:"conversation_id"`
	ProviderID     int64   `json:"provider_id"`
	Model          string  `json:"model"`
	Prompt         string  `json:"prompt"`
	Stream         bool    `json:"stream"`
	Temperature    float64 `json:"temperature"`
	NumCtx         int     `json:"num_ctx"`
}

// ListModelsPayload requests available models for the given provider.
type ListModelsPayload struct {
	ProviderID int64 `json:"provider_id"`
}

// CancelPayload identifies which in-flight generation to cancel.
type CancelPayload struct {
	ConversationID string `json:"conversation_id"`
}

// SendMessageResult is the ack payload returned immediately to the frontend.
type SendMessageResult struct {
	OK       bool   `json:"ok"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// ListModels returns available models for the configured provider.
func (s *Service) ListModels(ctx context.Context, payload ListModelsPayload) ([]ModelInfo, error) {
	cfg, err := s.getProviderCredentials(ctx, payload.ProviderID)
	if err != nil {
		return []ModelInfo{{ID: defaultModel, Provider: "fallback"}}, nil
	}

	if cfg.Type == "ollama" {
		models, listErr := s.listOllamaModels(ctx, cfg)
		if listErr == nil && len(models) > 0 {
			return models, nil
		}
		return []ModelInfo{{ID: cfg.DefaultModel, Provider: "fallback"}}, nil
	}

	models, listErr := s.listOpenAIModels(ctx, cfg)
	if listErr == nil && len(models) > 0 {
		return models, nil
	}

	// Last-resort: try native Ollama for local providers.
	if models, ollamaErr := s.listOllamaModels(ctx, cfg); ollamaErr == nil && len(models) > 0 {
		return models, nil
	}

	return []ModelInfo{{ID: cfg.DefaultModel, Provider: "fallback"}}, nil
}

// SendMessage starts a streaming generation and returns immediately.
func (s *Service) SendMessage(_ context.Context, payload SendMessagePayload) (SendMessageResult, error) {
	prompt := strings.TrimSpace(payload.Prompt)
	if prompt == "" {
		return SendMessageResult{OK: false, Reason: "empty-prompt"}, nil
	}

	cfg, err := s.getProviderCredentials(context.Background(), payload.ProviderID)
	if err != nil {
		return SendMessageResult{OK: false, Reason: "provider-not-found"}, nil
	}

	model := strings.TrimSpace(payload.Model)
	if model == "" {
		model = cfg.DefaultModel
	}

	convID := strings.TrimSpace(payload.ConversationID)
	if convID == "" {
		convID = "default"
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.setCancel(convID, cancel)

	go func() {
		defer s.clearCancel(convID)
		if err := s.streamWithFallback(ctx, convID, cfg, payload, model, prompt); err != nil {
			s.emitStructuredError(convID, err)
			s.emit("chat:done", map[string]any{"conversation_id": convID, "done": true})
		}
	}()

	return SendMessageResult{OK: true, Provider: cfg.Type, Model: model}, nil
}

// CancelGeneration cancels one in-flight generation if present.
func (s *Service) CancelGeneration(_ context.Context, payload CancelPayload) (map[string]any, error) {
	convID := strings.TrimSpace(payload.ConversationID)
	if convID == "" {
		convID = "default"
	}

	stopped := s.cancelStream(convID)
	if stopped {
		s.emit("chat:done", map[string]any{"conversation_id": convID, "done": true, "cancelled": true})
	}

	return map[string]any{"ok": true, "cancelled": stopped}, nil
}

// streamWithFallback attempts OpenAI SSE streaming with up to 3 retries on 429,
// then falls back to native Ollama streaming.
func (s *Service) streamWithFallback(ctx context.Context, convID string, cfg openAIConfig, payload SendMessagePayload, model, prompt string) error {
	if cfg.Type == "ollama" {
		return s.streamOllama(ctx, convID, cfg, payload, model, prompt)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		lastErr = s.streamOpenAI(ctx, convID, cfg, payload, model, prompt)
		if lastErr == nil {
			return nil
		}

		pe, isProvider := lastErr.(ProviderError)
		if !isProvider || !pe.Retryable || attempt >= 2 {
			break
		}

		var wait time.Duration
		if pe.RetryAfterSec > 0 {
			wait = time.Duration(pe.RetryAfterSec) * time.Second
		} else {
			wait = calcBackoff(attempt)
		}

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// OpenAI route exhausted — try native Ollama as a last resort for local providers.
	if ollamaErr := s.streamOllama(ctx, convID, cfg, payload, model, prompt); ollamaErr == nil {
		return nil
	}
	return lastErr
}

func (s *Service) streamOpenAI(ctx context.Context, convID string, cfg openAIConfig, payload SendMessagePayload, model, prompt string) error {
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
			s.emit("chat:chunk", map[string]any{
				"conversation_id": convID,
				"content":         "",
				"done":            true,
			})
			s.emit("chat:done", map[string]any{"conversation_id": convID, "done": true})
			s.persistAssistantMessage(ctx, convID, assistantBuilder.String())
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
	s.persistAssistantMessage(ctx, convID, assistantBuilder.String())

	return nil
}

func (s *Service) streamOllama(ctx context.Context, convID string, cfg openAIConfig, payload SendMessagePayload, model, prompt string) error {
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
			s.emit("chat:chunk", map[string]any{"conversation_id": convID, "content": "", "done": true})
			s.emit("chat:done", map[string]any{"conversation_id": convID, "done": true})
			s.persistAssistantMessage(ctx, convID, assistantBuilder.String())
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	s.emit("chat:chunk", map[string]any{"conversation_id": convID, "content": "", "done": true})
	s.emit("chat:done", map[string]any{"conversation_id": convID, "done": true})
	s.persistAssistantMessage(ctx, convID, assistantBuilder.String())

	return nil
}

// emitStructuredError emits a chat:error event with a user-friendly payload.
func (s *Service) emitStructuredError(convID string, err error) {
	var payload map[string]any
	if pe, ok := err.(ProviderError); ok {
		payload = map[string]any{
			"conversation_id": convID,
			"code":            pe.Code,
			"message":         pe.Message,
			"retryable":       pe.Retryable,
		}
	} else {
		payload = map[string]any{
			"conversation_id": convID,
			"code":            "unknown",
			"message":         fmt.Sprintf("Generation failed: %s", err.Error()),
			"retryable":       false,
		}
	}
	s.emit("chat:error", payload)
}

func (s *Service) emit(name string, payload any) {
	app := application.Get()
	if app == nil || app.Event == nil {
		return
	}
	app.Event.Emit(name, payload)
}

func (s *Service) setCancel(convID string, cancel context.CancelFunc) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()

	if previous, ok := s.cancels[convID]; ok {
		previous()
	}
	s.cancels[convID] = cancel
}

func (s *Service) clearCancel(convID string) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	delete(s.cancels, convID)
}

func (s *Service) cancelStream(convID string) bool {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()

	cancel, ok := s.cancels[convID]
	if !ok {
		return false
	}
	cancel()
	delete(s.cancels, convID)
	return true
}

// persistAssistantMessage saves a completed assistant reply to the conversation DB.
// Non-numeric or zero conversation IDs (e.g. "default") are silently skipped.
func (s *Service) persistAssistantMessage(ctx context.Context, convID, content string) {
	if content == "" {
		return
	}
	id, err := strconv.ParseInt(convID, 10, 64)
	if err != nil || id <= 0 {
		return
	}
	_ = s.SaveMessage(ctx, MessagePayload{
		ConversationID: id,
		Role:           "assistant",
		Content:        content,
	})
}
