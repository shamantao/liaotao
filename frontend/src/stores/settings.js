/**
 * settings.js -- User preferences store.
 * Responsibilities: language, theme, font size, expert mode,
 * response style, system prompt, update prefs, meta footer toggle.
 * Persisted in localStorage under liaotao.settings.v2.
 */

import { writable } from "svelte/store";

// ── Persistence key ────────────────────────────────────────────────────────
const STORAGE_KEY = "liaotao.settings.v2";

// ── Font size map (CORCT-01) ───────────────────────────────────────────────
export const FONT_SIZE_MAP = {
  xs: ["0.78rem", "20px"],
  s:  ["0.88rem", "22px"],
  m:  ["0.94rem", "24px"],
  L:  ["1rem",    "26px"],
  XL: ["1.12rem", "30px"],
};

// ── Default values ─────────────────────────────────────────────────────────
const DEFAULTS = {
  language:           "en",
  theme:              "dark",
  showMetaFooter:     true,
  defaultSystemPrompt: "",
  expertMode:         false,
  responseStyle:      "balanced",
  chatFontSize:       "L",
  autoCheckUpdates:   true,
};

// ── Settings store ─────────────────────────────────────────────────────────
export const settings = writable({ ...DEFAULTS });

// ── Actions ────────────────────────────────────────────────────────────────

/**
 * Update one or more settings properties.
 * Automatically persists to localStorage and applies CSS variables.
 */
export function updateSettings(patch) {
  settings.update((s) => {
    const updated = { ...s, ...patch };
    persistSettings(updated);
    applyCSSVariables(updated);
    return updated;
  });
}

/**
 * Apply font-size related CSS custom properties to the document root.
 */
function applyCSSVariables(s) {
  const [fontSize, iconSize] = FONT_SIZE_MAP[s.chatFontSize] ?? FONT_SIZE_MAP["L"];
  document.documentElement.style.setProperty("--chat-font-size", fontSize);
  document.documentElement.style.setProperty("--chat-icon-size", iconSize);
}

// ── Persistence ────────────────────────────────────────────────────────────

/**
 * Load settings from localStorage. Called once at app startup.
 */
export function loadSettings() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return;
    const parsed = JSON.parse(raw);
    if (parsed.settings) {
      const merged = { ...DEFAULTS, ...parsed.settings };
      settings.set(merged);
      applyCSSVariables(merged);
    }
  } catch {
    // Ignore corrupt storage
  }
}

/**
 * Persist current settings to localStorage.
 */
function persistSettings(s) {
  let current = {};
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) current = JSON.parse(raw);
  } catch {
    // Start fresh
  }
  localStorage.setItem(
    STORAGE_KEY,
    JSON.stringify({ ...current, settings: s }),
  );
}
