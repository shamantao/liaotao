/**
 * providers.js -- Provider & model store.
 * Responsibilities: provider list, active provider, connection status,
 * models cache, last-used models, quota/router state.
 * Reactive Svelte writable stores replacing appState provider properties.
 */

import { writable, derived } from "svelte/store";

// ── Persistence key (shared with settings store) ───────────────────────────
const STORAGE_KEY = "liaotao.settings.v2";

// ── Providers ──────────────────────────────────────────────────────────────
export const providers = writable([]);
export const activeProviderId = writable(null);
export const providerStatus = writable({});
export const modelFilterQuery = writable("");

// ── Last used models (persisted in localStorage) ───────────────────────────
export const lastUsedModels = writable([]);

// ── Derived: active provider record ────────────────────────────────────────
export const activeProvider = derived(
  [providers, activeProviderId],
  ([$providers, $id]) => $providers.find((p) => p.id === $id) || null,
);

// ── Actions ────────────────────────────────────────────────────────────────

/**
 * Record a model usage for the "last used" quick-pick list.
 * Keeps at most 6 entries, most recent first.
 */
export function recordModelUsage(providerId, providerName, model) {
  lastUsedModels.update((list) => {
    const filtered = list.filter(
      (m) => !(m.providerId === providerId && m.model === model),
    );
    return [
      { providerId, providerName, model, usedAt: Date.now() },
      ...filtered,
    ].slice(0, 6);
  });
}

/**
 * Update a single provider's connection status.
 */
export function setProviderStatus(providerId, status) {
  providerStatus.update((map) => ({ ...map, [providerId]: status }));
}

// ── Persistence ────────────────────────────────────────────────────────────

/**
 * Load persisted provider state from localStorage.
 * Called once at app startup.
 */
export function loadProviderState() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return;
    const parsed = JSON.parse(raw);
    if (parsed.activeProviderId != null) {
      activeProviderId.set(parsed.activeProviderId);
    }
    if (Array.isArray(parsed.lastUsedModels)) {
      lastUsedModels.set(parsed.lastUsedModels.slice(0, 6));
    }
  } catch {
    // Ignore corrupt storage
  }
}

/**
 * Persist provider state to localStorage.
 * Should be called after any mutation to activeProviderId or lastUsedModels.
 */
export function persistProviderState() {
  let currentSettings = {};
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) currentSettings = JSON.parse(raw);
  } catch {
    // Start fresh
  }

  let $activeProviderId;
  activeProviderId.subscribe((v) => ($activeProviderId = v))();

  let $lastUsedModels;
  lastUsedModels.subscribe((v) => ($lastUsedModels = v))();

  localStorage.setItem(
    STORAGE_KEY,
    JSON.stringify({
      ...currentSettings,
      activeProviderId: $activeProviderId,
      lastUsedModels: $lastUsedModels,
    }),
  );
}
