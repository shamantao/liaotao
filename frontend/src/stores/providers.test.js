/**
 * providers.test.js -- Unit tests for stores/providers.js.
 * Covers: recordModelUsage, setProviderStatus, activeProvider derived,
 * loadProviderState, persistProviderState.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { get } from "svelte/store";
import {
  providers,
  activeProviderId,
  providerStatus,
  lastUsedModels,
  activeProvider,
  recordModelUsage,
  setProviderStatus,
  loadProviderState,
  persistProviderState,
} from "../stores/providers.js";

describe("stores/providers.js", () => {
  beforeEach(() => {
    providers.set([]);
    activeProviderId.set(null);
    providerStatus.set({});
    lastUsedModels.set([]);
  });

  it("activeProvider is null when no provider matches", () => {
    providers.set([{ id: 1, name: "Groq" }]);
    activeProviderId.set(99);
    expect(get(activeProvider)).toBeNull();
  });

  it("activeProvider returns matching provider", () => {
    providers.set([
      { id: 1, name: "Groq" },
      { id: 2, name: "OpenAI" },
    ]);
    activeProviderId.set(2);
    expect(get(activeProvider)).toEqual({ id: 2, name: "OpenAI" });
  });

  it("recordModelUsage adds entry and keeps max 6", () => {
    for (let i = 1; i <= 8; i++) {
      recordModelUsage(i, `Provider${i}`, `model-${i}`);
    }
    const list = get(lastUsedModels);
    expect(list).toHaveLength(6);
    // Most recent first
    expect(list[0].model).toBe("model-8");
    expect(list[5].model).toBe("model-3");
  });

  it("recordModelUsage deduplicates by provider+model", () => {
    recordModelUsage(1, "Groq", "llama-3");
    recordModelUsage(2, "OpenAI", "gpt-4");
    recordModelUsage(1, "Groq", "llama-3"); // duplicate
    const list = get(lastUsedModels);
    expect(list).toHaveLength(2);
    // Deduplicated entry moves to front
    expect(list[0].model).toBe("llama-3");
    expect(list[1].model).toBe("gpt-4");
  });

  it("setProviderStatus updates status map", () => {
    setProviderStatus(1, "connected");
    setProviderStatus(2, "error");
    const map = get(providerStatus);
    expect(map[1]).toBe("connected");
    expect(map[2]).toBe("error");
  });

  it("persistProviderState saves to localStorage", () => {
    activeProviderId.set(5);
    lastUsedModels.set([{ providerId: 1, providerName: "X", model: "y", usedAt: 0 }]);
    persistProviderState();
    const raw = JSON.parse(localStorage.getItem("liaotao.settings.v2"));
    expect(raw.activeProviderId).toBe(5);
    expect(raw.lastUsedModels).toHaveLength(1);
  });

  it("loadProviderState restores from localStorage", () => {
    localStorage.setItem(
      "liaotao.settings.v2",
      JSON.stringify({
        activeProviderId: 42,
        lastUsedModels: [{ providerId: 1, providerName: "A", model: "b", usedAt: 0 }],
      }),
    );
    loadProviderState();
    expect(get(activeProviderId)).toBe(42);
    expect(get(lastUsedModels)).toHaveLength(1);
  });

  it("loadProviderState handles corrupt data gracefully", () => {
    localStorage.setItem("liaotao.settings.v2", "BROKEN");
    expect(() => loadProviderState()).not.toThrow();
  });

  it("loadProviderState does nothing if no saved data", () => {
    loadProviderState();
    expect(get(activeProviderId)).toBeNull();
    expect(get(lastUsedModels)).toEqual([]);
  });
});
