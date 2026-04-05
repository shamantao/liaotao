/*
  chat.go -- Public chat bindings exposed to the frontend via Wails v3.
  Responsibilities: ListModels, SendMessage, CancelGeneration, streamWithFallback.
  Heavy stream I/O is in chat_stream.go; internal helpers are in chat_helpers.go.
  API keys are resolved Go-side from the provider DB — never accepted from the frontend.
*/

package bindings

import (
	"context"
	"strings"
	"time"
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

