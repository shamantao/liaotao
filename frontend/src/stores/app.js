/**
 * app.js -- UI-level application store.
 * Responsibilities: sidebar state, active tab, theme selection.
 * Reactive Svelte writable stores replacing appState UI properties.
 */

import { writable, derived } from "svelte/store";

// ── Sidebar state ──────────────────────────────────────────────────────────
export const sidebarCollapsed = writable(false);
export const sidebarWidth = writable(290);
export const expandedSidebarWidth = writable(290);
export const sidebarMobileOpen = writable(false);

// ── Sidebar section toggles ────────────────────────────────────────────────
export const groupsExpanded = writable(true);
export const conversationsExpanded = writable(true);

// ── Active tab (Chat | Settings) ───────────────────────────────────────────
export const activeTab = writable("chat");

// ── Settings sub-section ───────────────────────────────────────────────────
export const settingsSection = writable("general");

// ── Derived convenience: is sidebar visible? ───────────────────────────────
export const sidebarVisible = derived(
  [sidebarCollapsed, sidebarMobileOpen],
  ([$collapsed, $mobileOpen]) => !$collapsed || $mobileOpen,
);

// ── Actions ────────────────────────────────────────────────────────────────

export function toggleSidebar() {
  sidebarCollapsed.update((v) => !v);
}

export function switchTab(tab) {
  activeTab.set(tab);
}
