/**
 * theme.test.js -- Unit tests for stores/theme.js.
 * Covers: theme registration, listing, applying, localStorage persistence,
 * fallback to default-dark, dynamic <link> element management.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { get } from "svelte/store";

// We need a fresh module for each test because theme.js runs applyTheme
// on module load. Use dynamic import + vi.resetModules().

describe("stores/theme.js", () => {
  beforeEach(() => {
    vi.resetModules();
    localStorage.clear();
    // Remove any leftover <link> elements from previous tests
    document.getElementById("liaotao-theme-link")?.remove();
  });

  async function loadThemeModule() {
    return await import("../stores/theme.js");
  }

  it("exports default-dark as the only built-in theme", async () => {
    const { listThemes } = await loadThemeModule();
    const all = listThemes();
    expect(all).toHaveLength(1);
    expect(all[0]).toEqual({ id: "default-dark", label: "Default Dark", path: null });
  });

  it("activeThemeId defaults to 'default-dark'", async () => {
    const { activeThemeId } = await loadThemeModule();
    expect(get(activeThemeId)).toBe("default-dark");
  });

  it("registerTheme adds a new theme to the list", async () => {
    const { registerTheme, listThemes } = await loadThemeModule();
    registerTheme("solarized", "Solarized Light", "./themes/solarized.css");
    const all = listThemes();
    expect(all).toHaveLength(2);
    expect(all[1].id).toBe("solarized");
    expect(all[1].label).toBe("Solarized Light");
    expect(all[1].path).toBe("./themes/solarized.css");
  });

  it("registerTheme prevents duplicate ids", async () => {
    const { registerTheme, listThemes } = await loadThemeModule();
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    registerTheme("custom", "Custom", "./c.css");
    registerTheme("custom", "Custom 2", "./c2.css");
    expect(listThemes()).toHaveLength(2); // default-dark + custom
    expect(warn).toHaveBeenCalledWith(expect.stringContaining("already registered"));
    warn.mockRestore();
  });

  it("applyTheme sets activeThemeId and persists to localStorage", async () => {
    const { registerTheme, applyTheme, activeThemeId } = await loadThemeModule();
    registerTheme("ocean", "Ocean", "./ocean.css");
    applyTheme("ocean");
    expect(get(activeThemeId)).toBe("ocean");
    expect(localStorage.getItem("liaotao-theme")).toBe("ocean");
  });

  it("applyTheme creates a <link> element for external themes", async () => {
    const { registerTheme, applyTheme } = await loadThemeModule();
    registerTheme("fancy", "Fancy", "./fancy.css");
    applyTheme("fancy");
    const link = document.getElementById("liaotao-theme-link");
    expect(link).not.toBeNull();
    expect(link.href).toContain("fancy.css");
    expect(link.rel).toBe("stylesheet");
  });

  it("applyTheme removes <link> when switching back to built-in theme", async () => {
    const { registerTheme, applyTheme } = await loadThemeModule();
    registerTheme("ext", "External", "./ext.css");
    applyTheme("ext");
    expect(document.getElementById("liaotao-theme-link")).not.toBeNull();
    applyTheme("default-dark");
    expect(document.getElementById("liaotao-theme-link")).toBeNull();
  });

  it("applyTheme falls back to default-dark for unknown id", async () => {
    const { applyTheme, activeThemeId } = await loadThemeModule();
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    applyTheme("nonexistent-theme");
    expect(get(activeThemeId)).toBe("default-dark");
    expect(warn).toHaveBeenCalledWith(expect.stringContaining("not found"));
    warn.mockRestore();
  });

  it("loads saved theme from localStorage on init", async () => {
    localStorage.setItem("liaotao-theme", "default-dark");
    const { activeThemeId } = await loadThemeModule();
    expect(get(activeThemeId)).toBe("default-dark");
  });

  it("themes derived store reflects registered themes reactively", async () => {
    const { themes, registerTheme } = await loadThemeModule();
    const initial = get(themes);
    expect(initial).toHaveLength(1);
    registerTheme("new-one", "New One", "./new.css");
    const updated = get(themes);
    expect(updated).toHaveLength(2);
  });
});
