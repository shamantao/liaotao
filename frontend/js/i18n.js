/*
  i18n.js -- Lightweight internationalization engine for liaotao.
  Responsibilities: load JSON translation bundles, expose t() lookup with
  hierarchical dot-notation keys, apply translations to static DOM via
  data-i18n / data-i18n-placeholder / data-i18n-title attributes, and
  expose setLanguage() for runtime language switching.

  Supported attributes:
    data-i18n="key"              → element.textContent
    data-i18n-placeholder="key" → element.placeholder
    data-i18n-title="key"       → element.title / aria-label (buttons)

  Interpolation: replace {{varName}} in the translated string.
    t("providers.get_api_key", { name: "OpenAI" })
*/

const SUPPORTED_LANGS = ["en", "fr", "zh-TW"];
const DEFAULT_LANG    = "en";

// In-memory store: { "en": { nav: { chat: "Chat", … }, … }, … }
const _bundles = {};
let _current   = DEFAULT_LANG;

// ── Load ───────────────────────────────────────────────────────────────────

/**
 * Fetch a language bundle from the i18n/ directory.
 * Caches the result so subsequent calls are instant.
 */
async function _loadBundle(lang) {
  if (_bundles[lang]) return _bundles[lang];
  try {
    const res  = await fetch(`./i18n/${lang}.json`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    _bundles[lang] = await res.json();
  } catch (err) {
    console.warn(`[i18n] Failed to load "${lang}", falling back to "${DEFAULT_LANG}".`, err);
    if (lang !== DEFAULT_LANG) {
      _bundles[lang] = await _loadBundle(DEFAULT_LANG);
    } else {
      _bundles[lang] = {};
    }
  }
  return _bundles[lang];
}

// ── Public API ─────────────────────────────────────────────────────────────

/**
 * Returns the current active language code (e.g. "en", "fr", "zh-TW").
 */
export function currentLang() {
  return _current;
}

/**
 * Returns an array of supported language codes.
 */
export function supportedLangs() {
  return [...SUPPORTED_LANGS];
}

/**
 * Translate a dot-notation key with optional variable interpolation.
 * Falls back to the English bundle, then to the key itself.
 *
 * @param {string} key   - e.g. "providers.get_api_key"
 * @param {object} [vars] - e.g. { name: "OpenAI" }
 * @returns {string}
 */
export function t(key, vars) {
  const value = _resolve(key, _bundles[_current]) ?? _resolve(key, _bundles[DEFAULT_LANG]) ?? key;
  if (!vars) return value;
  return value.replace(/\{\{(\w+)\}\}/g, (_, k) => (vars[k] ?? `{{${k}}}`));
}

/**
 * Load the given language bundle and make it the active language.
 * Resolves once the bundle is ready (no DOM update — call applyTranslations separately).
 *
 * @param {string} lang - Language code (e.g. "fr", "zh-TW")
 */
export async function setLanguage(lang) {
  const target = SUPPORTED_LANGS.includes(lang) ? lang : DEFAULT_LANG;
  // Always ensure EN is loaded as fallback.
  await Promise.all([_loadBundle(DEFAULT_LANG), _loadBundle(target)]);
  _current = target;
  document.documentElement.lang = target;
}

/**
 * Walk the DOM and apply translations to all elements with data-i18n* attributes.
 * Safe to call multiple times (idempotent).
 */
export function applyTranslations() {
  document.querySelectorAll("[data-i18n]").forEach((el) => {
    const key = el.dataset.i18n;
    if (key) el.textContent = t(key);
  });
  document.querySelectorAll("[data-i18n-placeholder]").forEach((el) => {
    const key = el.dataset.i18nPlaceholder;
    if (key) el.placeholder = t(key);
  });
  document.querySelectorAll("[data-i18n-title]").forEach((el) => {
    const key = el.dataset.i18nTitle;
    if (!key) return;
    const val = t(key);
    el.title = val;
    if (el.getAttribute("aria-label")) el.setAttribute("aria-label", val);
  });
}

/**
 * Convenience: load the language bundle once at application startup.
 * Call before any call to t() or applyTranslations().
 *
 * @param {string} lang - Language code to initialise with.
 */
export async function initI18n(lang) {
  await setLanguage(lang || DEFAULT_LANG);
}

// ── Internal ───────────────────────────────────────────────────────────────

/** Traverse a nested object by dot-notation path. Returns undefined if not found. */
function _resolve(key, bundle) {
  if (!bundle || typeof key !== "string") return undefined;
  return key.split(".").reduce((obj, part) => (obj && typeof obj === "object" ? obj[part] : undefined), bundle);
}
