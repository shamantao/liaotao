/*
  router_test.go -- Unit tests for Smart Router: quota check, candidate selection,
  token usage tracking, quota status API, and provider reordering.
*/

package bindings

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// newRouterTestService builds an in-memory DB with the full schema required by the router.
func newRouterTestService(t *testing.T) *Service {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		t.Fatalf("pragma: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := `
	CREATE TABLE providers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		type TEXT NOT NULL DEFAULT 'openai-compatible',
		url TEXT NOT NULL DEFAULT '',
		api_key TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		use_in_rag INTEGER NOT NULL DEFAULT 0,
		active INTEGER NOT NULL DEFAULT 1,
		temperature REAL NOT NULL DEFAULT 0.7,
		num_ctx INTEGER NOT NULL DEFAULT 1024,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE TABLE provider_priority (
		provider_id INTEGER PRIMARY KEY REFERENCES providers(id) ON DELETE CASCADE,
		priority    INTEGER NOT NULL DEFAULT 100
	);
	CREATE TABLE provider_quota_config (
		provider_id   INTEGER PRIMARY KEY REFERENCES providers(id) ON DELETE CASCADE,
		daily_limit   INTEGER NOT NULL DEFAULT 0,
		monthly_limit INTEGER NOT NULL DEFAULT 0,
		reset_day     INTEGER NOT NULL DEFAULT 1
	);
	CREATE TABLE provider_quota_usage (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		provider_id INTEGER NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
		period      TEXT    NOT NULL,
		period_type TEXT    NOT NULL,
		tokens_used INTEGER NOT NULL DEFAULT 0,
		updated_at  TEXT    NOT NULL DEFAULT (datetime('now')),
		UNIQUE(provider_id, period, period_type)
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return NewService(db)
}

// insertProvider adds a provider row and returns its ID.
func insertProvider(t *testing.T, svc *Service, name, pType string, active bool) int64 {
	t.Helper()
	ctx := context.Background()
	a := 1
	if !active {
		a = 0
	}
	res, err := svc.db.ExecContext(ctx,
		`INSERT INTO providers (name, type, active) VALUES (?, ?, ?)`, name, pType, a)
	if err != nil {
		t.Fatalf("insertProvider %q: %v", name, err)
	}
	id, _ := res.LastInsertId()
	return id
}

// ── isQuotaExceeded ────────────────────────────────────────────────────────────

func TestIsQuotaExceeded_NoLimits(t *testing.T) {
	svc := newRouterTestService(t)
	op := orderedProvider{ID: 1, DailyLimit: 0, MonthlyLimit: 0}
	exceeded, err := svc.isQuotaExceeded(context.Background(), op)
	if err != nil || exceeded {
		t.Fatalf("expected false/nil for unlimited provider, got exceeded=%v err=%v", exceeded, err)
	}
}

func TestIsQuotaExceeded_DailyLimitNotReached(t *testing.T) {
	svc := newRouterTestService(t)
	id := insertProvider(t, svc, "groq", "openai-compatible", true)
	ctx := context.Background()
	_ = svc.incrementTokenUsage(ctx, id, 500) // 500 < 1000 daily limit
	op := orderedProvider{ID: id, DailyLimit: 1000, MonthlyLimit: 0}
	exceeded, err := svc.isQuotaExceeded(ctx, op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exceeded {
		t.Fatal("expected quota NOT exceeded, got exceeded=true")
	}
}

func TestIsQuotaExceeded_DailyLimitReached(t *testing.T) {
	svc := newRouterTestService(t)
	id := insertProvider(t, svc, "groq-limited", "openai-compatible", true)
	ctx := context.Background()
	_ = svc.incrementTokenUsage(ctx, id, 1000) // exactly at limit
	op := orderedProvider{ID: id, DailyLimit: 1000, MonthlyLimit: 0}
	exceeded, err := svc.isQuotaExceeded(ctx, op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exceeded {
		t.Fatal("expected quota exceeded, got exceeded=false")
	}
}

func TestIsQuotaExceeded_MonthlyLimitReached(t *testing.T) {
	svc := newRouterTestService(t)
	id := insertProvider(t, svc, "openai-monthly", "openai-compatible", true)
	ctx := context.Background()
	_ = svc.incrementTokenUsage(ctx, id, 5000)
	op := orderedProvider{ID: id, DailyLimit: 0, MonthlyLimit: 5000}
	exceeded, err := svc.isQuotaExceeded(ctx, op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exceeded {
		t.Fatal("expected monthly quota exceeded")
	}
}

// ── incrementTokenUsage / getUsageForPeriod ───────────────────────────────────

func TestIncrementTokenUsage_Accumulates(t *testing.T) {
	svc := newRouterTestService(t)
	id := insertProvider(t, svc, "test-provider", "openai-compatible", true)
	ctx := context.Background()
	day := time.Now().Format("2006-01-02")

	_ = svc.incrementTokenUsage(ctx, id, 100)
	_ = svc.incrementTokenUsage(ctx, id, 200)
	used, err := svc.getUsageForPeriod(ctx, id, day, "daily")
	if err != nil {
		t.Fatalf("getUsageForPeriod error: %v", err)
	}
	if used != 300 {
		t.Fatalf("expected 300 tokens, got %d", used)
	}
}

func TestIncrementTokenUsage_SkipsEnvFallback(t *testing.T) {
	svc := newRouterTestService(t)
	ctx := context.Background()
	// providerID = 0 should be a no-op (env fallback, no row in DB)
	err := svc.incrementTokenUsage(ctx, 0, 500)
	if err != nil {
		t.Fatalf("unexpected error for providerID=0: %v", err)
	}
}

func TestGetUsageForPeriod_NoData(t *testing.T) {
	svc := newRouterTestService(t)
	id := insertProvider(t, svc, "fresh", "openai-compatible", true)
	used, err := svc.getUsageForPeriod(context.Background(), id, "2030-01-01", "daily")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 0 {
		t.Fatalf("expected 0 for no-data, got %d", used)
	}
}

// ── selectCandidates ──────────────────────────────────────────────────────────

func TestSelectCandidates_EnvFallbackWhenNoDB(t *testing.T) {
	svc := newRouterTestService(t)
	// No providers configured — should return the env-based fallback candidate.
	candidates, err := svc.selectCandidates(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 fallback candidate, got %d", len(candidates))
	}
	if candidates[0].ProviderID != 0 {
		t.Fatalf("fallback candidate should have ProviderID=0, got %d", candidates[0].ProviderID)
	}
}

func TestSelectCandidates_SkipsExhaustedProvider(t *testing.T) {
	svc := newRouterTestService(t)
	ctx := context.Background()
	id := insertProvider(t, svc, "exhausted-groq", "openai-compatible", true)
	// Set daily limit of 100 and consume it immediately.
	_ = svc.upsertQuotaConfig(ctx, id, 100, 0, 1)
	_ = svc.incrementTokenUsage(ctx, id, 100)

	candidates, err := svc.selectCandidates(ctx)
	// Should return the env fallback (only active provider is exhausted).
	if err != nil && err.Error() != "all-quotas-exhausted" {
		t.Fatalf("unexpected error type: %v", err)
	}
	if err == nil && len(candidates) == 1 && candidates[0].ProviderID == 0 {
		// Env fallback — acceptable when all DB providers are exhausted.
		return
	}
	if err != nil && err.Error() == "all-quotas-exhausted" {
		// No env fallback: expected when no providers are available.
		return
	}
	t.Fatalf("unexpected candidates state: err=%v candidates=%v", err, candidates)
}

func TestSelectCandidates_PriorityOrder(t *testing.T) {
	svc := newRouterTestService(t)
	ctx := context.Background()
	id1 := insertProvider(t, svc, "provider-low-prio", "openai-compatible", true)
	id2 := insertProvider(t, svc, "provider-high-prio", "openai-compatible", true)
	// Assign priorities: id2 higher (priority=0), id1 lower (priority=10).
	_ = svc.setProviderPriorities(ctx, map[int64]int{id1: 10, id2: 0})

	candidates, err := svc.selectCandidates(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) < 2 {
		t.Fatalf("expected at least 2 candidates, got %d", len(candidates))
	}
	if candidates[0].ProviderID != id2 {
		t.Fatalf("expected high-priority provider (id=%d) first, got id=%d", id2, candidates[0].ProviderID)
	}
}

// ── GetQuotaStatus ────────────────────────────────────────────────────────────

func TestGetQuotaStatus_ReturnsAllActive(t *testing.T) {
	svc := newRouterTestService(t)
	ctx := context.Background()
	id := insertProvider(t, svc, "quota-provider", "openai-compatible", true)
	_ = svc.upsertQuotaConfig(ctx, id, 1000, 30000, 1)
	_ = svc.incrementTokenUsage(ctx, id, 250)

	statuses, err := svc.GetQuotaStatus(ctx)
	if err != nil {
		t.Fatalf("GetQuotaStatus error: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	s := statuses[0]
	if s.ProviderID != id {
		t.Errorf("wrong provider ID: got %d want %d", s.ProviderID, id)
	}
	if s.DailyLimit != 1000 || s.MonthlyLimit != 30000 {
		t.Errorf("wrong limits: daily=%d monthly=%d", s.DailyLimit, s.MonthlyLimit)
	}
	if s.DailyUsed != 250 {
		t.Errorf("expected daily_used=250, got %d", s.DailyUsed)
	}
	if s.Exhausted {
		t.Error("provider should not be marked exhausted (250 < 1000)")
	}
}

// ── ReorderProviders ──────────────────────────────────────────────────────────

func TestReorderProviders_SetsCorrectPriority(t *testing.T) {
	svc := newRouterTestService(t)
	ctx := context.Background()
	id1 := insertProvider(t, svc, "p1", "openai-compatible", true)
	id2 := insertProvider(t, svc, "p2", "openai-compatible", true)
	id3 := insertProvider(t, svc, "p3", "openai-compatible", true)

	result, err := svc.ReorderProviders(ctx, ReorderProvidersPayload{
		ProviderIDs: []int64{id3, id1, id2},
	})
	if err != nil || result["ok"] != true {
		t.Fatalf("ReorderProviders failed: err=%v result=%v", err, result)
	}

	candidates, err := svc.selectCandidates(ctx)
	if err != nil {
		t.Fatalf("selectCandidates error: %v", err)
	}
	if candidates[0].ProviderID != id3 {
		t.Fatalf("expected id3 first (priority 0), got providerID=%d", candidates[0].ProviderID)
	}
}

func TestReorderProviders_RejectsEmptyList(t *testing.T) {
	svc := newRouterTestService(t)
	result, err := svc.ReorderProviders(context.Background(), ReorderProvidersPayload{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["ok"] == true {
		t.Fatal("expected ok=false for empty list")
	}
}
