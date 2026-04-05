/*
  connection.go -- TestConnection binding for provider connectivity checks.
  Measures round-trip latency and model count by calling the provider's model listing endpoint.
*/

package bindings

import (
	"context"
	"log/slog"
	"time"
)

// TestConnectionPayload identifies the provider to test.
type TestConnectionPayload struct {
	ProviderID int64 `json:"provider_id"`
}

// TestConnectionResult is the outcome of a connectivity check.
type TestConnectionResult struct {
	OK         bool   `json:"ok"`
	LatencyMs  int64  `json:"latency_ms"`
	ModelCount int    `json:"model_count"`
	Error      string `json:"error,omitempty"`
}

// TestConnection verifies connectivity for one provider and measures first-response latency.
func (s *Service) TestConnection(ctx context.Context, payload TestConnectionPayload) (TestConnectionResult, error) {
	slog.Debug("TestConnection called", "provider_id", payload.ProviderID)
	cfg, err := s.getProviderCredentials(ctx, payload.ProviderID)
	if err != nil {
		return TestConnectionResult{Error: "provider not found"}, nil
	}

	start := time.Now()
	var models []ModelInfo
	var testErr error

	if cfg.Type == "ollama" {
		models, testErr = s.listOllamaModels(ctx, cfg)
	} else {
		models, testErr = s.listOpenAIModels(ctx, cfg)
	}

	latency := time.Since(start).Milliseconds()

	if testErr != nil {
		slog.Warn("TestConnection failed", "provider_id", payload.ProviderID, "latency_ms", latency, "err", testErr)
		return TestConnectionResult{LatencyMs: latency, Error: testErr.Error()}, nil
	}

	slog.Debug("TestConnection ok", "provider_id", payload.ProviderID, "latency_ms", latency, "models", len(models))
	return TestConnectionResult{
		OK:         true,
		LatencyMs:  latency,
		ModelCount: len(models),
	}, nil
}
