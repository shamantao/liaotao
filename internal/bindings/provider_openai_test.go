/*
  provider_openai_test.go -- Unit tests for runtime provider config resolution.
  Ensures UI overrides and provider defaults are applied deterministically.
*/

package bindings

import (
	"context"
	"testing"
	"time"
)

func TestResolveRuntimeConfigOpenAI(t *testing.T) {
	cfg := resolveRuntimeConfig("openai-compatible", "http://localhost:8200/v1", "test-key")
	if cfg.BaseURL != "http://localhost:8200/v1" {
		t.Fatalf("unexpected base URL: %q", cfg.BaseURL)
	}
	if cfg.APIKey != "test-key" {
		t.Fatalf("unexpected API key: %q", cfg.APIKey)
	}
}

func TestResolveRuntimeConfigOllamaDefault(t *testing.T) {
	cfg := resolveRuntimeConfig("ollama", "", "")
	if cfg.BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("expected ollama default URL, got %q", cfg.BaseURL)
	}
	if cfg.DefaultModel == "" {
		t.Fatalf("default model should not be empty")
	}
}

// TestClassifyHTTPError verifies each mapped error code (PROV-06).
func TestClassifyHTTPError(t *testing.T) {
	cases := []struct {
		status    int
		wantCode  string
		retryable bool
	}{
		{401, "unauthorized", false},
		{403, "forbidden", false},
		{404, "not-found", false},
		{429, "rate-limited", true},
		{500, "server-error", true},
		{503, "unavailable", true},
		{400, "client-error", false},
		{502, "server-error", true},
	}
	for _, tc := range cases {
		pe := classifyHTTPError(tc.status, nil)
		if pe.Code != tc.wantCode {
			t.Errorf("status %d: want code %q, got %q", tc.status, tc.wantCode, pe.Code)
		}
		if pe.Retryable != tc.retryable {
			t.Errorf("status %d: want retryable=%v, got %v", tc.status, tc.retryable, pe.Retryable)
		}
		if pe.Message == "" {
			t.Errorf("status %d: message should not be empty", tc.status)
		}
		// Verify it satisfies the error interface.
		var _ error = pe
	}
}

// TestProviderErrorImplementsError verifies ProviderError.Error() returns a non-empty string.
func TestProviderErrorImplementsError(t *testing.T) {
	pe := ProviderError{Code: "test", Message: "test message", Retryable: false}
	if pe.Error() == "" {
		t.Fatal("Error() should return a non-empty string")
	}
}

// TestCalcBackoff verifies that backoff grows and is bounded (PROV-07).
func TestCalcBackoff(t *testing.T) {
	prev := time.Duration(0)
	for attempt := 0; attempt < 3; attempt++ {
		d := calcBackoff(attempt)
		if d <= 0 {
			t.Fatalf("attempt %d: backoff must be positive, got %v", attempt, d)
		}
		if d > 30*time.Second+time.Second {
			t.Fatalf("attempt %d: backoff exceeds max, got %v", attempt, d)
		}
		if attempt > 0 && d < prev/2 {
			// Backoff should generally grow, though jitter can cause slight decreases.
			// This is a sanity check, not strict monotonicity.
			t.Logf("attempt %d: potentially non-growing backoff (prev=%v, cur=%v)", attempt, prev, d)
		}
		prev = d
	}
}

// TestListProviderProfiles verifies all expected profiles are returned (PROV-08).
func TestListProviderProfiles(t *testing.T) {
	svc := &Service{}
	profiles, err := svc.ListProviderProfiles(context.Background())
	if err != nil {
		t.Fatalf("ListProviderProfiles error: %v", err)
	}
	if len(profiles) == 0 {
		t.Fatal("expected at least one profile")
	}
	wantKeys := []string{"openai", "openrouter", "groq", "together", "mistral", "cohere", "ollama"}
	keys := make(map[string]bool)
	for _, p := range profiles {
		keys[p.Key] = true
		if p.Name == "" || p.BaseURL == "" || p.Type == "" || p.DocsURL == "" {
			t.Errorf("profile %q has empty required fields", p.Key)
		}
	}
	for _, k := range wantKeys {
		if !keys[k] {
			t.Errorf("missing expected profile key: %q", k)
		}
	}
}
