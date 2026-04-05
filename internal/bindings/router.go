/*
  router.go -- Smart Router: priority-based, quota-aware provider selection.
  Implements PRD §6.11 (ROUTER-01..07).
  selectCandidates returns an ordered list of active providers with quota remaining.
  Manual override (providerID > 0 in SendMessage) bypasses this entirely.
*/

package bindings

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// routerCandidate holds a resolved provider config for a single routing attempt.
type routerCandidate struct {
	ProviderID int64
	Cfg        openAIConfig
}

// orderedProvider holds routing metadata joined from providers + quota tables.
type orderedProvider struct {
	ID           int64
	Type         string
	DailyLimit   int
	MonthlyLimit int
	Priority     int
}

// selectCandidates returns active providers ordered by ascending priority that
// still have quota remaining. Falls back to the env-based config when no
// providers are configured in the database.
func (s *Service) selectCandidates(ctx context.Context) ([]routerCandidate, error) {
	ordered, err := s.getOrderedProviders(ctx)
	if err != nil || len(ordered) == 0 {
		slog.Debug("no active providers in DB, using env fallback")
		cfg := loadOpenAIConfig()
		return []routerCandidate{{ProviderID: 0, Cfg: cfg}}, nil
	}

	var candidates []routerCandidate
	for _, p := range ordered {
		exceeded, checkErr := s.isQuotaExceeded(ctx, p)
		if checkErr != nil {
			// On DB error, include the provider — fail open is safer than locking user out.
			slog.Warn("quota check error, including provider", "provider_id", p.ID, "err", checkErr)
			exceeded = false
		}
		if exceeded {
			slog.Debug("provider quota exhausted, skipping", "provider_id", p.ID)
			continue
		}
		cfg, credErr := s.getProviderCredentials(ctx, p.ID)
		if credErr != nil {
			slog.Warn("credential fetch failed, skipping provider", "provider_id", p.ID, "err", credErr)
			continue
		}
		candidates = append(candidates, routerCandidate{ProviderID: p.ID, Cfg: cfg})
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("all-quotas-exhausted")
	}
	return candidates, nil
}

// getOrderedProviders fetches active providers joined with quota config and priority,
// sorted by priority ascending (lower number = higher priority), then by id ASC.
func (s *Service) getOrderedProviders(ctx context.Context) ([]orderedProvider, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.type,
		       COALESCE(qc.daily_limit,   0),
		       COALESCE(qc.monthly_limit, 0),
		       COALESCE(pp.priority,      100)
		FROM providers p
		LEFT JOIN provider_quota_config qc ON qc.provider_id = p.id
		LEFT JOIN provider_priority     pp ON pp.provider_id  = p.id
		WHERE p.active = 1
		ORDER BY COALESCE(pp.priority, 100) ASC, p.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []orderedProvider
	for rows.Next() {
		var op orderedProvider
		if err := rows.Scan(&op.ID, &op.Type, &op.DailyLimit, &op.MonthlyLimit, &op.Priority); err != nil {
			return nil, err
		}
		result = append(result, op)
	}
	return result, rows.Err()
}

// isQuotaExceeded returns true when the provider has quota limits configured and
// the current usage for either the daily or monthly period meets or exceeds them.
func (s *Service) isQuotaExceeded(ctx context.Context, op orderedProvider) (bool, error) {
	if op.DailyLimit == 0 && op.MonthlyLimit == 0 {
		return false, nil // No limits configured — provider is always eligible.
	}
	now := time.Now()
	if op.DailyLimit > 0 {
		used, err := s.getUsageForPeriod(ctx, op.ID, now.Format("2006-01-02"), "daily")
		if err != nil {
			return false, err
		}
		if used >= op.DailyLimit {
			return true, nil
		}
	}
	if op.MonthlyLimit > 0 {
		used, err := s.getUsageForPeriod(ctx, op.ID, now.Format("2006-01"), "monthly")
		if err != nil {
			return false, err
		}
		if used >= op.MonthlyLimit {
			return true, nil
		}
	}
	return false, nil
}

// resolveModelForCandidate fetches the first non-fallback model listed by a provider.
// Called only when the user has not pinned a specific model (Automat mode, model == "").
// Falls back to the config DefaultModel when the listing fails or returns no usable model.
func (s *Service) resolveModelForCandidate(ctx context.Context, c routerCandidate) string {
	if c.ProviderID <= 0 {
		return c.Cfg.DefaultModel
	}
	models, err := s.ListModels(ctx, ListModelsPayload{ProviderID: c.ProviderID})
	if err != nil || len(models) == 0 {
		return c.Cfg.DefaultModel
	}
	for _, m := range models {
		if m.Provider != "fallback" && m.ID != "" {
			return m.ID
		}
	}
	return c.Cfg.DefaultModel
}
