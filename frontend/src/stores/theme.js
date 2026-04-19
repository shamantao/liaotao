/**
 * theme.js -- Theme loader store for liaotao.
 * Responsibilities: load/apply CSS themes via dynamic <link>, switch without
 * page reload, fallback to default-dark. Exposes registerTheme / listThemes
 * for the plugin API.
 */

import { writable, derived } from "svelte/store";

// ── Theme registry ─────────────────────────────────────────────────────────

const BUILTIN_THEMES = [
  { id: "default-dark", label: "Default Dark", path: null },
];

const _registry = writable([...BUILTIN_THEMES]);

// ── Active theme ───────────────────────────────────────────────────────────

const STORAGE_KEY = "liaotao-theme";

function loadSavedTheme() {
  try {
    return localStorage.getItem(STORAGE_KEY) || "default-dark";
  } catch {
    return "default-dark";
  }
}

export const activeThemeId = writable(loadSavedTheme());

// ── Derived: list all registered themes ────────────────────────────────────

export const themes = derived(_registry, ($r) => $r);

// ── Internal: manage the dynamic <link> element ────────────────────────────

let themeLink = null;

function ensureLinkElement() {
  if (themeLink) return themeLink;
  themeLink = document.createElement("link");
  themeLink.rel = "stylesheet";
  themeLink.id = "liaotao-theme-link";
  document.head.appendChild(themeLink);
  return themeLink;
}

function removeLinkElement() {
  if (themeLink) {
    themeLink.remove();
    themeLink = null;
  }
}

// ── Apply a theme by id ────────────────────────────────────────────────────

export function applyTheme(id) {
  let entries;
  _registry.subscribe((v) => (entries = v))();
  const theme = entries.find((t) => t.id === id);
  if (!theme) {
    console.warn(`[liaotao] theme "${id}" not found, falling back to default-dark`);
    applyTheme("default-dark");
    return;
  }

  if (theme.path) {
    // External / community theme: load via dynamic <link>
    const link = ensureLinkElement();
    link.href = theme.path;
  } else {
    // Built-in theme: already imported statically in main.js, remove override
    removeLinkElement();
  }

  activeThemeId.set(id);
  try {
    localStorage.setItem(STORAGE_KEY, id);
  } catch {
    // localStorage may be unavailable
  }
}

// ── Plugin API: register a community theme ─────────────────────────────────

/**
 * Register an external theme.
 * @param {string} id    - Unique theme identifier (e.g. "solarized-light").
 * @param {string} label - Human-readable name shown in Settings.
 * @param {string} path  - URL or relative path to the theme CSS file.
 */
export function registerTheme(id, label, path) {
  _registry.update((list) => {
    if (list.some((t) => t.id === id)) {
      console.warn(`[liaotao] theme "${id}" already registered`);
      return list;
    }
    return [...list, { id, label, path }];
  });
}

/**
 * Return the current list of registered themes (snapshot).
 * @returns {{ id: string, label: string, path: string|null }[]}
 */
export function listThemes() {
  let result;
  _registry.subscribe((v) => (result = v))();
  return result;
}

// ── Init: apply saved theme on module load ─────────────────────────────────

applyTheme(loadSavedTheme());
