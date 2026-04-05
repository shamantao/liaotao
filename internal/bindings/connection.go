/*
  connection.go -- TestConnection binding for provider connectivity checks.
  Measures round-trip latency and model count by calling the provider's model listing endpoint.
*/

package bindings

import (
	"context"
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
		return TestConnectionResult{LatencyMs: latency, Error: testErr.Error()}, nil
	}

	return TestConnectionResult{
		OK:         true,
		LatencyMs:  latency,
		ModelCount: len(models),
	}, nil
}
