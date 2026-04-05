/*
  quota_db.go -- SQLite operations for provider quota tracking.
  Handles token usage increments, period-based usage reads,
  and CRUD for quota configuration and provider priority.
*/

package bindings

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// getUsageForPeriod returns the tokens_used for the given provider and period.
// Returns 0 with no error when no row exists yet (sql.ErrNoRows).
func (s *Service) getUsageForPeriod(ctx context.Context, providerID int64, period, periodType string) (int, error) {
	var used int
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(tokens_used, 0)
		 FROM provider_quota_usage
		 WHERE provider_id=? AND period=? AND period_type=?`,
		providerID, period, periodType,
	).Scan(&used)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return used, err
}

// incrementTokenUsage adds tokenDelta to both the daily and monthly usage counters.
// Silently returns nil when providerID <= 0 or tokenDelta <= 0 (env fallback or empty delta).
func (s *Service) incrementTokenUsage(ctx context.Context, providerID int64, tokenDelta int) error {
	if providerID <= 0 || tokenDelta <= 0 {
		return nil
	}
	now := time.Now()
	entries := []struct{ period, periodType string }{
		{now.Format("2006-01-02"), "daily"},
		{now.Format("2006-01"), "monthly"},
	}
	for _, e := range entries {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO provider_quota_usage (provider_id, period, period_type, tokens_used, updated_at)
			VALUES (?, ?, ?, ?, datetime('now'))
			ON CONFLICT(provider_id, period, period_type)
			DO UPDATE SET tokens_used = tokens_used + ?, updated_at = datetime('now')
		`, providerID, e.period, e.periodType, tokenDelta, tokenDelta)
		if err != nil {
			slog.Error("incrementTokenUsage failed", "provider_id", providerID, "period_type", e.periodType, "err", err)
			return err
		}
	}
	return nil
}

// upsertQuotaConfig sets or replaces quota limits for a provider.
// A limit of 0 means unlimited for that period type.
func (s *Service) upsertQuotaConfig(ctx context.Context, providerID int64, dailyLimit, monthlyLimit, resetDay int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO provider_quota_config (provider_id, daily_limit, monthly_limit, reset_day)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(provider_id) DO UPDATE SET
			daily_limit   = excluded.daily_limit,
			monthly_limit = excluded.monthly_limit,
			reset_day     = excluded.reset_day
	`, providerID, dailyLimit, monthlyLimit, resetDay)
	return err
}

// setProviderPriorities updates the priority for each provider in the map
// inside a single transaction. Key = providerID, Value = priority (0 = highest).
func (s *Service) setProviderPriorities(ctx context.Context, priorities map[int64]int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	for id, prio := range priorities {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO provider_priority (provider_id, priority)
			VALUES (?, ?)
			ON CONFLICT(provider_id) DO UPDATE SET priority = excluded.priority
		`, id, prio)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
