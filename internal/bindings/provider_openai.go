/*
  provider_openai.go -- OpenAI-compatible provider helpers.
  Responsibilities: load provider runtime config from environment or DB,
  list models, classify HTTP errors, exponential backoff, and pre-configured profiles.
*/

package bindings

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultBaseURL = "http://localhost:11434/v1"
	defaultModel   = "gpt-4o-mini"
)

// ModelInfo is the frontend payload for one available model.
type ModelInfo struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
}

// openAIConfig holds the resolved runtime configuration for one provider call.
type openAIConfig struct {
	BaseURL      string
	APIKey       string
	DefaultModel string
	Type         string // provider type: "openai-compatible", "ollama", etc.
}

// ProviderError is a structured, user-friendly error from a provider HTTP call.
// It implements the error interface and carries a Retryable flag for backoff logic.
type ProviderError struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	Retryable     bool   `json:"retryable"`
	RetryAfterSec int64  `json:"retry_after_sec,omitempty"`
}

func (e ProviderError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// classifyHTTPError maps an HTTP status code to a user-friendly ProviderError.
func classifyHTTPError(status int, _ []byte) ProviderError {
	switch status {
	case 401:
		return ProviderError{Code: "unauthorized", Message: "Invalid API key — check your credentials", Retryable: false}
	case 403:
		return ProviderError{Code: "forbidden", Message: "Access denied by the provider", Retryable: false}
	case 404:
		return ProviderError{Code: "not-found", Message: "Model not found or incorrect base URL", Retryable: false}
	case 429:
		return ProviderError{Code: "rate-limited", Message: "Rate limited — retrying shortly", Retryable: true}
	case 500:
		return ProviderError{Code: "server-error", Message: "Provider server error", Retryable: true}
	case 503:
		return ProviderError{Code: "unavailable", Message: "Provider temporarily unavailable", Retryable: true}
	default:
		if status >= 400 && status < 500 {
			return ProviderError{Code: "client-error", Message: fmt.Sprintf("Request error (HTTP %d)", status), Retryable: false}
		}
		return ProviderError{Code: "server-error", Message: fmt.Sprintf("Server error (HTTP %d)", status), Retryable: true}
	}
}

// calcBackoff returns exponential backoff for the given attempt index (0-based).
// Duration: 2^attempt seconds + 0–1 s random jitter, capped at 30 s.
func calcBackoff(attempt int) time.Duration {
	exp := time.Duration(1<<uint(attempt)) * time.Second
	jitter := time.Duration(rand.Intn(1000)) * time.Millisecond //nolint:gosec // jitter, not security
	wait := exp + jitter
	const maxWait = 30 * time.Second
	if wait > maxWait {
		wait = maxWait
	}
	return wait
}

// ProviderProfile is a pre-configured provider template shown in the settings form.
type ProviderProfile struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Type    string `json:"type"`
	DocsURL string `json:"docs_url"`
}

var builtinProfiles = []ProviderProfile{
	{Key: "openai", Name: "OpenAI", BaseURL: "https://api.openai.com/v1", Type: "openai-compatible", DocsURL: "https://platform.openai.com/api-keys"},
	{Key: "gemini", Name: "Google Gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", Type: "openai-compatible", DocsURL: "https://aistudio.google.com/app/apikey"},
	{Key: "openrouter", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", Type: "openai-compatible", DocsURL: "https://openrouter.ai/keys"},
	{Key: "groq", Name: "Groq", BaseURL: "https://api.groq.com/openai/v1", Type: "openai-compatible", DocsURL: "https://console.groq.com/keys"},
	{Key: "together", Name: "Together AI", BaseURL: "https://api.together.xyz/v1", Type: "openai-compatible", DocsURL: "https://api.together.ai/settings/api-keys"},
	{Key: "mistral", Name: "Mistral", BaseURL: "https://api.mistral.ai/v1", Type: "openai-compatible", DocsURL: "https://console.mistral.ai/api-keys"},
	{Key: "cohere", Name: "Cohere", BaseURL: "https://api.cohere.ai/compatibility/v1", Type: "openai-compatible", DocsURL: "https://dashboard.cohere.com/api-keys"},
	{Key: "ollama", Name: "Local Ollama", BaseURL: "http://localhost:11434/v1", Type: "ollama", DocsURL: "https://ollama.com"},
}

// ListProviderProfiles returns the list of built-in provider presets.
func (s *Service) ListProviderProfiles(_ context.Context) ([]ProviderProfile, error) {
	return builtinProfiles, nil
}

// --- internal structs for API payloads ---

type openAIModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature *float64            `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Tools       []openAIToolDef     `json:"tools,omitempty"`
	ToolChoice  string              `json:"tool_choice,omitempty"` // "auto" | "none"
}

type openAIChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // for role=tool messages
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // for role=assistant with tool calls
}

// openAIToolDef wraps an MCPTool into the OpenAI function-calling format.
type openAIToolDef struct {
	Type     string        `json:"type"` // always "function"
	Function openAIFuncDef `json:"function"`
}

type openAIFuncDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// loadOpenAIConfig reads provider config from environment variables.
func loadOpenAIConfig() openAIConfig {
	baseURL := strings.TrimSpace(firstNonEmpty(
		os.Getenv("LIAOTAO_OPENAI_BASE_URL"),
		os.Getenv("OPENAI_BASE_URL"),
		defaultBaseURL,
	))
	baseURL = strings.TrimRight(baseURL, "/")

	apiKey := strings.TrimSpace(firstNonEmpty(
		os.Getenv("LIAOTAO_OPENAI_API_KEY"),
		os.Getenv("OPENAI_API_KEY"),
		"ollama",
	))

	model := strings.TrimSpace(firstNonEmpty(
		os.Getenv("LIAOTAO_DEFAULT_MODEL"),
		defaultModel,
	))

	return openAIConfig{
		BaseURL:      baseURL,
		APIKey:       apiKey,
		DefaultModel: model,
	}
}

// resolveRuntimeConfig overlays UI-provided values on top of environment defaults.
// Kept for backward compatibility with tests.
func resolveRuntimeConfig(provider, baseURL, apiKey string) openAIConfig {
	cfg := loadOpenAIConfig()

	if strings.TrimSpace(baseURL) != "" {
		cfg.BaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}
	if strings.TrimSpace(apiKey) != "" {
		cfg.APIKey = strings.TrimSpace(apiKey)
	}
	cfg.Type = strings.TrimSpace(provider)

	if strings.TrimSpace(provider) == "ollama" && strings.TrimSpace(baseURL) == "" {
		cfg.BaseURL = defaultBaseURL
	}

	return cfg
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (s *Service) listOpenAIModels(ctx context.Context, cfg openAIConfig) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}

	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return nil, classifyHTTPError(res.StatusCode, body)
	}

	var payload openAIModelsResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make([]ModelInfo, 0, len(payload.Data))
	for _, model := range payload.Data {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		out = append(out, ModelInfo{ID: id, Provider: "openai-compatible"})
	}
	return out, nil
}

func (s *Service) listOllamaModels(ctx context.Context, cfg openAIConfig) ([]ModelInfo, error) {
	ollamaBase := strings.TrimSuffix(cfg.BaseURL, "/v1")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ollamaBase+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return nil, classifyHTTPError(res.StatusCode, body)
	}

	var payload ollamaTagsResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make([]ModelInfo, 0, len(payload.Models))
	for _, model := range payload.Models {
		id := strings.TrimSpace(model.Name)
		if id == "" {
			continue
		}
		out = append(out, ModelInfo{ID: id, Provider: "ollama"})
	}
	return out, nil
}
