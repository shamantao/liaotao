/**
 * app.test.js -- Unit tests for stores/app.js.
 * Covers: sidebar state, tab switching, derived sidebarVisible.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { get } from "svelte/store";
import {
  sidebarCollapsed,
  sidebarWidth,
  sidebarMobileOpen,
  groupsExpanded,
  conversationsExpanded,
  activeTab,
  settingsSection,
  sidebarVisible,
  toggleSidebar,
  switchTab,
} from "../stores/app.js";

describe("stores/app.js", () => {
  beforeEach(() => {
    sidebarCollapsed.set(false);
    sidebarWidth.set(290);
    sidebarMobileOpen.set(false);
    groupsExpanded.set(true);
    conversationsExpanded.set(true);
    activeTab.set("chat");
    settingsSection.set("general");
  });

  it("sidebar defaults to expanded", () => {
    expect(get(sidebarCollapsed)).toBe(false);
    expect(get(sidebarWidth)).toBe(290);
  });

  it("toggleSidebar flips collapsed state", () => {
    expect(get(sidebarCollapsed)).toBe(false);
    toggleSidebar();
    expect(get(sidebarCollapsed)).toBe(true);
    toggleSidebar();
    expect(get(sidebarCollapsed)).toBe(false);
  });

  it("sidebarVisible is true when not collapsed", () => {
    expect(get(sidebarVisible)).toBe(true);
  });

  it("sidebarVisible is false when collapsed and mobile closed", () => {
    sidebarCollapsed.set(true);
    expect(get(sidebarVisible)).toBe(false);
  });

  it("sidebarVisible is true when collapsed but mobile open", () => {
    sidebarCollapsed.set(true);
    sidebarMobileOpen.set(true);
    expect(get(sidebarVisible)).toBe(true);
  });

  it("switchTab changes activeTab", () => {
    switchTab("settings");
    expect(get(activeTab)).toBe("settings");
    switchTab("chat");
    expect(get(activeTab)).toBe("chat");
  });

  it("section toggles default to expanded", () => {
    expect(get(groupsExpanded)).toBe(true);
    expect(get(conversationsExpanded)).toBe(true);
  });
});
