/*
  chat.go -- Public chat bindings exposed to the frontend via Wails v3.
  Responsibilities: ListModels, SendMessage, CancelGeneration, streamWithFallback.
  Heavy stream I/O is in chat_stream.go; internal helpers are in chat_helpers.go.
  API keys are resolved Go-side from the provider DB — never accepted from the frontend.
*/

package bindings

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// SendMessagePayload is the frontend request payload for model generation.
// APIKey, Provider and BaseURL have been intentionally removed — resolved Go-side from DB.
type SendMessagePayload struct {
	ConversationID  string              `json:"conversation_id"`
	ProviderID      int64               `json:"provider_id"`
	Model           string              `json:"model"`
	Prompt          string              `json:"prompt"`
	Stream          bool                `json:"stream"`
	Temperature     float64             `json:"temperature"`
	MaxTokens       int                 `json:"max_tokens"`
	SystemPrompt    string              `json:"system_prompt"`
	NumCtx          int                 `json:"num_ctx"`
	ContextMessages []openAIChatMessage `json:"-"`
}

// streamMeta carries response metadata emitted via chat:meta after a successful generation.
type streamMeta struct {
	ProviderName    string // user-given name of the provider that answered
	Model           string // model name resolved at routing time
	TokensUsed      int    // estimated input tokens (len(prompt)/4)
	TokensOut       int    // estimated output tokens
	DurationMS      int64  // wall-clock duration for the assistant reply
	TokensRemaining int    // 0 = no quota configured; >0 = tokens left before quota switch
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
// When ProviderID > 0 the request is pinned to that provider (manual override, ROUTER-07).
// When ProviderID == 0 the Smart Router selects the best provider by priority and quota.
func (s *Service) SendMessage(_ context.Context, payload SendMessagePayload) (SendMessageResult, error) {
	userPrompt := strings.TrimSpace(payload.Prompt)
	if userPrompt == "" {
		return SendMessageResult{OK: false, Reason: "empty-prompt"}, nil
	}
	promptForModel := userPrompt

	// Resolve candidates: manual override when ProviderID > 0, router otherwise.
	var candidates []routerCandidate
	if payload.ProviderID > 0 {
		cfg, err := s.getProviderCredentials(context.Background(), payload.ProviderID)
		if err != nil {
			return SendMessageResult{OK: false, Reason: "provider-not-found"}, nil
		}
		name, _ := s.getProviderName(context.Background(), payload.ProviderID)
		daily, monthly := s.getQuotaLimits(context.Background(), payload.ProviderID)
		candidates = []routerCandidate{{
			ProviderID:   payload.ProviderID,
			Name:         name,
			Cfg:          cfg,
			DailyLimit:   daily,
			MonthlyLimit: monthly,
		}}
	} else {
		var routerErr error
		candidates, routerErr = s.selectCandidates(context.Background())
		if routerErr != nil {
			slog.Warn("smart router: all quotas exhausted")
			return SendMessageResult{OK: false, Reason: "all-quotas-exhausted"}, nil
		}
	}

	first := candidates[0].Cfg
	model := strings.TrimSpace(payload.Model)
	temperature := payload.Temperature
	if temperature <= 0 {
		temperature = 0.7
	}
	if temperature > 2 {
		temperature = 2
	}
	payload.Temperature = temperature
	if payload.MaxTokens < 0 {
		payload.MaxTokens = 0
	}
	payload.SystemPrompt = strings.TrimSpace(payload.SystemPrompt)
	// In manual-override mode, fall back to the provider's default when no model selected.
	// In Automat mode (model == ""), streamWithCandidates resolves the model per-candidate.
	if model == "" && payload.ProviderID > 0 {
		model = first.DefaultModel
	}

	convID := strings.TrimSpace(payload.ConversationID)
	if convID == "" {
		convID = "default"
	}

	if convNumericID, err := strconv.ParseInt(convID, 10, 64); err == nil && convNumericID > 0 {
		rc, ctxErr := s.buildRequestContext(context.Background(), convNumericID, userPrompt)
		if ctxErr != nil {
			slog.Warn("request context build failed", "conversation_id", convNumericID, "err", ctxErr)
		} else {
			payload.ContextMessages = requestContextMessagesToOpenAI(rc.RecentMessages)
			promptForModel = promptWithRequestContext(userPrompt, rc)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.setCancel(convID, cancel)

	go func() {
		defer s.clearCancel(convID)
		meta, err := s.streamWithCandidates(ctx, convID, candidates, payload, model, promptForModel, userPrompt)
		if err != nil {
			s.emitStructuredError(convID, err)
			s.emit("chat:done", map[string]any{"conversation_id": convID, "done": true})
			return
		}
		if meta.ProviderName != "" {
			s.emit("chat:meta", map[string]any{
				"conversation_id":  convID,
				"provider_name":    meta.ProviderName,
				"model":            meta.Model,
				"tokens_used":      meta.TokensUsed,
				"tokens_out":       meta.TokensOut,
				"duration_ms":      meta.DurationMS,
				"tokens_remaining": meta.TokensRemaining,
			})
		}
	}()

	return SendMessageResult{OK: true, Provider: first.Type, Model: model}, nil
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

// streamWithCandidates attempts streaming across an ordered candidate list (ROUTER-04).
// Resolves the model per-candidate when caller passes empty string (Automat mode).
// In single-candidate mode (manual override) any error is terminal.
// In multi-candidate mode (Automat) all providers are tried before giving up.
// Token usage is tracked for the provider that successfully serves the request (ROUTER-03).
// Returns streamMeta with provider/model/token info for the ROUTER-08 metadata footer.
func (s *Service) streamWithCandidates(ctx context.Context, convID string, candidates []routerCandidate, payload SendMessagePayload, model, promptForModel, userPrompt string) (streamMeta, error) {
	var lastErr error
	for _, c := range candidates {
		startedAt := time.Now()
		// Resolve model per-candidate in Automat mode (model == "").
		resolvedModel := model
		if resolvedModel == "" {
			resolvedModel = s.resolveModelForCandidate(ctx, c)
		}
		var err error
		// All provider types go through the OpenAI-compat tool-call loop.
		// Ollama exposes /v1/chat/completions (OpenAI-compat) and supports tool_calls.
		// streamOpenAIWithToolsRetry already falls back to streamOllama if the /v1 call fails.
		err = s.streamOpenAIWithToolsRetry(ctx, convID, c.Cfg, payload, resolvedModel, promptForModel)
		if err == nil {
			// Estimate input tokens (1 token ≈ 4 chars) and record usage (ROUTER-03).
			tokenEstimate := estimateTokenCount(userPrompt)
			_ = s.incrementTokenUsage(context.Background(), c.ProviderID, tokenEstimate)
			remaining := s.getTokensRemaining(context.Background(), c.ProviderID, c.DailyLimit, c.MonthlyLimit)
			durationMS := time.Since(startedAt).Milliseconds()
			return streamMeta{
				ProviderName:    c.Name,
				Model:           resolvedModel,
				TokensUsed:      tokenEstimate,
				TokensOut:       0,
				DurationMS:      durationMS,
				TokensRemaining: remaining,
			}, nil
		}
		lastErr = err
		// Single candidate: return immediately (manual override mode — no fallback).
		if len(candidates) == 1 {
			return streamMeta{}, err
		}
		slog.Debug("router: candidate failed, trying next", "provider_id", c.ProviderID, "err", err)
	}
	return streamMeta{}, lastErr
}

// streamOpenAIWithToolsRetry wraps streamOpenAIWithTools with up to 3 attempts on 429.
// Delegates to streamOpenAIWithTools which handles the full MCP tool-call loop.
func (s *Service) streamOpenAIWithToolsRetry(ctx context.Context, convID string, cfg openAIConfig, payload SendMessagePayload, model, prompt string) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		lastErr = s.streamOpenAIWithTools(ctx, convID, cfg, payload, model, prompt)
		if lastErr == nil {
			return nil
		}
		pe, isProvider := lastErr.(ProviderError)
		if !isProvider || !pe.Retryable || attempt >= 2 {
			break
		}
		wait := calcBackoff(attempt)
		if pe.RetryAfterSec > 0 {
			wait = time.Duration(pe.RetryAfterSec) * time.Second
		}
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	// Last resort for OpenAI-compat providers: fall back to native Ollama if reachable.
	if ollamaErr := s.streamOllama(ctx, convID, cfg, payload, model, prompt); ollamaErr == nil {
		return nil
	}
	return lastErr
}

// streamOpenAIWithRetry wraps streamOpenAI with up to 3 attempts on 429 responses.
func (s *Service) streamOpenAIWithRetry(ctx context.Context, convID string, cfg openAIConfig, payload SendMessagePayload, model, prompt string) error {
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
		wait := calcBackoff(attempt)
		if pe.RetryAfterSec > 0 {
			wait = time.Duration(pe.RetryAfterSec) * time.Second
		}
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	// Last resort for OpenAI-compat providers: fall back to native Ollama if reachable.
	if ollamaErr := s.streamOllama(ctx, convID, cfg, payload, model, prompt); ollamaErr == nil {
		return nil
	}
	return lastErr
}
