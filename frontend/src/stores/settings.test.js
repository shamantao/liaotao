/**
 * settings.test.js -- Unit tests for stores/settings.js.
 * Covers: defaults, updateSettings, loadSettings persistence,
 * CSS variable application, FONT_SIZE_MAP export.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { get } from "svelte/store";
import {
  settings,
  updateSettings,
  loadSettings,
  FONT_SIZE_MAP,
} from "../stores/settings.js";

describe("stores/settings.js", () => {
  beforeEach(() => {
    // Reset store to defaults
    settings.set({
      language: "en",
      theme: "dark",
      showMetaFooter: true,
      defaultSystemPrompt: "",
      expertMode: false,
      responseStyle: "balanced",
      chatFontSize: "L",
      autoCheckUpdates: true,
    });
  });

  it("exports FONT_SIZE_MAP with expected keys", () => {
    expect(Object.keys(FONT_SIZE_MAP)).toEqual(["xs", "s", "m", "L", "XL"]);
    expect(FONT_SIZE_MAP["L"]).toEqual(["1rem", "26px"]);
  });

  it("settings store has correct defaults", () => {
    const s = get(settings);
    expect(s.language).toBe("en");
    expect(s.showMetaFooter).toBe(true);
    expect(s.expertMode).toBe(false);
    expect(s.chatFontSize).toBe("L");
    expect(s.responseStyle).toBe("balanced");
  });

  it("updateSettings merges partial patch", () => {
    updateSettings({ language: "fr", expertMode: true });
    const s = get(settings);
    expect(s.language).toBe("fr");
    expect(s.expertMode).toBe(true);
    // Unchanged values remain
    expect(s.showMetaFooter).toBe(true);
  });

  it("updateSettings persists to localStorage", () => {
    updateSettings({ language: "zh-TW" });
    const raw = localStorage.getItem("liaotao.settings.v2");
    expect(raw).toBeTruthy();
    const parsed = JSON.parse(raw);
    expect(parsed.settings.language).toBe("zh-TW");
  });

  it("updateSettings applies CSS variables", () => {
    updateSettings({ chatFontSize: "xs" });
    const root = document.documentElement;
    expect(root.style.getPropertyValue("--chat-font-size")).toBe("0.78rem");
    expect(root.style.getPropertyValue("--chat-icon-size")).toBe("20px");
  });

  it("loadSettings restores from localStorage", () => {
    localStorage.setItem(
      "liaotao.settings.v2",
      JSON.stringify({ settings: { language: "fr", chatFontSize: "XL" } }),
    );
    loadSettings();
    const s = get(settings);
    expect(s.language).toBe("fr");
    expect(s.chatFontSize).toBe("XL");
    // Defaults fill in missing keys
    expect(s.expertMode).toBe(false);
  });

  it("loadSettings ignores corrupt localStorage gracefully", () => {
    localStorage.setItem("liaotao.settings.v2", "not valid json!!!");
    expect(() => loadSettings()).not.toThrow();
  });

  it("loadSettings does nothing if no saved data", () => {
    loadSettings();
    expect(get(settings).language).toBe("en");
  });
});
