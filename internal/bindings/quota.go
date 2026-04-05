/*
  quota.go -- Public Wails bindings for quota management and provider priority.
  Exposes GetQuotaStatus, SetProviderQuota, ReorderProviders to the frontend.
  Implements ROUTER-01, ROUTER-02, ROUTER-05 from PRD §6.11.
*/

package bindings

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// QuotaStatus represents the quota state for a single provider in the current period.
type QuotaStatus struct {
	ProviderID   int64  `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	Priority     int    `json:"priority"`
	DailyLimit   int    `json:"daily_limit"`
	MonthlyLimit int    `json:"monthly_limit"`
	DailyUsed    int    `json:"daily_used"`
	MonthlyUsed  int    `json:"monthly_used"`
	Exhausted    bool   `json:"exhausted"`
}

// SetProviderQuotaPayload configures daily/monthly token limits for one provider.
type SetProviderQuotaPayload struct {
	ProviderID   int64 `json:"provider_id"`
	DailyLimit   int   `json:"daily_limit"`
	MonthlyLimit int   `json:"monthly_limit"`
	ResetDay     int   `json:"reset_day"` // 1-28, day of month for monthly reset
}

// ReorderProvidersPayload sets the routing priority order.
// ProviderIDs[0] becomes priority 0 (highest), [1] priority 1, etc.
type ReorderProvidersPayload struct {
	ProviderIDs []int64 `json:"provider_ids"`
}

// GetQuotaStatus returns the current quota state for all active providers,
// ordered by priority (highest first). Intended for the Settings UI (ROUTER-05).
func (s *Service) GetQuotaStatus(ctx context.Context) ([]QuotaStatus, error) {
	slog.Debug("GetQuotaStatus called")
	providers, err := s.getOrderedProviders(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	day := now.Format("2006-01-02")
	month := now.Format("2006-01")

	result := make([]QuotaStatus, 0, len(providers))
	for _, p := range providers {
		name, nameErr := s.getProviderName(ctx, p.ID)
		if nameErr != nil {
			continue
		}
		dailyUsed, _ := s.getUsageForPeriod(ctx, p.ID, day, "daily")
		monthlyUsed, _ := s.getUsageForPeriod(ctx, p.ID, month, "monthly")
		exhausted := (p.DailyLimit > 0 && dailyUsed >= p.DailyLimit) ||
			(p.MonthlyLimit > 0 && monthlyUsed >= p.MonthlyLimit)

		result = append(result, QuotaStatus{
			ProviderID:   p.ID,
			ProviderName: name,
			Priority:     p.Priority,
			DailyLimit:   p.DailyLimit,
			MonthlyLimit: p.MonthlyLimit,
			DailyUsed:    dailyUsed,
			MonthlyUsed:  monthlyUsed,
			Exhausted:    exhausted,
		})
	}
	return result, nil
}

// SetProviderQuota configures daily/monthly token limits for a provider (ROUTER-02).
// Set DailyLimit and/or MonthlyLimit to 0 to remove that limit.
func (s *Service) SetProviderQuota(ctx context.Context, payload SetProviderQuotaPayload) (map[string]any, error) {
	slog.Debug("SetProviderQuota called", "provider_id", payload.ProviderID)
	if payload.ProviderID <= 0 {
		return map[string]any{"ok": false, "reason": "invalid-provider-id"}, nil
	}
	resetDay := payload.ResetDay
	if resetDay < 1 || resetDay > 28 {
		resetDay = 1
	}
	if err := s.upsertQuotaConfig(ctx, payload.ProviderID, payload.DailyLimit, payload.MonthlyLimit, resetDay); err != nil {
		return nil, fmt.Errorf("set quota: %w", err)
	}
	return map[string]any{"ok": true}, nil
}

// ReorderProviders assigns routing priority based on position in ProviderIDs
// (index 0 = highest priority = selected first by the router). Implements ROUTER-01.
func (s *Service) ReorderProviders(ctx context.Context, payload ReorderProvidersPayload) (map[string]any, error) {
	slog.Debug("ReorderProviders called", "count", len(payload.ProviderIDs))
	if len(payload.ProviderIDs) == 0 {
		return map[string]any{"ok": false, "reason": "empty-list"}, nil
	}
	priorities := make(map[int64]int, len(payload.ProviderIDs))
	for i, id := range payload.ProviderIDs {
		if id <= 0 {
			return map[string]any{"ok": false, "reason": "invalid-provider-id"}, nil
		}
		priorities[id] = i
	}
	if err := s.setProviderPriorities(ctx, priorities); err != nil {
		return nil, fmt.Errorf("reorder providers: %w", err)
	}
	return map[string]any{"ok": true}, nil
}

// getProviderName is an internal helper to fetch only the name of a provider row.
func (s *Service) getProviderName(ctx context.Context, id int64) (string, error) {
	var name string
	err := s.db.QueryRowContext(ctx, `SELECT name FROM providers WHERE id=?`, id).Scan(&name)
	return name, err
}
