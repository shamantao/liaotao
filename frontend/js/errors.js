/*
  errors.js -- Centralized Wails error parsing and field-level UI feedback.
  Responsibilities:
    - Parse the Wails RuntimeError JSON envelope into a structured object.
    - Map known SQLite / Go error patterns to human-friendly messages.
    - Apply / clear field-level error highlighting (red border + status text).
  Usage:
    import { parseWailsError, applyFieldError, clearFieldError } from "./errors.js";
    } catch (err) {
      const { message, field } = parseWailsError(err);
      applyFieldError(inputEl, statusEl, message);
    }
*/

// ── Known error patterns ───────────────────────────────────────────────────
// Each entry: { test(rawMsg), message, field } where field is the input id hint.
const ERROR_PATTERNS = [
  {
    // SQLite UNIQUE constraint on mcp_servers.name
    test: (m) => /unique constraint failed: mcp_servers\.name/i.test(m),
    message: "This name already exists. Please choose another one.",
    field: "msf-name",
  },
  {
    // SQLite UNIQUE constraint on providers.name
    test: (m) => /unique constraint failed: providers\.name/i.test(m),
    message: "This name already exists. Please choose another one.",
    field: "pf-name",
  },
  {
    // Generic UNIQUE constraint fallback
    test: (m) => /unique constraint failed/i.test(m),
    message: "This name already exists. Please choose another one.",
    field: null,
  },
  {
    // Not null / missing required field
    test: (m) => /not.null constraint/i.test(m),
    message: "A required field is missing.",
    field: null,
  },
];

/**
 * parseWailsError parses a raw Wails error (string | Error | object) into a
 * structured { message, field } result where field is the HTML input id hint.
 * @param {unknown} err
 * @returns {{ message: string, field: string|null }}
 */
export function parseWailsError(err) {
  // Wails v3 throws objects with a .message property that may contain JSON.
  let raw = "";
  if (err && typeof err === "object" && typeof err.message === "string") {
    raw = err.message;
  } else {
    raw = String(err || "unknown error");
  }

  // Try to extract nested .message from Wails JSON envelope:
  // {"message":"create mcp server: constraint failed: UNIQUE…","cause":{},"kind":"RuntimeError"}
  try {
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed.message === "string") {
      raw = parsed.message;
    }
  } catch {
    // Not JSON — raw is already a plain string.
  }

  for (const pattern of ERROR_PATTERNS) {
    if (pattern.test(raw)) {
      return { message: pattern.message, field: pattern.field };
    }
  }

  // Fallback: return the raw message, cleaned of Go-style "op: " prefix chains.
  const clean = raw.replace(/^[\w\s]+:\s+/g, "").trim() || "An unexpected error occurred.";
  return { message: clean, field: null };
}

/**
 * applyFieldError highlights a form input with a red border and writes the
 * human-friendly message into the status element.
 * @param {HTMLElement|null} inputEl  - the field to highlight (may be null)
 * @param {HTMLElement|null} statusEl - the status <p> element
 * @param {string} message
 */
export function applyFieldError(inputEl, statusEl, message) {
  if (inputEl) inputEl.classList.add("field-error");
  if (statusEl) statusEl.textContent = message;
}

/**
 * clearFieldError removes the red border from a field and clears the status.
 * @param {HTMLElement|null} inputEl
 * @param {HTMLElement|null} statusEl
 */
export function clearFieldError(inputEl, statusEl) {
  if (inputEl) inputEl.classList.remove("field-error");
  if (statusEl) statusEl.textContent = "";
}
