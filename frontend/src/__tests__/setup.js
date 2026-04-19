/**
 * setup.js -- Vitest global setup for jsdom environment.
 * Provides minimal mocks for browser APIs not available in jsdom.
 */

// Provide a minimal localStorage polyfill if jsdom's is incomplete.
if (typeof globalThis.localStorage === "undefined" || !globalThis.localStorage.clear) {
  const store = {};
  globalThis.localStorage = {
    getItem: (key) => (key in store ? store[key] : null),
    setItem: (key, val) => { store[key] = String(val); },
    removeItem: (key) => { delete store[key]; },
    clear: () => { for (const k of Object.keys(store)) delete store[k]; },
    get length() { return Object.keys(store).length; },
    key: (i) => Object.keys(store)[i] ?? null,
  };
}

beforeEach(() => {
  localStorage.clear();
});
